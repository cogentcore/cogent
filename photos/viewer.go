// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package photos

import (
	"fmt"
	"image"
	"strings"
	"sync"
	"time"

	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/video"
)

// viewMaxSize is the maximum size of the images of photos shown, which
// is enough for the screen.
const viewMaxSize = 3000

// videoMaxWidth is the maximum width to which the frames of videos are
// decoded for playing them.
const videoMaxWidth = 1920

// Viewer shows a photo, with a line of information about it below it.
// The left and right keys go to the previous and next photos in the grid.
// A video is shown in a [videoBox], as videos are in htmlcore: its first
// frame, and then its frames as it plays, with the standard controls of
// players over it ([video.Controls]), for which the viewer plays it
// ([viewerPlayer]).
type Viewer struct {
	core.Frame

	// Photo is the photo shown.
	Photo *Photo `set:"-"`

	// app is the app.
	app *App

	// image shows the photo, or the frames of the video, in box for a video.
	image *core.Image
	box   *videoBox
	info  *core.Text

	// player plays the video shown, if any, and controls are the controls
	// of the video, which control it through a [playerProxy], as the box
	// of a video is kept when the same video is shown again.
	player   *viewerPlayer
	controls *video.Controls

	// gen is the number of the photo being loaded, so that only the
	// latest one is shown, under mu.
	gen int
	mu  sync.Mutex
}

// videoBox is the box of a video in a [Viewer], with the image of its
// frames, and the controls over it, as the VideoBox of htmlcore.
type videoBox struct {
	core.Frame
}

func (vb *videoBox) Init() {
	vb.Frame.Init()
	vb.Styler(func(s *styles.Style) {
		// a click plays and pauses it (see [video.Controls.SetPlayer])
		s.SetAbilities(true, abilities.Clickable)
		s.Position = styles.PositionRelative // (the containing block of the controls)
		s.Background = colors.Uniform(colors.Black)
		s.Padding.Zero()
		s.Gap.Zero()
		s.Grow.Set(1, 1)
		s.Min.X.Zero()
		s.Min.Y.Zero()
	})
}

// RenderGroup returns true: the box is rendered as a whole when any widget
// in it needs to be rendered, as the controls are partly transparent over
// the frames.
func (vb *videoBox) RenderGroup() bool { return true }

func (v *Viewer) Init() {
	v.Frame.Init()
	v.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.SetAbilities(true, abilities.Focusable, abilities.Clickable)
		s.Background = colors.Scheme.SurfaceContainerLowest
		s.Grow.Set(1, 1)
	})
	v.OnClick(func(e events.Event) { v.SetFocus() })
	v.On(events.KeyChord, func(e events.Event) {
		g := v.app.grid
		switch keymap.Of(e.KeyChord()) {
		case keymap.MoveRight, keymap.MoveDown:
			e.SetHandled()
			g.SelectIndex(g.Current + 1)
		case keymap.MoveLeft, keymap.MoveUp:
			e.SetHandled()
			g.SelectIndex(g.Current - 1)
		}
	})
	v.Maker(func(p *tree.Plan) {
		ph := v.Photo
		if ph == nil || !ph.IsVideo() || v.player == nil {
			v.box, v.controls = nil, nil
			tree.AddAt(p, "image", func(w *core.Image) {
				initViewImage(w)
				v.image = w
				// dragged out of the app, as a file, or onto a tag
				w.Styler(func(s *styles.Style) {
					s.SetAbilities(true, abilities.Draggable)
				})
				w.On(events.DragStart, func(e events.Event) {
					ph := v.Photo
					if ph == nil || v.app == nil || v.app.lib == nil {
						return
					}
					md := mimedata.NewMime(photosMime, []byte(ph.Name))
					v.Scene.Events.DragStart(w, md, e)
					v.Scene.Events.SetDragFiles(v.app.lib.PhotoFile(ph))
					g := v.app.grid
					if img := g.scaledThumb(ph, g.scaledSize); img != nil {
						v.Scene.Events.SetDragImage(img)
					}
					e.SetHandled()
				})
			})
		} else {
			// a new box for each video, as the controls add handlers to it
			tree.AddAt(p, "video-"+ph.Name, func(w *videoBox) {
				v.box = w
				tree.AddChild(w, func(w *core.Image) {
					initViewImage(w)
					w.Styler(func(s *styles.Style) {
						s.Position = styles.PositionAbsolute // filling the box
						s.Inset.Set(units.Zero())
						s.InsetSet.Set(true)
					})
					v.image = w
				})
				box := w
				tree.AddChild(w, func(c *video.Controls) { // (in the plan of the box, to be kept)
					c.SetPlayer(box, playerProxy{v})
					v.controls = c
				})
			})
		}
		tree.AddAt(p, "info", func(w *core.Text) {
			v.info = w
			w.SetType(core.TextBodyLarge) // regular text
			w.Updater(func() {
				w.SetText(v.infoText())
			})
		})
	})
}

// initViewImage initializes an image of the viewer.
func initViewImage(w *core.Image) {
	w.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
		s.Min.X.Zero()
		s.Min.Y.Zero()
		s.ObjectFit = styles.FitContain
	})
}

func (v *Viewer) Destroy() {
	v.stopVideo()
	v.Frame.Destroy()
}

// infoText returns the line of information about the photo shown.
func (v *Viewer) infoText() string {
	ph := v.Photo
	if ph == nil {
		return ""
	}
	var parts []string
	parts = append(parts, ph.DateTaken.Format("Mon Jan 2, 2006 3:04 PM"))
	if len(ph.Tags) > 0 {
		var tags []string
		for _, t := range ph.Tags {
			y, n := splitYearTag(t)
			tags = append(tags, strings.TrimSpace(y+" "+n))
		}
		parts = append(parts, strings.Join(tags, ", "))
	}
	if ph.Desc != "" {
		parts = append(parts, ph.Desc)
	}
	sz := ph.DisplaySize()
	if sz.X > 0 {
		parts = append(parts, fmt.Sprintf("%d×%d", sz.X, sz.Y))
	}
	if ph.Duration > 0 {
		parts = append(parts, formatLength(ph.Duration))
	}
	if cam := strings.TrimSpace(ph.Make + " " + ph.Model); cam != "" {
		parts = append(parts, cam)
	}
	parts = append(parts, ph.Filename())
	return strings.Join(parts, " · ")
}

// stopVideo stops the video shown, if any.
func (v *Viewer) stopVideo() {
	if v.player != nil {
		v.player.close()
		v.player = nil
	}
}

// SetPhoto shows the given photo, loading its image in the background,
// with its thumbnail shown first. A video is opened for playing.
func (v *Viewer) SetPhoto(ph *Photo) {
	v.stopVideo()
	v.Photo = ph
	v.mu.Lock()
	v.gen++
	gen := v.gen
	v.mu.Unlock()
	if ph != nil && ph.IsVideo() {
		v.player = &viewerPlayer{viewer: v}
	}
	if ph == nil || v.app == nil || v.app.lib == nil {
		v.Update()
		if v.image != nil {
			v.image.SetImage(nil)
		}
		return
	}
	g := v.app.grid
	g.thumbsMu.Lock()
	thumb := g.thumbs[ph.Name]
	g.thumbsMu.Unlock()
	v.Update()
	v.image.SetImage(thumb)
	lib := v.app.lib
	current := func() bool {
		v.mu.Lock()
		defer v.mu.Unlock()
		return gen == v.gen
	}
	show := func(img image.Image, err error) {
		if !current() {
			return
		}
		v.AsyncLock()
		defer v.AsyncUnlock()
		if err != nil {
			core.ErrorSnackbar(v, err, "Error opening "+ph.Filename())
			return
		}
		v.image.SetImage(img)
		v.image.NeedsRender()
	}
	if ph.IsVideo() {
		go v.player.open(lib.PhotoFile(ph), show)
		return
	}
	go func() {
		img, err := openPhoto(ph, lib.PhotoFile(ph), viewMaxSize)
		if err == nil {
			img = scaleToFit(img, viewMaxSize)
		}
		show(img, err)
	}()
}

// PlayVideo plays or pauses the video shown, if any.
func (v *Viewer) PlayVideo() {
	if v.player != nil {
		v.player.TogglePlay()
		if v.controls != nil {
			v.controls.Refresh()
		}
	}
}

// viewerPlayer plays the video of a [Viewer] (a [video.Player] for its
// controls): it shows the first frame of the video, and then its frames as
// it plays, in the image of the viewer.
type viewerPlayer struct {
	viewer *Viewer

	// pb is the playback of the video, once it is opened, under mu.
	pb *video.Playback

	// playWanted is whether it is to play once it is opened, and closed
	// whether it is closed, under mu.
	playWanted, closed bool

	mu sync.Mutex
}

// open opens the video in the given file, showing its first frame with
// the given function, and then plays it if that was wanted already.
func (vp *viewerPlayer) open(filename string, show func(img image.Image, err error)) {
	m, err := video.OpenMedia(filename, nil)
	if err != nil {
		show(nil, err)
		return
	}
	width := min(m.Width, videoMaxWidth)
	pb := &video.Playback{Media: m, Width: width,
		OnFrame: vp.showFrame,
		OnEnd:   vp.refresh,
		OnError: vp.showError,
	}
	vp.mu.Lock()
	if vp.closed {
		vp.mu.Unlock()
		m.Close()
		return
	}
	vp.pb = pb
	play := vp.playWanted
	vp.mu.Unlock()
	if play {
		vp.play()
	} else if img, err := m.FirstFrame(width, 0); err == nil {
		show(img, nil)
	}
	vp.refresh()
}

// showFrame shows the given frame (in the goroutine of the playback).
func (vp *viewerPlayer) showFrame(f video.Frame) {
	vp.mu.Lock()
	closed := vp.closed
	vp.mu.Unlock()
	v := vp.viewer
	if closed || !v.AsyncLockIfShown() {
		return
	}
	defer v.AsyncUnlock()
	if v.image == nil || v.player != vp {
		return
	}
	v.image.SetImage(f.Image)
	v.image.NeedsRender()
	if v.controls != nil {
		v.controls.SetTime(f.Time)
	}
}

// showError shows the given error of playing the video.
func (vp *viewerPlayer) showError(err error) {
	v := vp.viewer
	if v.AsyncLockIfShown() {
		core.ErrorSnackbar(v, err, "Error playing the video")
		v.AsyncUnlock()
	}
}

// refresh updates the controls for the state of the video (in another
// goroutine than that of the widgets).
func (vp *viewerPlayer) refresh() {
	v := vp.viewer
	if !v.AsyncLockIfShown() {
		return
	}
	defer v.AsyncUnlock()
	if v.controls != nil && v.controls.This != nil && v.player == vp {
		v.controls.Refresh()
	}
}

// close stops the video.
func (vp *viewerPlayer) close() {
	vp.mu.Lock()
	defer vp.mu.Unlock()
	vp.closed = true
	if vp.pb != nil {
		vp.pb.Stop()
		vp.pb.Media.Close()
	}
}

// playback returns the playback of the video, or nil before it is opened.
func (vp *viewerPlayer) playback() *video.Playback {
	vp.mu.Lock()
	defer vp.mu.Unlock()
	return vp.pb
}

// TogglePlay plays the video if it is not playing, and pauses it if it is.
func (vp *viewerPlayer) TogglePlay() {
	if vp.IsPlaying() {
		vp.mu.Lock()
		vp.playWanted = false
		vp.mu.Unlock()
		vp.playback().Pause()
		return
	}
	vp.play()
}

// play plays the video, from where it is paused, or from its start, or
// once it is opened.
func (vp *viewerPlayer) play() {
	pb := vp.playback()
	switch {
	case pb == nil: // not opened yet
		vp.mu.Lock()
		vp.playWanted = true
		vp.mu.Unlock()
	case pb.IsPlaying():
	case pb.IsPaused():
		pb.Resume()
	default: // not played yet, or ended: it opens the media again
		go func() {
			if err := pb.Play(); err != nil {
				vp.showError(err)
			}
			vp.refresh()
		}()
	}
}

func (vp *viewerPlayer) IsPlaying() bool {
	pb := vp.playback()
	return pb != nil && pb.IsPlaying()
}

func (vp *viewerPlayer) Seek(t time.Duration) {
	if pb := vp.playback(); pb != nil {
		go func() {
			pb.Seek(t)
			vp.refresh()
		}()
	}
}

func (vp *viewerPlayer) SkipBy(d time.Duration) {
	vp.Seek(vp.CurrentTime() + d)
}

func (vp *viewerPlayer) StepFrame(n int) {
	if pb := vp.playback(); pb != nil {
		go func() {
			pb.StepFrame(n)
			vp.refresh()
		}()
	}
}

func (vp *viewerPlayer) CurrentTime() time.Duration {
	if pb := vp.playback(); pb != nil {
		return pb.Position()
	}
	return 0
}

func (vp *viewerPlayer) Duration() time.Duration {
	if pb := vp.playback(); pb != nil {
		return pb.Media.Duration
	}
	if ph := vp.viewer.Photo; ph != nil {
		return ph.Duration
	}
	return 0
}

func (vp *viewerPlayer) IsMuted() bool {
	if pb := vp.playback(); pb != nil {
		_, muted := pb.Audio()
		return muted
	}
	return false
}

func (vp *viewerPlayer) SetMuted(muted bool) {
	if pb := vp.playback(); pb != nil {
		pb.SetMuted(muted)
	}
}

func (vp *viewerPlayer) Volume() float32 {
	if pb := vp.playback(); pb != nil {
		volume, _ := pb.Audio()
		return volume
	}
	return 1
}

func (vp *viewerPlayer) SetVolume(volume float32) {
	if pb := vp.playback(); pb != nil {
		pb.SetVolume(volume)
	}
}

// playerProxy is the [video.Player] of the controls of the videos of a
// [Viewer], which is its current player, as the controls of a video are
// kept when it is shown again, with a new player.
type playerProxy struct {
	v *Viewer
}

// player returns the current player, or a closed one if there is none.
func (pp playerProxy) player() *viewerPlayer {
	if pp.v.player != nil {
		return pp.v.player
	}
	return &viewerPlayer{viewer: pp.v, closed: true}
}

func (pp playerProxy) TogglePlay()                { pp.player().TogglePlay() }
func (pp playerProxy) IsPlaying() bool            { return pp.player().IsPlaying() }
func (pp playerProxy) Seek(t time.Duration)       { pp.player().Seek(t) }
func (pp playerProxy) SkipBy(d time.Duration)     { pp.player().SkipBy(d) }
func (pp playerProxy) StepFrame(n int)            { pp.player().StepFrame(n) }
func (pp playerProxy) CurrentTime() time.Duration { return pp.player().CurrentTime() }
func (pp playerProxy) Duration() time.Duration    { return pp.player().Duration() }
func (pp playerProxy) IsMuted() bool              { return pp.player().IsMuted() }
func (pp playerProxy) SetMuted(muted bool)        { pp.player().SetMuted(muted) }
func (pp playerProxy) Volume() float32            { return pp.player().Volume() }
func (pp playerProxy) SetVolume(volume float32)   { pp.player().SetVolume(volume) }
