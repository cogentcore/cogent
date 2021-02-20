// Copyright (c) 2021, The Grid Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grid

import (
	"github.com/goki/gi/gi"
	"github.com/goki/gi/svg"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// AlignView provides a range of alignment actions on selected objects.
type AlignView struct {
	gi.Layout
	GridView *GridView `copy:"-" json:"-" xml:"-" view:"-" desc:"the parent gridview"`
}

var KiT_AlignView = kit.Types.AddType(&AlignView{}, AlignViewProps)

/////////////////////////////////////////////////////////////////////////
//  Actions

func (gv *GridView) AlignLeft() {
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sv := gv.SVG()
	sv.UndoSave("AlignLeft", es.SelectedNamesString())
}

// GatherAlignPoints gets all the potential points of alignment for objects not
// in selection group
func (sv *SVGView) GatherAlignPoints() {
	es := sv.EditState()
	if !es.HasSelected() {
		return
	}

	for ap := BBLeft; ap < BBoxPointsN; ap++ {
		es.AlignPts[ap] = make([]float32, 0)
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
			return ki.Continue
		}
		sg := sii.AsSVGNode()

		for ap := BBLeft; ap < BBoxPointsN; ap++ {
			es.AlignPts[ap] = append(es.AlignPts[ap], ap.ValRect(sg.WinBBox))
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

	atcb := gi.AddNewComboBox(all, "align-rel")
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
		av.GridView.AlignLeft()
	})

	lft := gi.AddNewAction(atyp, "left")
	lft.SetIcon("align-left")
	lft.SetProp("#icon", icprops)
	lft.Tooltip = "align left edges of all selected items"
	lft.ActionSig.Connect(av.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		av.GridView.AlignLeft()
	})

	ctr := gi.AddNewAction(atyp, "center")
	ctr.SetIcon("align-center")
	ctr.SetProp("#icon", icprops)
	ctr.Tooltip = "align centers of all selected items"
	ctr.ActionSig.Connect(av.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		av.GridView.AlignLeft()
	})

	rgt := gi.AddNewAction(atyp, "right")
	rgt.SetIcon("align-right")
	rgt.SetProp("#icon", icprops)
	rgt.Tooltip = "align right edges of all selected items"
	rgt.ActionSig.Connect(av.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		av.GridView.AlignLeft()
	})

	lta := gi.AddNewAction(atyp, "left-anchor")
	lta.SetIcon("align-left-anchor")
	lta.SetProp("#icon", icprops)
	lta.Tooltip = "align left edges of all selected items to right edge of anchor item"
	lta.ActionSig.Connect(av.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		av.GridView.AlignLeft()
	})

	bsh := gi.AddNewAction(atyp, "baseh")
	bsh.SetIcon("align-baseline-horiz")
	bsh.SetProp("#icon", icprops)
	bsh.Tooltip = "align left text baseline edges of all selected items"
	bsh.ActionSig.Connect(av.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		av.GridView.AlignLeft()
	})

	bta := gi.AddNewAction(atyp, "bottom-anchor")
	bta.SetIcon("align-bottom-anchor")
	bta.SetProp("#icon", icprops)
	bta.Tooltip = "align bottom edges of all selected items to top edge of anchor item"
	bta.ActionSig.Connect(av.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		av.GridView.AlignLeft()
	})

	top := gi.AddNewAction(atyp, "top")
	top.SetIcon("align-top")
	top.SetProp("#icon", icprops)
	top.Tooltip = "align top edges of all selected items"
	top.ActionSig.Connect(av.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		av.GridView.AlignLeft()
	})

	mid := gi.AddNewAction(atyp, "middle")
	mid.SetIcon("align-middle")
	mid.SetProp("#icon", icprops)
	mid.Tooltip = "align middle vertical point of all selected items"
	mid.ActionSig.Connect(av.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		av.GridView.AlignLeft()
	})

	bot := gi.AddNewAction(atyp, "bottom")
	bot.SetIcon("align-bottom")
	bot.SetProp("#icon", icprops)
	bot.Tooltip = "align bottom edges of all selected items"
	bot.ActionSig.Connect(av.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		av.GridView.AlignLeft()
	})

	tpa := gi.AddNewAction(atyp, "top-anchor")
	tpa.SetIcon("align-top-anchor")
	tpa.SetProp("#icon", icprops)
	tpa.Tooltip = "align top edges of all selected items to bottom edge of anchor item"
	tpa.ActionSig.Connect(av.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		av.GridView.AlignLeft()
	})

	bsv := gi.AddNewAction(atyp, "basev")
	bsv.SetIcon("align-baseline-vert")
	bsv.SetProp("#icon", icprops)
	bsv.Tooltip = "align baseline points of all selected items vertically"
	bsv.ActionSig.Connect(av.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		av.GridView.AlignLeft()
	})

	gi.AddNewStretch(av, "endstr")

	av.UpdateEnd(updt)
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
