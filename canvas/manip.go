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

// ManipStartInDrag is called at the start of a dragging action to ensure that
// the action has started if it hasn't already, and to reset the align sprites.
// sprites must already be locked.
func (sv *SVG) ManipStartInDrag(act Actions, data string) {
	es := sv.EditState()
	sprites := sv.SpritesNolock()
	InactivateSprites(sprites, SpAlignMatch)
	if !es.InAction() {
		sv.ManipStart(act, data)
		sv.GatherAlignPoints()
	}
}

// ManipDone happens when a manipulation has finished: resets action, does render
func (sv *SVG) ManipDone() {
	sprites := sv.SpritesLock()
	InactivateSprites(sprites, SpAlignMatch)
	es := sv.EditState()
	switch {
	case es.Action == BoxSelect:
		bbox := math32.Box2{Min: math32.FromPoint(es.DragStartPos), Max: math32.FromPoint(es.DragPos)}
		bbox = bbox.Canon()
		InactivateSprites(sprites, SpRubberBand)
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
	sprites.Unlock()
	es.ActDone()
	sv.UpdateSelect()
	es.DragSelStart(es.DragStartPos) // capture final state as new start
	sv.UpdateView()
	sv.Canvas.ChangeMade()
}

// SnapToPoint snaps value to given potential snap point, in screen pixel units.
// Tolerance is determined by settings.  Returns true if snapped.
func SnapToPoint(val, snap float32) (float32, bool) {
	d := math32.Abs(val - snap)
	if d <= float32(Settings.SnapZone) {
		return snap, true
	}
	return val, false
}

// SnapToIncr snaps value to given increment, first subtracting given offset.
// Tolerance is determined by settings, which is in screen pixels.
// Returns true if snapped.
func SnapToIncr(val, off, incr float32) float32 {
	nint := math32.Round((val-off)/incr)*incr + off
	dint := math32.Abs(val - nint)
	if dint <= float32(Settings.SnapZone) {
		return nint
	}
	return val
}

func (sv *SVG) SnapGridPoint(rawpt math32.Vector2) math32.Vector2 {
	if !Settings.SnapGrid {
		return rawpt
	}
	return math32.Vec2(SnapToIncr(rawpt.X, sv.GridOffset.X, sv.GridPixels.X), SnapToIncr(rawpt.Y, sv.GridOffset.Y, sv.GridPixels.Y))
}

// SnapPoint does grid and align snapping on one raw point, given that point,
// in window coordinates. returns the snapped point.
func (sv *SVG) SnapPoint(rawpt math32.Vector2) math32.Vector2 {
	es := sv.EditState()
	snpt := sv.SnapGridPoint(rawpt)
	if !Settings.SnapAlign {
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
		sval, snap := SnapToPoint(bv, clVals[dim][0].Dim(dim))
		if snap {
			continue
		}
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
	sv.ShowAlignMatches(alpts, altyps)
	return snpt
}

// SnapBBox does snapping on given raw bbox according to settings,
// aligning movement of bbox edges / centers relative to other bboxes..
// returns snapped bbox.
func (sv *SVG) SnapBBox(rawbb math32.Box2) math32.Box2 {
	if !Settings.SnapAlign {
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
		sval, snap := SnapToPoint(bv, clVals[dim][0].Dim(dim))
		if !snap {
			continue
		}
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

// ShowAlignMatches draws the align matches as given
// between BBox Min - Max. typs are corresponding bounding box sources.
// sprites must already be locked.
func (sv *SVG) ShowAlignMatches(pts []image.Rectangle, typs []BBoxPoints) {
	sv.SpritesNolock()
	sz := min(len(pts), 8)
	for i := 0; i < sz; i++ {
		pt := pts[i].Canon()
		lsz := pt.Max.Sub(pt.Min)
		sp := sv.Sprite(SpAlignMatch, Sprites(typs[i]), i, lsz, nil)
		sp.Properties["size"] = lsz
		sv.SetSpritePos(sp, pt.Min.X, pt.Min.Y)
	}
}

func (sv *SVG) DragDelta(e events.Event, node bool) (spt, mpt, dv math32.Vector2) {
	es := sv.EditState()
	spt = math32.FromPoint(es.DragStartPos)
	mpt = math32.FromPoint(e.Pos())

	if es.ConstrainPoint && e.HasAnyModifier(key.Control) {
		mpt, _ = sv.ConstrainPoint(spt, mpt)
	}
	if node && Settings.SnapNodes {
		mpt = sv.SnapPoint(mpt)
	}
	es.DragPos = mpt.ToPointRound()
	dv = mpt.Sub(spt)
	return
}

// DragMove is when dragging a selection for moving
func (sv *SVG) DragMove(e events.Event) {
	es := sv.EditState()
	sprites := sv.SpritesLock()
	sv.ManipStartInDrag(Move, es.SelectedNamesString())
	es.ConstrainPoint = true
	_, _, dv := sv.DragDelta(e, false)

	es.DragBBox = es.DragStartBBox
	es.DragBBox.Min.SetAdd(dv)
	es.DragBBox.Max.SetAdd(dv)

	es.DragSnapBBox.Min = sv.SnapGridPoint(es.DragBBox.Min)
	ndv := es.DragSnapBBox.Min.Sub(es.DragStartBBox.Min)
	es.DragSnapBBox.Max = es.DragStartBBox.Max.Add(ndv)

	es.DragSnapBBox = sv.SnapBBox(es.DragSnapBBox)

	pt := es.DragStartBBox.Min
	tdel := es.DragSnapBBox.Min.Sub(es.DragStartBBox.Min)
	for itm, ss := range es.Selected {
		svg.BitCopyFrom(itm, ss.InitState)
		xf := itm.AsNodeBase().DeltaTransform(tdel, math32.Vec2(1, 1), 0, pt)
		itm.ApplyTransform(sv.SVG, xf)
	}
	sv.SetBBoxSpritePos(SpReshapeBBox, 0, es.DragSnapBBox)
	sprites.Unlock()
	sv.UpdateView()
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
	sprites := sv.SpritesLock()
	sv.ManipStartInDrag(Reshape, es.SelectedNamesString())

	stsz := es.DragStartBBox.Size()
	stpos := es.DragStartBBox.Min
	bbX, bbY := ReshapeBBoxPoints(sp)

	es.ConstrainPoint = (bbX != BBCenter && bbY != BBMiddle)
	_, _, dv := sv.DragDelta(e, false)

	diag := false
	es.DragBBox = es.DragStartBBox
	switch sp {
	case SpUpL:
		es.DragBBox.Min.SetAdd(dv)
		es.DragSnapBBox.Min = sv.SnapPoint(es.DragBBox.Min)
	case SpUpC:
		es.DragBBox.Min.Y += dv.Y
		es.DragSnapBBox.Min.Y = sv.SnapPoint(es.DragBBox.Min).Y
	case SpUpR:
		es.DragBBox.Min.Y += dv.Y
		es.DragSnapBBox.Min.Y = sv.SnapPoint(es.DragBBox.Min).Y
		es.DragBBox.Max.X += dv.X
		es.DragSnapBBox.Max.X = sv.SnapPoint(es.DragBBox.Max).X
	case SpDnL:
		es.DragBBox.Min.X += dv.X
		es.DragSnapBBox.Min.X = sv.SnapPoint(es.DragBBox.Min).X
		es.DragBBox.Max.Y += dv.Y
		es.DragSnapBBox.Max.Y = sv.SnapPoint(es.DragBBox.Max).Y
	case SpDnC:
		es.DragBBox.Max.Y += dv.Y
		es.DragSnapBBox.Max.Y = sv.SnapPoint(es.DragBBox.Max).Y
	case SpDnR:
		es.DragBBox.Max.SetAdd(dv)
		es.DragSnapBBox.Max = sv.SnapPoint(es.DragBBox.Max)
	case SpLfM:
		es.DragBBox.Min.X += dv.X
		es.DragSnapBBox.Min.X = sv.SnapPoint(es.DragBBox.Min).X
	case SpRtM:
		es.DragBBox.Max.X += dv.X
		es.DragSnapBBox.Max.X = sv.SnapPoint(es.DragBBox.Max).X
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
			es.DragSnapBBox = SquareBBox(es.DragSnapBBox)
		} else {
			es.DragSnapBBox = ProportionalBBox(es.DragSnapBBox, es.DragStartBBox)
		}
	}

	npos := es.DragSnapBBox.Min
	nsz := es.DragSnapBBox.Size()
	pt := es.DragSnapBBox.Min
	// fmt.Println("npos:", npos, "stpos:", stpos, "pt:", pt)
	del := npos.Sub(stpos)
	sc := nsz.Div(stsz)
	for itm, ss := range es.Selected {
		svg.BitCopyFrom(itm, ss.InitState)
		xf := itm.AsNodeBase().DeltaTransform(del, sc, 0, pt)
		itm.ApplyTransform(sv.SVG, xf)
	}

	sprites.Unlock()
	sv.UpdateView()
}

// SpriteRotateDrag processes a mouse rotate drag event on a selection sprite
func (sv *SVG) SpriteRotateDrag(sp Sprites, e events.Event) {
	es := sv.EditState()
	if !es.InAction() {
		sv.ManipStart(Rotate, es.SelectedNamesString())
	}
	dv := math32.FromPoint(e.PrevDelta()) // not from start but just current delta: adding to control points
	pt := es.DragStartBBox.Min
	ctr := es.DragStartBBox.Min.Add(es.DragStartBBox.Max).MulScalar(.5)
	var dx, dy float32
	switch sp {
	case SpUpL:
		es.DragBBox.Min.SetAdd(dv)
		dy = es.DragStartBBox.Min.Y - es.DragBBox.Min.Y
		dx = es.DragStartBBox.Max.X - es.DragBBox.Min.X
		pt.X = es.DragStartBBox.Max.X
	case SpUpC:
		es.DragBBox.Min.Y += dv.Y
		es.DragBBox.Max.X += dv.X
		dy = es.DragBBox.Min.Y - es.DragStartBBox.Min.Y
		dx = es.DragBBox.Max.X - es.DragStartBBox.Min.X
		pt = ctr
	case SpUpR:
		es.DragBBox.Min.Y += dv.Y
		es.DragBBox.Max.X += dv.X
		dy = es.DragBBox.Min.Y - es.DragStartBBox.Min.Y
		dx = es.DragBBox.Max.X - es.DragStartBBox.Min.X
		pt = es.DragStartBBox.Min
	case SpDnL:
		es.DragBBox.Min.X += dv.X
		es.DragBBox.Max.Y += dv.Y
		dy = es.DragStartBBox.Max.Y - es.DragBBox.Max.Y
		dx = es.DragStartBBox.Max.X - es.DragBBox.Min.X
		pt = es.DragStartBBox.Max
	case SpDnC:
		es.DragBBox.Max.SetAdd(dv)
		dy = es.DragBBox.Max.Y - es.DragStartBBox.Max.Y
		dx = es.DragBBox.Max.X - es.DragStartBBox.Min.X
		pt = ctr
	case SpDnR:
		es.DragBBox.Max.SetAdd(dv)
		dy = es.DragBBox.Max.Y - es.DragStartBBox.Max.Y
		dx = es.DragBBox.Max.X - es.DragStartBBox.Min.X
		pt.X = es.DragStartBBox.Min.X
		pt.Y = es.DragStartBBox.Max.Y
	case SpLfM:
		es.DragBBox.Min.X += dv.X
		es.DragBBox.Max.Y += dv.Y
		dy = es.DragStartBBox.Max.Y - es.DragBBox.Max.Y
		dx = es.DragStartBBox.Max.X - es.DragBBox.Min.X
		pt = ctr
	case SpRtM:
		es.DragBBox.Max.SetAdd(dv)
		dy = es.DragBBox.Max.Y - es.DragStartBBox.Max.Y
		dx = es.DragBBox.Max.X - es.DragStartBBox.Min.X
		pt = ctr
	}
	ang := math32.Atan2(dy, dx)
	if !e.HasAnyModifier(key.Shift) {
		ang = SnapToIncr(math32.RadToDeg(ang), 0, 15)
	}
	ang = math32.DegToRad(ang)
	del := math32.Vector2{}
	sc := math32.Vec2(1, 1)
	for itm, ss := range es.Selected {
		svg.BitCopyFrom(itm, ss.InitState)
		xf := itm.AsNodeBase().DeltaTransform(del, sc, ang, pt)
		itm.ApplyTransform(sv.SVG, xf)
	}
	sv.UpdateView()
}
