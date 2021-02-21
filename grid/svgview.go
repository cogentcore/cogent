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
	"github.com/goki/ki/ints"
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
	case gi.KeyFunDuplicate:
		kt.SetProcessed()
		sv.GridView.DuplicateSelected()
	case gi.KeyFunCopy:
		kt.SetProcessed()
		sv.GridView.CopySelected()
	case gi.KeyFunCut:
		kt.SetProcessed()
		sv.GridView.CutSelected()
	case gi.KeyFunPaste:
		kt.SetProcessed()
		sv.GridView.PasteClip()
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
		if me.Action == mouse.Press {
			es.DragStartPos = me.Where
		}
		if me.Action != mouse.Release {
			return
		}
		if es.InAction() {
			ssvg.ManipDone()
			return
		}
		if me.Button == mouse.Right && es.HasSelected() {
			me.SetProcessed()
			obj := es.SelectedList(false)[0]
			ssvg.NodeContextMenu(obj, me.Where)
			return
		}
		obj := ssvg.SelectContainsPoint(me.Where, false)
		if obj != nil {
			sob := obj.(svg.NodeSVG)
			switch {
			case me.Button == mouse.Right:
				me.SetProcessed()
				ssvg.NodeContextMenu(obj, me.Where)
			case ToolDoesBasicSelect(es.Tool):
				me.SetProcessed()
				es.SelectAction(sob, me.SelectMode())
				ssvg.GridView.UpdateTabs()
				ssvg.UpdateSelSprites()
				ssvg.EditState().DragSelStart(me.Where)
				ssvg.GridView.UpdateSelectToolbar()
			}
		} else {
			// for any tool
			me.SetProcessed()
			es.ResetSelected()
			ssvg.UpdateSelSprites()
			ssvg.GridView.UpdateSelectToolbar()
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
			sv.EditState().DragSelStart(me.Where)
			// fmt.Printf("dragging: %s\n", win.SpriteDragging)
		} else if me.Action == mouse.Release {
			sv.UpdateSelSprites()
			sv.EditState().DragSelStart(me.Where)
			sv.ManipDone()
		}
	case oswin.MouseDragEvent:
		me := d.(*mouse.DragEvent)
		me.SetProcessed()
		// fmt.Printf("drag %v delta: %v\n", sp, me.Delta())
		if me.HasAnyModifier(key.Alt) {
			sv.SpriteRotateDrag(sp, me.Delta(), win)
		} else {
			sv.SpriteReshapeDrag(sp, me.Delta(), win, me)
		}
	}
}

// DragEvent processes a mouse drag event on the SVG canvas
func (sv *SVGView) DragEvent(me *mouse.DragEvent) {
	win := sv.GridView.ParentWindow()
	delta := me.Where.Sub(me.From)
	es := sv.EditState()
	me.SetProcessed()
	if me.HasAnyModifier(key.Shift) {
		if !sv.SetDragCursor {
			oswin.TheApp.Cursor(win.OSWin).Push(cursor.HandOpen)
			sv.SetDragCursor = true
		}
		sv.Trans.SetAdd(mat32.NewVec2FmPoint(delta))
		sv.SetTransform()
		sv.UpdateView(true)
		return
	}
	if es.HasSelected() {
		sv.DragMove(delta, win, me)
	} else {
		if !es.InAction() {
			switch es.Tool {
			case SelectTool:
				sv.SetRubberBand(me.From)
			case RectTool:
				sv.NewElDrag(svg.KiT_Rect, es.DragStartPos, me.Where)
			case EllipseTool:
				sv.NewElDrag(svg.KiT_Ellipse, es.DragStartPos, me.Where)
			}
		} else {
			switch {
			case es.Action == "BoxSelect":
				sv.SetRubberBand(me.Where)
			}
		}
	}
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

///////////////////////////////////////////////////////////////////////////
//  ContextMenu / Actions

// EditNode opens a structview editor on node
func (sv *SVGView) EditNode(kn ki.Ki) {
	giv.StructViewDialog(sv.Viewport, kn, giv.DlgOpts{Title: "SVG Element View"}, nil, nil)
}

// MakeNodeContextMenu makes the menu of options for context right click
func (sv *SVGView) MakeNodeContextMenu(m *gi.Menu, kn ki.Ki) {
	m.AddAction(gi.ActOpts{Label: "Edit"}, sv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		sv.EditNode(kn)
	})
	m.AddAction(gi.ActOpts{Label: "Select in Tree"}, sv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		sv.GridView.SelectNodeInTree(kn, mouse.SelectOne)
	})
	m.AddSeparator("sep-clip")
	m.AddAction(gi.ActOpts{Label: "Duplicate", ShortcutKey: gi.KeyFunDuplicate}, sv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		sv.GridView.DuplicateSelected()
	})
	m.AddAction(gi.ActOpts{Label: "Copy", ShortcutKey: gi.KeyFunCopy}, sv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		sv.GridView.CopySelected()
	})
	m.AddAction(gi.ActOpts{Label: "Cut", ShortcutKey: gi.KeyFunCut}, sv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		sv.GridView.CutSelected()
	})
	m.AddAction(gi.ActOpts{Label: "Paste", ShortcutKey: gi.KeyFunPaste}, sv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		sv.GridView.PasteClip()
	})
}

// ContextMenuPos returns position to use for context menu, based on input position
func (sv *SVGView) NodeContextMenuPos(pos image.Point) image.Point {
	if pos != image.ZP {
		return pos
	}
	pos.X = (sv.WinBBox.Min.X + sv.WinBBox.Max.X) / 2
	pos.Y = (sv.WinBBox.Min.Y + sv.WinBBox.Max.Y) / 2
	return pos
}

// NodeContextMenu pops up the right-click context menu for given node
func (sv *SVGView) NodeContextMenu(kn ki.Ki, pos image.Point) {
	var men gi.Menu
	sv.MakeNodeContextMenu(&men, kn)
	pos = sv.NodeContextMenuPos(pos)
	gi.PopupMenu(men, pos.X, pos.Y, sv.Viewport, "svNodeContextMenu")
}

///////////////////////////////////////////////////////////////////////////
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

	for i := ReshapeUpL; i <= ReshapeRtM; i++ {
		spi := i // key to get a unique local var
		sp := SpriteConnectEvent(spi, win, image.Point{}, sv.This(), func(recv, send ki.Ki, sig int64, d interface{}) {
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
	_, spsz := HandleSpriteSize()
	midX := int(0.5 * (bbox.Min.X + bbox.Max.X - float32(spsz.X)))
	midY := int(0.5 * (bbox.Min.Y + bbox.Max.Y - float32(spsz.Y)))
	SetSpritePos(ReshapeUpL, es.ActiveSprites[ReshapeUpL], image.Point{int(bbox.Min.X), int(bbox.Min.Y)})
	SetSpritePos(ReshapeUpC, es.ActiveSprites[ReshapeUpC], image.Point{midX, int(bbox.Min.Y)})
	SetSpritePos(ReshapeUpR, es.ActiveSprites[ReshapeUpR], image.Point{int(bbox.Max.X), int(bbox.Min.Y)})
	SetSpritePos(ReshapeDnL, es.ActiveSprites[ReshapeDnL], image.Point{int(bbox.Min.X), int(bbox.Max.Y)})
	SetSpritePos(ReshapeDnC, es.ActiveSprites[ReshapeDnC], image.Point{midX, int(bbox.Max.Y)})
	SetSpritePos(ReshapeDnR, es.ActiveSprites[ReshapeDnR], image.Point{int(bbox.Max.X), int(bbox.Max.Y)})
	SetSpritePos(ReshapeLfM, es.ActiveSprites[ReshapeLfM], image.Point{int(bbox.Min.X), midY})
	SetSpritePos(ReshapeRtM, es.ActiveSprites[ReshapeRtM], image.Point{int(bbox.Max.X), midY})
}

// SetRubberBand updates the rubber band postion
func (sv *SVGView) SetRubberBand(cur image.Point) {
	win := sv.GridView.ParentWindow()
	es := sv.EditState()

	if !es.InAction() {
		es.ActStart("BoxSelect", fmt.Sprintf("%v", es.DragStartPos))
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
	es.EnsureActiveSprites()
	rt := SpriteConnectEvent(RubberBandT, win, sz, nil, nil)
	rb := SpriteConnectEvent(RubberBandB, win, sz, nil, nil)
	rr := SpriteConnectEvent(RubberBandR, win, sz, nil, nil)
	rl := SpriteConnectEvent(RubberBandL, win, sz, nil, nil)
	SetSpritePos(RubberBandT, rt, bbox.Min)
	SetSpritePos(RubberBandB, rb, image.Point{bbox.Min.X, bbox.Max.Y})
	SetSpritePos(RubberBandR, rr, image.Point{bbox.Max.X, bbox.Min.Y})
	SetSpritePos(RubberBandL, rl, bbox.Min)

	win.RenderOverlays()
}

// ShowAlignMatches draws the align matches as given
// between BBox Min - Max.  typs are corresponding bounding box sources.
func (sv *SVGView) ShowAlignMatches(pts []image.Rectangle, typs []BBoxPoints) {
	win := sv.GridView.ParentWindow()
	es := sv.EditState()

	es.EnsureActiveSprites()
	sz := ints.MinInt(len(pts), 8)
	for i := 0; i < sz; i++ {
		spi := Sprites(int(AlignMatch1) + i)
		pt := pts[i].Canon()
		lsz := pt.Max.Sub(pt.Min)
		sp := SpriteConnectEvent(spi, win, lsz, nil, nil)
		sp.Props.Set("bbox", typs[i])
		SetSpritePos(spi, sp, pt.Min)
	}
}

///////////////////////////////////////////////////////////////////////
// New objects

// SetSVGName sets the name of the element to standard type + id name
func (sv *SVGView) SetSVGName(el svg.NodeSVG) {
	nwid := sv.NewUniqueId()
	nwnm := fmt.Sprintf("%s%d", el.SVGName(), nwid)
	el.SetName(nwnm)
}

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
	nwnm := fmt.Sprintf("%s_tmp_new_item_", typ.Name())
	par.SetChildAdded()
	nw := par.AddNewChild(typ, nwnm).(svg.NodeSVG)
	sv.SetSVGName(nw)
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
	sv.EditState().DragSelStart(start)
	sv.UpdateSelSprites()
	win.SpriteDragging = SpriteNames[ReshapeDnR]
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
