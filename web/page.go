// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package web provides a cross-platform web browser.
package web

//go:generate core generate -add-types

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/htmlcore"
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
}

var _ tree.Node = (*Page)(nil)

func (pg *Page) Init() {
	pg.Frame.Init()
	pg.Context = htmlcore.NewContext()
	pg.Context.OpenURL = pg.OpenURL
	pg.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
	})
}

// OpenURL sets the content of the page from the given url.
func (pg *Page) OpenURL(url string) {
	resp, err := htmlcore.Get(pg.Context, url)
	if err != nil {
		core.ErrorSnackbar(pg, err, "Error opening page")
		return
	}
	defer resp.Body.Close()
	url = resp.Request.URL.String()
	pg.Context.PageURL = url
	pg.History = append(pg.History, url)
	pg.DeleteChildren()
	err = htmlcore.ReadHTML(pg.Context, pg, resp.Body)
	if err != nil {
		core.ErrorSnackbar(pg, err, "Error opening page")
		return
	}
	pg.Update()
}

func (pg *Page) MakeToolbar(p *tree.Plan) {
	tree.AddInit(p, "back", func(w *core.Button) {
		w.OnClick(func(e events.Event) {
			if len(pg.History) > 1 {
				pg.OpenURL(pg.History[len(pg.History)-2])
				e.SetHandled()
			}
		})
	})

	// TODO(kai/abc)
	// ch := tb.AppChooser()
	// ch.AllowNew = true
	// ch.ItemsFunc = func() {
	// 	ch.Items = make([]any, len(pg.History))
	// 	for i, u := range pg.History {
	// 		// we reverse the order
	// 		ch.Items[len(pg.History)-i-1] = u
	// 	}
	// }
	// ch.OnChange(func(e events.Event) {
	// 	u, is := htmlcore.ParseURL(ch.CurLabel)
	// 	if is {
	// 		pg.OpenURL(u.String())
	// 	} else {
	// 		q := url.QueryEscape(ch.CurLabel)
	// 		pg.OpenURL("https://google.com/search?q=" + q)
	// 	}
	// 	e.SetHandled()
	// })
}
