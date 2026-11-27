// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package photos

import (
	"fmt"
	"os"
	"image"
	"testing"
	"time"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"github.com/stretchr/testify/assert"
)

// TestGridScrollClick tests that a photo clicked after scrolling far is
// the one under the mouse, which stays in view.
func TestGridScrollClick(t *testing.T) {
	b := core.NewBody()
	fr := core.NewFrame(b)
	fr.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Overflow.Set(styles.OverflowAuto)
		s.Min.Set(units.Dp(800), units.Dp(600))
		s.Max.Set(units.Dp(800), units.Dp(600))
	})
	g := NewGrid(fr)
	var phs []*Photo
	for i := range 200000 {
		phs = append(phs, &Photo{Name: fmt.Sprint(i), DateTaken: time.Unix(int64(i), 0)})
	}
	g.SetPhotos(phs)
	core.RenderImage(b, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		for _, frac := range []float32{0.001, 0.01, 0.1, 0.5, 0.9} {
			h := g.Geom.Size.Actual.Total.Y
			fr.ScrollDimToStart(math32.Y, int(fr.Geom.Pos.Content.Y+frac*h)+int(fr.Geom.Scroll.Y)*0)
			b.Scene.RenderWindow() // positions
			vis := fr.Geom.ContentBBox
			pt := image.Pt(vis.Min.X+g.cols/2*int(g.cell)+5, (vis.Min.Y+vis.Max.Y)/2)
			idx, ok := g.indexAt(pt)
			assert.True(t, ok)
			assert.True(t, pt.In(g.cellRect(idx)), "frac %v", frac)
			scroll := fr.Geom.Scroll.Y
			g.selectEvent(idx, events.SelectOne)
			assert.Equal(t, scroll, fr.Geom.Scroll.Y, "frac %v: the click scrolled", frac)
			t.Log(frac, idx, scroll, g.Geom.Pos.Content.Y, g.cell, g.cols, h)
		}
	})
}

// TestGridMultiSelect tests selecting photos with the mouse: Command
// selects or unselects one more, and Shift selects the range from the
// nearest selected one, before or after.
func TestGridMultiSelect(t *testing.T) {
	lib := smallLibrary(t, 30)
	b := core.NewBody()
	a := NewApp(b)
	a.Styler(func(s *styles.Style) {
		s.Min.Set(units.Dp(1200), units.Dp(600))
	})
	a.shown = allKey
	Settings.Sort = OldestFirst
	defer func() { Settings.Sort = NewestFirst }()
	selected := func() []int {
		var res []int
		for i := range a.grid.Photos {
			if a.grid.Selected[i] {
				res = append(res, i)
			}
		}
		return res
	}
	click := func(idx int, mods key.Modifiers) {
		b.AsyncLock()
		pt := a.grid.cellRect(idx).Min.Add(image.Pt(10, 10))
		b.AsyncUnlock()
		ev := a.grid.SystemEvents()
		ev.MouseButton(events.MouseDown, events.Left, pt, mods)
		ev.MouseButton(events.MouseUp, events.Left, pt, mods)
		time.Sleep(800 * time.Millisecond) // not a double click
	}
	var cmd key.Modifiers
	cmd.SetFlag(true, key.Meta)
	var shift key.Modifiers
	shift.SetFlag(true, key.Shift)
	core.RenderImage(b, func() {
		b.AsyncLock()
		a.lib = lib
		a.grid.lib = lib
		a.grid.CellSize = units.Dp(80) // all of them in view
		a.refresh()
		b.AsyncUnlock()
	}, func() {
		click(2, 0)
		click(5, cmd)
		b.AsyncLock()
		assert.Equal(t, []int{2, 5}, selected(), "Command adds one")
		b.AsyncUnlock()
		click(9, shift)
		b.AsyncLock()
		assert.Equal(t, []int{2, 5, 6, 7, 8, 9}, selected(), "Shift adds the range from the nearest before")
		b.AsyncUnlock()
		click(0, shift)
		b.AsyncLock()
		assert.Equal(t, []int{0, 1, 2, 5, 6, 7, 8, 9}, selected(), "Shift adds the range from the nearest after")
		b.AsyncUnlock()
		click(5, cmd)
		b.AsyncLock()
		assert.Equal(t, []int{0, 1, 2, 6, 7, 8, 9}, selected(), "Command removes one")
		b.AsyncUnlock()
		click(4, 0)
		b.AsyncLock()
		assert.Equal(t, []int{4}, selected(), "a click selects only it")
		opened := 0
		a.grid.OnFirst(events.DoubleClick, func(e events.Event) { // before that of the app, a window
			if !e.IsHandled() {
				opened++
				e.SetHandled()
			}
		})
		b.AsyncUnlock()
		pt := func(idx int) image.Point {
			b.AsyncLock()
			defer b.AsyncUnlock()
			return a.grid.cellRect(idx).Min.Add(image.Pt(10, 10))
		}
		ev := a.grid.SystemEvents()
		for range 2 { // a double click on 7
			ev.MouseButton(events.MouseDown, events.Left, pt(7), 0)
			ev.MouseButton(events.MouseUp, events.Left, pt(7), 0)
		}
		time.Sleep(200 * time.Millisecond)
		b.AsyncLock()
		assert.Equal(t, 1, opened, "a double click on a photo opens it")
		assert.Equal(t, []int{7}, selected())
		b.AsyncUnlock()
	})
}

// TestGridSelectedShot renders a grid with several photos selected, saved
// in PHOTOS_SHOT_IMAGE, for looking at how they are shown.
func TestGridSelectedShot(t *testing.T) {
	fn := os.Getenv("PHOTOS_SHOT_IMAGE")
	if fn == "" {
		t.Skip("PHOTOS_SHOT_IMAGE not set")
	}
	lib := smallLibrary(t, 12)
	b := core.NewBody()
	a := NewApp(b)
	a.shown = allKey
	img := core.RenderImage(b, func() {
		b.AsyncLock()
		a.lib = lib
		a.grid.lib = lib
		a.grid.CellSize = units.Dp(100)
		a.refresh()
		b.AsyncUnlock()
		time.Sleep(time.Second)
		b.AsyncLock()
		a.grid.Selected = map[int]bool{1: true, 2: true, 5: true}
		a.grid.Current = 5
		a.grid.NeedsRender()
		b.AsyncUnlock()
	})
	imagex.Save(img, fn)
}
