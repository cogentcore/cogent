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

func (gv *GridView) NodeToolbar() *gi.ToolBar {
	tbs := gv.ModalToolbarStack()
	tb := tbs.ChildByName("node-tb", 0).(*gi.ToolBar)
	return tb
}

// ConfigNodeToolbar configures the node modal toolbar (default tooblar)
func (gv *GridView) ConfigNodeToolbar() {
	tb := gv.NodeToolbar()
	if tb.HasChildren() {
		return
	}
	tb.SetStretchMaxWidth()

	grs := gi.AddNewCheckBox(tb, "snap-node")
	grs.SetText("Snap Node")
	grs.Tooltip = "snap movement and sizing of nodes, using overall snap settings"
	grs.SetChecked(Prefs.SnapNodes)
	grs.ButtonSig.Connect(gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.ButtonToggled) {
			Prefs.SnapNodes = grs.IsChecked()
		}
	})

	tb.AddSeparator("sep-snap")

	// tb.AddAction(gi.ActOpts{Icon: "sel-group", Tooltip: "Ctrl+G: Group items together", UpdateFunc: gv.NodeEnableFunc},
	// 	gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
	// 		grr := recv.Embed(KiT_GridView).(*GridView)
	// 		grr.SelGroup()
	// 	})
	//
	// tb.AddSeparator("sep-group")

	gi.AddNewLabel(tb, "posx-lab", "X: ").SetProp("vertical-align", gist.AlignMiddle)
	px := gi.AddNewSpinBox(tb, "posx")
	px.SetProp("step", 1)
	px.SetValue(0)
	px.Tooltip = "horizontal coordinate of node, in document units"
	px.SpinBoxSig.Connect(gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		grr := recv.Embed(KiT_GridView).(*GridView)
		grr.NodeSetXPos(px.Value)
	})

	gi.AddNewLabel(tb, "posy-lab", "Y: ").SetProp("vertical-align", gist.AlignMiddle)
	py := gi.AddNewSpinBox(tb, "posy")
	py.SetProp("step", 1)
	py.SetValue(0)
	py.Tooltip = "vertical coordinate of node, in document units"
	py.SpinBoxSig.Connect(gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		grr := recv.Embed(KiT_GridView).(*GridView)
		grr.NodeSetYPos(py.Value)
	})

}

// NodeEnableFunc is an ActionUpdateFunc that inactivates action if no node selected
func (gv *GridView) NodeEnableFunc(act *gi.Action) {
	// es := &gv.EditState
	// act.SetInactiveState(!es.HasNodeed())
}

// UpdateNodeToolbar updates the node toolbar based on current nodeion
func (gv *GridView) UpdateNodeToolbar() {
	tb := gv.NodeToolbar()
	tb.UpdateActions()
	es := &gv.EditState
	if es.Tool != NodeTool {
		return
	}
	px := tb.ChildByName("posx", 8).(*gi.SpinBox)
	px.SetValue(es.DragSelCurBBox.Min.X)
	py := tb.ChildByName("posy", 9).(*gi.SpinBox)
	py.SetValue(es.DragSelCurBBox.Min.Y)
}

///////////////////////////////////////////////////////////////////////
//   Actions

func (gv *GridView) NodeSetXPos(xp float32) {
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sv := gv.SVG()
	sv.UndoSave("NodeToX", fmt.Sprintf("%g", xp))
}

func (gv *GridView) NodeSetYPos(yp float32) {
	es := &gv.EditState
	if !es.HasSelected() {
		return
	}
	sv := gv.SVG()
	sv.UndoSave("NodeToY", fmt.Sprintf("%g", yp))
}

//////////////////////////////////////////////////////////////////////////
//  PathNode

// PathNode is info about each node in a path that is being edited
type PathNode struct {
	Cmd      svg.PathCmds `desc:"path command"`
	PrevCmd  svg.PathCmds `desc:"previous path command"`
	CmdIdx   int          `desc:"starting index of command"`
	Idx      int          `desc:"index of points in data stream"`
	PtIdx    int          `desc:"logical index of point within current command (0 = first point, etc)"`
	PCp      mat32.Vec2   `desc:"local coords abs previous current point that is starting point for this command"`
	Cp       mat32.Vec2   `desc:"local coords abs current point"`
	WinPt    mat32.Vec2   `desc:"main point coords in window (dot) coords"`
	WinCtrls []mat32.Vec2 `desc:"control point coords in window (dot) coords (nil until manipulated)"`
}

// PathNodes returns the PathNode data for given path data, and a list of indexes where commands start
func (sv *SVGView) PathNodes(path *svg.Path) ([]*PathNode, []int) {
	svoff := mat32.NewVec2FmPoint(sv.WinBBox.Min)
	pxf := path.ParXForm()

	lstCmdIdx := 0
	lstCmd := svg.PcErr
	nc := make([]*PathNode, 0)
	cidxs := make([]int, 0)
	var pcp mat32.Vec2
	svg.PathDataIterFunc(path.Data, func(idx int, cmd svg.PathCmds, ptIdx int, cp mat32.Vec2, ctrl []mat32.Vec2) bool {
		cw := pxf.MulVec2AsPt(cp).Add(svoff)

		if ptIdx == 0 {
			lstCmdIdx = idx - 1
			cidxs = append(cidxs, lstCmdIdx)
		}
		pn := &PathNode{Cmd: cmd, PrevCmd: lstCmd, CmdIdx: lstCmdIdx, Idx: idx, PtIdx: ptIdx, PCp: pcp, Cp: cp, WinPt: cw, WinCtrls: ctrl}
		nc = append(nc, pn)
		pcp = cp
		lstCmd = cmd
		return ki.Continue
	})
	return nc, cidxs
}

func (sv *SVGView) UpdateNodeSprites() {
	win := sv.GridView.ParentWindow()
	updt := win.UpdateStart()
	defer win.UpdateEnd(updt)

	es := sv.EditState()
	prvn := es.NNodeSprites

	path := es.FirstSelectedPath()

	if path == nil {
		sv.RemoveNodeSprites(win)
		win.RenderOverlays()
		return
	}

	es.PathNodes, es.PathCmds = sv.PathNodes(path)
	es.NNodeSprites = len(es.PathNodes)
	es.ActivePath = path

	for i, pn := range es.PathNodes {
		spi := SpritesN + Sprites(i) // key to get a unique local var
		sp := SpriteConnectEvent(spi, win, image.Point{}, sv.This(), func(recv, send ki.Ki, sig int64, d interface{}) {
			ssvg := recv.Embed(KiT_SVGView).(*SVGView)
			ssvg.NodeSpriteEvent(spi, oswin.EventType(sig), d)
		})
		es.ActiveSprites[spi] = sp

		SetSpritePos(spi, sp, image.Point{int(pn.WinPt.X), int(pn.WinPt.Y)})
	}

	// remove extra
	for i := es.NNodeSprites; i < prvn; i++ {
		sp, has := es.ActiveSprites[SpritesN+Sprites(i)]
		if has {
			win.DeleteSprite(sp.Name)
		}
	}

	sv.GridView.UpdateNodeToolbar()

	win.RenderOverlays()
}

func (sv *SVGView) RemoveNodeSprites(win *gi.Window) {
	es := sv.EditState()
	for i := 0; i < es.NNodeSprites; i++ {
		sp, has := es.ActiveSprites[SpritesN+Sprites(i)]
		if has {
			win.DeleteSprite(sp.Name)
		}
	}
	es.NNodeSprites = 0
	es.PathNodes = nil
	es.PathCmds = nil
	es.ActivePath = nil
}

func (sv *SVGView) NodeSpriteEvent(spi Sprites, et oswin.EventType, d interface{}) {
	win := sv.GridView.ParentWindow()
	es := sv.EditState()
	es.SelNoDrag = false
	switch et {
	case oswin.MouseEvent:
		me := d.(*mouse.Event)
		me.SetProcessed()
		if me.Action == mouse.Press {
			win.SpriteDragging = SpriteName(spi)
			es.DragNodeStart(me.Where)
		} else if me.Action == mouse.Release {
			sv.UpdateNodeSprites()
			sv.ManipDone()
		}
	case oswin.MouseDragEvent:
		me := d.(*mouse.DragEvent)
		me.SetProcessed()
		sv.SpriteNodeDrag(spi, win, me)
	}
}

// PathNodeMoveOnePoint moves given node index by given delta in window coords
// and all following points up to cmd = z or m are moved in the opposite
// direction to compensate, so only the one point is moved in effect.
// svoff is the window starting vector coordinate for view.
func (sv *SVGView) PathNodeSetOnePoint(path *svg.Path, pts []*PathNode, pidx int, dv mat32.Vec2, svoff mat32.Vec2) {
	for i := pidx; i < len(pts); i++ {
		pn := pts[i]
		wbmin := mat32.NewVec2FmPoint(path.WinBBox.Min)
		pt := wbmin.Sub(svoff)
		xf, lpt := path.DeltaXForm(dv, mat32.Vec2{1, 1}, 0, pt, true) // include self
		npt := xf.MulVec2AsPtCtr(pn.Cp, lpt)                          // transform point to new abs coords
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
func (sv *SVGView) PathNodeSetPoint(path *svg.Path, pn *PathNode, npt mat32.Vec2) {
	if pn.Idx == 1 || !svg.PathCmdIsRel(pn.Cmd) { // abs
		switch pn.Cmd {
		case svg.PcH:
			path.Data[pn.Idx] = svg.PathData(npt.X)
		case svg.PcV:
			path.Data[pn.Idx] = svg.PathData(npt.Y)
		default:
			path.Data[pn.Idx] = svg.PathData(npt.X)
			path.Data[pn.Idx+1] = svg.PathData(npt.Y)
		}
	} else {
		switch pn.Cmd {
		case svg.Pch:
			path.Data[pn.Idx] = svg.PathData(npt.X - pn.PCp.X)
		case svg.Pcv:
			path.Data[pn.Idx] = svg.PathData(npt.Y - pn.PCp.Y)
		default:
			path.Data[pn.Idx] = svg.PathData(npt.X - pn.PCp.X)
			path.Data[pn.Idx+1] = svg.PathData(npt.Y - pn.PCp.Y)
		}
	}
}

// SpriteNodeDrag processes a mouse node drag event on a path node sprite
func (sv *SVGView) SpriteNodeDrag(spi Sprites, win *gi.Window, me *mouse.DragEvent) {
	es := sv.EditState()
	if !es.InAction() {
		sv.ManipStart("NodeAdj", es.ActivePath.Nm)
		sv.GatherAlignPoints()
	}

	svoff := mat32.NewVec2FmPoint(sv.WinBBox.Min)
	pidx := int(spi - SpritesN)
	pn := es.PathNodes[pidx]

	InactivateSpriteRange(win, AlignMatch1, AlignMatch8)

	spt := mat32.NewVec2FmPoint(es.DragStartPos)
	mpt := mat32.NewVec2FmPoint(me.Where)

	if me.HasAnyModifier(key.Control) {
		mpt, _ = sv.ConstrainPoint(spt, mpt)
	}
	if Prefs.SnapNodes {
		mpt = sv.SnapPoint(mpt)
	}

	es.DragCurPos = mpt.ToPoint()
	mdel := es.DragCurPos.Sub(es.DragStartPos)
	dv := mat32.NewVec2FmPoint(mdel)

	nwc := pn.WinPt.Add(dv) // new window coord
	sv.PathNodeSetOnePoint(es.ActivePath, es.PathNodes, pidx, dv, svoff)

	sp := es.ActiveSprites[spi]
	SetSpritePos(spi, sp, image.Point{int(nwc.X), int(nwc.Y)})
	go sv.ManipUpdate()
	win.RenderOverlays()
}
