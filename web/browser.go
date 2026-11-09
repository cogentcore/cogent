// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	neturl "net/url"
	"slices"
	"strings"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
)

// Browser is a web browser with tabs, each of which has a [Page].
// Its toolbar ([Browser.MakeToolbar]) is for the page of the current tab.
type Browser struct {
	core.Frame

	// Tabs has the tabs of the pages.
	Tabs *core.Tabs `set:"-"`

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
	br.Tabs = core.NewTabs(br)
	br.Tabs.SetType(core.FunctionalTabs)
	br.Tabs.NewTabButton = true
	br.Tabs.NewTabFunc = func(index int) {
		fr, tab := br.Tabs.TabAtIndex(index)
		br.addPage(fr, tab)
	}
	br.Tabs.CloseTabFunc = func(index int) {
		if index < len(br.pages) {
			br.pages[index].Stop()
			br.pages = slices.Delete(br.pages, index, index+1)
		}
		br.opened()
	}
}

// opened calls [Browser.OnOpen], if it is set.
func (br *Browser) opened() {
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
	br.pages = append(br.pages, pg)
	pg.OpenNewTab = func(url string) { br.NewBackgroundTab(url) }
	if br.shown {
		pg.setShown()
	}
	if br.PageFunc != nil {
		br.PageFunc(pg)
	}
	pg.OnOpen = func(url string) {
		tab.SetText(tabLabel(pg, url)).SetTooltip(url)
		tab.Update()
		br.opened()
	}
	// the toolbar is updated for the page of a tab when it is selected
	pg.OnShow(func(e events.Event) { br.opened() })
	return pg
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
