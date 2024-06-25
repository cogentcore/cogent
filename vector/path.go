// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vector

import (
	"fmt"
	"image"

	"cogentcore.org/core/core"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/svg"
	"cogentcore.org/core/tree"
)

func (vv *Vector) MakeNodeToolbar(p *tree.Plan) {
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
func (vv *Vector) NodeEnableFunc(act *core.Button) {
	// es := &gv.EditState
	// act.SetInactiveState(!es.HasNodeed())
}

// UpdateNodeToolbar updates the node toolbar based on current nodeion
func (vv *Vector) UpdateNodeToolbar() {
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

func (vv *Vector) NodeSetXPos(xp float32) {
	es := &vv.EditState
	if !es.HasSelected() {
		return
	}
	sv := vv.SVG()
	sv.UndoSave("NodeToX", fmt.Sprintf("%g", xp))
	// todo
	vv.ChangeMade()
}

func (vv *Vector) NodeSetYPos(yp float32) {
	es := &vv.EditState
	if !es.HasSelected() {
		return
	}
	sv := vv.SVG()
	sv.UndoSave("NodeToY", fmt.Sprintf("%g", yp))
	// todo
	vv.ChangeMade()
}

//////////////////////////////////////////////////////////////////////////
//  PathNode

// PathNode is info about each node in a path that is being edited
type PathNode struct {

	// path command
	Cmd svg.PathCmds

	// previous path command
	PrevCmd svg.PathCmds

	// starting index of command
	CmdIndex int

	// index of points in data stream
	Index int

	// logical index of point within current command (0 = first point, etc)
	PtIndex int

	// local coords abs previous current point that is starting point for this command
	PCp math32.Vector2

	// local coords abs current point
	Cp math32.Vector2

	// main point coords in window (dot) coords
	WinPt math32.Vector2

	// control point coords in window (dot) coords (nil until manipulated)
	WinCtrls []math32.Vector2
}

// PathNodes returns the PathNode data for given path data, and a list of indexes where commands start
func (sv *SVG) PathNodes(path *svg.Path) ([]*PathNode, []int) {
	svoff := math32.Vector2FromPoint(sv.Geom.ContentBBox.Min)
	pxf := path.ParTransform(true) // include self

	lstCmdIndex := 0
	lstCmd := svg.PcErr
	nc := make([]*PathNode, 0)
	cidxs := make([]int, 0)
	var pcp math32.Vector2
	svg.PathDataIterFunc(path.Data, func(idx int, cmd svg.PathCmds, ptIndex int, cp math32.Vector2, ctrl []math32.Vector2) bool {
		cw := pxf.MulVector2AsPoint(cp).Add(svoff)

		if ptIndex == 0 {
			lstCmdIndex = idx - 1
			cidxs = append(cidxs, lstCmdIndex)
		}
		pn := &PathNode{Cmd: cmd, PrevCmd: lstCmd, CmdIndex: lstCmdIndex, Index: idx, PtIndex: ptIndex, PCp: pcp, Cp: cp, WinPt: cw, WinCtrls: ctrl}
		nc = append(nc, pn)
		pcp = cp
		lstCmd = cmd
		return tree.Continue
	})
	return nc, cidxs
}

func (sv *SVG) UpdateNodeSprites() {
	es := sv.EditState()
	prvn := es.NNodeSprites

	path := es.FirstSelectedPath()

	if path == nil {
		sv.RemoveNodeSprites()
		// win.UpdateSig()
		return
	}

	es.PathNodes, es.PathCommands = sv.PathNodes(path)
	es.NNodeSprites = len(es.PathNodes)
	es.ActivePath = path

	for i, pn := range es.PathNodes {
		// 	sp := SpriteConnectEvent(win, SpNodePoint, SpUnk, i, image.ZP, sv.This, func(recv, send tree.Node, sig int64, d any) {
		// 		ssvg := recv.Embed(KiT_SVG).(*SVG)
		// 		ssvg.NodeSpriteEvent(idx, events.EventType(sig), d)
		// 	})
		sp := Sprite(sv, SpNodePoint, SpUnk, i, image.Point{})
		SetSpritePos(sp, pn.WinPt.ToPoint())
	}

	// remove extra
	sprites := &sv.Scene.Stage.Sprites
	for i := es.NNodeSprites; i < prvn; i++ {
		spnm := SpriteName(SpNodePoint, SpUnk, i)
		sprites.InactivateSprite(spnm)
	}

	sv.Vector.UpdateNodeToolbar()
}

func (sv *SVG) RemoveNodeSprites() {
	es := sv.EditState()
	sprites := &sv.Scene.Stage.Sprites
	for i := 0; i < es.NNodeSprites; i++ {
		spnm := SpriteName(SpNodePoint, SpUnk, i)
		sprites.InactivateSprite(spnm)
	}
	es.NNodeSprites = 0
	es.PathNodes = nil
	es.PathCommands = nil
	es.ActivePath = nil
}

/*
func (sv *SVG) NodeSpriteEvent(idx int, et events.Type, d any) {
	// win := sv.Vector.ParentWindow()
	// es := sv.EditState()
	// es.SelNoDrag = false
	// switch et {
	// case events.MouseEvent:
	// 	me := d.(*mouse.Event)
	// 	me.SetProcessed()
	// 	if me.Action == mouse.Press {
	// 		win.SpriteDragging = SpriteName(SpNodePoint, SpUnk, idx)
	// 		es.DragNodeStart(me.Where)
	// 	} else if me.Action == mouse.Release {
	// 		sv.UpdateNodeSprites()
	// 		sv.ManipDone()
	// 	}
	// case events.MouseDragEvent:
	// 	me := d.(*mouse.DragEvent)
	// 	me.SetProcessed()
	// 	sv.SpriteNodeDrag(idx, win, me)
	// }
}
*/

// PathNodeMoveOnePoint moves given node index by given delta in window coords
// and all following points up to cmd = z or m are moved in the opposite
// direction to compensate, so only the one point is moved in effect.
// svoff is the window starting vector coordinate for view.
func (sv *SVG) PathNodeSetOnePoint(path *svg.Path, pts []*PathNode, pidx int, dv math32.Vector2, svoff math32.Vector2) {
	for i := pidx; i < len(pts); i++ {
		pn := pts[i]
		wbmin := math32.Vector2FromPoint(path.BBox.Min)
		pt := wbmin.Sub(svoff)
		xf, lpt := path.DeltaTransform(dv, math32.Vec2(1, 1), 0, pt, true) // include self
		npt := xf.MulVector2AsPointCenter(pn.Cp, lpt)                      // transform point to new abs coords
		sv.PathNodeSetPoint(path, pn, npt)
		if i == pidx {
			dv = dv.MulScalar(-1)
		} else {
			if !svg.PathCmdIsRel(pn.Cmd) || pn.Cmd == svg.PcZ || pn.Cmd == svg.Pcz || pn.Cmd == svg.Pcm || pn.Cmd == svg.PcM {
				break
			}
		}
	}
}

// PathNodeSetPoint sets data point for path node to given new point value
// which is in *absolute* (but local) coordinates -- translates into
// relative coordinates as needed.
func (sv *SVG) PathNodeSetPoint(path *svg.Path, pn *PathNode, npt math32.Vector2) {
	if pn.Index == 1 || !svg.PathCmdIsRel(pn.Cmd) { // abs
		switch pn.Cmd {
		case svg.PcH:
			path.Data[pn.Index] = svg.PathData(npt.X)
		case svg.PcV:
			path.Data[pn.Index] = svg.PathData(npt.Y)
		default:
			path.Data[pn.Index] = svg.PathData(npt.X)
			path.Data[pn.Index+1] = svg.PathData(npt.Y)
		}
	} else {
		switch pn.Cmd {
		case svg.Pch:
			path.Data[pn.Index] = svg.PathData(npt.X - pn.PCp.X)
		case svg.Pcv:
			path.Data[pn.Index] = svg.PathData(npt.Y - pn.PCp.Y)
		default:
			path.Data[pn.Index] = svg.PathData(npt.X - pn.PCp.X)
			path.Data[pn.Index+1] = svg.PathData(npt.Y - pn.PCp.Y)
		}
	}
}

/*
// SpriteNodeDrag processes a mouse node drag event on a path node sprite
func (sv *SVG) SpriteNodeDrag(idx int, win *core.Window, me *mouse.DragEvent) {
	es := sv.EditState()
	if !es.InAction() {
		sv.ManipStart("NodeAdj", es.ActivePath.Nm)
		sv.GatherAlignPoints()
	}

	svoff := math32.Vector2FromPoint(sv.Geom.ContentBBox.Min)
	pn := es.PathNodes[idx]

	InactivateSprites(sv, SpAlignMatch)

	spt := math32.Vector2FromPoint(es.DragStartPos)
	mpt := math32.Vector2FromPoint(me.Where)

	if me.HasAnyModifier(key.Control) {
		mpt, _ = sv.ConstrainPoint(spt, mpt)
	}
	if Settings.SnapNodes {
		mpt = sv.SnapPoint(mpt)
	}

	es.DragCurPos = mpt.ToPoint()
	mdel := es.DragCurPos.Sub(es.DragStartPos)
	dv := math32.Vector2FromPoint(mdel)

	nwc := pn.WinPt.Add(dv) // new window coord
	sv.PathNodeSetOnePoint(es.ActivePath, es.PathNodes, idx, dv, svoff)

	spnm := SpriteName(SpNodePoint, SpUnk, idx)
	sp, _ := win.SpriteByName(spnm)
	SetSpritePos(sp, image.Point{int(nwc.X), int(nwc.Y)})
	go sv.RenderSVG()
	win.UpdateSig()
}
*/
