// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package web provides a cross-platform web browser.
package web

//go:generate core generate -add-types

import (
	neturl "net/url"
	"strings"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/htmlcore"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
)

// Page represents one web browser page.
type Page struct {
	core.Frame

	// The history of URLs that have been visited. The oldest page is first.
	History []string `set:"-"`

	// Context is the page's [htmlcore.Context].
	Context *htmlcore.Context `set:"-"`

	// OnOpen, if set, is called with the URL of each page that is opened,
	// as for updating a toolbar.
	OnOpen func(url string) `set:"-" display:"-"`
}

var _ tree.Node = (*Page)(nil)

func (pg *Page) Init() {
	pg.Frame.Init()
	pg.Context = htmlcore.NewContext()
	pg.Context.BrowserStyle = true
	pg.Context.OpenURL = pg.OpenURL
	pg.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
		s.Direction = styles.Column
	})
}

// OpenURL sets the content of the page from the given url,
// which is added to the history.
func (pg *Page) OpenURL(url string, e events.Event) {
	pg.openURL(url, true, 0)
}

// maxRedirects is the largest number of meta refresh redirects
// that are followed from a page.
const maxRedirects = 5

// openURL opens the given url, adding it to the history if addHistory,
// after the given number of meta refresh redirects.
func (pg *Page) openURL(url string, addHistory bool, redirects int) {
	resp, err := htmlcore.Get(pg.Context, url)
	if err != nil {
		core.ErrorSnackbar(pg, err, "Error opening page")
		return
	}
	defer resp.Body.Close()
	url = resp.Request.URL.String()
	pg.Context.PageURL = url
	if addHistory {
		pg.History = append(pg.History, url)
	} else if len(pg.History) > 0 {
		pg.History[len(pg.History)-1] = url
	}
	pg.DeleteChildren()
	pg.Context.RefreshURL = ""
	err = htmlcore.ReadHTML(pg.Context, pg, resp.Body)
	if err != nil {
		core.ErrorSnackbar(pg, err, "Error opening page")
		return
	}
	// a page that redirects (as to a version for browsers without
	// JavaScript) is replaced by the page that it redirects to
	if ref := pg.Context.RefreshURL; ref != "" && redirects < maxRedirects {
		if u, err := htmlcore.ParseRelativeURL(ref, url); err == nil && u.String() != url {
			pg.openURL(u.String(), false, redirects+1)
			return
		}
	}
	pg.Update()
	if pg.OnOpen != nil {
		pg.OnOpen(url)
	}
}

// Back opens the previous page in the history, if there is one.
func (pg *Page) Back() {
	if len(pg.History) < 2 {
		return
	}
	prev := pg.History[len(pg.History)-2]
	pg.History = pg.History[:len(pg.History)-1]
	pg.openURL(prev, false, 0)
}

func (pg *Page) MakeToolbar(p *tree.Plan) {
	tree.AddAt(p, "back", func(w *core.Button) {
		w.SetIcon(icons.ArrowBack).SetTooltip("Back")
		w.OnClick(func(e events.Event) {
			e.SetHandled()
			pg.Back()
		})
		w.Updater(func() {
			w.SetEnabled(len(pg.History) > 1)
		})
	})
	tree.AddAt(p, "url", func(w *core.TextField) {
		w.SetPlaceholder("Address or search")
		w.Styler(func(s *styles.Style) {
			s.Grow.Set(1, 0)
			s.Min.X.Em(30)
		})
		w.Updater(func() {
			if len(pg.History) > 0 {
				w.SetText(pg.History[len(pg.History)-1])
			}
		})
		w.OnChange(func(e events.Event) {
			text := strings.TrimSpace(w.Text())
			if text == "" || (len(pg.History) > 0 && text == pg.History[len(pg.History)-1]) {
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
