// Copyright (c) 2021, The Grid Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grid

import (
	"image"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/svg"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
)

// AlignView provides a range of alignment actions on selected objects.
type AlignView struct {
	gi.Layout
	GridView *GridView `copy:"-" json:"-" xml:"-" view:"-" desc:"the parent gridview"`
}

var KiT_AlignView = kit.Types.AddType(&AlignView{}, AlignViewProps)

/////////////////////////////////////////////////////////////////////////
//  Actions

// AlignAnchorBBox returns the bounding box for given type of align anchor
// and the anchor node if non-nil
func (gv *GridView) AlignAnchorBBox(aa AlignAnchors) (image.Rectangle, svg.NodeSVG) {
	es := &gv.EditState
	sv := gv.SVG()
	svoff := sv.WinBBox.Min
	var an svg.NodeSVG
	var bb image.Rectangle
	switch aa {
	case AlignFirst:
		sl := es.SelectedList(false)
		an = sl[0]
		bb = an.AsSVGNode().WinBBox
	case AlignLast:
		sl := es.SelectedList(true) // descending
		an = sl[0]
		bb = an.AsSVGNode().WinBBox
	case AlignSelectBox:
		bb = image.Rectangle{Min: es.DragSelCurBBox.Min.ToPointFloor(), Max: es.DragSelCurBBox.Max.ToPointCeil()}
	}
	bb = bb.Sub(svoff)
	return bb, an
}

// AlignMin aligns to min coordinate (Left, Top) in bbox
func (gv *GridView) AlignMin(aa AlignAnchors, dim mat32.Dims, act string) {
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sv := gv.SVG()
	svoff := sv.WinBBox.Min
	sv.UndoSave(act, es.SelectedNamesString())
	abb, an := gv.AlignAnchorBBox(aa)
	sc := mat32.Vec2{1, 1}
	odim := mat32.OtherDim(dim)
	for sn := range es.Selected {
		if sn == an {
			continue
		}
		sng := sn.AsSVGNode()
		bb := sng.WinBBox.Sub(svoff)
		del := mat32.NewVec2FmPoint(abb.Min.Sub(bb.Min))
		del.SetDim(odim, 0)
		sn.ApplyDeltaXForm(del, sc, 0, mat32.NewVec2FmPoint(bb.Min))
	}
	sv.UpdateView(true)
	gv.ChangeMade()
}

func (gv *GridView) AlignMinAnchor(aa AlignAnchors, dim mat32.Dims, act string) {
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sv := gv.SVG()
	svoff := sv.WinBBox.Min
	sv.UndoSave(act, es.SelectedNamesString())
	abb, an := gv.AlignAnchorBBox(aa)
	sc := mat32.Vec2{1, 1}
	odim := mat32.OtherDim(dim)
	for sn := range es.Selected {
		if sn == an {
			continue
		}
		sng := sn.AsSVGNode()
		bb := sng.WinBBox.Sub(svoff)
		del := mat32.NewVec2FmPoint(abb.Max.Sub(bb.Min))
		del.SetDim(odim, 0)
		sn.ApplyDeltaXForm(del, sc, 0, mat32.NewVec2FmPoint(bb.Min))
	}
	sv.UpdateView(true)
	gv.ChangeMade()
}

func (gv *GridView) AlignMax(aa AlignAnchors, dim mat32.Dims, act string) {
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sv := gv.SVG()
	svoff := sv.WinBBox.Min
	sv.UndoSave(act, es.SelectedNamesString())
	abb, an := gv.AlignAnchorBBox(aa)
	sc := mat32.Vec2{1, 1}
	odim := mat32.OtherDim(dim)
	for sn := range es.Selected {
		if sn == an {
			continue
		}
		sng := sn.AsSVGNode()
		bb := sng.WinBBox.Sub(svoff)
		del := mat32.NewVec2FmPoint(abb.Max.Sub(bb.Max))
		del.SetDim(odim, 0)
		sn.ApplyDeltaXForm(del, sc, 0, mat32.NewVec2FmPoint(bb.Min))
	}
	sv.UpdateView(true)
	gv.ChangeMade()
}

func (gv *GridView) AlignMaxAnchor(aa AlignAnchors, dim mat32.Dims, act string) {
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sv := gv.SVG()
	svoff := sv.WinBBox.Min
	sv.UndoSave(act, es.SelectedNamesString())
	abb, an := gv.AlignAnchorBBox(aa)
	sc := mat32.Vec2{1, 1}
	odim := mat32.OtherDim(dim)
	for sn := range es.Selected {
		if sn == an {
			continue
		}
		sng := sn.AsSVGNode()
		bb := sng.WinBBox.Sub(svoff)
		del := mat32.NewVec2FmPoint(abb.Min.Sub(bb.Max))
		del.SetDim(odim, 0)
		sn.ApplyDeltaXForm(del, sc, 0, mat32.NewVec2FmPoint(bb.Min))
	}
	sv.UpdateView(true)
	gv.ChangeMade()
}

func (gv *GridView) AlignCenter(aa AlignAnchors, dim mat32.Dims, act string) {
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sv := gv.SVG()
	svoff := sv.WinBBox.Min
	sv.UndoSave(act, es.SelectedNamesString())
	abb, an := gv.AlignAnchorBBox(aa)
	ctr := mat32.NewVec2FmPoint(abb.Min.Add(abb.Max)).MulScalar(0.5)
	sc := mat32.Vec2{1, 1}
	odim := mat32.OtherDim(dim)
	for sn := range es.Selected {
		if sn == an {
			continue
		}
		sng := sn.AsSVGNode()
		bb := sng.WinBBox.Sub(svoff)
		nctr := mat32.NewVec2FmPoint(bb.Min.Add(bb.Max)).MulScalar(0.5)
		del := ctr.Sub(nctr)
		del.SetDim(odim, 0)
		sn.ApplyDeltaXForm(del, sc, 0, mat32.NewVec2FmPoint(bb.Min))
	}
	sv.UpdateView(true)
	gv.ChangeMade()
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

	sv.FuncDownMeFirst(0, sv.This(), func(k ki.Ki, level int, d interface{}) bool {
		if k == sv.This() {
			return ki.Continue
		}
		if k.IsDeleted() || k.IsDestroyed() {
			return ki.Break
		}
		if k == sv.Defs.This() || NodeIsMetaData(k) {
			return ki.Break
		}
		if NodeIsLayer(k) {
			return ki.Continue
		}
		sii, issvg := k.(svg.NodeSVG)
		if !issvg {
			return ki.Continue
		}
		if _, issel := es.Selected[sii]; issel {
			return ki.Break // go no further into kids
		}
		sg := sii.AsSVGNode()

		for ap := BBLeft; ap < BBoxPointsN; ap++ {
			es.AlignPts[ap] = append(es.AlignPts[ap], ap.PointRect(sg.WinBBox))
		}
		return ki.Continue
	})
}

///////////////////////////////////////////////////////////////
//  AlignView

func (av *AlignView) Config(gv *GridView) {
	if av.HasChildren() {
		return
	}
	updt := av.UpdateStart()

	av.GridView = gv
	av.Lay = gi.LayoutVert
	av.SetProp("spacing", gi.StdDialogVSpaceUnits)

	all := gi.AddNewLayout(av, "align-lab", gi.LayoutHoriz)
	gi.AddNewLabel(all, "align-lab", "<b>Align:  </b>")

	atcb := gi.AddNewComboBox(all, "align-anchor")
	atcb.ItemsFromEnum(KiT_AlignAnchors, true, 0)

	atyp := gi.AddNewLayout(av, "align-grid", gi.LayoutGrid)
	atyp.SetProp("columns", 6)
	atyp.SetProp("spacing", gi.StdDialogVSpaceUnits)

	icprops := ki.Props{
		"width":   units.NewEm(3),
		"height":  units.NewEm(3),
		"margin":  units.NewPx(0),
		"padding": units.NewPx(0),
		"fill":    &gi.Prefs.Colors.Icon,
		"stroke":  &gi.Prefs.Colors.Font,
	}

	rta := gi.AddNewAction(atyp, "right-anchor")
	rta.SetIcon("align-right-anchor")
	rta.SetProp("#icon", icprops)
	rta.Tooltip = "align right edges of selected items to left edge of anchor item"
	rta.ActionSig.Connect(av.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		av.GridView.AlignMaxAnchor(av.AlignAnchor(), mat32.X, "AlignRightAnchor")
	})

	lft := gi.AddNewAction(atyp, "left")
	lft.SetIcon("align-left")
	lft.SetProp("#icon", icprops)
	lft.Tooltip = "align left edges of all selected items"
	lft.ActionSig.Connect(av.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		av.GridView.AlignMin(av.AlignAnchor(), mat32.X, "AlignLeft")
	})

	ctr := gi.AddNewAction(atyp, "center")
	ctr.SetIcon("align-center")
	ctr.SetProp("#icon", icprops)
	ctr.Tooltip = "align centers of all selected items"
	ctr.ActionSig.Connect(av.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		av.GridView.AlignCenter(av.AlignAnchor(), mat32.X, "AlignCenter")
	})

	rgt := gi.AddNewAction(atyp, "right")
	rgt.SetIcon("align-right")
	rgt.SetProp("#icon", icprops)
	rgt.Tooltip = "align right edges of all selected items"
	rgt.ActionSig.Connect(av.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		av.GridView.AlignMax(av.AlignAnchor(), mat32.X, "AlignRight")
	})

	lta := gi.AddNewAction(atyp, "left-anchor")
	lta.SetIcon("align-left-anchor")
	lta.SetProp("#icon", icprops)
	lta.Tooltip = "align left edges of all selected items to right edge of anchor item"
	lta.ActionSig.Connect(av.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		av.GridView.AlignMinAnchor(av.AlignAnchor(), mat32.X, "AlignLeftAnchor")
	})

	bsh := gi.AddNewAction(atyp, "baseh")
	bsh.SetIcon("align-baseline-horiz")
	bsh.SetProp("#icon", icprops)
	bsh.Tooltip = "align left text baseline edges of all selected items"
	bsh.ActionSig.Connect(av.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		av.GridView.AlignMin(av.AlignAnchor(), mat32.X, "AlignBaseH")
	})

	bta := gi.AddNewAction(atyp, "bottom-anchor")
	bta.SetIcon("align-bottom-anchor")
	bta.SetProp("#icon", icprops)
	bta.Tooltip = "align bottom edges of all selected items to top edge of anchor item"
	bta.ActionSig.Connect(av.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		av.GridView.AlignMaxAnchor(av.AlignAnchor(), mat32.Y, "AlignBotAnchor")
	})

	top := gi.AddNewAction(atyp, "top")
	top.SetIcon("align-top")
	top.SetProp("#icon", icprops)
	top.Tooltip = "align top edges of all selected items"
	top.ActionSig.Connect(av.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		av.GridView.AlignMin(av.AlignAnchor(), mat32.Y, "AlignTop")
	})

	mid := gi.AddNewAction(atyp, "middle")
	mid.SetIcon("align-middle")
	mid.SetProp("#icon", icprops)
	mid.Tooltip = "align middle vertical point of all selected items"
	mid.ActionSig.Connect(av.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		av.GridView.AlignCenter(av.AlignAnchor(), mat32.Y, "AlignTop")
	})

	bot := gi.AddNewAction(atyp, "bottom")
	bot.SetIcon("align-bottom")
	bot.SetProp("#icon", icprops)
	bot.Tooltip = "align bottom edges of all selected items"
	bot.ActionSig.Connect(av.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		av.GridView.AlignMax(av.AlignAnchor(), mat32.Y, "AlignBottom")
	})

	tpa := gi.AddNewAction(atyp, "top-anchor")
	tpa.SetIcon("align-top-anchor")
	tpa.SetProp("#icon", icprops)
	tpa.Tooltip = "align top edges of all selected items to bottom edge of anchor item"
	tpa.ActionSig.Connect(av.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		av.GridView.AlignMinAnchor(av.AlignAnchor(), mat32.Y, "AlignTopAnchor")
	})

	bsv := gi.AddNewAction(atyp, "basev")
	bsv.SetIcon("align-baseline-vert")
	bsv.SetProp("#icon", icprops)
	bsv.Tooltip = "align baseline points of all selected items vertically"
	bsv.ActionSig.Connect(av.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		av.GridView.AlignMax(av.AlignAnchor(), mat32.Y, "AlignBaseV")
	})

	gi.AddNewStretch(av, "endstr")

	av.UpdateEnd(updt)
}

// AlignAnchor returns the align anchor currently selected
func (av *AlignView) AlignAnchor() AlignAnchors {
	cb := av.AlignAnchorComboBox()
	if cb == nil {
		return AlignFirst
	}
	return AlignAnchors(cb.CurIndex)
}

// AlignAnchorComboBox returns the combobox representing align anchor options
func (av *AlignView) AlignAnchorComboBox() *gi.ComboBox {
	cbi := av.ChildByName("align-lab", 0).ChildByName("align-anchor", 0)
	if cbi != nil {
		return cbi.(*gi.ComboBox)
	}
	return nil
}

var AlignViewProps = ki.Props{
	"EnumType:Flag":    gi.KiT_VpFlags,
	"background-color": &gi.Prefs.Colors.Background,
	"color":            &gi.Prefs.Colors.Font,
	"max-width":        -1,
	"max-height":       -1,
}

// AlignAnchors are ways of anchoring alignment
type AlignAnchors int

const (
	AlignFirst AlignAnchors = iota
	AlignLast
	AlignDrawing
	AlignSelectBox
	AlignAnchorsN
)

//go:generate stringer -type=AlignAnchors

var KiT_AlignAnchors = kit.Enums.AddEnum(AlignAnchorsN, kit.NotBitFlag, nil)

func (ev AlignAnchors) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *AlignAnchors) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }
