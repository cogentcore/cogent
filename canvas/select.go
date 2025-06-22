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
	"cogentcore.org/core/paint/ppath/intersect"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/svg"
	"cogentcore.org/core/tree"
)

// selectEnabledStyler sets the given widget to only be enabled when
// there is an item selected.
func (cv *Canvas) selectEnabledStyler(w core.Widget) {
	w.AsWidget().FirstStyler(func(s *styles.Style) {
		s.SetEnabled(cv.EditState.HasSelected())
	})
}

// MakeSelectToolbar adds the select toolbar to the given plan.
func (cv *Canvas) MakeSelectToolbar(p *tree.Plan) {
	if cv.EditState.Tool == TextTool || cv.EditState.SelectIsText {
		cv.MakeTextToolbar(p)
		tree.Add(p, func(w *core.Separator) {})
	}

	tree.Add(p, func(w *core.Switch) {
		core.Bind(&Settings.SnapAlign, w)
		w.SetText("Snap align")
		w.SetTooltip("Snap to align with other elements in the scene, as indicated")
	})
	tree.Add(p, func(w *core.Switch) {
		core.Bind(&Settings.SnapGrid, w)
		w.SetText("Snap grid")
		w.SetTooltip("Snap to the grid (edit grid in settings)")
	})
	tree.Add(p, func(w *core.Spinner) {
		if cv.SVG != nil {
			core.Bind(&cv.SVG.Grid, w)
		}
		w.Min = 0.01
		w.Step = 1
		w.Styler(func(s *styles.Style) {
			s.Max.X.Ch(5)
		})
		w.Updater(func() {
			if cv.SVG == nil {
				return
			}
			if w.ValueUpdate == nil {
				core.Bind(&cv.SVG.Grid, w) // was nil before
			}
		})
		w.OnChange(func(e events.Event) {
			cv.SVG.UpdateView()
		})
		w.SetTooltip("Grid spacing in the ViewBox units of the drawing. Saved if metadata is on.")
	})
	tree.Add(p, func(w *core.Separator) {})
	tree.Add(p, func(w *core.FuncButton) {
		cv.selectEnabledStyler(w)
		w.SetFunc(cv.SelectGroup).SetText("Group").SetIcon(cicons.SelGroup).SetShortcut("Command+G")
	})
	tree.Add(p, func(w *core.FuncButton) {
		cv.selectEnabledStyler(w)
		w.SetFunc(cv.SelectUnGroup).SetText("Ungroup").SetIcon(cicons.SelUngroup).SetShortcut("Command+Shift+G")
	})
	tree.Add(p, func(w *core.Separator) {})
	tree.Add(p, func(w *core.FuncButton) {
		cv.selectEnabledStyler(w)
		w.SetFunc(cv.SelectRotateLeft).SetText("").SetIcon(cicons.SelRotateLeft).SetShortcut("Command+[")
	})
	tree.Add(p, func(w *core.FuncButton) {
		cv.selectEnabledStyler(w)
		w.SetFunc(cv.SelectRotateRight).SetText("").SetIcon(cicons.SelRotateRight).SetShortcut("Command+]")
	})
	tree.Add(p, func(w *core.FuncButton) {
		cv.selectEnabledStyler(w)
		w.SetFunc(cv.SelectFlipHorizontal).SetText("").SetIcon(cicons.SelFlipHoriz)
	})
	tree.Add(p, func(w *core.FuncButton) {
		cv.selectEnabledStyler(w)
		w.SetFunc(cv.SelectFlipVertical).SetText("").SetIcon(cicons.SelFlipVert)
	})
	tree.Add(p, func(w *core.Separator) {})
	tree.Add(p, func(w *core.FuncButton) {
		cv.selectEnabledStyler(w)
		w.SetFunc(cv.SelectRaiseTop).SetText("").SetIcon(cicons.SelRaiseTop)
	})
	tree.Add(p, func(w *core.FuncButton) {
		cv.selectEnabledStyler(w)
		w.SetFunc(cv.SelectRaise).SetText("").SetIcon(cicons.SelRaise)
	})
	tree.Add(p, func(w *core.FuncButton) {
		cv.selectEnabledStyler(w)
		w.SetFunc(cv.SelectLower).SetText("").SetIcon(cicons.SelLower)
	})
	tree.Add(p, func(w *core.FuncButton) {
		cv.selectEnabledStyler(w)
		w.SetFunc(cv.SelectLowerBottom).SetText("").SetIcon(cicons.SelLowerBottom)
	})
	tree.Add(p, func(w *core.Separator) {})
	// tree.Add(p, func(w *core.Text) {
	// 	w.SetText("X: ")
	// })
	// TODO(config):
	// core.NewValue(tb, &gv.EditState.DragSnapBBox.Min.X).SetDoc("Horizontal coordinate of selection, in document units").OnChange(func(e events.Event) {
	// 	gv.SelectSetXPos(gv.EditState.DragSnapBBox.Min.X)
	// })

	// tree.Add(p, func(w *core.Text) {
	// 	w.SetText("Y: ")
	// })
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

// UpdateSelect should be called whenever selection changes
func (sv *SVG) UpdateSelect() {
	es := sv.EditState()
	sv.Canvas.UpdateTabs()
	sv.Canvas.UpdateModalToolbar()
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
	sv.NeedsRender()
	sprites := sv.SpritesLock()
	sv.InactivateSprites(SpReshapeBBox)
	sv.InactivateSprites(SpSelBBox)
	es := sv.EditState()
	es.NSelectSprites = 0
	sprites.Unlock()
}

func (sv *SVG) UpdateSelSprites() {
	sv.NeedsRender()
	es := sv.EditState()
	es.UpdateSelectBBox()
	if !es.HasSelected() {
		sv.RemoveSelSprites()
		return
	}
	sprites := sv.SpritesLock()
	sv.setSelSpritePos()
	for i := SpUpL; i <= SpRtM; i++ {
		sv.Sprite(SpReshapeBBox, i, 0, image.Point{}, func(sp *core.Sprite) {
			sp.OnSlideStart(func(e events.Event) {
				es.DragSelStart(e.Pos())
				e.SetHandled()
			})
			sp.OnSlideMove(func(e events.Event) {
				if e.HasAnyModifier(key.Alt) {
					sv.SpriteRotateDrag(i, e)
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
	sprites.Unlock()
}

// setSelSpritePos sets the selection sprites positions.
// only called by SetBBoxSpritePos.
func (sv *SVG) setSelSpritePos() {
	es := sv.EditState()
	nsel := es.NSelectSprites

	es.NSelectSprites = 0
	if len(es.Selected) > 1 {
		nbox := 0
		sl := es.SelectedList(false)
		for si, sii := range sl {
			sn := sii.AsNodeBase()
			if sn.BBox.Size() == (math32.Vector2{}) {
				continue
			}
			bb := sn.BBox
			sv.SetBBoxSpritePos(SpSelBBox, si, bb)
			nbox++
		}
		es.NSelectSprites = nbox
		return
	}

	sprites := sv.SpritesNoLock()
	for si := es.NSelectSprites; si < nsel; si++ {
		for i := SpUpL; i <= SpRtM; i++ {
			spnm := SpriteName(SpSelBBox, i, si)
			sprites.InactivateSpriteNoLock(spnm)
		}
	}
}

// SetBBoxSpritePos sets positions of given type of sprites.
func (sv *SVG) SetBBoxSpritePos(typ Sprites, idx int, bbox math32.Box2) {
	spbb, _ := sv.HandleSpriteSize(1, image.Point{})
	midX := int(0.5 * (bbox.Min.X + bbox.Max.X - float32(spbb.Dx())))
	midY := int(0.5 * (bbox.Min.Y + bbox.Max.Y - float32(spbb.Dy())))
	bbi := bbox.ToRect()
	for i := SpUpL; i <= SpRtM; i++ {
		sp := sv.Sprite(typ, i, idx, image.ZP, nil)
		switch i {
		case SpUpL:
			sv.SetSpritePos(sp, bbi.Min.X, bbi.Min.Y)
		case SpUpC:
			sv.SetSpritePos(sp, midX, bbi.Min.Y)
		case SpUpR:
			sv.SetSpritePos(sp, bbi.Max.X, bbi.Min.Y)
		case SpDnL:
			sv.SetSpritePos(sp, bbi.Min.X, bbi.Max.Y)
		case SpDnC:
			sv.SetSpritePos(sp, midX, bbi.Max.Y)
		case SpDnR:
			sv.SetSpritePos(sp, bbi.Max.X, bbi.Max.Y)
		case SpLfM:
			sv.SetSpritePos(sp, bbi.Min.X, midY)
		case SpRtM:
			sv.SetSpritePos(sp, bbi.Max.X, midY)
		}
	}
}

// SetRubberBand updates the rubber band position.
func (sv *SVG) SetRubberBand(cur image.Point) {
	es := sv.EditState()
	sprites := sv.SpritesLock()

	if !es.InAction() {
		es.ActStart(BoxSelect, fmt.Sprintf("%v", es.DragStartPos))
		es.ActUnlock()
	}
	es.DragPos = cur

	bbox := image.Rectangle{Min: es.DragStartPos, Max: es.DragPos}
	bbox = bbox.Canon()

	sz := bbox.Size()
	if sz.X < 4 {
		sz.X = 4
	}
	if sz.Y < 4 {
		sz.Y = 4
	}
	sp := sv.Sprite(SpRubberBand, SpNone, 0, sz, nil)
	sp.Properties["size"] = sz
	sv.SetSpritePos(sp, bbox.Min.X, bbox.Min.Y)

	sprites.Unlock()
	sv.NeedsRender()
}

////////   Actions

// SelectGroup groups items together
func (cv *Canvas) SelectGroup() { //types:add
	es := &cv.EditState
	if !es.HasSelected() {
		return
	}
	sv := cv.SVG
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

	cv.UpdateAll()
	cv.ChangeMade()
}

// SelectUnGroup ungroups items from each other
func (cv *Canvas) SelectUnGroup() { //types:add
	es := &cv.EditState
	if !es.HasSelected() {
		return
	}
	sv := cv.SVG
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
	sv.RemoveEmptyGroups()
	cv.UpdateAll()
	cv.ChangeMade()
}

func (cv *Canvas) SelectRotate(deg float32) {
	es := &cv.EditState
	if !es.HasSelected() {
		return
	}
	sv := cv.SVG
	sv.UndoSave("Rotate", fmt.Sprintf("%g", deg))

	del := math32.Vector2{}
	sc := math32.Vec2(1, 1)
	rot := math32.DegToRad(deg)
	for sn := range es.Selected {
		sng := sn.AsNodeBase()
		sz := sng.BBox.Size()
		ctr := sng.BBox.Min.Add(sz.MulScalar(.5))
		sn.ApplyTransform(sv.SVG, sng.DeltaTransform(del, sc, rot, ctr))
	}
	sv.UpdateView()
	cv.ChangeMade()
}

func (cv *Canvas) SelectScale(scx, scy float32) {
	es := &cv.EditState
	if !es.HasSelected() {
		return
	}
	sv := cv.SVG
	sv.UndoSave("Scale", fmt.Sprintf("%g,%g", scx, scy))

	del := math32.Vector2{}
	sc := math32.Vec2(scx, scy)
	for sn := range es.Selected {
		sng := sn.AsNodeBase()
		sz := sng.BBox.Size()
		ctr := sng.BBox.Min.Add(sz.MulScalar(.5))
		sn.ApplyTransform(sv.SVG, sng.DeltaTransform(del, sc, 0, ctr))
	}
	sv.UpdateView()
	cv.ChangeMade()
}

// SelectRotateLeft rotates the selection 90 degrees counter-clockwise
func (cv *Canvas) SelectRotateLeft() { //types:add
	cv.SelectRotate(-90)
}

// SelectRotateRight rotates the selection 90 degrees clockwise
func (cv *Canvas) SelectRotateRight() { //types:add
	cv.SelectRotate(90)
}

// SelectFlipHorizontal flips the selection horizontally
func (cv *Canvas) SelectFlipHorizontal() { //types:add
	cv.SelectScale(-1, 1)
}

// SelectFlipVertical flips the selection vertically
func (cv *Canvas) SelectFlipVertical() { //types:add
	cv.SelectScale(1, -1)
}

// SelectRaiseTop raises the selection to the top of the layer
func (cv *Canvas) SelectRaiseTop() { //types:add
	es := &cv.EditState
	if !es.HasSelected() {
		return
	}
	sv := cv.SVG
	sv.UndoSave("RaiseTop", es.SelectedNamesString())

	sl := es.SelectedList(true) // true = descending = reverse order
	for _, se := range sl {
		parent := se.AsTree().Parent
		ci := se.AsTree().IndexInParent()
		pt := parent.AsTree()
		pt.Children = slicesx.Move(pt.Children, ci, len(pt.Children)-1)
	}
	cv.UpdateSVG()
	cv.UpdateTree()
	cv.ChangeMade()
}

// SelectRaise raises the selection by one level in the layer
func (cv *Canvas) SelectRaise() { //types:add
	es := &cv.EditState
	if !es.HasSelected() {
		return
	}
	sv := cv.SVG
	sv.UndoSave("Raise", es.SelectedNamesString())

	sl := es.SelectedList(true) // true = descending = reverse order
	for _, se := range sl {
		parent := se.AsTree().Parent
		ci := se.AsTree().IndexInParent()
		if ci < parent.AsTree().NumChildren()-1 {
			pt := parent.AsTree()
			pt.Children = slicesx.Move(pt.Children, ci, ci+1)
		}
	}
	cv.UpdateSVG()
	cv.UpdateTree()
	cv.ChangeMade()
}

// SelectLowerBottom lowers the selection to the bottom of the layer
func (cv *Canvas) SelectLowerBottom() { //types:add
	es := &cv.EditState
	if !es.HasSelected() {
		return
	}
	sv := cv.SVG
	sv.UndoSave("LowerBottom", es.SelectedNamesString())

	sl := es.SelectedList(true) // true = descending = reverse order
	for _, se := range sl {
		parent := se.AsTree().Parent
		ci := se.AsTree().IndexInParent()
		pt := parent.AsTree()
		pt.Children = slicesx.Move(pt.Children, ci, 0)
	}
	cv.UpdateSVG()
	cv.UpdateTree()
	cv.ChangeMade()
}

// SelectLower lowers the selection by one level in the layer
func (cv *Canvas) SelectLower() { //types:add
	es := &cv.EditState
	if !es.HasSelected() {
		return
	}
	sv := cv.SVG
	sv.UndoSave("Lower", es.SelectedNamesString())

	sl := es.SelectedList(true) // true = descending = reverse order
	for _, se := range sl {
		parent := se.AsTree().Parent
		ci := se.AsTree().IndexInParent()
		if ci > 0 {
			pt := parent.AsTree()
			pt.Children = slicesx.Move(pt.Children, ci, ci-1)
		}
	}
	cv.UpdateSVG()
	cv.UpdateTree()
	cv.ChangeMade()
}

func (cv *Canvas) SelectSetXPos(xp float32) {
	es := &cv.EditState
	if !es.HasSelected() {
		return
	}
	sv := cv.SVG
	sv.UndoSave("MoveToX", fmt.Sprintf("%g", xp))
	// todo
	cv.ChangeMade()
}

func (cv *Canvas) SelectSetYPos(yp float32) {
	es := &cv.EditState
	if !es.HasSelected() {
		return
	}
	sv := cv.SVG
	sv.UndoSave("MoveToY", fmt.Sprintf("%g", yp))
	// todo
	cv.ChangeMade()
}

func (cv *Canvas) SelectSetWidth(wd float32) {
	es := &cv.EditState
	if !es.HasSelected() {
		return
	}
	sv := cv.SVG
	sv.UndoSave("SetWidth", fmt.Sprintf("%g", wd))
	// todo
	cv.ChangeMade()
}

func (cv *Canvas) SelectSetHeight(ht float32) {
	es := &cv.EditState
	if !es.HasSelected() {
		return
	}
	sv := cv.SVG
	sv.UndoSave("SetHeight", fmt.Sprintf("%g", ht))
	// todo
	cv.ChangeMade()
}

////////   Select tree traversal

// SelectWithinBBox returns a list of all nodes whose BBox is fully contained
// within the given BBox. SVG version excludes layer groups.
func (sv *SVG) SelectWithinBBox(bbox math32.Box2, leavesOnly bool) []svg.Node {
	var rval []svg.Node
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
		nl := NodeParentLayer(n)
		if nl != nil {
			if LayerIsLocked(nl) || !LayerIsVisible(nl) {
				return tree.Break
			}
		}
		if bbox.ContainsBox(nb.BBox) {
			rval = append(rval, n)
			return tree.Break // don't go into groups!
		}
		return tree.Continue
	})
	return rval
}

// SelectContainsPoint finds the first node that contains the given
// point in scene coordinates; nil if none. If leavesOnly is set then only nodes
// that have no nodes (leaves, terminal nodes) will be considered.
// If excludeSel, any leaf nodes that are within the current edit selection are
// excluded,
func (sv *SVG) SelectContainsPoint(pt image.Point, leavesOnly, excludeSel bool) svg.Node {
	ptv := math32.FromPoint(pt)
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
		nl := NodeParentLayer(n)
		if nl != nil {
			if (curlay != nil && nl != curlay) || LayerIsLocked(nl) || !LayerIsVisible(nl) {
				return tree.Break
			}
		}
		if !nb.BBox.ContainsPoint(ptv) {
			return tree.Continue
		}
		sz := nb.BBox.Size()
		p, isPath := n.(*svg.Path)
		if !isPath || sz.X < 12 || sz.Y < 12 { // small bbox footprint paths are too hard to hit
			rval = n
			return tree.Break
		}
		xf := p.ParentTransform(true).Inverse()
		pxf := xf.MulVector2AsPoint(ptv)
		if intersect.Contains(p.Data, pxf.X, pxf.Y, p.Paint.Fill.Rule) {
			rval = n
			return tree.Break
		}
		return tree.Continue
	})
	return rval
}
