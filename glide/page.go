// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package glide provides a web browser
package glide

//go:generate core generate -add-types

import (
	"net/url"

	"cogentcore.org/cogent/glide/gidom"
	"cogentcore.org/core/events"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/grr"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/styles"
)

// Page represents one web browser page
type Page struct {
	gi.Frame

	// The history of URLs that have been visited. The oldest page is first.
	History []string

	// PageURL is the current page URL
	PageURL string
}

var _ ki.Ki = (*Page)(nil)

func (pg *Page) OnInit() {
	pg.Frame.OnInit()
	pg.Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})
}

// OpenURL sets the content of the page from the given url.
func (pg *Page) OpenURL(url string) error {
	resp, err := gidom.Get(pg.Context(), url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	url = resp.Request.URL.String()
	pg.PageURL = url
	pg.History = append(pg.History, url)
	updt := pg.UpdateStart()
	pg.DeleteChildren(true)
	err = gidom.ReadHTML(pg.Context(), pg, resp.Body)
	if err != nil {
		return err
	}
	pg.Update()
	pg.UpdateEndLayout(updt)
	return nil
}

// AppBar is the default app bar for a [Page]
func (pg *Page) AppBar(tb *gi.Toolbar) {
	back := tb.ChildByName("back").(*gi.Button)
	back.OnClick(func(e events.Event) {
		if len(pg.History) > 1 {
			pg.OpenURL(pg.History[len(pg.History)-2])
		}
	})

	ch := tb.ChildByName("app-chooser").(*gi.AppChooser)
	ch.AllowNew = true
	ch.ItemsFunc = func() {
		ch.Items = make([]any, len(pg.History))
		for i, u := range pg.History {
			// we reverse the order
			ch.Items[len(pg.History)-i-1] = u
		}
	}
	ch.OnChange(func(e events.Event) {
		u, is := gidom.ParseURL(ch.CurLabel)
		if is {
			grr.Log(pg.OpenURL(u.String()))
		} else {
			q := url.QueryEscape(ch.CurLabel)
			grr.Log(pg.OpenURL("https://google.com/search?q=" + q))
		}
		e.SetHandled()
	})
}
