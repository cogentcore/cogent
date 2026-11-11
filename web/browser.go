// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"image"
	neturl "net/url"
	"slices"
	"strings"
	"sync/atomic"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
)

// Browser is a web browser with tabs, each of which has a [Page].
// Its toolbar ([Browser.MakeToolbar]) is for the page of the current tab.
type Browser struct {
	core.Frame

	// Tabs has the tabs of the pages.
	Tabs *core.Tabs `set:"-"`

	// Splits has the tabs, and the console panel below them.
	Splits *core.Splits `set:"-"`

	// Console is the console panel, which shows the console messages of
	// the scripts of the page of the current tab (see [Browser.ShowConsole]).
	Console *ConsolePanel `set:"-"`

	// consoleShown is whether the console panel is shown.
	consoleShown atomic.Bool

	// OnOpen, if set, is called when a page of a tab starts opening or is
	// opened (see [Page.OnOpen]) and when another tab is selected, as for
	// updating a toolbar.
	OnOpen func() `set:"-" display:"-"`

	// PageFunc, if set, is called with the page of each new tab,
	// as for setting its [htmlcore.Context].
	PageFunc func(pg *Page) `set:"-" display:"-"`

	// pages are the pages of the tabs, in their order.
	pages []*Page

	// shown is whether the browser has been shown, after which the pages
	// of tabs in the background can be opened too.
	shown bool
}

func (br *Browser) Init() {
	br.Frame.Init()
	br.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
		s.Direction = styles.Column
		s.Padding.Zero()
		s.Gap.Zero()
	})
	br.OnShow(func(e events.Event) {
		br.shown = true
		for _, pg := range br.pages {
			pg.setShown()
		}
	})
	br.Splits = core.NewSplits(br)
	br.Splits.Styler(func(s *styles.Style) { s.Direction = styles.Column })
	br.Tabs = core.NewTabs(br.Splits)
	br.Tabs.SetType(core.FunctionalTabs)
	br.Tabs.NewTabButton = true
	br.Tabs.NewTabFunc = func(index int) {
		fr, tab := br.Tabs.TabAtIndex(index)
		br.addPage(fr, tab)
	}
	br.Console = tree.New[ConsolePanel](br.Splits)
	br.Console.browser = br
	br.Splits.SetSplits(1, 0) // (the console is not shown)
	// a tab dragged out of the window goes into a new browser window
	br.Tabs.DragToWindow = func(b *core.Body) *core.Tabs {
		nb := InitBody(b)
		nb.PageFunc = br.PageFunc
		return nb.Tabs
	}
	br.Tabs.MoveTabFunc = br.tabMoved
	br.Tabs.MoveInFunc = br.tabMovedIn
	br.Tabs.CloseTabFunc = func(index int) {
		if index < len(br.pages) {
			br.pages[index].Stop()
			br.pages = slices.Delete(br.pages, index, index+1)
		}
		br.opened()
	}
}

// InitBody initializes the given body of a window as a browser window, with
// a browser and its toolbar in the top bar, and returns the browser, which
// has no tabs yet. It is used for the main window of the app, and for the
// new windows that tabs are dragged into (see [core.Tabs.DragToWindow]).
func InitBody(b *core.Body) *Browser {
	br := NewBrowser(b)
	b.AddTopBar(func(bar *core.Frame) {
		// the toolbar, with the bookmarks bar below it
		bars := core.NewFrame(bar)
		bars.Styler(func(s *styles.Style) {
			s.Direction = styles.Column
			s.Grow.Set(1, 0)
			s.Padding.Zero()
			s.Gap.Zero()
		})
		tb := core.NewToolbar(bars)
		tb.Maker(br.MakeToolbar)
		tb.Maker(func(p *tree.Plan) {
			tree.AddAt(p, "bookmark", func(w *core.Button) {
				w.SetIcon(icons.BookmarkAdd).SetTooltip("Bookmark this page, or edit the bookmarks")
				w.SetMenu(func(m *core.Scene, _ image.Point) { br.bookmarkMenu(m) })
			})
		})
		br.OnOpen = func() { tb.Update() }
		addBookmarksBar(bars, br)
	})
	return br
}

// tabMoved keeps the pages in the order of the tabs when a tab has moved
// from the given index to the given index in the given tabs (see
// [core.Tabs.MoveTabFunc]): these, or those of another browser (as in
// another window), which then adopts its page (see [Browser.tabMovedIn]).
func (br *Browser) tabMoved(from int, to *core.Tabs, at int) {
	if from < 0 || from >= len(br.pages) {
		return
	}
	pg := br.pages[from]
	br.pages = slices.Delete(br.pages, from, from+1)
	if to == br.Tabs {
		br.pages = slices.Insert(br.pages, min(at, len(br.pages)), pg)
	}
	br.opened()
}

// tabMovedIn adopts the page of the tab of another browser that has moved
// into the tabs of the browser at the given index (see
// [core.Tabs.MoveInFunc]).
func (br *Browser) tabMovedIn(index int) {
	fr, tab := br.Tabs.TabAtIndex(index)
	if fr == nil || !fr.HasChildren() {
		return
	}
	pg, ok := fr.Child(0).(*Page)
	if !ok {
		return
	}
	br.adoptPage(pg, tab, index)
	br.opened()
}

// opened calls [Browser.OnOpen], if it is set.
func (br *Browser) opened() {
	if br.ConsoleShown() {
		br.Console.refresh() // (of the page of the current tab)
	}
	if br.OnOpen != nil {
		br.OnOpen()
	}
}

// NewTab adds a new tab and selects it, opening the given URL in it,
// unless it is empty, and returns its page.
func (br *Browser) NewTab(url string) *Page {
	pg := br.NewBackgroundTab(url)
	br.Tabs.SelectTabIndex(len(br.pages) - 1)
	return pg
}

// NewBackgroundTab adds a new tab without selecting it, opening the given
// URL in it, unless it is empty, and returns its page.
func (br *Browser) NewBackgroundTab(url string) *Page {
	fr, tab := br.Tabs.NewTab("New tab")
	pg := br.addPage(fr, tab)
	if url != "" {
		pg.OpenURL(url, nil)
	}
	return pg
}

// addPage adds a page to the given frame of the given tab.
func (br *Browser) addPage(fr *core.Frame, tab *core.Tab) *Page {
	fr.Styler(func(s *styles.Style) {
		s.Padding.Zero()
		s.Gap.Zero()
	})
	pg := NewPage(fr)
	// the toolbar is updated for the page of a tab when it is selected
	// (of the browser that it is in then)
	pg.OnShow(func(e events.Event) {
		if pg.browser != nil {
			pg.browser.opened()
		}
	})
	br.adoptPage(pg, tab, len(br.pages))
	if br.PageFunc != nil {
		br.PageFunc(pg)
	}
	return pg
}

// adoptPage makes the given page, of the given tab, a page of the browser,
// at the given index in its pages: a new page, or one whose tab was moved
// from another browser (see [Browser.tabMoved]).
func (br *Browser) adoptPage(pg *Page, tab *core.Tab, at int) {
	pg.browser = br
	br.pages = slices.Insert(br.pages, min(at, len(br.pages)), pg)
	pg.OpenNewTab = func(url string) { br.NewBackgroundTab(url) }
	pg.onConsole = br.Console.changed
	pg.onShowConsole = func() { br.ShowConsole(true) }
	if br.shown {
		pg.setShown()
	}
	pg.OnOpen = func(url string) {
		tab.SetText(tabLabel(pg, url)).SetTooltip(url)
		tab.Update()
		br.opened()
	}
	if pg.Current >= 0 && pg.Current < len(pg.History) {
		url := pg.History[pg.Current]
		tab.SetText(tabLabel(pg, url)).SetTooltip(url)
	}
}

// tabLabel returns the label of the tab of the given page, which is
// opening or has opened the given URL: its title, or else the URL.
func tabLabel(pg *Page, url string) string {
	if title := strings.Join(strings.Fields(pg.Title), " "); pg.loading == "" && title != "" {
		return title
	}
	if url == "" {
		return "New tab"
	}
	return pageName(url)
}

// pageName returns the name of the page at the given URL, for a tab
// without a title: its host and path, without the scheme, query, or
// fragment (as github.com/cogentcore/core), or the URL if it has none.
func pageName(url string) string {
	u, err := neturl.Parse(url)
	if err != nil || u.Host == "" {
		return url
	}
	return strings.TrimSuffix(u.Host+u.Path, "/")
}

// CurrentPage returns the page of the current tab, or nil if there is none.
func (br *Browser) CurrentPage() *Page {
	_, idx := br.Tabs.CurrentTab()
	if idx < 0 || idx >= len(br.pages) {
		return nil
	}
	return br.pages[idx]
}

// Pages returns the pages of the tabs, in their order.
func (br *Browser) Pages() []*Page {
	return br.pages
}

// MakeToolbar adds the toolbar items for the page of the current tab.
func (br *Browser) MakeToolbar(p *tree.Plan) {
	makeToolbar(p, br.CurrentPage)
}
