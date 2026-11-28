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
	"strings"
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
		a.showTag("tag:2020/First")
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
		a.showTag("tag:2020/First")
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

// TestFixedButtons tests that All and Trash are buttons above the tree of
// tags, which show their photos, and that the tags are in the tree.
func TestFixedButtons(t *testing.T) {
	lib := smallLibrary(t, 6)
	phs := lib.Photos(nil)
	require.NoError(t, lib.AddTag("2020_First", phs[:3]...))
	require.NoError(t, lib.Trash(phs[5]))
	b := core.NewBody()
	a := NewApp(b)
	a.shown = "tag:2020/First"
	var trash *core.Button
	core.RenderImage(b, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		a.lib = lib
		a.grid.lib = lib
		a.refresh()
	}, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		for _, r := range a.tagRows {
			assert.NotEqual(t, allKey, r.Key)
			assert.NotEqual(t, trashKey, r.Key)
		}
		assert.Len(t, a.grid.Photos, 3)
		assert.Equal(t, 5, a.allCount)
		assert.Equal(t, 1, a.trashCount)
		a.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
			if bt, ok := cw.(*core.Button); ok && strings.HasPrefix(bt.Text, "Trash") {
				trash = bt
			}
			return true
		})
		require.NotNil(t, trash)
		assert.Equal(t, "Trash  1", trash.Text)
		trash.Send(events.Click)
	}, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		assert.Equal(t, trashKey, a.shown)
		assert.Len(t, a.grid.Photos, 1)
		assert.Equal(t, -1, a.tags.SelectedIndex, "no tag selected")
		assert.Contains(t, trash.Text, "Trash")
	})
}

// TestDragToTag tests dragging selected photos from the grid onto a tag
// in the tree of tags, which tags them.
func TestDragToTag(t *testing.T) {
	lib := smallLibrary(t, 8)
	phs := lib.Photos(nil)
	require.NoError(t, lib.AddTag("Other", phs[7]))
	require.NoError(t, lib.AddTag("Target", phs[6]))
	b := core.NewBody()
	a := NewApp(b)
	a.Styler(func(s *styles.Style) {
		s.Min.Set(units.Dp(1200), units.Dp(600))
	})
	a.shown = allKey
	Settings.Sort = OldestFirst
	defer func() { Settings.Sort = NewestFirst }()
	core.RenderImage(b, func() {
		b.AsyncLock()
		a.lib = lib
		a.grid.lib = lib
		a.grid.CellSize = units.Dp(80)
		a.refresh()
		a.grid.Selected = map[int]bool{0: true, 1: true}
		a.grid.Current = 1
		b.AsyncUnlock()
	}, func() {
		b.AsyncLock()
		from := a.grid.cellRect(1).Min.Add(image.Pt(10, 10))
		var to image.Point
		rows := *a.tags.Slice.(*[]*tagRow)
		for i, r := range rows {
			if r.tag == "Target" {
				nw, _ := a.tags.RowWidgetNs()
				cell := a.tags.ListGrid.Child(i * nw).(core.Widget).AsWidget()
				to = cell.Geom.ContentBBox.Min.Add(image.Pt(20, 5))
			}
		}
		require.NotEqual(t, image.Point{}, to)
		b.AsyncUnlock()
		ev := a.grid.SystemEvents()
		ev.MouseButton(events.MouseDown, events.Left, from, 0)
		time.Sleep(200 * time.Millisecond)
		for i := 1; i <= 10; i++ {
			ev.MouseMove(from.Add(to.Sub(from).Mul(i).Div(10)))
			time.Sleep(30 * time.Millisecond)
		}
		ev.MouseButton(events.MouseUp, events.Left, to, 0)
		time.Sleep(300 * time.Millisecond)
	}, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		assert.Equal(t, 3, lib.Tags()["Target"], "the two dragged are tagged")
	})
}

// TestYearToggle tests that a click on the branch of a year in the tree of
// tags opens it, while All is shown, with no tag selected.
func TestYearToggle(t *testing.T) {
	lib := smallLibrary(t, 60)
	phs := lib.Photos(nil)
	for i, ph := range phs { // many years, as in a real library, more than fit
		require.NoError(t, lib.AddTag(fmt.Sprintf("%d_Tag%d", 1990+i/2, i%2), ph))
	}
	Settings.ClosedGroups = nil
	b := core.NewBody()
	a := NewApp(b)
	a.Styler(func(s *styles.Style) {
		s.Min.Set(units.Dp(1200), units.Dp(600))
	})
	a.shown = allKey
	branch := func(year string) (core.Widget, int) { // its widget, and visible row
		rows := *a.tags.Slice.(*[]*tagRow)
		nw, _ := a.tags.RowWidgetNs()
		for i, r := range rows {
			if r.group == year {
				w := i - a.tags.StartIndex
				cell := a.tags.ListGrid.Child(w * nw).(core.Widget)
				return cell.AsWidget().Child(0).(core.Widget), i
			}
		}
		return nil, -1
	}
	var year *tagRow
	core.RenderImage(b, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		a.lib = lib
		a.grid.lib = lib
		a.refresh()
	}, func() {
		b.AsyncLock()
		bb := a.tags.ListGrid.Geom.ContentBBox
		b.AsyncUnlock()
		for range 4 { // scrolled down
			b.Scene.RenderWindow().SystemWindow.Events().Scroll(bb.Min.Add(bb.Size().Div(2)), math32.Vec2(0, 30))
			time.Sleep(100 * time.Millisecond)
		}
		b.AsyncLock()
		t.Log("start", a.tags.StartIndex)
		br, i := branch("2010") // down in the tree
		require.NotNil(t, br)
		year = (*a.tags.Slice.(*[]*tagRow))[i]
		require.True(t, year.Closed, "closed at first")
		pt := br.AsWidget().Geom.ContentBBox.Min.Add(br.AsWidget().Geom.ContentBBox.Size().Div(2))
		photo := a.grid.cellRect(1).Min.Add(image.Pt(10, 10))
		t.Log("row", i, "visible", a.tags.VisibleRows, "start", a.tags.StartIndex, "pt", pt, "tree", a.tags.Geom.ContentBBox, "grid", a.tags.ListGrid.Geom.ContentBBox)
		b.AsyncUnlock()
		ev := a.SystemEvents()
		click := func(p image.Point) {
			ev.MouseButton(events.MouseDown, events.Left, p, 0)
			ev.MouseButton(events.MouseUp, events.Left, p, 0)
			time.Sleep(600 * time.Millisecond)
		}
		click(photo) // the grid has the focus
		click(pt)
		b.AsyncLock()
		assert.False(t, year.Closed, "opened by its branch")
		assert.Equal(t, allKey, a.shown, "still All")
		b.AsyncUnlock()
		click(pt)
		b.AsyncLock()
		assert.True(t, year.Closed, "closed again")
		b.AsyncUnlock()
	})
}

// TestShowInAll tests showing the current photo of a tag in All, where it
// is current and in view, with the photos around it.
func TestShowInAll(t *testing.T) {
	lib := smallLibrary(t, 200)
	phs := lib.Photos(nil)
	require.NoError(t, lib.AddTag("2020_Some", phs[100], phs[150]))
	Settings.Views = nil
	b := core.NewBody()
	a := NewApp(b)
	a.Styler(func(s *styles.Style) {
		s.Min.Set(units.Dp(1200), units.Dp(600))
		s.Max.Set(units.Dp(1200), units.Dp(600))
	})
	a.shown = "tag:2020/Some"
	core.RenderImage(b, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		a.lib = lib
		a.grid.lib = lib
		a.refresh()
	}, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		require.Len(t, a.grid.Photos, 2)
		a.grid.SelectIndex(1)
		a.ShowInAll()
	}, func() {
		waitFor(t, b, func() bool { return a.shown == allKey && a.gridFrame().HasScroll[math32.Y] })
		b.AsyncLock()
		defer b.AsyncUnlock()
		require.Equal(t, allKey, a.shown)
		cur := a.grid.CurrentPhoto()
		require.NotNil(t, cur)
		assert.Contains(t, []string{phs[100].Name, phs[150].Name}, cur.Name)
		assert.Equal(t, 200, len(a.grid.Photos))
		r := a.grid.cellRect(a.grid.Current)
		vis := a.gridFrame().Geom.ContentBBox
		assert.True(t, r.Overlaps(vis), "in view: %v in %v", r, vis)
	})
}
