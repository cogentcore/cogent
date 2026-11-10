// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"image"
	"log/slog"
	"net/url"
	"strings"
	"sync"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/htmlcore"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/video"
	"golang.org/x/net/html"
)

// The videos of a page: htmlcore makes each video element an image of its
// poster, with the URL of its video (see htmlcore.VideoSourceOf), and the
// page plays the video in it with a [video.Playback], which FFmpeg decodes
// (from its URL, with the cookies and user agent of the page). The video
// of an element is kept while the element is in the document, as the
// widgets of the page are read again (as after its scripts change it).
// A video plays at first if it has autoplay, and otherwise shows its
// poster, or its first frame; a click on it plays and pauses it.

// maxVideoWidth is the maximum width in pixels to which the frames of a
// video are decoded before its box is laid out.
const maxVideoWidth = 1280

// pageVideo is the video of a video element of a page.
type pageVideo struct {
	// page is the page of the video, whose lock is that of its images
	// (which are added to the scene of the page after they are made).
	page *Page

	// node is the video element.
	node *html.Node

	// src is the URL of the video.
	src string

	// playback plays the video, once its media is opened.
	playback *video.Playback

	mu sync.Mutex

	// img is the current image of the video element, which the frames
	// are shown in (a new one after each reading of the page).
	img *core.Image

	// frame is the last frame shown, to show in a new image.
	frame *image.RGBA

	// width is the width of the video, once it is opened, which is the
	// natural width of its image, whose frames are decoded at the width
	// of the image in pixels (see [core.Image.Density]).
	width int

	// opened is closed once the media is opened (or failed to open).
	opened chan struct{}

	// closed is whether the video has been closed.
	closed bool
}

// attachVideos attaches the videos of the video elements of the given
// content to their images in it, making the videos of new elements, and
// closes the videos of the elements that are no longer in it.
func (pg *Page) attachVideos(content *core.Frame) {
	seen := map[*html.Node]bool{}
	content.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
		img, ok := cw.(*core.Image)
		if !ok {
			return tree.Continue
		}
		src := htmlcore.VideoSourceOf(img)
		n := htmlcore.NodeOf(img)
		if src == "" || n == nil {
			return tree.Continue
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
		pv.attach(img)
		return tree.Continue
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
		OnFrame: func(f video.Frame) { pv.show(f.Image) },
		OnError: func(err error) { slog.Info("error playing a video", "url", src, "err", err) },
	}
	go func() {
		defer close(pv.opened)
		m, err := video.OpenMedia(src, options)
		if err != nil {
			slog.Info("error opening a video", "url", src, "err", err)
			return
		}
		pb.Width = pv.frameWidth(m)
		pb.Media = m
		pv.mu.Lock()
		if pv.closed {
			pv.mu.Unlock()
			return
		}
		pv.playback = pb
		pv.width = m.Width
		pv.mu.Unlock()
		switch {
		case autoplay:
			if err := pb.Play(); err != nil {
				slog.Info("error playing a video", "url", src, "err", err)
			}
			pv.mu.Lock()
			if pv.closed { // (while it was starting)
				pb.Stop()
			}
			pv.mu.Unlock()
		case !poster: // its first frame, until it plays
			if img, err := m.FirstFrame(pb.Width, 0); err == nil {
				pv.show(img)
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

// frameWidth returns the width in pixels to decode the frames of the video
// at: that of its image, or the width of the video up to [maxVideoWidth]
// before its image is laid out.
func (pv *pageVideo) frameWidth(m *video.Media) int {
	pv.mu.Lock()
	img := pv.img
	pv.mu.Unlock()
	w := 0
	if img != nil && pv.page.AsyncLockIfShown() { // (its geometry is of the lock)
		w = int(math32.Ceil(img.Geom.Size.Actual.Content.X))
		pv.page.AsyncUnlock()
	}
	if w > 0 {
		return min(w, m.Width)
	}
	return min(m.Width, maxVideoWidth)
}

// attach makes the given image the image of the video, showing the last
// frame in it, if any, and a click on it plays and pauses the video.
func (pv *pageVideo) attach(img *core.Image) {
	pv.mu.Lock()
	pv.img = img
	frame, width := pv.frame, pv.width
	pv.mu.Unlock()
	if frame != nil {
		setFrame(img, frame, width)
	}
	img.OnClick(func(e events.Event) {
		pv.mu.Lock()
		pb := pv.playback
		pv.mu.Unlock()
		switch {
		case pb == nil: // not opened (yet)
		case pb.IsPlaying():
			pb.Pause()
		case pb.IsPaused():
			pb.Resume()
		default:
			if err := pb.Play(); err != nil {
				slog.Info("error playing a video", "url", pv.src, "err", err)
			}
		}
		e.SetHandled()
	})
}

// show shows the given frame in the image of the video (in the goroutine
// of the playback), laying the page out again when its size changes
// (as the size of the video is that of its first frame).
func (pv *pageVideo) show(frame *image.RGBA) {
	pv.mu.Lock()
	img, prev, closed, width := pv.img, pv.frame, pv.closed, pv.width
	pv.frame = frame
	pv.mu.Unlock()
	if closed || img == nil || !pv.page.AsyncLockIfShown() {
		return
	}
	defer pv.page.AsyncUnlock()
	if img.This == nil { // (deleted, as the page closed)
		return
	}
	setFrame(img, frame, width)
	if prev == nil || prev.Bounds().Size() != frame.Bounds().Size() {
		img.Update()
		return
	}
	img.NeedsRender()
}

// setFrame sets the image of the given image of a video to the given frame,
// of the video of the given width, which is the natural width of the image.
func setFrame(img *core.Image, frame *image.RGBA, width int) {
	if width > 0 {
		img.Density = float32(frame.Bounds().Dx()) / float32(width)
	}
	img.SetImage(frame)
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

// Destroy stops the videos of the page (as when its tab is closed).
func (pg *Page) Destroy() {
	pg.closeVideos()
	pg.Frame.Destroy()
}
