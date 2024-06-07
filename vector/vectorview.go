// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package vector implements a 2D vector graphics program.
package vector

//go:generate core generate

import (
	"fmt"
	"image"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"cogentcore.org/core/base/dirs"
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/svg"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/views"
)

// VectorView is the Vector SVG vector drawing program
type VectorView struct {
	core.Frame

	// full path to current drawing filename
	Filename core.Filename `ext:".svg" set:"-"`

	// current edit state
	EditState EditState `set:"-"`
}

func (vv *VectorView) Init() {
	vv.Frame.Init()
	vv.EditState.ConfigDefaultGradient()
	vv.Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})
	vv.OnWidgetAdded(func(w core.Widget) {
		switch w.PathFrom(vv) {
		case "splits/tabs": // TODO(config)
			w.(*core.Tabs).SetType(core.FunctionalTabs)
		}
	})

	vv.AddCloseDialog()

	core.AddChild(vv, func(w *core.Frame) {
		w.SetName("modal-tb")
		w.Style(func(s *styles.Style) {
			s.Display = styles.Stacked
		})
	})

	core.AddChildAt(vv, "hbox", func(w *core.Frame) {

		core.AddChildAt(w, "tools", func(w *core.Toolbar) {
			w.Style(func(s *styles.Style) {
				s.Direction = styles.Column
			})
		})

		core.AddChildAt(w, "splits", func(w *core.Splits) {
			w.SetSplits(0.15, 0.60, 0.25)

			core.AddChildAt(w, "layer-tree", func(w *core.Frame) {
				w.Style(func(s *styles.Style) {
					s.Direction = styles.Column
				})

				core.AddChild(w, func(w *views.FuncButton) {
					w.SetFunc(vv.AddLayer)
				})

				core.AddChildAt(w, "layers", func(w *views.TableView) {
					w.SetSlice(&vv.EditState.Layers)
				})

				core.AddChildAt(w, "tree-frame", func(w *core.Frame) {
					w.Style(func(s *styles.Style) {
						s.Direction = styles.Column
					})
					core.AddChildAt(w, "treeview", func(w *views.TreeView) {
						// w.VectorView = vv
						w.OpenDepth = 4
						w.Updater(func() {
							// TODO: get SVG
							// tv.SyncTree(sv.Root())
						})
					})
				})
			})

			core.AddChildAt(w, "svg", func(w *SVGView) {
				w.VectorView = vv
				w.UpdateGradients(vv.EditState.Gradients)
			})
			core.AddChildAt(w, "tabs", func(w *core.Tabs) {
			})
		})

		core.AddChildAt(w, "statusbar", func(w *core.Frame) {
			w.Style(func(s *styles.Style) {
				s.Grow.Set(1, 0)
			})
		})
	})

	// tv.TreeViewSig.Connect(vv.This(), func(recv, send tree.Node, sig int64, data any) {
	// 	gvv := recv.Embed(KiT_VectorView).(*VectorView)
	// 	if data == nil {
	// 		return
	// 	}
	// 	if sig == int64(views.TreeViewInserted) {
	// 		sn, ok := data.(svg.Node)
	// 		if ok {
	// 			gvv.SVG().NodeEnsureUniqueId(sn)
	// 			svg.CloneNodeGradientProp(sn, "fill")
	// 			svg.CloneNodeGradientProp(sn, "stroke")
	// 		}
	// 		return
	// 	}
	// 	if sig == int64(views.TreeViewDeleted) {
	// 		sn, ok := data.(svg.Node)
	// 		if ok {
	// 			svg.DeleteNodeGradientProp(sn, "fill")
	// 			svg.DeleteNodeGradientProp(sn, "stroke")
	// 		}
	// 		return
	// 	}
	// 	if sig != int64(views.TreeViewOpened) {
	// 		return
	// 	}
	// 	tvn, _ := data.(tree.Node).Embed(KiT_TreeView).(*TreeView)
	// 	_, issvg := tvn.SrcNode.(svg.Node)
	// 	if !issvg {
	// 		return
	// 	}
	// 	if tvn.SrcNode.HasChildren() {
	// 		return
	// 	}
	// 	views.StructViewDialog(gvv.Viewport, tvn.SrcNode, views.DlgOpts{Title: "SVG Element View"}, nil, nil)
	// 	// ggv, _ := recv.Embed(KiT_VectorView).(*VectorView)
	// 	// 		stv := ggv.RecycleTab("Obj", views.KiT_StructView, true).(*views.StructView)
	// 	// 		stv.SetStruct(tvn.SrcNode)
	// })

	vv.ConfigStatusBar()
	vv.ConfigModalToolbar()
	vv.ConfigTools()
	vv.ConfigTabs()

	vv.SetPhysSize(&Settings.Size)

	vv.SyncLayers()

}

// OpenDrawingFile opens a new .svg drawing file -- just the basic opening
func (vv *VectorView) OpenDrawingFile(fnm core.Filename) error {
	path, _ := filepath.Abs(string(fnm))
	vv.Filename = core.Filename(path)
	sv := vv.SVG()
	err := errors.Log(sv.SSVG().OpenXML(path))
	// SavedPaths.AddPath(path, core.Settings.Params.SavedPathsMax)
	// SavePaths()
	fdir, _ := filepath.Split(path)
	errors.Log(os.Chdir(fdir))
	vv.EditState.Init(vv)
	vv.UpdateLayerView()

	vv.EditState.Gradients = sv.Gradients()
	sv.SSVG().GatherIDs() // also ensures uniqueness, key for json saving
	sv.ZoomToContents(false)
	sv.ReadMetaData()
	sv.SetTransform()
	return err
}

// OpenDrawing opens a new .svg drawing
func (vv *VectorView) OpenDrawing(fnm core.Filename) error { //types:add
	err := vv.OpenDrawingFile(fnm)

	sv := vv.SVG()
	vv.SetTitle()
	tv := vv.TreeView()
	tv.CloseAll()
	tv.ReSync()
	vv.SetStatus("Opened: " + string(vv.Filename))
	tv.CloseAll()
	sv.bgVectorEff = 0
	sv.UpdateView(true)
	vv.NeedsRender()
	return err
}

// NewDrawing creates a new drawing of the given size
func (vv *VectorView) NewDrawing(sz PhysSize) *VectorView {
	ngr := NewDrawing(sz)
	return ngr
}

// PromptPhysSize prompts for the physical size of the drawing and sets it
func (vv *VectorView) PromptPhysSize() { //types:add
	sv := vv.SVG()
	sz := &PhysSize{}
	sz.SetFromSVG(sv)
	d := core.NewBody().AddTitle("SVG physical size")
	views.NewStructView(d).SetStruct(sz)
	d.AddBottomBar(func(parent core.Widget) {
		d.AddCancel(parent)
		d.AddOK(parent).OnClick(func(e events.Event) {
			vv.SetPhysSize(sz)
			sv.bgVectorEff = -1
			sv.UpdateView(true)
		})
	})
	d.RunDialog(vv)
}

// SetPhysSize sets physical size of drawing
func (vv *VectorView) SetPhysSize(sz *PhysSize) {
	if sz == nil {
		return
	}
	if sz.Size == (math32.Vector2{}) {
		sz.SetStandardSize(Settings.Size.StandardSize)
	}
	sv := vv.SVG()
	sz.SetToSVG(sv)
	sv.SetMetaData()
	sv.ZoomToPage(false)
}

// SaveDrawing saves .svg drawing to current filename
func (vv *VectorView) SaveDrawing() error { //types:add
	if vv.Filename != "" {
		return vv.SaveDrawingAs(vv.Filename)
	}
	views.CallFunc(vv, vv.SaveDrawingAs)
	return nil
}

// SaveDrawingAs saves .svg drawing to given filename
func (vv *VectorView) SaveDrawingAs(fname core.Filename) error { //types:add
	if fname == "" {
		return errors.New("SaveDrawingAs: filename is empty")
	}
	path, _ := filepath.Abs(string(fname))
	vv.Filename = core.Filename(path)
	// SavedPaths.AddPath(path, core.Settings.Params.SavedPathsMax)
	// SavePaths()
	sv := vv.SVG()
	sv.SSVG().RemoveOrphanedDefs()
	sv.SetMetaData()
	err := sv.SSVG().SaveXML(path)
	if errors.Log(err) == nil {
		vv.AutoSaveDelete()
	}
	vv.SetTitle()
	vv.SetStatus("Saved: " + path)
	vv.EditState.Changed = false
	return err
}

// TODO(kai): don't use inkscape for exporting

// ExportPNG exports drawing to a PNG image (auto-names to same name
// with .png suffix).  Calls inkscape -- needs to be on the PATH.
// specify either width or height of resulting image, or nothing for
// physical size as set.  Renders full current page -- do ResizeToContents
// to render just current contents.
func (vv *VectorView) ExportPNG(width, height float32) error { //types:add
	path, _ := filepath.Split(string(vv.Filename))
	fnm := filepath.Join(path, "export_png.svg")
	sv := vv.SVG()
	err := sv.SSVG().SaveXML(fnm)
	if errors.Log(err) != nil {
		return err
	}
	fext := filepath.Ext(string(vv.Filename))
	onm := strings.TrimSuffix(string(vv.Filename), fext) + ".png"
	cstr := "inkscape"
	args := []string{`--export-type=png`, "-o", onm}
	if width > 0 {
		args = append(args, fmt.Sprintf("--export-width=%g", width))
	}
	if height > 0 {
		args = append(args, fmt.Sprintf("--export-height=%g", height))
	}
	args = append(args, fnm)
	cmd := exec.Command(cstr, args...)
	fmt.Printf("executing command: %s %v\n", cstr, args)
	out, err := cmd.CombinedOutput()
	// if err != nil {
	fmt.Println(string(out))
	// }
	os.Remove(fnm)
	return err
}

// ExportPDF exports drawing to a PDF file (auto-names to same name
// with .pdf suffix).  Calls inkscape -- needs to be on the PATH.
// specify DPI of resulting image for effects rendering.
// Renders full current page -- do ResizeToContents
// to render just current contents.
func (vv *VectorView) ExportPDF(dpi float32) error { //types:add
	path, _ := filepath.Split(string(vv.Filename))
	fnm := filepath.Join(path, "export_pdf.svg")
	sv := vv.SVG()
	err := sv.SSVG().SaveXML(fnm)
	if errors.Log(err) != nil {
		return err
	}
	fext := filepath.Ext(string(vv.Filename))
	onm := strings.TrimSuffix(string(vv.Filename), fext) + ".pdf"
	cstr := "inkscape"
	args := []string{`--export-type=pdf`, "-o", onm}
	if dpi > 0 {
		args = append(args, fmt.Sprintf("--export-dpi=%g", dpi))
	}
	args = append(args, fnm)
	cmd := exec.Command(cstr, args...)
	fmt.Printf("executing command: %s %v\n", cstr, args)
	out, err := cmd.CombinedOutput()
	// if err != nil {
	fmt.Println(string(out))
	// }
	os.Remove(fnm)
	return err
}

// ResizeToContents resizes the drawing to just fit the current contents,
// including moving everything to start at upper-left corner,
// preserving the current grid offset, so grid snapping
// is preserved.
func (vv *VectorView) ResizeToContents() { //types:add
	sv := vv.SVG()
	sv.ResizeToContents(true)
	sv.UpdateView(true)
}

// AddImage adds a new image node set to the given image
func (vv *VectorView) AddImage(fname core.Filename, width, height float32) error { //types:add
	sv := vv.SVG()
	sv.UndoSave("AddImage", string(fname))
	ind := sv.NewEl(svg.ImageType).(*svg.Image)
	ind.Pos.X = 100 // todo: default pos
	ind.Pos.Y = 100 // todo: default pos
	err := ind.OpenImage(string(fname), width, height)
	sv.UpdateView(true)
	vv.ChangeMade()
	return err
}

//////////////////////////////////////////////////////////////////////////
//  GUI Config

func (vv *VectorView) ModalToolbarStack() *core.Frame {
	return vv.ChildByName("modal-tb", 1).(*core.Frame)
}

// SetModalSelect sets the modal toolbar to be the select one
func (vv *VectorView) SetModalSelect() {
	tbs := vv.ModalToolbarStack()
	vv.UpdateSelectToolbar()
	tbs.StackTop = 0
	tbs.NeedsLayout()
}

// SetModalNode sets the modal toolbar to be the node editing one
func (vv *VectorView) SetModalNode() {
	tbs := vv.ModalToolbarStack()
	vv.UpdateNodeToolbar()
	tbs.StackTop = 1
	tbs.NeedsLayout()
}

// SetModalText sets the modal toolbar to be the text editing one
func (vv *VectorView) SetModalText() {
	tbs := vv.ModalToolbarStack()
	vv.UpdateTextToolbar()
	tbs.StackTop = 2
	tbs.NeedsLayout()
}

func (vv *VectorView) HBox() *core.Frame {
	return vv.ChildByName("hbox", 2).(*core.Frame)
}

func (vv *VectorView) Tools() *core.Toolbar {
	return vv.HBox().ChildByName("tools", 0).(*core.Toolbar)
}

func (vv *VectorView) Splits() *core.Splits {
	return vv.HBox().ChildByName("splits", 1).(*core.Splits)
}

func (vv *VectorView) LayerTree() *core.Frame {
	return vv.Splits().ChildByName("layer-tree", 0).(*core.Frame)
}

func (vv *VectorView) LayerView() *views.TableView {
	return vv.LayerTree().ChildByName("layers", 0).(*views.TableView)
}

func (vv *VectorView) TreeView() *TreeView {
	return vv.LayerTree().ChildByName("tree-frame", 1).Child(0).(*TreeView)
}

// SVG returns the [SVGView].
func (vv *VectorView) SVG() *SVGView {
	return vv.Splits().Child(1).(*SVGView)
}

// SSVG returns the underlying [svg.SVG].
func (vv *VectorView) SSVG() *svg.SVG {
	return vv.SVG().SSVG()
}

func (vv *VectorView) Tabs() *core.Tabs {
	return vv.Splits().ChildByName("tabs", 2).(*core.Tabs)
}

// StatusBar returns the statusbar widget
func (vv *VectorView) StatusBar() *core.Frame {
	return vv.ChildByName("statusbar", 4).(*core.Frame)
}

// StatusText returns the status bar text widget
func (vv *VectorView) StatusText() *core.Text {
	return vv.StatusBar().Child(0).(*core.Text)
}

// PasteAvailFunc is an ActionUpdateFunc that inactivates action if no paste avail
func (vv *VectorView) PasteAvailFunc(bt *core.Button) {
	bt.SetEnabled(!vv.Clipboard().IsEmpty())
}

func (vv *VectorView) MakeToolbar(tb *core.Toolbar) { // TODO(config)
	// TODO(kai): remove Update
	views.NewFuncButton(tb, vv.UpdateAll).SetText("Update").SetIcon(icons.Update)
	core.NewButton(tb).SetText("New").SetIcon(icons.Add).
		OnClick(func(e events.Event) {
			ndr := vv.NewDrawing(Settings.Size)
			ndr.PromptPhysSize()
		})

	core.NewButton(tb).SetText("Size").SetIcon(icons.FormatSize).SetMenu(func(m *core.Scene) {
		views.NewFuncButton(m, vv.PromptPhysSize).SetText("Set size").
			SetIcon(icons.FormatSize)
		views.NewFuncButton(m, vv.ResizeToContents).SetIcon(icons.Resize)
	})

	views.NewFuncButton(tb, vv.OpenDrawing).SetText("Open").SetIcon(icons.Open)
	views.NewFuncButton(tb, vv.SaveDrawing).SetText("Save").SetIcon(icons.Save)
	views.NewFuncButton(tb, vv.SaveDrawingAs).SetText("Save as").SetIcon(icons.SaveAs)

	core.NewButton(tb).SetText("Export").SetIcon(icons.ExportNotes).SetMenu(func(m *core.Scene) {
		views.NewFuncButton(m, vv.ExportPNG).SetIcon(icons.Image)
		views.NewFuncButton(m, vv.ExportPDF).SetIcon(icons.PictureAsPdf)
	})

	core.NewSeparator(tb)

	views.NewFuncButton(tb, vv.Undo).StyleFirst(func(s *styles.Style) {
		s.SetEnabled(vv.EditState.Undos.HasUndoAvail())
	})
	views.NewFuncButton(tb, vv.Redo).StyleFirst(func(s *styles.Style) {
		s.SetEnabled(vv.EditState.Undos.HasRedoAvail())
	})

	core.NewSeparator(tb)

	views.NewFuncButton(tb, vv.DuplicateSelected).SetText("Duplicate").SetIcon(icons.Copy).SetKey(keymap.Duplicate)
	views.NewFuncButton(tb, vv.CopySelected).SetText("Copy").SetIcon(icons.Copy).SetKey(keymap.Copy)
	views.NewFuncButton(tb, vv.CutSelected).SetText("Cut").SetIcon(icons.Cut).SetKey(keymap.Cut)
	views.NewFuncButton(tb, vv.PasteClip).SetText("Paste").SetIcon(icons.Paste).SetKey(keymap.Paste)

	core.NewSeparator(tb)
	views.NewFuncButton(tb, vv.AddImage).SetIcon(icons.Image)
	core.NewSeparator(tb)

	core.NewButton(tb).SetText("Zoom page").SetIcon(icons.ZoomOut).
		SetTooltip("Zoom to see the entire page size for drawing").
		OnClick(func(e events.Event) {
			sv := vv.SVG()
			sv.ZoomToPage(false)
			sv.UpdateView(true)
		})

	core.NewButton(tb).SetText("Zoom all").SetIcon(icons.ZoomOut).
		SetTooltip("Zoom to see all elements").
		OnClick(func(e events.Event) {
			sv := vv.SVG()
			sv.ZoomToContents(false)
			sv.UpdateView(true)
		})
}

func (vv *VectorView) ConfigModalToolbar() {
	tb := vv.ModalToolbarStack()
	if tb == nil || tb.HasChildren() {
		return
	}
	core.NewToolbar(tb).SetName("select-tb")
	core.NewToolbar(tb).SetName("node-tb")
	core.NewToolbar(tb).SetName("text-tb")

	vv.ConfigSelectToolbar()
	vv.ConfigNodeToolbar()
	vv.ConfigTextToolbar()
}

// ConfigStatusBar configures statusbar with text
func (vv *VectorView) ConfigStatusBar() {
	sb := vv.StatusBar()
	if sb == nil || sb.HasChildren() {
		return
	}
	core.NewText(sb).SetName("sb-text")
}

// SetStatus updates the status bar text with the given message, along with other status info
func (vv *VectorView) SetStatus(msg string) {
	sb := vv.StatusBar()
	if sb == nil {
		return
	}
	text := vv.StatusText()
	es := &vv.EditState
	str := "<b>" + strings.TrimSuffix(es.Tool.String(), "Tool") + "</b>\t"
	if es.CurLayer != "" {
		str += "Layer: " + es.CurLayer + "\t\t"
	}
	str += msg
	text.SetText(str)
}

// AddCloseDialog adds the close dialog that prompts the user to save the
// file when they try to close the scene containing this vector view.
func (vv *VectorView) AddCloseDialog() {
	vv.WidgetBase.AddCloseDialog(func(d *core.Body) bool {
		if !vv.EditState.Changed {
			return false
		}
		d.AddTitle("Unsaved changes").
			AddText(fmt.Sprintf("There are unsaved changes in %s", dirs.DirAndFile(string(vv.Filename))))
		d.AddBottomBar(func(parent core.Widget) {
			d.AddOK(parent).SetText("Close without saving").OnClick(func(e events.Event) {
				vv.Scene.Close()
			})
			d.AddOK(parent).SetText("Save and close").OnClick(func(e events.Event) {
				vv.SaveDrawing()
				vv.Scene.Close()
			})
		})
		return true
	})
}

func (vv *VectorView) SetTitle() {
	if vv.Filename == "" {
		return
	}
	win := vv.Scene.RenderWindow()
	if win == nil {
		return
	}
	dfnm := dirs.DirAndFile(string(vv.Filename))
	winm := "Cogent Vector • " + dfnm
	win.SetName(winm)
	win.SetTitle(winm)
	vv.Scene.Body.Title = winm
}

// NewDrawing opens a new drawing window
func NewDrawing(sz PhysSize) *VectorView {
	ngr := NewVectorWindow("")
	ngr.SetPhysSize(&sz)
	return ngr
}

// NewVectorWindow returns a new VectorWindow loading given file if non-empty
func NewVectorWindow(fnm string) *VectorView {
	path := ""
	dfnm := "blank"
	if fnm != "" {
		path, _ = filepath.Abs(fnm)
		dfnm = dirs.DirAndFile(path)
	}
	winm := "Cogent Vector • " + dfnm

	if win, found := core.AllRenderWindows.FindName(winm); found {
		sc := win.MainScene()
		if vv, ok := sc.Body.ChildByType(VectorViewType, tree.NoEmbeds).(*VectorView); ok {
			if string(vv.Filename) == path {
				win.Raise()
				return vv
			}
		}
	}

	b := core.NewBody(winm).SetTitle(winm)

	vv := NewVectorView(b)
	// b.AddAppBar(vv.MakeToolbar) // TODO(config):

	b.OnShow(func(e events.Event) {
		if fnm != "" {
			vv.OpenDrawingFile(core.Filename(path))
		} else {
			vv.EditState.Init(vv)
		}
	})

	b.RunWindow()

	return vv
}

/////////////////////////////////////////////////////////////////////////
//   Controls

// RecycleTab returns the tab with given the name, first by looking for
// an existing one, and if not found, making a new one.
// If sel, then select it.
func (gv *VectorView) RecycleTab(name string, sel bool) *core.Frame {
	tv := gv.Tabs()
	return tv.RecycleTab(name, sel)
}

// Tab returns the tab with the given name
func (gv *VectorView) Tab(name string) *core.Frame {
	return gv.Tabs().TabByName(name)
}

func (vv *VectorView) ConfigTabs() {
	pt := vv.RecycleTab("Paint", false)
	NewPaintView(pt).SetVectorView(vv)
	at := vv.RecycleTab("Align", false)
	NewAlignView(at).SetVectorView(vv)
	vv.EditState.Text.Defaults()
	tt := vv.RecycleTab("Text", false)
	views.NewStructView(tt).SetStruct(&vv.EditState.Text)
}

func (vv *VectorView) PaintView() *PaintView {
	return vv.Tab("Paint").Child(0).(*PaintView)
}

// UpdateAll updates the display
func (vv *VectorView) UpdateAll() { //types:add
	vv.UpdateTabs()
	vv.UpdateTreeView()
	vv.UpdateDisp()
}

func (vv *VectorView) UpdateDisp() {
	sv := vv.SVG()
	sv.UpdateView(true)
}

func (vv *VectorView) UpdateTreeView() {
	tv := vv.TreeView()
	tv.ReSync()
}

func (vv *VectorView) SetDefaultStyle() {
	// pv := vv.PaintView()
	// es := &vv.EditState
	// switch es.Tool {
	// case TextTool:
	// 	pv.Update(&Settings.TextStyle, nil)
	// case BezierTool:
	// 	pv.Update(&Settings.PathStyle, nil)
	// default:
	// 	pv.Update(&Settings.ShapeStyle, nil)
	// }
}

func (vv *VectorView) UpdateTabs() {
	// es := &vv.EditState
	// fsel := es.FirstSelectedNode()
	// if fsel != nil {
	// 	sel := fsel.AsNodeBase()
	// 	pv := vv.PaintView()
	// 	pv.Update(&sel.Paint, sel.This())
	// 	txt, istxt := fsel.(*svg.Text)
	// 	if istxt {
	// 		es.Text.SetFromNode(txt)
	// 		txv := vv.Tab("Text").(*views.StructView)
	// 		txv.UpdateFields()
	// 		// todo: only show text toolbar on double-click
	// 		// gv.SetModalText()
	// 		// gv.UpdateTextToolbar()
	// 	} else {
	// 		vv.SetModalToolbar()
	// 	}
	// }
}

// SelectNodeInSVG selects given svg node in SVG drawing
func (vv *VectorView) SelectNodeInSVG(kn tree.Node, mode events.SelectModes) {
	sii, ok := kn.(svg.Node)
	if !ok {
		return
	}
	sv := vv.SVG()
	es := &vv.EditState
	es.SelectAction(sii, mode, image.Point{})
	sv.UpdateView(false)
}

// Undo undoes the last action
func (vv *VectorView) Undo() string { //types:add
	sv := vv.SVG()
	act := sv.Undo()
	if act != "" {
		vv.SetStatus("Undid: " + act)
	} else {
		vv.SetStatus("Undo: no more to undo")
	}
	vv.UpdateAll()
	return act
}

// Redo redoes the previously undone action
func (vv *VectorView) Redo() string { //types:add
	sv := vv.SVG()
	act := sv.Redo()
	if act != "" {
		vv.SetStatus("Redid: " + act)
	} else {
		vv.SetStatus("Redo: no more to redo")
	}
	vv.UpdateAll()
	return act
}

// ChangeMade should be called after any change is completed on the drawing.
// Calls autosave.
func (vv *VectorView) ChangeMade() {
	go vv.AutoSave()
}

/////////////////////////////////////////////////////////////////////////
//   Basic infrastructure

/*
func (gv *VectorView) OSFileEvent() {
	gv.ConnectEvent(oswin.OSOpenFilesEvent, core.RegPri, func(recv, send tree.Node, sig int64, d any) {
		ofe := d.(*osevent.OpenFilesEvent)
		for _, fn := range ofe.Files {
			NewVectorWindow(fn)
		}
	})
}
*/

// OpenRecent opens a recently used file
func (vv *VectorView) OpenRecent(filename core.Filename) {
	// if string(filename) == VectorViewResetRecents {
	// 	SavedPaths = nil
	// 	core.StringsAddExtras((*[]string)(&SavedPaths), SavedPathsExtras)
	// } else if string(filename) == VectorViewEditRecents {
	// 	vv.EditRecents()
	// } else {
	// 	vv.OpenDrawing(filename)
	// }
}

// RecentsEdit opens a dialog editor for deleting from the recents project list
func (vv *VectorView) EditRecents() {
	// tmp := make([]string, len(SavedPaths))
	// copy(tmp, SavedPaths)
	// core.StringsRemoveExtras((*[]string)(&tmp), SavedPathsExtras)
	// opts := views.DlgOpts{Title: "Recent Project Paths", Prompt: "Delete paths you no longer use", Ok: true, Cancel: true, NoAdd: true}
	// views.SliceViewDialog(vv.Viewport, &tmp, opts,
	// 	nil, vv, func(recv, send tree.Node, sig int64, data any) {
	// 		if sig == int64(core.DialogAccepted) {
	// 			SavedPaths = nil
	// 			SavedPaths = append(SavedPaths, tmp...)
	// 			core.StringsAddExtras((*[]string)(&SavedPaths), SavedPathsExtras)
	// 		}
	// 	})
}

// SplitsSetView sets split view splitters to given named setting
func (vv *VectorView) SplitsSetView(split SplitName) {
	sv := vv.Splits()
	sp, _, ok := AvailableSplits.SplitByName(split)
	if ok {
		sv.SetSplits(sp.Splits...).NeedsLayout()
		Settings.SplitName = split
	}
}

// SplitsSave saves current splitter settings to named splitter settings under
// existing name, and saves to prefs file
func (vv *VectorView) SplitsSave(split SplitName) {
	sv := vv.Splits()
	sp, _, ok := AvailableSplits.SplitByName(split)
	if ok {
		sp.SaveSplits(sv.Splits)
		AvailableSplits.SaveSettings()
	}
}

// SplitsSaveAs saves current splitter settings to new named splitter settings, and
// saves to prefs file
func (vv *VectorView) SplitsSaveAs(name, desc string) {
	spv := vv.Splits()
	AvailableSplits.Add(name, desc, spv.Splits)
	AvailableSplits.SaveSettings()
}

// SplitsEdit opens the SplitsView editor to customize saved splitter settings
func (vv *VectorView) SplitsEdit() {
	SplitsView(&AvailableSplits)
}

// HelpWiki opens wiki page for grid on github
func (vv *VectorView) HelpWiki() {
	core.TheApp.OpenURL("https://vector.cogentcore.org")
}

////////////////////////////////////////////////////////////////////////////////////////
//		AutoSave

// AutoSaveFilename returns the autosave filename
func (vv *VectorView) AutoSaveFilename() string {
	path, fn := filepath.Split(string(vv.Filename))
	if fn == "" {
		fn = "new_file_" + vv.Nm + ".svg"
	}
	asfn := filepath.Join(path, "#"+fn+"#")
	return asfn
}

// AutoSave does the autosave -- safe to call in a separate goroutine
func (vv *VectorView) AutoSave() error {
	// if vv.HasFlag(int(VectorViewAutoSaving)) {
	// 	return nil
	// }
	// vv.SetFlag(int(VectorViewAutoSaving))
	// asfn := vv.AutoSaveFilename()
	// sv := vv.SVG()
	// err := sv.SaveXML(core.Filename(asfn))
	// if err != nil && err != io.EOF {
	// 	log.Println(err)
	// }
	// vv.ClearFlag(int(VectorViewAutoSaving))
	// return err
	return nil
}

// AutoSaveDelete deletes any existing autosave file
func (vv *VectorView) AutoSaveDelete() {
	asfn := vv.AutoSaveFilename()
	os.Remove(asfn)
}

// AutoSaveCheck checks if an autosave file exists -- logic for dealing with
// it is left to larger app -- call this before opening a file
func (vv *VectorView) AutoSaveCheck() bool {
	asfn := vv.AutoSaveFilename()
	if _, err := os.Stat(asfn); os.IsNotExist(err) {
		return false // does not exist
	}
	return true
}

// VectorViewFlags extend WidgetFlags to hold VectorView state
type VectorViewFlags core.WidgetFlags //enums:bitflag -trim-prefix VectorViewFlag

const (
	// VectorViewAutoSaving means
	VectorViewAutoSaving VectorViewFlags = VectorViewFlags(core.WidgetFlagsN) + iota
)
