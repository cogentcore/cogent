// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vector

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"strings"

	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/giv"
	"cogentcore.org/core/grows/jsons"
	"cogentcore.org/core/grr"
	"cogentcore.org/core/gti"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keyfun"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/laser"
	"cogentcore.org/core/mat32"
	"cogentcore.org/core/svg"
)

// SVGView is the element for viewing, interacting with the SVG
type SVGView struct {
	gi.SVG

	// the parent vectorview
	VectorView *VectorView `copier:"-" json:"-" xml:"-" view:"-" set:"-"`

	// view translation offset (from dragging)
	Trans mat32.Vec2 `set:"-"`

	// view scaling (from zooming)
	Scale float32 `set:"-"`

	// grid spacing, in native ViewBox units
	Grid float32 ` set:"-"`

	// effective grid spacing given Scale level
	VectorEff float32 `edit:"-" set:"-"`

	// background pixels, includes page outline and grid
	BgPixels *image.RGBA `copier:"-" json:"-" xml:"-" view:"-" set:"-"`

	// render state for background rendering
	// BgRender girl.State `copier:"-" json:"-" xml:"-" view:"-" set:"-"`

	// bg rendered translation
	bgTrans mat32.Vec2 `copier:"-" json:"-" xml:"-" view:"-" set:"-"`

	// bg rendered scale
	bgScale float32 `copier:"-" json:"-" xml:"-" view:"-" set:"-"`

	// bg rendered grid
	bgVectorEff float32 `copier:"-" json:"-" xml:"-" view:"-" set:"-"`
}

func (sv *SVGView) OnInit() {
	sv.SVG.OnInit()
	sv.HandleEvents()
	sv.SetReadOnly(false)
	sv.Grid = Prefs.Size.Grid
	sv.Scale = 1
}

// SSVG returns the underlying [svg.SVG].
func (sv *SVGView) SSVG() *svg.SVG {
	return sv.SVG.SVG
}

// Root returns the root [svg.SVGNode].
func (sv *SVGView) Root() *svg.SVGNode {
	return &sv.SVG.SVG.Root
}

// EditState returns the EditState for this view
func (sv *SVGView) EditState() *EditState {
	if sv.VectorView == nil {
		return nil
	}
	return &sv.VectorView.EditState
}

// UpdateView updates the view, optionally with a full re-render
func (sv *SVGView) UpdateView(full bool) {
	sv.UpdateSelSprites()
}

func (sv *SVGView) HandleEvents() {
	sv.OnKeyChord(func(e events.Event) {
		kc := e.KeyChord()
		if gi.DebugSettings.KeyEventTrace {
			fmt.Printf("SVGView KeyInput: %v\n", sv.Path())
		}
		kf := keyfun.Of(kc)
		switch kf {
		case keyfun.Abort:
			// todo: maybe something else
			e.SetHandled()
			sv.VectorView.SetTool(SelectTool)
		case keyfun.Undo:
			e.SetHandled()
			sv.VectorView.Undo()
		case keyfun.Redo:
			e.SetHandled()
			sv.VectorView.Redo()
		case keyfun.Duplicate:
			e.SetHandled()
			sv.VectorView.DuplicateSelected()
		case keyfun.Copy:
			e.SetHandled()
			sv.VectorView.CopySelected()
		case keyfun.Cut:
			e.SetHandled()
			sv.VectorView.CutSelected()
		case keyfun.Paste:
			e.SetHandled()
			sv.VectorView.PasteClip()
		case keyfun.Delete, keyfun.Backspace:
			e.SetHandled()
			sv.VectorView.DeleteSelected()
		}
	})
	sv.On(events.MouseDown, func(e events.Event) {
		if e.MouseButton() != events.Left {
			return
		}
		e.SetHandled()
		es := sv.EditState()
		sob := sv.SelectContainsPoint(e.Pos(), false, true) // not leavesonly, yes exclude existing sels

		es.SelectNoDrag = false
		switch {
		case es.HasSelected() && es.SelectBBox.ContainsPoint(mat32.V2FromPoint(e.Pos())):
			// note: this absorbs potential secondary selections within selection -- handled
			// on release below, if nothing else happened
			es.SelectNoDrag = true
			es.DragSelStart(e.Pos())
		case sob != nil && es.Tool == SelectTool:
			es.SelectAction(sob, e.SelectMode(), e.Pos())
			sv.EditState().DragSelStart(e.Pos())
			sv.UpdateSelect()
		case sob != nil && es.Tool == NodeTool:
			es.SelectAction(sob, events.SelectOne, e.Pos())
			sv.EditState().DragSelStart(e.Pos())
			sv.UpdateNodeSprites()
		case sob == nil:
			es.ResetSelected()
			sv.UpdateSelect()
		}
	})
	sv.On(events.MouseUp, func(e events.Event) {
		es := sv.EditState()
		sob := sv.SelectContainsPoint(e.Pos(), false, true) // not leavesonly, yes exclude existing sels

		if es.InAction() {
			es.SelectNoDrag = false
			es.NewTextMade = false
			sv.ManipDone()
			return
		}
		if e.MouseButton() == events.Left {
			// release on select -- do extended selection processing
			if (es.SelectNoDrag && es.Tool == SelectTool) || (es.Tool != SelectTool && ToolDoesBasicSelect(es.Tool)) {
				es.SelectNoDrag = false
				e.SetHandled()
				if sob == nil {
					sob = sv.SelectContainsPoint(e.Pos(), false, false) // don't exclude existing sel
				}
				if sob != nil {
					es.SelectAction(sob, e.SelectMode(), e.Pos())
					sv.UpdateSelect()
				}
			}
		}
		// if e.MouseButton() == events.Right {
		// 	e.SetHandled()
		// 	if es.HasSelected() {
		// 		fobj := es.FirstSelectedNode()
		// 		if fobj != nil {
		// 			sv.NodeContextMenu(fobj, me.Where)
		// 		}
		// 	} else if sob != nil {
		// 		ssvg.NodeContextMenu(sob, me.Where)
		// 	}
		// }
	})
	sv.On(events.SlideMove, func(e events.Event) {
		es := sv.EditState()
		es.SelectNoDrag = false
		e.SetHandled()
		es.DragStartPos = e.StartPos()
		if e.HasAnyModifier(key.Shift) {
			e.ClearHandled() // base gi.SVG handles it
			return
		}
		if es.HasSelected() {
			if !es.NewTextMade {
				sv.DragMove(e)
			}
			return
		}
		if !es.InAction() {
			switch es.Tool {
			case SelectTool:
				sv.SetRubberBand(e.Pos())
			case RectTool:
				sv.NewElDrag(svg.RectType, es.DragStartPos, e.Pos())
				es.SelectBBox.Min.X += 1
				es.SelectBBox.Min.Y += 1
				es.DragSelectStartBBox = es.SelectBBox
				es.DragSelectCurrentBBox = es.SelectBBox
				es.DragSelectEffectiveBBox = es.SelectBBox
			case EllipseTool:
				sv.NewElDrag(svg.EllipseType, es.DragStartPos, e.Pos())
			case TextTool:
				sv.NewText(es.DragStartPos, e.Pos())
				es.NewTextMade = true
			case BezierTool:
				sv.NewPath(es.DragStartPos, e.Pos())
			}
		} else if es.Action == "BoxSelect" {
			sv.SetRubberBand(e.Pos())
		}
	})
}

/*
func (sv *SVGView) MouseHover() {
	sv.ConnectEvent(oswin.MouseHoverEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d any) {
		me := d.(*mouse.HoverEvent)
		me.SetHandled()
		ssvg := recv.Embed(KiT_SVGView).(*SVGView)
		obj := ssvg.FirstContainingPoint(me.Where, true)
		if obj != nil {
			pos := me.Where
			ttxt := fmt.Sprintf("element name: %v -- use right mouse click to edit", obj.Name())
			gi.PopupTooltip(obj.Name(), pos.X, pos.Y, sv.ViewportSafe(), ttxt)
		}
	})
}
*/

// ContentsBBox returns the object-level box of the entire contents
func (sv *SVGView) ContentsBBox() mat32.Box2 {
	bbox := mat32.Box2{}
	bbox.SetEmpty()
	sv.WalkPre(func(k ki.Ki) bool {
		if k.This() == sv.This() {
			return ki.Continue
		}
		if k.This() == sv.SSVG().Defs.This() {
			return ki.Break
		}
		sni, issv := k.(svg.Node)
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
		sn := sni.AsNodeBase()
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

// TransformAllLeaves transforms all the leaf items in the drawing (not groups)
// uses ApplyDeltaTransform manipulation.
func (sv *SVGView) TransformAllLeaves(trans mat32.Vec2, scale mat32.Vec2, rot float32, pt mat32.Vec2) {
	sv.WalkPre(func(k ki.Ki) bool {
		if k.This() == sv.This() {
			return ki.Continue
		}
		if k.This() == sv.SSVG().Defs.This() {
			return ki.Break
		}
		sni, issv := k.(svg.Node)
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
		sni.ApplyDeltaTransform(sv.SSVG(), trans, scale, rot, pt)
		return ki.Continue
	})
}

// ZoomToPage sets the scale to fit the current viewbox
func (sv *SVGView) ZoomToPage(width bool) {
	vb := mat32.V2FromPoint(sv.Root().BBox.Size())
	if vb == (mat32.Vec2{}) {
		return
	}
	bsz := sv.Root().ViewBox.Size
	if bsz.X <= 0 || bsz.Y <= 0 {
		return
	}
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
	vb := mat32.V2FromPoint(sv.Root().BBox.Size())
	if vb == (mat32.Vec2{}) {
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

// ResizeToContents resizes the drawing to just fit the current contents,
// including moving everything to start at upper-left corner,
// optionally preserving the current grid offset, so grid snapping
// is preserved -- recommended.
func (sv *SVGView) ResizeToContents(grid_off bool) {
	sv.UndoSave("ResizeToContents", "")
	sv.ZoomToPage(false)
	sv.UpdateView(true)
	bb := sv.ContentsBBox()
	bsz := bb.Size()
	if bsz.X <= 0 || bsz.Y <= 0 {
		return
	}
	trans := bb.Min
	incr := sv.Grid * sv.Scale // our zoom factor
	treff := trans
	if grid_off {
		treff.X = mat32.Floor(trans.X/incr) * incr
		treff.Y = mat32.Floor(trans.Y/incr) * incr
	}
	bsz.SetAdd(trans.Sub(treff))
	treff = treff.Negate()

	bsz = bsz.DivScalar(sv.Scale)

	sv.TransformAllLeaves(treff, mat32.V2(1, 1), 0, mat32.V2(0, 0))
	sv.Root().ViewBox.Size = bsz
	sv.SSVG().PhysWidth.Value = bsz.X
	sv.SSVG().PhysHeight.Value = bsz.Y
	sv.ZoomToPage(false)
	sv.VectorView.ChangeMade()
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

	mpt := mat32.V2FromPoint(pt.Sub(sv.Root().BBox.Min))
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
func (sv *SVGView) MetaData(mknew bool) (main, grid *svg.MetaData) {
	if sv.NumChildren() > 0 {
		kd := sv.Root().Kids[0]
		if md, ismd := kd.(*svg.MetaData); ismd {
			main = md
		}
	}
	if main == nil && mknew {
		id := sv.SSVG().NewUniqueID()
		main = sv.InsertNewChild(svg.MetaDataType, 0, svg.NameID("namedview", id)).(*svg.MetaData)
	}
	if main == nil {
		return
	}
	if main.NumChildren() > 0 {
		kd := main.Kids[0]
		if md, ismd := kd.(*svg.MetaData); ismd {
			grid = md
		}
	}
	if grid == nil && mknew {
		id := sv.SSVG().NewUniqueID()
		grid = main.InsertNewChild(svg.MetaDataType, 0, svg.NameID("grid", id)).(*svg.MetaData)
	}
	return
}

// SetMetaData sets meta data of drawing
func (sv *SVGView) SetMetaData() {
	es := sv.EditState()
	nv, gr := sv.MetaData(true)

	uts := strings.ToLower(sv.SSVG().PhysWidth.Unit.String())

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
		sv.Trans.X, _ = laser.ToFloat32(cx)
	}
	if cy := nv.Prop("cy"); cy != nil {
		sv.Trans.Y, _ = laser.ToFloat32(cy)
	}
	if zm := nv.Prop("zoom"); zm != nil {
		sc, _ := laser.ToFloat32(zm)
		if sc > 0 {
			sv.Scale = sc
		}
	}
	if cl := nv.Prop("current-layer"); cl != nil {
		es.CurLayer = laser.ToString(cl)
	}

	if gr == nil {
		return
	}
	if gs := gr.Prop("spacingx"); gs != nil {
		gv, _ := laser.ToFloat32(gs)
		if gv > 0 {
			sv.Grid = gv
		}
	}
}

///////////////////////////////////////////////////////////////////////////
//  ContextMenu / Actions

// EditNode opens a structview editor on node
func (sv *SVGView) EditNode(kn ki.Ki) {
	giv.StructViewDialog(sv, kn, "SVG Element View", true)
}

// MakeNodeContextMenu makes the menu of options for context right click
func (sv *SVGView) MakeNodeContextMenu(m *gi.Scene, kn ki.Ki) {
	gi.NewButton(m).SetText("Edit").SetIcon(icons.Edit).OnClick(func(e events.Event) {
		sv.EditNode(kn)
	})
	gi.NewButton(m).SetText("Select in tree").SetIcon(icons.Select).OnClick(func(e events.Event) {
		sv.VectorView.SelectNodeInTree(kn, events.SelectOne)
	})

	gi.NewSeparator(m)

	giv.NewFuncButton(m, sv.VectorView.DuplicateSelected).SetText("Duplicate").SetIcon(icons.Copy).SetKey(keyfun.Duplicate)
	giv.NewFuncButton(m, sv.VectorView.CopySelected).SetText("Copy").SetIcon(icons.Copy).SetKey(keyfun.Copy)
	giv.NewFuncButton(m, sv.VectorView.CutSelected).SetText("Cut").SetIcon(icons.Cut).SetKey(keyfun.Cut)
	giv.NewFuncButton(m, sv.VectorView.PasteClip).SetText("Paste").SetIcon(icons.Paste).SetKey(keyfun.Paste)
}

// ContextMenuPos returns position to use for context menu, based on input position
func (sv *SVGView) NodeContextMenuPos(pos image.Point) image.Point {
	if pos != image.ZP {
		return pos
	}
	bbox := sv.Root().BBox
	pos.X = (bbox.Min.X + bbox.Max.X) / 2
	pos.Y = (bbox.Min.Y + bbox.Max.Y) / 2
	return pos
}

///////////////////////////////////////////////////////////////////////////
// Undo

// UndoSave save current state for potential undo
func (sv *SVGView) UndoSave(action, data string) {
	es := sv.EditState()
	if es == nil {
		return
	}
	es.Changed = true
	b := &bytes.Buffer{}
	grr.Log(jsons.Write(sv.Root(), b))
	bs := strings.Split(b.String(), "\n")
	es.UndoMgr.Save(action, data, bs)
}

// UndoSaveReplace save current state to replace current
func (sv *SVGView) UndoSaveReplace(action, data string) {
	es := sv.EditState()
	b := &bytes.Buffer{}
	grr.Log(jsons.Write(sv.Root(), b))
	bs := strings.Split(b.String(), "\n")
	es.UndoMgr.SaveReplace(action, data, bs)
}

// Undo undoes one step, returning the action that was undone
func (sv *SVGView) Undo() string {
	es := sv.EditState()
	es.ResetSelected()
	if es.UndoMgr.MustSaveUndoStart() { // need to save current state!
		b := &bytes.Buffer{}
		grr.Log(jsons.Write(sv.Root(), b))
		bs := strings.Split(b.String(), "\n")
		es.UndoMgr.SaveUndoStart(bs)
	}
	act, _, state := es.UndoMgr.Undo()
	if state == nil {
		return act
	}
	sb := strings.Join(state, "\n")
	b := bytes.NewBufferString(sb)
	grr.Log(jsons.Read(sv.Root(), b))
	sv.UpdateSelect()
	return act
}

// Redo redoes one step, returning the action that was redone
func (sv *SVGView) Redo() string {
	es := sv.EditState()
	es.ResetSelected()
	act, _, state := es.UndoMgr.Redo()
	if state == nil {
		return act
	}
	sb := strings.Join(state, "\n")
	b := bytes.NewBufferString(sb)
	grr.Log(jsons.Read(sv.Root(), b))
	sv.UpdateSelect()
	return act
}

///////////////////////////////////////////////////////////////////
// selection processing

// ShowAlignMatches draws the align matches as given
// between BBox Min - Max.  typs are corresponding bounding box sources.
func (sv *SVGView) ShowAlignMatches(pts []image.Rectangle, typs []BBoxPoints) {
	// win := sv.VectorView.ParentWindow()

	// sz := min(len(pts), 8)
	// for i := 0; i < sz; i++ {
	// 	pt := pts[i].Canon()
	// 	lsz := pt.Max.Sub(pt.Min)
	// 	sp := Sprite(win, SpAlignMatch, Sprites(typs[i]), i, lsz)
	// 	SetSpritePos(sp, pt.Min)
	// }
}

// DepthMap returns a map of all nodes and their associated depth count
// counting up from 0 as the deepest, first drawn node.
func (sv *SVGView) DepthMap() map[ki.Ki]int {
	m := make(map[ki.Ki]int)
	depth := 0
	n := ki.Next(sv.This())
	for n != nil {
		m[n] = depth
		depth++
		n = ki.Next(n)
	}
	return m
}

///////////////////////////////////////////////////////////////////////
// New objects

// SetSVGName sets the name of the element to standard type + id name
func (sv *SVGView) SetSVGName(el svg.Node) {
	nwid := sv.SSVG().NewUniqueID()
	nwnm := fmt.Sprintf("%s%d", el.SVGName(), nwid)
	el.SetName(nwnm)
}

// NewEl makes a new SVG element, giving it a new unique name.
// Uses currently active layer if set.
func (sv *SVGView) NewEl(typ *gti.Type) svg.Node {
	es := sv.EditState()
	parent := ki.Ki(sv.Root())
	if es.CurLayer != "" {
		ly := sv.ChildByName(es.CurLayer, 1)
		if ly != nil {
			parent = ly
		}
	}
	nwnm := fmt.Sprintf("%s_tmp_new_item_", typ.Name)
	nw := parent.NewChild(typ, nwnm).(svg.Node)
	sv.SetSVGName(nw)
	sv.VectorView.PaintView().SetProps(nw)
	sv.VectorView.UpdateTreeView()
	return nw
}

// NewElDrag makes a new SVG element during the drag operation
func (sv *SVGView) NewElDrag(typ *gti.Type, start, end image.Point) svg.Node {
	minsz := float32(10)
	es := sv.EditState()
	dv := mat32.V2FromPoint(end.Sub(start))
	if !es.InAction() && mat32.Abs(dv.X) < minsz && mat32.Abs(dv.Y) < minsz {
		return nil
	}
	// win := sv.VectorView.ParentWindow()
	tn := typ.Name
	sv.ManipStart("New"+tn, "")
	// sv.SetFullReRender()
	nr := sv.NewEl(typ)
	xfi := sv.Root().Paint.Transform.Inverse()
	svoff := mat32.V2FromPoint(sv.Geom.ContentBBox.Min)
	pos := mat32.V2FromPoint(start).Sub(svoff)
	nr.SetNodePos(xfi.MulVec2AsPoint(pos))
	sz := dv.Abs().Max(mat32.V2Scalar(minsz / 2))
	nr.SetNodeSize(xfi.MulVec2AsVec(sz))
	es.SelectAction(nr, events.SelectOne, end)
	sv.ManipDone()
	sv.NeedsRender()
	sv.UpdateSelSprites()
	es.DragSelStart(start)
	// win.SpriteDragging = SpriteName(SpReshapeBBox, SpBBoxDnR, 0)
	return nr
}

// NewText makes a new Text element with embedded tspan
func (sv *SVGView) NewText(start, end image.Point) svg.Node {
	es := sv.EditState()
	sv.ManipStart("NewText", "")
	nr := sv.NewEl(svg.TextType)
	tsnm := fmt.Sprintf("tspan%d", sv.SSVG().NewUniqueID())
	tspan := svg.NewText(nr, tsnm)
	tspan.Text = "Text"
	tspan.Width = 200
	xfi := sv.Root().Paint.Transform.Inverse()
	svoff := mat32.V2FromPoint(sv.Geom.ContentBBox.Min)
	pos := mat32.V2FromPoint(start).Sub(svoff)
	// minsz := float32(20)
	pos.Y += 20 // todo: need the font size..
	pos = xfi.MulVec2AsPoint(pos)
	sv.VectorView.SetTextPropsNode(nr, es.Text.TextProps())
	// nr.Pos = pos
	// tspan.Pos = pos
	// // dv := mat32.V2FromPoint(end.Sub(start))
	// // sz := dv.Abs().Max(mat32.NewVec2Scalar(minsz / 2))
	// nr.Width = 100
	// tspan.Width = 100
	es.SelectAction(nr, events.SelectOne, end)
	// sv.UpdateView(true)
	// sv.UpdateSelect()
	return nr
}

// NewPath makes a new SVG Path element during the drag operation
func (sv *SVGView) NewPath(start, end image.Point) *svg.Path {
	minsz := float32(10)
	es := sv.EditState()
	dv := mat32.V2FromPoint(end.Sub(start))
	if !es.InAction() && mat32.Abs(dv.X) < minsz && mat32.Abs(dv.Y) < minsz {
		return nil
	}
	// win := sv.VectorView.ParentWindow()
	sv.ManipStart("NewPath", "")
	// sv.SetFullReRender()
	nr := sv.NewEl(svg.PathType).(*svg.Path)
	xfi := sv.Root().Paint.Transform.Inverse()
	svoff := mat32.V2FromPoint(sv.Geom.ContentBBox.Min)
	pos := mat32.V2FromPoint(start).Sub(svoff)
	pos = xfi.MulVec2AsPoint(pos)
	sz := dv
	// sz := dv.Abs().Max(mat32.NewVec2Scalar(minsz / 2))
	sz = xfi.MulVec2AsVec(sz)

	nr.SetData(fmt.Sprintf("m %g,%g %g,%g", pos.X, pos.Y, sz.X, sz.Y))

	es.SelectAction(nr, events.SelectOne, end)
	sv.UpdateSelSprites()
	sv.EditState().DragSelStart(start)

	es.SelectBBox.Min.X += 1
	es.SelectBBox.Min.Y += 1
	es.DragSelectStartBBox = es.SelectBBox
	es.DragSelectCurrentBBox = es.SelectBBox
	es.DragSelectEffectiveBBox = es.SelectBBox

	// win.SpriteDragging = SpriteName(SpReshapeBBox, SpBBoxDnR, 0)
	return nr
}

///////////////////////////////////////////////////////////////////////
// Gradients

// Gradients returns the currently-defined gradients with stops
// that are shared among obj-specific ones
func (sv *SVGView) Gradients() []*Gradient {
	gl := make([]*Gradient, 0)
	for _, gii := range sv.SSVG().Defs.Kids {
		g, ok := gii.(*svg.Gradient)
		if !ok {
			continue
		}
		if g.StopsName != "" {
			continue
		}
		gr := &Gradient{}
		// gr.UpdateFromGrad(g)
		gl = append(gl, gr)
	}
	return gl
}

// UpdateGradients update SVG gradients from given gradient list
func (sv *SVGView) UpdateGradients(gl []*Gradient) {
	nms := make(map[string]bool)
	for _, gr := range gl {
		if _, has := nms[gr.Name]; has {
			id := sv.SSVG().NewUniqueID()
			gr.Name = fmt.Sprintf("%d", id)
		}
		nms[gr.Name] = true
	}

	// for _, gr := range gl {
	// 	radial := false
	// 	if strings.HasPrefix(gr.Name, "radial") {
	// 		radial = true
	// 	}
	// 	var g *svg.Gradient
	// 	gg := sv.SSVG().FindDefByName(gr.Name)
	// 	if gg == nil {
	// 		g, _ = svg.NewGradient(radial)
	// 	} else {
	// 		g = gg.(*svg.Gradient)
	// 	}

	// 	gr.UpdateGrad(g)
	// }
	// sv.UpdateAllGradientStops()
}

///////////////////////////////////////////////////////////////////////
//  Bg render

// func (sv *SVGView) Render() {
// 	if sv.PushBounds() {
// 		sv.SSVG().IsRendering = true
// 		sv.FillViewportWithBg()
// 		rs := &sv.Render
// 		rs.PushTransform(sv.Pnt.Transform)
// 		sv.Render2DChildren() // we must do children first, then us!
// 		sv.PopBounds()
// 		rs.PopTransform()
// 		sv.RenderViewport2D() // update our parent image
// 		sv.ClearFlag(int(svg.Rendering))
// 	}
// }

func (sv *SVGView) BgNeedsUpdate() bool {
	updt := sv.EnsureBgSize() || (sv.Trans != sv.bgTrans) || (sv.Scale != sv.bgScale) || (sv.VectorEff != sv.bgVectorEff)
	// fmt.Printf("updt: %v\n", updt)
	return updt
}

func (sv *SVGView) FillViewportWithBg() {
	if sv.BgNeedsUpdate() {
		sv.RenderBg()
	}
	draw.Draw(sv.Scene.Pixels, sv.Geom.ContentBBox, sv.BgPixels, sv.Geom.ScrollOffset(), draw.Over) // draw the bg first
}

// EnsureBgSize ensures Bg is set to the right size -- returns true if resized
func (sv *SVGView) EnsureBgSize() bool {
	sz := sv.Geom.ContentBBox.Size()
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
	// sv.BgRender.Init(sz.X, sz.Y, sv.BgPixels)
	return true
}

// UpdateVectorEff updates the GirdEff value based on current scale
func (sv *SVGView) UpdateVectorEff() {
	sv.VectorEff = sv.Grid
	sp := sv.VectorEff * sv.Scale
	for sp <= 2*(float32(Prefs.SnapTol)+1) {
		sv.VectorEff *= 2
		sp = sv.VectorEff * sv.Scale
	}
}

// RenderBg renders our background image
func (sv *SVGView) RenderBg() {
	/*
		rs := &sv.BgRender
		pc := &rs.Paint
		sv.EnsureBgSize()

		sv.UpdateVectorEff()

		bb := sv.BgPixels.Bounds()

		draw.Draw(sv.BgPixels, bb, &image.Uniform{Prefs.Colors.Background}, image.ZP, draw.Src)

		rs.PushBounds(bb)
		rs.PushTransform(sv.Pnt.Transform)

		pc.StrokeStyle.SetColor(&Prefs.Colors.Border)

		sc := sv.Scale

		wd := 1 / sc
		pc.StrokeStyle.Width.Dots = wd
		pos := sv.ViewBox.Min
		sz := sv.ViewBox.Size
		pc.FillStyle.SetColor(nil)

		pc.DrawRectangle(rs, pos.X, pos.Y, sz.X, sz.Y)
		pc.FillStrokeClear(rs)

		if Prefs.VectorDisp {
			gsz := float32(sv.VectorEff)
			pc.StrokeStyle.SetColor(&Prefs.Colors.Vector)
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
		sv.bgVectorEff = sv.VectorEff

		rs.PopTransform()
		rs.PopBounds()
	*/
}
