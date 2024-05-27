// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vector

import (
	"image"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/svg"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/views"
)

// AlignView provides a range of alignment actions on selected objects.
type AlignView struct {
	core.Frame

	AlignAnchorView views.EnumValue

	// the parent vectorview
	VectorView *VectorView `copier:"-" json:"-" xml:"-" view:"-"`
}

func (av *AlignView) OnInit() {
	av.Frame.OnInit()
	av.Maker(func(p *core.Plan) { // TODO(config)
		if av.HasChildren() {
			return
		}
		av.Style(func(s *styles.Style) {
			s.Direction = styles.Column
		})

		all := core.NewFrame(av)
		core.NewText(all).SetText("<b>Align:  </b>")
		core.NewChooser(all).SetEnum(AlignAnchorsN)

		agrid := core.NewFrame(av).Style(func(s *styles.Style) {
			s.Display = styles.Grid
			s.Columns = 6
		})

		for _, al := range AlignsValues() {
			al := al
			core.NewButton(agrid, al.String()).SetIcon(icons.Icon(al.String())).
				SetTooltip(al.Desc()).SetType(core.ButtonTonal).
				OnClick(func(e events.Event) {
					av.VectorView.Align(av.AlignAnchor(), al)
				})
		}
	})
}

/////////////////////////////////////////////////////////////////////////
//  Actions

func (vv *VectorView) Align(aa AlignAnchors, al Aligns) {
	astr := al.String()
	switch al {
	case AlignRightAnchor:
		vv.AlignMaxAnchor(aa, math32.X, astr)
	case AlignLeft:
		vv.AlignMin(aa, math32.X, astr)
	case AlignCenter:
		vv.AlignCenter(aa, math32.X, astr)
	case AlignRight:
		vv.AlignMax(aa, math32.X, astr)
	case AlignLeftAnchor:
		vv.AlignMinAnchor(aa, math32.X, astr)
	case AlignBaselineHoriz:
		vv.AlignMin(aa, math32.X, astr) // todo: should be baseline, not min

	case AlignBottomAnchor:
		vv.AlignMaxAnchor(aa, math32.Y, astr)
	case AlignTop:
		vv.AlignMin(aa, math32.Y, astr)
	case AlignMiddle:
		vv.AlignCenter(aa, math32.Y, astr)
	case AlignBottom:
		vv.AlignMax(aa, math32.Y, astr)
	case AlignTopAnchor:
		vv.AlignMinAnchor(aa, math32.Y, astr)
	case AlignBaselineVert:
		vv.AlignMin(aa, math32.Y, astr) // todo: should be baseline, not min
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
		bb = image.Rectangle{Min: es.DragSelectCurrentBBox.Min.ToPointFloor(), Max: es.DragSelectCurrentBBox.Max.ToPointCeil()}
	}
	bb = bb.Sub(svoff)
	return bb, an
}

// AlignMin aligns to min coordinate (Left, Top) in bbox
func (vv *VectorView) AlignMin(aa AlignAnchors, dim math32.Dims, act string) {
	es := &vv.EditState
	if !es.HasSelected() {
		return
	}
	sv := vv.SVG()
	svoff := sv.Root().BBox.Min
	sv.UndoSave(act, es.SelectedNamesString())
	abb, an := vv.AlignAnchorBBox(aa)
	sc := math32.Vec2(1, 1)
	odim := math32.OtherDim(dim)
	for sn := range es.Selected {
		if sn == an {
			continue
		}
		sng := sn.AsNodeBase()
		bb := sng.BBox.Sub(svoff)
		del := math32.Vector2FromPoint(abb.Min.Sub(bb.Min))
		del.SetDim(odim, 0)
		sn.ApplyDeltaTransform(vv.SSVG(), del, sc, 0, math32.Vector2FromPoint(bb.Min))
	}
	sv.UpdateView(true)
	vv.ChangeMade()
}

func (vv *VectorView) AlignMinAnchor(aa AlignAnchors, dim math32.Dims, act string) {
	es := &vv.EditState
	if !es.HasSelected() {
		return
	}
	sv := vv.SVG()
	svoff := sv.Root().BBox.Min
	sv.UndoSave(act, es.SelectedNamesString())
	abb, an := vv.AlignAnchorBBox(aa)
	sc := math32.Vec2(1, 1)
	odim := math32.OtherDim(dim)
	for sn := range es.Selected {
		if sn == an {
			continue
		}
		sng := sn.AsNodeBase()
		bb := sng.BBox.Sub(svoff)
		del := math32.Vector2FromPoint(abb.Max.Sub(bb.Min))
		del.SetDim(odim, 0)
		sn.ApplyDeltaTransform(vv.SSVG(), del, sc, 0, math32.Vector2FromPoint(bb.Min))
	}
	sv.UpdateView(true)
	vv.ChangeMade()
}

func (vv *VectorView) AlignMax(aa AlignAnchors, dim math32.Dims, act string) {
	es := &vv.EditState
	if !es.HasSelected() {
		return
	}
	sv := vv.SVG()
	svoff := sv.Root().BBox.Min
	sv.UndoSave(act, es.SelectedNamesString())
	abb, an := vv.AlignAnchorBBox(aa)
	sc := math32.Vec2(1, 1)
	odim := math32.OtherDim(dim)
	for sn := range es.Selected {
		if sn == an {
			continue
		}
		sng := sn.AsNodeBase()
		bb := sng.BBox.Sub(svoff)
		del := math32.Vector2FromPoint(abb.Max.Sub(bb.Max))
		del.SetDim(odim, 0)
		sn.ApplyDeltaTransform(vv.SSVG(), del, sc, 0, math32.Vector2FromPoint(bb.Min))
	}
	sv.UpdateView(true)
	vv.ChangeMade()
}

func (vv *VectorView) AlignMaxAnchor(aa AlignAnchors, dim math32.Dims, act string) {
	es := &vv.EditState
	if !es.HasSelected() {
		return
	}
	sv := vv.SVG()
	svoff := sv.Root().BBox.Min
	sv.UndoSave(act, es.SelectedNamesString())
	abb, an := vv.AlignAnchorBBox(aa)
	sc := math32.Vec2(1, 1)
	odim := math32.OtherDim(dim)
	for sn := range es.Selected {
		if sn == an {
			continue
		}
		sng := sn.AsNodeBase()
		bb := sng.BBox.Sub(svoff)
		del := math32.Vector2FromPoint(abb.Min.Sub(bb.Max))
		del.SetDim(odim, 0)
		sn.ApplyDeltaTransform(vv.SSVG(), del, sc, 0, math32.Vector2FromPoint(bb.Min))
	}
	sv.UpdateView(true)
	vv.ChangeMade()
}

func (vv *VectorView) AlignCenter(aa AlignAnchors, dim math32.Dims, act string) {
	es := &vv.EditState
	if !es.HasSelected() {
		return
	}
	sv := vv.SVG()
	svoff := sv.Root().BBox.Min
	sv.UndoSave(act, es.SelectedNamesString())
	abb, an := vv.AlignAnchorBBox(aa)
	ctr := math32.Vector2FromPoint(abb.Min.Add(abb.Max)).MulScalar(0.5)
	sc := math32.Vec2(1, 1)
	odim := math32.OtherDim(dim)
	for sn := range es.Selected {
		if sn == an {
			continue
		}
		sng := sn.AsNodeBase()
		bb := sng.BBox.Sub(svoff)
		nctr := math32.Vector2FromPoint(bb.Min.Add(bb.Max)).MulScalar(0.5)
		del := ctr.Sub(nctr)
		del.SetDim(odim, 0)
		sn.ApplyDeltaTransform(vv.SSVG(), del, sc, 0, math32.Vector2FromPoint(bb.Min))
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
		es.AlignPts[ap] = make([]math32.Vector2, 0)
	}

	svg.SVGWalkDownNoDefs(sv.Root(), func(kni svg.Node, knb *svg.NodeBase) bool {
		if kni.This() == sv.Root().This() {
			return tree.Continue
		}
		if NodeIsLayer(kni) {
			return tree.Continue
		}
		if _, issel := es.Selected[kni]; issel {
			return tree.Break // go no further into kids
		}
		for ap := BBLeft; ap < BBoxPointsN; ap++ {
			es.AlignPts[ap] = append(es.AlignPts[ap], ap.PointRect(knb.BBox))
		}
		return tree.Continue
	})
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
