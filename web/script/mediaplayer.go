// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package script

import (
	"github.com/dop251/goja"
	"golang.org/x/net/html"
)

// MediaPlayer plays the video elements of a page for its scripts, as the
// player of the app (as with FFmpeg): their play, pause, currentTime,
// volume and muted act on it, and it sends the events of the playing to
// them ([Engine.MediaEvents]), with its state, which their properties are
// (as paused, currentTime and duration). Players of web pages (as video.js)
// play HLS (.m3u8) by setting it as the source when canPlayType says that
// it can be played, as in Safari, and the player plays it. Its methods are
// called in the tasks of the scripts, with the lock of the page held (which
// the tasks take for them).
type MediaPlayer interface {
	// MediaLoad opens the media of the given video element (its source),
	// which sends its loadedmetadata (or error) event once it is opened.
	MediaLoad(n *html.Node)

	// MediaPlay plays the media of the given video element, returning an
	// error if it cannot be played.
	MediaPlay(n *html.Node) error

	// MediaPause pauses the media of the given video element.
	MediaPause(n *html.Node)

	// MediaSeek seeks the media of the given video element to the given
	// time in seconds.
	MediaSeek(n *html.Node, t float64)

	// MediaTime returns the current time in seconds of the media of the
	// given video element, as it plays (for its currentTime between its
	// timeupdate events, which players check often, as for stalls), or -1
	// if it is not known.
	MediaTime(n *html.Node) float64

	// MediaVolume sets the volume (0 to 1) of the media of the given video
	// element, and whether it is muted.
	MediaVolume(n *html.Node, volume float64, muted bool)
}

// MediaState is the state of the playing of a video element, which is sent
// with its events ([Engine.MediaEvents]).
type MediaState struct {
	// Paused is whether it is paused (or not played yet), and Ended
	// whether it has played to its end.
	Paused, Ended bool

	// CurrentTime and Duration are in seconds (Duration is 0 if not known).
	CurrentTime, Duration float64

	// Width and Height are the size of the video, once it is opened.
	Width, Height int

	// Volume is the volume, from 0 to 1, and Muted whether it is muted.
	Volume float64
	Muted  bool

	// Error is the error of opening or playing it, if any.
	Error string
}

// MediaEvents dispatches the given events (as "play", "timeupdate") to the
// given video element in a task, after setting its state to the given one.
func (e *Engine) MediaEvents(n *html.Node, st MediaState, types ...string) {
	e.Post(func() {
		e.callPrelude("mediaEvents", e.nodes.id(n), map[string]any{
			"paused": st.Paused, "ended": st.Ended, "currentTime": st.CurrentTime,
			"duration": st.Duration, "width": st.Width, "height": st.Height,
			"volume": st.Volume, "muted": st.Muted, "error": st.Error,
		}, types)
	})
}

// setMediaPlayerFuncs sets the D functions of the media player (see
// [MediaPlayer]), which do nothing without one.
func (e *Engine) setMediaPlayerFuncs(set func(name string, f func(call goja.FunctionCall) goja.Value)) {
	vm := e.vm
	mp := e.opts.Media
	// the node of the element of the call, with the lock of the page for
	// the player (see [Engine.lockTask]), or nil without a player or lock
	node := func(call goja.FunctionCall) *html.Node {
		if mp == nil || !e.lockTask() {
			return nil
		}
		return e.nodes.node(int(call.Argument(0).ToInteger()))
	}
	set("hasMediaPlayer", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(mp != nil)
	})
	set("mediaLoad", func(call goja.FunctionCall) goja.Value {
		if n := node(call); n != nil {
			mp.MediaLoad(n)
		}
		return goja.Undefined()
	})
	set("mediaPlay", func(call goja.FunctionCall) goja.Value {
		if n := node(call); n != nil {
			if err := mp.MediaPlay(n); err != nil {
				return vm.ToValue(err.Error())
			}
			return vm.ToValue("")
		}
		return vm.ToValue("media playback is not supported")
	})
	set("mediaPause", func(call goja.FunctionCall) goja.Value {
		if n := node(call); n != nil {
			mp.MediaPause(n)
		}
		return goja.Undefined()
	})
	set("mediaTime", func(call goja.FunctionCall) goja.Value {
		if n := node(call); n != nil {
			return vm.ToValue(mp.MediaTime(n))
		}
		return vm.ToValue(-1)
	})
	set("mediaSeek", func(call goja.FunctionCall) goja.Value {
		if n := node(call); n != nil {
			mp.MediaSeek(n, call.Argument(1).ToFloat())
		}
		return goja.Undefined()
	})
	set("mediaVolume", func(call goja.FunctionCall) goja.Value {
		if n := node(call); n != nil {
			mp.MediaVolume(n, call.Argument(1).ToFloat(), call.Argument(2).ToBoolean())
		}
		return goja.Undefined()
	})
}
