// Copyright (c) 2021, The Grid Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grid

import (
	"image"
	"math"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/gi/svg"
	"github.com/goki/ki/ints"
	"github.com/goki/ki/ki"
	"github.com/goki/mat32"
)

// node = path

// PathNode is info about each node in a path that is being edited
type PathNode struct {
	Cmd      svg.PathCmds `desc:"path command"`
	PrevCmd  svg.PathCmds `desc:"previous path command"`
	CmdIdx   int          `desc:"starting index of command"`
	Idx      int          `desc:"index of points in data stream"`
	PtIdx    int          `desc:"logical index of point within current command (0 = first point, etc)"`
	Cp       mat32.Vec2   `desc:"local coords abs current point that is starting point for this command"`
	Pt       mat32.Vec2   `desc:"local coords abs point"`
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
	var cp mat32.Vec2
	svg.PathDataIterFunc(path.Data, func(idx int, cmd svg.PathCmds, ptIdx int, cx, cy float32) bool {
		c := mat32.Vec2{cx, cy}
		cw := pxf.MulVec2AsPt(c).Add(svoff)

		if ptIdx == 0 {
			lstCmdIdx = idx - 1
			cidxs = append(cidxs, lstCmdIdx)
		}
		pn := &PathNode{Cmd: cmd, PrevCmd: lstCmd, CmdIdx: lstCmdIdx, Idx: idx, PtIdx: ptIdx, Cp: cp, Pt: c, WinPt: cw}
		nc = append(nc, pn)
		cp = c
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
		// fmt.Printf("click %s\n", spi)
		if me.Action == mouse.Press {
			win.SpriteDragging = SpriteName(spi)
			es.DragNodeStart(me.Where)
			// fmt.Printf("dragging: %s\n", win.SpriteDragging)
		} else if me.Action == mouse.Release {
			sv.UpdateNodeSprites()
			sv.ManipDone()
		}
	case oswin.MouseDragEvent:
		me := d.(*mouse.DragEvent)
		me.SetProcessed()
		// fmt.Printf("drag %v delta: %v\n", sp, me.Delta())
		// if me.HasAnyModifier(key.Alt) {
		// 	sv.SpriteRotateDrag(sp, me.Delta(), win)
		// } else {
		sv.SpriteNodeDrag(spi, me.Delta(), win, me)
		// }
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
			path.Data[pn.Idx] = svg.PathData(npt.X - pn.Cp.X)
		case svg.Pcv:
			path.Data[pn.Idx] = svg.PathData(npt.Y - pn.Cp.Y)
		default:
			path.Data[pn.Idx] = svg.PathData(npt.X - pn.Cp.X)
			path.Data[pn.Idx+1] = svg.PathData(npt.Y - pn.Cp.Y)
		}
	}
}

// SnapPoint does snapping on one raw point, given that point,
// in window coordinates. returns the snapped point.
func (sv *SVGView) SnapPoint(rawpt mat32.Vec2) mat32.Vec2 {
	es := sv.EditState()
	snapped := false
	snpt := rawpt
	if Prefs.SnapGuide {
		clDst := [2]float32{float32(math.MaxFloat32), float32(math.MaxFloat32)}
		var clPts [2][]BBoxPoints
		var clVals [2][]mat32.Vec2
		for ap := BBLeft; ap < BBoxPointsN; ap++ {
			pts := es.AlignPts[ap]
			dim := ap.Dim()
			for _, pt := range pts {
				pv := pt.Dim(dim)
				bv := rawpt.Dim(dim)
				dst := mat32.Abs(pv - bv)
				if dst < clDst[dim] {
					clDst[dim] = dst
					clPts[dim] = []BBoxPoints{ap}
					clVals[dim] = []mat32.Vec2{pt}
				} else if mat32.Abs(dst-clDst[dim]) < 1.0e-4 {
					clPts[dim] = append(clPts[dim], ap)
					clVals[dim] = append(clVals[dim], pt)
				}
			}
		}
		var alpts []image.Rectangle
		var altyps []BBoxPoints
		for dim := mat32.X; dim <= mat32.Y; dim++ {
			if len(clVals[dim]) == 0 {
				continue
			}
			bv := rawpt.Dim(dim)
			sval, snap := SnapToPt(bv, clVals[dim][0].Dim(dim))
			if snap {
				snpt.SetDim(dim, sval)
				mx := ints.MinInt(len(clVals[dim]), 4)
				for i := 0; i < mx; i++ {
					pt := clVals[dim][i]
					rpt := image.Rectangle{}
					rpt.Min = rawpt.ToPoint()
					rpt.Max = pt.ToPoint()
					if dim == mat32.X {
						rpt.Min.X = rpt.Max.X
					} else {
						rpt.Min.Y = rpt.Max.Y
					}
					alpts = append(alpts, rpt)
					altyps = append(altyps, clPts[dim][i])
				}
				snapped = true
			}
		}
		sv.ShowAlignMatches(alpts, altyps)
	}
	if !snapped && Prefs.SnapGrid {
		// grinc, groff := sv.GridDots()
		// todo: moving check Min, else ?
	}
	return snpt
}

// SpriteNodeDrag processes a mouse node drag event on a path node sprite
func (sv *SVGView) SpriteNodeDrag(spi Sprites, delta image.Point, win *gi.Window, me *mouse.DragEvent) {
	es := sv.EditState()
	if !es.InAction() {
		sv.ManipStart("NodeAdj", es.ActivePath.Nm)
		sv.GatherAlignPoints()
	}

	svoff := mat32.NewVec2FmPoint(sv.WinBBox.Min)
	spt := int(spi - SpritesN)
	pn := es.PathNodes[spt]

	InactivateSpriteRange(win, AlignMatch1, AlignMatch8)
	// switch {
	// case me.HasAnyModifier(key.Control):
	// 	if bbX != BBCenter && bbY != BBMiddle {
	// 		sv.ConstrainCurBBox(false, bbX, bbY) // reshape
	// 	}
	// default:
	snpt := sv.SnapPoint(mat32.NewVec2FmPoint(me.Where))
	// }

	es.DragCurPos = snpt.ToPoint()
	mdel := es.DragCurPos.Sub(es.DragStartPos)
	dv := mat32.NewVec2FmPoint(mdel)
	nwc := pn.WinPt.Add(dv) // new window coord

	wbmin := mat32.NewVec2FmPoint(es.ActivePath.WinBBox.Min)
	pt := wbmin.Sub(svoff)
	xf, lpt := es.ActivePath.DeltaXForm(dv, mat32.Vec2{1, 1}, 0, pt, true) // include self

	npt := xf.MulVec2AsPtCtr(pn.Pt, lpt) // transform point to new abs coords
	sv.PathNodeSetPoint(es.ActivePath, pn, npt)

	sp := es.ActiveSprites[spi]
	SetSpritePos(spi, sp, image.Point{int(nwc.X), int(nwc.Y)})
	go sv.ManipUpdate()
	win.RenderOverlays()
}
