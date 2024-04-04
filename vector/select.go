// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vector

import (
	"fmt"
	"image"

	"cogentcore.org/core/events"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/giv"
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

// ConfigSelectToolbar configures the selection modal toolbar (default toolbar)
func (gv *VectorView) ConfigSelectToolbar() {
	tb := gv.SelectToolbar()
	if tb.HasChildren() {
		return
	}

	grs := gi.NewSwitch(tb).SetText("Snap grid").
		SetTooltip("snap movement and sizing of selection to grid").
		SetChecked(Prefs.SnapVector)
	grs.OnChange(func(e events.Event) {
		Prefs.SnapVector = grs.IsChecked()
	})

	gis := gi.NewSwitch(tb).SetText("Snap guide").
		SetTooltip("snap movement and sizing of selection to align with other elements in the scene").
		SetChecked(Prefs.SnapGuide)
	gis.OnChange(func(e events.Event) {
		Prefs.SnapGuide = gis.IsChecked()
	})

	gi.NewSeparator(tb)

	gv.NewSelectFuncButton(tb, gv.SelectGroup).SetText("Group").
		SetIcon("sel-group").SetShortcut("Command+G")

	gv.NewSelectFuncButton(tb, gv.SelectUnGroup).SetText("Ungroup").
		SetIcon("sel-ungroup").SetShortcut("Shift+Command+G")

	gi.NewSeparator(tb)

	gv.NewSelectFuncButton(tb, gv.SelectRotateLeft).SetText("").
		SetIcon("sel-rotate-left").SetShortcut("Command+[")
	gv.NewSelectFuncButton(tb, gv.SelectRotateRight).SetText("").
		SetIcon("sel-rotate-right").SetShortcut("Command+]")

	gv.NewSelectFuncButton(tb, gv.SelectFlipHorizontal).SetText("").SetIcon("sel-flip-horiz")
	gv.NewSelectFuncButton(tb, gv.SelectFlipVertical).SetText("").SetIcon("sel-flip-vert")

	gi.NewSeparator(tb)

	gv.NewSelectFuncButton(tb, gv.SelectRaiseTop).SetText("").SetIcon("sel-raise-top")
	gv.NewSelectFuncButton(tb, gv.SelectRaise).SetText("").SetIcon("sel-raise")
	gv.NewSelectFuncButton(tb, gv.SelectLowerBottom).SetText("").SetIcon("sel-lower-bottom")
	gv.NewSelectFuncButton(tb, gv.SelectLower).SetText("").SetIcon("sel-lower")

	gi.NewSeparator(tb)

	gi.NewLabel(tb).SetText("X: ")
	giv.NewValue(tb, &gv.EditState.DragSelectEffectiveBBox.Min.X).SetDoc("Horizontal coordinate of selection, in document units").OnChange(func(e events.Event) {
		gv.SelectSetXPos(gv.EditState.DragSelectEffectiveBBox.Min.X)
	})

	gi.NewLabel(tb).SetText("Y: ")
	py := gi.NewSpinner(tb, "posy").SetStep(1).SetTooltip("Vertical coordinate of selection, in document units")
	py.OnChange(func(e events.Event) {
		gv.SelectSetYPos(py.Value)
	})

	gi.NewLabel(tb).SetText("W: ")
	wd := gi.NewSpinner(tb, "width").SetStep(1).SetTooltip("Width of selection, in document units")
	wd.OnChange(func(e events.Event) {
		gv.SelectSetWidth(wd.Value)
	})

	gi.NewLabel(tb).SetText("H: ")
	ht := gi.NewSpinner(tb, "height").SetStep(1).SetTooltip("Height of selection, in document units")
	ht.OnChange(func(e events.Event) {
		gv.SelectSetHeight(ht.Value)
	})
}

// NewSelectFuncButton returns a new func button that is only enabled when
// there is an item selected.
func (gv *VectorView) NewSelectFuncButton(parent ki.Ki, fun any) *giv.FuncButton {
	bt := giv.NewFuncButton(parent, fun)
	bt.StyleFirst(func(s *styles.Style) {
		s.SetEnabled(gv.EditState.HasSelected())
	})
	return bt
}

// UpdateSelectToolbar updates the select toolbar based on current selection
func (gv *VectorView) UpdateSelectToolbar() {
	tb := gv.SelectToolbar()
	tb.NeedsRender()
	// tb.Update()
	// es := &gv.EditState
	// if !es.HasSelected() {
	// 	return
	// }
	// sz := es.DragSelEffBBox.Size()
	// tb.ChildByName("posx", 8).(*gi.Spinner).SetValue(es.DragSelEffBBox.Min.X)
	// tb.ChildByName("posy", 9).(*gi.Spinner).SetValue(es.DragSelEffBBox.Min.Y)
	// tb.ChildByName("width", 10).(*gi.Spinner).SetValue(sz.X)
	// tb.ChildByName("height", 11).(*gi.Spinner).SetValue(sz.Y)
}

// UpdateSelect should be called whenever selection changes
func (sv *SVGView) UpdateSelect() {
	es := sv.EditState()
	sv.VectorView.UpdateTabs()
	sv.VectorView.UpdateSelectToolbar()
	if es.Tool == NodeTool {
		sv.UpdateNodeSprites()
		sv.RemoveSelSprites()
	} else {
		sv.RemoveNodeSprites()
		sv.UpdateSelSprites()
	}
	sv.NeedsRender()
}

func (sv *SVGView) RemoveSelSprites() {
	InactivateSprites(sv, SpReshapeBBox)
	InactivateSprites(sv, SpSelBBox)
	es := sv.EditState()
	es.NSelectSprites = 0
	// win.UpdateSig()
}

func (sv *SVGView) UpdateSelSprites() {
	// win := sv.VectorView.ParentWindow()
	// updt := win.UpdateStart()
	// defer win.UpdateEnd(updt)

	es := sv.EditState()
	es.UpdateSelectBBox()
	if !es.HasSelected() {
		sv.RemoveSelSprites()
		return
	}

	for i := SpBBoxUpL; i <= SpBBoxRtM; i++ {
		// SpriteConnectEvent(win, SpReshapeBBox, spi, 0, image.ZP, sv.This(), func(recv, send ki.Ki, sig int64, d any) {
		// 	ssvg := recv.Embed(KiT_SVGView).(*SVGView)
		// 	ssvg.SelSpriteEvent(spi, events.EventType(sig), d)
		// })
		Sprite(sv, SpReshapeBBox, i, 0, image.Point{})
	}
	sv.SetBBoxSpritePos(SpReshapeBBox, 0, es.SelectBBox)
	sv.SetSelSpritePos()

	// win.UpdateSig()
}

func (sv *SVGView) SetSelSpritePos() {
	es := sv.EditState()
	nsel := es.NSelectSprites

	es.NSelectSprites = 0
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
		es.NSelectSprites = nbox
	}

	sprites := &sv.Scene.Stage.Sprites
	for si := es.NSelectSprites; si < nsel; si++ {
		for i := SpBBoxUpL; i <= SpBBoxRtM; i++ {
			spnm := SpriteName(SpSelBBox, i, si)
			sprites.InactivateSprite(spnm)
		}
	}
}

// SetBBoxSpritePos sets positions of given type of sprites
func (sv *SVGView) SetBBoxSpritePos(typ Sprites, idx int, bbox mat32.Box2) {
	bbox.Min.SetAdd(mat32.V2FromPoint(sv.Geom.ContentBBox.Min))
	bbox.Max.SetAdd(mat32.V2FromPoint(sv.Geom.ContentBBox.Min))

	_, spsz := HandleSpriteSize(1)
	midX := int(0.5 * (bbox.Min.X + bbox.Max.X - float32(spsz.X)))
	midY := int(0.5 * (bbox.Min.Y + bbox.Max.Y - float32(spsz.Y)))
	for i := SpBBoxUpL; i <= SpBBoxRtM; i++ {
		sp := Sprite(sv, typ, i, idx, image.ZP)
		switch i {
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
	rt := Sprite(sv, SpRubberBand, SpBBoxUpC, 0, sz)
	rb := Sprite(sv, SpRubberBand, SpBBoxDnC, 0, sz)
	rr := Sprite(sv, SpRubberBand, SpBBoxRtM, 0, sz)
	rl := Sprite(sv, SpRubberBand, SpBBoxLfM, 0, sz)
	SetSpritePos(rt, bbox.Min)
	SetSpritePos(rb, image.Point{bbox.Min.X, bbox.Max.Y})
	SetSpritePos(rr, image.Point{bbox.Max.X, bbox.Min.Y})
	SetSpritePos(rl, bbox.Min)

	sv.NeedsRender()
}

///////////////////////////////////////////////////////////////////////
//   Actions

// SelectGroup groups items together
func (gv *VectorView) SelectGroup() { //gti:add
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sv := gv.SVG()
	sv.UndoSave("Group", es.SelectedNamesString())

	sl := es.SelectedListDepth(sv, false) // ascending depth order

	fsel := sl[len(sl)-1] // first selected -- use parent of this for new group

	fidx := fsel.IndexInParent()

	ng := fsel.Parent().InsertNewChild(svg.GroupType, fidx).(svg.Node)
	sv.SetSVGName(ng)

	for _, se := range sl {
		ki.MoveToParent(se, ng)
	}

	es.ResetSelected()
	es.Select(ng)

	gv.UpdateAll()
	gv.ChangeMade()
}

// SelectUnGroup ungroups items from each other
func (gv *VectorView) SelectUnGroup() { //gti:add
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sv := gv.SVG()
	sv.UndoSave("UnGroup", es.SelectedNamesString())

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
			se := k.(svg.Node)
			if !gp.Paint.Transform.IsIdentity() {
				se.ApplyTransform(sv.SSVG(), gp.Paint.Transform) // group no longer there!
			}
		}
	}
	gv.UpdateAll()
	gv.ChangeMade()
}

func (gv *VectorView) SelectRotate(deg float32) {
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sv := gv.SVG()
	sv.UndoSave("Rotate", fmt.Sprintf("%g", deg))

	svoff := sv.Geom.ContentBBox.Min
	del := mat32.Vec2{}
	sc := mat32.V2(1, 1)
	rot := mat32.DegToRad(deg)
	for sn := range es.Selected {
		sng := sn.AsNodeBase()
		sz := mat32.V2FromPoint(sng.BBox.Size())
		mn := mat32.V2FromPoint(sng.BBox.Min.Sub(svoff))
		ctr := mn.Add(sz.MulScalar(.5))
		sn.ApplyDeltaTransform(sv.SSVG(), del, sc, rot, ctr)
	}
	sv.UpdateView(true)
	gv.ChangeMade()
}

func (gv *VectorView) SelectScale(scx, scy float32) {
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sv := gv.SVG()
	sv.UndoSave("Scale", fmt.Sprintf("%g,%g", scx, scy))

	svoff := sv.Geom.ContentBBox.Min
	del := mat32.Vec2{}
	sc := mat32.V2(scx, scy)
	for sn := range es.Selected {
		sng := sn.AsNodeBase()
		sz := mat32.V2FromPoint(sng.BBox.Size())
		mn := mat32.V2FromPoint(sng.BBox.Min.Sub(svoff))
		ctr := mn.Add(sz.MulScalar(.5))
		sn.ApplyDeltaTransform(sv.SSVG(), del, sc, 0, ctr)
	}
	sv.UpdateView(true)
	gv.ChangeMade()
}

// SelectRotateLeft rotates the selection 90 degrees counter-clockwise
func (gv *VectorView) SelectRotateLeft() { //gti:add
	gv.SelectRotate(-90)
}

// SelectRotateRight rotates the selection 90 degrees clockwise
func (gv *VectorView) SelectRotateRight() { //gti:add
	gv.SelectRotate(90)
}

// SelectFlipHorizontal flips the selection horizontally
func (gv *VectorView) SelectFlipHorizontal() { //gti:add
	gv.SelectScale(-1, 1)
}

// SelectFlipVertical flips the selection vertically
func (gv *VectorView) SelectFlipVertical() { //gti:add
	gv.SelectScale(1, -1)
}

// SelectRaiseTop raises the selection to the top of the layer
func (gv *VectorView) SelectRaiseTop() { //gti:add
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sv := gv.SVG()
	sv.UndoSave("RaiseTop", es.SelectedNamesString())

	sl := es.SelectedList(true) // true = descending = reverse order
	for _, se := range sl {
		parent := se.Parent()
		if !(NodeIsLayer(parent) || parent == sv.This()) {
			continue
		}
		ci := se.IndexInParent()
		parent.Children().Move(ci, parent.NumChildren()-1)
	}
	gv.UpdateDisp()
	gv.ChangeMade()
}

// SelectRaise raises the selection by one level in the layer
func (gv *VectorView) SelectRaise() { //gti:add
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sv := gv.SVG()
	sv.UndoSave("Raise", es.SelectedNamesString())

	sl := es.SelectedList(true) // true = descending = reverse order
	for _, se := range sl {
		parent := se.Parent()
		if !(NodeIsLayer(parent) || parent == sv.This()) {
			continue
		}
		ci := se.IndexInParent()
		if ci < parent.NumChildren()-1 {
			parent.Children().Move(ci, ci+1)
		}
	}
	gv.UpdateDisp()
	gv.ChangeMade()
}

// SelectLowerBottom lowers the selection to the bottom of the layer
func (gv *VectorView) SelectLowerBottom() { //gti:add
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sv := gv.SVG()
	sv.UndoSave("LowerBottom", es.SelectedNamesString())

	sl := es.SelectedList(true) // true = descending = reverse order
	for _, se := range sl {
		parent := se.Parent()
		if !(NodeIsLayer(parent) || parent == sv.This()) {
			continue
		}
		ci := se.IndexInParent()
		parent.Children().Move(ci, 0)
	}
	gv.UpdateDisp()
	gv.ChangeMade()
}

// SelectLower lowers the selection by one level in the layer
func (gv *VectorView) SelectLower() { //gti:add
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sv := gv.SVG()
	sv.UndoSave("Lower", es.SelectedNamesString())

	sl := es.SelectedList(true) // true = descending = reverse order
	for _, se := range sl {
		parent := se.Parent()
		if !(NodeIsLayer(parent) || parent == sv.This()) {
			continue
		}
		ci := se.IndexInParent()
		if ci > 0 {
			parent.Children().Move(ci, ci-1)
		}
	}
	gv.UpdateDisp()
	gv.ChangeMade()
}

func (gv *VectorView) SelectSetXPos(xp float32) {
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sv := gv.SVG()
	sv.UndoSave("MoveToX", fmt.Sprintf("%g", xp))
	// todo
	gv.ChangeMade()
}

func (gv *VectorView) SelectSetYPos(yp float32) {
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sv := gv.SVG()
	sv.UndoSave("MoveToY", fmt.Sprintf("%g", yp))
	// todo
	gv.ChangeMade()
}

func (gv *VectorView) SelectSetWidth(wd float32) {
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sv := gv.SVG()
	sv.UndoSave("SetWidth", fmt.Sprintf("%g", wd))
	// todo
	gv.ChangeMade()
}

func (gv *VectorView) SelectSetHeight(ht float32) {
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
	svg.SVGWalkPreNoDefs(sv.Root(), func(kni svg.Node, knb *svg.NodeBase) bool {
		if kni.This() == sv.Root().This() {
			return ki.Continue
		}
		if leavesOnly && kni.HasChildren() {
			return ki.Continue
		}
		if NodeIsLayer(kni) {
			return ki.Continue
		}
		if txt, istxt := kni.(*svg.Text); istxt { // no tspans
			if txt.Text != "" {
				if _, istxt := txt.Par.(*svg.Text); istxt {
					return ki.Break
				}
			}
		}
		if knb.Paint.Off {
			return ki.Break
		}
		nl := NodeParentLayer(kni)
		if nl != nil {
			if (curlay != nil && nl != curlay) || LayerIsLocked(nl) || !LayerIsVisible(nl) {
				return ki.Break
			}
		}
		if knb.BBox.In(bbox) {
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
// point in scene coordinates; nil if none.  If leavesOnly is set then only nodes that have no
// nodes (leaves, terminal nodes) will be considered.
// if leavesOnly, only terminal leaves (no children) are included
// if excludeSel, any leaf nodes that are within the current edit selection are
// excluded,
func (sv *SVGView) SelectContainsPoint(pt image.Point, leavesOnly, excludeSel bool) svg.Node {
	pt = pt.Sub(sv.Geom.ContentBBox.Min)
	es := sv.EditState()
	var curlay ki.Ki
	fn := es.FirstSelectedNode()
	if fn != nil {
		curlay = NodeParentLayer(fn)
	}
	var rval svg.Node
	svg.SVGWalkPreNoDefs(sv.Root(), func(kni svg.Node, knb *svg.NodeBase) bool {
		if kni.This() == sv.Root().This() {
			return ki.Continue
		}
		if leavesOnly && kni.HasChildren() {
			return ki.Continue
		}
		if NodeIsLayer(kni) {
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
		if knb.Paint.Off {
			return ki.Break
		}
		nl := NodeParentLayer(kni)
		if nl != nil {
			if (curlay != nil && nl != curlay) || LayerIsLocked(nl) || !LayerIsVisible(nl) {
				return ki.Break
			}
		}
		if pt.In(knb.BBox) {
			rval = kni
			return ki.Break
		}
		return ki.Continue
	})
	return rval
}
