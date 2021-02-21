// Copyright (c) 2021, The Grid Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grid

import (
	"fmt"
	"image"
	"math"
	"strings"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/gi/svg"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ints"
	"github.com/goki/mat32"
)

// ManipStart is called at the start of a manipulation, saving the state prior to the action
func (sv *SVGView) ManipStart(act, data string) {
	es := sv.EditState()
	es.ActStart(act, data)
	help := ActionHelpMap[act]
	sv.GridView.SetStatus(fmt.Sprintf("<b>%s</b>: %s", act, help))
	sv.UndoSave(act, data)
	es.ActUnlock()
}

// ManipDone happens when a manipulation has finished: resets action, does render
func (sv *SVGView) ManipDone() {
	win := sv.GridView.ParentWindow()
	InactivateSpriteRange(win, AlignMatch1, AlignMatch8)
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
	}
	es.DragReset()
	es.ActDone()
	sv.GridView.UpdateSelectToolbar()
	sv.UpdateSig()
	win.RenderOverlays()
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

// GridDots is the current grid spacing and offset in dots
func (sv *SVGView) GridDots() (float32, float32) {
	grid := sv.GridView.Prefs.Grid
	if grid <= 0 {
		grid = 12
	}
	un := units.NewValue(float32(grid), sv.GridView.Prefs.Units)
	un.ToDots(&sv.Pnt.UnContext)
	incr := un.Dots * sv.Scale // our zoom factor
	// todo: offset!
	return incr, 0
}

// SnapToPt snaps value to given potential snap point, in screen pixel units.
// Tolerance is determined by preferences.  Returns true if snapped.
func SnapToPt(val, snap float32) (float32, bool) {
	d := mat32.Abs(val - snap)
	if d <= float32(Prefs.SnapTol) {
		return snap, true
	}
	return val, false
}

// SnapToIncr snaps value to given increment, first subtracting given offset.
// Tolerance is determined by preferences, which is in screen pixels.
// Returns true if snapped.
func SnapToIncr(val, off, incr float32) (float32, bool) {
	nint := mat32.Round((val-off)/incr)*incr + off
	dint := mat32.Abs(val - nint)
	if dint <= float32(Prefs.SnapTol) {
		return nint, true
	}
	return val, false
}

// SnapCurBBox does snapping on current bbox according to preferences
// if move is true, then is for moving, else reshaping, with given target points
// in X and Y axes
func (sv *SVGView) SnapCurBBox(move bool, trgX, trgY BBoxPoints) {
	es := sv.EditState()
	snapped := false
	if Prefs.SnapGuide {
		clDst := [2]float32{float32(math.MaxFloat32), float32(math.MaxFloat32)}
		var clPts [2][]BBoxPoints
		var clVals [2][]mat32.Vec2
		var bbval [2]mat32.Vec2
		for ap := BBLeft; ap < BBoxPointsN; ap++ {
			if !move && (ap != trgX && ap != trgY) {
				continue
			}
			bbp := ap.PointBox(es.DragSelCurBBox)
			pts := es.AlignPts[ap]
			dim := ap.Dim()
			for _, pt := range pts {
				pv := pt.Dim(dim)
				bv := bbp.Dim(dim)
				dst := mat32.Abs(pv - bv)
				if dst < clDst[dim] {
					clDst[dim] = dst
					clPts[dim] = []BBoxPoints{ap}
					clVals[dim] = []mat32.Vec2{pt}
					bbval[dim] = bbp
				} else if mat32.Abs(dst-clDst[dim]) < 1.0e-4 {
					clPts[dim] = append(clPts[dim], ap)
					clVals[dim] = append(clVals[dim], pt)
				}
			}
		}
		var alpts []image.Rectangle
		var altyps []BBoxPoints
		for dim := mat32.X; dim <= mat32.Y; dim++ {
			if len(clVals[dim]) == 0 {
				continue
			}
			bv := bbval[dim].Dim(dim)
			sval, snap := SnapToPt(bv, clVals[dim][0].Dim(dim))
			if snap {
				if move {
					clPts[dim][0].MoveDelta(&es.DragSelEffBBox, sval-bv)
				} else {
					BBoxReshapeDelta(&es.DragSelEffBBox, sval-bv, trgX, trgY)
				}
				mx := ints.MinInt(len(clVals[dim]), 4)
				for i := 0; i < mx; i++ {
					pt := clVals[dim][i]
					rpt := image.Rectangle{}
					rpt.Min = bbval[dim].ToPoint()
					rpt.Max = pt.ToPoint()
					if dim == mat32.X {
						rpt.Min.X = rpt.Max.X
					} else {
						rpt.Min.Y = rpt.Max.Y
					}
					alpts = append(alpts, rpt)
					altyps = append(altyps, clPts[dim][i])
				}
				snapped = true
			}
		}
		sv.ShowAlignMatches(alpts, altyps)
	}
	if !snapped && Prefs.SnapGrid {
		// grinc, groff := sv.GridDots()
		// todo: moving check Min, else ?
	}
}

// ConstrainCurBBox constrains bounding box to dimension with smallest change
// e.g., when using control key.
func (sv *SVGView) ConstrainCurBBox(move bool, trgX, trgY BBoxPoints) {
	es := sv.EditState()
	dmin := es.DragSelCurBBox.Min.Sub(es.DragSelStartBBox.Min)
	dmax := es.DragSelCurBBox.Max.Sub(es.DragSelStartBBox.Max)
	bb := mat32.Box2{Min: dmin, Max: dmax}

	var alpts []image.Rectangle
	var altyps []BBoxPoints

	xval := trgX.ValBox(bb)
	yval := trgY.ValBox(bb)
	if mat32.Abs(yval) < mat32.Abs(xval) {
		trgY.SetValBox(&es.DragSelEffBBox, trgY.ValBox(es.DragSelStartBBox))
		if move {
			es.DragSelEffBBox.Max.Y = es.DragSelStartBBox.Max.Y
		}
		rpt := image.Rectangle{}
		rpt.Min.X = int(trgX.ValBox(es.DragSelStartBBox))
		rpt.Min.Y = int(trgY.ValBox(es.DragSelStartBBox))
		rpt.Max = rpt.Min
		rpt.Max.X = int(trgX.ValBox(es.DragSelEffBBox))
		alpts = append(alpts, rpt)
		altyps = append(altyps, trgY)
	} else {
		trgX.SetValBox(&es.DragSelEffBBox, trgX.ValBox(es.DragSelStartBBox))
		if move {
			es.DragSelEffBBox.Max.X = es.DragSelStartBBox.Max.X
		}
		rpt := image.Rectangle{}
		rpt.Min.X = int(trgX.ValBox(es.DragSelStartBBox))
		rpt.Min.Y = int(trgY.ValBox(es.DragSelStartBBox))
		rpt.Max = rpt.Min
		rpt.Max.Y = int(trgY.ValBox(es.DragSelEffBBox))
		alpts = append(alpts, rpt)
		altyps = append(altyps, trgX)
	}
	sv.ShowAlignMatches(alpts, altyps)
}

// DragMove is when dragging a selection for moving
func (sv *SVGView) DragMove(delta image.Point, win *gi.Window, me *mouse.DragEvent) {
	es := sv.EditState()
	dv := mat32.NewVec2FmPoint(delta)
	svoff := mat32.NewVec2FmPoint(sv.WinBBox.Min)

	es.DragSelCurBBox.Min.SetAdd(dv)
	es.DragSelCurBBox.Max.SetAdd(dv)

	if !es.InAction() {
		sv.ManipStart("Move", es.SelectedNamesString())
		sv.GatherAlignPoints()
	}

	InactivateSpriteRange(win, AlignMatch1, AlignMatch8)
	es.DragSelEffBBox = es.DragSelCurBBox
	switch {
	case me.HasAnyModifier(key.Alt):
	case me.HasAnyModifier(key.Control):
		sv.ConstrainCurBBox(true, BBLeft, BBTop) // move
	default:
		sv.SnapCurBBox(true, BBLeft, BBTop) // move
	}

	pt := es.DragSelStartBBox.Min.Sub(svoff)
	tdel := es.DragSelEffBBox.Min.Sub(es.DragSelStartBBox.Min)
	for itm, ss := range es.Selected {
		itm.ReadGeom(ss.InitGeom)
		itm.ApplyDeltaXForm(tdel, mat32.Vec2{1, 1}, 0, pt)
	}
	sv.SetSelSprites(es.DragSelEffBBox)
	go sv.ManipUpdate()
	win.RenderOverlays()

}

// SpriteReshapeDrag processes a mouse reshape drag event on a selection sprite
func (sv *SVGView) SpriteReshapeDrag(sp Sprites, delta image.Point, win *gi.Window, me *mouse.DragEvent) {
	es := sv.EditState()
	if !es.InAction() {
		sv.ManipStart("Reshape", es.SelectedNamesString())
	}
	stsz := es.DragSelStartBBox.Size()
	stpos := es.DragSelStartBBox.Min
	dv := mat32.NewVec2FmPoint(delta)
	switch sp {
	case ReshapeUpL:
		es.DragSelCurBBox.Min.SetAdd(dv)
	case ReshapeUpC:
		es.DragSelCurBBox.Min.Y += dv.Y
	case ReshapeUpR:
		es.DragSelCurBBox.Min.Y += dv.Y
		es.DragSelCurBBox.Max.X += dv.X
	case ReshapeDnL:
		es.DragSelCurBBox.Min.X += dv.X
		es.DragSelCurBBox.Max.Y += dv.Y
	case ReshapeDnC:
		es.DragSelCurBBox.Max.Y += dv.Y
	case ReshapeDnR:
		es.DragSelCurBBox.Max.SetAdd(dv)
	case ReshapeLfM:
		es.DragSelCurBBox.Min.X += dv.X
	case ReshapeRtM:
		es.DragSelCurBBox.Max.X += dv.X
	}
	es.DragSelCurBBox.Min.SetMin(es.DragSelCurBBox.Max.SubScalar(1)) // don't allow flipping

	InactivateSpriteRange(win, AlignMatch1, AlignMatch8)
	es.DragSelEffBBox = es.DragSelCurBBox
	bbX, bbY := ReshapeBBoxPoints(sp)
	switch {
	case me.HasAnyModifier(key.Control):
		if bbX != BBCenter && bbY != BBMiddle {
			sv.ConstrainCurBBox(false, bbX, bbY) // reshape
		}
	default:
		sv.SnapCurBBox(false, bbX, bbY) // reshape
	}

	npos := es.DragSelEffBBox.Min
	nsz := es.DragSelEffBBox.Size()
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

	sv.SetSelSprites(es.DragSelEffBBox)
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
	case ReshapeUpL:
		es.DragSelCurBBox.Min.SetAdd(dv)
		dy = es.DragSelStartBBox.Min.Y - es.DragSelCurBBox.Min.Y
		dx = es.DragSelStartBBox.Max.X - es.DragSelCurBBox.Min.X
		pt.X = es.DragSelStartBBox.Max.X
	case ReshapeUpC:
		es.DragSelCurBBox.Min.Y += dv.Y
		es.DragSelCurBBox.Max.X += dv.X
		dy = es.DragSelCurBBox.Min.Y - es.DragSelStartBBox.Min.Y
		dx = es.DragSelCurBBox.Max.X - es.DragSelStartBBox.Min.X
		pt = ctr
	case ReshapeUpR:
		es.DragSelCurBBox.Min.Y += dv.Y
		es.DragSelCurBBox.Max.X += dv.X
		dy = es.DragSelCurBBox.Min.Y - es.DragSelStartBBox.Min.Y
		dx = es.DragSelCurBBox.Max.X - es.DragSelStartBBox.Min.X
		pt = es.DragSelStartBBox.Min
	case ReshapeDnL:
		es.DragSelCurBBox.Min.X += dv.X
		es.DragSelCurBBox.Max.Y += dv.Y
		dy = es.DragSelStartBBox.Max.Y - es.DragSelCurBBox.Max.Y
		dx = es.DragSelStartBBox.Max.X - es.DragSelCurBBox.Min.X
		pt = es.DragSelStartBBox.Max
	case ReshapeDnC:
		es.DragSelCurBBox.Max.SetAdd(dv)
		dy = es.DragSelCurBBox.Max.Y - es.DragSelStartBBox.Max.Y
		dx = es.DragSelCurBBox.Max.X - es.DragSelStartBBox.Min.X
		pt = ctr
	case ReshapeDnR:
		es.DragSelCurBBox.Max.SetAdd(dv)
		dy = es.DragSelCurBBox.Max.Y - es.DragSelStartBBox.Max.Y
		dx = es.DragSelCurBBox.Max.X - es.DragSelStartBBox.Min.X
		pt.X = es.DragSelStartBBox.Min.X
		pt.Y = es.DragSelStartBBox.Max.Y
	case ReshapeLfM:
		es.DragSelCurBBox.Min.X += dv.X
		es.DragSelCurBBox.Max.Y += dv.Y
		dy = es.DragSelStartBBox.Max.Y - es.DragSelCurBBox.Max.Y
		dx = es.DragSelStartBBox.Max.X - es.DragSelCurBBox.Min.X
		pt = ctr
	case ReshapeRtM:
		es.DragSelCurBBox.Max.SetAdd(dv)
		dy = es.DragSelCurBBox.Max.Y - es.DragSelStartBBox.Max.Y
		dx = es.DragSelCurBBox.Max.X - es.DragSelStartBBox.Min.X
		pt = ctr
	}
	ang := mat32.Atan2(dy, dx)
	ang, _ = SnapToIncr(mat32.RadToDeg(ang), 0, 15)
	ang = mat32.DegToRad(ang)
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
