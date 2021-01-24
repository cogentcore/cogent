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
			del := me.Where.Sub(me.From)
			ssvg.Trans.X += float32(del.X)
			ssvg.Trans.Y += float32(del.Y)
			ssvg.SetTransform()
			ssvg.SetFullReRender()
			ssvg.UpdateSig()
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
			switch {
			case me.Button == mouse.Right:
				me.SetProcessed()
				giv.StructViewDialog(ssvg.Viewport, obj, giv.DlgOpts{Title: "SVG Element View"}, nil, nil)
			case es.Tool == SelectTool:
				me.SetProcessed()
				es.SelectAction(obj, me.SelectMode())
				ssvg.UpdateSelects()
			}
		} else {
			switch {
			case es.Tool == SelectTool:
				me.SetProcessed()
				es.ResetSelected()
				ssvg.UpdateSelects()
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
		sv.UpdateSelects()
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

func (sv *SVGView) UpdateSelects() {
	win := sv.GridView.ParentWindow()
	updt := win.UpdateStart()
	defer win.UpdateEnd(updt)

	es := sv.EditState()
	sls := es.SelectedList(false)
	if len(sls) == 0 {
		InactivateSprites(win)
		win.RenderOverlays()
		return
	}
	bbox := mat32.Box2{}
	bbox.SetEmpty()
	for _, itm := range sls {
		// fmt.Printf("%d:\t%s\n", i, itm.Name())
		bb := mat32.Box2{}
		_, gi := gi.KiToNode2D(itm)
		bb.Min.Set(float32(gi.WinBBox.Min.X), float32(gi.WinBBox.Min.Y))
		bb.Max.Set(float32(gi.WinBBox.Max.X), float32(gi.WinBBox.Max.Y))
		bbox.ExpandByBox(bb)
	}
	SetSpritePos(SizeUpL, image.Point{int(bbox.Min.X), int(bbox.Min.Y)}, win, sv.This(),
		func(recv, send ki.Ki, sig int64, d interface{}) {
			ssvg := recv.Embed(KiT_SVGView).(*SVGView)
			ssvg.SpriteEvent(SizeUpL, oswin.EventType(sig), d)
		})
	SetSpritePos(SizeUpR, image.Point{int(bbox.Max.X), int(bbox.Min.Y)}, win, sv.This(),
		func(recv, send ki.Ki, sig int64, d interface{}) {
			ssvg := recv.Embed(KiT_SVGView).(*SVGView)
			ssvg.SpriteEvent(SizeUpR, oswin.EventType(sig), d)
		})
	SetSpritePos(SizeDnL, image.Point{int(bbox.Min.X), int(bbox.Max.Y)}, win, sv.This(),
		func(recv, send ki.Ki, sig int64, d interface{}) {
			ssvg := recv.Embed(KiT_SVGView).(*SVGView)
			ssvg.SpriteEvent(SizeDnL, oswin.EventType(sig), d)
		})
	SetSpritePos(SizeDnR, image.Point{int(bbox.Max.X), int(bbox.Max.Y)}, win, sv.This(),
		func(recv, send ki.Ki, sig int64, d interface{}) {
			ssvg := recv.Embed(KiT_SVGView).(*SVGView)
			ssvg.SpriteEvent(SizeDnR, oswin.EventType(sig), d)
		})
	win.RenderOverlays()
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

func (sv *SVGView) SpriteDrag(sp Sprites, delta image.Point, win *gi.Window) {
	es := sv.EditState()
	sls := es.SelectedList(false)
	/*
		// could use this to directly update control points:
			switch sp {
			case SizeUpL:
				ActiveSprites[SizeUpL].Geom.Pos.X += delta.X
				ActiveSprites[SizeUpL].Geom.Pos.Y += delta.Y
				ActiveSprites[SizeDnL].Geom.Pos.X += delta.X
				ActiveSprites[SizeUpR].Geom.Pos.Y += delta.Y
			case SizeUpR:
				ActiveSprites[SizeUpR].Geom.Pos.X += delta.X
				ActiveSprites[SizeUpR].Geom.Pos.Y += delta.Y
				ActiveSprites[SizeDnR].Geom.Pos.X += delta.X
				ActiveSprites[SizeUpL].Geom.Pos.Y += delta.Y
			case SizeDnL:
				ActiveSprites[SizeDnL].Geom.Pos.X += delta.X
				ActiveSprites[SizeDnL].Geom.Pos.Y += delta.Y
				ActiveSprites[SizeUpL].Geom.Pos.X += delta.X
				ActiveSprites[SizeDnR].Geom.Pos.Y += delta.Y
			case SizeDnR:
				ActiveSprites[SizeDnR].Geom.Pos.X += delta.X
				ActiveSprites[SizeDnR].Geom.Pos.Y += delta.Y
				ActiveSprites[SizeUpR].Geom.Pos.X += delta.X
				ActiveSprites[SizeDnL].Geom.Pos.Y += delta.Y
			}
	*/
	// todo: need to convert delta to local coords for obj
	del := mat32.Vec2{float32(delta.X), float32(delta.Y)}
	for _, itm := range sls {
		switch ob := itm.(type) {
		case *svg.Rect:
			switch sp {
			case SizeUpL:
				ob.Size.SetAdd(mat32.Vec2{-del.X, -del.Y})
				ob.Pos.SetAdd(del)
			case SizeUpR:
				ob.Size.SetAdd(mat32.Vec2{del.X, -del.Y})
				ob.Pos.SetAdd(mat32.Vec2{0, del.Y})
			case SizeDnL:
				ob.Size.SetAdd(mat32.Vec2{-del.X, del.Y})
				ob.Pos.SetAdd(mat32.Vec2{del.X, 0})
			case SizeDnR:
				ob.Size.SetAdd(del)
			}
		}
	}
	sv.UpdateSig()
	// win.RenderOverlays()
}
