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

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/iox/jsonx"
	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/svg"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/types"
)

// SVG is the element for viewing and interacting with the SVG.
type SVG struct {
	core.SVG

	// the parent vector
	Vector *Vector `copier:"-" json:"-" xml:"-" display:"-" set:"-"`

	// view translation offset (from dragging)
	Trans math32.Vector2 `set:"-"`

	// view scaling (from zooming)
	Scale float32 `set:"-"`

	// grid spacing, in native ViewBox units
	Grid float32 ` set:"-"`

	// effective grid spacing given Scale level
	VectorEff float32 `edit:"-" set:"-"`

	// background pixels, includes page outline and grid
	BgPixels *image.RGBA `copier:"-" json:"-" xml:"-" display:"-" set:"-"`

	// render state for background rendering
	// BgRender girl.State `copier:"-" json:"-" xml:"-" display:"-" set:"-"`

	// bg rendered translation
	bgTrans math32.Vector2 `copier:"-" json:"-" xml:"-" display:"-" set:"-"`

	// bg rendered scale
	bgScale float32 `copier:"-" json:"-" xml:"-" display:"-" set:"-"`

	// bg rendered grid
	bgVectorEff float32 `copier:"-" json:"-" xml:"-" display:"-" set:"-"`
}

func (sv *SVG) Init() {
	sv.SVG.Init()
	sv.SetReadOnly(false)
	sv.Grid = Settings.Size.Grid
	sv.Scale = 1

	sv.OnKeyChord(func(e events.Event) {
		kc := e.KeyChord()
		if core.DebugSettings.KeyEventTrace {
			fmt.Printf("SVG KeyInput: %v\n", sv.Path())
		}
		kf := keymap.Of(kc)
		switch kf {
		case keymap.Abort:
			// todo: maybe something else
			e.SetHandled()
			sv.Vector.SetTool(SelectTool)
		case keymap.Undo:
			e.SetHandled()
			sv.Vector.Undo()
		case keymap.Redo:
			e.SetHandled()
			sv.Vector.Redo()
		case keymap.Duplicate:
			e.SetHandled()
			sv.Vector.DuplicateSelected()
		case keymap.Copy:
			e.SetHandled()
			sv.Vector.CopySelected()
		case keymap.Cut:
			e.SetHandled()
			sv.Vector.CutSelected()
		case keymap.Paste:
			e.SetHandled()
			sv.Vector.PasteClip()
		case keymap.Delete, keymap.Backspace:
			e.SetHandled()
			sv.Vector.DeleteSelected()
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
		case es.HasSelected() && es.SelectBBox.ContainsPoint(math32.Vector2FromPoint(e.Pos())):
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
			e.ClearHandled() // base core.SVG handles it
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
				sv.NewElementDrag(svg.RectType, es.DragStartPos, e.Pos())
				es.SelectBBox.Min.X += 1
				es.SelectBBox.Min.Y += 1
				es.DragSelectStartBBox = es.SelectBBox
				es.DragSelectCurrentBBox = es.SelectBBox
				es.DragSelectEffectiveBBox = es.SelectBBox
			case EllipseTool:
				sv.NewElementDrag(svg.EllipseType, es.DragStartPos, e.Pos())
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

// SSVG returns the underlying [svg.SVG].
func (sv *SVG) SSVG() *svg.SVG {
	return sv.SVG.SVG
}

// Root returns the root [svg.Root].
func (sv *SVG) Root() *svg.Root {
	return sv.SVG.SVG.Root
}

// EditState returns the EditState for this view
func (sv *SVG) EditState() *EditState {
	if sv.Vector == nil {
		return nil
	}
	return &sv.Vector.EditState
}

// UpdateView updates the view, optionally with a full re-render
func (sv *SVG) UpdateView(full bool) { // TODO(config)
	sv.UpdateSelSprites()
}

/*
func (sv *SVG) MouseHover() {
	sv.ConnectEvent(oswin.MouseHoverEvent, core.RegPri, func(recv, send tree.Node, sig int64, d any) {
		me := d.(*mouse.HoverEvent)
		me.SetHandled()
		ssvg := recv.Embed(KiT_SVG).(*SVG)
		obj := ssvg.FirstContainingPoint(me.Where, true)
		if obj != nil {
			pos := me.Where
			ttxt := fmt.Sprintf("element name: %v -- use right mouse click to edit", obj.Name)
			core.PopupTooltip(obj.Name, pos.X, pos.Y, sv.ViewportSafe(), ttxt)
		}
	})
}
*/

// ContentsBBox returns the object-level box of the entire contents
func (sv *SVG) ContentsBBox() math32.Box2 {
	bbox := math32.Box2{}
	bbox.SetEmpty()
	sv.WalkDown(func(n tree.Node) bool {
		if n == sv.This {
			return tree.Continue
		}
		if n == sv.SSVG().Defs {
			return tree.Break
		}
		sni, issv := n.(svg.Node)
		if !issv {
			return tree.Break
		}
		if NodeIsLayer(n) {
			return tree.Continue
		}
		if txt, istxt := sni.(*svg.Text); istxt { // no tspans
			if txt.Text != "" {
				return tree.Break
			}
		}
		sn := sni.AsNodeBase()
		bb := math32.Box2{}
		bb.SetFromRect(sn.BBox)
		bbox.ExpandByBox(bb)
		if _, isgp := sni.(*svg.Group); isgp { // subsumes all
			return tree.Break
		}
		return tree.Continue
	})
	if bbox.IsEmpty() {
		bbox = math32.Box2{}
	}
	return bbox
}

// TransformAllLeaves transforms all the leaf items in the drawing (not groups)
// uses ApplyDeltaTransform manipulation.
func (sv *SVG) TransformAllLeaves(trans math32.Vector2, scale math32.Vector2, rot float32, pt math32.Vector2) {
	sv.WalkDown(func(n tree.Node) bool {
		if n == sv.This {
			return tree.Continue
		}
		if n == sv.SSVG().Defs {
			return tree.Break
		}
		sni, issv := n.(svg.Node)
		if !issv {
			return tree.Break
		}
		if NodeIsLayer(n) {
			return tree.Continue
		}
		if _, isgp := sni.(*svg.Group); isgp {
			return tree.Continue
		}
		if txt, istxt := sni.(*svg.Text); istxt { // no tspans
			if txt.Text != "" {
				return tree.Break
			}
		}
		sni.ApplyDeltaTransform(sv.SSVG(), trans, scale, rot, pt)
		return tree.Continue
	})
}

// ZoomToPage sets the scale to fit the current viewbox
func (sv *SVG) ZoomToPage(width bool) {
	vb := math32.Vector2FromPoint(sv.Root().BBox.Size())
	if vb == (math32.Vector2{}) {
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
		sv.Scale = math32.Min(sc.X, sc.Y)
	}
	sv.SetTransform()
}

// ZoomToContents sets the scale to fit the current contents into view
func (sv *SVG) ZoomToContents(width bool) {
	vb := math32.Vector2FromPoint(sv.Root().BBox.Size())
	if vb == (math32.Vector2{}) {
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
		sv.Scale *= math32.Min(sc.X, sc.Y)
	}
	sv.SetTransform()
}

// ResizeToContents resizes the drawing to just fit the current contents,
// including moving everything to start at upper-left corner,
// optionally preserving the current grid offset, so grid snapping
// is preserved -- recommended.
func (sv *SVG) ResizeToContents(grid_off bool) {
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
		treff.X = math32.Floor(trans.X/incr) * incr
		treff.Y = math32.Floor(trans.Y/incr) * incr
	}
	bsz.SetAdd(trans.Sub(treff))
	treff = treff.Negate()

	bsz = bsz.DivScalar(sv.Scale)

	sv.TransformAllLeaves(treff, math32.Vec2(1, 1), 0, math32.Vec2(0, 0))
	sv.Root().ViewBox.Size = bsz
	sv.SSVG().PhysicalWidth.Value = bsz.X
	sv.SSVG().PhysicalHeight.Value = bsz.Y
	sv.ZoomToPage(false)
	sv.Vector.ChangeMade()
}

// ZoomAt updates the scale and translate parameters at given point
// by given delta: + means zoom in, - means zoom out,
// delta should always be < 1)
func (sv *SVG) ZoomAt(pt image.Point, delta float32) {
	sc := float32(1)
	if delta > 1 {
		sc += delta
	} else {
		sc *= (1 - math32.Min(-delta, .5))
	}

	nsc := sv.Scale * sc

	mpt := math32.Vector2FromPoint(pt.Sub(sv.Root().BBox.Min))
	lpt := mpt.DivScalar(sv.Scale).Sub(sv.Trans) // point in drawing coords

	dt := lpt.Add(sv.Trans).MulScalar((nsc - sv.Scale) / nsc) // delta from zooming
	sv.Trans.SetSub(dt)

	sv.Scale = nsc
	sv.SetTransform()
}

// SetTransform sets the transform based on Trans and Scale values
func (sv *SVG) SetTransform() {
	sv.SetProperty("transform", fmt.Sprintf("scale(%v,%v) translate(%v,%v)", sv.Scale, sv.Scale, sv.Trans.X, sv.Trans.Y))
}

// MetaData returns the overall metadata and grid if present.
// if mknew is true, it will create new ones if not found.
func (sv *SVG) MetaData(mknew bool) (main, grid *svg.MetaData) {
	if sv.NumChildren() > 0 {
		kd := sv.Root().Children[0]
		if md, ismd := kd.(*svg.MetaData); ismd {
			main = md
		}
	}
	if main == nil && mknew {
		id := sv.SSVG().NewUniqueID()
		main = sv.Root().InsertNewChild(svg.MetaDataType, 0).(*svg.MetaData)
		main.SetName(svg.NameID("namedview", id))
	}
	if main == nil {
		return
	}
	if main.NumChildren() > 0 {
		kd := main.Children[0]
		if md, ismd := kd.(*svg.MetaData); ismd {
			grid = md
		}
	}
	if grid == nil && mknew {
		id := sv.SSVG().NewUniqueID()
		grid = main.InsertNewChild(svg.MetaDataType, 0).(*svg.MetaData)
		grid.SetName(svg.NameID("grid", id))
	}
	return
}

// SetMetaData sets meta data of drawing
func (sv *SVG) SetMetaData() {
	es := sv.EditState()
	nv, gr := sv.MetaData(true)

	uts := strings.ToLower(sv.SSVG().PhysicalWidth.Unit.String())

	nv.SetProperty("inkscape:current-layer", es.CurLayer)
	nv.SetProperty("inkscape:cx", fmt.Sprintf("%g", sv.Trans.X))
	nv.SetProperty("inkscape:cy", fmt.Sprintf("%g", sv.Trans.Y))
	nv.SetProperty("inkscape:zoom", fmt.Sprintf("%g", sv.Scale))
	nv.SetProperty("inkscape:document-units", uts)

	//	get rid of inkscape properties we don't set
	nv.DeleteProperty("cx")
	nv.DeleteProperty("cy")
	nv.DeleteProperty("zoom")
	nv.DeleteProperty("document-units")
	nv.DeleteProperty("current-layer")
	nv.DeleteProperty("objecttolerance")
	nv.DeleteProperty("guidetolerance")
	nv.DeleteProperty("gridtolerance")
	nv.DeleteProperty("pageopacity")
	nv.DeleteProperty("borderopacity")
	nv.DeleteProperty("bordercolor")
	nv.DeleteProperty("pagecolor")
	nv.DeleteProperty("pageshadow")
	nv.DeleteProperty("pagecheckerboard")
	nv.DeleteProperty("showgrid")

	spc := fmt.Sprintf("%g", sv.Grid)
	gr.SetProperty("spacingx", spc)
	gr.SetProperty("spacingy", spc)
	gr.SetProperty("type", "xygrid")
	gr.SetProperty("units", uts)
}

// ReadMetaData reads meta data of drawing
func (sv *SVG) ReadMetaData() {
	es := sv.EditState()
	nv, gr := sv.MetaData(false)
	if nv == nil {
		return
	}
	if cx := nv.Property("cx"); cx != nil {
		sv.Trans.X, _ = reflectx.ToFloat32(cx)
	}
	if cy := nv.Property("cy"); cy != nil {
		sv.Trans.Y, _ = reflectx.ToFloat32(cy)
	}
	if zm := nv.Property("zoom"); zm != nil {
		sc, _ := reflectx.ToFloat32(zm)
		if sc > 0 {
			sv.Scale = sc
		}
	}
	if cl := nv.Property("current-layer"); cl != nil {
		es.CurLayer = reflectx.ToString(cl)
	}

	if gr == nil {
		return
	}
	if gs := gr.Property("spacingx"); gs != nil {
		gv, _ := reflectx.ToFloat32(gs)
		if gv > 0 {
			sv.Grid = gv
		}
	}
}

///////////////////////////////////////////////////////////////////////////
//  ContextMenu / Actions

// EditNode opens a form editor on node
func (sv *SVG) EditNode(kn tree.Node) {
	// core.FormDialog(sv, kn, "SVG Element View", true) // TODO:
}

// MakeNodeContextMenu makes the menu of options for context right click
func (sv *SVG) MakeNodeContextMenu(m *core.Scene, kn tree.Node) {
	core.NewButton(m).SetText("Edit").SetIcon(icons.Edit).OnClick(func(e events.Event) {
		sv.EditNode(kn)
	})
	core.NewButton(m).SetText("Select in tree").SetIcon(icons.Select).OnClick(func(e events.Event) {
		sv.Vector.SelectNodeInTree(kn, events.SelectOne)
	})

	core.NewSeparator(m)

	core.NewFuncButton(m).SetFunc(sv.Vector.DuplicateSelected).SetText("Duplicate").SetIcon(icons.Copy).SetKey(keymap.Duplicate)
	core.NewFuncButton(m).SetFunc(sv.Vector.CopySelected).SetText("Copy").SetIcon(icons.Copy).SetKey(keymap.Copy)
	core.NewFuncButton(m).SetFunc(sv.Vector.CutSelected).SetText("Cut").SetIcon(icons.Cut).SetKey(keymap.Cut)
	core.NewFuncButton(m).SetFunc(sv.Vector.PasteClip).SetText("Paste").SetIcon(icons.Paste).SetKey(keymap.Paste)
}

// ContextMenuPos returns position to use for context menu, based on input position
func (sv *SVG) NodeContextMenuPos(pos image.Point) image.Point {
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
func (sv *SVG) UndoSave(action, data string) {
	es := sv.EditState()
	if es == nil {
		return
	}
	es.Changed = true
	b := &bytes.Buffer{}
	errors.Log(jsonx.Write(sv.Root(), b))
	bs := strings.Split(b.String(), "\n")
	es.Undos.Save(action, data, bs)
}

// UndoSaveReplace save current state to replace current
func (sv *SVG) UndoSaveReplace(action, data string) {
	es := sv.EditState()
	b := &bytes.Buffer{}
	errors.Log(jsonx.Write(sv.Root(), b))
	bs := strings.Split(b.String(), "\n")
	es.Undos.SaveReplace(action, data, bs)
}

// Undo undoes one step, returning the action that was undone
func (sv *SVG) Undo() string {
	es := sv.EditState()
	es.ResetSelected()
	if es.Undos.MustSaveUndoStart() { // need to save current state!
		b := &bytes.Buffer{}
		errors.Log(jsonx.Write(sv.Root(), b))
		bs := strings.Split(b.String(), "\n")
		es.Undos.SaveUndoStart(bs)
	}
	act, _, state := es.Undos.Undo()
	if state == nil {
		return act
	}
	sb := strings.Join(state, "\n")
	b := bytes.NewBufferString(sb)
	errors.Log(jsonx.Read(sv.Root(), b))
	sv.UpdateSelect()
	return act
}

// Redo redoes one step, returning the action that was redone
func (sv *SVG) Redo() string {
	es := sv.EditState()
	es.ResetSelected()
	act, _, state := es.Undos.Redo()
	if state == nil {
		return act
	}
	sb := strings.Join(state, "\n")
	b := bytes.NewBufferString(sb)
	errors.Log(jsonx.Read(sv.Root(), b))
	sv.UpdateSelect()
	return act
}

///////////////////////////////////////////////////////////////////
// selection processing

// ShowAlignMatches draws the align matches as given
// between BBox Min - Max.  typs are corresponding bounding box sources.
func (sv *SVG) ShowAlignMatches(pts []image.Rectangle, typs []BBoxPoints) {
	// win := sv.Vector.ParentWindow()

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
func (sv *SVG) DepthMap() map[tree.Node]int {
	m := make(map[tree.Node]int)
	depth := 0
	n := tree.Next(sv.This)
	for n != nil {
		m[n] = depth
		depth++
		n = tree.Next(n)
	}
	return m
}

///////////////////////////////////////////////////////////////////////
// New objects

// SetSVGName sets the name of the element to standard type + id name
func (sv *SVG) SetSVGName(el svg.Node) {
	nwid := sv.SSVG().NewUniqueID()
	nwnm := fmt.Sprintf("%s%d", el.SVGName(), nwid)
	el.AsTree().SetName(nwnm)
}

// NewElement makes a new SVG element, giving it a new unique name.
// Uses currently active layer if set.
func (sv *SVG) NewElement(typ *types.Type) svg.Node {
	es := sv.EditState()
	parent := tree.Node(sv.Root())
	if es.CurLayer != "" {
		ly := sv.ChildByName(es.CurLayer, 1)
		if ly != nil {
			parent = ly
		}
	}
	newName := fmt.Sprintf("%s_tmp_new_item_", typ.Name)
	nw := parent.AsTree().NewChild(typ).(svg.Node)
	nw.AsTree().SetName(newName)
	sv.SetSVGName(nw)
	sv.Vector.PaintView().SetProperties(nw)
	sv.Vector.UpdateTree()
	return nw
}

// NewElementDrag makes a new SVG element during the drag operation
func (sv *SVG) NewElementDrag(typ *types.Type, start, end image.Point) svg.Node {
	minsz := float32(10)
	es := sv.EditState()
	dv := math32.Vector2FromPoint(end.Sub(start))
	if !es.InAction() && math32.Abs(dv.X) < minsz && math32.Abs(dv.Y) < minsz {
		return nil
	}
	// win := sv.Vector.ParentWindow()
	tn := typ.Name
	sv.ManipStart("New"+tn, "")
	// sv.SetFullReRender()
	nr := sv.NewElement(typ)
	xfi := sv.Root().Paint.Transform.Inverse()
	svoff := math32.Vector2FromPoint(sv.Geom.ContentBBox.Min)
	pos := math32.Vector2FromPoint(start).Sub(svoff)
	nr.SetNodePos(xfi.MulVector2AsPoint(pos))
	sz := dv.Abs().Max(math32.Vector2Scalar(minsz / 2))
	nr.SetNodeSize(xfi.MulVector2AsVector(sz))
	es.SelectAction(nr, events.SelectOne, end)
	sv.ManipDone()
	sv.NeedsRender()
	sv.UpdateSelSprites()
	es.DragSelStart(start)
	// win.SpriteDragging = SpriteName(SpReshapeBBox, SpBBoxDnR, 0)
	return nr
}

// NewText makes a new Text element with embedded tspan
func (sv *SVG) NewText(start, end image.Point) svg.Node {
	es := sv.EditState()
	sv.ManipStart("NewText", "")
	nr := sv.NewElement(svg.TextType)
	tsnm := fmt.Sprintf("tspan%d", sv.SSVG().NewUniqueID())
	tspan := svg.NewText(nr)
	tspan.SetName(tsnm)
	tspan.Text = "Text"
	tspan.Width = 200
	xfi := sv.Root().Paint.Transform.Inverse()
	svoff := math32.Vector2FromPoint(sv.Geom.ContentBBox.Min)
	pos := math32.Vector2FromPoint(start).Sub(svoff)
	// minsz := float32(20)
	pos.Y += 20 // todo: need the font size..
	pos = xfi.MulVector2AsPoint(pos)
	sv.Vector.SetTextPropertiesNode(nr, es.Text.TextProperties())
	// nr.Pos = pos
	// tspan.Pos = pos
	// // dv := math32.Vector2FromPoint(end.Sub(start))
	// // sz := dv.Abs().Max(math32.NewVector2Scalar(minsz / 2))
	// nr.Width = 100
	// tspan.Width = 100
	es.SelectAction(nr, events.SelectOne, end)
	// sv.UpdateView(true)
	// sv.UpdateSelect()
	return nr
}

// NewPath makes a new SVG Path element during the drag operation
func (sv *SVG) NewPath(start, end image.Point) *svg.Path {
	minsz := float32(10)
	es := sv.EditState()
	dv := math32.Vector2FromPoint(end.Sub(start))
	if !es.InAction() && math32.Abs(dv.X) < minsz && math32.Abs(dv.Y) < minsz {
		return nil
	}
	// win := sv.Vector.ParentWindow()
	sv.ManipStart("NewPath", "")
	// sv.SetFullReRender()
	nr := sv.NewElement(svg.PathType).(*svg.Path)
	xfi := sv.Root().Paint.Transform.Inverse()
	svoff := math32.Vector2FromPoint(sv.Geom.ContentBBox.Min)
	pos := math32.Vector2FromPoint(start).Sub(svoff)
	pos = xfi.MulVector2AsPoint(pos)
	sz := dv
	// sz := dv.Abs().Max(math32.NewVector2Scalar(minsz / 2))
	sz = xfi.MulVector2AsVector(sz)

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

// Gradients returns the currently defined gradients with stops
// that are shared among obj-specific ones
func (sv *SVG) Gradients() []*Gradient {
	gl := make([]*Gradient, 0)
	for _, gii := range sv.SSVG().Defs.Children {
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
func (sv *SVG) UpdateGradients(gl []*Gradient) {
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

// func (sv *SVG) Render() {
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

func (sv *SVG) BgNeedsUpdate() bool {
	update := sv.EnsureBgSize() || (sv.Trans != sv.bgTrans) || (sv.Scale != sv.bgScale) || (sv.VectorEff != sv.bgVectorEff)
	// fmt.Printf("update: %v\n", update)
	return update
}

func (sv *SVG) FillViewportWithBg() {
	if sv.BgNeedsUpdate() {
		sv.RenderBg()
	}
	draw.Draw(sv.Scene.Pixels, sv.Geom.ContentBBox, sv.BgPixels, sv.Geom.ScrollOffset(), draw.Over) // draw the bg first
}

// EnsureBgSize ensures Bg is set to the right size -- returns true if resized
func (sv *SVG) EnsureBgSize() bool {
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
func (sv *SVG) UpdateVectorEff() {
	sv.VectorEff = sv.Grid
	sp := sv.VectorEff * sv.Scale
	for sp <= 2*(float32(Settings.SnapTol)+1) {
		sv.VectorEff *= 2
		sp = sv.VectorEff * sv.Scale
	}
}

// RenderBg renders our background image
func (sv *SVG) RenderBg() {
	/*
		rs := &sv.BgRender
		pc := &rs.Paint
		sv.EnsureBgSize()

		sv.UpdateVectorEff()

		bb := sv.BgPixels.Bounds()

		draw.Draw(sv.BgPixels, bb, &image.Uniform{Settings.Colors.Background}, image.ZP, draw.Src)

		rs.PushBounds(bb)
		rs.PushTransform(sv.Pnt.Transform)

		pc.StrokeStyle.SetColor(&Settings.Colors.Border)

		sc := sv.Scale

		wd := 1 / sc
		pc.StrokeStyle.Width.Dots = wd
		pos := sv.ViewBox.Min
		sz := sv.ViewBox.Size
		pc.FillStyle.SetColor(nil)

		pc.DrawRectangle(rs, pos.X, pos.Y, sz.X, sz.Y)
		pc.FillStrokeClear(rs)

		if Settings.VectorDisp {
			gsz := float32(sv.VectorEff)
			pc.StrokeStyle.SetColor(&Settings.Colors.Vector)
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
