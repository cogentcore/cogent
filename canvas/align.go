// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package canvas

import (
	"image"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/svg"
	"cogentcore.org/core/tree"
)

// AlignView provides a range of alignment actions on selected objects.
type AlignView struct {
	core.Frame

	// Anchor is the alignment anchor
	Anchor AlignAnchors

	// the parent [Canvas]
	Canvas *Canvas `copier:"-" json:"-" xml:"-" display:"-"`
}

func (av *AlignView) Init() {
	av.Frame.Init()
	av.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
	})

	tree.AddChild(av, func(w *core.Frame) {
		w.Styler(func(s *styles.Style) {
			s.Direction = styles.Row
		})

		tree.AddChild(w, func(w *core.Text) {
			w.SetText("<b>Align:  </b>")
		})

		tree.AddChild(w, func(w *core.Chooser) {
			w.SetEnum(av.Anchor)
			w.OnChange(func(e events.Event) {
				if aval, ok := w.CurrentItem.Value.(AlignAnchors); ok {
					av.Anchor = aval
				}
			})
			w.Updater(func() {
				w.SetCurrentValue(av.Anchor)
			})
		})
	})

	tree.AddChild(av, func(w *core.Frame) {
		w.Styler(func(s *styles.Style) {
			s.Display = styles.Grid
			s.Columns = 6
		})

		for _, al := range AlignsValues() {
			tree.AddChildAt(w, al.String(), func(w *core.Button) {
				w.SetIcon(icons.Icon(al.String())).SetType(core.ButtonTonal).
					SetTooltip(al.Desc()).
					OnClick(func(e events.Event) {
						av.Canvas.Align(av.Anchor, al)
					})
			})
		}
	})
}

/////////////////////////////////////////////////////////////////////////
//  Actions

func (vv *Canvas) Align(aa AlignAnchors, al Aligns) {
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
func (vv *Canvas) AlignAnchorBBox(aa AlignAnchors) (image.Rectangle, svg.Node) {
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
func (vv *Canvas) AlignMin(aa AlignAnchors, dim math32.Dims, act string) {
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

func (vv *Canvas) AlignMinAnchor(aa AlignAnchors, dim math32.Dims, act string) {
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

func (vv *Canvas) AlignMax(aa AlignAnchors, dim math32.Dims, act string) {
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

func (vv *Canvas) AlignMaxAnchor(aa AlignAnchors, dim math32.Dims, act string) {
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

func (vv *Canvas) AlignCenter(aa AlignAnchors, dim math32.Dims, act string) {
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
func (sv *SVG) GatherAlignPoints() {
	es := sv.EditState()
	if !es.HasSelected() {
		return
	}

	for ap := BBLeft; ap < BBoxPointsN; ap++ {
		es.AlignPts[ap] = make([]math32.Vector2, 0)
	}

	svg.SVGWalkDownNoDefs(sv.Root(), func(n svg.Node, nb *svg.NodeBase) bool {
		if n == sv.Root() {
			return tree.Continue
		}
		if NodeIsLayer(n) {
			return tree.Continue
		}
		if _, issel := es.Selected[n]; issel {
			return tree.Break // go no further into kids
		}
		for ap := BBLeft; ap < BBoxPointsN; ap++ {
			es.AlignPts[ap] = append(es.AlignPts[ap], ap.PointRect(nb.BBox))
		}
		return tree.Continue
	})
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
