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
