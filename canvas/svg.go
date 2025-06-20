// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package canvas

import (
	"bytes"
	"fmt"
	"strings"

	"cogentcore.org/cogent/canvas/cicons"
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/ppath"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/svg"
	"cogentcore.org/core/tree"
)

// SVG is the element for viewing and interacting with the SVG.
type SVG struct {
	core.Frame

	// SVG is the SVG drawing to display in this widget
	SVG *svg.SVG `set:"-"`

	// the parent [Canvas]
	Canvas *Canvas `copier:"-" json:"-" xml:"-" display:"-" set:"-"`

	// Grid spacing, in native ViewBox units.
	Grid float32 ` set:"-"`

	// GridPixels is the current pixel (dot) grid spacing, with scaling.
	// Will be a multiple of underlying Grid if < SnapZone.
	GridPixels math32.Vector2 `edit:"-" set:"-"`

	// GridOffset is the current pixel offset in mouse coordinates.
	GridOffset math32.Vector2 `edit:"-" set:"-"`
}

func (sv *SVG) Init() {
	sv.Frame.Init()
	sv.SVG = svg.NewSVG(math32.Vec2(10, 10))
	sv.SVG.Background = nil
	sv.Grid = Settings.Size.Grid
	sv.AddContextMenu(sv.contextMenu)
	sv.Styler(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Slideable, abilities.Activatable, abilities.Scrollable, abilities.Focusable, abilities.ScrollableUnattended, abilities.DoubleClickable)
		s.ObjectFit = styles.FitNone
		sv.SVG.Root.ViewBox.PreserveAspectRatio.SetFromStyle(s)
		sv.SVG.TextShaper = sv.Scene.TextShaper()
		s.StateLayer = 0 // always focused..
	})
	sv.OnKeyChord(func(e events.Event) {
		es := sv.EditState()
		kc := e.KeyChord()
		kf := keymap.Of(kc)
		// if core.DebugSettings.KeyEventTrace {
		// fmt.Println("SVG KeyInput:", kf, kc)
		// }
		switch kc {
		case " ", "s", "S":
			sv.Canvas.SetTool(SelectTool)
			e.SetHandled()
		case "b", "B":
			sv.Canvas.SetTool(SelBoxTool)
			e.SetHandled()
		case "n", "N":
			sv.Canvas.SetTool(NodeTool)
			e.SetHandled()
		case "r", "R":
			sv.Canvas.SetTool(RectTool)
			e.SetHandled()
		case "e", "E":
			sv.Canvas.SetTool(EllipseTool)
			e.SetHandled()
		case "d", "D":
			sv.Canvas.SetTool(BezierTool)
			e.SetHandled()
		case "t", "T":
			sv.Canvas.SetTool(TextTool)
			e.SetHandled()
		case "Alt+ReturnEnter":
			e.SetHandled()
			if es.Tool == BezierTool && es.ActivePath != nil {
				es.ActivePath.Data.Close()
				es.ActivePath = nil
				sv.UpdateView()
			}
		}
		if e.IsHandled() {
			return
		}
		switch kf {
		case keymap.Abort:
			e.SetHandled()
			if es.Tool == BezierTool {
				if es.ActivePath != nil && len(es.PathNodes) <= 1 {
					sv.Canvas.DeleteItems(es.ActivePath)
				}
				sv.NodeDeleteLast()
			}
			sv.Canvas.SetTool(SelectTool)
		case keymap.Enter:
			e.SetHandled()
			if es.Tool == BezierTool {
				es.ActivePath = nil
				sv.UpdateView()
			}
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
			if es.Tool == BezierTool {
				sv.NodeDeleteLast()
			} else {
				sv.Canvas.DeleteSelected()
			}
		}
	})
	sv.OnFirst(events.MouseDown, func(e events.Event) {
		if e.MouseButton() != events.Left {
			return
		}
		es := sv.EditState()
		isSelTool := (es.Tool == SelectTool) || ToolDoesBasicSelect(es.Tool)
		sv.SetFocusQuiet()
		var sob svg.Node
		if es.Tool == NodeTool {
			sob = sv.SelectContainsPoint(e.Pos(), true, true) // yes leavesonly, yes exclude existing sels
		} else {
			sob = sv.SelectContainsPoint(e.Pos(), false, true) // not leavesonly, yes exclude existing sels
		}
		es.MouseDownSel = sob

		// fmt.Println(sv.Styles.Cursor)

		es.SelectNoDrag = false
		switch {
		case es.Tool == BezierTool:
			pt := sv.DrawPoint(e)
			newPt := false
			if es.ActivePath == nil {
				es.ActivePath = NewSVGElement[svg.Path](sv, false)
				newPt = true
				es.DrawStartPos = pt
				sv.GatherAlignPoints()
			}
			switch {
			case newPt:
				sv.DrawNodeAdd(SpMoveTo, pt)
			case e.HasAnyModifier(key.Alt):
				sv.DrawNodeAdd(SpCubeTo, pt)
			default:
				sv.DrawNodeAdd(SpLineTo, pt)
			}
			sv.Canvas.PaintSetter().SetProperties(es.ActivePath)
			sv.UpdateView()
			e.SetHandled() // allows control to work here
		case isSelTool && es.HasSelected() && es.SelectBBox.ContainsPoint(math32.FromPoint(e.Pos())):
			// note: this absorbs potential secondary selections within selection -- handled
			// on release below, if nothing else happened
			es.SelectNoDrag = true // will be reset if drag
			es.DragSelStart(e.Pos())
		case sob == nil || es.Tool == SelBoxTool:
			es.DragStartPos = e.Pos()
			es.ResetSelected()
			es.ResetSelectedNodes()
			sv.RemoveNodeSprites()
			sv.UpdateSelect()
		case sob != nil && isSelTool:
			es.SelectAction(sob, e.SelectMode(), e.Pos())
			sv.EditState().DragSelStart(e.Pos())
			sv.UpdateSelect()
		case sob != nil && es.Tool == NodeTool:
			es.ResetSelectedNodes()
			es.ActivePath = nil
			es.SelectAction(sob, events.SelectOne, e.Pos())
			es.ActivePath = es.FirstSelectedPath()
			es.SelectedToRecents()
			sv.UpdateNodeSprites()
			sv.Canvas.UpdateTabs()
		}
	})
	sv.On(events.MouseMove, func(e events.Event) {
		sv.Styles.Cursor = cursors.Arrow // default
		es := sv.EditState()
		if es.Tool == BezierTool {
			sv.DrawPoint(e)
			sv.UpdateLineAddSprite()
		}
	})
	sv.On(events.MouseUp, func(e events.Event) {
		if e.MouseButton() != events.Left {
			return
		}
		es := sv.EditState()
		if es.Tool == BezierTool {
			es.DrawPos = e.Pos()
			return
		}
		isSelTool := (es.Tool == SelectTool) || ToolDoesBasicSelect(es.Tool)
		sob := es.MouseDownSel
		if es.SelectNoDrag && isSelTool { // do select on up to allow for drag of selected item on down
			es.SelectNoDrag = false
			e.SetHandled()
			if sob == nil {
				sob = sv.SelectContainsPoint(e.Pos(), false, false) // don't exclude existing sel
			}
			if sob != nil {
				es.SelectAction(sob, e.SelectMode(), e.Pos())
				sv.UpdateSelect()
			} else {
				if es.Tool != SelectTool { // click off = go to select
					sv.Canvas.SetTool(SelectTool)
				}
			}
		}
	})
	sv.On(events.SlideMove, func(e events.Event) {
		es := sv.EditState()
		if es.Tool == BezierTool {
			e.SetHandled()
			sv.DrawPoint(e)
			sv.UpdateLineAddSprite()
			return
		}
		// fmt.Println(sv.Styles.Cursor)
		es.SelectNoDrag = false
		es.DragStartPos = e.StartPos() // this is the operative start

		if e.HasAnyModifier(key.Shift) {
			e.SetHandled()
			sv.SVG.Translate.SetAdd(math32.FromPoint(e.PrevDelta()))
			sv.UpdateView()
			return
		}
		if es.HasSelected() {
			e.SetHandled()
			switch es.Action {
			case NewElement:
				sv.SpriteReshapeDrag(SpDnR, e)
			default:
				sv.DragMove(e)
			}
			return
		}
		if !es.InAction() {
			switch es.Tool {
			case SelectTool:
				if core.TheApp.SystemPlatform().IsMobile() { // fallthrough to frame scroll
					return
				}
				sv.SetRubberBand(e.Pos())
			case SelBoxTool:
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
			e.SetHandled()
		} else if es.Action == BoxSelect || es.Tool == SelBoxTool {
			sv.SetRubberBand(e.Pos())
			e.SetHandled()
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
		if (es.SelectNoDrag && es.Tool == SelectTool) || ToolDoesBasicSelect(es.Tool) {
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
		sv.SVG.ZoomAtScroll(se.Delta.Y, se.Pos())
		sv.UpdateView()
	})
	sv.On(events.DoubleClick, func(e events.Event) {
		es := sv.EditState()
		isSelTool := (es.Tool == SelectTool) || ToolDoesBasicSelect(es.Tool)
		if !isSelTool {
			return
		}
		itm := es.FirstSelected()
		if itm == nil {
			itm = sv.SelectContainsPoint(e.Pos(), false, true) // not leavesonly, yes exclude existing sels
		}
		if itm == nil {
			return
		}
		sv.EditNode(itm)
		sv.Canvas.SelectNodeInTree(itm, events.SelectOne)
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

// UpdateView updates the SVG view
func (sv *SVG) UpdateView() {
	sv.UpdateGridPixels()
	sv.SVG.UpdateBBoxes() // needs this to be updated
	sv.UpdateSelSprites()
	sv.UpdateNodeSprites()
	sv.NeedsRender()
	sv.SetFocus()
}

// SpritesNoLock returns the [core.Sprites] without locking.
func (sv *SVG) SpritesNoLock() *core.Sprites {
	return &sv.Scene.Stage.Sprites
}

// SpritesLock returns the [core.Sprites] under mutex lock.
func (sv *SVG) SpritesLock() *core.Sprites {
	sprites := sv.SpritesNoLock()
	sprites.Lock()
	return sprites
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
		sni.ApplyTransform(sv.SVG, sni.AsNodeBase().DeltaTransform(trans, scale, rot, pt))
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
	root := sv.Root()
	if root.NumChildren() > 0 {
		kd := root.Children[0]
		if md, ismd := kd.(*svg.MetaData); ismd {
			main = md
		}
	}
	if main == nil && mknew && Settings.MetaData {
		id := sv.SVG.NewUniqueID()
		main = svg.NewMetaData()
		root.InsertChild(main, 0)
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
	if nv == nil {
		return
	}

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

func (sv *SVG) contextMenu(m *core.Scene) {
	es := sv.EditState()
	itm := es.FirstSelected()
	if itm != nil {
		sv.contextMenuNode(m, itm)
		return
	}
	core.NewFuncButton(m).SetFunc(sv.Canvas.PasteClip).
		SetText("Paste").SetIcon(icons.Paste).SetKey(keymap.Paste)
}

func (sv *SVG) contextMenuNode(m *core.Scene, nd tree.Node) {
	cv := sv.Canvas
	es := sv.EditState()
	core.NewButton(m).SetText("Edit").SetIcon(icons.Edit).OnClick(func(e events.Event) {
		sv.EditNode(nd)
	})
	core.NewButton(m).SetText("Select in tree").SetIcon(icons.Select).OnClick(func(e events.Event) {
		cv.SelectNodeInTree(nd, events.SelectOne)
	})

	core.NewSeparator(m)

	core.NewFuncButton(m).SetFunc(cv.DuplicateSelected).
		SetText("Duplicate").SetIcon(icons.Copy).SetKey(keymap.Duplicate)
	core.NewFuncButton(m).SetFunc(cv.CopySelected).
		SetText("Copy").SetIcon(icons.Copy).SetKey(keymap.Copy)
	core.NewFuncButton(m).SetFunc(cv.CutSelected).
		SetText("Cut").SetIcon(icons.Cut).SetKey(keymap.Cut)
	core.NewFuncButton(m).SetFunc(cv.PasteClip).
		SetText("Paste").SetIcon(icons.Paste).SetKey(keymap.Paste)

	core.NewSeparator(m)

	added := false
	if len(es.Selected) > 1 {
		added = true
		core.NewFuncButton(m).SetFunc(cv.SelectGroup).SetText("Group").
			SetIcon(cicons.SelGroup).SetShortcut("Command+G")
	} else {
		if _, isgp := nd.(*svg.Group); isgp {
			added = true
			core.NewFuncButton(m).SetFunc(cv.SelectUnGroup).SetText("Ungroup").
				SetIcon(cicons.SelUngroup).SetShortcut("Command+Shift+G")
		}
	}

	if added {
		core.NewSeparator(m)
	}

	core.NewFuncButton(m).SetFunc(cv.SelectRotateLeft).SetText("").
		SetIcon(cicons.SelRotateLeft).SetShortcut("Command+[")
	core.NewFuncButton(m).SetFunc(cv.SelectRotateRight).SetText("").
		SetIcon(cicons.SelRotateRight).SetShortcut("Command+]")
	core.NewFuncButton(m).SetFunc(cv.SelectFlipHorizontal).SetText("").
		SetIcon(cicons.SelFlipHoriz)
	core.NewFuncButton(m).SetFunc(cv.SelectFlipVertical).SetText("").
		SetIcon(cicons.SelFlipVert)
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
	errors.Log(sv.SVG.WriteXML(b, false))
	bs := strings.Split(b.String(), "\n")
	es.Undos.Save(action, data, bs)
}

// UndoSaveReplace save current state to replace current
func (sv *SVG) UndoSaveReplace(action, data string) {
	es := sv.EditState()
	b := &bytes.Buffer{}
	errors.Log(sv.SVG.WriteXML(b, false))
	bs := strings.Split(b.String(), "\n")
	es.Undos.SaveReplace(action, data, bs)
}

// Undo undoes one step, returning the action that was undone
func (sv *SVG) Undo() string {
	es := sv.EditState()
	es.ResetSelected()
	es.ResetSelectedNodes()
	if es.Undos.MustSaveUndoStart() { // need to save current state!
		b := &bytes.Buffer{}
		errors.Log(sv.SVG.WriteXML(b, false))
		bs := strings.Split(b.String(), "\n")
		es.Undos.SaveUndoStart(bs)
	}
	act, _, state := es.Undos.Undo()
	if state == nil {
		return act
	}
	sb := strings.Join(state, "\n")
	b := bytes.NewBufferString(sb)
	errors.Log(sv.SVG.ReadXML(b))
	sv.UpdateSelect()
	return act
}

// Redo redoes one step, returning the action that was redone
func (sv *SVG) Redo() string {
	es := sv.EditState()
	es.ResetSelected()
	es.ResetSelectedNodes()
	act, _, state := es.Undos.Redo()
	if state == nil {
		return act
	}
	sb := strings.Join(state, "\n")
	b := bytes.NewBufferString(sb)
	errors.Log(sv.SVG.ReadXML(b))
	sv.UpdateSelect()
	return act
}

////////  Grid render

// UpdateGridPixels updates the GirdEff value based on current scale
func (sv *SVG) UpdateGridPixels() {
	if sv.Grid == 0 { // shouldn't happen!
		sv.Grid = 16
	}
	tx, ty, _, sx, sy, _ := sv.Root().Paint.Transform.Decompose()
	sv.GridPixels = math32.Vec2(sx, sy).MulScalar(sv.Grid)
	tol := 2 * (float32(Settings.SnapZone) + 1)
	for sv.GridPixels.X < tol || sv.GridPixels.Y < tol {
		sv.GridPixels.SetMulScalar(2)
	}
	tl := math32.Vec2(tx, ty)
	sv.GridOffset = tl
}

// RenderGrid renders the background grid
func (sv *SVG) RenderGrid() {
	root := sv.Root()
	if root == nil || !Settings.ShowGrid {
		return
	}

	pc := &sv.Scene.Painter
	pc.PushContext(&root.Paint, nil) // gets root transform
	pc.Stroke.Color = colors.Scheme.OutlineVariant
	pc.Fill.Color = nil

	// sc := sv.SVG.Scale
	// wd := 1 / sc
	pc.VectorEffect = ppath.VectorEffectNonScalingStroke
	pc.Stroke.Width.Dp(1)
	pc.Stroke.Width.Dots = 1
	pos := root.ViewBox.Min
	sz := root.ViewBox.Size

	pc.Rectangle(pos.X, pos.Y, sz.X, sz.Y)
	if Settings.ShowGrid {
		gsz := float32(sv.Grid)
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
