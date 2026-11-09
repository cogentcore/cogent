// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"github.com/stretchr/testify/assert"
)

func TestBrowserTabs(t *testing.T) {
	b := core.NewBody()
	br := NewBrowser(b)
	br.PageFunc = func(pg *Page) {
		pg.Context.GetURL = func(u string) (*http.Response, error) {
			pu, _ := url.Parse(u)
			r := strings.NewReader(`<html><head><title>Page ` + pu.Path + `</title></head><body><p>x</p></body></html>`)
			return &http.Response{StatusCode: 200, Status: "200 OK", Request: &http.Request{URL: pu}, Body: io.NopCloser(r)}, nil
		}
	}
	one := br.NewTab("https://example.com/one")
	two := br.NewTab("https://example.com/two")
	wait := func() {
		for _, pg := range br.Pages() {
			pg.wait()
		}
	}
	locked := func(f func()) func() {
		return func() {
			b.AsyncLock()
			defer b.AsyncUnlock()
			f()
		}
	}
	b.RenderImage(wait, locked(func() {
		assert.Equal(t, two, br.CurrentPage(), "a new tab is selected")
		_, tab := br.Tabs.TabAtIndex(0)
		assert.Equal(t, "Page /one", tab.Text, "tabs have the titles of their pages")
		br.Tabs.SelectTabIndex(0)
		assert.Equal(t, one, br.CurrentPage())
		br.Tabs.DeleteTabIndex(0)
		assert.Equal(t, []*Page{two}, br.Pages())
		assert.Equal(t, two, br.CurrentPage())
	}))
}

// TestBrowserNewTabClick tests that a link clicked with the meta key
// opens in a new tab, without selecting it.
func TestBrowserNewTabClick(t *testing.T) {
	b := core.NewBody()
	br := NewBrowser(b)
	br.PageFunc = func(pg *Page) {
		pg.Context.GetURL = func(u string) (*http.Response, error) {
			pu, _ := url.Parse(u)
			r := strings.NewReader(`<p>x</p>`)
			return &http.Response{StatusCode: 200, Status: "200 OK", Request: &http.Request{URL: pu}, Body: io.NopCloser(r)}, nil
		}
	}
	one := br.NewTab("https://example.com/one")
	wait := func() {
		for _, pg := range br.Pages() {
			pg.wait()
		}
	}
	b.RenderImage(wait, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		e := events.NewKey(events.KeyDown, 0, 0, key.Modifiers(0))
		e.Mods.SetFlag(true, key.Meta)
		one.OpenURL("two", e)
	}, wait, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		if assert.Len(t, br.Pages(), 2) {
			assert.Equal(t, one, br.CurrentPage(), "the new tab is not selected")
			assert.Equal(t, []string{"https://example.com/two"}, br.Pages()[1].History)
			assert.Equal(t, []string{"https://example.com/one"}, one.History)
		}
	})
}

// TestTabLabelEmptyTitle tests that the tab of a page with a blank title
// shows the name of the page (its host and path).
func TestTabLabelEmptyTitle(t *testing.T) {
	pg := NewPage()
	pg.Title = " \n "
	assert.Equal(t, "github.com/cogentcore/core", tabLabel(pg, "https://github.com/cogentcore/core?tab=readme#top"))
	pg.Title = " GitHub  home "
	assert.Equal(t, "GitHub home", tabLabel(pg, "https://github.com/"))
	pg.Title = ""
	assert.Equal(t, "github.com", tabLabel(pg, "https://github.com/"))
}
