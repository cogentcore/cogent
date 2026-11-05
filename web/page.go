// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package web provides a cross-platform web browser.
package web

//go:generate core generate -add-types

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	neturl "net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
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

	// OpenNewTab, if set, opens the given (absolute) URL in a new tab, for
	// links that are clicked with the meta or control key.
	OpenNewTab func(url string) `set:"-" display:"-"`

	// Title is the title of the page that is shown (of its title element).
	Title string `set:"-"`

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

	// media is what is needed to read the page that is shown again when the
	// width of its viewport crosses a boundary of its media (see
	// [Page.SizeFinal]), checkedWidth is the last width checked for that,
	// and mediaTimer is the pending read of it for that width, if any.
	media        *pageMedia
	checkedWidth float32
	mediaTimer   *time.Timer
}

// pageMedia is what is needed to read a page that is shown again for
// another width of its viewport: its context, HTML and resources.
type pageMedia struct {
	ctx   *htmlcore.Context
	html  []byte
	url   string // the final URL of the page
	cache *resourceCache
}

// mediaDelay is how long the width of the viewport of a page must stay
// across a boundary of its media before the page is read again for it,
// so that it is not read again for each step of resizing a window.
const mediaDelay = 250 * time.Millisecond

var _ tree.Node = (*Page)(nil)

func (pg *Page) Init() {
	pg.Frame.Init()
	pg.Context = htmlcore.NewContext()
	pg.Context.OpenURL = pg.OpenURL
	pg.shown = make(chan struct{})
	pg.OnShow(func(e events.Event) { pg.setShown() })
	pg.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
		s.Shrink.X = 1 // to the width of the frame it scrolls in, as a viewport
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
	// a link clicked with the meta or control key opens in a new tab,
	// as in browsers
	if e != nil && pg.OpenNewTab != nil && e.HasAnyModifier(key.Meta, key.Control) {
		if cur := pg.currentURL(); cur != "" {
			if u, err := htmlcore.ParseRelativeURL(url, cur); err == nil {
				url = u.String()
			}
		}
		pg.OpenNewTab(url)
		return
	}
	// a link to a fragment of the page that is shown scrolls to it
	if cur := pg.currentURL(); cur != "" && pg.loading == "" {
		if u, err := htmlcore.ParseRelativeURL(url, cur); err == nil && u.Fragment != "" {
			c, err := neturl.Parse(cur)
			if err == nil && u.Scheme == c.Scheme && u.Host == c.Host && u.Path == c.Path && u.RawQuery == c.RawQuery {
				pg.History = append(pg.History[:min(pg.Current+1, len(pg.History))], u.String())
				pg.Current = len(pg.History) - 1
				pg.scrollToFragment(u.Fragment)
				pg.opened(u.String())
				return
			}
		}
	}
	pg.open(url, historyNew, 0, 0)
}

// scrollToFragment scrolls to the widget of the element with the given id
// (or the a element with the given name), returning whether there is one.
func (pg *Page) scrollToFragment(id string) bool {
	if id == "" {
		return false
	}
	var target core.Widget
	pg.WalkDown(func(n tree.Node) bool {
		w, ok := n.(core.Widget)
		if !ok {
			return tree.Continue
		}
		wb := w.AsWidget()
		if pid, _ := wb.Property("id").(string); pid == id {
			target = w
			return tree.Break
		}
		if name, _ := wb.Property("name").(string); name == id && wb.Property("tag") == "a" {
			target = w
			return tree.Break
		}
		return tree.Continue
	})
	if target == nil {
		return false
	}
	target.AsWidget().ScrollThisToTop()
	return true
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
	if ctx.MediaWidth <= 0 { // before the page has a size, as it is read in another goroutine
		ctx.MediaWidth = htmlcore.ScreenMediaWidth()
	}
	cache := &resourceCache{get: ctx.GetURL}
	ctx.GetURL = func(u string) (*http.Response, error) {
		if canceled() {
			return nil, errStopped
		}
		return cache.Get(u)
	}
	pg.loads.Add(1)
	go func() {
		defer pg.loads.Done()
		final := url
		content := newPageContent()
		var page []byte
		resp, err := htmlcore.Get(ctx, url)
		if err == nil {
			final = resp.Request.URL.String()
			ctx.PageURL = final
			page, err = io.ReadAll(resp.Body)
			resp.Body.Close()
			if err == nil {
				err = htmlcore.ReadHTML(ctx, content, bytes.NewReader(page))
			}
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
		pg.Title = ctx.Title
		pg.setMedia(&pageMedia{ctx: ctx, html: page, url: final, cache: cache})
		pg.DeleteChildren()
		pg.AddChild(content)
		pg.Update()
		if u, err := neturl.Parse(final); err == nil && u.Fragment != "" {
			// after it is laid out, as for scrolling to its position
			pg.Defer(func() { pg.scrollToFragment(u.Fragment) })
		} else if mode == historyNew {
			pg.scrollToTop()
		}
		pg.opened(final)
	}()
}

// newPageContent returns a new frame for the content of a page.
func newPageContent() *core.Frame {
	content := core.NewFrame()
	content.SetName("content")
	content.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
		s.Direction = styles.Column
		s.Padding.Zero()
		s.Gap.Zero()
	})
	return content
}

// SizeFinal sizes the page, and checks whether the page needs to be read
// again for the width of its viewport, as when its window is resized:
// when the width crosses a boundary of its media, it is read again (with
// its resources from a cache) once the width has stayed there for
// [mediaDelay], as browsers apply the @media rules of the new width.
func (pg *Page) SizeFinal() {
	pg.Frame.SizeFinal()
	pg.checkMedia()
}

// checkMedia checks whether the page needs to be read again for the
// width of its viewport (see [Page.SizeFinal]).
func (pg *Page) checkMedia() {
	m := pg.media
	if m == nil || pg.loading != "" {
		return
	}
	w := pg.viewportWidth()
	if w <= 0 || w == pg.checkedWidth {
		return
	}
	pg.checkedWidth = w
	pg.stopMediaTimer() // for an earlier width
	if !m.ctx.MediaChanged(w) {
		return
	}
	pg.loads.Add(1)
	pg.mediaTimer = time.AfterFunc(mediaDelay, func() {
		defer pg.loads.Done()
		if !pg.AsyncLockIfShown() {
			return
		}
		defer pg.AsyncUnlock()
		if pg.media == m && pg.loading == "" {
			pg.mediaTimer = nil
			pg.readMedia(m, pg.checkedWidth)
		}
	})
}

// stopMediaTimer stops the pending read of the page for another width, if any.
func (pg *Page) stopMediaTimer() {
	if pg.mediaTimer != nil && pg.mediaTimer.Stop() {
		pg.loads.Done()
	}
	pg.mediaTimer = nil
}

// setMedia sets what is needed to read the page that is shown again
// for another width of its viewport.
func (pg *Page) setMedia(m *pageMedia) {
	pg.stopMediaTimer()
	pg.media = m
	pg.checkedWidth = m.ctx.MediaWidth
}

// readMedia reads the page that is shown again for the given width of
// its viewport, from the given media of it, in another goroutine. It
// replaces the page when it has been read, unless the opening of another
// page has started by then.
func (pg *Page) readMedia(m *pageMedia, width float32) {
	gen := pg.load.Add(1)
	canceled := func() bool { return pg.load.Load() != gen }
	ctx := pg.Context.NewPage()
	ctx.MediaWidth = width
	ctx.PageURL = m.url
	ctx.GetURL = func(u string) (*http.Response, error) {
		if canceled() {
			return nil, errStopped
		}
		return m.cache.Get(u)
	}
	pg.loads.Add(1)
	go func() {
		defer pg.loads.Done()
		content := newPageContent()
		err := htmlcore.ReadHTML(ctx, content, bytes.NewReader(m.html))
		if canceled() || err != nil {
			return
		}
		if !pg.AsyncLockIfShown() {
			return
		}
		defer pg.AsyncUnlock()
		if canceled() {
			return
		}
		pg.setMedia(&pageMedia{ctx: ctx, html: m.html, url: m.url, cache: m.cache})
		pg.DeleteChildren()
		pg.AddChild(content)
		pg.Update()
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

// setShown records that the page (or the window that it is in, as for a
// tab in the background) has been shown, so that pages can be opened in it.
func (pg *Page) setShown() {
	pg.shownOnce.Do(func() { close(pg.shown) })
}

// wait waits for the opening of pages to finish, for tests.
func (pg *Page) wait() {
	pg.loads.Wait()
}

// MakeToolbar adds the toolbar items for the page (back, forward,
// reload or stop, and the address field) to the given plan.
func (pg *Page) MakeToolbar(p *tree.Plan) {
	makeToolbar(p, func() *Page { return pg })
}

// makeToolbar adds the toolbar items for the page that the given function
// returns, which is the current one when they are used (as for tabs),
// or nil if there is none.
func makeToolbar(p *tree.Plan, page func() *Page) {
	tree.AddAt(p, "back", func(w *core.Button) {
		w.SetIcon(icons.ArrowBack).SetTooltip("Back")
		w.OnClick(func(e events.Event) {
			e.SetHandled()
			if pg := page(); pg != nil {
				pg.Back()
			}
		})
		w.Updater(func() {
			pg := page()
			w.SetEnabled(pg != nil && pg.Current > 0)
		})
	})
	tree.AddAt(p, "forward", func(w *core.Button) {
		w.SetIcon(icons.ArrowForward).SetTooltip("Forward")
		w.OnClick(func(e events.Event) {
			e.SetHandled()
			if pg := page(); pg != nil {
				pg.Forward()
			}
		})
		w.Updater(func() {
			pg := page()
			w.SetEnabled(pg != nil && pg.Current < len(pg.History)-1)
		})
	})
	tree.AddAt(p, "reload", func(w *core.Button) {
		w.OnClick(func(e events.Event) {
			e.SetHandled()
			pg := page()
			switch {
			case pg == nil:
			case pg.IsLoading():
				pg.Stop()
			default:
				pg.Reload()
			}
		})
		w.Updater(func() {
			pg := page()
			if pg != nil && pg.IsLoading() {
				w.SetIcon(icons.Close).SetTooltip("Stop loading this page")
			} else {
				w.SetIcon(icons.Refresh).SetTooltip("Reload this page")
			}
			w.SetEnabled(pg != nil && (pg.IsLoading() || len(pg.History) > 0))
		})
	})
	tree.AddAt(p, "url", func(w *core.TextField) {
		w.SetPlaceholder("Address or search")
		w.Styler(func(s *styles.Style) {
			s.Grow.Set(1, 0)
			s.Min.X.Em(30)
			s.Max.X.Ch(80)
		})
		w.Updater(func() {
			pg := page()
			switch {
			case pg == nil:
				w.SetText("")
			case pg.loading != "":
				w.SetText(pg.loading)
			default:
				w.SetText(pg.currentURL())
			}
		})
		w.OnChange(func(e events.Event) {
			pg := page()
			text := strings.TrimSpace(w.Text())
			if pg == nil || text == "" || text == pg.currentURL() {
				return
			}
			pg.OpenURL(LocationURL(text), e)
		})
	})
}

// LocationURL returns the URL for the given location, as typed in the
// address field or given on the command line: a URL (with a scheme), a
// local file (an absolute or relative path, or one that exists), as a
// file:// URL, a domain (with a dot), as https, or otherwise a search.
func LocationURL(loc string) string {
	loc = strings.TrimSpace(loc)
	if strings.Contains(loc, "://") || strings.HasPrefix(loc, "about:") || strings.HasPrefix(loc, "data:") {
		return loc
	}
	path := loc
	if rest, ok := strings.CutPrefix(path, "~/"); ok {
		if home, err := os.UserHomeDir(); err == nil {
			path = filepath.Join(home, rest)
		}
	}
	isPath := filepath.IsAbs(path) || strings.HasPrefix(loc, "./") || strings.HasPrefix(loc, "../") || strings.HasPrefix(loc, "~/")
	if _, err := os.Stat(path); err == nil {
		isPath = true
	}
	if isPath {
		if abs, err := filepath.Abs(path); err == nil {
			return (&neturl.URL{Scheme: "file", Path: filepath.ToSlash(abs)}).String()
		}
	}
	if strings.Contains(loc, ".") && !strings.Contains(loc, " ") {
		return "https://" + loc
	}
	return "https://html.duckduckgo.com/html/?q=" + neturl.QueryEscape(loc) // works without JavaScript
}

// viewportWidth returns the width of the page in dp, which is the width of
// the viewport for the @media rules of its style sheets, or 0 (for the
// width of the screen) if it does not have one yet. As in browsers, it
// includes the vertical scrollbar of the frame that it scrolls in, so
// that the scrollbar does not change it (which, as a page can be read
// again for another width, could make it read again and again).
func (pg *Page) viewportWidth() float32 {
	dp := pg.Styles.UnitContext.Dots(units.UnitDp)
	if w := pg.Geom.Size.Alloc.Content.X; w > 0 && dp > 0 {
		if fr := core.AsFrame(pg.Parent); fr != nil && fr.HasScroll[math32.Y] {
			w += math32.Ceil(fr.Styles.ScrollbarWidth.Dots)
		}
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
