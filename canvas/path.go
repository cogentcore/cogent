// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package canvas

import (
	"fmt"
	"image"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/ppath"
	"cogentcore.org/core/svg"
	"cogentcore.org/core/tree"
)

func (vv *Canvas) MakeNodeToolbar(p *tree.Plan) {
	tree.Add(p, func(w *core.Switch) {
		core.Bind(&Settings.SnapNodes, w)
		w.SetText("Snap nodes").SetTooltip("Snap movement and sizing of nodes, using overall snap settings")
	})
	tree.Add(p, func(w *core.Separator) {})

	// tb.AddAction(core.ActOpts{Icon: "sel-group", Tooltip: "Ctrl+G: Group items together", UpdateFunc: gv.NodeEnableFunc},
	// 	gv.This, func(recv, send tree.Node, sig int64, data interface{}) {
	// 		grr := recv.Embed(KiT_Vector).(*Vector)
	// 		grr.SelGroup()
	// 	})
	//
	// core.NewSeparator(tb, "sep-group")

	tree.Add(p, func(w *core.Text) {
		w.SetText("X: ")
	})
	// px := core.NewSpinner(tb).SetStep(1).SetValue(0).
	// 	SetTooltip("horizontal coordinate of node, in document units")
	// px.OnChange(func(e events.Event) {
	// 	// vv.NodeSetXPos(px.Value)
	// })

	tree.Add(p, func(w *core.Text) {
		w.SetText("Y: ")
	})
	// py := core.NewSpinner(tb).SetStep(1).SetValue(0).
	// 	SetTooltip("vertical coordinate of node, in document units")
	// py.OnChange(func(e events.Event) {
	// 	// vv.NodeSetYPos(py.Value)
	// })
}

// NodeEnableFunc is an ActionUpdateFunc that inactivates action if no node selected
func (vv *Canvas) NodeEnableFunc(act *core.Button) {
	// es := &gv.EditState
	// act.SetInactiveState(!es.HasNodeed())
}

// UpdateNodeToolbar updates the node toolbar based on current nodeion
func (vv *Canvas) UpdateNodeToolbar() {
	// tb := vv.NodeToolbar()
	// es := &vv.EditState
	// if es.Tool != NodeTool {
	// 	return
	// }
	// px := tb.ChildByName("posx", 8).(*core.Spinner)
	// px.SetValue(es.DragSelectCurrentBBox.Min.X)
	// py := tb.ChildByName("posy", 9).(*core.Spinner)
	// py.SetValue(es.DragSelectCurrentBBox.Min.Y)
}

///////////////////////////////////////////////////////////////////////
//   Actions

func (vv *Canvas) NodeSetXPos(xp float32) {
	es := &vv.EditState
	if !es.HasSelected() {
		return
	}
	sv := vv.SVG
	sv.UndoSave("NodeToX", fmt.Sprintf("%g", xp))
	// todo
	vv.ChangeMade()
}

func (vv *Canvas) NodeSetYPos(yp float32) {
	es := &vv.EditState
	if !es.HasSelected() {
		return
	}
	sv := vv.SVG
	sv.UndoSave("NodeToY", fmt.Sprintf("%g", yp))
	// todo
	vv.ChangeMade()
}

////////  PathNode

// PathNode is info about each node in a path that is being edited
type PathNode struct {
	// path command
	Cmd Sprites

	// index of start of command in path
	Index int

	// logical index of point within current command (0 = first point, etc)
	PtIndex int

	// original data points:
	Start, End, Cp1, Cp2 math32.Vector2

	// transformed to scene coordinates:
	TStart, TEnd, TCp1, TCp2 math32.Vector2
}

// PathNodes returns the PathNode data for given path data, and a list of indexes where commands start
func (sv *SVG) PathNodes(path *svg.Path) ([]*PathNode, []int) {
	xf := path.ParentTransform(true) // include self
	pts := path.Data

	nc := make([]*PathNode, 0)
	cidxs := make([]int, 0)
	pidx := 0

	var pn *PathNode
	pti := 0
	for scanner := pts.Scanner(); scanner.Scan(); {
		end := scanner.End()
		start := scanner.Start()
		tend := xf.MulVector2AsPoint(end)
		tstart := xf.MulVector2AsPoint(end)

		pn = &PathNode{Index: pidx, PtIndex: pti, Start: start, End: end, TStart: tstart, TEnd: tend}
		cidxs = append(cidxs, pidx)
		pidx = scanner.Index() + 1
		nc = append(nc, pn)
		pti++

		switch scanner.Cmd() {
		case ppath.MoveTo:
			pn.Cmd = SpMoveTo
		case ppath.LineTo:
			pn.Cmd = SpLineTo
		case ppath.QuadTo:
			pn.Cmd = SpQuadTo
			pn.Cp1 = scanner.CP1()
			pn.TCp1 = xf.MulVector2AsPoint(pn.Cp1)
		case ppath.CubeTo:
			pn.Cmd = SpCubeTo
			pn.Cp1 = scanner.CP1()
			pn.Cp2 = scanner.CP2()
			pn.TCp1 = xf.MulVector2AsPoint(pn.Cp1)
			pn.TCp2 = xf.MulVector2AsPoint(pn.Cp2)
		case ppath.Close:
			pn.Cmd = SpClose
		}
	}
	return nc, cidxs
}

func (sv *SVG) UpdateNodeSprites(path *svg.Path) {
	es := sv.EditState()
	prvn := es.NNodeSprites

	if path == nil {
		sv.RemoveNodeSprites()
		sv.UpdateView()
		return
	}

	sprites := sv.SpritesLock()

	es.PathNodes, es.PathCommands = sv.PathNodes(path)
	es.NNodeSprites = len(es.PathNodes)
	es.ActivePath = path

	for i, pn := range es.PathNodes {
		sp := sv.Sprite(SpNodePoint, pn.Cmd, i, image.Point{}, func(sp *core.Sprite) {
			sp.OnClick(func(e events.Event) {
				es.NodeSelectAction(i, e.SelectMode())
				sv.NeedsRender()
				e.SetHandled()
			})
			sp.OnSlideStart(func(e events.Event) {
				// if !es.NodeIsSelected(i) {
				// 	es.SelectNode(i)
				// }
				es.DragNodeStart(e.Pos())
				sv.NeedsRender()
				e.SetHandled()
			})
			sp.OnSlideMove(func(e events.Event) {
				// if e.HasAnyModifier(key.Alt) {
				// 	sv.SpriteRotateDrag(i, e.PrevDelta())
				// } else {
				sv.SpriteNodeDrag(i, e)
				e.SetHandled()
			})
			sp.OnSlideStop(func(e events.Event) {
				sv.ManipDone()
				e.SetHandled()
			})
		})
		SetSpritePos(sp, int(pn.TEnd.X), int(pn.TEnd.Y))
	}

	// remove extra
	for i := es.NNodeSprites; i < prvn; i++ {
		spnm := SpriteName(SpNodePoint, SpUnknown, i)
		sprites.InactivateSpriteLocked(spnm)
	}
	sprites.Unlock()

	sv.Canvas.UpdateNodeToolbar()
	sv.UpdateView()
}

func (sv *SVG) RemoveNodeSprites() {
	es := sv.EditState()
	sprites := sv.SpritesLock()
	for i := 0; i < es.NNodeSprites; i++ {
		spnm := SpriteName(SpNodePoint, SpUnknown, i)
		sprites.InactivateSpriteLocked(spnm)
	}
	es.NNodeSprites = 0
	es.PathNodes = nil
	es.PathCommands = nil
	es.ActivePath = nil
	sprites.Unlock()
}

// PathNodeSetOnePoint moves given node index by given delta in window coords
// and all following points up to cmd = z or m are moved in the opposite
// direction to compensate, so only the one point is moved in effect.
// svoff is the window starting vector coordinate for view.
func (sv *SVG) PathNodeSetOnePoint(path *svg.Path, pts []*PathNode, pidx int, dxf math32.Matrix2) {
	pn := pts[pidx]
	end := pn.End
	end = dxf.MulVector2AsPoint(end)
	switch pn.Cmd {
	case SpMoveTo, SpLineTo, SpClose:
		path.Data[pn.Index+1] = end.X
		path.Data[pn.Index+2] = end.Y
	case SpQuadTo:
		path.Data[pn.Index+3] = end.X
		path.Data[pn.Index+4] = end.Y
	case SpCubeTo:
		path.Data[pn.Index+5] = end.X
		path.Data[pn.Index+6] = end.Y
		// todo: arc
	}
}

// SpriteNodeDrag processes a mouse node drag event on a path node sprite
func (sv *SVG) SpriteNodeDrag(idx int, e events.Event) {
	es := sv.EditState()
	if !es.InAction() {
		sv.ManipStart(NodeMove, es.ActivePath.Name)
		sv.GatherAlignPoints()
	}
	sprites := sv.SpritesLock()
	InactivateSprites(sprites, SpAlignMatch)

	spt := math32.FromPoint(es.DragStartPos)
	mpt := math32.FromPoint(e.Pos())

	if e.HasAnyModifier(key.Control) {
		mpt, _ = sv.ConstrainPoint(spt, mpt)
	}
	if Settings.SnapNodes {
		mpt = sv.SnapPoint(mpt)
	}

	es.DragCurPos = mpt.ToPoint()
	mdel := es.DragCurPos.Sub(es.DragStartPos)
	dv := math32.FromPoint(mdel)

	dxf := es.ActivePath.DeltaTransform(dv, math32.Vector2{1, 1}, 0, spt)

	for i, _ := range es.NodeSelect {
		sv.PathNodeSetOnePoint(es.ActivePath, es.PathNodes, i, dxf)
		pn := es.PathNodes[i]
		nwc := pn.TEnd.Add(dv)
		spnm := SpriteName(SpNodePoint, SpUnknown, i)
		sp, _ := sprites.SpriteByNameLocked(spnm)
		SetSpritePos(sp, int(nwc.X), int(nwc.Y))
	}

	sprites.Unlock()
	sv.UpdateView()
}
