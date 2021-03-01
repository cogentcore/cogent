// Copyright (c) 2021, The Grid Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grid

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"reflect"
	"strings"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/girl"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/cursor"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/gi/svg"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ints"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
)

// SVGView is the element for viewing, interacting with the SVG
type SVGView struct {
	svg.SVG
	GridView      *GridView   `copy:"-" json:"-" xml:"-" view:"-" desc:"the parent gridview"`
	Trans         mat32.Vec2  `desc:"view translation offset (from dragging)"`
	Scale         float32     `desc:"view scaling (from zooming)"`
	Grid          float32     `desc:"grid spacing, in native ViewBox units"`
	GridEff       float32     `view:"inactive" desc:"effective grid spacing given Scale level"`
	SetDragCursor bool        `view:"-" desc:"has dragging cursor been set yet?"`
	BgPixels      *image.RGBA `copy:"-" json:"-" xml:"-" view:"-" desc:"background pixels, includes page outline and grid"`
	BgRender      girl.State  `copy:"-" json:"-" xml:"-" view:"-" desc:"render state for background rendering"`
	bgTrans       mat32.Vec2  `copy:"-" json:"-" xml:"-" view:"-" desc:"bg rendered translation"`
	bgScale       float32     `copy:"-" json:"-" xml:"-" view:"-" desc:"bg rendered scale"`
	bgGridEff     float32     `copy:"-" json:"-" xml:"-" view:"-" desc:"bg rendered grid"`
}

var KiT_SVGView = kit.Types.AddType(&SVGView{}, SVGViewProps)

var SVGViewProps = ki.Props{
	"EnumType:Flag":    svg.KiT_SVGFlags,
	"background-color": &Prefs.Colors.Background,
	"border-width":     units.NewDot(1),
}

// AddNewSVGView adds a new editor to given parent node, with given name.
func AddNewSVGView(parent ki.Ki, name string, gv *GridView) *SVGView {
	sv := parent.AddNewChild(KiT_SVGView, name).(*SVGView)
	sv.GridView = gv
	sv.Grid = Prefs.Size.Grid
	sv.Scale = 1
	sv.Fill = false // managed separately
	sv.Norm = false
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
	wupdt := sv.TopUpdateStart()
	defer sv.TopUpdateEnd(wupdt)
	if full || sv.BgNeedsUpdate() {
		sv.SetFullReRender()
	}
	sv.UpdateSig()
	if sv.BgNeedsUpdate() {
		sv.RenderBg()
	}
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
	case gi.KeyFunDelete, gi.KeyFunBackspace:
		kt.SetProcessed()
		sv.GridView.DeleteSelected()
	}
	if kt.IsProcessed() {
		return
	}
	// fmt.Println(kc)
	switch kc {
	case "Control+G", "Meta+G":
		kt.SetProcessed()
		sv.GridView.SelGroup()
	case "Shift+Control+G", "Shift+Meta+G":
		kt.SetProcessed()
		sv.GridView.SelUnGroup()
	case "s", "Shift+S", " ":
		kt.SetProcessed()
		sv.GridView.SetTool(SelectTool)
	case "n", "Shift+N":
		kt.SetProcessed()
		sv.GridView.SetTool(NodeTool)
	case "r", "Shift+R":
		kt.SetProcessed()
		sv.GridView.SetTool(RectTool)
	case "e", "Shift+E":
		kt.SetProcessed()
		sv.GridView.SetTool(EllipseTool)
	case "b", "Shift+B":
		kt.SetProcessed()
		sv.GridView.SetTool(BezierTool)
	case "t", "Shift+T":
		kt.SetProcessed()
		sv.GridView.SetTool(TextTool)
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
		delta := float32(me.NonZeroDelta(false)) / 50
		sv.ZoomAt(me.Where, delta)
		// ssvg.InitScale()
		// ssvg.Scale +=
		// if ssvg.Scale <= 0 {
		// 	ssvg.Scale = 0.01
		// }
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
		sob := ssvg.SelectContainsPoint(me.Where, false, true) // not leavesonly, yes exclude existing sels
		if me.Action == mouse.Press && me.Button == mouse.Left {
			me.SetProcessed()
			es.SelNoDrag = false
			es.DragStartPos = me.Where
			switch {
			case es.HasSelected() && es.SelBBox.ContainsPoint(mat32.NewVec2FmPoint(me.Where)):
				// note: this absorbs potential secondary selections within selection -- handled
				// on release below, if nothing else happened
				es.SelNoDrag = true
				ssvg.EditState().DragSelStart(me.Where)
			case sob != nil && es.Tool == SelectTool:
				es.SelectAction(sob, me.SelectMode(), me.Where)
				ssvg.EditState().DragSelStart(me.Where)
				ssvg.UpdateSelect()
			case sob != nil && es.Tool == NodeTool:
				es.SelectAction(sob, mouse.SelectOne, me.Where)
				ssvg.EditState().DragSelStart(me.Where)
				ssvg.UpdateNodeSprites()
			case sob == nil:
				es.ResetSelected()
				ssvg.UpdateSelect()
			}
		}
		if me.Action != mouse.Release {
			return
		}
		if es.InAction() {
			es.SelNoDrag = false
			es.NewTextMade = false
			ssvg.ManipDone()
			return
		}
		if me.Button == mouse.Left {
			// release on select -- do extended selection processing

			if (es.SelNoDrag && es.Tool == SelectTool) || (es.Tool != SelectTool && ToolDoesBasicSelect(es.Tool)) {
				es.SelNoDrag = false
				me.SetProcessed()
				if sob == nil {
					sob = ssvg.SelectContainsPoint(me.Where, false, false) // don't exclude existing sel
				}
				if sob != nil {
					es.SelectAction(sob, me.SelectMode(), me.Where)
					ssvg.UpdateSelect()
				}
			}
			return
		}
		if me.Button == mouse.Right {
			me.SetProcessed()
			if es.HasSelected() {
				fobj := es.FirstSelectedNode()
				if fobj != nil {
					ssvg.NodeContextMenu(fobj, me.Where)
				}
			} else if sob != nil {
				ssvg.NodeContextMenu(sob, me.Where)
			}
			return
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

// DragEvent processes a mouse drag event on the SVG canvas
func (sv *SVGView) DragEvent(me *mouse.DragEvent) {
	win := sv.GridView.ParentWindow()
	delta := me.Where.Sub(me.From)
	es := sv.EditState()
	es.SelNoDrag = false
	me.SetProcessed()
	if me.HasAnyModifier(key.Shift) {
		if !sv.SetDragCursor {
			oswin.TheApp.Cursor(win.OSWin).Push(cursor.HandOpen)
			sv.SetDragCursor = true
		}
		sv.Trans.SetAdd(mat32.NewVec2FmPoint(delta).DivScalar(sv.Scale))
		sv.SetTransform()
		sv.UpdateView(true)
		return
	}
	if es.HasSelected() {
		if !es.NewTextMade {
			sv.DragMove(win, me) // in manip
		}
	} else {
		if !es.InAction() {
			switch es.Tool {
			case SelectTool:
				sv.SetRubberBand(me.From)
			case RectTool:
				sv.NewElDrag(svg.KiT_Rect, es.DragStartPos, me.Where)
			case EllipseTool:
				sv.NewElDrag(svg.KiT_Ellipse, es.DragStartPos, me.Where)
			case TextTool:
				sv.NewText(es.DragStartPos, me.Where)
				es.NewTextMade = true
			case BezierTool:
				sv.NewPath(es.DragStartPos, me.Where)
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

// ContentsBBox returns the object-level box of the entire contents
func (sv *SVGView) ContentsBBox() mat32.Box2 {
	bbox := mat32.Box2{}
	bbox.SetEmpty()
	sv.FuncDownMeFirst(0, nil, func(k ki.Ki, level int, d interface{}) bool {
		if k.This() == sv.This() {
			return ki.Continue
		}
		if k.This() == sv.Defs.This() {
			return ki.Break
		}
		sni, issv := k.(svg.NodeSVG)
		if !issv {
			return ki.Break
		}
		if NodeIsLayer(k) {
			return ki.Continue
		}
		if txt, istxt := sni.(*svg.Text); istxt { // no tspans
			if txt.Text != "" {
				return ki.Break
			}
		}
		sn := sni.AsSVGNode()
		bb := mat32.Box2{}
		bb.SetFromRect(sn.BBox)
		bbox.ExpandByBox(bb)
		if _, isgp := sni.(*svg.Group); isgp { // subsumes all
			return ki.Break
		}
		return ki.Continue
	})
	if bbox.IsEmpty() {
		bbox = mat32.Box2{}
	}
	return bbox
}

// XFormAllLeaves transforms all the leaf items in the drawing (not groups)
// uses ApplyDeltaXForm manipulation.
func (sv *SVGView) XFormAllLeaves(trans mat32.Vec2, scale mat32.Vec2, rot float32, pt mat32.Vec2) {
	sv.FuncDownMeFirst(0, nil, func(k ki.Ki, level int, d interface{}) bool {
		if k.This() == sv.This() {
			return ki.Continue
		}
		if k.This() == sv.Defs.This() {
			return ki.Break
		}
		sni, issv := k.(svg.NodeSVG)
		if !issv {
			return ki.Break
		}
		if NodeIsLayer(k) {
			return ki.Continue
		}
		if _, isgp := sni.(*svg.Group); isgp {
			return ki.Continue
		}
		if txt, istxt := sni.(*svg.Text); istxt { // no tspans
			if txt.Text != "" {
				return ki.Break
			}
		}
		sni.ApplyDeltaXForm(trans, scale, rot, pt)
		return ki.Continue
	})
}

// ZoomToPage sets the scale to fit the current viewbox
func (sv *SVGView) ZoomToPage(width bool) {
	vb := mat32.NewVec2FmPoint(sv.WinBBox.Size())
	if vb.IsNil() {
		return
	}
	bsz := sv.ViewBox.Size
	sc := vb.Div(bsz)
	sv.Trans.Set(0, 0)
	if width {
		sv.Scale = sc.X
	} else {
		sv.Scale = mat32.Min(sc.X, sc.Y)
	}
	sv.SetTransform()
}

// ZoomToContents sets the scale to fit the current contents into view
func (sv *SVGView) ZoomToContents(width bool) {
	vb := mat32.NewVec2FmPoint(sv.WinBBox.Size())
	if vb.IsNil() {
		return
	}
	sv.ZoomToPage(width)
	sv.UpdateView(true)
	bb := sv.ContentsBBox()
	bsz := bb.Size()
	if bsz.X <= 0 || bsz.Y <= 0 {
		return
	}
	sc := vb.Div(bsz)
	sv.Trans = bb.Min.DivScalar(sv.Scale).Negate()
	if width {
		sv.Scale *= sc.X
	} else {
		sv.Scale *= mat32.Min(sc.X, sc.Y)
	}
	sv.SetTransform()
}

// ZoomAt updates the scale and translate parameters at given point
// by given delta: + means zoom in, - means zoom out,
// delta should always be < 1)
func (sv *SVGView) ZoomAt(pt image.Point, delta float32) {
	sc := float32(1)
	if delta > 1 {
		sc += delta
	} else {
		sc *= (1 - mat32.Min(-delta, .5))
	}

	nsc := sv.Scale * sc

	mpt := mat32.NewVec2FmPoint(pt.Sub(sv.WinBBox.Min))
	lpt := mpt.DivScalar(sv.Scale).Sub(sv.Trans) // point in drawing coords

	dt := lpt.Add(sv.Trans).MulScalar((nsc - sv.Scale) / nsc) // delta from zooming
	sv.Trans.SetSub(dt)

	sv.Scale = nsc
	sv.SetTransform()
}

// SetTransform sets the transform based on Trans and Scale values
func (sv *SVGView) SetTransform() {
	sv.SetProp("transform", fmt.Sprintf("scale(%v,%v) translate(%v,%v)", sv.Scale, sv.Scale, sv.Trans.X, sv.Trans.Y))
}

// MetaData returns the overall metadata and grid if present.
// if mknew is true, it will create new ones if not found.
func (sv *SVGView) MetaData(mknew bool) (main, grid *gi.MetaData2D) {
	if sv.NumChildren() > 0 {
		kd := sv.Kids[0]
		if md, ismd := kd.(*gi.MetaData2D); ismd {
			main = md
		}
	}
	if main == nil && mknew {
		id := sv.NewUniqueId()
		main = sv.InsertNewChild(gi.KiT_MetaData2D, 0, svg.NameId("namedview", id)).(*gi.MetaData2D)
	}
	if main == nil {
		return
	}
	if main.NumChildren() > 0 {
		kd := main.Kids[0]
		if md, ismd := kd.(*gi.MetaData2D); ismd {
			grid = md
		}
	}
	if grid == nil && mknew {
		id := sv.NewUniqueId()
		grid = main.InsertNewChild(gi.KiT_MetaData2D, 0, svg.NameId("grid", id)).(*gi.MetaData2D)
	}
	return
}

// SetMetaData sets meta data of drawing
func (sv *SVGView) SetMetaData() {
	es := sv.EditState()
	nv, gr := sv.MetaData(true)

	uts := strings.ToLower(sv.PhysWidth.Un.String())

	nv.SetProp("inkscape:current-layer", es.CurLayer)
	nv.SetProp("inkscape:cx", fmt.Sprintf("%g", sv.Trans.X))
	nv.SetProp("inkscape:cy", fmt.Sprintf("%g", sv.Trans.Y))
	nv.SetProp("inkscape:zoom", fmt.Sprintf("%g", sv.Scale))
	nv.SetProp("inkscape:document-units", uts)

	//	get rid of inkscape props we don't set
	nv.DeleteProp("cx")
	nv.DeleteProp("cy")
	nv.DeleteProp("zoom")
	nv.DeleteProp("document-units")
	nv.DeleteProp("current-layer")
	nv.DeleteProp("objecttolerance")
	nv.DeleteProp("guidetolerance")
	nv.DeleteProp("gridtolerance")
	nv.DeleteProp("pageopacity")
	nv.DeleteProp("borderopacity")
	nv.DeleteProp("bordercolor")
	nv.DeleteProp("pagecolor")
	nv.DeleteProp("pageshadow")
	nv.DeleteProp("pagecheckerboard")
	nv.DeleteProp("showgrid")

	spc := fmt.Sprintf("%g", sv.Grid)
	gr.SetProp("spacingx", spc)
	gr.SetProp("spacingy", spc)
	gr.SetProp("type", "xygrid")
	gr.SetProp("units", uts)
}

// ReadMetaData reads meta data of drawing
func (sv *SVGView) ReadMetaData() {
	es := sv.EditState()
	nv, gr := sv.MetaData(false)
	if nv == nil {
		return
	}
	if cx := nv.Prop("cx"); cx != nil {
		sv.Trans.X, _ = kit.ToFloat32(cx)
	}
	if cy := nv.Prop("cy"); cy != nil {
		sv.Trans.Y, _ = kit.ToFloat32(cy)
	}
	if zm := nv.Prop("zoom"); zm != nil {
		sc, _ := kit.ToFloat32(zm)
		if sc > 0 {
			sv.Scale = sc
		}
	}
	if cl := nv.Prop("current-layer"); cl != nil {
		es.CurLayer = kit.ToString(cl)
	}

	if gr == nil {
		return
	}
	if gs := gr.Prop("spacingx"); gs != nil {
		gv, _ := kit.ToFloat32(gs)
		if gv > 0 {
			sv.Grid = gv
		}
	}
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
	es.Changed = true
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
	es.ResetSelected()
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
	sv.UpdateSelect()
	return act
}

// Redo redoes one step, returning the action that was redone
func (sv *SVGView) Redo() string {
	es := sv.EditState()
	es.ResetSelected()
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
	sv.UpdateSelect()
	return act
}

///////////////////////////////////////////////////////////////////
// selection processing

// ShowAlignMatches draws the align matches as given
// between BBox Min - Max.  typs are corresponding bounding box sources.
func (sv *SVGView) ShowAlignMatches(pts []image.Rectangle, typs []BBoxPoints) {
	win := sv.GridView.ParentWindow()

	sz := ints.MinInt(len(pts), 8)
	for i := 0; i < sz; i++ {
		pt := pts[i].Canon()
		lsz := pt.Max.Sub(pt.Min)
		sp := Sprite(win, SpAlignMatch, Sprites(typs[i]), i, lsz)
		SetSpritePos(sp, pt.Min)
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
	es.SelectAction(nr, mouse.SelectOne, end)
	sv.UpdateEnd(updt)
	sv.UpdateSelSprites()
	sv.EditState().DragSelStart(start)
	win.SpriteDragging = SpriteName(SpReshapeBBox, SpBBoxDnR, 0)
	return nr
}

// NewText makes a new Text element with embedded tspan
func (sv *SVGView) NewText(start, end image.Point) svg.NodeSVG {
	// win := sv.GridView.ParentWindow()
	es := sv.EditState()
	sv.ManipStart("NewText", "")
	sv.SetFullReRender()
	nr := sv.NewEl(svg.KiT_Text).(*svg.Text)
	tsnm := fmt.Sprintf("tspan%d", sv.NewUniqueId())
	tspan := nr.AddNewChild(svg.KiT_Text, tsnm).(*svg.Text)
	tspan.Text = "Text"
	tspan.Width = 200
	xfi := sv.Pnt.XForm.Inverse()
	svoff := mat32.NewVec2FmPoint(sv.WinBBox.Min)
	pos := mat32.NewVec2FmPoint(start).Sub(svoff)
	minsz := float32(20)
	pos.SetSubScalar(minsz)
	pos = xfi.MulVec2AsPt(pos)
	sv.GridView.SetTextPropsNode(nr, es.Text.TextProps())
	nr.Pos = pos
	tspan.Pos = pos
	// dv := mat32.NewVec2FmPoint(end.Sub(start))
	// sz := dv.Abs().Max(mat32.NewVec2Scalar(minsz / 2))
	nr.Width = 100
	tspan.Width = 100
	es.SelectAction(nr, mouse.SelectOne, end)
	sv.UpdateView(true)
	sv.UpdateSelect()
	return nr
}

// NewPath makes a new SVG Path element during the drag operation
func (sv *SVGView) NewPath(start, end image.Point) *svg.Path {
	win := sv.GridView.ParentWindow()
	es := sv.EditState()
	sv.ManipStart("NewPath", "")
	updt := sv.UpdateStart()
	sv.SetFullReRender()
	nr := sv.NewEl(svg.KiT_Path).(*svg.Path)
	xfi := sv.Pnt.XForm.Inverse()
	svoff := mat32.NewVec2FmPoint(sv.WinBBox.Min)
	pos := mat32.NewVec2FmPoint(start).Sub(svoff)
	minsz := float32(20)
	pos.SetSubScalar(minsz)
	pos = xfi.MulVec2AsPt(pos)

	dv := mat32.NewVec2FmPoint(end.Sub(start))
	sz := dv.Abs().Max(mat32.NewVec2Scalar(minsz / 2))
	sz = xfi.MulVec2AsVec(sz)

	nr.SetData(fmt.Sprintf("m %g,%g %g,%g", pos.X, pos.Y, sz.X, sz.Y))

	es.SelectAction(nr, mouse.SelectOne, end)
	sv.UpdateEnd(updt)
	sv.UpdateSelSprites()
	sv.EditState().DragSelStart(start)
	win.SpriteDragging = SpriteName(SpReshapeBBox, SpBBoxDnR, 0)
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

///////////////////////////////////////////////////////////////////////
//  Bg render

func (sv *SVGView) Render2D() {
	if sv.PushBounds() {
		sv.SetFlag(int(svg.Rendering))
		sv.This().(gi.Node2D).ConnectEvents2D()
		sv.FillViewportWithBg()
		rs := &sv.Render
		rs.PushXForm(sv.Pnt.XForm)
		sv.Render2DChildren() // we must do children first, then us!
		sv.PopBounds()
		rs.PopXForm()
		sv.RenderViewport2D() // update our parent image
		sv.ClearFlag(int(svg.Rendering))
	}
}

func (sv *SVGView) BgNeedsUpdate() bool {
	updt := sv.EnsureBgSize() || (sv.Trans != sv.bgTrans) || (sv.Scale != sv.bgScale) || (sv.GridEff != sv.bgGridEff)
	// fmt.Printf("updt: %v\n", updt)
	return updt
}

func (sv *SVGView) FillViewportWithBg() {
	if sv.BgNeedsUpdate() {
		sv.RenderBg()
	}
	draw.Draw(sv.Pixels, sv.Pixels.Bounds(), sv.BgPixels, image.ZP, draw.Over) // draw the bg first
}

// EnsureBgSize ensures Bg is set to the right size -- returns true if resized
func (sv *SVGView) EnsureBgSize() bool {
	sz := sv.Pixels.Bounds().Size()
	if sv.BgPixels != nil {
		ib := sv.BgPixels.Bounds().Size()
		if ib == sz {
			return false
		}
	}
	if sv.BgPixels != nil {
		sv.BgPixels = nil
	}
	sv.BgPixels = image.NewRGBA(image.Rectangle{Max: sz})
	sv.BgRender.Init(sz.X, sz.Y, sv.BgPixels)
	return true
}

// UpdateGridEff updates the GirdEff value based on current scale
func (sv *SVGView) UpdateGridEff() {
	sv.GridEff = sv.Grid
	sp := sv.GridEff * sv.Scale
	for sp <= 2*(float32(Prefs.SnapTol)+1) {
		sv.GridEff *= 2
		sp = sv.GridEff * sv.Scale
	}
}

// RenderBg renders our background image
func (sv *SVGView) RenderBg() {
	rs := &sv.BgRender
	pc := &rs.Paint
	sv.EnsureBgSize()

	sv.UpdateGridEff()

	bb := sv.BgPixels.Bounds()

	draw.Draw(sv.BgPixels, bb, &image.Uniform{Prefs.Colors.Background}, image.ZP, draw.Src)

	rs.PushBounds(bb)
	rs.PushXForm(sv.Pnt.XForm)

	pc.StrokeStyle.SetColor(&Prefs.Colors.Border)

	sc := sv.Scale

	wd := 1 / sc
	pc.StrokeStyle.Width.Dots = wd
	pos := sv.ViewBox.Min
	sz := sv.ViewBox.Size
	pc.FillStyle.SetColor(nil)

	pc.DrawRectangle(rs, pos.X, pos.Y, sz.X, sz.Y)
	pc.FillStrokeClear(rs)

	if Prefs.GridDisp {
		gsz := float32(sv.GridEff)
		pc.StrokeStyle.SetColor(&Prefs.Colors.Grid)
		for x := gsz; x < sz.X; x += gsz {
			pc.DrawLine(rs, x, 0, x, sz.Y)
		}
		for y := gsz; y < sz.Y; y += gsz {
			pc.DrawLine(rs, 0, y, sz.X, y)
		}
		pc.FillStrokeClear(rs)
	}

	sv.bgTrans = sv.Trans
	sv.bgScale = sv.Scale
	sv.bgGridEff = sv.GridEff

	rs.PopXForm()
	rs.PopBounds()
}
