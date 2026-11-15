// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package script

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
)

// fakePlayer is a media player (see [MediaPlayer]) that records the calls
// to it.
type fakePlayer struct {
	mu    sync.Mutex
	calls []string

	// playErr is the error of playing.
	playErr error
}

func (p *fakePlayer) record(format string, args ...any) {
	p.mu.Lock()
	p.calls = append(p.calls, fmt.Sprintf(format, args...))
	p.mu.Unlock()
}

func (p *fakePlayer) MediaLoad(n *html.Node) { p.record("load") }
func (p *fakePlayer) MediaPlay(n *html.Node) error {
	p.record("play")
	return p.playErr
}
func (p *fakePlayer) MediaPause(n *html.Node)           { p.record("pause") }
func (p *fakePlayer) MediaSeek(n *html.Node, t float64) { p.record("seek %v", t) }
func (p *fakePlayer) MediaTime(n *html.Node) float64    { return 1.5 }
func (p *fakePlayer) MediaVolume(n *html.Node, v float64, m bool) {
	p.record("volume %v %v", v, m)
}

// has returns whether the player had the given call.
func (p *fakePlayer) has(call string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, c := range p.calls {
		if c == call {
			return true
		}
	}
	return false
}

// TestMediaPlayer tests that the video elements of the page are played by
// the player, which sends their events with their states.
func TestMediaPlayer(t *testing.T) {
	mp := &fakePlayer{}
	e, _ := testPageEngine(t, `<html><body><video id="v"></video></body></html>`, nil, func(o *Options) { o.Media = mp })
	run(t, e, `
		window.events = [];
		const v = document.getElementById("v");
		for (const ev of ["play", "timeupdate", "pause", "error"]) v.addEventListener(ev, () => events.push(ev));
		v.src = "/video.mp4";
		window.played = "";
		v.play().then(() => { played = "ok" }, err => { played = err.name });
		v.currentTime = 3;
		v.volume = 0.5;
		v.muted = true;`)
	e.Wait()
	v := findByID(e.Document(), "v")
	e.MediaEvents(v, MediaState{CurrentTime: 2, Duration: 10, Width: 64, Height: 48, Volume: 0.5}, "play", "timeupdate")
	e.Wait()
	assert.Equal(t, "timeupdate,play,timeupdate", run(t, e, `events.join()`), "the seek, then the events of the player")
	assert.Equal(t, int64(10), run(t, e, `document.getElementById("v").duration`))
	run(t, e, `document.getElementById("v").pause()`)
	e.Wait()
	for _, c := range []string{"load", "play", "seek 3", "volume 0.5 false", "volume 0.5 true", "pause"} {
		assert.True(t, mp.has(c), c)
	}
	mp.playErr = errors.New("no codec")
	run(t, e, `window.played2 = ""; document.getElementById("v").play().catch(err => { played2 = err.message })`)
	assert.Eventually(t, func() bool {
		v, _ := e.RunString(`played2`)
		return strings.Contains(fmt.Sprint(v), "no codec")
	}, 5*time.Second, 10*time.Millisecond)
}

// TestMediaSourceScripts tests the MediaSource of the scripts, played by
// the player: its buffers, which media is appended to (as players append
// what they fetch), removed from, and aborted, its duration, and its
// streams for the player.
func TestMediaSourceScripts(t *testing.T) {
	data, err := os.ReadFile("../testdata/frag-video.mp4")
	require.NoError(t, err)
	mp := &fakePlayer{}
	e, _ := testPageEngine(t, `<html><body><video id="v"></video></body></html>`, map[string]string{"/frag.mp4": string(data)}, func(o *Options) { o.Media = mp })
	assert.Equal(t, true, run(t, e, `MediaSource.isTypeSupported('video/mp4; codecs="avc1.64000a"')`))
	assert.Equal(t, false, run(t, e, `MediaSource.isTypeSupported("video/x-unknown")`))
	run(t, e, `
		window.log = [];
		const v = document.getElementById("v");
		const ms = new MediaSource();
		window.ms = ms;
		v.src = URL.createObjectURL(ms);
		ms.addEventListener("sourceopen", async () => {
			if (ms.sourceBuffers.length) return; // (opened again)
			const sb = ms.addSourceBuffer('video/mp4; codecs="avc1.64000a"');
			window.sb = sb;
			const buf = await (await fetch("/frag.mp4")).arrayBuffer();
			sb.timestampOffset = 0;
			sb.addEventListener("updateend", () => log.push("updateend " + sb.buffered.length + " " + sb.buffered.end(0).toFixed(1)), { once: true });
			sb.appendBuffer(new Uint8Array(buf));
			ms.duration = 1;
		});`)
	require.Eventually(t, func() bool {
		v, _ := e.RunString(`log.join()`)
		return v == "updateend 1 1.0"
	}, 5*time.Second, 10*time.Millisecond)
	u := run(t, e, `document.getElementById("v").src`).(string)
	ms := e.MediaSource(u)
	require.NotNil(t, ms)
	assert.Equal(t, time.Second, ms.Duration())
	assert.Nil(t, e.MediaSource("blob:none"))

	run(t, e, `sb.remove(0, 0.3)`)
	e.Wait()
	time.Sleep(20 * time.Millisecond)
	run(t, e, `sb.abort(); ms.endOfStream()`)
	rs, err := ms.Streams(0)
	require.NoError(t, err)
	require.Len(t, rs, 1)
	got, err := io.ReadAll(rs[0])
	require.NoError(t, err)
	assert.Less(t, len(got), len(data), "without the removed fragment")
	assert.Greater(t, len(got), 0)

	run(t, e, `ms.removeSourceBuffer(sb); URL.revokeObjectURL(document.getElementById("v").src)`)
	e.Wait()
	_, err = ms.Streams(0)
	assert.Error(t, err, "no buffers, ended")
	assert.Equal(t, "InvalidStateError", run(t, e, `try { new MediaSource().duration = 1 } catch (err) { err.name }`), "not open")
}
