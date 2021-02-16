// Copyright (c) 2021, The Grid Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grid

import (
	"bytes"
	"fmt"
	"image"
	"reflect"
	"strings"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/cursor"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/gi/svg"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
)

// SVGView is the element for viewing, interacting with the SVG
type SVGView struct {
	svg.SVG
	GridView      *GridView  `copy:"-" json:"-" xml:"-" view:"-" desc:"the parent gridview"`
	Trans         mat32.Vec2 `desc:"view translation offset (from dragging)"`
	Scale         float32    `desc:"view scaling (from zooming)"`
	SetDragCursor bool       `view:"-" desc:"has dragging cursor been set yet?"`
}

var KiT_SVGView = kit.Types.AddType(&SVGView{}, SVGViewProps)

var SVGViewProps = ki.Props{
	"EnumType:Flag": svg.KiT_SVGFlags,
}

// AddNewSVGView adds a new editor to given parent node, with given name.
func AddNewSVGView(parent ki.Ki, name string, gv *GridView) *SVGView {
	sv := parent.AddNewChild(KiT_SVGView, name).(*SVGView)
	sv.GridView = gv
	sv.Scale = 1
	sv.Fill = true
	sv.Norm = false
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

// EditState returns the EditState for this view
func (sv *SVGView) EditState() *EditState {
	return &sv.GridView.EditState
}

// UpdateView updates the view, optionally with a full re-render
func (sv *SVGView) UpdateView(full bool) {
	if full {
		sv.SetFullReRender()
	}
	sv.UpdateSig()
	sv.UpdateSelSprites()
}

func (sv *SVGView) SVGViewKeys(kt *key.ChordEvent) {
	kc := kt.Chord()
	if gi.KeyEventTrace {
		fmt.Printf("SVGView KeyInput: %v\n", sv.Path())
	}
	kf := gi.KeyFun(kc)
	switch kf {
	case gi.KeyFunAbort:
		// todo: maybe something else
		kt.SetProcessed()
		sv.GridView.SetTool(SelectTool)
	case gi.KeyFunUndo:
		kt.SetProcessed()
		sv.GridView.Undo()
	case gi.KeyFunRedo:
		kt.SetProcessed()
		sv.GridView.Redo()
	case gi.KeyFunCopy:
		kt.SetProcessed()
		// todo: gridview treeview copy
		// sv.GridView.Redo()
	}
	if kt.IsProcessed() {
		return
	}
	switch kc {
	case "s", "S", " ":
		kt.SetProcessed()
		sv.GridView.SetTool(SelectTool)
	case "R", "r":
		kt.SetProcessed()
		sv.GridView.SetTool(RectTool)
	case "E", "e":
		kt.SetProcessed()
		sv.GridView.SetTool(EllipseTool)
	case "B", "b":
		kt.SetProcessed()
		sv.GridView.SetTool(BezierTool)
	}
}

func (sv *SVGView) KeyChordEvent() {
	// need hipri to prevent 2-seq guys from being captured by others
	sv.ConnectEvent(oswin.KeyChordEvent, gi.HiPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		svv := recv.Embed(KiT_SVGView).(*SVGView)
		kt := d.(*key.ChordEvent)
		svv.SVGViewKeys(kt)
	})
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
			ssvg.DragEvent(me) // for both scene drag and
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
		ssvg.UpdateView(true)
	})
}

func (sv *SVGView) MouseEvent() {
	sv.ConnectEvent(oswin.MouseEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.Event)
		ssvg := recv.Embed(KiT_SVGView).(*SVGView)
		ssvg.GrabFocus()
		es := ssvg.EditState()
		if ssvg.SetDragCursor {
			oswin.TheApp.Cursor(ssvg.ParentWindow().OSWin).Pop()
			ssvg.SetDragCursor = false
		}
		if me.Action != mouse.Release {
			return
		}
		if es.InAction() {
			ssvg.ManipDone()
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
				ssvg.GridView.UpdateTabs()
				ssvg.UpdateSelSprites()
				ssvg.EditState().DragStart()
			}
		} else {
			// for any tool
			me.SetProcessed()
			es.ResetSelected()
			ssvg.UpdateSelSprites()
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
			sv.UpdateSelSprites()
			sv.EditState().DragStart()
			sv.ManipDone()
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
	win := sv.GridView.ParentWindow()
	delta := me.Where.Sub(me.From)
	dv := mat32.NewVec2FmPoint(delta)
	es := sv.EditState()
	svoff := mat32.NewVec2FmPoint(sv.WinBBox.Min)
	me.SetProcessed()
	if me.HasAnyModifier(key.Shift) {
		sv.Trans.SetAdd(dv)
		sv.SetTransform()
		sv.UpdateView(true)
		return
	}
	if es.HasSelected() {
		es.DragCurBBox.Min.SetAdd(dv)
		es.DragCurBBox.Max.SetAdd(dv)
		if !es.InAction() {
			sv.ManipStart("Move", es.SelectedNamesString())
		}
		pt := es.DragStartBBox.Min.Sub(svoff)
		tdel := es.DragCurBBox.Min.Sub(es.DragStartBBox.Min)
		for itm, ss := range es.Selected {
			itm.ReadGeom(ss.InitGeom)
			itm.ApplyDeltaXForm(tdel, mat32.Vec2{1, 1}, 0, pt)
		}
		sv.SetSelSprites(es.DragCurBBox)
		go sv.ManipUpdate()
		win.RenderOverlays()
	} else {
		if !es.InAction() {
			switch es.Tool {
			case SelectTool:
				// rubberband select
			case RectTool:
				sv.NewElDrag(svg.KiT_Rect, me.From, me.Where)
			case EllipseTool:
				sv.NewElDrag(svg.KiT_Ellipse, me.From, me.Where)
			}
		}
	}
}

// SpriteDrag processes a mouse drag event on a selection sprite
func (sv *SVGView) SpriteDrag(sp Sprites, delta image.Point, win *gi.Window) {
	es := sv.EditState()
	if !es.InAction() {
		sv.ManipStart("Reshape", es.SelectedNamesString())
	}
	stsz := es.DragStartBBox.Size()
	stpos := es.DragStartBBox.Min
	dv := mat32.NewVec2FmPoint(delta)
	switch sp {
	case SizeUpL:
		es.DragCurBBox.Min.SetAdd(dv)
	case SizeUpM:
		es.DragCurBBox.Min.Y += dv.Y
	case SizeUpR:
		es.DragCurBBox.Min.Y += dv.Y
		es.DragCurBBox.Max.X += dv.X
	case SizeDnL:
		es.DragCurBBox.Min.X += dv.X
		es.DragCurBBox.Max.Y += dv.Y
	case SizeDnM:
		es.DragCurBBox.Max.Y += dv.Y
	case SizeDnR:
		es.DragCurBBox.Max.SetAdd(dv)
	case SizeLfC:
		es.DragCurBBox.Min.X += dv.X
	case SizeRtC:
		es.DragCurBBox.Max.X += dv.X
	}
	es.DragCurBBox.Min.SetMin(es.DragCurBBox.Max.SubScalar(1)) // don't allow flipping
	npos := es.DragCurBBox.Min
	nsz := es.DragCurBBox.Size()
	svoff := mat32.NewVec2FmPoint(sv.WinBBox.Min)
	pt := es.DragStartBBox.Min.Sub(svoff)
	del := npos.Sub(stpos)
	sc := nsz.Div(stsz)
	for itm, ss := range es.Selected {
		itm.ReadGeom(ss.InitGeom)
		itm.ApplyDeltaXForm(del, sc, 0, pt)
		if strings.HasPrefix(es.Action, "New") {
			svg.UpdateNodeGradientPoints(itm, "fill")
			svg.UpdateNodeGradientPoints(itm, "stroke")
		}
	}

	sv.SetSelSprites(es.DragCurBBox)

	go sv.ManipUpdate()

	win.RenderOverlays()
}

// ManipStart is called at the start of a manipulation, saving the state prior to the action
func (sv *SVGView) ManipStart(act, data string) {
	es := sv.EditState()
	es.ActStart(act, data)
	// astr := act + ": " +
	// sv.GridView.SetStatus(fmt.Sprintf("save undo: %s: %s", act, data))
	sv.UndoSave(act, data)
	es.ActUnlock()
}

// ManipDone happens when a manipulation has finished: resets action, does render
func (sv *SVGView) ManipDone() {
	es := sv.EditState()
	es.ActDone()
	sv.UpdateSig()
}

// ManipUpdate is called from goroutine: 'go sv.ManipUpdate()' to update the
// current display while manipulating.  It checks if already rendering and if so,
// just returns immediately, so that updates are not stacked up and laggy.
func (sv *SVGView) ManipUpdate() {
	if sv.IsRendering() {
		return
	}
	sv.UpdateSig()
}

func (sv *SVGView) SVGViewEvents() {
	sv.SetCanFocus()
	sv.MouseDrag()
	sv.MouseScroll()
	sv.MouseEvent()
	sv.MouseHover()
	sv.KeyChordEvent()
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

/////////////////////////////////////////////////
// Undo

// UndoSave save current state for potential undo
func (sv *SVGView) UndoSave(action, data string) {
	es := sv.EditState()
	b := &bytes.Buffer{}
	// sv.WriteXML(b, false)
	err := sv.WriteJSON(b, true) // should be false
	if err != nil {
		fmt.Printf("SaveUndo Error: %s\n", err)
	}
	// fmt.Printf("%s\n", string(b.Bytes()))
	bs := strings.Split(string(b.Bytes()), "\n")
	es.UndoMgr.Save(action, data, bs)
	// fmt.Println(es.UndoMgr.MemStats(true))
}

// UndoSaveReplace save current state to replace current
func (sv *SVGView) UndoSaveReplace(action, data string) {
	es := sv.EditState()
	b := &bytes.Buffer{}
	// sv.WriteXML(b, false)
	err := sv.WriteJSON(b, true) // should be false
	if err != nil {
		fmt.Printf("SaveUndo Error: %s\n", err)
	}
	bs := strings.Split(string(b.Bytes()), "\n")
	es.UndoMgr.SaveReplace(action, data, bs)
	// fmt.Println(es.UndoMgr.MemStats(true))
}

// Undo undoes one step, returning the action that was undone
func (sv *SVGView) Undo() string {
	es := sv.EditState()
	if es.UndoMgr.MustSaveUndoStart() { // need to save current state!
		b := &bytes.Buffer{}
		// sv.WriteXML(b, false)
		err := sv.WriteJSON(b, false)
		if err != nil {
			fmt.Printf("SaveUndo Error: %s\n", err)
		}
		bs := strings.Split(string(b.Bytes()), "\n")
		es.UndoMgr.SaveUndoStart(bs)
	}
	// fmt.Printf("undo idx: %d\n", es.UndoMgr.Idx)
	act, _, state := es.UndoMgr.Undo()
	if state == nil {
		return act
	}
	sb := strings.Join(state, "\n")
	b := bytes.NewBufferString(sb)
	updt := sv.UpdateStart()
	err := sv.ReadJSON(b)
	_ = err
	// if err != nil {
	// 	fmt.Printf("Undo load Error: %s\n", err)
	// }
	sv.UpdateEnd(updt)
	sv.UpdateSelSprites()
	return act
}

// Redo redoes one step, returning the action that was redone
func (sv *SVGView) Redo() string {
	es := sv.EditState()
	// fmt.Printf("redo idx: %d\n", es.UndoMgr.Idx)
	act, _, state := es.UndoMgr.Redo()
	if state == nil {
		return act
	}
	sb := strings.Join(state, "\n")
	b := bytes.NewBufferString(sb)
	// sv.ReadXML(b)
	updt := sv.UpdateStart()
	err := sv.ReadJSON(b) // json preserves all objects
	_ = err
	// if err != nil {
	// 	fmt.Printf("Redo load Error: %s\n", err)
	// }
	sv.UpdateEnd(updt)
	sv.UpdateSelSprites()
	return act
}

///////////////////////////////////////////////////////////////////
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

	for i := SizeUpL; i <= SizeRtC; i++ {
		spi := i // key to get a unique local var
		sp := SpriteConnectEvent(spi, win, sv.This(), func(recv, send ki.Ki, sig int64, d interface{}) {
			ssvg := recv.Embed(KiT_SVGView).(*SVGView)
			ssvg.SpriteEvent(spi, oswin.EventType(sig), d)
		})
		es.ActiveSprites[spi] = sp
	}
	sv.SetSelSprites(es.SelBBox)

	win.RenderOverlays()
}

// SetSelSprites sets active selection sprite locations based on given bounding box
func (sv *SVGView) SetSelSprites(bbox mat32.Box2) {
	es := sv.EditState()
	spsz := float32(SpriteSize())
	midX := int(0.5 * (bbox.Min.X + bbox.Max.X - spsz))
	midY := int(0.5 * (bbox.Min.Y + bbox.Max.Y - spsz))
	SetSpritePos(SizeUpL, es.ActiveSprites[SizeUpL], image.Point{int(bbox.Min.X), int(bbox.Min.Y)})
	SetSpritePos(SizeUpM, es.ActiveSprites[SizeUpM], image.Point{midX, int(bbox.Min.Y)})
	SetSpritePos(SizeUpR, es.ActiveSprites[SizeUpR], image.Point{int(bbox.Max.X), int(bbox.Min.Y)})
	SetSpritePos(SizeDnL, es.ActiveSprites[SizeDnL], image.Point{int(bbox.Min.X), int(bbox.Max.Y)})
	SetSpritePos(SizeDnM, es.ActiveSprites[SizeDnM], image.Point{midX, int(bbox.Max.Y)})
	SetSpritePos(SizeDnR, es.ActiveSprites[SizeDnR], image.Point{int(bbox.Max.X), int(bbox.Max.Y)})
	SetSpritePos(SizeLfC, es.ActiveSprites[SizeLfC], image.Point{int(bbox.Min.X), midY})
	SetSpritePos(SizeRtC, es.ActiveSprites[SizeRtC], image.Point{int(bbox.Max.X), midY})
}

///////////////////////////////////////////////////////////////////////
// New objects

// NewEl makes a new SVG element, giving it a new unique name.
// Uses currently active layer if set.
func (sv *SVGView) NewEl(typ reflect.Type) svg.NodeSVG {
	es := sv.EditState()
	par := sv.This()
	if es.CurLayer != "" {
		ly := sv.ChildByName(es.CurLayer, 1)
		if ly != nil {
			par = ly
		}
	}
	nwid := sv.NewUniqueId()
	nwnm := fmt.Sprintf("%s%d", typ.Name(), nwid)
	par.SetChildAdded()
	nw := par.AddNewChild(typ, nwnm).(svg.NodeSVG)
	nwnm = fmt.Sprintf("%s%d", nw.SVGName(), nwid)
	nw.SetName(nwnm)
	sv.GridView.PaintView().SetProps(nw)
	return nw
}

// NewElDrag makes a new SVG element during the drag operation
func (sv *SVGView) NewElDrag(typ reflect.Type, start, end image.Point) svg.NodeSVG {
	win := sv.GridView.ParentWindow()
	es := sv.EditState()
	tn := typ.Name()
	sv.ManipStart("New"+tn, "")
	updt := sv.UpdateStart()
	sv.SetFullReRender()
	nr := sv.NewEl(typ)
	xfi := sv.Pnt.XForm.Inverse()
	svoff := mat32.NewVec2FmPoint(sv.WinBBox.Min)
	pos := mat32.NewVec2FmPoint(start).Sub(svoff)
	dv := mat32.NewVec2FmPoint(end.Sub(start))
	minsz := float32(20)
	pos.SetSubScalar(minsz)
	nr.SetPos(xfi.MulVec2AsPt(pos))
	sz := dv.Abs().Max(mat32.NewVec2Scalar(minsz / 2))
	nr.SetSize(xfi.MulVec2AsVec(sz))
	es.SelectAction(nr, mouse.SelectOne)
	sv.UpdateEnd(updt)
	sv.EditState().DragStart()
	sv.UpdateSelSprites()
	win.SpriteDragging = SpriteNames[SizeDnR]
	return nr
}

///////////////////////////////////////////////////////////////////////
// Gradients

// Gradients returns the currently-defined gradients with stops
// that are shared among obj-specific ones
func (sv *SVGView) Gradients() []*Gradient {
	gl := make([]*Gradient, 0)
	for _, gii := range sv.Defs.Kids {
		g, ok := gii.(*gi.Gradient)
		if !ok {
			continue
		}
		if g.StopsName != "" {
			continue
		}
		gr := &Gradient{}
		gr.UpdateFromGrad(g)
		gl = append(gl, gr)
	}
	return gl
}

// UpdateGradients update SVG gradients from given gradient list
func (sv *SVGView) UpdateGradients(gl []*Gradient) {
	for _, gr := range gl {
		radial := false
		if strings.HasPrefix(gr.Name, "radial") {
			radial = true
		}
		var g *gi.Gradient
		gg := sv.FindDefByName(gr.Name)
		if gg == nil {
			g, _ = sv.AddNewGradient(radial)
		} else {
			g = gg.(*gi.Gradient)
		}
		gr.UpdateGrad(g)
	}
}
