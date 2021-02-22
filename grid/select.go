// Copyright (c) 2021, The Grid Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grid

import (
	"fmt"
	"image"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/gist"
	"github.com/goki/gi/svg"
	"github.com/goki/ki/ki"
)

func (gv *GridView) SelectToolbar() *gi.ToolBar {
	tbs := gv.ModalToolbarStack()
	tb := tbs.ChildByName("select-tb", 0).(*gi.ToolBar)
	return tb
}

// ConfigSelectToolbar configures the selection modal toolbar (default tooblar)
func (gv *GridView) ConfigSelectToolbar() {
	tb := gv.SelectToolbar()
	if tb.HasChildren() {
		return
	}
	tb.SetStretchMaxWidth()

	grs := gi.AddNewCheckBox(tb, "snap-grid")
	grs.SetText("Snap Grid")
	grs.Tooltip = "snap movement and sizing of selection to grid"
	grs.SetChecked(Prefs.SnapGrid)
	grs.ButtonSig.Connect(gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.ButtonToggled) {
			Prefs.SnapGrid = grs.IsChecked()
		}
	})

	gis := gi.AddNewCheckBox(tb, "snap-guide")
	gis.SetText("Guide")
	gis.Tooltip = "snap movement and sizing of selection to align with other elements in the scene"
	gis.SetChecked(Prefs.SnapGuide)
	gis.ButtonSig.Connect(gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.ButtonToggled) {
			Prefs.SnapGuide = gis.IsChecked()
		}
	})
	tb.AddSeparator("sep-snap")

	tb.AddAction(gi.ActOpts{Icon: "sel-group", Tooltip: "Ctrl+G: Group items together", UpdateFunc: gv.SelectedEnableFunc},
		gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			grr.SelGroup()
		})

	tb.AddAction(gi.ActOpts{Icon: "sel-ungroup", Tooltip: "Shift+Ctrl+G: ungroup items", UpdateFunc: gv.SelectedEnableFunc},
		gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			grr.SelUnGroup()
		})

	tb.AddSeparator("sep-group")

	tb.AddAction(gi.ActOpts{Icon: "sel-rotate-left", Tooltip: "Ctrl-[: rotate selection 90deg counter-clockwise", UpdateFunc: gv.SelectedEnableFunc},
		gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			grr.SelRotateLeft()
		})
	tb.AddAction(gi.ActOpts{Icon: "sel-rotate-right", Tooltip: "Ctrl-]: rotate selection 90deg clockwise", UpdateFunc: gv.SelectedEnableFunc},
		gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			grr.SelRotateRight()
		})
	tb.AddAction(gi.ActOpts{Icon: "sel-flip-horiz", Tooltip: "H: flip selection horizontally", UpdateFunc: gv.SelectedEnableFunc},
		gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			grr.SelFlipHoriz()
		})
	tb.AddAction(gi.ActOpts{Icon: "sel-flip-vert", Tooltip: "V: flip selection vertically", UpdateFunc: gv.SelectedEnableFunc},
		gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			grr.SelFlipVert()
		})

	tb.AddSeparator("sep-rot")
	tb.AddAction(gi.ActOpts{Icon: "sel-raise-top", Tooltip: "Raise selection to top (within layer)", UpdateFunc: gv.SelectedEnableFunc},
		gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			grr.SelRaiseTop()
		})
	tb.AddAction(gi.ActOpts{Icon: "sel-raise", Tooltip: "Raise selection one level (within layer)", UpdateFunc: gv.SelectedEnableFunc},
		gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			grr.SelRaise()
		})
	tb.AddAction(gi.ActOpts{Icon: "sel-lower-bottom", Tooltip: "Lower selection to bottom (within layer)", UpdateFunc: gv.SelectedEnableFunc},
		gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			grr.SelLowerBot()
		})
	tb.AddAction(gi.ActOpts{Icon: "sel-lower", Tooltip: "Lower selection one level (within layer)", UpdateFunc: gv.SelectedEnableFunc},
		gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			grr.SelLower()
		})
	tb.AddSeparator("sep-size")

	gi.AddNewLabel(tb, "posx-lab", "X: ").SetProp("vertical-align", gist.AlignMiddle)
	px := gi.AddNewSpinBox(tb, "posx")
	px.SetProp("step", 1)
	px.SetValue(0)
	px.Tooltip = "horizontal coordinate of selection, in document units"
	px.SpinBoxSig.Connect(gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		grr := recv.Embed(KiT_GridView).(*GridView)
		grr.SelSetXPos(px.Value)
	})

	gi.AddNewLabel(tb, "posy-lab", "Y: ").SetProp("vertical-align", gist.AlignMiddle)
	py := gi.AddNewSpinBox(tb, "posy")
	py.SetProp("step", 1)
	py.SetValue(0)
	py.Tooltip = "vertical coordinate of selection, in document units"
	py.SpinBoxSig.Connect(gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		grr := recv.Embed(KiT_GridView).(*GridView)
		grr.SelSetYPos(py.Value)
	})

	gi.AddNewLabel(tb, "width-lab", "W: ").SetProp("vertical-align", gist.AlignMiddle)
	wd := gi.AddNewSpinBox(tb, "width")
	wd.SetProp("step", 1)
	wd.SetValue(0)
	wd.Tooltip = "width of selection, in document units"
	wd.SpinBoxSig.Connect(gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		grr := recv.Embed(KiT_GridView).(*GridView)
		grr.SelSetWidth(wd.Value)
	})

	gi.AddNewLabel(tb, "height-lab", "H: ").SetProp("vertical-align", gist.AlignMiddle)
	ht := gi.AddNewSpinBox(tb, "height")
	ht.SetProp("step", 1)
	ht.SetValue(0)
	ht.Tooltip = "height of selection, in document units"
	ht.SpinBoxSig.Connect(gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		grr := recv.Embed(KiT_GridView).(*GridView)
		grr.SelSetHeight(ht.Value)
	})
}

// SelectedEnableFunc is an ActionUpdateFunc that inactivates action if no selected items
func (gv *GridView) SelectedEnableFunc(act *gi.Action) {
	es := &gv.EditState
	act.SetInactiveState(!es.HasSelected())
}

// UpdateSelectToolbar updates the select toolbar based on current selection
func (gv *GridView) UpdateSelectToolbar() {
	tb := gv.SelectToolbar()
	tb.UpdateActions()
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sz := es.DragSelCurBBox.Size()
	px := tb.ChildByName("posx", 8).(*gi.SpinBox)
	px.SetValue(es.DragSelCurBBox.Min.X)
	py := tb.ChildByName("posy", 9).(*gi.SpinBox)
	py.SetValue(es.DragSelCurBBox.Min.Y)
	wd := tb.ChildByName("width", 10).(*gi.SpinBox)
	wd.SetValue(sz.X)
	ht := tb.ChildByName("height", 11).(*gi.SpinBox)
	ht.SetValue(sz.Y)
}

// UpdateSelect should be called whenever selection changes
func (sv *SVGView) UpdateSelect() {
	wupdt := sv.TopUpdateStart()
	defer sv.TopUpdateEnd(wupdt)
	sv.GridView.UpdateTabs()
	sv.GridView.UpdateSelectToolbar()
	sv.UpdateSelSprites()
}

///////////////////////////////////////////////////////////////////////
//   Actions

func (gv *GridView) SelGroup() {
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sv := gv.SVG()
	sv.UndoSave("Group", es.SelectedNamesString())

	updt := sv.UpdateStart()
	sl := es.SelectedList(true) // true = descending = reverse order

	fsel := sl[len(sl)-1] // first selected -- use parent of this for new group

	ng := fsel.Parent().AddNewChild(svg.KiT_Group, "newgp").(svg.NodeSVG)
	sv.SetSVGName(ng)

	for _, se := range sl {
		ki.MoveToParent(se, ng)
	}

	es.ResetSelected()
	es.Select(ng)

	sv.UpdateEnd(updt)
	gv.UpdateAll()
}

func (gv *GridView) SelUnGroup() {
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sv := gv.SVG()
	sv.UndoSave("UnGroup", es.SelectedNamesString())

	updt := sv.UpdateStart()

	sl := es.SelectedList(true) // true = descending = reverse order
	for _, se := range sl {
		gp, isgp := se.(*svg.Group)
		if !isgp {
			continue
		}
		np := gp.Par
		klist := make(ki.Slice, len(gp.Kids)) // make a temp copy of list of kids
		for i, k := range gp.Kids {
			klist[i] = k
		}
		for _, k := range klist {
			ki.MoveToParent(k, np)
			se := k.(svg.NodeSVG)
			se.ApplyXForm(gp.Pnt.XForm) // group no longer there!
		}
		gp.Delete(ki.DestroyKids)
	}
	sv.UpdateEnd(updt)
	gv.UpdateAll()
}

func (gv *GridView) SelRotate(deg float32) {
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sv := gv.SVG()
	sv.UndoSave("Rotate", fmt.Sprintf("%g", deg))

	gv.UpdateSelectToolbar()
}

func (gv *GridView) SelScale(scx, scy float32) {
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sv := gv.SVG()
	sv.UndoSave("Scale", fmt.Sprintf("%g,%g", scx, scy))

	gv.UpdateSelectToolbar()
}

func (gv *GridView) SelRotateLeft() {
	gv.SelRotate(-90)
}

func (gv *GridView) SelRotateRight() {
	gv.SelRotate(90)
}

func (gv *GridView) SelFlipHoriz() {
	gv.SelScale(-1, 1)
}

func (gv *GridView) SelFlipVert() {
	gv.SelScale(1, -1)
}

func (gv *GridView) SelRaiseTop() {
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sv := gv.SVG()
	sv.UndoSave("RaiseTop", es.SelectedNamesString())

	sl := es.SelectedList(true) // true = descending = reverse order
	for _, se := range sl {
		par := se.Parent()
		if !(NodeIsLayer(par) || par == sv.This()) {
			continue
		}
		ci, _ := se.IndexInParent()
		par.Children().Move(ci, par.NumChildren()-1)
	}
	gv.UpdateDisp()
}

func (gv *GridView) SelRaise() {
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sv := gv.SVG()
	sv.UndoSave("Raise", es.SelectedNamesString())

	sl := es.SelectedList(true) // true = descending = reverse order
	for _, se := range sl {
		par := se.Parent()
		if !(NodeIsLayer(par) || par == sv.This()) {
			continue
		}
		ci, _ := se.IndexInParent()
		if ci < par.NumChildren()-1 {
			par.Children().Move(ci, ci+1)
		}
	}
	gv.UpdateDisp()
}

func (gv *GridView) SelLowerBot() {
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sv := gv.SVG()
	sv.UndoSave("LowerBottom", es.SelectedNamesString())

	sl := es.SelectedList(true) // true = descending = reverse order
	for _, se := range sl {
		par := se.Parent()
		if !(NodeIsLayer(par) || par == sv.This()) {
			continue
		}
		ci, _ := se.IndexInParent()
		par.Children().Move(ci, 0)
	}
	gv.UpdateDisp()
}

func (gv *GridView) SelLower() {
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sv := gv.SVG()
	sv.UndoSave("Lower", es.SelectedNamesString())

	sl := es.SelectedList(true) // true = descending = reverse order
	for _, se := range sl {
		par := se.Parent()
		if !(NodeIsLayer(par) || par == sv.This()) {
			continue
		}
		ci, _ := se.IndexInParent()
		if ci > 0 {
			par.Children().Move(ci, ci-1)
		}
	}
	gv.UpdateDisp()
}

func (gv *GridView) SelSetXPos(xp float32) {
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sv := gv.SVG()
	sv.UndoSave("MoveToX", fmt.Sprintf("%g", xp))
}

func (gv *GridView) SelSetYPos(yp float32) {
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sv := gv.SVG()
	sv.UndoSave("MoveToY", fmt.Sprintf("%g", yp))
}

func (gv *GridView) SelSetWidth(wd float32) {
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sv := gv.SVG()
	sv.UndoSave("SetWidth", fmt.Sprintf("%g", wd))
}

func (gv *GridView) SelSetHeight(ht float32) {
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sv := gv.SVG()
	sv.UndoSave("SetHeight", fmt.Sprintf("%g", ht))
}

///////////////////////////////////////////////////////////////////////
//   Select tree traversal

// SelectWithinBBox returns a list of all nodes whose WinBBox is fully contained
// within the given BBox. SVG version excludes layer groups.
func (sv *SVGView) SelectWithinBBox(bbox image.Rectangle, leavesOnly bool) []svg.NodeSVG {
	var rval []svg.NodeSVG
	sv.FuncDownMeFirst(0, sv.This(), func(k ki.Ki, level int, d interface{}) bool {
		if k == sv.This() {
			return ki.Continue
		}
		if k.IsDeleted() || k.IsDestroyed() {
			return ki.Break
		}
		if leavesOnly && k.HasChildren() {
			return ki.Continue
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
		sg := sii.AsSVGNode()
		if sg.WinBBoxInBBox(bbox) {
			rval = append(rval, sii)
			return ki.Break // don't go into groups!
		}
		return ki.Continue
	})
	return rval
}

// SelectContainsPoint finds the first node whose WinBBox contains the given
// point -- nil if none.  If leavesOnly is set then only nodes that have no
// nodes (leaves, terminal nodes) will be considered.
// if leavesOnly, only terminal leaves (no children) are included
// if excludeSel, any leaf nodes that are within the current edit selection are
// excluded,
func (sv *SVGView) SelectContainsPoint(pt image.Point, leavesOnly, excludeSel bool) svg.NodeSVG {
	es := sv.EditState()
	var rval svg.NodeSVG
	sv.FuncDownMeFirst(0, sv.This(), func(k ki.Ki, level int, d interface{}) bool {
		if k == sv.This() {
			return ki.Continue
		}
		if k.IsDeleted() || k.IsDestroyed() {
			return ki.Break
		}
		if leavesOnly && k.HasChildren() {
			return ki.Continue
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
		if excludeSel {
			if _, issel := es.Selected[sii]; issel {
				return ki.Continue
			}
		}
		sg := sii.AsSVGNode()
		if sg.PosInWinBBox(pt) {
			rval = sii
			return ki.Break
		}
		return ki.Continue
	})
	return rval
}
