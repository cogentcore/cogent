// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vector

import (
	"fmt"
	"image"

	"cogentcore.org/core/gi"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/mat32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/svg"
)

func (gv *VectorView) SelectToolbar() *gi.Toolbar {
	tbs := gv.ModalToolbarStack()
	tb := tbs.ChildByName("select-tb", 0).(*gi.Toolbar)
	return tb
}

// ConfigSelectToolbar configures the selection modal toolbar (default tooblar)
func (gv *VectorView) ConfigSelectToolbar() {
	tb := gv.SelectToolbar()
	if tb.HasChildren() {
		return
	}

	grs := gi.NewSwitch(tb, "snap-grid")
	grs.SetText("Snap Vector")
	grs.Tooltip = "snap movement and sizing of selection to grid"
	grs.SetChecked(Prefs.SnapVector)
	grs.ButtonSig.Connect(gv.This(), func(recv, send ki.Ki, sig int64, data any) {
		if sig == int64(gi.ButtonToggled) {
			Prefs.SnapVector = grs.IsChecked()
		}
	})

	gis := gi.NewSwitch(tb, "snap-guide")
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
			grr := recv.Embed(KiT_VectorView).(*VectorView)
			grr.SelGroup()
		})

	tb.AddAction(gi.ActOpts{Icon: "sel-ungroup", Tooltip: "Shift+Ctrl+G: ungroup items", UpdateFunc: gv.SelectedEnableFunc},
		gv.This(), func(recv, send ki.Ki, sig int64, data any) {
			grr := recv.Embed(KiT_VectorView).(*VectorView)
			grr.SelUnGroup()
		})

	gi.NewSeparator(tb, "sep-group")

	tb.AddAction(gi.ActOpts{Icon: "sel-rotate-left", Tooltip: "Ctrl-[: rotate selection 90deg counter-clockwise", UpdateFunc: gv.SelectedEnableFunc},
		gv.This(), func(recv, send ki.Ki, sig int64, data any) {
			grr := recv.Embed(KiT_VectorView).(*VectorView)
			grr.SelRotateLeft()
		})
	tb.AddAction(gi.ActOpts{Icon: "sel-rotate-right", Tooltip: "Ctrl-]: rotate selection 90deg clockwise", UpdateFunc: gv.SelectedEnableFunc},
		gv.This(), func(recv, send ki.Ki, sig int64, data any) {
			grr := recv.Embed(KiT_VectorView).(*VectorView)
			grr.SelRotateRight()
		})
	tb.AddAction(gi.ActOpts{Icon: "sel-flip-horiz", Tooltip: "H: flip selection horizontally", UpdateFunc: gv.SelectedEnableFunc},
		gv.This(), func(recv, send ki.Ki, sig int64, data any) {
			grr := recv.Embed(KiT_VectorView).(*VectorView)
			grr.SelFlipHoriz()
		})
	tb.AddAction(gi.ActOpts{Icon: "sel-flip-vert", Tooltip: "V: flip selection vertically", UpdateFunc: gv.SelectedEnableFunc},
		gv.This(), func(recv, send ki.Ki, sig int64, data any) {
			grr := recv.Embed(KiT_VectorView).(*VectorView)
			grr.SelFlipVert()
		})

	gi.NewSeparator(tb, "sep-rot")
	tb.AddAction(gi.ActOpts{Icon: "sel-raise-top", Tooltip: "Raise selection to top (within layer)", UpdateFunc: gv.SelectedEnableFunc},
		gv.This(), func(recv, send ki.Ki, sig int64, data any) {
			grr := recv.Embed(KiT_VectorView).(*VectorView)
			grr.SelRaiseTop()
		})
	tb.AddAction(gi.ActOpts{Icon: "sel-raise", Tooltip: "Raise selection one level (within layer)", UpdateFunc: gv.SelectedEnableFunc},
		gv.This(), func(recv, send ki.Ki, sig int64, data any) {
			grr := recv.Embed(KiT_VectorView).(*VectorView)
			grr.SelRaise()
		})
	tb.AddAction(gi.ActOpts{Icon: "sel-lower-bottom", Tooltip: "Lower selection to bottom (within layer)", UpdateFunc: gv.SelectedEnableFunc},
		gv.This(), func(recv, send ki.Ki, sig int64, data any) {
			grr := recv.Embed(KiT_VectorView).(*VectorView)
			grr.SelLowerBot()
		})
	tb.AddAction(gi.ActOpts{Icon: "sel-lower", Tooltip: "Lower selection one level (within layer)", UpdateFunc: gv.SelectedEnableFunc},
		gv.This(), func(recv, send ki.Ki, sig int64, data any) {
			grr := recv.Embed(KiT_VectorView).(*VectorView)
			grr.SelLower()
		})
	gi.NewSeparator(tb, "sep-size")

	gi.NewLabel(tb, "posx-lab", "X: ").SetProp("vertical-align", styles.AlignMiddle)
	px := gi.NewSpinner(tb, "posx")
	px.SetProp("step", 1)
	px.SetValue(0)
	px.Tooltip = "horizontal coordinate of selection, in document units"
	px.SpinnerSig.Connect(gv.This(), func(recv, send ki.Ki, sig int64, data any) {
		grr := recv.Embed(KiT_VectorView).(*VectorView)
		grr.SelSetXPos(px.Value)
	})

	gi.NewLabel(tb, "posy-lab", "Y: ").SetProp("vertical-align", styles.AlignMiddle)
	py := gi.NewSpinner(tb, "posy")
	py.SetProp("step", 1)
	py.SetValue(0)
	py.Tooltip = "vertical coordinate of selection, in document units"
	py.SpinnerSig.Connect(gv.This(), func(recv, send ki.Ki, sig int64, data any) {
		grr := recv.Embed(KiT_VectorView).(*VectorView)
		grr.SelSetYPos(py.Value)
	})

	gi.NewLabel(tb, "width-lab", "W: ").SetProp("vertical-align", styles.AlignMiddle)
	wd := gi.NewSpinner(tb, "width")
	wd.SetProp("step", 1)
	wd.SetValue(0)
	wd.Tooltip = "width of selection, in document units"
	wd.SpinnerSig.Connect(gv.This(), func(recv, send ki.Ki, sig int64, data any) {
		grr := recv.Embed(KiT_VectorView).(*VectorView)
		grr.SelSetWidth(wd.Value)
	})

	gi.NewLabel(tb, "height-lab", "H: ").SetProp("vertical-align", styles.AlignMiddle)
	ht := gi.NewSpinner(tb, "height")
	ht.SetProp("step", 1)
	ht.SetValue(0)
	ht.Tooltip = "height of selection, in document units"
	ht.SpinnerSig.Connect(gv.This(), func(recv, send ki.Ki, sig int64, data any) {
		grr := recv.Embed(KiT_VectorView).(*VectorView)
		grr.SelSetHeight(ht.Value)
	})
}

// SelectedEnableFunc is an ActionUpdateFunc that inactivates action if no selected items
func (gv *VectorView) SelectedEnableFunc(act *gi.Button) {
	es := &gv.EditState
	act.SetInactiveState(!es.HasSelected())
}

// UpdateSelectToolbar updates the select toolbar based on current selection
func (gv *VectorView) UpdateSelectToolbar() {
	tb := gv.SelectToolbar()
	tb.UpdateActions()
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sz := es.DragSelEffBBox.Size()
	px := tb.ChildByName("posx", 8).(*gi.Spinner)
	px.SetValue(es.DragSelEffBBox.Min.X)
	py := tb.ChildByName("posy", 9).(*gi.Spinner)
	py.SetValue(es.DragSelEffBBox.Min.Y)
	wd := tb.ChildByName("width", 10).(*gi.Spinner)
	wd.SetValue(sz.X)
	ht := tb.ChildByName("height", 11).(*gi.Spinner)
	ht.SetValue(sz.Y)
}

// UpdateSelect should be called whenever selection changes
func (sv *SVGView) UpdateSelect() {
	wupdt := sv.TopUpdateStart()
	defer sv.TopUpdateEnd(wupdt)
	win := sv.VectorView.ParentWindow()
	es := sv.EditState()
	sv.VectorView.UpdateTabs()
	sv.VectorView.UpdateSelectToolbar()
	if es.Tool == NodeTool {
		sv.UpdateNodeSprites()
		sv.RemoveSelSprites(win)
	} else {
		sv.RemoveNodeSprites(win)
		sv.UpdateSelSprites()
	}
}

/*
func (sv *SVGView) RemoveSelSprites(win *gi.Window) {
	InactivateSprites(win, SpReshapeBBox)
	InactivateSprites(win, SpSelBBox)
	es := sv.EditState()
	es.NSelSprites = 0
	win.UpdateSig()
}
*/

func (sv *SVGView) UpdateSelSprites() {
	win := sv.VectorView.ParentWindow()
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
			ssvg.SelSpriteEvent(spi, events.EventType(sig), d)
		})
	}
	sv.SetBBoxSpritePos(SpReshapeBBox, 0, es.SelBBox)
	sv.SetSelSpritePos()

	win.UpdateSig()
}

func (sv *SVGView) SetSelSpritePos() {
	win := sv.VectorView.ParentWindow()
	es := sv.EditState()
	nsel := es.NSelSprites

	es.NSelSprites = 0
	if len(es.Selected) > 1 {
		nbox := 0
		sl := es.SelectedList(false)
		for si, sii := range sl {
			sn := sii.AsNodeBase()
			if sn.BBox.Size() == image.ZP {
				continue
			}
			bb := mat32.Box2{}
			bb.SetFromRect(sn.BBox)
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
	win := sv.VectorView.ParentWindow()
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

/*
func (sv *SVGView) SelSpriteEvent(sp Sprites, et events.EventType, d any) {
	win := sv.VectorView.ParentWindow()
	es := sv.EditState()
	es.SelNoDrag = false
	switch et {
	case events.MouseEvent:
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
	case events.MouseDragEvent:
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
*/

// SetRubberBand updates the rubber band postion
func (sv *SVGView) SetRubberBand(cur image.Point) {
	win := sv.VectorView.ParentWindow()
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

func (gv *VectorView) SelGroup() {
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

	ng := fsel.Parent().InsertNewChild(svg.KiT_Group, fidx, "newgp").(svg.Node)
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

func (gv *VectorView) SelUnGroup() {
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
			se := k.(svg.Node)
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

func (gv *VectorView) SelRotate(deg float32) {
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sv := gv.SVG()
	sv.UndoSave("Rotate", fmt.Sprintf("%g", deg))

	svoff := sv.BBox.Min
	del := mat32.Vec2{}
	sc := mat32.V2(1, 1)
	rot := mat32.DegToRad(deg)
	for sn := range es.Selected {
		sng := sn.AsNodeBase()
		sz := mat32.V2FromPoint(sng.BBox.Size())
		mn := mat32.V2FromPoint(sng.BBox.Min.Sub(svoff))
		ctr := mn.Add(sz.MulScalar(.5))
		sn.ApplyDeltaTransform(del, sc, rot, ctr)
	}
	sv.UpdateView(true)
	gv.ChangeMade()
}

func (gv *VectorView) SelScale(scx, scy float32) {
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sv := gv.SVG()
	sv.UndoSave("Scale", fmt.Sprintf("%g,%g", scx, scy))

	svoff := sv.BBox.Min
	del := mat32.Vec2{}
	sc := mat32.V2(scx, scy)
	for sn := range es.Selected {
		sng := sn.AsNodeBase()
		sz := mat32.V2FromPoint(sng.BBox.Size())
		mn := mat32.V2FromPoint(sng.BBox.Min.Sub(svoff))
		ctr := mn.Add(sz.MulScalar(.5))
		sn.ApplyDeltaTransform(del, sc, 0, ctr)
	}
	sv.UpdateView(true)
	gv.ChangeMade()
}

func (gv *VectorView) SelRotateLeft() {
	gv.SelRotate(-90)
}

func (gv *VectorView) SelRotateRight() {
	gv.SelRotate(90)
}

func (gv *VectorView) SelFlipHoriz() {
	gv.SelScale(-1, 1)
}

func (gv *VectorView) SelFlipVert() {
	gv.SelScale(1, -1)
}

func (gv *VectorView) SelRaiseTop() {
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

func (gv *VectorView) SelRaise() {
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

func (gv *VectorView) SelLowerBot() {
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

func (gv *VectorView) SelLower() {
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

func (gv *VectorView) SelSetXPos(xp float32) {
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sv := gv.SVG()
	sv.UndoSave("MoveToX", fmt.Sprintf("%g", xp))
	// todo
	gv.ChangeMade()
}

func (gv *VectorView) SelSetYPos(yp float32) {
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sv := gv.SVG()
	sv.UndoSave("MoveToY", fmt.Sprintf("%g", yp))
	// todo
	gv.ChangeMade()
}

func (gv *VectorView) SelSetWidth(wd float32) {
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sv := gv.SVG()
	sv.UndoSave("SetWidth", fmt.Sprintf("%g", wd))
	// todo
	gv.ChangeMade()
}

func (gv *VectorView) SelSetHeight(ht float32) {
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

// SelectWithinBBox returns a list of all nodes whose BBox is fully contained
// within the given BBox. SVG version excludes layer groups.
func (sv *SVGView) SelectWithinBBox(bbox image.Rectangle, leavesOnly bool) []svg.Node {
	var rval []svg.Node
	var curlay ki.Ki
	svg.SVGWalkPreNoDefs(sv, func(kni svg.Node, knb *svg.NodeBase) bool {
		if kni.This() == sv.This() {
			return ki.Continue
		}
		if leavesOnly && k.HasChildren() {
			return ki.Continue
		}
		if NodeIsLayer(k) {
			return ki.Continue
		}
		if txt, istxt := kni.(*svg.Text); istxt { // no tspans
			if txt.Text != "" {
				if _, istxt := txt.Par.(*svg.Text); istxt {
					return ki.Break
				}
			}
		}
		if knb.Pnt.Off {
			return ki.Break
		}
		nl := NodeParentLayer(k)
		if nl != nil {
			if (curlay != nil && nl != curlay) || LayerIsLocked(nl) || !LayerIsVisible(nl) {
				return ki.Break
			}
		}
		if knb.BBoxInBBox(bbox) {
			// fmt.Printf("%s sel bb: %v in: %v\n", knb.Name(), knb.BBox, bbox)
			rval = append(rval, kni)
			if curlay == nil && nl != nil {
				curlay = nl
			}
			return ki.Break // don't go into groups!
		}
		return ki.Continue
	})
	return rval
}

// SelectContainsPoint finds the first node whose BBox contains the given
// point -- nil if none.  If leavesOnly is set then only nodes that have no
// nodes (leaves, terminal nodes) will be considered.
// if leavesOnly, only terminal leaves (no children) are included
// if excludeSel, any leaf nodes that are within the current edit selection are
// excluded,
func (sv *SVGView) SelectContainsPoint(pt image.Point, leavesOnly, excludeSel bool) svg.Node {
	es := sv.EditState()
	var curlay ki.Ki
	fn := es.FirstSelectedNode()
	if fn != nil {
		curlay = NodeParentLayer(fn)
	}
	var rval svg.Node
	svg.SVGWalkPreNoDefs(sv, func(kni svg.Node, knb *svg.NodeBase) bool {
		if kni.This() == sv.This() {
			return ki.Continue
		}
		if leavesOnly && k.HasChildren() {
			return ki.Continue
		}
		if NodeIsLayer(k) {
			return ki.Continue
		}
		if txt, istxt := kni.(*svg.Text); istxt { // no tspans
			if txt.Text != "" {
				if _, istxt := txt.Par.(*svg.Text); istxt {
					return ki.Break
				}
			}
		}
		if excludeSel {
			if _, issel := es.Selected[kni]; issel {
				return ki.Continue
			}
			if _, issel := es.RecentlySelected[kni]; issel {
				return ki.Continue
			}
		}
		if knb.Pnt.Off {
			return ki.Break
		}
		nl := NodeParentLayer(k)
		if nl != nil {
			if (curlay != nil && nl != curlay) || LayerIsLocked(nl) || !LayerIsVisible(nl) {
				return ki.Break
			}
		}
		if knb.PosInBBox(pt) {
			rval = kni
			return ki.Break
		}
		return ki.Continue
	})
	return rval
}
