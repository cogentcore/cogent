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
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
)

// TestBrowserNoHorizontalScroll tests that a page that is taller than
// the window, with text that wraps, has no horizontal scrollbar in its tab:
// the page is the width of the frame it scrolls in (as a viewport),
// less its vertical scrollbar.
func TestBrowserNoHorizontalScroll(t *testing.T) {
	for _, width := range []float32{1400, 1000} {
		b := core.NewBody()
		b.Styler(func(s *styles.Style) {
			s.Min.X.Dp(width)
			s.Min.Y.Dp(600)
		})
		br := NewBrowser(b)
		b.AddTopBar(func(bar *core.Frame) {
			tb := core.NewToolbar(bar)
			tb.Maker(br.MakeToolbar)
			br.OnOpen = func() { tb.Update() }
		})
		br.PageFunc = func(pg *Page) {
			pg.Context.GetURL = func(u string) (*http.Response, error) {
				pu, _ := url.Parse(u)
				para := "<p>" + strings.Repeat("words that wrap in the width of the page ", 40) + "</p>\n"
				r := strings.NewReader("<html><body><h1>Long page</h1>" + strings.Repeat(para, 30) + "</body></html>")
				return &http.Response{StatusCode: 200, Status: "200 OK", Request: &http.Request{URL: pu}, Body: io.NopCloser(r)}, nil
			}
		}
		pg := br.NewTab("https://example.com/")
		b.RenderImage(pg.wait, func() {
			b.AsyncLock()
			defer b.AsyncUnlock()
			vertical := false
			b.WalkDown(func(n tree.Node) bool {
				if fr, ok := n.(*core.Frame); ok {
					assert.False(t, fr.HasScroll[0], "horizontal scroll of %s at %g", fr.Path(), width)
					vertical = vertical || fr.HasScroll[1]
				}
				return true
			})
			assert.True(t, vertical, "the page scrolls vertically at %g", width)
		})
	}
}

// TestBrowserFullHeightNoScroll tests that a page of the height of its
// viewport (html, body and a flex column of height: 100%, with a footer at
// its bottom, as google.com) does not scroll in the tab of a browser with
// a toolbar: its percentage heights are of the viewport of the page, not
// of the window, which with the toolbar and tabs is higher, so that the
// footer was below the bottom of the page, which always had a scrollbar.
// It is read again when the height of its viewport is known, as it uses it.
func TestBrowserFullHeightNoScroll(t *testing.T) {
	b := core.NewBody()
	b.Styler(func(s *styles.Style) {
		s.Min.X.Dp(900)
		s.Min.Y.Dp(600)
	})
	br := NewBrowser(b)
	b.AddTopBar(func(bar *core.Frame) {
		tb := core.NewToolbar(bar)
		tb.Maker(br.MakeToolbar)
		br.OnOpen = func() { tb.Update() }
	})
	br.PageFunc = func(pg *Page) {
		pg.Scripts = false
		pg.Context.GetURL = func(u string) (*http.Response, error) {
			pu, _ := url.Parse(u)
			r := strings.NewReader(`<html style="height: 100%; margin: 0"><body style="height: 100%; margin: 0"><div style="display: flex; flex-direction: column; height: 100%">` +
				`<div style="height: 60px">header</div><div style="flex-grow: 1">main</div><div id="foot" style="height: 40px">footer</div></div></body></html>`)
			return &http.Response{StatusCode: 200, Status: "200 OK", Request: &http.Request{URL: pu}, Body: io.NopCloser(r)}, nil
		}
	}
	pg := br.NewTab("https://example.com/")
	b.RenderImage(func() {
		pg.wait()
		time.Sleep(2 * mediaDelay) // (read again for the height of its viewport)
		pg.wait()
	}, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		b.WalkDown(func(n tree.Node) bool {
			if n == tree.Node(br.Console) {
				return false // (hidden)
			}
			if fr, ok := n.(*core.Frame); ok {
				assert.False(t, fr.HasScroll[1], "vertical scroll of %s", fr.Path())
			}
			return true
		})
		var foot *core.WidgetBase
		pg.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
			if cwb.Property("id") == "foot" {
				foot = cwb
			}
			return true
		})
		if assert.NotNil(t, foot) {
			fr := core.AsWidget(pg.Parent).Geom.TotalBBox
			assert.InDelta(t, fr.Max.Y, foot.Geom.TotalBBox.Max.Y, 2, "the footer is at the bottom of the viewport")
		}
	})
}
