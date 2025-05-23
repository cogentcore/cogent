// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package canvas

import (
	"fmt"
	"image"

	"cogentcore.org/cogent/canvas/cicons"
	"cogentcore.org/core/base/slicesx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/svg"
	"cogentcore.org/core/tree"
)

// InitSelectButton sets the given widget to only be enabled when
// there is an item selected.
func (vc *Canvas) InitSelectButton(w core.Widget) {
	w.AsWidget().FirstStyler(func(s *styles.Style) {
		s.SetEnabled(vc.EditState.HasSelected())
	})
}

// MakeSelectToolbar adds the select toolbar to the given plan.
func (vc *Canvas) MakeSelectToolbar(p *tree.Plan) {
	tree.Add(p, func(w *core.Switch) {
		core.Bind(&Settings.SnapGrid, w)
		w.SetText("Snap grid")
		w.SetTooltip("Whether to snap movement and sizing of selection to the grid")
	})
	tree.Add(p, func(w *core.Switch) {
		core.Bind(&Settings.SnapGuide, w)
		w.SetText("Snap guide")
		w.SetTooltip("snap movement and sizing of selection to align with other elements in the scene")
	})
	tree.Add(p, func(w *core.Separator) {})
	tree.Add(p, func(w *core.FuncButton) {
		vc.InitSelectButton(w)
		w.SetFunc(vc.SelectGroup).SetText("Group").SetIcon(cicons.SelGroup).SetShortcut("Command+G")
	})
	tree.Add(p, func(w *core.FuncButton) {
		vc.InitSelectButton(w)
		w.SetFunc(vc.SelectUnGroup).SetText("Ungroup").SetIcon(cicons.SelUngroup).SetShortcut("Command+Shift+G")
	})
	tree.Add(p, func(w *core.Separator) {})
	tree.Add(p, func(w *core.FuncButton) {
		vc.InitSelectButton(w)
		w.SetFunc(vc.SelectRotateLeft).SetText("").SetIcon(cicons.SelRotateLeft).SetShortcut("Command+[")
	})
	tree.Add(p, func(w *core.FuncButton) {
		vc.InitSelectButton(w)
		w.SetFunc(vc.SelectRotateRight).SetText("").SetIcon(cicons.SelRotateRight).SetShortcut("Command+]")
	})
	tree.Add(p, func(w *core.FuncButton) {
		vc.InitSelectButton(w)
		w.SetFunc(vc.SelectFlipHorizontal).SetText("").SetIcon(cicons.SelFlipHoriz)
	})
	tree.Add(p, func(w *core.FuncButton) {
		vc.InitSelectButton(w)
		w.SetFunc(vc.SelectFlipVertical).SetText("").SetIcon(cicons.SelFlipVert)
	})
	tree.Add(p, func(w *core.Separator) {})
	tree.Add(p, func(w *core.FuncButton) {
		vc.InitSelectButton(w)
		w.SetFunc(vc.SelectRaiseTop).SetText("").SetIcon(cicons.SelRaiseTop)
	})
	tree.Add(p, func(w *core.FuncButton) {
		vc.InitSelectButton(w)
		w.SetFunc(vc.SelectRaise).SetText("").SetIcon(cicons.SelRaise)
	})
	tree.Add(p, func(w *core.FuncButton) {
		vc.InitSelectButton(w)
		w.SetFunc(vc.SelectLowerBottom).SetText("").SetIcon(cicons.SelLowerBottom)
	})
	tree.Add(p, func(w *core.FuncButton) {
		vc.InitSelectButton(w)
		w.SetFunc(vc.SelectLower).SetText("").SetIcon(cicons.SelLower)
	})
	tree.Add(p, func(w *core.Separator) {})
	tree.Add(p, func(w *core.Text) {
		w.SetText("X: ")
	})
	// TODO(config):
	// core.NewValue(tb, &gv.EditState.DragSelectEffectiveBBox.Min.X).SetDoc("Horizontal coordinate of selection, in document units").OnChange(func(e events.Event) {
	// 	gv.SelectSetXPos(gv.EditState.DragSelectEffectiveBBox.Min.X)
	// })

	tree.Add(p, func(w *core.Text) {
		w.SetText("Y: ")
	})
	// py := core.NewSpinner(tb).SetStep(1).SetTooltip("Vertical coordinate of selection, in document units")
	// py.OnChange(func(e events.Event) {
	// 	// gv.SelectSetYPos(py.Value)
	// })

	// core.NewText(tb).SetText("W: ")
	// wd := core.NewSpinner(tb).SetStep(1).SetTooltip("Width of selection, in document units")
	// wd.OnChange(func(e events.Event) {
	// 	// gv.SelectSetWidth(wd.Value)
	// })

	// core.NewText(tb).SetText("H: ")
	// ht := core.NewSpinner(tb).SetStep(1).SetTooltip("Height of selection, in document units")
	// ht.OnChange(func(e events.Event) {
	// 	// gv.SelectSetHeight(ht.Value)
	// })
}

// UpdateSelectToolbar updates the select toolbar based on current selection
func (vc *Canvas) UpdateSelectToolbar() {
	// tb := vc.SelectToolbar()
	// tb.NeedsRender()
	// tb.Update()
	// es := &gv.EditState
	// if !es.HasSelected() {
	// 	return
	// }
	// sz := es.DragSelEffBBox.Size()
	// tb.ChildByName("posx", 8).(*core.Spinner).SetValue(es.DragSelEffBBox.Min.X)
	// tb.ChildByName("posy", 9).(*core.Spinner).SetValue(es.DragSelEffBBox.Min.Y)
	// tb.ChildByName("width", 10).(*core.Spinner).SetValue(sz.X)
	// tb.ChildByName("height", 11).(*core.Spinner).SetValue(sz.Y)
}

// UpdateSelect should be called whenever selection changes
func (sv *SVG) UpdateSelect() {
	es := sv.EditState()
	sv.Canvas.UpdateTabs()
	sv.Canvas.UpdateSelectToolbar()
	if es.Tool == NodeTool {
		sv.UpdateNodeSprites()
		sv.RemoveSelSprites()
	} else {
		sv.RemoveNodeSprites()
		sv.UpdateSelSprites()
	}
	sv.NeedsRender()
}

func (sv *SVG) RemoveSelSprites() {
	InactivateSprites(sv, SpReshapeBBox)
	InactivateSprites(sv, SpSelBBox)
	es := sv.EditState()
	es.NSelectSprites = 0
}

func (sv *SVG) UpdateSelSprites() {
	es := sv.EditState()
	es.UpdateSelectBBox()
	if !es.HasSelected() {
		sv.RemoveSelSprites()
		return
	}

	for i := SpBBoxUpL; i <= SpBBoxRtM; i++ {
		Sprite(sv, SpReshapeBBox, i, 0, image.Point{}, func(sp *core.Sprite) {
			sp.OnSlideStart(func(e events.Event) {
				es.DragSelStart(e.Pos())
				e.SetHandled()
			})
			sp.OnSlideMove(func(e events.Event) {
				if e.HasAnyModifier(key.Alt) {
					sv.SpriteRotateDrag(i, e.PrevDelta())
				} else {
					sv.SpriteReshapeDrag(i, e)
				}
				e.SetHandled()
			})
			sp.OnSlideStop(func(e events.Event) {
				sv.ManipDone()
				e.SetHandled()
			})
		})
	}
	sv.SetBBoxSpritePos(SpReshapeBBox, 0, es.SelectBBox)
	sv.SetSelSpritePos()
}

func (sv *SVG) SetSelSpritePos() {
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
			bb := math32.Box2{}
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
func (sv *SVG) SetBBoxSpritePos(typ Sprites, idx int, bbox math32.Box2) {
	bbox.Min.SetAdd(math32.FromPoint(sv.Geom.ContentBBox.Min))
	bbox.Max.SetAdd(math32.FromPoint(sv.Geom.ContentBBox.Min))

	_, spsz := HandleSpriteSize(1)
	midX := int(0.5 * (bbox.Min.X + bbox.Max.X - float32(spsz.X)))
	midY := int(0.5 * (bbox.Min.Y + bbox.Max.Y - float32(spsz.Y)))
	for i := SpBBoxUpL; i <= SpBBoxRtM; i++ {
		sp := Sprite(sv, typ, i, idx, image.ZP, nil)
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
func (sv *SVG) SelSpriteEvent(sp Sprites, et events.EventType, d any) {
	win := sv.Vector.ParentWindow()
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

// SetRubberBand updates the rubber band position.
func (sv *SVG) SetRubberBand(cur image.Point) {
	es := sv.EditState()

	if !es.InAction() {
		es.ActStart(BoxSelect, fmt.Sprintf("%v", es.DragStartPos))
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
	rt := Sprite(sv, SpRubberBand, SpBBoxUpC, 0, sz, nil)
	rb := Sprite(sv, SpRubberBand, SpBBoxDnC, 0, sz, nil)
	rr := Sprite(sv, SpRubberBand, SpBBoxRtM, 0, sz, nil)
	rl := Sprite(sv, SpRubberBand, SpBBoxLfM, 0, sz, nil)
	SetSpritePos(rt, bbox.Min)
	SetSpritePos(rb, image.Point{bbox.Min.X, bbox.Max.Y})
	SetSpritePos(rr, image.Point{bbox.Max.X, bbox.Min.Y})
	SetSpritePos(rl, bbox.Min)

	sv.NeedsRender()
}

///////////////////////////////////////////////////////////////////////
//   Actions

// SelectGroup groups items together
func (gv *Canvas) SelectGroup() { //types:add
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sv := gv.SVG()
	sv.UndoSave("Group", es.SelectedNamesString())

	sl := es.SelectedListDepth(sv, false) // ascending depth order

	fsel := sl[len(sl)-1] // first selected -- use parent of this for new group

	fidx := fsel.AsTree().IndexInParent()

	ng := svg.NewGroup()
	fsel.AsTree().Parent.AsTree().InsertChild(ng, fidx)
	sv.SetSVGName(ng)

	for _, se := range sl {
		tree.MoveToParent(se, ng)
	}

	es.ResetSelected()
	es.Select(ng)

	gv.UpdateAll()
	gv.ChangeMade()
}

// SelectUnGroup ungroups items from each other
func (gv *Canvas) SelectUnGroup() { //types:add
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
		np := gp.Parent
		klist := make([]tree.Node, len(gp.Children)) // make a temp copy of list of kids
		for i, k := range gp.Children {
			klist[i] = k
		}
		for _, k := range klist {
			tree.MoveToParent(k, np)
			se := k.(svg.Node)
			if !gp.Paint.Transform.IsIdentity() {
				se.ApplyTransform(sv.SVG, gp.Paint.Transform) // group no longer there!
			}
		}
	}
	gv.UpdateAll()
	gv.ChangeMade()
}

func (gv *Canvas) SelectRotate(deg float32) {
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sv := gv.SVG()
	sv.UndoSave("Rotate", fmt.Sprintf("%g", deg))

	svoff := sv.Geom.ContentBBox.Min
	del := math32.Vector2{}
	sc := math32.Vec2(1, 1)
	rot := math32.DegToRad(deg)
	for sn := range es.Selected {
		sng := sn.AsNodeBase()
		sz := math32.FromPoint(sng.BBox.Size())
		mn := math32.FromPoint(sng.BBox.Min.Sub(svoff))
		ctr := mn.Add(sz.MulScalar(.5))
		sn.ApplyDeltaTransform(sv.SVG, del, sc, rot, ctr)
	}
	sv.UpdateView(true)
	gv.ChangeMade()
}

func (gv *Canvas) SelectScale(scx, scy float32) {
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sv := gv.SVG()
	sv.UndoSave("Scale", fmt.Sprintf("%g,%g", scx, scy))

	svoff := sv.Geom.ContentBBox.Min
	del := math32.Vector2{}
	sc := math32.Vec2(scx, scy)
	for sn := range es.Selected {
		sng := sn.AsNodeBase()
		sz := math32.FromPoint(sng.BBox.Size())
		mn := math32.FromPoint(sng.BBox.Min.Sub(svoff))
		ctr := mn.Add(sz.MulScalar(.5))
		sn.ApplyDeltaTransform(sv.SVG, del, sc, 0, ctr)
	}
	sv.UpdateView(true)
	gv.ChangeMade()
}

// SelectRotateLeft rotates the selection 90 degrees counter-clockwise
func (gv *Canvas) SelectRotateLeft() { //types:add
	gv.SelectRotate(-90)
}

// SelectRotateRight rotates the selection 90 degrees clockwise
func (gv *Canvas) SelectRotateRight() { //types:add
	gv.SelectRotate(90)
}

// SelectFlipHorizontal flips the selection horizontally
func (gv *Canvas) SelectFlipHorizontal() { //types:add
	gv.SelectScale(-1, 1)
}

// SelectFlipVertical flips the selection vertically
func (gv *Canvas) SelectFlipVertical() { //types:add
	gv.SelectScale(1, -1)
}

// SelectRaiseTop raises the selection to the top of the layer
func (gv *Canvas) SelectRaiseTop() { //types:add
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sv := gv.SVG()
	sv.UndoSave("RaiseTop", es.SelectedNamesString())

	sl := es.SelectedList(true) // true = descending = reverse order
	for _, se := range sl {
		parent := se.AsTree().Parent
		if !(NodeIsLayer(parent) || parent == sv.This) {
			continue
		}
		ci := se.AsTree().IndexInParent()
		pt := parent.AsTree()
		pt.Children = slicesx.Move(pt.Children, ci, len(pt.Children)-1)
	}
	gv.UpdateDisp()
	gv.ChangeMade()
}

// SelectRaise raises the selection by one level in the layer
func (gv *Canvas) SelectRaise() { //types:add
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sv := gv.SVG()
	sv.UndoSave("Raise", es.SelectedNamesString())

	sl := es.SelectedList(true) // true = descending = reverse order
	for _, se := range sl {
		parent := se.AsTree().Parent
		if !(NodeIsLayer(parent) || parent == sv.This) {
			continue
		}
		ci := se.AsTree().IndexInParent()
		if ci < parent.AsTree().NumChildren()-1 {
			pt := parent.AsTree()
			pt.Children = slicesx.Move(pt.Children, ci, ci+1)
		}
	}
	gv.UpdateDisp()
	gv.ChangeMade()
}

// SelectLowerBottom lowers the selection to the bottom of the layer
func (gv *Canvas) SelectLowerBottom() { //types:add
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sv := gv.SVG()
	sv.UndoSave("LowerBottom", es.SelectedNamesString())

	sl := es.SelectedList(true) // true = descending = reverse order
	for _, se := range sl {
		parent := se.AsTree().Parent
		if !(NodeIsLayer(parent) || parent == sv.This) {
			continue
		}
		ci := se.AsTree().IndexInParent()
		pt := parent.AsTree()
		pt.Children = slicesx.Move(pt.Children, ci, 0)
	}
	gv.UpdateDisp()
	gv.ChangeMade()
}

// SelectLower lowers the selection by one level in the layer
func (gv *Canvas) SelectLower() { //types:add
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sv := gv.SVG()
	sv.UndoSave("Lower", es.SelectedNamesString())

	sl := es.SelectedList(true) // true = descending = reverse order
	for _, se := range sl {
		parent := se.AsTree().Parent
		if !(NodeIsLayer(parent) || parent == sv.This) {
			continue
		}
		ci := se.AsTree().IndexInParent()
		if ci > 0 {
			pt := parent.AsTree()
			pt.Children = slicesx.Move(pt.Children, ci, ci-1)
		}
	}
	gv.UpdateDisp()
	gv.ChangeMade()
}

func (gv *Canvas) SelectSetXPos(xp float32) {
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sv := gv.SVG()
	sv.UndoSave("MoveToX", fmt.Sprintf("%g", xp))
	// todo
	gv.ChangeMade()
}

func (gv *Canvas) SelectSetYPos(yp float32) {
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sv := gv.SVG()
	sv.UndoSave("MoveToY", fmt.Sprintf("%g", yp))
	// todo
	gv.ChangeMade()
}

func (gv *Canvas) SelectSetWidth(wd float32) {
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sv := gv.SVG()
	sv.UndoSave("SetWidth", fmt.Sprintf("%g", wd))
	// todo
	gv.ChangeMade()
}

func (gv *Canvas) SelectSetHeight(ht float32) {
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
func (sv *SVG) SelectWithinBBox(bbox image.Rectangle, leavesOnly bool) []svg.Node {
	var rval []svg.Node
	var curlay tree.Node
	svg.SVGWalkDownNoDefs(sv.Root(), func(n svg.Node, nb *svg.NodeBase) bool {
		if n == sv.Root().This {
			return tree.Continue
		}
		if leavesOnly && nb.HasChildren() {
			return tree.Continue
		}
		if NodeIsLayer(n) {
			return tree.Continue
		}
		if txt, istxt := n.(*svg.Text); istxt { // no tspans
			if txt.Text != "" {
				if _, istxt := txt.Parent.(*svg.Text); istxt {
					return tree.Break
				}
			}
		}
		if nb.Paint.Off {
			return tree.Break
		}
		nl := NodeParentLayer(n)
		if nl != nil {
			if (curlay != nil && nl != curlay) || LayerIsLocked(nl) || !LayerIsVisible(nl) {
				return tree.Break
			}
		}
		if nb.BBox.In(bbox) {
			rval = append(rval, n)
			if curlay == nil && nl != nil {
				curlay = nl
			}
			return tree.Break // don't go into groups!
		}
		return tree.Continue
	})
	return rval
}

// SelectContainsPoint finds the first node whose BBox contains the given
// point in scene coordinates; nil if none.  If leavesOnly is set then only nodes that have no
// nodes (leaves, terminal nodes) will be considered.
// if leavesOnly, only terminal leaves (no children) are included
// if excludeSel, any leaf nodes that are within the current edit selection are
// excluded,
func (sv *SVG) SelectContainsPoint(pt image.Point, leavesOnly, excludeSel bool) svg.Node {
	pt = pt.Sub(sv.Geom.ContentBBox.Min)
	es := sv.EditState()
	var curlay tree.Node
	fn := es.FirstSelectedNode()
	if fn != nil {
		curlay = NodeParentLayer(fn)
	}
	var rval svg.Node
	svg.SVGWalkDownNoDefs(sv.Root(), func(n svg.Node, nb *svg.NodeBase) bool {
		if n == sv.Root().This {
			return tree.Continue
		}
		if leavesOnly && nb.HasChildren() {
			return tree.Continue
		}
		if NodeIsLayer(n) {
			return tree.Continue
		}
		if txt, istxt := n.(*svg.Text); istxt { // no tspans
			if txt.Text != "" {
				if _, istxt := txt.Parent.(*svg.Text); istxt {
					return tree.Break
				}
			}
		}
		if excludeSel {
			if _, issel := es.Selected[n]; issel {
				return tree.Continue
			}
			if _, issel := es.RecentlySelected[n]; issel {
				return tree.Continue
			}
		}
		if nb.Paint.Off {
			return tree.Break
		}
		nl := NodeParentLayer(n)
		if nl != nil {
			if (curlay != nil && nl != curlay) || LayerIsLocked(nl) || !LayerIsVisible(nl) {
				return tree.Break
			}
		}
		if pt.In(nb.BBox) {
			rval = n
			return tree.Break
		}
		return tree.Continue
	})
	return rval
}
