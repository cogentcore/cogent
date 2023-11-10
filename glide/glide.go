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
	"goki.dev/glide/gidom"
	"goki.dev/goosi/events"
	"goki.dev/grr"
	"goki.dev/ki/v2"
)

// Page represents one web browser page
type Page struct {
	gi.Frame
}

// needed for interface import
var _ ki.Ki = (*Page)(nil)

func (pg *Page) OnInit() {
	pg.Frame.OnInit()
	pg.SetLayout(gi.LayoutVert)
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
	updt := pg.UpdateStart()
	pg.DeleteChildren(true)
	err = gidom.ReadHTML(pg, resp.Body, url)
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
	ch := tb.ChildByName("nav-bar").(*gi.Chooser)
	ch.AllowNew = true
	ch.ItemsFunc = nil
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
