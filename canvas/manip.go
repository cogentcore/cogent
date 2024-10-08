// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package canvas

import (
	"fmt"
	"image"
	"math"

	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/svg"
)

// ManipStart is called at the start of a manipulation, saving the state prior to the action
func (sv *SVG) ManipStart(act Actions, data string) {
	es := sv.EditState()
	es.ActStart(act, data)
	help := ActionHelpMap[act]
	sv.Canvas.SetStatus(fmt.Sprintf("<b>%s</b>: %s", act, help))
	sv.UndoSave(act.String(), data)
	es.ActUnlock()
}

// ManipDone happens when a manipulation has finished: resets action, does render
func (sv *SVG) ManipDone() {
	InactivateSprites(sv, SpAlignMatch)
	es := sv.EditState()
	switch {
	case es.Action == BoxSelect:
		bbox := image.Rectangle{Min: es.DragStartPos, Max: es.DragCurPos}
		bbox = bbox.Canon().Sub(sv.Geom.ContentBBox.Min)
		InactivateSprites(sv, SpRubberBand)
		fmt.Println(bbox)
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
	sv.UpdateSelect()
	es.DragSelStart(es.DragStartPos) // capture final state as new start
	sv.UpdateView(true)
	sv.Canvas.ChangeMade()
}

// GridDots returns the current grid spacing and offsets in dots.
func (sv *SVG) GridDots() (float32, math32.Vector2) {
	svoff := math32.FromPoint(sv.Geom.ContentBBox.Min)
	grid := sv.GridEff
	if grid <= 0 {
		grid = 12
	}
	incr := grid * sv.SVG.Scale // our zoom factor

	org := math32.Vector2{}
	org = sv.Root().Paint.Transform.MulVector2AsPoint(org)

	// fmt.Printf("org: %v\n", org)

	org.SetAdd(svoff)
	// fmt.Printf("org: %v   svgoff: %v\n", org, svoff)

	org.X = math32.Mod(org.X, incr)
	org.Y = math32.Mod(org.Y, incr)

	// fmt.Printf("mod org: %v   incr: %v\n", org, incr)

	return incr, org
}

// SnapToPt snaps value to given potential snap point, in screen pixel units.
// Tolerance is determined by settings.  Returns true if snapped.
func SnapToPt(val, snap float32) (float32, bool) {
	d := math32.Abs(val - snap)
	if d <= float32(Settings.SnapTol) {
		return snap, true
	}
	return val, false
}

// SnapToIncr snaps value to given increment, first subtracting given offset.
// Tolerance is determined by settings, which is in screen pixels.
// Returns true if snapped.
func SnapToIncr(val, off, incr float32) (float32, bool) {
	nint := math32.Round((val-off)/incr)*incr + off
	dint := math32.Abs(val - nint)
	if dint <= float32(Settings.SnapTol) {
		return nint, true
	}
	return val, false
}

func (sv *SVG) SnapPointToVector(rawpt math32.Vector2) math32.Vector2 {
	if !Settings.SnapGrid {
		return rawpt
	}
	grinc, groff := sv.GridDots()
	var snpt math32.Vector2
	snpt.X, _ = SnapToIncr(rawpt.X, groff.X, grinc)
	snpt.Y, _ = SnapToIncr(rawpt.Y, groff.Y, grinc)
	return snpt
}

// SnapPoint does snapping on one raw point, given that point,
// in window coordinates. returns the snapped point.
func (sv *SVG) SnapPoint(rawpt math32.Vector2) math32.Vector2 {
	es := sv.EditState()
	snpt := sv.SnapPointToVector(rawpt)
	if !Settings.SnapGuide {
		return snpt
	}
	clDst := [2]float32{float32(math.MaxFloat32), float32(math.MaxFloat32)}
	var clPts [2][]BBoxPoints
	var clVals [2][]math32.Vector2
	for ap := BBLeft; ap < BBoxPointsN; ap++ {
		pts := es.AlignPts[ap]
		dim := ap.Dim()
		for _, pt := range pts {
			pv := pt.Dim(dim)
			bv := rawpt.Dim(dim)
			dst := math32.Abs(pv - bv)
			if dst < clDst[dim] {
				clDst[dim] = dst
				clPts[dim] = []BBoxPoints{ap}
				clVals[dim] = []math32.Vector2{pt}
			} else if math32.Abs(dst-clDst[dim]) < 1.0e-4 {
				clPts[dim] = append(clPts[dim], ap)
				clVals[dim] = append(clVals[dim], pt)
			}
		}
	}
	var alpts []image.Rectangle
	var altyps []BBoxPoints
	for dim := math32.X; dim <= math32.Y; dim++ {
		if len(clVals[dim]) == 0 {
			continue
		}
		bv := rawpt.Dim(dim)
		sval, snap := SnapToPt(bv, clVals[dim][0].Dim(dim))
		if snap {
			snpt.SetDim(dim, sval)
			mx := min(len(clVals[dim]), 4)
			for i := 0; i < mx; i++ {
				pt := clVals[dim][i]
				rpt := image.Rectangle{}
				rpt.Min = rawpt.ToPoint()
				rpt.Max = pt.ToPoint()
				if dim == math32.X {
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

// SnapBBox does snapping on given raw bbox according to settings,
// aligning movement of bbox edges / centers relative to other bboxes..
// returns snapped bbox.
func (sv *SVG) SnapBBox(rawbb math32.Box2) math32.Box2 {
	if !Settings.SnapGuide {
		return rawbb
	}
	es := sv.EditState()
	snapbb := rawbb
	clDst := [2]float32{float32(math.MaxFloat32), float32(math.MaxFloat32)}
	var clPts [2][]BBoxPoints
	var clVals [2][]math32.Vector2
	var bbval [2]math32.Vector2
	for ap := BBLeft; ap < BBoxPointsN; ap++ {
		bbp := ap.PointBox(rawbb)
		pts := es.AlignPts[ap]
		dim := ap.Dim()
		for _, pt := range pts {
			pv := pt.Dim(dim)
			bv := bbp.Dim(dim)
			dst := math32.Abs(pv - bv)
			if dst < clDst[dim] {
				clDst[dim] = dst
				clPts[dim] = []BBoxPoints{ap}
				clVals[dim] = []math32.Vector2{pt}
				bbval[dim] = bbp
			} else if math32.Abs(dst-clDst[dim]) < 1.0e-4 {
				clPts[dim] = append(clPts[dim], ap)
				clVals[dim] = append(clVals[dim], pt)
			}
		}
	}
	var alpts []image.Rectangle
	var altyps []BBoxPoints
	for dim := math32.X; dim <= math32.Y; dim++ {
		if len(clVals[dim]) == 0 {
			continue
		}
		bv := bbval[dim].Dim(dim)
		sval, snap := SnapToPt(bv, clVals[dim][0].Dim(dim))
		if snap {
			clPts[dim][0].MoveDelta(&snapbb, sval-bv)
			mx := min(len(clVals[dim]), 4)
			for i := 0; i < mx; i++ {
				pt := clVals[dim][i]
				rpt := image.Rectangle{}
				rpt.Min = bbval[dim].ToPoint()
				rpt.Max = pt.ToPoint()
				if dim == math32.X {
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
func (sv *SVG) ConstrainPoint(st, rawpt math32.Vector2) (math32.Vector2, bool) {
	del := rawpt.Sub(st)

	var alpts []image.Rectangle
	var altyps []BBoxPoints

	var cpts [4]math32.Vector2

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
		d := del.DistanceTo(cpts[i])
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
func (sv *SVG) DragMove(e events.Event) {
	es := sv.EditState()

	InactivateSprites(sv, SpAlignMatch)

	if !es.InAction() {
		sv.ManipStart(Move, es.SelectedNamesString())
		sv.GatherAlignPoints()
	}

	svoff := math32.FromPoint(sv.Geom.ContentBBox.Min)
	spt := math32.FromPoint(es.DragStartPos)
	mpt := math32.FromPoint(e.Pos())
	if e.HasAnyModifier(key.Control) {
		mpt, _ = sv.ConstrainPoint(spt, mpt)
	}
	dv := mpt.Sub(spt)

	es.DragSelectCurrentBBox = es.DragSelectStartBBox
	es.DragSelectCurrentBBox.Min.SetAdd(dv)
	es.DragSelectCurrentBBox.Max.SetAdd(dv)

	es.DragSelectEffectiveBBox.Min = sv.SnapPointToVector(es.DragSelectCurrentBBox.Min)
	ndv := es.DragSelectEffectiveBBox.Min.Sub(es.DragSelectStartBBox.Min)
	es.DragSelectEffectiveBBox.Max = es.DragSelectStartBBox.Max.Add(ndv)

	es.DragSelectEffectiveBBox = sv.SnapBBox(es.DragSelectEffectiveBBox)

	pt := es.DragSelectStartBBox.Min.Sub(svoff)
	tdel := es.DragSelectEffectiveBBox.Min.Sub(es.DragSelectStartBBox.Min)
	for itm, ss := range es.Selected {
		itm.ReadGeom(sv.SVG, ss.InitGeom)
		itm.ApplyDeltaTransform(sv.SVG, tdel, math32.Vec2(1, 1), 0, pt)
	}
	sv.SetBBoxSpritePos(SpReshapeBBox, 0, es.DragSelectEffectiveBBox)
	sv.SetSelSpritePos()
	go sv.RenderSVG()
}

func SquareBBox(bb math32.Box2) math32.Box2 {
	del := bb.Size()
	if del.X > del.Y {
		del.Y = del.X
	} else {
		del.X = del.Y
	}
	bb.Max = bb.Min.Add(del)
	return bb
}

func ProportionalBBox(bb, orig math32.Box2) math32.Box2 {
	prop := orig.Size()
	if prop.X == 0 || prop.Y == 0 {
		return bb
	}
	del := bb.Size()
	if del.X > del.Y {
		del.Y = del.X * (prop.Y / prop.X)
	} else {
		del.X = del.Y * (prop.X / prop.Y)
	}
	bb.Max = bb.Min.Add(del)
	return bb
}

// SpriteReshapeDrag processes a mouse reshape drag event on a selection sprite
func (sv *SVG) SpriteReshapeDrag(sp Sprites, e events.Event) {
	es := sv.EditState()

	InactivateSprites(sv, SpAlignMatch)

	if !es.InAction() {
		sv.ManipStart(Reshape, es.SelectedNamesString())
		sv.GatherAlignPoints()
	}
	stsz := es.DragSelectStartBBox.Size()
	stpos := es.DragSelectStartBBox.Min
	bbX, bbY := ReshapeBBoxPoints(sp)

	spt := math32.FromPoint(es.DragStartPos)
	mpt := math32.FromPoint(e.Pos())
	diag := false
	if e.HasAnyModifier(key.Control) && (bbX != BBCenter && bbY != BBMiddle) {
		mpt, diag = sv.ConstrainPoint(spt, mpt)
	}
	dv := mpt.Sub(spt)
	es.DragSelectCurrentBBox = es.DragSelectStartBBox
	switch sp {
	case SpBBoxUpL:
		es.DragSelectCurrentBBox.Min.SetAdd(dv)
		es.DragSelectEffectiveBBox.Min = sv.SnapPoint(es.DragSelectCurrentBBox.Min)
	case SpBBoxUpC:
		es.DragSelectCurrentBBox.Min.Y += dv.Y
		es.DragSelectEffectiveBBox.Min.Y = sv.SnapPoint(es.DragSelectCurrentBBox.Min).Y
	case SpBBoxUpR:
		es.DragSelectCurrentBBox.Min.Y += dv.Y
		es.DragSelectEffectiveBBox.Min.Y = sv.SnapPoint(es.DragSelectCurrentBBox.Min).Y
		es.DragSelectCurrentBBox.Max.X += dv.X
		es.DragSelectEffectiveBBox.Max.X = sv.SnapPoint(es.DragSelectCurrentBBox.Max).X
	case SpBBoxDnL:
		es.DragSelectCurrentBBox.Min.X += dv.X
		es.DragSelectEffectiveBBox.Min.X = sv.SnapPoint(es.DragSelectCurrentBBox.Min).X
		es.DragSelectCurrentBBox.Max.Y += dv.Y
		es.DragSelectEffectiveBBox.Max.Y = sv.SnapPoint(es.DragSelectCurrentBBox.Max).Y
	case SpBBoxDnC:
		es.DragSelectCurrentBBox.Max.Y += dv.Y
		es.DragSelectEffectiveBBox.Max.Y = sv.SnapPoint(es.DragSelectCurrentBBox.Max).Y
	case SpBBoxDnR:
		es.DragSelectCurrentBBox.Max.SetAdd(dv)
		es.DragSelectEffectiveBBox.Max = sv.SnapPoint(es.DragSelectCurrentBBox.Max)
	case SpBBoxLfM:
		es.DragSelectCurrentBBox.Min.X += dv.X
		es.DragSelectEffectiveBBox.Min.X = sv.SnapPoint(es.DragSelectCurrentBBox.Min).X
	case SpBBoxRtM:
		es.DragSelectCurrentBBox.Max.X += dv.X
		es.DragSelectEffectiveBBox.Max.X = sv.SnapPoint(es.DragSelectCurrentBBox.Max).X
	}

	if diag {
		sq := false
		if len(es.Selected) == 1 {
			so := es.SelectedList(false)[0]
			switch so.(type) {
			case *svg.Rect:
				sq = true
			case *svg.Ellipse:
				sq = true
			case *svg.Circle:
				sq = true
			}
		}
		if sq {
			es.DragSelectEffectiveBBox = SquareBBox(es.DragSelectEffectiveBBox)
		} else {
			es.DragSelectEffectiveBBox = ProportionalBBox(es.DragSelectEffectiveBBox, es.DragSelectStartBBox)
		}
	}

	npos := es.DragSelectEffectiveBBox.Min
	nsz := es.DragSelectEffectiveBBox.Size()
	pt := es.DragSelectStartBBox.Min
	// fmt.Println("npos:", npos, "stpos:", stpos, "pt:", pt)
	del := npos.Sub(stpos)
	sc := nsz.Div(stsz)
	for itm, ss := range es.Selected {
		itm.ReadGeom(sv.SVG, ss.InitGeom)
		itm.ApplyDeltaTransform(sv.SVG, del, sc, 0, pt)
		// if strings.HasPrefix(es.Action, "New") {
		// 	svg.UpdateNodeGradientPoints(itm, "fill")
		// 	svg.UpdateNodeGradientPoints(itm, "stroke")
		// }
	}

	sv.SetBBoxSpritePos(SpReshapeBBox, 0, es.DragSelectEffectiveBBox)
	sv.SetSelSpritePos()
	go sv.RenderSVG()
}

// SpriteRotateDrag processes a mouse rotate drag event on a selection sprite
func (sv *SVG) SpriteRotateDrag(sp Sprites, delta image.Point) {
	fmt.Println("rotate", delta)
	es := sv.EditState()
	if !es.InAction() {
		sv.ManipStart(Rotate, es.SelectedNamesString())
	}
	dv := math32.FromPoint(delta)
	pt := es.DragSelectStartBBox.Min
	ctr := es.DragSelectStartBBox.Min.Add(es.DragSelectStartBBox.Max).MulScalar(.5)
	var dx, dy float32
	switch sp {
	case SpBBoxUpL:
		es.DragSelectCurrentBBox.Min.SetAdd(dv)
		dy = es.DragSelectStartBBox.Min.Y - es.DragSelectCurrentBBox.Min.Y
		dx = es.DragSelectStartBBox.Max.X - es.DragSelectCurrentBBox.Min.X
		pt.X = es.DragSelectStartBBox.Max.X
	case SpBBoxUpC:
		es.DragSelectCurrentBBox.Min.Y += dv.Y
		es.DragSelectCurrentBBox.Max.X += dv.X
		dy = es.DragSelectCurrentBBox.Min.Y - es.DragSelectStartBBox.Min.Y
		dx = es.DragSelectCurrentBBox.Max.X - es.DragSelectStartBBox.Min.X
		pt = ctr
	case SpBBoxUpR:
		es.DragSelectCurrentBBox.Min.Y += dv.Y
		es.DragSelectCurrentBBox.Max.X += dv.X
		dy = es.DragSelectCurrentBBox.Min.Y - es.DragSelectStartBBox.Min.Y
		dx = es.DragSelectCurrentBBox.Max.X - es.DragSelectStartBBox.Min.X
		pt = es.DragSelectStartBBox.Min
	case SpBBoxDnL:
		es.DragSelectCurrentBBox.Min.X += dv.X
		es.DragSelectCurrentBBox.Max.Y += dv.Y
		dy = es.DragSelectStartBBox.Max.Y - es.DragSelectCurrentBBox.Max.Y
		dx = es.DragSelectStartBBox.Max.X - es.DragSelectCurrentBBox.Min.X
		pt = es.DragSelectStartBBox.Max
	case SpBBoxDnC:
		es.DragSelectCurrentBBox.Max.SetAdd(dv)
		dy = es.DragSelectCurrentBBox.Max.Y - es.DragSelectStartBBox.Max.Y
		dx = es.DragSelectCurrentBBox.Max.X - es.DragSelectStartBBox.Min.X
		pt = ctr
	case SpBBoxDnR:
		es.DragSelectCurrentBBox.Max.SetAdd(dv)
		dy = es.DragSelectCurrentBBox.Max.Y - es.DragSelectStartBBox.Max.Y
		dx = es.DragSelectCurrentBBox.Max.X - es.DragSelectStartBBox.Min.X
		pt.X = es.DragSelectStartBBox.Min.X
		pt.Y = es.DragSelectStartBBox.Max.Y
	case SpBBoxLfM:
		es.DragSelectCurrentBBox.Min.X += dv.X
		es.DragSelectCurrentBBox.Max.Y += dv.Y
		dy = es.DragSelectStartBBox.Max.Y - es.DragSelectCurrentBBox.Max.Y
		dx = es.DragSelectStartBBox.Max.X - es.DragSelectCurrentBBox.Min.X
		pt = ctr
	case SpBBoxRtM:
		es.DragSelectCurrentBBox.Max.SetAdd(dv)
		dy = es.DragSelectCurrentBBox.Max.Y - es.DragSelectStartBBox.Max.Y
		dx = es.DragSelectCurrentBBox.Max.X - es.DragSelectStartBBox.Min.X
		pt = ctr
	}
	ang := math32.Atan2(dy, dx)
	ang, _ = SnapToIncr(math32.RadToDeg(ang), 0, 15)
	ang = math32.DegToRad(ang)
	del := math32.Vector2{}
	sc := math32.Vec2(1, 1)
	for itm, ss := range es.Selected {
		itm.ReadGeom(sv.SVG, ss.InitGeom)
		itm.ApplyDeltaTransform(sv.SVG, del, sc, ang, pt)
		// if strings.HasPrefix(es.Action, "New") {
		// 	sv.UpdateNodeGradientPoints(itm, "fill")
		// 	sv.UpdateNodeGradientPoints(itm, "stroke")
		// }
	}

	sv.SetBBoxSpritePos(SpReshapeBBox, 0, es.DragSelectCurrentBBox)
	sv.SetSelSpritePos()
	go sv.RenderSVG()
}
