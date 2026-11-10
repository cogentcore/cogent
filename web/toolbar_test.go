// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
)

// TestToolbar tests the toolbar of a page in a top bar, as in the web app.
func TestToolbar(t *testing.T) {
	b := core.NewBody()
	pg := NewPage(b)
	pg.Context.GetURL = func(u string) (*http.Response, error) {
		pu, _ := url.Parse(u)
		r := strings.NewReader(`<p>A page</p>`)
		return &http.Response{StatusCode: 200, Status: "200 OK", Request: &http.Request{URL: pu}, Body: io.NopCloser(r)}, nil
	}
	b.AddTopBar(func(bar *core.Frame) {
		tb := core.NewToolbar(bar)
		tb.Maker(pg.MakeToolbar)
		pg.OnOpen = func(url string) { tb.Update() }
	})
	pg.OpenURL("https://example.com/", nil)
	var back, forward, reload *core.Button
	var addr *core.TextField
	b.RenderImage(pg.wait, func() {
		b.Scene.WalkDown(func(n tree.Node) bool {
			switch w := n.(type) {
			case *core.Button:
				switch w.Name {
				case "back":
					back = w
				case "forward":
					forward = w
				case "reload":
					reload = w
				}
			case *core.TextField:
				if w.Name == "url" {
					addr = w
				}
			}
			return tree.Continue
		})
	})
	if assert.NotNil(t, back) && assert.NotNil(t, addr) {
		assert.Equal(t, "https://example.com/", addr.Text())
		assert.True(t, back.StateIs(states.Disabled), "no page to go back to")
	}
	if assert.NotNil(t, forward) && assert.NotNil(t, reload) {
		assert.True(t, forward.StateIs(states.Disabled), "no page to go forward to")
		assert.False(t, reload.StateIs(states.Disabled))
		assert.Equal(t, "Reload this page", reload.Tooltip)
	}
}

// TestHistory tests going back and forward, reloading, and stopping.
func TestHistory(t *testing.T) {
	b := core.NewBody()
	pg := NewPage(b)
	var mu sync.Mutex // for gets, which is counted in the goroutines of opening
	gets := 0
	block := make(chan struct{})
	pg.Context.GetURL = func(u string) (*http.Response, error) {
		pu, _ := url.Parse(u)
		if strings.Contains(u, "slow") {
			<-block
		}
		mu.Lock()
		gets++
		mu.Unlock()
		r := strings.NewReader(`<p>Page ` + pu.Path + `</p>`)
		return &http.Response{StatusCode: 200, Status: "200 OK", Request: &http.Request{URL: pu}, Body: io.NopCloser(r)}, nil
	}
	pg.OpenURL("https://example.com/one", nil)
	// locked runs the given steps with the lock, as event handlers do.
	locked := func(f func()) func() {
		return func() {
			b.AsyncLock()
			defer b.AsyncUnlock()
			f()
		}
	}
	n := 0
	b.RenderImage(pg.wait, locked(func() {
		pg.OpenURL("https://example.com/two", nil)
	}), pg.wait, locked(func() {
		assert.Equal(t, []string{"https://example.com/one", "https://example.com/two"}, pg.History)
		assert.Equal(t, 1, pg.Current)
		pg.Back()
	}), pg.wait, locked(func() {
		assert.Equal(t, 0, pg.Current)
		assert.Len(t, pg.History, 2, "going back keeps the pages after")
		pg.Forward()
	}), pg.wait, locked(func() {
		assert.Equal(t, 1, pg.Current)
		mu.Lock()
		n = gets
		mu.Unlock()
		pg.Reload()
	}), pg.wait, locked(func() {
		mu.Lock()
		assert.Equal(t, n+1, gets, "reloading gets the page again")
		mu.Unlock()
		pg.OpenURL("https://example.com/slow", nil)
		assert.True(t, pg.IsLoading())
		pg.Stop()
		assert.False(t, pg.IsLoading())
		close(block)
	}), pg.wait, locked(func() {
		assert.Equal(t, 1, pg.Current, "the stopped page is not opened")
		assert.Len(t, pg.History, 2)
		pg.Back()
	}), pg.wait, locked(func() {
		pg.OpenURL("https://example.com/three", nil)
	}), pg.wait, locked(func() {
		assert.Equal(t, []string{"https://example.com/one", "https://example.com/three"}, pg.History,
			"opening a page after going back replaces the pages after")
	}))
}

// TestFragment tests that a link to a fragment of the page that is shown
// does not open it again.
func TestFragment(t *testing.T) {
	b := core.NewBody()
	pg := NewPage(b)
	var mu sync.Mutex
	gets := 0
	pg.Context.GetURL = func(u string) (*http.Response, error) {
		mu.Lock()
		gets++
		mu.Unlock()
		pu, _ := url.Parse(u)
		r := strings.NewReader(`<p>top</p><div style="height:3000px"></div><h2 id="sec">Section</h2><div style="height:3000px"></div><div><h2 id="inl" style="display: inline">Inline</h2> <a href="#e">edit</a></div>`)
		return &http.Response{StatusCode: 200, Status: "200 OK", Request: &http.Request{URL: pu}, Body: io.NopCloser(r)}, nil
	}
	pg.OpenURL("https://example.com/page", nil)
	b.RenderImage(pg.wait, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		assert.True(t, pg.scrollToFragment("sec"))
		assert.True(t, pg.scrollToFragment("inl"), "an inline element, without a widget of its own (as the headings of Wikipedia)")
		assert.False(t, pg.scrollToFragment("none"))
		pg.OpenURL("#sec", nil)
		assert.Equal(t, []string{"https://example.com/page", "https://example.com/page#sec"}, pg.History)
		assert.False(t, pg.IsLoading())
	}, pg.wait, func() {
		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, 1, gets, "the page is not gotten again")
	})
}

// TestToolbarLongURL tests that the address field is one line for a long
// URL (as of a search with all of its parameters, which made it many
// lines high).
func TestToolbarLongURL(t *testing.T) {
	b := core.NewBody()
	pg := NewPage(b)
	pg.Context.GetURL = func(u string) (*http.Response, error) {
		pu, _ := url.Parse(u)
		return &http.Response{StatusCode: 200, Status: "200 OK", Request: &http.Request{URL: pu}, Body: io.NopCloser(strings.NewReader(`<p>A page</p>`))}, nil
	}
	b.AddTopBar(func(bar *core.Frame) {
		tb := core.NewToolbar(bar)
		tb.Maker(pg.MakeToolbar)
		pg.OnOpen = func(url string) { tb.Update() }
	})
	long := "https://www.google.com/search?hl=nl&source=hp&biw=&bih=&q=ebikes&iflsig=" + strings.Repeat("AHSINFcAAAAAavL9Etj4wy55EX7LBKYo1QOS3rvbcsNP", 4) + "&gbv=2&btnG=Google+Zoeken"
	pg.OpenURL(long, nil)
	var addr *core.TextField
	b.RenderImage(pg.wait, func() {
		b.Scene.WalkDown(func(n tree.Node) bool {
			if w, ok := n.(*core.TextField); ok && w.Name == "url" {
				addr = w
			}
			return tree.Continue
		})
		if assert.NotNil(t, addr) {
			assert.Equal(t, long, addr.Text())
			lh := addr.Styles.LineHeightDots()
			assert.Less(t, addr.Geom.Size.Actual.Content.Y, 1.6*lh, "one line")
		}
	})
}
