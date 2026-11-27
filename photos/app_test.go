// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package photos

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
	"time"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMain uses temporary settings.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "photos-test")
	if err != nil {
		panic(err)
	}
	Settings.File = filepath.Join(dir, "settings.toml")
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// TestAppShot renders the app with the library in PHOTOS_SHOT, saving
// the image in PHOTOS_SHOT_IMAGE, for looking at it.
func TestAppShot(t *testing.T) {
	dir := os.Getenv("PHOTOS_SHOT")
	if dir == "" {
		t.Skip("PHOTOS_SHOT not set")
	}
	b := core.NewBody("Cogent Photos")
	a := NewApp(b)
	a.Styler(func(s *styles.Style) {
		s.Min.Set(units.Dp(1400), units.Dp(850))
	})
	b.AddTopBar(func(bar *core.Frame) {
		core.NewToolbar(bar).Maker(a.MakeToolbar)
	})
	if tag := os.Getenv("PHOTOS_SHOT_TAG"); tag != "" {
		a.shown = tag
	}
	img := core.RenderImage(b, func() {
		lib, err := OpenLibrary(dir)
		require.NoError(t, err)
		lib.ThumbDir = filepath.Join(t.TempDir(), "thumbs") // not the real ones
		if lib.Len() == 0 {
			require.NoError(t, lib.Scan(nil, nil))
		}
		b.AsyncLock()
		a.lib = lib
		a.grid.lib = lib
		a.refresh()
		b.AsyncUnlock()
		time.Sleep(3 * time.Second) // the thumbnails
		b.AsyncLock()
		a.grid.NeedsRender()
		if os.Getenv("PHOTOS_SHOT_VIDEO") != "" { // the first video
			for i, ph := range a.grid.Photos {
				if ph.IsVideo() {
					a.grid.SelectIndex(i)
					break
				}
			}
		}
		b.AsyncUnlock()
		if os.Getenv("PHOTOS_SHOT_VIDEO") == "play" {
			time.Sleep(time.Second)
			b.AsyncLock()
			box := a.viewer.box
			require.NotNil(t, box, "the box of the video")
			pt := box.Geom.ContentBBox.Min.Add(box.Geom.ContentBBox.Size().Div(3)) // above the controls
			b.AsyncUnlock()
			// a real click on the video plays it
			ev := a.viewer.SystemEvents()
			ev.MouseButton(events.MouseDown, events.Left, pt, 0)
			ev.MouseButton(events.MouseUp, events.Left, pt, 0)
			time.Sleep(2 * time.Second)
			b.AsyncLock()
			assert.True(t, a.viewer.player.IsPlaying(), "a click played it")
			b.AsyncUnlock()
		} else {
			time.Sleep(time.Second)
		}
	})
	require.NotNil(t, img)
	require.NoError(t, imagex.Save(img, os.Getenv("PHOTOS_SHOT_IMAGE")))
}

// smallLibrary returns a new library in a temporary directory with the
// given number of small photos, a minute apart, without metadata.
func smallLibrary(t *testing.T, n int) *Library {
	dir := t.TempDir()
	all := filepath.Join(dir, "All")
	require.NoError(t, os.MkdirAll(all, 0o755))
	start := time.Date(2020, 1, 1, 12, 0, 0, 0, time.Local)
	for i := range n {
		img := image.NewRGBA(image.Rect(0, 0, 64, 48))
		draw.Draw(img, img.Bounds(), image.NewUniform(color.RGBA{uint8(i * 7), uint8(255 - i*3), 128, 255}), image.Point{}, draw.Src)
		fn := filepath.Join(all, fmt.Sprintf("p%03d.jpg", i))
		f, err := os.Create(fn)
		require.NoError(t, err)
		require.NoError(t, jpeg.Encode(f, img, nil))
		f.Close()
		d := start.Add(time.Duration(i) * time.Minute)
		os.Chtimes(fn, d, d)
	}
	lib, err := OpenLibrary(dir)
	require.NoError(t, err)
	lib.ThumbDir = filepath.Join(dir, "thumbs")
	require.NoError(t, lib.Scan(nil, nil))
	t.Cleanup(func() { lib.Close() })
	return lib
}

// TestTagViews tests that each tag, and All, goes back to where its photos
// were viewed: the scroll position and the current photo.
func TestTagViews(t *testing.T) {
	lib := smallLibrary(t, 120)
	phs := lib.Photos(nil)
	require.NoError(t, lib.AddTag("2020_First", phs[:60]...))
	require.NoError(t, lib.AddTag("2020_Second", phs[60:]...))
	Settings.Views = nil
	b := core.NewBody()
	a := NewApp(b)
	a.Styler(func(s *styles.Style) {
		s.Min.Set(units.Dp(1200), units.Dp(600))
		s.Max.Set(units.Dp(1200), units.Dp(600))
	})
	a.shown = allKey
	var allScroll, tagScroll float32
	core.RenderImage(b, func() {
		b.AsyncLock()
		a.lib = lib
		a.grid.lib = lib
		a.refresh()
		b.AsyncUnlock()
	}, func() {
		waitFor(t, b, func() bool { return a.gridFrame().HasScroll[math32.Y] }) // laid out
		b.AsyncLock()
		defer b.AsyncUnlock()
		fr := a.gridFrame()
		require.True(t, fr.HasScroll[math32.Y], "the grid scrolls")
		a.grid.SelectIndex(80) // scrolls to it
		allScroll = fr.Geom.Scroll.Y
		require.Less(t, allScroll, float32(0))
		a.showTag("tag:2020_First")
	}, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		a.grid.SelectIndex(50)
		tagScroll = a.gridFrame().Geom.Scroll.Y
		a.showTag(allKey)
	}, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		assert.Equal(t, 80, a.grid.Current, "the current photo of All")
		assert.Equal(t, allScroll, a.gridFrame().Geom.Scroll.Y, "the scroll of All")
		a.showTag("tag:2020_First")
	}, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		assert.Equal(t, 50, a.grid.Current, "the current photo of the tag")
		assert.Equal(t, tagScroll, a.gridFrame().Geom.Scroll.Y, "the scroll of the tag")
		t.Log("scrolls", allScroll, tagScroll)
	})
}

// waitFor waits up to five seconds for the given condition, checked with
// the lock of the window of the given body.
func waitFor(t *testing.T, b *core.Body, cond func() bool) {
	for range 100 {
		b.AsyncLock()
		ok := cond()
		b.AsyncUnlock()
		if ok {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
}
