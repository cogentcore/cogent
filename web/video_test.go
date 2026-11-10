// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"image"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/htmlcore"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// videoImages returns the images of the video elements of the page,
// by the id of their elements.
func videoImages(pg *Page) map[string]*core.Image {
	imgs := map[string]*core.Image{}
	pg.WalkDown(func(n tree.Node) bool {
		if img, ok := n.(*core.Image); ok && htmlcore.VideoSourceOf(img) != "" {
			imgs[htmlcore.GetAttr(htmlcore.NodeOf(img), "id")] = img
		}
		return tree.Continue
	})
	return imgs
}

// colorful returns whether the given image has more than a few colors
// (a frame of the test video, not a blank image).
func colorful(img image.Image) bool {
	if img == nil {
		return false
	}
	colors := map[[4]uint32]bool{}
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y += max(1, b.Dy()/16) {
		for x := b.Min.X; x < b.Max.X; x += max(1, b.Dx()/16) {
			r, g, bl, a := img.At(x, y).RGBA()
			colors[[4]uint32{r, g, bl, a}] = true
		}
	}
	return len(colors) > 4
}

// TestPageVideo tests that the videos of a page play in the images of their
// elements: one with autoplay at once, and the other, which shows its
// first frame, once it is clicked; that the video is requested with the
// cookies and user agent of the page; and that the videos stop when
// another page is opened.
func TestPageVideo(t *testing.T) {
	var mu sync.Mutex
	var agents, cookies []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "abc", Path: "/"})
			io.WriteString(w, `<html><body><video id="auto" autoplay muted loop width="320" src="/short.mp4"></video>
				<video id="click" muted><source src="/short.mp4" type="video/mp4">no video</video></body></html>`)
		case "/short.mp4":
			mu.Lock()
			agents = append(agents, r.UserAgent())
			cookies = append(cookies, r.Header.Get("Cookie"))
			mu.Unlock()
			http.ServeFile(w, r, "testdata/short.mp4")
		default:
			io.WriteString(w, `<html><body><p>another page</p></body></html>`)
		}
	}))
	defer srv.Close()

	b := core.NewBody()
	b.Styler(func(s *styles.Style) { s.Min.X.Dp(800); s.Min.Y.Dp(600) })
	pg := NewPage(b)
	pg.OpenURL(srv.URL+"/", nil)
	b.RenderImage(func() { settle(pg) }, func() {
		// the autoplay video shows its frames, at the width of its box
		var imgs map[string]*core.Image
		require.True(t, eventually(func() bool {
			b.AsyncLock()
			defer b.AsyncUnlock()
			imgs = videoImages(pg)
			return len(imgs) == 2 && colorful(imgs["auto"].Image) && colorful(imgs["click"].Image)
		}), "frames shown")
		b.AsyncLock()
		auto, click := pg.videos[htmlcore.NodeOf(imgs["auto"])], pg.videos[htmlcore.NodeOf(imgs["click"])]
		require.NotNil(t, auto)
		require.NotNil(t, click)
		assert.True(t, auto.playback.IsPlaying())
		assert.False(t, click.playback.IsPlaying(), "only its first frame")
		dp := imgs["auto"].Styles.UnitContext.Dots(units.UnitDp)
		assert.InDelta(t, 320*dp, imgs["auto"].Geom.Size.Actual.Content.X, 2)
		assert.InDelta(t, 240*dp, imgs["auto"].Geom.Size.Actual.Content.Y, 2, "of the aspect of the video")
		assert.InDelta(t, 64*dp, imgs["click"].Geom.Size.Actual.Content.X, 2, "of the size of the video")

		// a click plays the other one, and pauses the first
		imgs["click"].Send(events.Click)
		imgs["auto"].Send(events.Click)
		b.AsyncUnlock()
		require.True(t, eventually(click.playback.IsPlaying), "plays on a click")
		assert.True(t, auto.playback.IsPaused(), "pauses on a click")

		mu.Lock()
		require.NotEmpty(t, agents)
		assert.Equal(t, htmlcore.UserAgent, agents[0])
		assert.Contains(t, cookies[0], "session=abc") // (with those of other tests)
		mu.Unlock()

		// another page stops them
		pg.OpenURL(srv.URL+"/other", nil)
		settle(pg)
		b.AsyncLock()
		assert.Empty(t, pg.videos)
		b.AsyncUnlock()
		assert.False(t, click.playback.IsPlaying())
	})
}
