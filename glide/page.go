// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package glide provides a web browser
package glide

//go:generate core generate -add-types

import (
	"cogentcore.org/core/coredom"
	"cogentcore.org/core/events"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
)

// Page represents one web browser page
type Page struct {
	gi.Frame

	// The history of URLs that have been visited. The oldest page is first.
	History []string `set:"-"`

	// Context is the page's [coredom.Context].
	Context *coredom.Context `set:"-"`
}

var _ tree.Node = (*Page)(nil)

func (pg *Page) OnInit() {
	pg.Frame.OnInit()
	pg.Context = coredom.NewContext()
	pg.Context.OpenURL = pg.OpenURL
	pg.Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})
}

// OpenURL sets the content of the page from the given url.
func (pg *Page) OpenURL(url string) {
	resp, err := coredom.Get(pg.Context, url)
	if err != nil {
		gi.ErrorSnackbar(pg, err, "Error opening page")
		return
	}
	defer resp.Body.Close()
	url = resp.Request.URL.String()
	pg.Context.PageURL = url
	pg.History = append(pg.History, url)
	pg.DeleteChildren()
	err = coredom.ReadHTML(pg.Context, pg, resp.Body)
	if err != nil {
		gi.ErrorSnackbar(pg, err, "Error opening page")
		return
	}
	pg.Update()
}

// AppBar is the default app bar for a [Page]
func (pg *Page) AppBar(tb *gi.Toolbar) {
	back := tb.ChildByName("back").(*gi.Button)
	back.OnClick(func(e events.Event) {
		if len(pg.History) > 1 {
			pg.OpenURL(pg.History[len(pg.History)-2])
		}
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
	// 	u, is := coredom.ParseURL(ch.CurLabel)
	// 	if is {
	// 		pg.OpenURL(u.String())
	// 	} else {
	// 		q := url.QueryEscape(ch.CurLabel)
	// 		pg.OpenURL("https://google.com/search?q=" + q)
	// 	}
	// 	e.SetHandled()
	// })
}
