// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package glide provides a web browser
package glide

import (
	"fmt"
	"net/http"
	"net/url"

	"goki.dev/gi/v2/gi"
	"goki.dev/girl/styles"
	"goki.dev/glide/gidom"
	"goki.dev/goosi/events"
	"goki.dev/grr"
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
)

// Page represents one web browser page
type Page struct {
	gi.Frame

	// The history of URLs that have been visited. The oldest page is first.
	History []string

	// PgURL is the current page URL
	PgURL string

	// PageStyles contains the accumulated global CSS styles for the page
	PageStyles string
}

// needed for interface import
var _ ki.Ki = (*Page)(nil)

func (pg *Page) OnInit() {
	pg.Frame.OnInit()
	pg.Style(func(s *styles.Style) {
		s.MainAxis = mat32.Y
	})
	gidom.OpenURLFunc = func(url string) {
		grr.Log0(pg.OpenURL(url))
	}
}

// OpenURL sets the content of the page from the given url.
func (pg *Page) OpenURL(url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("got error status %q", resp.Status)
	}
	pg.PgURL = url
	pg.History = append(pg.History, url)
	updt := pg.UpdateStart()
	pg.DeleteChildren(true)
	err = gidom.ReadHTML(pg, pg, resp.Body)
	if err != nil {
		return err
	}
	pg.Update()
	pg.UpdateEndLayout(updt)
	return nil
}

// TopAppBar is the default [gi.TopAppBar] for a [Page]
func (pg *Page) TopAppBar(tb *gi.TopAppBar) {
	gi.DefaultTopAppBarStd(tb)

	back := tb.ChildByName("back").(*gi.Button)
	back.OnClick(func(e events.Event) {
		if len(pg.History) > 1 {
			pg.OpenURL(pg.History[len(pg.History)-2])
		}
	})

	ch := tb.ChildByName("nav-bar").(*gi.Chooser)
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
			grr.Log0(pg.OpenURL(u.String()))
		} else {
			q := url.QueryEscape(ch.CurLabel)
			grr.Log0(pg.OpenURL("https://google.com/search?q=" + q))
		}
		e.SetHandled()
	})
}

/////////////////// Context Interface

// PageURL returns the current page URL
func (pg *Page) PageURL() string {
	return pg.PgURL
}

// SetStyle adds the given CSS styles to the page's styles.
func (pg *Page) SetStyle(style string) {
	pg.PageStyles += style
}

// GetStyle returns the page's styles in CSS.
func (pg *Page) GetStyle() string {
	return pg.PageStyles
}
