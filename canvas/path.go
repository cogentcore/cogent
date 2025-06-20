// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package canvas

import (
	"fmt"
	"image"
	"slices"

	"cogentcore.org/cogent/canvas/cicons"
	"cogentcore.org/core/core"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/ppath"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/svg"
	"cogentcore.org/core/tree"
)

// PathNode is info about each node in a path that is being edited
type PathNode struct {
	// path command using sprite encoding
	Cmd Sprites

	// CmdPath command using ppath float32 encoding
	CmdPath float32

	// index of start of command in path
	Index int

	// logical index of point within current command (0 = first point, etc)
	PtIndex int

	// original data points:
	Start, End, Cp1, Cp2 math32.Vector2

	// transformed to scene coordinates:
	TStart, TEnd, TCp1, TCp2 math32.Vector2
}

// EndIndex returns the exclusive index for the end of this command:
// Index + CmdLen for this command.
func (pn *PathNode) EndIndex() int {
	return pn.Index + ppath.CmdLen(pn.CmdPath)
}

// PathNodes returns the PathNode data for given svg.Path.
func (sv *SVG) PathNodes(path *svg.Path) []*PathNode {
	xf := path.ParentTransform(true) // include self
	pts := path.Data
	pns := make([]*PathNode, 0)
	pti := 0
	for scan := pts.Scanner(); scan.Scan(); {
		cmd := scan.Cmd()
		end := scan.End()
		start := scan.Start()
		tend := xf.MulVector2AsPoint(end)
		tstart := xf.MulVector2AsPoint(start)

		pn := &PathNode{CmdPath: cmd, Index: scan.Index(), PtIndex: pti, Start: start, End: end, TStart: tstart, TEnd: tend}
		pns = append(pns, pn)
		pti++

		switch cmd {
		case ppath.MoveTo:
			pn.Cmd = SpMoveTo
		case ppath.LineTo:
			pn.Cmd = SpLineTo
		case ppath.QuadTo:
			pn.Cmd = SpQuadTo
			pn.Cp1 = scan.CP1()
			pn.TCp1 = xf.MulVector2AsPoint(pn.Cp1)
		case ppath.CubeTo:
			pn.Cmd = SpCubeTo
			pn.Cp1 = scan.CP1()
			pn.Cp2 = scan.CP2()
			pn.TCp1 = xf.MulVector2AsPoint(pn.Cp1)
			pn.TCp2 = xf.MulVector2AsPoint(pn.Cp2)
		// todo: arc
		case ppath.Close:
			pn.Cmd = SpClose
		}
		// fmt.Println(pti-1, pn.Cmd, end)
	}
	return pns
}

func (sv *SVG) UpdateNodeSprites() {
	es := sv.EditState()
	path := es.ActivePath
	if path == nil || tree.IsNil(path) {
		sv.RemoveNodeSprites()
		return
	}

	prvn := es.NNodeSprites
	sprites := sv.SpritesLock()

	es.PathNodes = sv.PathNodes(path)
	es.NNodeSprites = len(es.PathNodes)
	es.ActivePath = path

	for i, pn := range es.PathNodes {
		ept := pn.TEnd.ToPoint()
		sp := sv.Sprite(SpNodePoint, pn.Cmd, i, image.Point{}, func(sp *core.Sprite) {
			sp.On(events.MouseEnter, func(e events.Event) {
				es.NodeHover = i
				e.SetHandled()
				sv.NeedsRender()
			})
			sp.On(events.MouseLeave, func(e events.Event) {
				es.NodeHover = -1
				e.SetHandled()
				sv.NeedsRender()
			})
			sp.OnClick(func(e events.Event) {
				e.SetHandled()
				if es.Tool == BezierTool {
					if i == 0 {
						es.ActivePath.Data.Close()
						es.ActivePath = nil
						sv.UpdateView()
					} else if i == len(es.PathNodes)-1 {
						es.ActivePath = nil
						sv.UpdateView()
					}
					return
				}
				es.NodeSelectAction(i, e.SelectMode())
				sv.UpdateView()
			})
			sp.OnSlideStart(func(e events.Event) {
				if es.Tool == BezierTool {
					e.SetHandled()
					return
				}
				if len(es.NodeSelect) == 0 {
					es.SelectNode(i)
				} else if !e.HasAnyModifier(key.Shift, key.Meta) && len(es.NodeSelect) == 1 && !es.NodeIsSelected(i) {
					es.ResetSelectedNodes()
					es.SelectNode(i)
				}
				es.DragNodeStart(e.Pos())
				es.PathNodesOrig = sv.PathNodes(es.ActivePath)
				sv.Styles.Cursor = cursors.Move
				sv.NeedsRender()
				e.SetHandled()
			})
			sp.OnSlideMove(func(e events.Event) {
				if es.Tool == BezierTool {
					e.SetHandled()
					return
				}
				sv.SpriteNodeDrag(i, e)
				sv.Styles.Cursor = cursors.Move
				e.SetHandled()
			})
			sp.OnSlideStop(func(e events.Event) {
				if es.Tool == BezierTool {
					sv.InactivateAlignSprites()
					e.SetHandled()
					return
				}
				sv.ManipDone()
				e.SetHandled()
			})
		})
		sv.SetSpritePos(sp, ept.X, ept.Y)

		ctrlEvents := func(sp *core.Sprite, i int, ctyp Sprites) {
			sp.On(events.MouseEnter, func(e events.Event) {
				es.CtrlHover = i
				es.CtrlHoverType = ctyp
				e.SetHandled()
				sv.NeedsRender()
			})
			sp.On(events.MouseLeave, func(e events.Event) {
				es.CtrlHover = -1
				e.SetHandled()
				sv.NeedsRender()
			})
			sp.OnSlideStart(func(e events.Event) {
				es.DragCtrlStart(e.Pos(), i, ctyp)
				es.PathNodesOrig = sv.PathNodes(es.ActivePath)
				sv.NeedsRender()
				sv.Styles.Cursor = cursors.Move
				e.SetHandled()
			})
			sp.OnSlideMove(func(e events.Event) {
				sv.SpriteCtrlDrag(i, ctyp, e)
				sv.Styles.Cursor = cursors.Move
				e.SetHandled()
			})
			sp.OnSlideStop(func(e events.Event) {
				es.CtrlDragIndex = -1
				sv.ManipDone()
				e.SetHandled()
			})
		}

		InactivateNodeCtrls(sprites, i)
		if es.Tool == BezierTool {
			if i == len(es.PathNodes)-1 {
				sv.UpdateLineAddSprite()
			}
		} else {
			if _, isSel := es.NodeSelect[i]; isSel {
				switch pn.Cmd {
				case SpQuadTo:
					sp1 := sv.Sprite(SpNodeCtrl, SpQuad1, i, ept, func(sp *core.Sprite) {
						ctrlEvents(sp, i, SpQuad1)
					})
					sp1.Properties["nodePoint"] = ept
					sv.SetSpritePos(sp1, int(pn.TCp1.X), int(pn.TCp1.Y))
				case SpCubeTo:
					spt := pn.TStart.ToPoint()
					sp1 := sv.Sprite(SpNodeCtrl, SpCube1, i, spt, func(sp *core.Sprite) {
						ctrlEvents(sp, i, SpCube1)
					})
					sp1.Properties["nodePoint"] = spt
					sv.SetSpritePos(sp1, int(pn.TCp1.X), int(pn.TCp1.Y))
					sp2 := sv.Sprite(SpNodeCtrl, SpCube2, i, ept, func(sp *core.Sprite) {
						ctrlEvents(sp, i, SpCube2)
					})
					sp2.Properties["nodePoint"] = ept
					sv.SetSpritePos(sp2, int(pn.TCp2.X), int(pn.TCp2.Y))
				}
			}
		}
	}

	// remove extra
	for i := es.NNodeSprites; i < prvn; i++ {
		InactivateNodePoint(sprites, i)
	}
	sprites.Unlock()
}

func (sv *SVG) UpdateLineAddSprite() {
	es := sv.EditState()
	np := len(es.PathNodes)
	if np == 0 {
		return
	}
	pn := es.PathNodes[np-1]
	ept := pn.TEnd.ToPoint()
	sp := sv.Sprite(SpLineAdd, SpNone, 0, ept, nil)
	sp.Properties["lastPoint"] = ept
	sv.SetSpritePos(sp, es.DrawSnapPos.X, es.DrawSnapPos.Y)
}

func InactivateNodePoint(sprites *core.Sprites, i int) {
	sprites.InactivateSpriteNoLock(SpriteName(SpNodePoint, SpNone, i))
	InactivateNodeCtrls(sprites, i)
}

func InactivateNodeCtrls(sprites *core.Sprites, i int) {
	sprites.InactivateSpriteNoLock(SpriteName(SpNodeCtrl, SpQuad1, i))
	sprites.InactivateSpriteNoLock(SpriteName(SpNodeCtrl, SpCube1, i))
	sprites.InactivateSpriteNoLock(SpriteName(SpNodeCtrl, SpCube2, i))
	sprites.InactivateSpriteNoLock(SpriteName(SpLineAdd, SpNone, 0))
}

func (sv *SVG) RemoveNodeSprites() {
	es := sv.EditState()
	sprites := sv.SpritesLock()
	for i := 0; i < es.NNodeSprites; i++ {
		InactivateNodePoint(sprites, i)
	}
	es.NNodeSprites = 0
	es.PathNodes = nil
	es.ActivePath = nil
	es.PathNodesOrig = nil
	es.CtrlDragIndex = -1
	es.NodeHover = -1
	es.CtrlHover = -1
	sprites.Unlock()
}

// DrawPoint updates the DrawPos and DrawSnapPos from event position.
// Returns the DrawSnapPos, which is the point to use for drawing.
func (sv *SVG) DrawPoint(e events.Event) image.Point {
	es := sv.EditState()
	spt := math32.FromPoint(es.DrawStartPos)
	mpt := math32.FromPoint(e.Pos())
	if e.HasAnyModifier(key.Control) {
		mpt, _ = sv.ConstrainPoint(spt, mpt)
	}
	es.DrawPos = mpt.ToPointRound()
	es.DrawSnapPos = es.DrawPos
	if Settings.SnapNodes {
		sprites := sv.SpritesLock()
		sv.InactivateSprites(SpAlignMatch)
		es.DrawSnapPos = sv.SnapPoint(mpt).ToPointRound()
		sprites.Unlock()
	}
	return es.DrawSnapPos
}

// SpriteNodeDrag processes a mouse node drag event on a path node sprite
func (sv *SVG) SpriteNodeDrag(idx int, e events.Event) {
	es := sv.EditState()
	sprites := sv.SpritesLock()
	sv.ManipStartInDrag(NodeMove, es.ActivePath.Name)
	es.ConstrainPoint = true
	spt, _, dv := sv.DragDelta(e, true)

	pointOnly := e.HasAnyModifier(key.Alt)
	dxf := es.ActivePath.DeltaTransform(dv, math32.Vector2{1, 1}, 0, spt)

	for i, _ := range es.NodeSelect {
		sv.PathNodeMove(i, pointOnly, dv, dxf)
		pn := es.PathNodes[i]
		nwc := pn.TEnd.Add(dv).ToPoint()
		spnm := SpriteName(SpNodePoint, SpNone, i)
		sp, _ := sprites.SpriteByNameNoLock(spnm)
		sv.SetSpritePos(sp, nwc.X, nwc.Y)

		switch pn.Cmd {
		case SpQuadTo:
			if sp1, ok := sprites.SpriteByNameNoLock(SpriteName(SpNodeCtrl, SpQuad1, i)); ok {
				sp1.Properties["nodePoint"] = nwc
			}
		case SpCubeTo:
			if sp2, ok := sprites.SpriteByNameNoLock(SpriteName(SpNodeCtrl, SpCube2, i)); ok {
				sp2.Properties["nodePoint"] = nwc
			}
		}
		// next node is using us as a start
		if sp1, ok := sprites.SpriteByNameNoLock(SpriteName(SpNodeCtrl, SpCube1, i+1)); ok {
			sp1.Properties["nodePoint"] = nwc
		}
	}

	sprites.Unlock()
	sv.UpdateView()
}

// SpriteCtrlDrag processes a mouse node drag event on a path control sprite
func (sv *SVG) SpriteCtrlDrag(idx int, ctyp Sprites, e events.Event) {
	es := sv.EditState()
	sprites := sv.SpritesLock()
	sv.ManipStartInDrag(CtrlMove, es.ActivePath.Name)

	es.ConstrainPoint = true
	spt, _, dv := sv.DragDelta(e, true)
	dxf := es.ActivePath.DeltaTransform(dv, math32.Vector2{1, 1}, 0, spt)

	pos := sv.PathCtrlMove(idx, ctyp, dxf)
	nwc := pos.Add(dv)
	spnm := SpriteName(SpNodeCtrl, ctyp, idx)
	sp, _ := sprites.SpriteByNameNoLock(spnm)
	sv.SetSpritePos(sp, int(nwc.X), int(nwc.Y))

	sprites.Unlock()
	sv.UpdateView()
}

// PathNodeMove moves given node index by given delta transform.
// pointOnly = true moves just the end point, otherwise all move.
func (sv *SVG) PathNodeMove(pidx int, pointOnly bool, dv math32.Vector2, dxf math32.Matrix2) {
	es := sv.EditState()
	sprites := sv.SpritesNoLock()
	path := es.ActivePath
	pn := es.PathNodesOrig[pidx]
	_, isSel := es.NodeSelect[pidx]
	end := dxf.MulVector2AsPoint(pn.End)
	switch pn.Cmd {
	case SpMoveTo, SpLineTo, SpClose:
		path.Data[pn.Index+1] = end.X
		path.Data[pn.Index+2] = end.Y
	case SpQuadTo:
		path.Data[pn.Index+3] = end.X
		path.Data[pn.Index+4] = end.Y
		if !pointOnly && isSel {
			cp1 := dxf.MulVector2AsPoint(pn.Cp1)
			path.Data[pn.Index+1] = cp1.X
			path.Data[pn.Index+2] = cp1.Y
			if sp1, ok := sprites.SpriteByNameNoLock(SpriteName(SpNodeCtrl, SpQuad1, pidx)); ok {
				sv.SetSpritePos(sp1, int(pn.TCp1.X+dv.X), int(pn.TCp1.Y+dv.Y))
			}
		}
	case SpCubeTo:
		path.Data[pn.Index+5] = end.X
		path.Data[pn.Index+6] = end.Y
		if !pointOnly && isSel {
			cp2 := dxf.MulVector2AsPoint(pn.Cp2)
			path.Data[pn.Index+3] = cp2.X
			path.Data[pn.Index+4] = cp2.Y
			if sp2, ok := sprites.SpriteByNameNoLock(SpriteName(SpNodeCtrl, SpCube2, pidx)); ok {
				sv.SetSpritePos(sp2, int(pn.TCp2.X+dv.X), int(pn.TCp2.Y+dv.Y))
			}
		}
		// todo: arc
	}
	if pointOnly || !isSel || pidx+1 >= len(es.PathNodes) {
		return
	}
	// update next node control point b/c it uses start which is this guy
	pidx++
	pn = es.PathNodesOrig[pidx]
	if pn.Cmd != SpCubeTo {
		return
	}
	cp1 := dxf.MulVector2AsPoint(pn.Cp1)
	path.Data[pn.Index+1] = cp1.X
	path.Data[pn.Index+2] = cp1.Y
	sp1, ok := sprites.SpriteByNameNoLock(SpriteName(SpNodeCtrl, SpCube1, pidx))
	if ok {
		sv.SetSpritePos(sp1, int(pn.TCp1.X+dv.X), int(pn.TCp1.Y+dv.Y))
	}
}

// PathCtrlMove moves given node control point index by given delta transform.
// returns scene position of given point.
func (sv *SVG) PathCtrlMove(pidx int, ctyp Sprites, dxf math32.Matrix2) math32.Vector2 {
	es := sv.EditState()
	path := es.ActivePath
	pn := es.PathNodesOrig[pidx]
	switch ctyp {
	case SpQuad1, SpCube1:
		cp1 := dxf.MulVector2AsPoint(pn.Cp1)
		path.Data[pn.Index+1] = cp1.X
		path.Data[pn.Index+2] = cp1.Y
		return pn.TCp1
	case SpCube2:
		cp2 := dxf.MulVector2AsPoint(pn.Cp2)
		path.Data[pn.Index+3] = cp2.X
		path.Data[pn.Index+4] = cp2.Y
		return pn.TCp2
	}
	return math32.Vector2{}
}

// nodeEnabledStyler sets the given widget to only be enabled when
// in NodeTool mode.
func (cv *Canvas) nodeEnabledStyler(w core.Widget) {
	es := &cv.EditState
	w.AsWidget().FirstStyler(func(s *styles.Style) {
		s.SetEnabled(es.Tool == NodeTool)
	})
}

// nodeSelectEnabledStyler sets the given widget to only be enabled when
// there is a node selected.
func (cv *Canvas) nodeSelectEnabledStyler(w core.Widget) {
	es := &cv.EditState
	w.AsWidget().FirstStyler(func(s *styles.Style) {
		s.SetEnabled(es.Tool == NodeTool && es.HasNodeSelected())
	})
}

func (cv *Canvas) MakeNodeToolbar(p *tree.Plan) {
	// es := &cv.EditState
	tree.Add(p, func(w *core.Switch) {
		core.Bind(&Settings.SnapNodes, w)
		w.SetText("Snap nodes").SetTooltip("Snap movement and sizing of nodes, using overall snap settings")
	})
	tree.Add(p, func(w *core.Separator) {})

	tree.Add(p, func(w *core.FuncButton) {
		cv.nodeEnabledStyler(w)
		w.SetFunc(cv.InsertLineNode).SetIcon(cicons.NodeAdd).SetText("Line")
	})
	tree.Add(p, func(w *core.FuncButton) {
		cv.nodeEnabledStyler(w)
		w.SetFunc(cv.InsertCubicNode).SetIcon(cicons.NodeAdd).SetText("Curve")
	})
	tree.Add(p, func(w *core.FuncButton) {
		cv.nodeSelectEnabledStyler(w)
		w.SetFunc(cv.InsertBreak).SetIcon(cicons.NodeBreak).SetText("Break")
	})
	tree.Add(p, func(w *core.FuncButton) {
		cv.nodeSelectEnabledStyler(w)
		w.SetFunc(cv.NodeDelete).SetIcon(cicons.NodeDelete).SetText("Delete")
	})
	tree.Add(p, func(w *core.FuncButton) {
		cv.nodeSelectEnabledStyler(w)
		w.SetFunc(cv.NodeReplaceCubic).SetIcon(cicons.NodeSmooth).SetText("Smooth")
	})
}

////////  Actions

// InsertLineNode inserts a LineTo node into current active path:
// If no selection: at end, otherwise after first selected node.
func (cv *Canvas) InsertLineNode() { //types:add
	cv.NodeInsert(SpLineTo)
}

// InsertCubicNode inserts a cubic node into current active path:
// If no selection: at end, otherwise after first selected node.
func (cv *Canvas) InsertCubicNode() { //types:add
	cv.NodeInsert(SpCubeTo)
}

// InsertBreak inserts a break (move) into current active path:
// If no selection: at end, otherwise after first selected node.
func (cv *Canvas) InsertBreak() { //types:add
	cv.NodeInsert(SpMoveTo)
}

// NodeReplaceCubic replaces selected non-cubic nodes with cubic ones
// into current active path
func (cv *Canvas) NodeReplaceCubic() { //types:add
	cv.NodeReplace(SpCubeTo)
}

// NodeDelete deletes the selected node(s) from current active path.
func (cv *Canvas) NodeDelete() { //types:add
	es := &cv.EditState
	sv := cv.SVG
	sls := es.NodeSelectedList()
	n := len(sls)
	if n == 0 {
		return
	}
	sv.UndoSave("NodeDelete", "")
	for i := n - 1; i >= 0; i-- {
		idx := sls[i]
		sv.NodeDelete(idx)
	}
	sv.UpdateView()
}

// NodeInsert inserts a node of given type into current active path:
// If no selection: at end, otherwise after first selected node.
func (cv *Canvas) NodeInsert(ntyp Sprites) {
	es := &cv.EditState
	sv := cv.SVG
	if es.ActivePath == nil {
		return
	}
	sv.UndoSave("NodeInsert", ntyp.String())
	sls := es.NodeSelectedList()
	nsel := len(sls)
	if nsel == 0 {
		sv.NodeAppend(ntyp)
		return
	}
	sv.NodeInsert(ntyp, sls[0])
	sv.UpdateView()
}

// NodeReplace replaces current node with new one of given type
// into current active path.
func (cv *Canvas) NodeReplace(ntyp Sprites) {
	es := &cv.EditState
	sv := cv.SVG
	if es.ActivePath == nil {
		return
	}
	sls := es.NodeSelectedList()
	n := len(sls)
	if n == 0 {
		return
	}
	sv.UndoSave("NodeReplace", ntyp.String())
	for i := n - 1; i >= 0; i-- {
		idx := sls[i]
		sv.NodeReplace(ntyp, idx)
	}
	sv.UpdateView()
}

func (sv *SVG) NodeAdd(ntyp Sprites, p ppath.Path, end, start math32.Vector2) ppath.Path {
	es := sv.EditState()
	path := es.ActivePath
	xf := path.ParentTransform(true).Inverse()
	lend := xf.MulVector2AsPoint(end)
	lstart := xf.MulVector2AsPoint(start)
	del := lend.Sub(lstart)
	switch ntyp {
	case SpClose:
		fmt.Println("add close")
		p.Close()
	case SpMoveTo:
		p.MoveTo(lend.X, lend.Y)
	case SpLineTo:
		p.LineTo(lend.X, lend.Y)
	case SpQuadTo:
		cp1 := lend.Add(del.MulScalar(0.25))
		p.QuadTo(cp1.X, cp1.Y, lend.X, lend.Y)
	case SpCubeTo:
		cp1 := lstart.Add(del.MulScalar(0.25))
		cp2 := lend.Sub(del.MulScalar(0.25))
		p.CubeTo(cp1.X, cp1.Y, cp2.X, cp2.Y, lend.X, lend.Y)
	}
	return p
}

func (sv *SVG) NodeAppend(ntyp Sprites) {
	es := sv.EditState()
	var end, start math32.Vector2
	np := len(es.PathNodes)
	ctr := math32.FromPoint(sv.SVG.Geom.Size).MulScalar(0.5)
	switch np {
	case 0:
		end = math32.FromPoint(sv.SVG.Geom.Pos).Add(ctr)
		start = end.Add(ctr.MulScalar(0.1))
	case 1:
		start = es.PathNodes[np-1].TEnd
		end = start.Add(ctr.MulScalar(0.1))
	default:
		start = es.PathNodes[np-1].TEnd
		del := start.Sub(es.PathNodes[np-2].TEnd)
		end = start.Add(del)
	}
	path := es.ActivePath
	path.Data = sv.NodeAdd(ntyp, path.Data, end, start)
}

func (sv *SVG) NodeInsert(ntyp Sprites, idx int) {
	es := sv.EditState()
	np := len(es.PathNodes)
	if idx >= np-1 {
		sv.NodeAppend(ntyp)
		return
	}
	snd := es.PathNodes[idx]
	nnd := es.PathNodes[idx+1]
	start := snd.TEnd
	end := nnd.TEnd
	del := end.Sub(start)
	mid := start.Add(del.MulScalar(0.5))
	path := es.ActivePath
	sp := path.Data[:nnd.Index]
	rest := path.Data[nnd.Index:].Clone()
	sp = sv.NodeAdd(ntyp, sp, mid, start)
	path.Data = append(sp, rest...)
}

func (sv *SVG) NodeReplace(ntyp Sprites, idx int) {
	es := sv.EditState()
	path := es.ActivePath
	snd := es.PathNodes[idx]
	if snd.Cmd == ntyp {
		return
	}
	start := snd.TStart
	end := snd.TEnd
	sp := path.Data[:snd.Index]
	rest := path.Data[snd.EndIndex():].Clone()
	sp = sv.NodeAdd(ntyp, sp, end, start)
	path.Data = append(sp, rest...)
}

// NodeDelete deletes node from active path at given index.
// if doing multiple deletes in a loop, go in reverse.
func (sv *SVG) NodeDelete(idx int) {
	es := sv.EditState()
	pn := es.PathNodes[idx]
	es.ActivePath.Data = slices.Delete(es.ActivePath.Data, pn.Index, pn.EndIndex())
}

// NodeDeleteLast deletes the last node from ActivePath if n > 1.
func (sv *SVG) NodeDeleteLast() {
	es := sv.EditState()
	if es.ActivePath == nil {
		return
	}
	np := len(es.PathNodes)
	if np <= 1 {
		return
	}
	sv.NodeDelete(np - 1)
	sv.UpdateView()
}

func (sv *SVG) DrawNodeAdd(ntyp Sprites, pos image.Point) {
	es := sv.EditState()
	np := len(es.PathNodes)
	var start math32.Vector2
	if ntyp != SpMoveTo && np > 0 {
		start = es.PathNodes[np-1].TEnd
	}
	end := math32.FromPoint(pos)
	path := es.ActivePath
	path.Data = sv.NodeAdd(ntyp, path.Data, end, start)
}
