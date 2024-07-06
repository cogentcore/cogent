// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package canvas

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"strings"
	"sync"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/iox/jsonx"
	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/svg"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/types"
)

// SVG is the element for viewing and interacting with the SVG.
type SVG struct {
	core.WidgetBase

	// SVG is the SVG drawing to display in this widget
	SVG *svg.SVG `set:"-"`

	// the parent [Canvas]
	Canvas *Canvas `copier:"-" json:"-" xml:"-" display:"-" set:"-"`

	// grid spacing, in native ViewBox units
	Grid float32 ` set:"-"`

	// effective grid spacing given Scale level
	GridEff float32 `edit:"-" set:"-"`

	// pixelMu is a mutex protecting the updating of curPixels
	// from svg render
	pixelMu sync.Mutex

	// currentPixels are the current rendered pixels, with the SVG
	// on top of the background pixel grid.  updated in separate
	// goroutine, protected by pixelMu, to ensure fluid interaction
	currentPixels *image.RGBA

	// background pixels, includes page outline and grid
	backgroundPixels *image.RGBA

	// background paint rendering context
	backgroundPaint paint.Context

	// in svg Rendering
	inRender bool

	// size of bg image rendered
	backgroundSize image.Point

	// bg rendered transform
	backgroundTransform math32.Matrix2

	// bg rendered grid
	backgroundGridEff float32
}

func (sv *SVG) Init() {
	sv.WidgetBase.Init()
	sv.SVG = svg.NewSVG(10, 10)
	sv.SVG.Background = nil
	sv.Grid = Settings.Size.Grid
	sv.Styler(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Slideable, abilities.Activatable, abilities.Scrollable, abilities.Focusable)
		s.ObjectFit = styles.FitNone
		sv.SVG.Root.ViewBox.PreserveAspectRatio.SetFromStyle(s)
		s.Cursor = cursors.Arrow // todo: modulate based on tool etc
	})
	sv.OnKeyChord(func(e events.Event) {
		kc := e.KeyChord()
		// if core.DebugSettings.KeyEventTrace {
		fmt.Printf("SVG KeyInput: %v\n", sv.Path())
		// }
		kf := keymap.Of(kc)
		switch kf {
		case keymap.Abort:
			// todo: maybe something else
			e.SetHandled()
			sv.Canvas.SetTool(SelectTool)
		case keymap.Undo:
			e.SetHandled()
			sv.Canvas.Undo()
		case keymap.Redo:
			e.SetHandled()
			sv.Canvas.Redo()
		case keymap.Duplicate:
			e.SetHandled()
			sv.Canvas.DuplicateSelected()
		case keymap.Copy:
			e.SetHandled()
			sv.Canvas.CopySelected()
		case keymap.Cut:
			e.SetHandled()
			sv.Canvas.CutSelected()
		case keymap.Paste:
			e.SetHandled()
			sv.Canvas.PasteClip()
		case keymap.Delete, keymap.Backspace:
			e.SetHandled()
			sv.Canvas.DeleteSelected()
		}
	})
	sv.On(events.MouseDown, func(e events.Event) {
		if e.MouseButton() != events.Left {
			return
		}
		sv.SetFocus()
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
			es.DragStartPos = e.Pos()
			es.ResetSelected()
			sv.UpdateSelect()
		}
	})
	sv.On(events.MouseUp, func(e events.Event) {
		if e.MouseButton() != events.Left {
			return
		}
		es := sv.EditState()
		sob := sv.SelectContainsPoint(e.Pos(), false, true) // not leavesonly, yes exclude existing sels
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
	})
	sv.On(events.SlideMove, func(e events.Event) {
		es := sv.EditState()
		es.SelectNoDrag = false
		e.SetHandled()
		es.DragStartPos = e.StartPos()
		if e.HasAnyModifier(key.Shift) {
			del := e.PrevDelta()
			sv.SVG.Translate.X += float32(del.X)
			sv.SVG.Translate.Y += float32(del.Y)
			go sv.RenderSVG()
			return
		}
		if es.HasSelected() {
			switch es.Action {
			case NewElement:
				sv.SpriteReshapeDrag(SpBBoxDnR, e)
			default:
				sv.DragMove(e)
			}
			return
		}
		if !es.InAction() {
			switch es.Tool {
			case SelectTool:
				sv.SetRubberBand(e.Pos())
			case RectTool:
				NewSVGElementDrag[svg.Rect](sv, es.DragStartPos, e.Pos())
			case EllipseTool:
				NewSVGElementDrag[svg.Ellipse](sv, es.DragStartPos, e.Pos())
			case TextTool:
				sv.NewText(es.DragStartPos, e.Pos())
				es.NewTextMade = true
			case BezierTool:
				sv.NewPath(es.DragStartPos, e.Pos())
			}
		} else if es.Action == BoxSelect {
			sv.SetRubberBand(e.Pos())
		}
	})
	sv.On(events.SlideStop, func(e events.Event) {
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
	})
	sv.On(events.Scroll, func(e events.Event) {
		e.SetHandled()
		se := e.(*events.MouseScroll)
		svoff := sv.Geom.ContentBBox.Min
		sv.ZoomAt(se.Pos().Sub(svoff), se.Delta.Y/100)
		// sv.SVG.Scale += float32(se.Delta.Y) / 100
		// if sv.SVG.Scale <= 0.0000001 {
		// 	sv.SVG.Scale = 0.01
		// }
		go sv.RenderSVG()
	})
}

func (sv *SVG) SizeFinal() {
	sv.WidgetBase.SizeFinal()
	sv.SVG.Resize(sv.Geom.Size.Actual.Content.ToPoint())
	sv.ResizeBg(sv.Geom.Size.Actual.Content.ToPoint())
}

// RenderSVG renders the SVG, typically called in a goroutine
func (sv *SVG) RenderSVG() {
	if sv.SVG == nil || sv.inRender {
		return
	}
	sv.inRender = true
	defer func() { sv.inRender = false }()

	if sv.BackgroundNeedsUpdate() {
		sv.RenderBackground()
	}
	// need to make the image again to prevent it from
	// rendering over itself
	sv.SVG.Background = nil
	sv.SVG.Pixels = image.NewRGBA(sv.SVG.Pixels.Rect)
	sv.SVG.RenderState.Init(sv.SVG.Pixels.Rect.Dx(), sv.SVG.Pixels.Rect.Dy(), sv.SVG.Pixels)
	sv.SVG.Render()
	sv.pixelMu.Lock()

	bgsz := sv.backgroundPixels.Bounds()
	sv.currentPixels = image.NewRGBA(bgsz)
	draw.Draw(sv.currentPixels, bgsz, sv.backgroundPixels, image.ZP, draw.Src)
	draw.Draw(sv.currentPixels, bgsz, sv.SVG.Pixels, image.ZP, draw.Over)
	sv.NeedsRender()
	sv.pixelMu.Unlock()
}

func (sv *SVG) Render() {
	sv.WidgetBase.Render()
	if sv.SVG == nil {
		return
	}
	sv.pixelMu.Lock()
	if sv.currentPixels == nil || sv.BackgroundNeedsUpdate() {
		sv.pixelMu.Unlock()
		sv.RenderSVG()
		sv.pixelMu.Lock()
	}
	r := sv.Geom.ContentBBox
	sp := sv.Geom.ScrollOffset()
	draw.Draw(sv.Scene.Pixels, r, sv.currentPixels, sp, draw.Over)
	sv.pixelMu.Unlock()
}

// Root returns the root [svg.Root].
func (sv *SVG) Root() *svg.Root {
	return sv.SVG.Root
}

// EditState returns the EditState for this view
func (sv *SVG) EditState() *EditState {
	if sv.Canvas == nil {
		return nil
	}
	return &sv.Canvas.EditState
}

// UpdateView updates the view, optionally with a full re-render
func (sv *SVG) UpdateView(full bool) { // TODO(config)
	sv.UpdateSelSprites()
	go sv.RenderSVG()
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
		if n == sv.SVG.Defs {
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
		if n == sv.SVG.Defs {
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
		sni.ApplyDeltaTransform(sv.SVG, trans, scale, rot, pt)
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
	sv.SVG.Translate.Set(0, 0)
	if width {
		sv.SVG.Scale = sc.X
	} else {
		sv.SVG.Scale = math32.Min(sc.X, sc.Y)
	}
}

// ZoomToContents sets the scale to fit the current contents into view
func (sv *SVG) ZoomToContents(width bool) {
	vb := math32.Vector2FromPoint(sv.Root().BBox.Size())
	if vb == (math32.Vector2{}) {
		return
	}
	sv.ZoomToPage(width)
	bb := sv.ContentsBBox()
	bsz := bb.Size()
	if bsz.X <= 0 || bsz.Y <= 0 {
		return
	}
	sc := vb.Div(bsz)
	sv.SVG.Translate = bb.Min.DivScalar(sv.SVG.Scale).Negate()
	if width {
		sv.SVG.Scale *= sc.X
	} else {
		sv.SVG.Scale *= math32.Min(sc.X, sc.Y)
	}
	sv.UpdateView(true)
}

// ResizeToContents resizes the drawing to just fit the current contents,
// including moving everything to start at upper-left corner,
// optionally preserving the current grid offset, so grid snapping
// is preserved -- recommended.
func (sv *SVG) ResizeToContents(grid_off bool) {
	sv.UndoSave("ResizeToContents", "")
	sv.ZoomToPage(false)
	bb := sv.ContentsBBox()
	bsz := bb.Size()
	if bsz.X <= 0 || bsz.Y <= 0 {
		return
	}
	trans := bb.Min
	incr := sv.Grid * sv.SVG.Scale // our zoom factor
	treff := trans
	if grid_off {
		treff.X = math32.Floor(trans.X/incr) * incr
		treff.Y = math32.Floor(trans.Y/incr) * incr
	}
	bsz.SetAdd(trans.Sub(treff))
	treff = treff.Negate()

	bsz = bsz.DivScalar(sv.SVG.Scale)

	sv.TransformAllLeaves(treff, math32.Vec2(1, 1), 0, math32.Vec2(0, 0))
	sv.Root().ViewBox.Size = bsz
	sv.SVG.PhysicalWidth.Value = bsz.X
	sv.SVG.PhysicalHeight.Value = bsz.Y
	sv.ZoomToPage(false)
	sv.Canvas.ChangeMade()
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

	nsc := sv.SVG.Scale * sc

	mpt := math32.Vector2FromPoint(pt)
	lpt := mpt.DivScalar(sv.SVG.Scale).Sub(sv.SVG.Translate) // point in drawing coords

	dt := lpt.Add(sv.SVG.Translate).MulScalar((nsc - sv.SVG.Scale) / nsc) // delta from zooming
	sv.SVG.Translate.SetSub(dt)

	sv.SVG.Scale = nsc
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
		id := sv.SVG.NewUniqueID()
		main = svg.NewMetaData()
		sv.Root().InsertChild(main, 0)
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
		id := sv.SVG.NewUniqueID()
		grid = svg.NewMetaData()
		main.InsertChild(grid, 0)
		grid.SetName(svg.NameID("grid", id))
	}
	return
}

// SetMetaData sets meta data of drawing
func (sv *SVG) SetMetaData() {
	es := sv.EditState()
	nv, gr := sv.MetaData(true)

	uts := strings.ToLower(sv.SVG.PhysicalWidth.Unit.String())

	nv.SetProperty("inkscape:current-layer", es.CurLayer)
	nv.SetProperty("inkscape:cx", fmt.Sprintf("%g", sv.SVG.Translate.X))
	nv.SetProperty("inkscape:cy", fmt.Sprintf("%g", sv.SVG.Translate.Y))
	nv.SetProperty("inkscape:zoom", fmt.Sprintf("%g", sv.SVG.Scale))
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
		sv.SVG.Translate.X, _ = reflectx.ToFloat32(cx)
	}
	if cy := nv.Property("cy"); cy != nil {
		sv.SVG.Translate.Y, _ = reflectx.ToFloat32(cy)
	}
	if zm := nv.Property("zoom"); zm != nil {
		sc, _ := reflectx.ToFloat32(zm)
		if sc > 0 {
			sv.SVG.Scale = sc
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
func (sv *SVG) EditNode(kn tree.Node) { //types:add
	core.FormDialog(sv, kn, "SVG Element View", true)
}

// MakeNodeContextMenu makes the menu of options for context right click
func (sv *SVG) MakeNodeContextMenu(m *core.Scene, kn tree.Node) {
	core.NewButton(m).SetText("Edit").SetIcon(icons.Edit).OnClick(func(e events.Event) {
		sv.EditNode(kn)
	})
	core.NewButton(m).SetText("Select in tree").SetIcon(icons.Select).OnClick(func(e events.Event) {
		sv.Canvas.SelectNodeInTree(kn, events.SelectOne)
	})

	core.NewSeparator(m)

	core.NewFuncButton(m).SetFunc(sv.Canvas.DuplicateSelected).SetText("Duplicate").SetIcon(icons.Copy).SetKey(keymap.Duplicate)
	core.NewFuncButton(m).SetFunc(sv.Canvas.CopySelected).SetText("Copy").SetIcon(icons.Copy).SetKey(keymap.Copy)
	core.NewFuncButton(m).SetFunc(sv.Canvas.CutSelected).SetText("Cut").SetIcon(icons.Cut).SetKey(keymap.Cut)
	core.NewFuncButton(m).SetFunc(sv.Canvas.PasteClip).SetText("Paste").SetIcon(icons.Paste).SetKey(keymap.Paste)
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
	sz := min(len(pts), 8)
	svoff := sv.Geom.ContentBBox.Min
	for i := 0; i < sz; i++ {
		pt := pts[i].Canon()
		lsz := pt.Max.Sub(pt.Min)
		sp := Sprite(sv, SpAlignMatch, Sprites(typs[i]), i, lsz, nil)
		SetSpritePos(sp, pt.Min.Add(svoff))
	}
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
	nwid := sv.SVG.NewUniqueID()
	nwnm := fmt.Sprintf("%s%d", el.SVGName(), nwid)
	el.AsTree().SetName(nwnm)
}

// NewSVGElement makes a new SVG element of the given type.
// It uses the current active layer if it is set.
func NewSVGElement[T tree.NodeValue](sv *SVG) *T {
	es := sv.EditState()
	parent := tree.Node(sv.Root())
	if es.CurLayer != "" {
		ly := sv.ChildByName(es.CurLayer, 1)
		if ly != nil {
			parent = ly
		}
	}
	n := tree.New[T](parent)
	sn := any(n).(svg.Node)
	sv.SetSVGName(sn)
	sv.Canvas.PaintView().SetProperties(sn)
	sv.Canvas.UpdateTree()
	return n
}

// NewSVGElementDrag makes a new SVG element of the given type during the drag operation.
func NewSVGElementDrag[T tree.NodeValue](sv *SVG, start, end image.Point) *T {
	minsz := float32(10)
	es := sv.EditState()
	dv := math32.Vector2FromPoint(end.Sub(start))
	if !es.InAction() && math32.Abs(dv.X) < minsz && math32.Abs(dv.Y) < minsz {
		// fmt.Println("dv under min:", dv, minsz)
		return nil
	}
	sv.ManipStart(NewElement, types.For[T]().IDName)
	n := NewSVGElement[T](sv)
	sn := any(n).(svg.Node)
	xfi := sv.Root().Paint.Transform.Inverse()
	svoff := math32.Vector2FromPoint(sv.Geom.ContentBBox.Min)
	pos := math32.Vector2FromPoint(start).Sub(svoff)
	pos = xfi.MulVector2AsPoint(pos)
	sn.SetNodePos(pos)
	sz := dv.Abs().Max(math32.Vector2Scalar(minsz / 2))
	sz = xfi.MulVector2AsVector(sz)
	sn.SetNodeSize(sz)
	sv.RenderSVG() // needed to get bb
	es.SelectAction(sn, events.SelectOne, end)
	sv.NeedsRender()
	sv.UpdateSelSprites()
	es.DragSelStart(start)
	return n
}

// NewText makes a new Text element with embedded tspan
func (sv *SVG) NewText(start, end image.Point) svg.Node {
	es := sv.EditState()
	sv.ManipStart(NewText, "")
	n := NewSVGElement[svg.Text](sv)
	tsnm := fmt.Sprintf("tspan%d", sv.SVG.NewUniqueID())
	tspan := svg.NewText(n)
	tspan.SetName(tsnm)
	tspan.Text = "Text"
	tspan.Width = 200
	xfi := sv.Root().Paint.Transform.Inverse()
	svoff := math32.Vector2FromPoint(sv.Geom.ContentBBox.Min)
	pos := math32.Vector2FromPoint(start).Sub(svoff)
	// minsz := float32(20)
	pos.Y += 20 // todo: need the font size..
	pos = xfi.MulVector2AsPoint(pos)
	sv.Canvas.SetTextPropertiesNode(n, es.Text.TextProperties())
	// nr.Pos = pos
	// tspan.Pos = pos
	// // dv := math32.Vector2FromPoint(end.Sub(start))
	// // sz := dv.Abs().Max(math32.NewVector2Scalar(minsz / 2))
	// nr.Width = 100
	// tspan.Width = 100
	es.SelectAction(n, events.SelectOne, end)
	// sv.UpdateView(true)
	// sv.UpdateSelect()
	return n
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
	sv.ManipStart(NewPath, "")
	// sv.SetFullReRender()
	n := NewSVGElement[svg.Path](sv)
	xfi := sv.Root().Paint.Transform.Inverse()
	// svoff := math32.Vector2FromPoint(sv.Geom.ContentBBox.Min)
	pos := math32.Vector2FromPoint(start)
	pos = xfi.MulVector2AsPoint(pos)
	sz := dv
	// sz := dv.Abs().Max(math32.NewVector2Scalar(minsz / 2))
	sz = xfi.MulVector2AsVector(sz)

	n.SetData(fmt.Sprintf("m %g,%g %g,%g", pos.X, pos.Y, sz.X, sz.Y))

	es.SelectAction(n, events.SelectOne, end)
	sv.UpdateSelSprites()
	sv.EditState().DragSelStart(start)

	es.SelectBBox.Min.X += 1
	es.SelectBBox.Min.Y += 1
	es.DragSelectStartBBox = es.SelectBBox
	es.DragSelectCurrentBBox = es.SelectBBox
	es.DragSelectEffectiveBBox = es.SelectBBox

	// win.SpriteDragging = SpriteName(SpReshapeBBox, SpBBoxDnR, 0)
	return n
}

///////////////////////////////////////////////////////////////////////
// Gradients

// Gradients returns the currently defined gradients with stops
// that are shared among obj-specific ones
func (sv *SVG) Gradients() []*Gradient {
	gl := make([]*Gradient, 0)
	for _, gii := range sv.SVG.Defs.Children {
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
			id := sv.SVG.NewUniqueID()
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
	// 	gg := sv.SVG.FindDefByName(gr.Name)
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

func (sv *SVG) BackgroundNeedsUpdate() bool {
	root := sv.Root()
	if root == nil {
		return false
	}
	return sv.backgroundPixels == nil || sv.backgroundPixels.Bounds().Size() != sv.backgroundSize || sv.backgroundTransform != root.Paint.Transform || sv.GridEff != sv.backgroundGridEff || sv.NeedsRebuild()
}

func (sv *SVG) ResizeBg(sz image.Point) {
	if sv.backgroundPaint.State == nil {
		sv.backgroundPaint.State = &paint.State{}
	}
	if sv.backgroundPaint.Paint == nil {
		sv.backgroundPaint.Paint = &styles.Paint{}
		sv.backgroundPaint.Paint.Defaults()
	}
	if sv.backgroundPixels == nil || sv.backgroundPixels.Bounds().Size() != sz {
		sv.backgroundPixels = image.NewRGBA(image.Rectangle{Max: sz})
		sv.backgroundPaint.Init(sz.X, sz.Y, sv.backgroundPixels)
	}
}

// UpdateGridEff updates the GirdEff value based on current scale
func (sv *SVG) UpdateGridEff() {
	sv.GridEff = sv.Grid
	sp := sv.GridEff * sv.SVG.Scale
	for sp <= 2*(float32(Settings.SnapTol)+1) {
		sv.GridEff *= 2
		sp = sv.GridEff * sv.SVG.Scale
	}
}

// RenderBackground renders our background grid image
func (sv *SVG) RenderBackground() {
	root := sv.Root()
	if root == nil {
		return
	}
	sv.UpdateGridEff()
	bb := sv.backgroundPixels.Bounds()
	draw.Draw(sv.backgroundPixels, bb, colors.Scheme.Surface, image.ZP, draw.Src)

	pc := &sv.backgroundPaint
	pc.PushBounds(bb)
	pc.PushTransform(root.Paint.Transform)

	pc.StrokeStyle.Color = colors.Scheme.Outline

	sc := sv.SVG.Scale

	wd := 1 / sc
	pc.StrokeStyle.Width.Dots = wd
	pos := math32.Vec2(0, 0)
	sz := root.ViewBox.Size
	pc.FillStyle.Color = nil

	pc.DrawRectangle(pos.X, pos.Y, sz.X, sz.Y)
	pc.FillStrokeClear()

	if Settings.GridDisp {
		gsz := float32(sv.GridEff)
		pc.StrokeStyle.Color = colors.Scheme.OutlineVariant
		for x := gsz; x < sz.X; x += gsz {
			pc.DrawLine(x, 0, x, sz.Y)
		}
		for y := gsz; y < sz.Y; y += gsz {
			pc.DrawLine(0, y, sz.X, y)
		}
		pc.FillStrokeClear()
	}

	sv.backgroundTransform = root.Paint.Transform
	sv.backgroundGridEff = sv.GridEff
	sv.backgroundSize = bb.Size()

	pc.PopTransform()
	pc.PopBounds()
}
