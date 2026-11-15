// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/htmlcore"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/video"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// audioServer serves the given page at /, the test audio (1.5 seconds of
// a tone, MP3) at /tone.mp3, and the same audio at /live.mp3 as a live
// radio stream does (as Icecast): without a length.
func audioServer(t *testing.T, page string) *httptest.Server {
	data, err := os.ReadFile("testdata/tone.mp3")
	require.NoError(t, err)
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			io.WriteString(w, page)
		case "/live.mp3":
			w.Header().Set("Content-Type", "audio/mpeg")
			for i := 0; i < len(data); i += 1024 { // (chunked, without a length)
				w.Write(data[i:min(i+1024, len(data))])
				w.(http.Flusher).Flush()
			}
		default:
			http.ServeFile(w, r, "testdata"+r.URL.Path)
		}
	}))
}

// openAudioPage opens a page of the given server, calling test with an
// evaluator of scripts in it once it is shown.
func openAudioPage(t *testing.T, srv *httptest.Server, test func(b *core.Body, pg *Page, eval func(src string) any)) {
	b := core.NewBody()
	b.Styler(func(s *styles.Style) { s.Min.X.Dp(800); s.Min.Y.Dp(600) })
	pg := NewPage(b)
	pg.OpenURL(srv.URL+"/", nil)
	core.RenderImage(b, func() { settle(pg) }, func() {
		eval := func(src string) any {
			v, err := pg.scripts.engine.RunString(src)
			require.NoError(t, err)
			return v
		}
		test(b, pg, eval)
	})
}

// audioLog is a script that logs the events of the audio a (once each)
// in the text of the element with the id log.
const audioLog = `const log = [];
	for (const t of ["loadedmetadata", "play", "playing", "timeupdate", "pause", "ended", "error"]) {
		a.addEventListener(t, () => {
			if (!log.includes(t)) log.push(t);
			document.getElementById("log").textContent = log.join(",");
		});
	}`

// TestPageAudioScript tests that new Audio(url).play() of a script of a page
// plays the audio, which is not in the document, sending the events of the
// playing to it, whose state (paused, currentTime, duration) is that of the
// playing, and that canPlayType says that MP3 can be played (which players,
// as jPlayer of npr.org, require).
func TestPageAudioScript(t *testing.T) {
	srv := audioServer(t, `<html><body><p id="log">none</p><script>
		const a = new Audio("/tone.mp3");
		a.muted = true;
		`+audioLog+`
		a.play();
	</script></body></html>`)
	defer srv.Close()
	openAudioPage(t, srv, func(b *core.Body, pg *Page, eval func(src string) any) {
		assert.Equal(t, "maybe", eval(`a.canPlayType("audio/mpeg;")`))
		assert.Equal(t, "maybe", eval(`document.createElement("audio").canPlayType('audio/mp4; codecs="mp4a.40.2"')`))
		assert.Equal(t, srv.URL+"/tone.mp3", eval("a.src"))
		assert.Equal(t, "auto", eval("a.preload"))
		assert.Equal(t, true, eval("a instanceof HTMLAudioElement"))
		assert.Equal(t, false, eval("a.isConnected"))

		// it plays, with the events, as long as its audio
		require.True(t, eventually(func() bool {
			return strings.Contains(pageTexts(b, pg), "loadedmetadata,play,playing,timeupdate")
		}), pageTexts(b, pg))
		assert.Equal(t, false, eval("a.paused"))
		assert.InDelta(t, 1.5, eval("a.duration"), 0.1)
		require.True(t, eventually(func() bool { return eval("a.currentTime").(float64) > 0.2 }), "its time moves")

		// paused by the script, and played again to its end
		eval("a.pause()")
		assert.Equal(t, true, eval("a.paused"))
		paused := eval("a.currentTime").(float64)
		require.True(t, eventually(func() bool { return strings.Contains(pageTexts(b, pg), "pause") }), pageTexts(b, pg))
		assert.InDelta(t, paused, eval("a.currentTime"), 0.15)
		eval("a.play()")
		require.True(t, eventually(func() bool { return strings.Contains(pageTexts(b, pg), "ended") }), pageTexts(b, pg))
		assert.Equal(t, true, eval("a.ended"))
		assert.InDelta(t, 1.5, eval("a.currentTime"), 0.2)
		assert.NotContains(t, pageTexts(b, pg), "error")
	})
}

// TestPageAudioStream tests that a script plays a live audio stream (MP3
// over HTTP without a length, as the stream of npr.org), as jPlayer of
// npr.org does (it sets the src of an audio element, loads, and plays it),
// whose duration is Infinity, as in browsers.
func TestPageAudioStream(t *testing.T) {
	srv := audioServer(t, `<html><body><p id="log">none</p><div id="player"></div><script>
		const a = document.createElement("audio");
		a.id = "jp_audio_0";
		a.preload = "metadata";
		document.getElementById("player").appendChild(a);
		a.muted = true;
		`+audioLog+`
		if (a.canPlayType("audio/mpeg;")) {
			a.src = "/live.mp3";
			a.load();
			a.play();
		}
	</script></body></html>`)
	defer srv.Close()
	openAudioPage(t, srv, func(b *core.Body, pg *Page, eval func(src string) any) {
		require.True(t, eventually(func() bool {
			return strings.Contains(pageTexts(b, pg), "loadedmetadata,play,playing,timeupdate")
		}), pageTexts(b, pg))
		assert.True(t, math.IsInf(eval("a.duration").(float64), 1), "the duration of a live stream")
		assert.Empty(t, videoBoxes(pg), "an audio element without controls is not displayed")
		require.True(t, eventually(func() bool { return strings.Contains(pageTexts(b, pg), "ended") }), pageTexts(b, pg))
	})
}

// TestPageAudioControls tests that an audio element with controls is a bar
// with the controls of an audio player, which play it.
func TestPageAudioControls(t *testing.T) {
	srv := audioServer(t, `<html><body><p id="log">none</p>
		<audio id="a" controls muted src="/tone.mp3"></audio>
		<script>const a = document.getElementById("a"); `+audioLog+`</script></body></html>`)
	defer srv.Close()
	openAudioPage(t, srv, func(b *core.Body, pg *Page, eval func(src string) any) {
		var box *htmlcore.VideoBox
		var c *video.Controls
		require.True(t, eventually(func() bool {
			b.AsyncLock()
			defer b.AsyncUnlock()
			box = videoBoxes(pg)["a"]
			if box == nil {
				return false
			}
			for _, k := range box.Children {
				if kc, ok := k.(*video.Controls); ok {
					c = kc
				}
			}
			return c != nil
		}), "the bar of the audio, with its controls")
		assert.True(t, box.Audio)
		b.AsyncLock()
		pv := pg.videos[htmlcore.NodeOf(box)]
		assert.True(t, c.IsShown())
		b.AsyncUnlock()
		require.NotNil(t, pv)
		require.True(t, eventually(func() bool { return pv.Duration() > 0 }), "opened")

		// its play button plays it (as a click on the bar), to its end
		b.AsyncLock()
		pv.TogglePlay()
		c.Refresh()
		b.AsyncUnlock()
		require.True(t, eventually(func() bool {
			return strings.Contains(pageTexts(b, pg), "loadedmetadata,play,playing,timeupdate")
		}), pageTexts(b, pg))
		b.AsyncLock()
		assert.True(t, c.IsShown(), "always shown")
		b.AsyncUnlock()
		require.True(t, eventually(func() bool { return strings.Contains(pageTexts(b, pg), "ended") }), pageTexts(b, pg))
	})
}

// TestPageAudioAutoplay tests that an audio element of a page with autoplay
// and no controls (which has no box), which no script uses, plays, as in
// browsers.
func TestPageAudioAutoplay(t *testing.T) {
	srv := audioServer(t, `<html><body><p>page</p><audio autoplay muted src="/tone.mp3"></audio><script>window.x = 1</script></body></html>`)
	defer srv.Close()
	openAudioPage(t, srv, func(b *core.Body, pg *Page, eval func(src string) any) {
		require.True(t, eventually(func() bool {
			b.AsyncLock()
			defer b.AsyncUnlock()
			for _, pv := range pg.videos {
				pv.mu.Lock()
				pb := pv.playback
				pv.mu.Unlock()
				if pv.audio() && pb != nil && pb.IsPlaying() {
					return true
				}
			}
			return false
		}), "the audio plays")
	})
}
