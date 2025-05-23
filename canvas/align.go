// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package canvas

import (
	"cogentcore.org/cogent/canvas/cicons"
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
				w.SetIcon(AlignIcons[al]).SetType(core.ButtonTonal).SetTooltip(al.Desc())
				w.OnClick(func(e events.Event) {
					av.Canvas.SVG.Align(av.Anchor, al)
				})
			})
		}
	})
}

////////  Actions

func (sv *SVG) Align(aa AlignAnchors, al Aligns) {
	astr := al.String()
	switch al {
	case AlignRightAnchor:
		sv.AlignMaxAnchor(aa, math32.X, astr)
	case AlignLeft:
		sv.AlignMin(aa, math32.X, astr)
	case AlignCenter:
		sv.AlignCenter(aa, math32.X, astr)
	case AlignRight:
		sv.AlignMax(aa, math32.X, astr)
	case AlignLeftAnchor:
		sv.AlignMinAnchor(aa, math32.X, astr)
	case AlignBaselineHoriz:
		sv.AlignMin(aa, math32.X, astr) // todo: should be baseline, not min

	case AlignBottomAnchor:
		sv.AlignMaxAnchor(aa, math32.Y, astr)
	case AlignTop:
		sv.AlignMin(aa, math32.Y, astr)
	case AlignMiddle:
		sv.AlignCenter(aa, math32.Y, astr)
	case AlignBottom:
		sv.AlignMax(aa, math32.Y, astr)
	case AlignTopAnchor:
		sv.AlignMinAnchor(aa, math32.Y, astr)
	case AlignBaselineVert:
		sv.AlignMin(aa, math32.Y, astr) // todo: should be baseline, not min
	}
	sv.Canvas.ChangeMade()
	sv.UpdateView()
}

// alignAnchorBBox returns the bounding box for given type of align anchor
// and the anchor node if non-nil
func (sv *SVG) alignAnchorBBox(aa AlignAnchors) (math32.Box2, svg.Node) {
	es := sv.EditState()
	// svoff := sv.Root().BBox.Min
	var an svg.Node
	var bb math32.Box2
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
		bb = es.DragSelectCurrentBBox
	}
	// bb = bb.Sub(svoff)
	return bb, an
}

// alignImpl does alignment
func (sv *SVG) alignImpl(apos math32.Vector2, an svg.Node, useMax bool, dim math32.Dims, act string) {
	es := sv.EditState()
	if !es.HasSelected() {
		return
	}
	sv.UndoSave(act, es.SelectedNamesString())
	sc := math32.Vec2(1, 1)
	odim := math32.OtherDim(dim)
	for sn := range es.Selected {
		if sn == an {
			continue
		}
		sng := sn.AsNodeBase()
		bb := sng.BBox
		var del math32.Vector2
		if useMax {
			del = apos.Sub(bb.Max)
		} else {
			del = apos.Sub(bb.Min)
		}
		del.SetDim(odim, 0)
		sn.ApplyDeltaTransform(sv.SVG, del, sc, 0, bb.Min)
	}
}

// AlignMin aligns to min coordinate (Left, Top) in bbox
func (sv *SVG) AlignMin(aa AlignAnchors, dim math32.Dims, act string) {
	if !sv.EditState().HasSelected() {
		return
	}
	abb, an := sv.alignAnchorBBox(aa)
	sv.alignImpl(abb.Min, an, false, dim, act)
}

func (sv *SVG) AlignMinAnchor(aa AlignAnchors, dim math32.Dims, act string) {
	if !sv.EditState().HasSelected() {
		return
	}
	abb, an := sv.alignAnchorBBox(aa)
	sv.alignImpl(abb.Max, an, false, dim, act)
}

func (sv *SVG) AlignMax(aa AlignAnchors, dim math32.Dims, act string) {
	if !sv.EditState().HasSelected() {
		return
	}
	abb, an := sv.alignAnchorBBox(aa)
	sv.alignImpl(abb.Max, an, true, dim, act)
}

func (sv *SVG) AlignMaxAnchor(aa AlignAnchors, dim math32.Dims, act string) {
	if !sv.EditState().HasSelected() {
		return
	}
	abb, an := sv.alignAnchorBBox(aa)
	sv.alignImpl(abb.Min, an, true, dim, act)
}

func (sv *SVG) AlignCenter(aa AlignAnchors, dim math32.Dims, act string) {
	es := sv.EditState()
	if !es.HasSelected() {
		return
	}
	abb, an := sv.alignAnchorBBox(aa)
	ctr := abb.Min.Add(abb.Max).MulScalar(0.5)
	sc := math32.Vec2(1, 1)
	odim := math32.OtherDim(dim)
	for sn := range es.Selected {
		if sn == an {
			continue
		}
		sng := sn.AsNodeBase()
		bb := sng.BBox // .Sub(svoff)
		nctr := bb.Min.Add(bb.Max).MulScalar(0.5)
		del := ctr.Sub(nctr)
		del.SetDim(odim, 0)
		sn.ApplyDeltaTransform(sv.SVG, del, sc, 0, bb.Min)
	}
	sv.UpdateView()
	sv.Canvas.ChangeMade()
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
			es.AlignPts[ap] = append(es.AlignPts[ap], ap.PointBox(nb.BBox))
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

var AlignIcons = map[Aligns]icons.Icon{AlignRightAnchor: cicons.AlignRightAnchor, AlignLeft: cicons.AlignLeft, AlignCenter: cicons.AlignCenter, AlignRight: cicons.AlignRight, AlignLeftAnchor: cicons.AlignLeftAnchor, AlignBaselineHoriz: cicons.AlignBaselineHoriz, AlignBottomAnchor: cicons.AlignBottomAnchor, AlignTop: cicons.AlignTop, AlignMiddle: cicons.AlignMiddle, AlignBottom: cicons.AlignBottom, AlignTopAnchor: cicons.AlignTopAnchor, AlignBaselineVert: cicons.AlignBaselineVert}
