// Copyright (c) 2021, The Grid Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grid

import (
	"image"
	"strings"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/svg"
	"github.com/goki/mat32"
)

// ManipStart is called at the start of a manipulation, saving the state prior to the action
func (sv *SVGView) ManipStart(act, data string) {
	es := sv.EditState()
	es.ActStart(act, data)
	// astr := act + ": " +
	// sv.GridView.SetStatus(fmt.Sprintf("save undo: %s: %s", act, data))
	sv.UndoSave(act, data)
	es.ActUnlock()
}

// ManipDone happens when a manipulation has finished: resets action, does render
func (sv *SVGView) ManipDone() {
	win := sv.GridView.ParentWindow()
	es := sv.EditState()
	switch {
	case es.Action == "BoxSelect":
		bbox := image.Rectangle{Min: es.DragStartPos, Max: es.DragCurPos}
		bbox = bbox.Canon()
		InactivateSprites(win)
		win.RenderOverlays()
		sel := sv.SelectWithinBBox(bbox, false)
		if len(sel) > 0 {
			es.ResetSelected() // todo: extend select -- need mouse mod
			for _, se := range sel {
				es.Select(se)
			}
			sv.GridView.UpdateTabs()
			sv.UpdateSelSprites()
			sv.EditState().DragSelStart(es.DragCurPos)
		}
		win.RenderOverlays()
	}
	es.DragReset()
	es.ActDone()
	sv.GridView.UpdateSelectToolbar()
	sv.UpdateSig()
}

// ManipUpdate is called from goroutine: 'go sv.ManipUpdate()' to update the
// current display while manipulating.  It checks if already rendering and if so,
// just returns immediately, so that updates are not stacked up and laggy.
func (sv *SVGView) ManipUpdate() {
	if sv.IsRendering() {
		return
	}
	sv.UpdateSig()
}

// SnapTo snaps value to given increment, first subtracting given offset.
// Tolerance is determined by preferences.
func (sv *SVGView) SnapTo(val, off, incr float32) float32 {
	tol := Prefs.SnapTol
	nint := mat32.Round((val-off)/incr)*incr + off
	dint := mat32.Abs(val - nint)
	if dint < tol*incr {
		return nint
	}
	return val
}

// SpriteReshapeDrag processes a mouse reshape drag event on a selection sprite
func (sv *SVGView) SpriteReshapeDrag(sp Sprites, delta image.Point, win *gi.Window) {
	es := sv.EditState()
	if !es.InAction() {
		sv.ManipStart("Reshape", es.SelectedNamesString())
	}
	stsz := es.DragSelStartBBox.Size()
	stpos := es.DragSelStartBBox.Min
	dv := mat32.NewVec2FmPoint(delta)
	switch sp {
	case SizeUpL:
		es.DragSelCurBBox.Min.SetAdd(dv)
	case SizeUpM:
		es.DragSelCurBBox.Min.Y += dv.Y
	case SizeUpR:
		es.DragSelCurBBox.Min.Y += dv.Y
		es.DragSelCurBBox.Max.X += dv.X
	case SizeDnL:
		es.DragSelCurBBox.Min.X += dv.X
		es.DragSelCurBBox.Max.Y += dv.Y
	case SizeDnM:
		es.DragSelCurBBox.Max.Y += dv.Y
	case SizeDnR:
		es.DragSelCurBBox.Max.SetAdd(dv)
	case SizeLfC:
		es.DragSelCurBBox.Min.X += dv.X
	case SizeRtC:
		es.DragSelCurBBox.Max.X += dv.X
	}
	es.DragSelCurBBox.Min.SetMin(es.DragSelCurBBox.Max.SubScalar(1)) // don't allow flipping
	npos := es.DragSelCurBBox.Min
	nsz := es.DragSelCurBBox.Size()
	svoff := mat32.NewVec2FmPoint(sv.WinBBox.Min)
	pt := es.DragSelStartBBox.Min.Sub(svoff)
	del := npos.Sub(stpos)
	sc := nsz.Div(stsz)
	for itm, ss := range es.Selected {
		itm.ReadGeom(ss.InitGeom)
		itm.ApplyDeltaXForm(del, sc, 0, pt)
		if strings.HasPrefix(es.Action, "New") {
			svg.UpdateNodeGradientPoints(itm, "fill")
			svg.UpdateNodeGradientPoints(itm, "stroke")
		}
	}

	sv.SetSelSprites(es.DragSelCurBBox)
	go sv.ManipUpdate()
	win.RenderOverlays()
}

// SpriteRotateDrag processes a mouse rotate drag event on a selection sprite
func (sv *SVGView) SpriteRotateDrag(sp Sprites, delta image.Point, win *gi.Window) {
	es := sv.EditState()
	if !es.InAction() {
		sv.ManipStart("Rotate", es.SelectedNamesString())
	}
	dv := mat32.NewVec2FmPoint(delta)
	pt := es.DragSelStartBBox.Min
	ctr := es.DragSelStartBBox.Min.Add(es.DragSelStartBBox.Max).MulScalar(.5)
	var dx, dy float32
	switch sp {
	case SizeUpL:
		es.DragSelCurBBox.Min.SetAdd(dv)
		dy = es.DragSelStartBBox.Min.Y - es.DragSelCurBBox.Min.Y
		dx = es.DragSelStartBBox.Max.X - es.DragSelCurBBox.Min.X
		pt.X = es.DragSelStartBBox.Max.X
	case SizeUpM:
		es.DragSelCurBBox.Min.Y += dv.Y
		es.DragSelCurBBox.Max.X += dv.X
		dy = es.DragSelCurBBox.Min.Y - es.DragSelStartBBox.Min.Y
		dx = es.DragSelCurBBox.Max.X - es.DragSelStartBBox.Min.X
		pt = ctr
	case SizeUpR:
		es.DragSelCurBBox.Min.Y += dv.Y
		es.DragSelCurBBox.Max.X += dv.X
		dy = es.DragSelCurBBox.Min.Y - es.DragSelStartBBox.Min.Y
		dx = es.DragSelCurBBox.Max.X - es.DragSelStartBBox.Min.X
		pt = es.DragSelStartBBox.Min
	case SizeDnL:
		es.DragSelCurBBox.Min.X += dv.X
		es.DragSelCurBBox.Max.Y += dv.Y
		dy = es.DragSelStartBBox.Max.Y - es.DragSelCurBBox.Max.Y
		dx = es.DragSelStartBBox.Max.X - es.DragSelCurBBox.Min.X
		pt = es.DragSelStartBBox.Max
	case SizeDnM:
		es.DragSelCurBBox.Max.SetAdd(dv)
		dy = es.DragSelCurBBox.Max.Y - es.DragSelStartBBox.Max.Y
		dx = es.DragSelCurBBox.Max.X - es.DragSelStartBBox.Min.X
		pt = ctr
	case SizeDnR:
		es.DragSelCurBBox.Max.SetAdd(dv)
		dy = es.DragSelCurBBox.Max.Y - es.DragSelStartBBox.Max.Y
		dx = es.DragSelCurBBox.Max.X - es.DragSelStartBBox.Min.X
		pt.X = es.DragSelStartBBox.Min.X
		pt.Y = es.DragSelStartBBox.Max.Y
	case SizeLfC:
		es.DragSelCurBBox.Min.X += dv.X
		es.DragSelCurBBox.Max.Y += dv.Y
		dy = es.DragSelStartBBox.Max.Y - es.DragSelCurBBox.Max.Y
		dx = es.DragSelStartBBox.Max.X - es.DragSelCurBBox.Min.X
		pt = ctr
	case SizeRtC:
		es.DragSelCurBBox.Max.SetAdd(dv)
		dy = es.DragSelCurBBox.Max.Y - es.DragSelStartBBox.Max.Y
		dx = es.DragSelCurBBox.Max.X - es.DragSelStartBBox.Min.X
		pt = ctr
	}
	ang := mat32.Atan2(dy, dx)
	ang = mat32.DegToRad(sv.SnapTo(mat32.RadToDeg(ang), 0, 15))
	svoff := mat32.NewVec2FmPoint(sv.WinBBox.Min)
	pt = pt.Sub(svoff)
	del := mat32.Vec2{}
	sc := mat32.Vec2{1, 1}
	for itm, ss := range es.Selected {
		itm.ReadGeom(ss.InitGeom)
		itm.ApplyDeltaXForm(del, sc, ang, pt)
		if strings.HasPrefix(es.Action, "New") {
			svg.UpdateNodeGradientPoints(itm, "fill")
			svg.UpdateNodeGradientPoints(itm, "stroke")
		}
	}

	sv.SetSelSprites(es.DragSelCurBBox)
	go sv.ManipUpdate()
	win.RenderOverlays()
}
