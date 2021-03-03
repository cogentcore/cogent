// Copyright (c) 2021, The Grid Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grid

import (
	"fmt"
	"image"
	"math"
	"strings"

	"github.com/chewxy/math32"
	"github.com/goki/gi/gi"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/gi/svg"
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
	InactivateSprites(win, SpAlignMatch)
	es := sv.EditState()
	switch {
	case es.Action == "BoxSelect":
		bbox := image.Rectangle{Min: es.DragStartPos, Max: es.DragCurPos}
		bbox = bbox.Canon()
		InactivateSprites(win, SpRubberBand)
		win.RenderOverlays()
		sel := sv.SelectWithinBBox(bbox, false)
		if len(sel) > 0 {
			es.ResetSelected() // todo: extend select -- need mouse mod
			for _, se := range sel {
				es.Select(se)
			}
		}
	default:
	}
	es.DragReset()
	es.ActDone()
	sv.UpdateView(true)
	sv.UpdateSelect()
	sv.GridView.ChangeMade()
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

// GridDots is the current grid spacing and offsets in dots
func (sv *SVGView) GridDots() (float32, mat32.Vec2) {
	svoff := mat32.NewVec2FmPoint(sv.WinBBox.Min)
	grid := sv.GridEff
	if grid <= 0 {
		grid = 12
	}
	incr := grid * sv.Scale // our zoom factor

	org := mat32.Vec2{}
	org = sv.Pnt.XForm.MulVec2AsPt(org)

	// fmt.Printf("org: %v\n", org)

	org.SetAdd(svoff)
	// fmt.Printf("org: %v   svgoff: %v\n", org, svoff)

	org.X = math32.Mod(org.X, incr)
	org.Y = math32.Mod(org.Y, incr)

	// fmt.Printf("mod org: %v   incr: %v\n", org, incr)

	return incr, org
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

func (sv *SVGView) SnapPointToGrid(rawpt mat32.Vec2) mat32.Vec2 {
	if !Prefs.SnapGrid {
		return rawpt
	}
	grinc, groff := sv.GridDots()
	var snpt mat32.Vec2
	snpt.X, _ = SnapToIncr(rawpt.X, groff.X, grinc)
	snpt.Y, _ = SnapToIncr(rawpt.Y, groff.Y, grinc)
	return snpt
}

// SnapPoint does snapping on one raw point, given that point,
// in window coordinates. returns the snapped point.
func (sv *SVGView) SnapPoint(rawpt mat32.Vec2) mat32.Vec2 {
	es := sv.EditState()
	snpt := sv.SnapPointToGrid(rawpt)
	if !Prefs.SnapGuide {
		return snpt
	}
	clDst := [2]float32{float32(math.MaxFloat32), float32(math.MaxFloat32)}
	var clPts [2][]BBoxPoints
	var clVals [2][]mat32.Vec2
	for ap := BBLeft; ap < BBoxPointsN; ap++ {
		pts := es.AlignPts[ap]
		dim := ap.Dim()
		for _, pt := range pts {
			pv := pt.Dim(dim)
			bv := rawpt.Dim(dim)
			dst := mat32.Abs(pv - bv)
			if dst < clDst[dim] {
				clDst[dim] = dst
				clPts[dim] = []BBoxPoints{ap}
				clVals[dim] = []mat32.Vec2{pt}
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
		bv := rawpt.Dim(dim)
		sval, snap := SnapToPt(bv, clVals[dim][0].Dim(dim))
		if snap {
			snpt.SetDim(dim, sval)
			mx := ints.MinInt(len(clVals[dim]), 4)
			for i := 0; i < mx; i++ {
				pt := clVals[dim][i]
				rpt := image.Rectangle{}
				rpt.Min = rawpt.ToPoint()
				rpt.Max = pt.ToPoint()
				if dim == mat32.X {
					rpt.Min.X = rpt.Max.X
				} else {
					rpt.Min.Y = rpt.Max.Y
				}
				alpts = append(alpts, rpt)
				altyps = append(altyps, clPts[dim][i])
			}
		}
	}
	sv.ShowAlignMatches(alpts, altyps)
	return snpt
}

// SnapBBox does snapping on given raw bbox according to preferences,
// aligning movement of bbox edges / centers relative to other bboxes..
// returns snapped bbox.
func (sv *SVGView) SnapBBox(rawbb mat32.Box2) mat32.Box2 {
	if !Prefs.SnapGuide {
		return rawbb
	}
	es := sv.EditState()
	snapbb := rawbb
	clDst := [2]float32{float32(math.MaxFloat32), float32(math.MaxFloat32)}
	var clPts [2][]BBoxPoints
	var clVals [2][]mat32.Vec2
	var bbval [2]mat32.Vec2
	for ap := BBLeft; ap < BBoxPointsN; ap++ {
		bbp := ap.PointBox(rawbb)
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
			clPts[dim][0].MoveDelta(&snapbb, sval-bv)
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
		}
	}
	sv.ShowAlignMatches(alpts, altyps)
	return snapbb
}

// ConstrainPoint constrains movement of point relative to starting point
// to either X, Y or diagonal.  returns constrained point, and whether the
// constraint is along the diagonal, which can then trigger reshaping the
// object to be along the diagonal as well.
// also adds constraint to AlignMatches.
func (sv *SVGView) ConstrainPoint(st, rawpt mat32.Vec2) (mat32.Vec2, bool) {
	del := rawpt.Sub(st)

	var alpts []image.Rectangle
	var altyps []BBoxPoints

	var cpts [4]mat32.Vec2

	cpts[0] = del
	cpts[0].Y = 0

	cpts[1] = del
	cpts[1].X = 0

	cpts[2] = del
	if (del.Y < 0 && del.X > 0) || (del.Y > 0 && del.X < 0) {
		cpts[2].Y = -cpts[2].X
	} else {
		cpts[2].Y = cpts[2].X
	}
	cpts[3] = del
	if (del.Y < 0 && del.X > 0) || (del.Y > 0 && del.X < 0) {
		cpts[3].X = -cpts[3].Y
	} else {
		cpts[3].X = cpts[3].Y
	}

	mind := float32(math.MaxFloat32)
	mini := 0
	for i := 0; i < 4; i++ {
		d := del.DistTo(cpts[i])
		if d < mind {
			mini = i
			mind = d
		}
	}

	cp := cpts[mini].Add(st)

	rpt := image.Rectangle{}
	rpt.Min = st.ToPoint()
	rpt.Max = cp.ToPoint()
	rpt = rpt.Canon()
	alpts = append(alpts, rpt)
	altyps = append(altyps, BBRight)

	diag := mini >= 2
	if diag {
		rpt.Max.X++ // make it horizontal
		alpts = append(alpts, rpt)
		altyps = append(altyps, BBBottom)
	}

	sv.ShowAlignMatches(alpts, altyps)
	return cp, diag
}

// DragMove is when dragging a selection for moving
func (sv *SVGView) DragMove(win *gi.Window, me *mouse.DragEvent) {
	es := sv.EditState()

	InactivateSprites(win, SpAlignMatch)

	if !es.InAction() {
		sv.ManipStart("Move", es.SelectedNamesString())
		sv.GatherAlignPoints()
	}

	svoff := mat32.NewVec2FmPoint(sv.WinBBox.Min)
	spt := mat32.NewVec2FmPoint(es.DragStartPos)
	mpt := mat32.NewVec2FmPoint(me.Where)
	if me.HasAnyModifier(key.Control) {
		mpt, _ = sv.ConstrainPoint(spt, mpt)
	}
	dv := mpt.Sub(spt)

	es.DragSelCurBBox = es.DragSelStartBBox
	es.DragSelCurBBox.Min.SetAdd(dv)
	es.DragSelCurBBox.Max.SetAdd(dv)

	es.DragSelEffBBox.Min = sv.SnapPointToGrid(es.DragSelCurBBox.Min)
	ndv := es.DragSelEffBBox.Min.Sub(es.DragSelStartBBox.Min)
	es.DragSelEffBBox.Max = es.DragSelStartBBox.Max.Add(ndv)

	es.DragSelEffBBox = sv.SnapBBox(es.DragSelEffBBox)

	pt := es.DragSelStartBBox.Min.Sub(svoff)
	tdel := es.DragSelEffBBox.Min.Sub(es.DragSelStartBBox.Min)
	for itm, ss := range es.Selected {
		itm.ReadGeom(ss.InitGeom)
		itm.ApplyDeltaXForm(tdel, mat32.Vec2{1, 1}, 0, pt)
	}
	sv.SetBBoxSpritePos(SpReshapeBBox, 0, es.DragSelEffBBox)
	sv.SetSelSpritePos()
	go sv.ManipUpdate()
	win.RenderOverlays()

}

func SquareBBox(bb mat32.Box2) mat32.Box2 {
	del := bb.Max.Sub(bb.Min)
	if del.X > del.Y {
		del.Y = del.X
	} else {
		del.X = del.Y
	}
	bb.Max = bb.Min.Add(del)
	return bb
}

// SpriteReshapeDrag processes a mouse reshape drag event on a selection sprite
func (sv *SVGView) SpriteReshapeDrag(sp Sprites, win *gi.Window, me *mouse.DragEvent) {
	es := sv.EditState()

	InactivateSprites(win, SpAlignMatch)

	if !es.InAction() {
		sv.ManipStart("Reshape", es.SelectedNamesString())
		sv.GatherAlignPoints()
	}
	stsz := es.DragSelStartBBox.Size()
	stpos := es.DragSelStartBBox.Min
	bbX, bbY := ReshapeBBoxPoints(sp)

	spt := mat32.NewVec2FmPoint(es.DragStartPos)
	mpt := mat32.NewVec2FmPoint(me.Where)
	diag := false
	if me.HasAnyModifier(key.Control) && (bbX != BBCenter && bbY != BBMiddle) {
		mpt, diag = sv.ConstrainPoint(spt, mpt)
	}
	dv := mpt.Sub(spt)
	es.DragSelCurBBox = es.DragSelStartBBox
	if diag {
		es.DragSelCurBBox = SquareBBox(es.DragSelStartBBox)
	}
	switch sp {
	case SpBBoxUpL:
		es.DragSelCurBBox.Min.SetAdd(dv)
		es.DragSelEffBBox.Min = sv.SnapPoint(es.DragSelCurBBox.Min)
	case SpBBoxUpC:
		es.DragSelCurBBox.Min.Y += dv.Y
		es.DragSelEffBBox.Min.Y = sv.SnapPoint(es.DragSelCurBBox.Min).Y
	case SpBBoxUpR:
		es.DragSelCurBBox.Min.Y += dv.Y
		es.DragSelEffBBox.Min.Y = sv.SnapPoint(es.DragSelCurBBox.Min).Y
		es.DragSelCurBBox.Max.X += dv.X
		es.DragSelEffBBox.Max.X = sv.SnapPoint(es.DragSelCurBBox.Max).X
	case SpBBoxDnL:
		es.DragSelCurBBox.Min.X += dv.X
		es.DragSelEffBBox.Min.X = sv.SnapPoint(es.DragSelCurBBox.Min).X
		es.DragSelCurBBox.Max.Y += dv.Y
		es.DragSelEffBBox.Max.Y = sv.SnapPoint(es.DragSelCurBBox.Max).Y
	case SpBBoxDnC:
		es.DragSelCurBBox.Max.Y += dv.Y
		es.DragSelEffBBox.Max.Y = sv.SnapPoint(es.DragSelCurBBox.Max).Y
	case SpBBoxDnR:
		es.DragSelCurBBox.Max.SetAdd(dv)
		es.DragSelEffBBox.Max = sv.SnapPoint(es.DragSelCurBBox.Max)
	case SpBBoxLfM:
		es.DragSelCurBBox.Min.X += dv.X
		es.DragSelEffBBox.Min.X = sv.SnapPoint(es.DragSelCurBBox.Min).X
	case SpBBoxRtM:
		es.DragSelCurBBox.Max.X += dv.X
		es.DragSelEffBBox.Max.X = sv.SnapPoint(es.DragSelCurBBox.Max).X
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

	sv.SetBBoxSpritePos(SpReshapeBBox, 0, es.DragSelEffBBox)
	sv.SetSelSpritePos()
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
	case SpBBoxUpL:
		es.DragSelCurBBox.Min.SetAdd(dv)
		dy = es.DragSelStartBBox.Min.Y - es.DragSelCurBBox.Min.Y
		dx = es.DragSelStartBBox.Max.X - es.DragSelCurBBox.Min.X
		pt.X = es.DragSelStartBBox.Max.X
	case SpBBoxUpC:
		es.DragSelCurBBox.Min.Y += dv.Y
		es.DragSelCurBBox.Max.X += dv.X
		dy = es.DragSelCurBBox.Min.Y - es.DragSelStartBBox.Min.Y
		dx = es.DragSelCurBBox.Max.X - es.DragSelStartBBox.Min.X
		pt = ctr
	case SpBBoxUpR:
		es.DragSelCurBBox.Min.Y += dv.Y
		es.DragSelCurBBox.Max.X += dv.X
		dy = es.DragSelCurBBox.Min.Y - es.DragSelStartBBox.Min.Y
		dx = es.DragSelCurBBox.Max.X - es.DragSelStartBBox.Min.X
		pt = es.DragSelStartBBox.Min
	case SpBBoxDnL:
		es.DragSelCurBBox.Min.X += dv.X
		es.DragSelCurBBox.Max.Y += dv.Y
		dy = es.DragSelStartBBox.Max.Y - es.DragSelCurBBox.Max.Y
		dx = es.DragSelStartBBox.Max.X - es.DragSelCurBBox.Min.X
		pt = es.DragSelStartBBox.Max
	case SpBBoxDnC:
		es.DragSelCurBBox.Max.SetAdd(dv)
		dy = es.DragSelCurBBox.Max.Y - es.DragSelStartBBox.Max.Y
		dx = es.DragSelCurBBox.Max.X - es.DragSelStartBBox.Min.X
		pt = ctr
	case SpBBoxDnR:
		es.DragSelCurBBox.Max.SetAdd(dv)
		dy = es.DragSelCurBBox.Max.Y - es.DragSelStartBBox.Max.Y
		dx = es.DragSelCurBBox.Max.X - es.DragSelStartBBox.Min.X
		pt.X = es.DragSelStartBBox.Min.X
		pt.Y = es.DragSelStartBBox.Max.Y
	case SpBBoxLfM:
		es.DragSelCurBBox.Min.X += dv.X
		es.DragSelCurBBox.Max.Y += dv.Y
		dy = es.DragSelStartBBox.Max.Y - es.DragSelCurBBox.Max.Y
		dx = es.DragSelStartBBox.Max.X - es.DragSelCurBBox.Min.X
		pt = ctr
	case SpBBoxRtM:
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

	sv.SetBBoxSpritePos(SpReshapeBBox, 0, es.DragSelCurBBox)
	sv.SetSelSpritePos()
	go sv.ManipUpdate()
	win.RenderOverlays()
}
