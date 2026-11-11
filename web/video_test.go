// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"image"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/htmlcore"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/video"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// videoBoxes returns the boxes of the video elements of the page,
// by the id of their elements.
func videoBoxes(pg *Page) map[string]*htmlcore.VideoBox {
	boxes := map[string]*htmlcore.VideoBox{}
	pg.WalkDown(func(n tree.Node) bool {
		if box, ok := n.(*htmlcore.VideoBox); ok && htmlcore.VideoSourceOf(box) != "" {
			boxes[htmlcore.GetAttr(htmlcore.NodeOf(box), "id")] = box
		}
		return tree.Continue
	})
	return boxes
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
		var boxes map[string]*htmlcore.VideoBox
		require.True(t, eventually(func() bool {
			b.AsyncLock()
			defer b.AsyncUnlock()
			boxes = videoBoxes(pg)
			return len(boxes) == 2 && colorful(boxes["auto"].Image.Image) && colorful(boxes["click"].Image.Image)
		}), "frames shown")
		b.AsyncLock()
		auto, click := pg.videos[htmlcore.NodeOf(boxes["auto"])], pg.videos[htmlcore.NodeOf(boxes["click"])]
		require.NotNil(t, auto)
		require.NotNil(t, click)
		assert.True(t, auto.playback.IsPlaying())
		assert.False(t, click.playback.IsPlaying(), "only its first frame")
		dp := boxes["auto"].Styles.UnitContext.Dots(units.UnitDp)
		assert.InDelta(t, 320*dp, boxes["auto"].Geom.Size.Actual.Content.X, 2)
		assert.InDelta(t, 240*dp, boxes["auto"].Geom.Size.Actual.Content.Y, 2, "of the aspect of the video")
		assert.InDelta(t, 64*dp, boxes["click"].Geom.Size.Actual.Content.X, 2, "of the size of the video")
		// the image fills the box (within the rounding of absolute sizes)
		bsz, isz := boxes["auto"].Geom.Size.Actual.Content, boxes["auto"].Image.Geom.Size.Actual.Total
		assert.InDelta(t, bsz.X, isz.X, 0.5, "the image fills the box")
		assert.InDelta(t, bsz.Y, isz.Y, 0.5, "the image fills the box")

		// the standard controls are over each video
		for _, id := range []string{"auto", "click"} {
			var c *video.Controls
			for _, k := range boxes[id].Children {
				if cc, ok := k.(*video.Controls); ok {
					c = cc
				}
			}
			require.NotNil(t, c, id)
			assert.True(t, c.IsShown() || id == "auto", id+": shown while it does not play")
		}

		// a click plays the other one, and pauses the first
		boxes["click"].Send(events.Click)
		boxes["auto"].Send(events.Click)
		b.AsyncUnlock()
		require.True(t, eventually(click.playback.IsPlaying), "plays on a click")
		assert.True(t, auto.playback.IsPaused(), "pauses on a click")

		// a page video is a player of the controls: it seeks and sets its volume
		click.Seek(500 * time.Millisecond)
		require.True(t, eventually(func() bool { return click.CurrentTime() >= 500*time.Millisecond }), "seeks")
		assert.InDelta(t, 1.0, click.Duration().Seconds(), 0.05)
		click.SetVolume(0.5)
		assert.Equal(t, float32(0.5), click.Volume())
		assert.True(t, click.IsMuted())
		click.SetMuted(false)
		assert.False(t, click.IsMuted())

		mu.Lock()
		require.NotEmpty(t, agents)
		assert.Equal(t, htmlcore.UserAgent, agents[0])
		assert.Contains(t, cookies[0], "session=abc") // (with those of other tests)
		mu.Unlock()

		// another page stops them
		b.AsyncLock() // (as the event loop does)
		pg.OpenURL(srv.URL+"/other", nil)
		b.AsyncUnlock()
		settle(pg)
		b.AsyncLock()
		assert.Empty(t, pg.videos)
		b.AsyncUnlock()
		assert.False(t, click.playback.IsPlaying())
	})
}

// TestPageVideoHLS tests that a script of a page plays an HLS video as the
// players of web pages (as video.js) do in Safari: canPlayType says that
// HLS can be played, so it sets the playlist (.m3u8) as the source of a
// video element and plays it, which the player of the page (FFmpeg) plays,
// sending the events of the playing to the element, whose state (paused,
// currentTime, duration) is that of the playing, and which it pauses.
func TestPageVideoHLS(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			io.WriteString(w, `<html><body><video id="v" muted style="width: 320px"></video><p id="log">none</p>
				<script>
					const v = document.getElementById("v");
					const log = [];
					for (const t of ["loadedmetadata", "play", "playing", "timeupdate", "pause", "ended"]) {
						v.addEventListener(t, () => {
							if (!log.includes(t)) log.push(t);
							document.getElementById("log").textContent = log.join(",");
						});
					}
					if (v.canPlayType("application/vnd.apple.mpegurl")) {
						v.src = "/hls/short.m3u8";
						v.play();
					} else {
						document.getElementById("log").textContent = "no hls";
					}
				</script></body></html>`)
			return
		}
		http.ServeFile(w, r, "testdata"+r.URL.Path)
	}))
	defer srv.Close()

	b := core.NewBody()
	b.Styler(func(s *styles.Style) { s.Min.X.Dp(800); s.Min.Y.Dp(600) })
	pg := NewPage(b)
	pg.OpenURL(srv.URL+"/", nil)
	b.RenderImage(func() { settle(pg) }, func() {
		// it plays, shown in the box of the element, with the events
		require.True(t, eventually(func() bool {
			return strings.Contains(pageTexts(b, pg), "loadedmetadata,play,playing,timeupdate")
		}), pageTexts(b, pg))
		require.True(t, eventually(func() bool {
			b.AsyncLock()
			defer b.AsyncUnlock()
			box := videoBoxes(pg)["v"]
			return box != nil && colorful(box.Image.Image)
		}), "frames shown")

		// one controls bar, after the parts of the page were read again
		b.AsyncLock()
		bars := 0
		for _, k := range videoBoxes(pg)["v"].Children {
			if _, ok := k.(*video.Controls); ok {
				bars++
			}
		}
		b.AsyncUnlock()
		assert.Equal(t, 1, bars)

		// its state is that of the playing
		eval := func(src string) any {
			v, err := pg.scripts.engine.RunString(src)
			require.NoError(t, err)
			return v
		}
		assert.Equal(t, false, eval("v.paused"))
		assert.InDelta(t, 1.0, eval("v.duration"), 0.05)
		assert.Equal(t, int64(64), eval("v.videoWidth"))
		assert.Greater(t, eval("v.currentTime"), 0.0)

		// a script pauses it
		eval("v.pause()")
		assert.Equal(t, true, eval("v.paused"))
		b.AsyncLock()
		pv := pg.videos[htmlcore.NodeOf(videoBoxes(pg)["v"])]
		b.AsyncUnlock()
		require.True(t, eventually(func() bool { return !pv.IsPlaying() }), "paused")

		// played again to its end
		eval("v.play()")
		require.True(t, eventually(func() bool { return strings.Contains(pageTexts(b, pg), "ended") }), pageTexts(b, pg))
		assert.Equal(t, true, eval("v.ended"))
	})
}

// TestPageVideoMSE tests that a script of a page plays a video with Media
// Source Extensions, as the players of web pages (as hls.js) do: it
// appends fragmented MP4 (of the video and of the audio, in pieces, as it
// fetches them) to the SourceBuffers of a MediaSource whose object URL is
// the source of the video, which the player of the page plays.
func TestPageVideoMSE(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			io.WriteString(w, `<html><body><video id="v" muted style="width: 320px"></video><p id="log">none</p>
				<script>
					const v = document.getElementById("v");
					const log = [];
					const add = t => { if (!log.includes(t)) log.push(t); document.getElementById("log").textContent = log.join(","); };
					for (const t of ["loadedmetadata", "play", "playing", "ended"]) v.addEventListener(t, () => add(t));
					const ms = new MediaSource();
					v.src = URL.createObjectURL(ms);
					ms.addEventListener("sourceopen", async () => {
						add("sourceopen");
						URL.revokeObjectURL(v.src);
						ms.duration = 1;
						const vb = ms.addSourceBuffer('video/mp4; codecs="avc1.64000a"');
						const ab = ms.addSourceBuffer('audio/mp4; codecs="mp4a.40.2"');
						const append = (sb, data) => new Promise(r => { sb.addEventListener("updateend", r, { once: true }); sb.appendBuffer(data); });
						const [vd, ad] = await Promise.all(["/frag-video.mp4", "/frag-audio.mp4"].map(u => fetch(u).then(r => r.arrayBuffer())));
						for (let i = 0; i < vd.byteLength; i += 1000) await append(vb, vd.slice(i, i + 1000));
						for (let i = 0; i < ad.byteLength; i += 1000) await append(ab, ad.slice(i, i + 1000));
						add("buffered " + v.buffered.length + " " + v.buffered.end(0).toFixed(1));
						ms.endOfStream();
						v.play();
					});
					if (!MediaSource.isTypeSupported('video/mp4; codecs="avc1.64000a"') || MediaSource.isTypeSupported('video/webm; codecs="vp9"')) add("bad types");
				</script></body></html>`)
			return
		}
		http.ServeFile(w, r, "testdata"+r.URL.Path)
	}))
	defer srv.Close()

	b := core.NewBody()
	b.Styler(func(s *styles.Style) { s.Min.X.Dp(800); s.Min.Y.Dp(600) })
	pg := NewPage(b)
	pg.OpenURL(srv.URL+"/", nil)
	b.RenderImage(func() { settle(pg) }, func() {
		require.True(t, eventually(func() bool {
			return strings.Contains(pageTexts(b, pg), "playing")
		}), pageTexts(b, pg))
		txt := pageTexts(b, pg)
		assert.Contains(t, txt, "sourceopen")
		assert.Contains(t, txt, "buffered 1 1.0")
		assert.NotContains(t, txt, "bad types")
		require.True(t, eventually(func() bool {
			b.AsyncLock()
			defer b.AsyncUnlock()
			box := videoBoxes(pg)["v"]
			return box != nil && colorful(box.Image.Image)
		}), "frames shown")
		eval := func(src string) any {
			v, err := pg.scripts.engine.RunString(src)
			require.NoError(t, err)
			return v
		}
		assert.Equal(t, int64(64), eval("v.videoWidth"))
		assert.InDelta(t, 1.0, eval("v.duration"), 0.01)
		require.True(t, eventually(func() bool { return strings.Contains(pageTexts(b, pg), "ended") }), pageTexts(b, pg))

		// a seek plays it from the time, of its fragments
		eval("v.currentTime = 0.5; v.play()")
		require.True(t, eventually(func() bool { return eval("v.currentTime").(float64) > 0.55 }), "plays from the seek")
	})
}
