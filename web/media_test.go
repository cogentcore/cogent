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

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
)

// mediaPage is a page whose text is blue in a viewport of at most 600px.
const mediaPage = `<html><head><style>p { color: red }
@media (max-width: 600px) { p { color: blue } }</style></head><body><p>Text</p></body></html>`

// TestPageMediaResize tests that a page is read again when the width of
// its viewport crosses a boundary of its media, with its resources from
// the cache, and not when it changes within the same range.
func TestPageMediaResize(t *testing.T) {
	for _, narrow := range []float32{500, 700} {
		width := float32(800)
		b := core.NewBody()
		b.Styler(func(s *styles.Style) { s.Min.X.Dp(900); s.Min.Y.Dp(400) })
		fr := core.NewFrame(b)
		fr.Styler(func(s *styles.Style) {
			s.Min.X.Dp(width)
			s.Max.X.Dp(width)
			s.Grow.Set(0, 1)
			s.Overflow.Y = styles.OverflowAuto
		})
		pg := NewPage(fr)
		gets := 0
		pg.Context.GetURL = func(u string) (*http.Response, error) {
			gets++
			pu, _ := url.Parse(u)
			return &http.Response{StatusCode: 200, Status: "200 OK", Request: &http.Request{URL: pu},
				Body: io.NopCloser(strings.NewReader(mediaPage))}, nil
		}
		// textColor returns the color of the text of the page.
		textColor := func() string {
			b.AsyncLock()
			defer b.AsyncUnlock()
			cl := ""
			pg.WalkDown(func(n tree.Node) bool {
				if tx, ok := n.(*core.Text); ok && tx.Text == "Text" {
					cl = colors.AsHex(colors.ToUniform(tx.Styles.Color))
				}
				return tree.Continue
			})
			return cl
		}
		var wide, after string
		var readWidth float32
		b.RenderImage(func() {
			pg.OpenURL("https://example.com/", nil)
			pg.wait()
		}, func() {
			wide = textColor()
			b.AsyncLock()
			width = narrow
			fr.Update()
			b.AsyncUnlock()
		}, func() {
			// the page is read again after mediaDelay
			time.Sleep(2 * mediaDelay)
			pg.wait()
		}, func() {
			after = textColor()
			b.AsyncLock()
			readWidth = pg.media.ctx.MediaWidth
			b.AsyncUnlock()
		})
		assert.Equal(t, "#FF0000", wide, "red in the wide viewport")
		assert.Equal(t, 1, gets, "the page is gotten once, then from the cache")
		if narrow < 600 {
			assert.Equal(t, "#0000FF", after, "blue in the narrow viewport")
			assert.InDelta(t, narrow, readWidth, 1, "read again for the narrow width")
		} else {
			assert.Equal(t, "#FF0000", after, "still red")
			assert.InDelta(t, 800, readWidth, 1, "not read again within the same range")
		}
	}
}
