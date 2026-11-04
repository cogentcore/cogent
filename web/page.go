// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package web provides a cross-platform web browser.
package web

//go:generate core generate -add-types

import (
	"errors"
	"net/http"
	neturl "net/url"
	"strings"
	"sync"
	"sync/atomic"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/htmlcore"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
)

// Page represents one web browser page.
type Page struct {
	core.Frame

	// The history of URLs that have been visited. The oldest page is first.
	History []string `set:"-"`

	// Current is the index in [Page.History] of the page that is shown,
	// which is before the end of it after going back.
	Current int `set:"-"`

	// Context has the settings of the [htmlcore.Context] that each page is
	// read with (a new one for each page, made by [htmlcore.Context.NewPage]).
	Context *htmlcore.Context `set:"-"`

	// OnOpen, if set, is called with the URL of a page when it starts
	// opening and when it is opened (or stopped), as for updating a toolbar.
	OnOpen func(url string) `set:"-" display:"-"`

	// loading is the URL of the page that is being opened, if any.
	loading string

	// load is the number of the latest page that has started opening, so
	// that the opening of another one (and stopping) cancels earlier ones.
	load atomic.Int64

	// loads is the opening of pages in progress, for tests.
	loads sync.WaitGroup

	// shown is closed when the page is first shown, which the opening of
	// a page waits for before it updates the page (with AsyncLock).
	shown     chan struct{}
	shownOnce sync.Once
}

var _ tree.Node = (*Page)(nil)

func (pg *Page) Init() {
	pg.Frame.Init()
	pg.Context = htmlcore.NewContext()
	pg.Context.BrowserStyle = true
	pg.Context.OpenURL = pg.OpenURL
	pg.shown = make(chan struct{})
	pg.OnShow(func(e events.Event) {
		pg.shownOnce.Do(func() { close(pg.shown) })
	})
	pg.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
		s.Direction = styles.Column
	})
}

// historyMode is how the opening of a page changes the history.
type historyMode int

const (
	// historyNew adds the page after the current one, removing those
	// after it (that were gone back from).
	historyNew historyMode = iota

	// historyAt makes the page at the given index the current one,
	// as for going back, going forward, and reloading.
	historyAt
)

// errStopped is the error of resources that are not gotten after
// the opening of their page has been stopped.
var errStopped = errors.New("stopped")

// OpenURL opens the page at the given url, which is added to the history.
func (pg *Page) OpenURL(url string, e events.Event) {
	pg.open(url, historyNew, 0, 0)
}

// Back opens the previous page in the history, if there is one.
func (pg *Page) Back() {
	if pg.Current > 0 {
		pg.open(pg.History[pg.Current-1], historyAt, pg.Current-1, 0)
	}
}

// Forward opens the next page in the history, after going back.
func (pg *Page) Forward() {
	if pg.Current < len(pg.History)-1 {
		pg.open(pg.History[pg.Current+1], historyAt, pg.Current+1, 0)
	}
}

// Reload opens the current page again.
func (pg *Page) Reload() {
	if pg.Current < len(pg.History) {
		pg.open(pg.History[pg.Current], historyAt, pg.Current, 0)
	}
}

// Stop stops the opening of a page, keeping the page that is shown.
func (pg *Page) Stop() {
	if pg.loading == "" {
		return
	}
	pg.load.Add(1) // cancels it
	pg.loading = ""
	pg.opened(pg.currentURL())
}

// IsLoading returns whether a page is being opened.
func (pg *Page) IsLoading() bool {
	return pg.loading != ""
}

// currentURL returns the URL of the page that is shown, if any.
func (pg *Page) currentURL() string {
	if pg.Current < len(pg.History) {
		return pg.History[pg.Current]
	}
	return ""
}

// opened calls [Page.OnOpen] with the given url, if it is set.
func (pg *Page) opened(url string) {
	if pg.OnOpen != nil {
		pg.OnOpen(url)
	}
}

// maxRedirects is the largest number of meta refresh redirects
// that are followed from a page.
const maxRedirects = 5

// open opens the given url in another goroutine, with the given change to
// the history (at the given index for [historyAt]), after the given number
// of meta refresh redirects. It is read with a new context into a new frame,
// which replaces the content of the page when it has been read, unless the
// opening of another page has started (or it was stopped) by then.
func (pg *Page) open(url string, mode historyMode, index, redirects int) {
	gen := pg.load.Add(1)
	canceled := func() bool { return pg.load.Load() != gen }
	pg.loading = url
	pg.opened(url)
	ctx := pg.Context.NewPage()
	ctx.MediaWidth = pg.viewportWidth()
	get := ctx.GetURL
	ctx.GetURL = func(u string) (*http.Response, error) {
		if canceled() {
			return nil, errStopped
		}
		return get(u)
	}
	pg.loads.Add(1)
	go func() {
		defer pg.loads.Done()
		final := url
		content := core.NewFrame()
		content.SetName("content")
		content.Styler(func(s *styles.Style) {
			s.Grow.Set(1, 1)
			s.Direction = styles.Column
			s.Padding.Zero()
			s.Gap.Zero()
		})
		resp, err := htmlcore.Get(ctx, url)
		if err == nil {
			defer resp.Body.Close()
			final = resp.Request.URL.String()
			ctx.PageURL = final
			err = htmlcore.ReadHTML(ctx, content, resp.Body)
		}
		if canceled() {
			return
		}
		<-pg.shown // AsyncLock is not safe before then
		pg.AsyncLock()
		defer pg.AsyncUnlock()
		if canceled() {
			return
		}
		pg.loading = ""
		if err != nil {
			core.ErrorSnackbar(pg, err, "Error opening page")
			pg.opened(pg.currentURL())
			return
		}
		// a page that redirects (as to a version for browsers without
		// JavaScript) is replaced by the page that it redirects to
		if ref := ctx.RefreshURL; ref != "" && redirects < maxRedirects {
			if u, err := htmlcore.ParseRelativeURL(ref, final); err == nil && u.String() != final {
				pg.open(u.String(), mode, index, redirects+1)
				return
			}
		}
		switch mode {
		case historyNew:
			pg.History = append(pg.History[:min(pg.Current+1, len(pg.History))], final)
			pg.Current = len(pg.History) - 1
		case historyAt:
			pg.Current = index
			pg.History[index] = final
		}
		pg.Context.PageURL = final
		pg.DeleteChildren()
		pg.AddChild(content)
		pg.Update()
		if mode == historyNew {
			pg.scrollToTop()
		}
		pg.opened(final)
	}()
}

// scrollToTop scrolls the frames that the page is in (and the page) to
// the top, as for a new page.
func (pg *Page) scrollToTop() {
	for n := tree.Node(pg); n != nil; n = n.AsTree().Parent {
		if fr := core.AsFrame(n); fr != nil {
			fr.ScrollDimToContentStart(math32.Y)
			fr.ScrollDimToContentStart(math32.X)
		}
	}
}

// wait waits for the opening of pages to finish, for tests.
func (pg *Page) wait() {
	pg.loads.Wait()
}

func (pg *Page) MakeToolbar(p *tree.Plan) {
	tree.AddAt(p, "back", func(w *core.Button) {
		w.SetIcon(icons.ArrowBack).SetTooltip("Back")
		w.OnClick(func(e events.Event) {
			e.SetHandled()
			pg.Back()
		})
		w.Updater(func() {
			w.SetEnabled(pg.Current > 0)
		})
	})
	tree.AddAt(p, "forward", func(w *core.Button) {
		w.SetIcon(icons.ArrowForward).SetTooltip("Forward")
		w.OnClick(func(e events.Event) {
			e.SetHandled()
			pg.Forward()
		})
		w.Updater(func() {
			w.SetEnabled(pg.Current < len(pg.History)-1)
		})
	})
	tree.AddAt(p, "reload", func(w *core.Button) {
		w.OnClick(func(e events.Event) {
			e.SetHandled()
			if pg.IsLoading() {
				pg.Stop()
			} else {
				pg.Reload()
			}
		})
		w.Updater(func() {
			if pg.IsLoading() {
				w.SetIcon(icons.Close).SetTooltip("Stop loading this page")
			} else {
				w.SetIcon(icons.Refresh).SetTooltip("Reload this page")
			}
			w.SetEnabled(pg.IsLoading() || len(pg.History) > 0)
		})
	})
	tree.AddAt(p, "url", func(w *core.TextField) {
		w.SetPlaceholder("Address or search")
		w.Styler(func(s *styles.Style) {
			s.Grow.Set(1, 0)
			s.Min.X.Em(30)
		})
		w.Updater(func() {
			if pg.loading != "" {
				w.SetText(pg.loading)
			} else if u := pg.currentURL(); u != "" {
				w.SetText(u)
			}
		})
		w.OnChange(func(e events.Event) {
			text := strings.TrimSpace(w.Text())
			if text == "" || text == pg.currentURL() {
				return
			}
			if !strings.Contains(text, "://") {
				if strings.Contains(text, ".") && !strings.Contains(text, " ") {
					text = "https://" + text
				} else {
					text = "https://html.duckduckgo.com/html/?q=" + neturl.QueryEscape(text) // works without JavaScript
				}
			}
			pg.OpenURL(text, e)
		})
	})
}

// viewportWidth returns the width of the page in dp, which is the width of
// the viewport for the @media rules of its style sheets, or 0 (for the
// width of the screen) if it does not have one yet.
func (pg *Page) viewportWidth() float32 {
	dp := pg.Styles.UnitContext.Dots(units.UnitDp)
	if w := pg.Geom.Size.Alloc.Content.X; w > 0 && dp > 0 {
		return w / dp
	}
	if pg.Scene == nil {
		return 0
	}
	if rw := pg.Scene.RenderWindow(); rw != nil && rw.SystemWindow != nil && dp > 0 {
		return float32(rw.SystemWindow.Size().X) / dp
	}
	return 0
}
