// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"errors"
	"image"
	"log/slog"
	"net/url"
	"strings"
	"sync"
	"time"

	"cogentcore.org/cogent/web/script"
	"cogentcore.org/core/core"
	"cogentcore.org/core/htmlcore"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/video"
	"golang.org/x/net/html"
)

// The videos of a page: htmlcore makes each video element a box
// ([htmlcore.VideoBox]) with an image of its poster, with the URL of its
// video (see htmlcore.VideoSourceOf), and the page plays the video in the
// image with a [video.Playback], which FFmpeg decodes (from its URL, with
// the cookies and user agent of the page), with the standard controls of
// players over it ([video.Controls], for which a video of a page is a
// [video.Player]). The video of an element is kept while the element is
// in the document, as the widgets of the page are read again (as after its
// scripts change it). A video plays at first if it has autoplay, and
// otherwise shows its poster, or its first frame; a click on it plays and
// pauses it.

// maxVideoWidth is the maximum width in pixels to which the frames of a
// video are decoded before its box is laid out.
const maxVideoWidth = 1280

// pageVideo is the video of a video element of a page.
type pageVideo struct {
	// page is the page of the video, whose lock is that of its widgets
	// (which are added to the scene of the page after they are made).
	page *Page

	// node is the video element.
	node *html.Node

	// src is the URL of the video.
	src string

	// playback plays the video, once its media is opened.
	playback *video.Playback

	mu sync.Mutex

	// box is the current box of the video element, in whose image the
	// frames are shown (a new one after each reading of the page), and
	// controls are its controls.
	box      *htmlcore.VideoBox
	controls *video.Controls

	// frame is the last frame shown, to show in a new box.
	frame *image.RGBA

	// size is the size of the video, once it is opened.
	size image.Point

	// opened is closed once the media is opened (or failed to open).
	opened chan struct{}

	// playWanted is whether the video is to play once it is opened (as
	// by the play of a script before then).
	playWanted bool

	// lastTimeUpdate is when the last timeupdate event was sent to the
	// scripts of the page (see [pageVideo.notify]).
	lastTimeUpdate time.Time

	// openError is the error of opening the media, if any.
	openError string

	// closed is whether the video has been closed.
	closed bool
}

// attachVideos attaches the videos of the video elements of the given
// content to their boxes in it, making the videos of new elements, and
// closes the videos of the elements that are no longer in it.
func (pg *Page) attachVideos(content *core.Frame) {
	seen := map[*html.Node]bool{}
	content.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
		box, ok := cw.(*htmlcore.VideoBox)
		if !ok {
			return tree.Continue
		}
		src := htmlcore.VideoSourceOf(box)
		n := htmlcore.NodeOf(box)
		if src == "" || n == nil {
			return tree.Break
		}
		seen[n] = true
		pv := pg.videos[n]
		if pv != nil && pv.src != src { // (a script changed its src)
			pv.close()
			pv = nil
		}
		if pv == nil {
			pv = pg.newVideo(n, src)
		}
		pv.attach(box)
		return tree.Break // (its image and controls)
	})
	for n, pv := range pg.videos {
		if !seen[n] {
			pv.close()
			delete(pg.videos, n)
		}
	}
}

// closeVideos closes all of the videos of the page.
func (pg *Page) closeVideos() {
	for n, pv := range pg.videos {
		pv.close()
		delete(pg.videos, n)
	}
}

// newVideo returns a new video for the given video element of the page,
// whose media is opened in the background (as it is read over the network).
func (pg *Page) newVideo(n *html.Node, src string) *pageVideo {
	pv := &pageVideo{page: pg, node: n, src: src, opened: make(chan struct{})}
	if pg.videos == nil {
		pg.videos = map[*html.Node]*pageVideo{}
	}
	pg.videos[n] = pv
	options := mediaOptions(src, pg.Context.PageURL)
	autoplay := htmlcore.HasAttr(n, "autoplay")
	poster := htmlcore.HasAttr(n, "poster")
	pb := &video.Playback{
		Muted:   htmlcore.HasAttr(n, "muted"),
		Loop:    htmlcore.HasAttr(n, "loop"),
		OnFrame: func(f video.Frame) { pv.show(f.Image, f.Time) },
		OnEnd: func() {
			pv.refreshControls()
			pv.notify("timeupdate", "pause", "ended")
		},
		OnError: func(err error) { slog.Info("error playing a video", "url", src, "err", err) },
	}
	go func() {
		defer close(pv.opened)
		// the media of a MediaSource of the page (MSE), or of the URL
		var m *video.Media
		var err error
		if ms := pv.mediaSource(src); ms != nil {
			m, err = video.OpenSource(ms)
		} else {
			m, err = video.OpenMedia(src, options)
		}
		if err != nil {
			slog.Info("error opening a video", "url", src, "err", err)
			pv.mu.Lock()
			pv.openError = err.Error()
			pv.mu.Unlock()
			pv.notify("error")
			return
		}
		pb.Media = m
		pv.opening(m)
		pb.Width = pv.frameWidth(m)
		pv.mu.Lock()
		if pv.closed {
			pv.mu.Unlock()
			return
		}
		pv.playback = pb
		play := autoplay || pv.playWanted
		pv.mu.Unlock()
		pv.refreshControls()
		pv.notify("durationchange", "loadedmetadata", "loadeddata", "canplay", "canplaythrough")
		switch {
		case play:
			pv.play()
			pv.mu.Lock()
			if pv.closed { // (while it was starting)
				pb.Stop()
			}
			pv.mu.Unlock()
			pv.refreshControls()
		case !poster: // its first frame, until it plays
			if img, err := m.FirstFrame(pb.Width, 0); err == nil {
				pv.show(img, 0)
			}
		}
	}()
	return pv
}

// mediaOptions returns the options of FFmpeg for getting the video at the
// given URL of a page at the given URL: the user agent of the pages, their
// cookies for it, and the page as the referer.
func mediaOptions(src, pageURL string) map[string]string {
	opts := map[string]string{"user_agent": htmlcore.UserAgent}
	var headers strings.Builder
	if u, err := url.Parse(src); err == nil && (u.Scheme == "http" || u.Scheme == "https") {
		var cs []string
		for _, c := range httpClient.Jar.Cookies(u) {
			cs = append(cs, c.Name+"="+c.Value)
		}
		if len(cs) > 0 {
			headers.WriteString("Cookie: " + strings.Join(cs, "; ") + "\r\n")
		}
	}
	if strings.HasPrefix(pageURL, "http") {
		headers.WriteString("Referer: " + pageURL + "\r\n")
	}
	if headers.Len() > 0 {
		opts["headers"] = headers.String()
	}
	return opts
}

// opening sets the size of the box of the video to that of the opened
// media (in the goroutine that opens it), as the size of a video element
// is that of its video (unless the CSS of the page sets it).
func (pv *pageVideo) opening(m *video.Media) {
	pv.mu.Lock()
	pv.size = image.Pt(m.Width, m.Height)
	box := pv.box
	pv.mu.Unlock()
	if box == nil || !pv.page.AsyncLockIfShown() {
		return
	}
	defer pv.page.AsyncUnlock()
	if box.This != nil {
		box.SetVideoSize(m.Width, m.Height)
		box.Update()
	}
}

// frameWidth returns the width in pixels to decode the frames of the video
// at: that at which they fit in its box, or the width of the video up to
// [maxVideoWidth] before its box is laid out.
func (pv *pageVideo) frameWidth(m *video.Media) int {
	pv.mu.Lock()
	box := pv.box
	pv.mu.Unlock()
	w := 0
	if box != nil && pv.page.AsyncLockIfShown() { // (its geometry is of the lock)
		sz := box.Geom.Size.Actual.Content
		if sz.X > 0 && sz.Y > 0 && m.Width > 0 && m.Height > 0 {
			w = int(min(sz.X, sz.Y*float32(m.Width)/float32(m.Height)))
		}
		pv.page.AsyncUnlock()
	}
	if w > 0 {
		return min(w, m.Width)
	}
	return min(m.Width, maxVideoWidth)
}

// attach makes the given box the box of the video, showing the last frame
// in its image, if any, with the controls of the video over it.
func (pv *pageVideo) attach(box *htmlcore.VideoBox) {
	pv.mu.Lock()
	if pv.box == box { // (kept by a reading of a part of the page)
		pv.mu.Unlock()
		return
	}
	pv.box = box
	frame, size := pv.frame, pv.size
	pv.mu.Unlock()
	if size.X > 0 {
		box.SetVideoSize(size.X, size.Y)
	}
	if frame != nil {
		box.Image.SetImage(frame)
	}
	c := video.NewControls(box)
	c.SetPlayer(box, pv)
	pv.mu.Lock()
	pv.controls = c
	pv.mu.Unlock()
}

// show shows the given frame, at the given time of the video, in the image
// of the video (in the goroutine of the playback).
func (pv *pageVideo) show(frame *image.RGBA, t time.Duration) {
	pv.mu.Lock()
	box, closed, c := pv.box, pv.closed, pv.controls
	pv.frame = frame
	pv.mu.Unlock()
	if closed || box == nil || !pv.page.AsyncLockIfShown() {
		return
	}
	defer pv.page.AsyncUnlock()
	if box.This == nil { // (deleted, as the page closed)
		return
	}
	box.Image.SetImage(frame)
	box.Image.NeedsRender()
	if c != nil {
		c.SetTime(t)
	}
	// the time for the scripts, four times a second, as browsers do
	pv.mu.Lock()
	update := time.Since(pv.lastTimeUpdate) >= 250*time.Millisecond
	if update {
		pv.lastTimeUpdate = time.Now()
	}
	pv.mu.Unlock()
	if update {
		pv.notify("timeupdate")
	}
}

// refreshControls updates the controls for the state of the video
// (in another goroutine than that of the widgets).
func (pv *pageVideo) refreshControls() {
	pv.mu.Lock()
	c := pv.controls
	pv.mu.Unlock()
	if c == nil || !pv.page.AsyncLockIfShown() {
		return
	}
	defer pv.page.AsyncUnlock()
	c.Refresh()
}

// pb returns the playback of the video, or nil before it is opened.
func (pv *pageVideo) pb() *video.Playback {
	pv.mu.Lock()
	defer pv.mu.Unlock()
	return pv.playback
}

// TogglePlay plays the video if it is not playing, and pauses it if it is
// (part of [video.Player]).
func (pv *pageVideo) TogglePlay() {
	if pv.IsPlaying() {
		pv.pause()
	} else {
		pv.play()
	}
}

// play plays the video (from where it is paused, or from the start), or
// once it is opened, sending the play events to the scripts.
func (pv *pageVideo) play() {
	pb := pv.pb()
	switch {
	case pb == nil: // not opened (yet)
		pv.mu.Lock()
		pv.playWanted = true
		pv.mu.Unlock()
		return
	case pb.IsPlaying():
		return
	case pb.IsPaused():
		pb.Resume()
		pv.notify("play", "playing")
	default: // not played yet, or ended
		// (in another goroutine, as it opens the media again, as over the
		// network, which would hold the page)
		go func() {
			if err := pb.Play(); err != nil {
				slog.Info("error playing a video", "url", pv.src, "err", err)
				return
			}
			pv.notify("play", "playing")
			pv.refreshControls()
		}()
	}
}

// pause pauses the video, sending the pause event to the scripts.
func (pv *pageVideo) pause() {
	pv.mu.Lock()
	pv.playWanted = false
	pv.mu.Unlock()
	if pb := pv.pb(); pb != nil && pb.IsPlaying() {
		pb.Pause()
		pv.notify("pause")
	}
}

// mediaSource returns the MediaSource of the scripts of the page whose
// object URL is the given source of the video, if any (as a video.Source).
func (pv *pageVideo) mediaSource(src string) video.Source {
	if !strings.HasPrefix(src, "blob:") {
		return nil
	}
	if eng := pv.page.mediaEngine.Load(); eng != nil {
		if ms := eng.MediaSource(src); ms != nil {
			return ms
		}
	}
	return nil
}

// notify sends the given events (as "play") to the video element in the
// scripts of the page, if it has them, with the state of the video.
func (pv *pageVideo) notify(types ...string) {
	if eng := pv.page.mediaEngine.Load(); eng != nil {
		eng.MediaEvents(pv.node, pv.state(), types...)
	}
}

// state returns the state of the playing of the video for the scripts.
func (pv *pageVideo) state() script.MediaState {
	pv.mu.Lock()
	st := script.MediaState{Paused: !pv.playWanted, Width: pv.size.X, Height: pv.size.Y, Error: pv.openError}
	pv.mu.Unlock()
	st.Volume, st.Muted = float64(pv.Volume()), pv.IsMuted()
	if pb := pv.pb(); pb != nil {
		st.Paused = !pb.IsPlaying()
		st.Ended = pb.IsEnded()
		st.CurrentTime = pb.Position().Seconds()
		st.Duration = pb.Media.Duration.Seconds()
	}
	return st
}

// IsPlaying returns whether the video is playing.
func (pv *pageVideo) IsPlaying() bool {
	pb := pv.pb()
	return pb != nil && pb.IsPlaying()
}

// Seek plays the video from the given time, staying paused if it is.
func (pv *pageVideo) Seek(t time.Duration) {
	pv.seek(t, true)
}

// seek seeks the video in another goroutine, as it opens the media again
// (as over the network), with the events of the scripts if events (not for
// a seek of the scripts, which have them).
func (pv *pageVideo) seek(t time.Duration, events bool) {
	pb := pv.pb()
	if pb == nil {
		return
	}
	go func() {
		if err := pb.Seek(t); err != nil {
			slog.Info("error seeking a video", "url", pv.src, "err", err)
			return
		}
		if events {
			pv.notify("seeking", "seeked", "timeupdate")
		}
		pv.refreshControls()
	}()
}

// SkipBy seeks the video by the given time from its current time.
func (pv *pageVideo) SkipBy(d time.Duration) {
	pv.Seek(pv.CurrentTime() + d)
}

// StepFrame pauses the video and shows its next frame, or the one before
// it for a negative n.
func (pv *pageVideo) StepFrame(n int) {
	if pb := pv.pb(); pb != nil {
		playing := pb.IsPlaying()
		go func() { // (as for seek)
			if err := pb.StepFrame(n); err != nil {
				slog.Info("error stepping a video", "url", pv.src, "err", err)
				return
			}
			if playing {
				pv.notify("pause")
			}
			pv.notify("seeking", "seeked", "timeupdate")
			pv.refreshControls()
		}()
	}
}

// CurrentTime returns the current time of the video.
func (pv *pageVideo) CurrentTime() time.Duration {
	if pb := pv.pb(); pb != nil {
		return pb.Position()
	}
	return 0
}

// Duration returns the duration of the video, or 0 if it is not known.
func (pv *pageVideo) Duration() time.Duration {
	if pb := pv.pb(); pb != nil {
		return pb.Media.Duration
	}
	return 0
}

// IsMuted returns whether the audio is not heard.
func (pv *pageVideo) IsMuted() bool {
	if pb := pv.pb(); pb != nil {
		_, muted := pb.Audio()
		return muted
	}
	return htmlcore.HasAttr(pv.node, "muted")
}

// SetMuted sets whether the audio is not heard.
func (pv *pageVideo) SetMuted(muted bool) {
	if pb := pv.pb(); pb != nil {
		pb.SetMuted(muted)
		pv.notify("volumechange")
	}
}

// Volume returns the volume of the audio, from 0 to 1.
func (pv *pageVideo) Volume() float32 {
	if pb := pv.pb(); pb != nil {
		volume, _ := pb.Audio()
		return volume
	}
	return 1
}

// SetVolume sets the volume of the audio, from 0 to 1.
func (pv *pageVideo) SetVolume(volume float32) {
	if pb := pv.pb(); pb != nil {
		pb.SetVolume(volume)
		pv.notify("volumechange")
	}
}

// close stops the video.
func (pv *pageVideo) close() {
	pv.mu.Lock()
	defer pv.mu.Unlock()
	pv.closed = true
	if pv.playback != nil {
		pv.playback.Stop()
	}
}

// The page is the player of the video elements for its scripts (see
// script.MediaPlayer): its methods are called in their tasks, with the
// lock of the page held, and the videos send their events
// ([pageVideo.notify]) to the engine of the scripts (Page.mediaEngine).

// videoFor returns the video of the given video element, making it (and
// opening its media) if it has none yet, as when a script plays it before
// it is shown, or nil if the element has no source.
func (pg *Page) videoFor(n *html.Node) *pageVideo {
	src := pg.Context.VideoSource(n)
	if src == "" {
		return nil
	}
	if pv := pg.videos[n]; pv != nil {
		if pv.src == src {
			return pv
		}
		pv.close() // (a script changed its source)
	}
	return pg.newVideo(n, src)
}

// MediaLoad opens the media of the given video element.
func (pg *Page) MediaLoad(n *html.Node) {
	pg.videoFor(n)
}

// MediaPlay plays the media of the given video element.
func (pg *Page) MediaPlay(n *html.Node) error {
	pv := pg.videoFor(n)
	if pv == nil {
		return errors.New("the video has no source")
	}
	pv.play()
	pv.refreshControlsLocked()
	return nil
}

// MediaPause pauses the media of the given video element.
func (pg *Page) MediaPause(n *html.Node) {
	if pv := pg.videos[n]; pv != nil {
		pv.pause()
		pv.refreshControlsLocked()
	}
}

// MediaSeek seeks the media of the given video element.
func (pg *Page) MediaSeek(n *html.Node, t float64) {
	if pv := pg.videos[n]; pv != nil {
		pv.seek(time.Duration(t*float64(time.Second)), false)
	}
}

// MediaTime returns the current time of the media of the given video
// element, as it plays, or -1 if it is not known.
func (pg *Page) MediaTime(n *html.Node) float64 {
	if pv := pg.videos[n]; pv != nil {
		if pb := pv.pb(); pb != nil {
			return pb.Position().Seconds()
		}
	}
	return -1
}

// MediaVolume sets the volume of the media of the given video element.
func (pg *Page) MediaVolume(n *html.Node, volume float64, muted bool) {
	if pv := pg.videos[n]; pv != nil {
		if pb := pv.pb(); pb != nil {
			pb.SetVolume(float32(volume))
			pb.SetMuted(muted)
		}
		pv.refreshControlsLocked()
	}
}

// refreshControlsLocked updates the controls for the state of the video,
// with the lock of the widgets held.
func (pv *pageVideo) refreshControlsLocked() {
	pv.mu.Lock()
	c := pv.controls
	pv.mu.Unlock()
	if c != nil {
		c.Refresh()
	}
}

// Destroy stops the videos of the page (as when its tab is closed).
func (pg *Page) Destroy() {
	pg.closeVideos()
	pg.Frame.Destroy()
}
