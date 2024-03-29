// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vector

import (
	"image"

	"cogentcore.org/core/events"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/giv"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/mat32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/svg"
)

// AlignView provides a range of alignment actions on selected objects.
type AlignView struct {
	gi.Layout

	AlignAnchorView giv.EnumValue

	// the parent vectorview
	VectorView *VectorView `copier:"-" json:"-" xml:"-" view:"-"`
}

/////////////////////////////////////////////////////////////////////////
//  Actions

func (vv *VectorView) Align(aa AlignAnchors, al Aligns) {
	astr := al.String()
	switch al {
	case AlignRightAnchor:
		vv.AlignMaxAnchor(aa, mat32.X, astr)
	case AlignLeft:
		vv.AlignMin(aa, mat32.X, astr)
	case AlignCenter:
		vv.AlignCenter(aa, mat32.X, astr)
	case AlignRight:
		vv.AlignMax(aa, mat32.X, astr)
	case AlignLeftAnchor:
		vv.AlignMinAnchor(aa, mat32.X, astr)
	case AlignBaselineHoriz:
		vv.AlignMin(aa, mat32.X, astr) // todo: should be baseline, not min

	case AlignBottomAnchor:
		vv.AlignMaxAnchor(aa, mat32.Y, astr)
	case AlignTop:
		vv.AlignMin(aa, mat32.Y, astr)
	case AlignMiddle:
		vv.AlignCenter(aa, mat32.Y, astr)
	case AlignBottom:
		vv.AlignMax(aa, mat32.Y, astr)
	case AlignTopAnchor:
		vv.AlignMinAnchor(aa, mat32.Y, astr)
	case AlignBaselineVert:
		vv.AlignMin(aa, mat32.Y, astr) // todo: should be baseline, not min
	}
}

// AlignAnchorBBox returns the bounding box for given type of align anchor
// and the anchor node if non-nil
func (vv *VectorView) AlignAnchorBBox(aa AlignAnchors) (image.Rectangle, svg.Node) {
	es := &vv.EditState
	sv := vv.SVG()
	svoff := sv.Root().BBox.Min
	var an svg.Node
	var bb image.Rectangle
	switch aa {
	case AlignFirst:
		sl := es.SelectedList(false)
		an = sl[0]
		bb = an.AsNodeBase().BBox
	case AlignLast:
		sl := es.SelectedList(true) // descending
		an = sl[0]
		bb = an.AsNodeBase().BBox
	case AlignSelectBox:
		bb = image.Rectangle{Min: es.DragSelCurBBox.Min.ToPointFloor(), Max: es.DragSelCurBBox.Max.ToPointCeil()}
	}
	bb = bb.Sub(svoff)
	return bb, an
}

// AlignMin aligns to min coordinate (Left, Top) in bbox
func (vv *VectorView) AlignMin(aa AlignAnchors, dim mat32.Dims, act string) {
	es := &vv.EditState
	if !es.HasSelected() {
		return
	}
	sv := vv.SVG()
	svoff := sv.Root().BBox.Min
	sv.UndoSave(act, es.SelectedNamesString())
	abb, an := vv.AlignAnchorBBox(aa)
	sc := mat32.V2(1, 1)
	odim := mat32.OtherDim(dim)
	for sn := range es.Selected {
		if sn == an {
			continue
		}
		sng := sn.AsNodeBase()
		bb := sng.BBox.Sub(svoff)
		del := mat32.V2FromPoint(abb.Min.Sub(bb.Min))
		del.SetDim(odim, 0)
		sn.ApplyDeltaTransform(vv.SSVG(), del, sc, 0, mat32.V2FromPoint(bb.Min))
	}
	sv.UpdateView(true)
	vv.ChangeMade()
}

func (vv *VectorView) AlignMinAnchor(aa AlignAnchors, dim mat32.Dims, act string) {
	es := &vv.EditState
	if !es.HasSelected() {
		return
	}
	sv := vv.SVG()
	svoff := sv.Root().BBox.Min
	sv.UndoSave(act, es.SelectedNamesString())
	abb, an := vv.AlignAnchorBBox(aa)
	sc := mat32.V2(1, 1)
	odim := mat32.OtherDim(dim)
	for sn := range es.Selected {
		if sn == an {
			continue
		}
		sng := sn.AsNodeBase()
		bb := sng.BBox.Sub(svoff)
		del := mat32.V2FromPoint(abb.Max.Sub(bb.Min))
		del.SetDim(odim, 0)
		sn.ApplyDeltaTransform(vv.SSVG(), del, sc, 0, mat32.V2FromPoint(bb.Min))
	}
	sv.UpdateView(true)
	vv.ChangeMade()
}

func (vv *VectorView) AlignMax(aa AlignAnchors, dim mat32.Dims, act string) {
	es := &vv.EditState
	if !es.HasSelected() {
		return
	}
	sv := vv.SVG()
	svoff := sv.Root().BBox.Min
	sv.UndoSave(act, es.SelectedNamesString())
	abb, an := vv.AlignAnchorBBox(aa)
	sc := mat32.V2(1, 1)
	odim := mat32.OtherDim(dim)
	for sn := range es.Selected {
		if sn == an {
			continue
		}
		sng := sn.AsNodeBase()
		bb := sng.BBox.Sub(svoff)
		del := mat32.V2FromPoint(abb.Max.Sub(bb.Max))
		del.SetDim(odim, 0)
		sn.ApplyDeltaTransform(vv.SSVG(), del, sc, 0, mat32.V2FromPoint(bb.Min))
	}
	sv.UpdateView(true)
	vv.ChangeMade()
}

func (vv *VectorView) AlignMaxAnchor(aa AlignAnchors, dim mat32.Dims, act string) {
	es := &vv.EditState
	if !es.HasSelected() {
		return
	}
	sv := vv.SVG()
	svoff := sv.Root().BBox.Min
	sv.UndoSave(act, es.SelectedNamesString())
	abb, an := vv.AlignAnchorBBox(aa)
	sc := mat32.V2(1, 1)
	odim := mat32.OtherDim(dim)
	for sn := range es.Selected {
		if sn == an {
			continue
		}
		sng := sn.AsNodeBase()
		bb := sng.BBox.Sub(svoff)
		del := mat32.V2FromPoint(abb.Min.Sub(bb.Max))
		del.SetDim(odim, 0)
		sn.ApplyDeltaTransform(vv.SSVG(), del, sc, 0, mat32.V2FromPoint(bb.Min))
	}
	sv.UpdateView(true)
	vv.ChangeMade()
}

func (vv *VectorView) AlignCenter(aa AlignAnchors, dim mat32.Dims, act string) {
	es := &vv.EditState
	if !es.HasSelected() {
		return
	}
	sv := vv.SVG()
	svoff := sv.Root().BBox.Min
	sv.UndoSave(act, es.SelectedNamesString())
	abb, an := vv.AlignAnchorBBox(aa)
	ctr := mat32.V2FromPoint(abb.Min.Add(abb.Max)).MulScalar(0.5)
	sc := mat32.V2(1, 1)
	odim := mat32.OtherDim(dim)
	for sn := range es.Selected {
		if sn == an {
			continue
		}
		sng := sn.AsNodeBase()
		bb := sng.BBox.Sub(svoff)
		nctr := mat32.V2FromPoint(bb.Min.Add(bb.Max)).MulScalar(0.5)
		del := ctr.Sub(nctr)
		del.SetDim(odim, 0)
		sn.ApplyDeltaTransform(vv.SSVG(), del, sc, 0, mat32.V2FromPoint(bb.Min))
	}
	sv.UpdateView(true)
	vv.ChangeMade()
}

// GatherAlignPoints gets all the potential points of alignment for objects not
// in selection group
func (sv *SVGView) GatherAlignPoints() {
	es := sv.EditState()
	if !es.HasSelected() {
		return
	}

	for ap := BBLeft; ap < BBoxPointsN; ap++ {
		es.AlignPts[ap] = make([]mat32.Vec2, 0)
	}

	svg.SVGWalkPreNoDefs(sv.Root(), func(kni svg.Node, knb *svg.NodeBase) bool {
		if kni.This() == sv.Root().This() {
			return ki.Continue
		}
		if NodeIsLayer(kni) {
			return ki.Continue
		}
		if _, issel := es.Selected[kni]; issel {
			return ki.Break // go no further into kids
		}
		for ap := BBLeft; ap < BBoxPointsN; ap++ {
			es.AlignPts[ap] = append(es.AlignPts[ap], ap.PointRect(knb.BBox))
		}
		return ki.Continue
	})
}

///////////////////////////////////////////////////////////////
//  AlignView

func (av *AlignView) Config() {
	if av.HasChildren() {
		return
	}
	av.Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})

	all := gi.NewLayout(av)
	gi.NewLabel(all).SetText("<b>Align:  </b>")
	gi.NewChooser(all).SetEnum(AlignAnchorsN)

	agrid := gi.NewLayout(av).Style(func(s *styles.Style) {
		s.Display = styles.Grid
		s.Columns = 6
	})

	for _, al := range AlignsValues() {
		al := al
		gi.NewButton(agrid, al.String()).SetIcon(icons.Icon(al.String())).
			SetTooltip(al.Desc()).SetType(gi.ButtonTonal).
			OnClick(func(e events.Event) {
				av.VectorView.Align(av.AlignAnchor(), al)
			})
	}
}

// AlignAnchor returns the align anchor currently selected
func (av *AlignView) AlignAnchor() AlignAnchors {
	return av.AlignAnchorView.Value.Interface().(AlignAnchors)
}

// AlignAnchors are ways of anchoring alignment
type AlignAnchors int32 //enums:enum

const (
	AlignFirst AlignAnchors = iota
	AlignLast
	AlignDrawing
	AlignSelectBox
)

// Aligns are ways of aligning items
type Aligns int32 //enums:enum -transform kebab

const (
	// align right edges to left edge of anchor item
	AlignRightAnchor Aligns = iota

	// align left edges
	AlignLeft

	// align horizontal centers
	AlignCenter

	// align right edges
	AlignRight

	// align left edges to right edge of anchor item
	AlignLeftAnchor

	// align left text baseline edges
	AlignBaselineHoriz

	// align bottom edges to top edge of anchor item
	AlignBottomAnchor

	// align top edges
	AlignTop

	// align middle vertical point
	AlignMiddle

	// align bottom edges
	AlignBottom

	// align top edges to bottom edge of anchor item
	AlignTopAnchor

	// align baseline points vertically
	AlignBaselineVert
)
