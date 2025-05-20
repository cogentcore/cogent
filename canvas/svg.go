// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package canvas

import (
	"bytes"
	"fmt"
	"strings"

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
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/svg"
	"cogentcore.org/core/tree"
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

	// bg rendered grid
	backgroundGridEff float32
}

func (sv *SVG) Init() {
	sv.WidgetBase.Init()
	sv.SVG = svg.NewSVG(math32.Vec2(10, 10))
	sv.SVG.Background = nil
	sv.Grid = Settings.Size.Grid
	sv.Styler(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Slideable, abilities.Activatable, abilities.Scrollable, abilities.Focusable, abilities.ScrollableUnattended)
		s.ObjectFit = styles.FitNone
		sv.SVG.Root.ViewBox.PreserveAspectRatio.SetFromStyle(s)
		s.Cursor = cursors.Arrow // todo: modulate based on tool etc
		sv.SVG.TextShaper = sv.Scene.TextShaper()
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
		sv.SetFocusQuiet()
		e.SetHandled()
		es := sv.EditState()
		sob := sv.SelectContainsPoint(e.Pos(), false, true) // not leavesonly, yes exclude existing sels

		es.SelectNoDrag = false
		switch {
		case es.HasSelected() && es.SelectBBox.ContainsPoint(math32.FromPoint(e.Pos())):
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
			// fmt.Println("Drag start:", es.DragStartPos) // todo: not nec
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
		es.DragStartPos = e.StartPos() // this is the operative start
		// fmt.Println("sm drag start:", es.DragStartPos)
		if e.HasAnyModifier(key.Shift) {
			del := math32.FromPoint(e.PrevDelta())
			if sv.SVG.Scale > 0 {
				del.SetDivScalar(min(1, sv.SVG.Scale))
			}
			sv.SVG.Translate.SetAdd(del)
			sv.UpdateSelSprites()
			sv.NeedsRender()
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
	sv.On(events.Scroll, func(e events.Event) {
		e.SetHandled()
		se := e.(*events.MouseScroll)
		del := se.Delta.Y / 100
		if sv.SVG.Scale > 0 {
			del /= min(1, sv.SVG.Scale)
		}
		sv.SVG.ZoomAt(se.Pos(), del)
		sv.UpdateSelSprites()
		sv.NeedsRender()
	})
}

func (sv *SVG) SizeFinal() {
	sv.WidgetBase.SizeFinal()
	sv.SVG.SetSize(sv.Geom.Size.Actual.Content)
}

func (sv *SVG) Render() {
	sv.WidgetBase.Render()
	if sv.SVG == nil {
		return
	}
	sv.RenderGrid()
	sv.SVG.SetSize(sv.Geom.Size.Actual.Content)
	sv.SVG.Geom.Pos = sv.Geom.Pos.Content.ToPointCeil()
	sv.SVG.Render(&sv.Scene.Painter)
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
	sv.SVG.UpdateBBoxes() // needs this to be updated
	sv.UpdateSelSprites()
	sv.NeedsRender()
}

// SpritesNolock returns the [core.Sprites] without locking.
func (sv *SVG) SpritesNolock() *core.Sprites {
	return &sv.Scene.Stage.Sprites
}

// SpritesLock returns the [core.Sprites] under mutex lock.
func (sv *SVG) SpritesLock() *core.Sprites {
	sprites := sv.SpritesNolock()
	sprites.Lock()
	return sprites
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

// ResizeToContents resizes the drawing to just fit the current contents,
// including moving everything to start at upper-left corner,
// optionally preserving the current grid sizing, so grid snapping
// is preserved, which is recommended.
func (sv *SVG) ResizeToContents(gridIncr bool) {
	sv.UndoSave("ResizeToContents", "")
	grid := float32(1)
	if gridIncr {
		grid = sv.Grid
	}
	sv.SVG.ResizeToContents(grid)
	sv.Canvas.ChangeMade()
	sv.NeedsRender()
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

	// get rid of inkscape properties we don't set
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

// EditNode opens a [core.Form] dialog on the given node.
func (sv *SVG) EditNode(n tree.Node) { //types:add
	d := core.NewBody("Edit node")
	core.NewForm(d).SetStruct(n)
	d.RunWindowDialog(sv)
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

	core.NewFuncButton(m).SetFunc(sv.Canvas.DuplicateSelected).
		SetText("Duplicate").SetIcon(icons.Copy).SetKey(keymap.Duplicate)
	core.NewFuncButton(m).SetFunc(sv.Canvas.CopySelected).
		SetText("Copy").SetIcon(icons.Copy).SetKey(keymap.Copy)
	core.NewFuncButton(m).SetFunc(sv.Canvas.CutSelected).
		SetText("Cut").SetIcon(icons.Cut).SetKey(keymap.Cut)
	core.NewFuncButton(m).SetFunc(sv.Canvas.PasteClip).
		SetText("Paste").SetIcon(icons.Paste).SetKey(keymap.Paste)
}

//////// Undo

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

/////// Gradients

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

////////  Bg render

// UpdateGridEff updates the GirdEff value based on current scale
func (sv *SVG) UpdateGridEff() {
	sv.GridEff = sv.Grid
	sp := sv.GridEff * sv.SVG.Scale
	for sp <= 2*(float32(Settings.SnapTol)+1) {
		sv.GridEff *= 2
		sp = sv.GridEff * sv.SVG.Scale
	}
}

// RenderGrid renders the background grid
func (sv *SVG) RenderGrid() {
	root := sv.Root()
	if root == nil {
		return
	}
	sv.UpdateGridEff()

	pc := &sv.Scene.Painter
	pc.PushContext(&root.Paint, nil)
	pc.Stroke.Color = colors.Scheme.OutlineVariant
	pc.Fill.Color = nil

	sc := sv.SVG.Scale

	wd := 1 / sc
	pc.Stroke.Width.Dots = wd
	pos := root.ViewBox.Min
	sz := root.ViewBox.Size

	pc.Rectangle(pos.X, pos.Y, sz.X, sz.Y)
	if Settings.GridDisp {
		gsz := float32(sv.GridEff)
		for x := gsz; x < sz.X; x += gsz {
			pc.Line(pos.X+x, pos.Y, pos.X+x, pos.Y+sz.Y)
		}
		for y := gsz; y < sz.Y; y += gsz {
			pc.Line(pos.X, pos.Y+y, pos.X+sz.X, pos.Y+y)
		}
	}
	pc.Draw()
	pc.PopContext()
}
