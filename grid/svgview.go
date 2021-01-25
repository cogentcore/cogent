// Copyright (c) 2021, The Grid Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grid

import (
	"fmt"
	"image"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/cursor"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/gi/svg"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
)

// SVGView is the element for viewing, interacting with the SVG
type SVGView struct {
	svg.SVG
	GridView      *GridView  `json:"-" xml:"-" view:"-" desc:"the parent gridview"`
	Trans         mat32.Vec2 `desc:"view translation offset (from dragging)"`
	Scale         float32    `desc:"view scaling (from zooming)"`
	SetDragCursor bool       `view:"-" desc:"has dragging cursor been set yet?"`
}

var KiT_SVGView = kit.Types.AddType(&SVGView{}, SVGViewProps)

var SVGViewProps = ki.Props{
	"EnumType:Flag": gi.KiT_VpFlags,
}

// AddNewSVGView adds a new editor to given parent node, with given name.
func AddNewSVGView(parent ki.Ki, name string, gv *GridView) *SVGView {
	sv := parent.AddNewChild(KiT_SVGView, name).(*SVGView)
	sv.GridView = gv
	sv.Scale = 1
	sv.Fill = true
	sv.SetProp("background-color", "white")
	sv.SetStretchMax()
	return sv
}

func (g *SVGView) CopyFieldsFrom(frm interface{}) {
	fr := frm.(*SVGView)
	g.SVG.CopyFieldsFrom(&fr.SVG)
	g.Trans = fr.Trans
	g.Scale = fr.Scale
	g.SetDragCursor = fr.SetDragCursor
}

func (g *SVGView) EditState() *EditState {
	return &g.GridView.EditState
}

func (sv *SVGView) MouseDrag() {
	sv.ConnectEvent(oswin.MouseDragEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.DragEvent)
		me.SetProcessed()
		ssvg := recv.Embed(KiT_SVGView).(*SVGView)
		if ssvg.IsDragging() {
			if !ssvg.SetDragCursor {
				oswin.TheApp.Cursor(ssvg.ParentWindow().OSWin).Push(cursor.HandOpen)
				ssvg.SetDragCursor = true
			}
			ssvg.DragEvent(me)
		} else {
			if ssvg.SetDragCursor {
				oswin.TheApp.Cursor(ssvg.ParentWindow().OSWin).Pop()
				ssvg.SetDragCursor = false
			}
		}

	})
}

func (sv *SVGView) MouseScroll() {
	sv.ConnectEvent(oswin.MouseScrollEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.ScrollEvent)
		me.SetProcessed()
		ssvg := recv.Embed(KiT_SVGView).(*SVGView)
		if ssvg.SetDragCursor {
			oswin.TheApp.Cursor(ssvg.ParentWindow().OSWin).Pop()
			ssvg.SetDragCursor = false
		}
		ssvg.InitScale()
		ssvg.Scale += float32(me.NonZeroDelta(false)) / 20
		if ssvg.Scale <= 0 {
			ssvg.Scale = 0.01
		}
		ssvg.SetTransform()
		ssvg.SetFullReRender()
		ssvg.UpdateSelSprites()
		ssvg.UpdateSig()
	})
}

func (sv *SVGView) MouseEvent() {
	sv.ConnectEvent(oswin.MouseEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.Event)
		ssvg := recv.Embed(KiT_SVGView).(*SVGView)
		es := ssvg.EditState()
		if ssvg.SetDragCursor {
			oswin.TheApp.Cursor(ssvg.ParentWindow().OSWin).Pop()
			ssvg.SetDragCursor = false
		}
		if me.Action != mouse.Release {
			return
		}
		obj := ssvg.FirstContainingPoint(me.Where, true)
		if obj != nil {
			sob := obj.(svg.NodeSVG)
			switch {
			case me.Button == mouse.Right:
				me.SetProcessed()
				giv.StructViewDialog(ssvg.Viewport, obj, giv.DlgOpts{Title: "SVG Element View"}, nil, nil)
			case es.Tool == SelectTool:
				me.SetProcessed()
				es.SelectAction(sob, me.SelectMode())
				ssvg.UpdateSelSprites()
				ssvg.EditState().DragStart()
			}
		} else {
			switch {
			case es.Tool == SelectTool:
				me.SetProcessed()
				es.ResetSelected()
				ssvg.UpdateSelSprites()
			}
		}
	})
}

func (sv *SVGView) MouseHover() {
	sv.ConnectEvent(oswin.MouseHoverEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.HoverEvent)
		me.SetProcessed()
		ssvg := recv.Embed(KiT_SVGView).(*SVGView)
		obj := ssvg.FirstContainingPoint(me.Where, true)
		if obj != nil {
			pos := me.Where
			ttxt := fmt.Sprintf("element name: %v -- use right mouse click to edit", obj.Name())
			gi.PopupTooltip(obj.Name(), pos.X, pos.Y, sv.ViewportSafe(), ttxt)
		}
	})
}

func (sv *SVGView) SVGViewEvents() {
	sv.MouseDrag()
	sv.MouseScroll()
	sv.MouseEvent()
	sv.MouseHover()
}

func (sv *SVGView) ConnectEvents2D() {
	sv.SVGViewEvents()
}

// InitScale ensures that Scale is initialized and non-zero
func (sv *SVGView) InitScale() {
	if sv.Scale == 0 {
		mvp := sv.ViewportSafe()
		if mvp != nil {
			sv.Scale = sv.ParentWindow().LogicalDPI() / 96.0
		} else {
			sv.Scale = 1
		}
	}
}

// SetTransform sets the transform based on Trans and Scale values
func (sv *SVGView) SetTransform() {
	sv.InitScale()
	sv.SetProp("transform", fmt.Sprintf("translate(%v,%v) scale(%v,%v)", sv.Trans.X, sv.Trans.Y, sv.Scale, sv.Scale))
}

func (sv *SVGView) Render2D() {
	if sv.PushBounds() {
		rs := &sv.Render
		sv.This().(gi.Node2D).ConnectEvents2D()
		if sv.Fill {
			sv.FillViewport()
		}
		if sv.Norm {
			sv.SetNormXForm()
		}
		rs.PushXForm(sv.Pnt.XForm)
		sv.Render2DChildren() // we must do children first, then us!
		sv.PopBounds()
		rs.PopXForm()
		// fmt.Printf("geom.bounds: %v  geom: %v\n", svg.Geom.Bounds(), svg.Geom)
		sv.RenderViewport2D() // update our parent image
	}
}

/////////////////////////////////////////////////
// selection processing

func (sv *SVGView) UpdateSelSprites() {
	win := sv.GridView.ParentWindow()
	updt := win.UpdateStart()
	defer win.UpdateEnd(updt)

	es := sv.EditState()
	es.UpdateSelBBox()
	if !es.HasSelected() {
		InactivateSprites(win)
		win.RenderOverlays()
		return
	}
	sp := SpriteConnectEvent(SizeUpL, win, sv.This(), func(recv, send ki.Ki, sig int64, d interface{}) {
		ssvg := recv.Embed(KiT_SVGView).(*SVGView)
		ssvg.SpriteEvent(SizeUpL, oswin.EventType(sig), d)
	})
	es.ActiveSprites[SizeUpL] = sp

	sp = SpriteConnectEvent(SizeUpR, win, sv.This(), func(recv, send ki.Ki, sig int64, d interface{}) {
		ssvg := recv.Embed(KiT_SVGView).(*SVGView)
		ssvg.SpriteEvent(SizeUpR, oswin.EventType(sig), d)
	})
	es.ActiveSprites[SizeUpR] = sp

	sp = SpriteConnectEvent(SizeDnL, win, sv.This(), func(recv, send ki.Ki, sig int64, d interface{}) {
		ssvg := recv.Embed(KiT_SVGView).(*SVGView)
		ssvg.SpriteEvent(SizeDnL, oswin.EventType(sig), d)
	})
	es.ActiveSprites[SizeDnL] = sp

	sp = SpriteConnectEvent(SizeDnR, win, sv.This(), func(recv, send ki.Ki, sig int64, d interface{}) {
		ssvg := recv.Embed(KiT_SVGView).(*SVGView)
		ssvg.SpriteEvent(SizeDnR, oswin.EventType(sig), d)
	})
	es.ActiveSprites[SizeDnR] = sp

	sv.SetSelSprites(es.SelBBox)

	win.RenderOverlays()
}

// SetSelSprites sets active selection sprite locations based on given bounding box
func (sv *SVGView) SetSelSprites(bbox mat32.Box2) {
	es := sv.EditState()
	SetSpritePos(SizeUpL, es.ActiveSprites[SizeUpL], image.Point{int(bbox.Min.X), int(bbox.Min.Y)})
	SetSpritePos(SizeUpR, es.ActiveSprites[SizeUpR], image.Point{int(bbox.Max.X), int(bbox.Min.Y)})
	SetSpritePos(SizeDnL, es.ActiveSprites[SizeDnL], image.Point{int(bbox.Min.X), int(bbox.Max.Y)})
	SetSpritePos(SizeDnR, es.ActiveSprites[SizeDnR], image.Point{int(bbox.Max.X), int(bbox.Max.Y)})
}

func (sv *SVGView) SpriteEvent(sp Sprites, et oswin.EventType, d interface{}) {
	win := sv.GridView.ParentWindow()
	switch et {
	case oswin.MouseEvent:
		me := d.(*mouse.Event)
		me.SetProcessed()
		// fmt.Printf("click %s\n", sp)
		if me.Action == mouse.Press {
			win.SpriteDragging = SpriteNames[sp]
			sv.EditState().DragStart()
			// fmt.Printf("dragging: %s\n", win.SpriteDragging)
		} else if me.Action == mouse.Release {
			sv.UpdateSig()
		}
	case oswin.MouseDragEvent:
		me := d.(*mouse.DragEvent)
		me.SetProcessed()
		// fmt.Printf("drag %v delta: %v\n", sp, me.Delta())
		sv.SpriteDrag(sp, me.Delta(), win)
	}
}

// DragEvent processes a mouse drag event on the SVG canvas
func (sv *SVGView) DragEvent(me *mouse.DragEvent) {
	delta := me.Where.Sub(me.From)
	dv := mat32.NewVec2FmPoint(delta)
	es := sv.EditState()
	if es.HasSelected() {
		win := sv.GridView.ParentWindow()
		es.DragCurBBox.Min.SetAdd(dv)
		es.DragCurBBox.Max.SetAdd(dv)
		tdel := es.DragCurBBox.Min.Sub(es.DragStartBBox.Min)
		xf := mat32.Identity2D()
		xf.X0 = tdel.X
		xf.Y0 = tdel.Y
		for itm, ss := range es.Selected {
			itm.ReadGeom(ss.InitGeom)
			itm.ApplyDeltaXForm(xf)
		}
		sv.SetSelSprites(es.DragCurBBox)
		sv.UpdateSig()
		win.RenderOverlays()
	} else {
		sv.Trans.SetAdd(dv)
		sv.SetTransform()
		sv.SetFullReRender()
		sv.UpdateSig()
	}
}

// SpriteDrag processes a mouse drag event on a selection sprite
func (sv *SVGView) SpriteDrag(sp Sprites, delta image.Point, win *gi.Window) {
	es := sv.EditState()
	stsz := es.DragStartBBox.Size()
	stpos := es.DragStartBBox.Min
	dv := mat32.NewVec2FmPoint(delta)
	switch sp {
	case SizeUpL:
		es.DragCurBBox.Min.SetAdd(dv)
	case SizeUpR:
		es.DragCurBBox.Min.Y += dv.Y
		es.DragCurBBox.Max.X += dv.X
	case SizeDnL:
		es.DragCurBBox.Min.X += dv.X
		es.DragCurBBox.Max.Y += dv.Y
	case SizeDnR:
		es.DragCurBBox.Max.SetAdd(dv)
	}
	es.DragCurBBox.Min.SetMin(es.DragCurBBox.Max.SubScalar(1)) // don't allow flipping
	npos := es.DragCurBBox.Min
	nsz := es.DragCurBBox.Size()
	xf := mat32.Identity2D()
	xf.XX = nsz.X / stsz.X
	xf.YY = nsz.Y / stsz.Y
	xf.X0 = npos.X - stpos.X
	xf.Y0 = npos.Y - stpos.Y
	for itm, ss := range es.Selected {
		itm.ReadGeom(ss.InitGeom)
		itm.ApplyDeltaXForm(xf)
	}

	sv.SetSelSprites(es.DragCurBBox)

	sv.UpdateSig()
	win.RenderOverlays()
}
