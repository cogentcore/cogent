// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package canvas

import (
	"image"
	"testing"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/svg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// dragMods drags the mouse as [core.DragMouse] does, with the given
// modifier keys held down, and longer waits, so that the event loop has
// started the drag and handled each move on a slow machine (as in CI or
// with -race), where the moves are otherwise merged before the drag
// starts, and the canvas gets no drag move.
func dragMods(b *core.Body, from, to image.Point, mods ...key.Modifiers) {
	var m key.Modifiers
	for _, mod := range mods {
		m.SetFlag(true, mod)
	}
	ev := sysEvents(b)
	ev.MouseMove(from)
	ev.MouseButton(events.MouseDown, events.Left, from, m)
	time.Sleep(max(core.TimingSettings.DragStartTime, core.TimingSettings.SlideStartTime) + 20*time.Millisecond)
	waitHandled(b)
	for i := 1; i <= 5; i++ {
		ev.MouseMove(from.Add(to.Sub(from).Mul(i).Div(5)))
		waitHandled(b) // (each one, before the next: not merged)
	}
	ev.MouseButton(events.MouseUp, events.Left, to, m)
}

// waitHandled waits for the event loop of the window of the given body to
// handle the events sent to it, as best it can from outside: the loop holds
// the lock of the window while it handles an event and renders, so taking
// the lock, after the time for the loop to get the event, waits for those.
func waitHandled(b *core.Body) {
	for range 3 {
		time.Sleep(15 * time.Millisecond)
		b.AsyncLock()
		b.AsyncUnlock()
	}
}

// handle returns the center of the given reshape handle of the selection.
func handle(t *testing.T, cv *Canvas, sp Sprites) image.Point {
	t.Helper()
	s, ok := cv.SVG.SpritesNoLock().Final.AtTry(SpriteName(SpReshapeBBox, sp, 0))
	require.True(t, ok, sp.String())
	r := s.EventBBox
	return r.Min.Add(r.Max).Div(2)
}

// dragEvent returns a mouse drag event from the given start to the given
// point, for calling the drag functions directly.
func dragEvent(start, to image.Point, mods ...key.Modifiers) events.Event {
	var m key.Modifiers
	for _, mod := range mods {
		m.SetFlag(true, mod)
	}
	e := events.NewMouseDrag(events.Left, to, start, start, m)
	e.Init()
	return e
}

// TestReshape tests reshaping the selection by dragging a handle, and by
// the drags of each of the handles.
func TestReshape(t *testing.T) {
	checks := map[Sprites]func(o, n math32.Box2) bool{
		SpUpL: func(o, n math32.Box2) bool { return n.Min.X < o.Min.X-20 && n.Min.Y < o.Min.Y-20 },
		SpUpC: func(o, n math32.Box2) bool { return n.Min.Y < o.Min.Y-20 && n.Min.X == o.Min.X },
		SpUpR: func(o, n math32.Box2) bool { return n.Max.X > o.Max.X+20 && n.Min.Y < o.Min.Y-20 },
		SpDnL: func(o, n math32.Box2) bool { return n.Min.X < o.Min.X-20 && n.Max.Y > o.Max.Y+20 },
		SpDnC: func(o, n math32.Box2) bool { return n.Max.Y > o.Max.Y+20 && n.Max.X == o.Max.X },
		SpDnR: func(o, n math32.Box2) bool { return n.Max.X > o.Max.X+20 && n.Max.Y > o.Max.Y+20 },
		SpLfM: func(o, n math32.Box2) bool { return n.Min.X < o.Min.X-20 && n.Min.Y == o.Min.Y },
		SpRtM: func(o, n math32.Box2) bool { return n.Max.X > o.Max.X+20 && n.Max.Y == o.Max.Y },
	}
	dirs := map[Sprites]image.Point{SpUpL: {-30, -30}, SpUpC: {0, -30}, SpUpR: {30, -30}, SpDnL: {-30, 30}, SpDnC: {0, 30}, SpDnR: {30, 30}, SpLfM: {-30, 0}, SpRtM: {30, 0}}
	var orig math32.Box2
	var from image.Point
	bbox := func(cv *Canvas) math32.Box2 { return node(t, cv, "rect318").AsNodeBase().BBox }
	runCanvas(t, "shapes.svg", func(cv *Canvas) {
		selectNodes(t, cv, "rect318")
		orig = bbox(cv)
	}, func(cv *Canvas) { // after the handles are rendered
		from = handle(t, cv, SpDnR)
	}, unlocked(func(b *core.Body, cv *Canvas) {
		dragMods(b, from, from.Add(dirs[SpDnR]))
	}), func(cv *Canvas) {
		assert.True(t, checks[SpDnR](orig, bbox(cv)), "dragged: %v -> %v", orig, bbox(cv))
		assert.Equal(t, "Reshape", cv.Undo())
		sv := cv.SVG
		es := &cv.EditState
		for sp := SpUpL; sp <= SpRtM; sp++ {
			selectNodes(t, cv, "rect318")
			orig = bbox(cv)
			start := image.Pt(500, 500)
			es.DragSelStart(start)
			sv.SpriteReshapeDrag(sp, dragEvent(start, start.Add(dirs[sp])))
			sv.ManipDone()
			assert.True(t, checks[sp](orig, bbox(cv)), "%v: %v -> %v", sp, orig, bbox(cv))
			cv.Undo()
		}
	})
}

// TestRotate tests rotating the selection by dragging a handle with Alt,
// and the rotating drags of each of the handles.
func TestRotate(t *testing.T) {
	var from image.Point
	runCanvas(t, "shapes.svg", func(cv *Canvas) {
		selectNodes(t, cv, "rect318")
	}, func(cv *Canvas) {
		from = handle(t, cv, SpUpR)
	}, unlocked(func(b *core.Body, cv *Canvas) {
		dragMods(b, from, from.Add(image.Pt(25, -35)), key.Alt)
	}), func(cv *Canvas) {
		assert.False(t, node(t, cv, "rect318").AsNodeBase().Paint.Transform.IsIdentity(), "rotated")
		assert.Equal(t, "Rotate", cv.Undo())
		sv := cv.SVG
		es := &cv.EditState
		for sp := SpUpL; sp <= SpRtM; sp++ {
			for _, mods := range [][]key.Modifiers{nil, {key.Shift}} { // Shift: no snapping to 15 degrees
				selectNodes(t, cv, "rect318")
				start := image.Pt(500, 500)
				es.DragSelStart(start)
				sv.SpriteRotateDrag(sp, dragEvent(start, start.Add(image.Pt(25, -35)), mods...))
				sv.ManipDone()
				assert.Equal(t, "Rotate", cv.Undo(), sp.String())
			}
		}
	})
}

// TestConstrain tests constraining moves to an axis or diagonal with Control.
func TestConstrain(t *testing.T) {
	var p318 image.Point
	var bb math32.Box2
	runCanvas(t, "shapes.svg", func(cv *Canvas) {
		p318 = center(node(t, cv, "rect318"))
		bb = node(t, cv, "rect318").AsNodeBase().BBox
		selectNodes(t, cv, "rect318") // not by a click, which would make the press a double click
	}, unlocked(func(b *core.Body, cv *Canvas) {
		dragMods(b, p318, p318.Add(image.Pt(60, 5)), key.Control)
	}), func(cv *Canvas) {
		nb := node(t, cv, "rect318").AsNodeBase().BBox
		assert.InDelta(t, bb.Min.Y, nb.Min.Y, float64(Settings.SnapZone+1), "constrained to horizontal (but snapped)")
		assert.Greater(t, nb.Min.X, bb.Min.X+40)
		cv.SVG.SpritesNoLock().Lock()
		pt, diag := cv.SVG.ConstrainPoint(math32.Vec2(0, 0), math32.Vec2(10, 9))
		cv.SVG.SpritesNoLock().Unlock()
		assert.True(t, diag)
		assert.Equal(t, pt.X, pt.Y)
		cv.SVG.SpritesNoLock().Lock()
		pt, diag = cv.SVG.ConstrainPoint(math32.Vec2(0, 0), math32.Vec2(-10, 9))
		cv.SVG.SpritesNoLock().Unlock()
		assert.True(t, diag)
		assert.Equal(t, pt.X, -pt.Y)
	})
}

// TestBBoxPoints tests the points of bounding boxes and the box shapes.
func TestBBoxPoints(t *testing.T) {
	bb := math32.Box2{Min: math32.Vec2(0, 0), Max: math32.Vec2(10, 20)}
	vals := map[BBoxPoints]float32{BBLeft: 0, BBCenter: 5, BBRight: 10, BBTop: 0, BBMiddle: 10, BBBottom: 20}
	for p, v := range vals {
		assert.Equal(t, v, p.ValueBox(bb), p.String())
		b := bb
		p.SetValueBox(&b, 3)
		if p != BBCenter && p != BBMiddle {
			assert.Equal(t, float32(3), p.ValueBox(b), p.String())
		}
		b = bb
		p.MoveDelta(&b, 2)
		assert.Equal(t, bb.Size(), b.Size(), "moved")
		assert.Equal(t, v+2, p.ValueBox(b))
	}
	assert.Equal(t, math32.X, BBCenter.Dim())
	assert.Equal(t, math32.Y, BBMiddle.Dim())
	assert.Equal(t, math32.Vec2(0, 10), BBLeft.PointBox(bb))
	for sp := SpUpL; sp <= SpRtM; sp++ {
		x, y := ReshapeBBoxPoints(sp)
		b := bb
		BBoxReshapeDelta(&b, 1, x, y)
		assert.NotEqual(t, bb, b, sp.String())
	}
	x, y := ReshapeBBoxPoints(SpNone)
	assert.Equal(t, BBLeft, x)
	assert.Equal(t, BBLeft, y)
	assert.Equal(t, math32.Vec2(20, 20), SquareBBox(bb).Size())
	assert.Equal(t, math32.Vec2(10, 10), SquareBBox(math32.Box2{Max: math32.Vec2(10, 5)}).Size())
	assert.Equal(t, math32.Vec2(20, 40), ProportionalBBox(math32.Box2{Max: math32.Vec2(5, 40)}, bb).Size())
	assert.Equal(t, math32.Vec2(40, 80), ProportionalBBox(math32.Box2{Max: math32.Vec2(40, 5)}, bb).Size())
	assert.Equal(t, bb, ProportionalBBox(bb, math32.Box2{}))
	v, ok := SnapToPoint(10, 11)
	assert.True(t, ok)
	assert.Equal(t, float32(11), v)
	_, ok = SnapToPoint(10, 100)
	assert.False(t, ok)
	assert.Equal(t, float32(12), SnapToIncr(11, 0, 4))
}

// TestSprites tests the names and properties of sprites.
func TestSprites(t *testing.T) {
	for _, typ := range []Sprites{SpReshapeBBox, SpSelBBox, SpNodePoint, SpNodeCtrl, SpRubberBand, SpAlignMatch, SpLineAdd} {
		sp := core.NewSprite("", nil)
		SetSpriteProperties(sp, typ, SpUpL, 2)
		ty, st, idx := SpriteProperties(sp)
		assert.Equal(t, typ, ty)
		assert.Equal(t, SpUpL, st)
		assert.Equal(t, 2, idx)
		assert.NotEmpty(t, SpriteName(typ, SpUpL, 2))
	}
	ty, _, _ := SpriteProperties(&core.Sprite{})
	assert.Equal(t, SpNone, ty)
	sp := core.NewSprite("", nil)
	sp.InitProperties()
	ty, _, _ = SpriteProperties(sp)
	assert.Equal(t, SpNone, ty)
	runCanvas(t, "shapes.svg", func(cv *Canvas) {
		selectNodes(t, cv, "rect59", "rect318") // selection boxes for each
		_, ok := SpriteByName(cv.SVG, SpSelBBox, SpUpL, 0)
		assert.True(t, ok)
	})
}

// TestGroupLeaves tests the leaves of groups in files with groups.
func TestGroupLeaves(t *testing.T) {
	runCanvas(t, "rectgp.svg", func(cv *Canvas) {
		es := &cv.EditState
		var gp svg.Node
		for _, k := range cv.SVG.Root().Children {
			if g, ok := k.(*svg.Group); ok {
				gp = g
			}
		}
		if gp == nil {
			return
		}
		es.ResetSelected()
		es.Select(gp)
		cv.SelectUnGroup()
		assert.False(t, es.HasSelected() && es.FirstSelected() == gp)
	})
}
