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
