// Copyright (c) 2021, The Grid Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grid

import (
	"fmt"
	"image"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/gist"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/gi/svg"
	"github.com/goki/ki/ki"
	"github.com/goki/mat32"
)

func (gv *GridView) SelectToolbar() *gi.Toolbar {
	tbs := gv.ModalToolbarStack()
	tb := tbs.ChildByName("select-tb", 0).(*gi.Toolbar)
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
	grs.ButtonSig.Connect(gv.This(), func(recv, send ki.Ki, sig int64, data any) {
		if sig == int64(gi.ButtonToggled) {
			Prefs.SnapGrid = grs.IsChecked()
		}
	})

	gis := gi.AddNewCheckBox(tb, "snap-guide")
	gis.SetText("Guide")
	gis.Tooltip = "snap movement and sizing of selection to align with other elements in the scene"
	gis.SetChecked(Prefs.SnapGuide)
	gis.ButtonSig.Connect(gv.This(), func(recv, send ki.Ki, sig int64, data any) {
		if sig == int64(gi.ButtonToggled) {
			Prefs.SnapGuide = gis.IsChecked()
		}
	})
	gi.NewSeparator(tb, "sep-snap")

	tb.AddAction(gi.ActOpts{Icon: "sel-group", Tooltip: "Ctrl+G: Group items together", UpdateFunc: gv.SelectedEnableFunc},
		gv.This(), func(recv, send ki.Ki, sig int64, data any) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			grr.SelGroup()
		})

	tb.AddAction(gi.ActOpts{Icon: "sel-ungroup", Tooltip: "Shift+Ctrl+G: ungroup items", UpdateFunc: gv.SelectedEnableFunc},
		gv.This(), func(recv, send ki.Ki, sig int64, data any) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			grr.SelUnGroup()
		})

	gi.NewSeparator(tb, "sep-group")

	tb.AddAction(gi.ActOpts{Icon: "sel-rotate-left", Tooltip: "Ctrl-[: rotate selection 90deg counter-clockwise", UpdateFunc: gv.SelectedEnableFunc},
		gv.This(), func(recv, send ki.Ki, sig int64, data any) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			grr.SelRotateLeft()
		})
	tb.AddAction(gi.ActOpts{Icon: "sel-rotate-right", Tooltip: "Ctrl-]: rotate selection 90deg clockwise", UpdateFunc: gv.SelectedEnableFunc},
		gv.This(), func(recv, send ki.Ki, sig int64, data any) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			grr.SelRotateRight()
		})
	tb.AddAction(gi.ActOpts{Icon: "sel-flip-horiz", Tooltip: "H: flip selection horizontally", UpdateFunc: gv.SelectedEnableFunc},
		gv.This(), func(recv, send ki.Ki, sig int64, data any) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			grr.SelFlipHoriz()
		})
	tb.AddAction(gi.ActOpts{Icon: "sel-flip-vert", Tooltip: "V: flip selection vertically", UpdateFunc: gv.SelectedEnableFunc},
		gv.This(), func(recv, send ki.Ki, sig int64, data any) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			grr.SelFlipVert()
		})

	gi.NewSeparator(tb, "sep-rot")
	tb.AddAction(gi.ActOpts{Icon: "sel-raise-top", Tooltip: "Raise selection to top (within layer)", UpdateFunc: gv.SelectedEnableFunc},
		gv.This(), func(recv, send ki.Ki, sig int64, data any) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			grr.SelRaiseTop()
		})
	tb.AddAction(gi.ActOpts{Icon: "sel-raise", Tooltip: "Raise selection one level (within layer)", UpdateFunc: gv.SelectedEnableFunc},
		gv.This(), func(recv, send ki.Ki, sig int64, data any) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			grr.SelRaise()
		})
	tb.AddAction(gi.ActOpts{Icon: "sel-lower-bottom", Tooltip: "Lower selection to bottom (within layer)", UpdateFunc: gv.SelectedEnableFunc},
		gv.This(), func(recv, send ki.Ki, sig int64, data any) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			grr.SelLowerBot()
		})
	tb.AddAction(gi.ActOpts{Icon: "sel-lower", Tooltip: "Lower selection one level (within layer)", UpdateFunc: gv.SelectedEnableFunc},
		gv.This(), func(recv, send ki.Ki, sig int64, data any) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			grr.SelLower()
		})
	gi.NewSeparator(tb, "sep-size")

	gi.AddNewLabel(tb, "posx-lab", "X: ").SetProp("vertical-align", gist.AlignMiddle)
	px := gi.AddNewSpinBox(tb, "posx")
	px.SetProp("step", 1)
	px.SetValue(0)
	px.Tooltip = "horizontal coordinate of selection, in document units"
	px.SpinBoxSig.Connect(gv.This(), func(recv, send ki.Ki, sig int64, data any) {
		grr := recv.Embed(KiT_GridView).(*GridView)
		grr.SelSetXPos(px.Value)
	})

	gi.AddNewLabel(tb, "posy-lab", "Y: ").SetProp("vertical-align", gist.AlignMiddle)
	py := gi.AddNewSpinBox(tb, "posy")
	py.SetProp("step", 1)
	py.SetValue(0)
	py.Tooltip = "vertical coordinate of selection, in document units"
	py.SpinBoxSig.Connect(gv.This(), func(recv, send ki.Ki, sig int64, data any) {
		grr := recv.Embed(KiT_GridView).(*GridView)
		grr.SelSetYPos(py.Value)
	})

	gi.AddNewLabel(tb, "width-lab", "W: ").SetProp("vertical-align", gist.AlignMiddle)
	wd := gi.AddNewSpinBox(tb, "width")
	wd.SetProp("step", 1)
	wd.SetValue(0)
	wd.Tooltip = "width of selection, in document units"
	wd.SpinBoxSig.Connect(gv.This(), func(recv, send ki.Ki, sig int64, data any) {
		grr := recv.Embed(KiT_GridView).(*GridView)
		grr.SelSetWidth(wd.Value)
	})

	gi.AddNewLabel(tb, "height-lab", "H: ").SetProp("vertical-align", gist.AlignMiddle)
	ht := gi.AddNewSpinBox(tb, "height")
	ht.SetProp("step", 1)
	ht.SetValue(0)
	ht.Tooltip = "height of selection, in document units"
	ht.SpinBoxSig.Connect(gv.This(), func(recv, send ki.Ki, sig int64, data any) {
		grr := recv.Embed(KiT_GridView).(*GridView)
		grr.SelSetHeight(ht.Value)
	})
}

// SelectedEnableFunc is an ActionUpdateFunc that inactivates action if no selected items
func (gv *GridView) SelectedEnableFunc(act *gi.Button) {
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
	sz := es.DragSelEffBBox.Size()
	px := tb.ChildByName("posx", 8).(*gi.SpinBox)
	px.SetValue(es.DragSelEffBBox.Min.X)
	py := tb.ChildByName("posy", 9).(*gi.SpinBox)
	py.SetValue(es.DragSelEffBBox.Min.Y)
	wd := tb.ChildByName("width", 10).(*gi.SpinBox)
	wd.SetValue(sz.X)
	ht := tb.ChildByName("height", 11).(*gi.SpinBox)
	ht.SetValue(sz.Y)
}

// UpdateSelect should be called whenever selection changes
func (sv *SVGView) UpdateSelect() {
	wupdt := sv.TopUpdateStart()
	defer sv.TopUpdateEnd(wupdt)
	win := sv.GridView.ParentWindow()
	es := sv.EditState()
	sv.GridView.UpdateTabs()
	sv.GridView.UpdateSelectToolbar()
	if es.Tool == NodeTool {
		sv.UpdateNodeSprites()
		sv.RemoveSelSprites(win)
	} else {
		sv.RemoveNodeSprites(win)
		sv.UpdateSelSprites()
	}
}

func (sv *SVGView) RemoveSelSprites(win *gi.Window) {
	InactivateSprites(win, SpReshapeBBox)
	InactivateSprites(win, SpSelBBox)
	es := sv.EditState()
	es.NSelSprites = 0
	win.UpdateSig()
}

func (sv *SVGView) UpdateSelSprites() {
	win := sv.GridView.ParentWindow()
	updt := win.UpdateStart()
	defer win.UpdateEnd(updt)

	es := sv.EditState()
	es.UpdateSelBBox()
	if !es.HasSelected() {
		sv.RemoveSelSprites(win)
		return
	}

	for i := SpBBoxUpL; i <= SpBBoxRtM; i++ {
		spi := i // key to get a unique local var
		SpriteConnectEvent(win, SpReshapeBBox, spi, 0, image.ZP, sv.This(), func(recv, send ki.Ki, sig int64, d any) {
			ssvg := recv.Embed(KiT_SVGView).(*SVGView)
			ssvg.SelSpriteEvent(spi, oswin.EventType(sig), d)
		})
	}
	sv.SetBBoxSpritePos(SpReshapeBBox, 0, es.SelBBox)
	sv.SetSelSpritePos()

	win.UpdateSig()
}

func (sv *SVGView) SetSelSpritePos() {
	win := sv.GridView.ParentWindow()
	es := sv.EditState()
	nsel := es.NSelSprites

	es.NSelSprites = 0
	if len(es.Selected) > 1 {
		nbox := 0
		sl := es.SelectedList(false)
		for si, sii := range sl {
			sn := sii.AsSVGNode()
			if sn.WinBBox.Size() == image.ZP {
				continue
			}
			bb := mat32.Box2{}
			bb.SetFromRect(sn.WinBBox)
			sv.SetBBoxSpritePos(SpSelBBox, si, bb)
			nbox++
		}
		es.NSelSprites = nbox
	}

	for si := es.NSelSprites; si < nsel; si++ {
		for i := SpBBoxUpL; i <= SpBBoxRtM; i++ {
			spnm := SpriteName(SpSelBBox, i, si)
			win.InactivateSprite(spnm)
		}
	}
}

// SetBBoxSpritePos sets positions of given type of sprites
func (sv *SVGView) SetBBoxSpritePos(typ Sprites, idx int, bbox mat32.Box2) {
	win := sv.GridView.ParentWindow()
	_, spsz := HandleSpriteSize(1)
	midX := int(0.5 * (bbox.Min.X + bbox.Max.X - float32(spsz.X)))
	midY := int(0.5 * (bbox.Min.Y + bbox.Max.Y - float32(spsz.Y)))
	for i := SpBBoxUpL; i <= SpBBoxRtM; i++ {
		spi := i // key to get a unique local var
		sp := Sprite(win, typ, spi, idx, image.ZP)
		switch spi {
		case SpBBoxUpL:
			SetSpritePos(sp, image.Point{int(bbox.Min.X), int(bbox.Min.Y)})
		case SpBBoxUpC:
			SetSpritePos(sp, image.Point{midX, int(bbox.Min.Y)})
		case SpBBoxUpR:
			SetSpritePos(sp, image.Point{int(bbox.Max.X), int(bbox.Min.Y)})
		case SpBBoxDnL:
			SetSpritePos(sp, image.Point{int(bbox.Min.X), int(bbox.Max.Y)})
		case SpBBoxDnC:
			SetSpritePos(sp, image.Point{midX, int(bbox.Max.Y)})
		case SpBBoxDnR:
			SetSpritePos(sp, image.Point{int(bbox.Max.X), int(bbox.Max.Y)})
		case SpBBoxLfM:
			SetSpritePos(sp, image.Point{int(bbox.Min.X), midY})
		case SpBBoxRtM:
			SetSpritePos(sp, image.Point{int(bbox.Max.X), midY})
		}
	}
}

func (sv *SVGView) SelSpriteEvent(sp Sprites, et oswin.EventType, d any) {
	win := sv.GridView.ParentWindow()
	es := sv.EditState()
	es.SelNoDrag = false
	switch et {
	case oswin.MouseEvent:
		me := d.(*mouse.Event)
		me.SetProcessed()
		// fmt.Printf("click %s\n", sp)
		if me.Action == mouse.Press {
			win.SpriteDragging = SpriteName(SpReshapeBBox, sp, 0)
			sv.EditState().DragSelStart(me.Where)
			// fmt.Printf("dragging: %s\n", win.SpriteDragging)
		} else if me.Action == mouse.Release {
			sv.ManipDone()
		}
	case oswin.MouseDragEvent:
		me := d.(*mouse.DragEvent)
		me.SetProcessed()
		// fmt.Printf("drag %v delta: %v\n", sp, me.Delta())
		if me.HasAnyModifier(key.Alt) {
			sv.SpriteRotateDrag(sp, me.Delta(), win)
		} else {
			sv.SpriteReshapeDrag(sp, win, me)
		}
	}
}

// SetRubberBand updates the rubber band postion
func (sv *SVGView) SetRubberBand(cur image.Point) {
	win := sv.GridView.ParentWindow()
	es := sv.EditState()

	if !es.InAction() {
		es.ActStart("BoxSelect", fmt.Sprintf("%v", es.DragStartPos))
		es.ActUnlock()
	}
	es.DragCurPos = cur

	bbox := image.Rectangle{Min: es.DragStartPos, Max: es.DragCurPos}
	bbox = bbox.Canon()

	sz := bbox.Size()
	if sz.X < 4 {
		sz.X = 4
	}
	if sz.Y < 4 {
		sz.Y = 4
	}
	rt := Sprite(win, SpRubberBand, SpBBoxUpC, 0, sz)
	rb := Sprite(win, SpRubberBand, SpBBoxDnC, 0, sz)
	rr := Sprite(win, SpRubberBand, SpBBoxRtM, 0, sz)
	rl := Sprite(win, SpRubberBand, SpBBoxLfM, 0, sz)
	SetSpritePos(rt, bbox.Min)
	SetSpritePos(rb, image.Point{bbox.Min.X, bbox.Max.Y})
	SetSpritePos(rr, image.Point{bbox.Max.X, bbox.Min.Y})
	SetSpritePos(rl, bbox.Min)

	win.UpdateSig()
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
	sl := es.SelectedListDepth(sv, false) // ascending depth order

	fsel := sl[len(sl)-1] // first selected -- use parent of this for new group

	fidx, _ := fsel.IndexInParent()

	ng := fsel.Parent().InsertNewChild(svg.KiT_Group, fidx, "newgp").(svg.NodeSVG)
	sv.SetSVGName(ng)

	for _, se := range sl {
		ki.MoveToParent(se, ng)
	}

	es.ResetSelected()
	es.Select(ng)

	sv.UpdateEnd(updt)
	gv.UpdateAll()
	gv.ChangeMade()
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
		gidx, _ := gp.IndexInParent()
		klist := make(ki.Slice, len(gp.Kids)) // make a temp copy of list of kids
		for i, k := range gp.Kids {
			klist[i] = k
		}
		for i, k := range klist {
			ki.SetParent(k, nil)
			gp.DeleteChild(k, false) // no destroy
			np.InsertChild(k, gidx+i)
			se := k.(svg.NodeSVG)
			if !gp.Pnt.Transform.IsIdentity() {
				se.ApplyTransform(gp.Pnt.Transform) // group no longer there!
			}
		}
		gp.Delete(ki.DestroyKids)
	}
	sv.UpdateEnd(updt)
	gv.UpdateAll()
	gv.ChangeMade()
}

func (gv *GridView) SelRotate(deg float32) {
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sv := gv.SVG()
	sv.UndoSave("Rotate", fmt.Sprintf("%g", deg))

	svoff := sv.WinBBox.Min
	del := mat32.Vec2{}
	sc := mat32.Vec2{1, 1}
	rot := mat32.DegToRad(deg)
	for sn := range es.Selected {
		sng := sn.AsSVGNode()
		sz := mat32.NewVec2FmPoint(sng.WinBBox.Size())
		mn := mat32.NewVec2FmPoint(sng.WinBBox.Min.Sub(svoff))
		ctr := mn.Add(sz.MulScalar(.5))
		sn.ApplyDeltaTransform(del, sc, rot, ctr)
	}
	sv.UpdateView(true)
	gv.ChangeMade()
}

func (gv *GridView) SelScale(scx, scy float32) {
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sv := gv.SVG()
	sv.UndoSave("Scale", fmt.Sprintf("%g,%g", scx, scy))

	svoff := sv.WinBBox.Min
	del := mat32.Vec2{}
	sc := mat32.Vec2{scx, scy}
	for sn := range es.Selected {
		sng := sn.AsSVGNode()
		sz := mat32.NewVec2FmPoint(sng.WinBBox.Size())
		mn := mat32.NewVec2FmPoint(sng.WinBBox.Min.Sub(svoff))
		ctr := mn.Add(sz.MulScalar(.5))
		sn.ApplyDeltaTransform(del, sc, 0, ctr)
	}
	sv.UpdateView(true)
	gv.ChangeMade()
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
	gv.ChangeMade()
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
	gv.ChangeMade()
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
	gv.ChangeMade()
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
	gv.ChangeMade()
}

func (gv *GridView) SelSetXPos(xp float32) {
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sv := gv.SVG()
	sv.UndoSave("MoveToX", fmt.Sprintf("%g", xp))
	// todo
	gv.ChangeMade()
}

func (gv *GridView) SelSetYPos(yp float32) {
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sv := gv.SVG()
	sv.UndoSave("MoveToY", fmt.Sprintf("%g", yp))
	// todo
	gv.ChangeMade()
}

func (gv *GridView) SelSetWidth(wd float32) {
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sv := gv.SVG()
	sv.UndoSave("SetWidth", fmt.Sprintf("%g", wd))
	// todo
	gv.ChangeMade()
}

func (gv *GridView) SelSetHeight(ht float32) {
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sv := gv.SVG()
	sv.UndoSave("SetHeight", fmt.Sprintf("%g", ht))
	// todo
	gv.ChangeMade()
}

///////////////////////////////////////////////////////////////////////
//   Select tree traversal

// SelectWithinBBox returns a list of all nodes whose WinBBox is fully contained
// within the given BBox. SVG version excludes layer groups.
func (sv *SVGView) SelectWithinBBox(bbox image.Rectangle, leavesOnly bool) []svg.NodeSVG {
	var rval []svg.NodeSVG
	var curlay ki.Ki
	sv.FuncDownMeFirst(0, sv.This(), func(k ki.Ki, level int, d any) bool {
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
			return ki.Break
		}
		if txt, istxt := sii.(*svg.Text); istxt { // no tspans
			if txt.Text != "" {
				if _, istxt := txt.Par.(*svg.Text); istxt {
					return ki.Break
				}
			}
		}
		sg := sii.AsSVGNode()
		if sg.Pnt.Off {
			return ki.Break
		}
		nl := NodeParentLayer(k)
		if nl != nil {
			if (curlay != nil && nl != curlay) || LayerIsLocked(nl) || !LayerIsVisible(nl) {
				return ki.Break
			}
		}
		if sg.WinBBoxInBBox(bbox) {
			// fmt.Printf("%s sel bb: %v in: %v\n", sg.Name(), sg.WinBBox, bbox)
			rval = append(rval, sii)
			if curlay == nil && nl != nil {
				curlay = nl
			}
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
	var curlay ki.Ki
	fn := es.FirstSelectedNode()
	if fn != nil {
		curlay = NodeParentLayer(fn)
	}
	var rval svg.NodeSVG
	sv.FuncDownMeFirst(0, sv.This(), func(k ki.Ki, level int, d any) bool {
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
			return ki.Break
		}
		if txt, istxt := sii.(*svg.Text); istxt { // no tspans
			if txt.Text != "" {
				if _, istxt := txt.Par.(*svg.Text); istxt {
					return ki.Break
				}
			}
		}
		if excludeSel {
			if _, issel := es.Selected[sii]; issel {
				return ki.Continue
			}
			if _, issel := es.RecentlySelected[sii]; issel {
				return ki.Continue
			}
		}
		sg := sii.AsSVGNode()
		if sg.Pnt.Off {
			return ki.Break
		}
		nl := NodeParentLayer(k)
		if nl != nil {
			if (curlay != nil && nl != curlay) || LayerIsLocked(nl) || !LayerIsVisible(nl) {
				return ki.Break
			}
		}
		if sg.PosInWinBBox(pt) {
			rval = sii
			return ki.Break
		}
		return ki.Continue
	})
	return rval
}
