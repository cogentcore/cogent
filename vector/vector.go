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

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/svg"
	"cogentcore.org/core/tree"
)

// Vector is the main widget of the Cogent Vector SVG vector graphics program.
type Vector struct {
	core.Frame

	// full path to current drawing filename
	Filename core.Filename `ext:".svg" set:"-"`

	// current edit state
	EditState EditState `set:"-"`
}

func (vc *Vector) Init() {
	vc.Frame.Init()
	vc.EditState.ConfigDefaultGradient()
	vc.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
	})

	vc.AddCloseDialog(func(d *core.Body) bool {
		if !vc.EditState.Changed {
			return false
		}
		d.AddTitle("Unsaved changes").
			AddText(fmt.Sprintf("There are unsaved changes in %s", fsx.DirAndFile(string(vc.Filename))))
		d.AddBottomBar(func(parent core.Widget) {
			d.AddOK(parent).SetText("Close without saving").OnClick(func(e events.Event) {
				vc.Scene.Close()
			})
			d.AddOK(parent).SetText("Save and close").OnClick(func(e events.Event) {
				vc.SaveDrawing()
				vc.Scene.Close()
			})
		})
		return true
	})

	tree.AddChildAt(vc, "modal-tb", func(w *core.Toolbar) {
		w.Maker(func(p *tree.Plan) {
			switch vc.EditState.Tool {
			case NodeTool:
				vc.MakeNodeToolbar(p)
			case TextTool:
				vc.MakeTextToolbar(p)
			default:
				vc.MakeSelectToolbar(p)
			}
		})
	})

	tree.AddChildAt(vc, "hbox", func(w *core.Frame) {
		w.Styler(func(s *styles.Style) {
			s.Grow.Set(1, 1)
		})
		tree.AddChildAt(w, "tools", func(w *core.Toolbar) {
			w.Styler(func(s *styles.Style) {
				s.Direction = styles.Column
			})
			w.Maker(vc.MakeTools)
		})
		tree.AddChildAt(w, "splits", func(w *core.Splits) {
			w.SetSplits(0.15, 0.60, 0.25)
			tree.AddChildAt(w, "layer-tree", func(w *core.Frame) {
				w.Styler(func(s *styles.Style) {
					s.Direction = styles.Column
				})
				tree.AddChild(w, func(w *core.FuncButton) {
					w.SetFunc(vc.AddLayer)
				})
				tree.AddChildAt(w, "layers", func(w *core.Table) {
					w.Styler(func(s *styles.Style) {
						s.Max.Y.Em(10)
					})
					w.SetSlice(&vc.EditState.Layers)
				})
				tree.AddChildAt(w, "tree-frame", func(w *core.Frame) {
					w.Styler(func(s *styles.Style) {
						s.Direction = styles.Column
						s.Grow.Set(0, 1)
					})
					tree.AddChildAt(w, "tree", func(w *Tree) {
						w.Vector = vc
						w.OpenDepth = 4
						w.SyncTree(vc.SVG().Root())
					})
				})
			})
			tree.AddChildAt(w, "svg", func(w *SVG) {
				w.Vector = vc
				w.UpdateGradients(vc.EditState.Gradients)
				vc.SetPhysSize(&Settings.Size)
				vc.SyncLayers()
			})
			tree.AddChildAt(w, "tabs", func(w *core.Tabs) {
				w.SetType(core.FunctionalTabs)
				pt := w.NewTab("Paint")
				NewPaintView(pt).SetVector(vc)
				at := w.NewTab("Align")
				NewAlignView(at).SetVector(vc)
				vc.EditState.Text.Defaults()
				tt := w.NewTab("Text")
				core.NewForm(tt).SetStruct(&vc.EditState.Text)
			})
		})
	})
	tree.AddChildAt(vc, "status-bar", func(w *core.Frame) {
		w.Styler(func(s *styles.Style) {
			s.Grow.Set(1, 0)
		})
		tree.AddChildAt(w, "status-text", func(w *core.Text) {})
	})

	// tv.TreeSig.Connect(vv.This, func(recv, send tree.Node, sig int64, data any) {
	// 	gvv := recv.Embed(KiT_Vector).(*Vector)
	// 	if data == nil {
	// 		return
	// 	}
	// 	if sig == int64(core.TreeInserted) {
	// 		sn, ok := data.(svg.Node)
	// 		if ok {
	// 			gvv.SVG().NodeEnsureUniqueID(sn)
	// 			svg.CloneNodeGradientProp(sn, "fill")
	// 			svg.CloneNodeGradientProp(sn, "stroke")
	// 		}
	// 		return
	// 	}
	// 	if sig == int64(core.TreeDeleted) {
	// 		sn, ok := data.(svg.Node)
	// 		if ok {
	// 			svg.DeleteNodeGradientProp(sn, "fill")
	// 			svg.DeleteNodeGradientProp(sn, "stroke")
	// 		}
	// 		return
	// 	}
	// 	if sig != int64(core.TreeOpened) {
	// 		return
	// 	}
	// 	tvn, _ := data.(tree.Node).Embed(KiT_Tree).(*Tree)
	// 	_, issvg := tvn.SrcNode.(svg.Node)
	// 	if !issvg {
	// 		return
	// 	}
	// 	if tvn.SrcNode.HasChildren() {
	// 		return
	// 	}
	// 	core.FormDialog(gvv.Viewport, tvn.SrcNode, core.DlgOpts{Title: "SVG Element View"}, nil, nil)
	// 	// ggv, _ := recv.Embed(KiT_Vector).(*Vector)
	// 	// 		stv := ggv.RecycleTab("Obj", core.KiT_Form, true).(*core.Form)
	// 	// 		stv.SetStruct(tvn.SrcNode)
	// })

	// vc.ConfigTools()
	// vc.ConfigTabs()
}

// OpenDrawingFile opens a new .svg drawing file -- just the basic opening
func (vv *Vector) OpenDrawingFile(fnm core.Filename) error {
	path, _ := filepath.Abs(string(fnm))
	vv.Filename = core.Filename(path)
	sv := vv.SVG()
	err := errors.Log(sv.SVG.OpenXML(path))
	// SavedPaths.AddPath(path, core.Settings.Params.SavedPathsMax)
	// SavePaths()
	fdir, _ := filepath.Split(path)
	errors.Log(os.Chdir(fdir))
	vv.EditState.Init(vv)
	vv.UpdateLayerView()

	vv.EditState.Gradients = sv.Gradients()
	sv.SVG.GatherIDs() // also ensures uniqueness, key for json saving
	sv.ZoomToContents(false)
	sv.ReadMetaData()
	sv.SetTransform()
	return err
}

// OpenDrawing opens a new .svg drawing
func (vv *Vector) OpenDrawing(fnm core.Filename) error { //types:add
	err := vv.OpenDrawingFile(fnm)

	sv := vv.SVG()
	vv.SetTitle()
	tv := vv.Tree()
	tv.CloseAll()
	tv.ReSync()
	vv.SetStatus("Opened: " + string(vv.Filename))
	tv.CloseAll()
	sv.bgGridEff = 0
	sv.UpdateView(true)
	vv.NeedsRender()
	return err
}

// NewDrawing creates a new drawing of the given size
func (vv *Vector) NewDrawing(sz PhysSize) *Vector {
	ngr := NewDrawing(sz)
	return ngr
}

// PromptPhysSize prompts for the physical size of the drawing and sets it
func (vv *Vector) PromptPhysSize() { //types:add
	sv := vv.SVG()
	sz := &PhysSize{}
	sz.SetFromSVG(sv)
	d := core.NewBody().AddTitle("SVG physical size")
	core.NewForm(d).SetStruct(sz)
	d.AddBottomBar(func(parent core.Widget) {
		d.AddCancel(parent)
		d.AddOK(parent).OnClick(func(e events.Event) {
			vv.SetPhysSize(sz)
			sv.bgGridEff = -1
			sv.UpdateView(true)
		})
	})
	d.RunDialog(vv)
}

// SetPhysSize sets physical size of drawing
func (vv *Vector) SetPhysSize(sz *PhysSize) {
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
func (vv *Vector) SaveDrawing() error { //types:add
	if vv.Filename != "" {
		return vv.SaveDrawingAs(vv.Filename)
	}
	core.CallFunc(vv, vv.SaveDrawingAs)
	return nil
}

// SaveDrawingAs saves .svg drawing to given filename
func (vv *Vector) SaveDrawingAs(fname core.Filename) error { //types:add
	if fname == "" {
		return errors.New("SaveDrawingAs: filename is empty")
	}
	path, _ := filepath.Abs(string(fname))
	vv.Filename = core.Filename(path)
	// SavedPaths.AddPath(path, core.Settings.Params.SavedPathsMax)
	// SavePaths()
	sv := vv.SVG()
	sv.SVG.RemoveOrphanedDefs()
	sv.SetMetaData()
	err := sv.SVG.SaveXML(path)
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
func (vv *Vector) ExportPNG(width, height float32) error { //types:add
	path, _ := filepath.Split(string(vv.Filename))
	fnm := filepath.Join(path, "export_png.svg")
	sv := vv.SVG()
	err := sv.SVG.SaveXML(fnm)
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
func (vv *Vector) ExportPDF(dpi float32) error { //types:add
	path, _ := filepath.Split(string(vv.Filename))
	fnm := filepath.Join(path, "export_pdf.svg")
	sv := vv.SVG()
	err := sv.SVG.SaveXML(fnm)
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
func (vv *Vector) ResizeToContents() { //types:add
	sv := vv.SVG()
	sv.ResizeToContents(true)
	sv.UpdateView(true)
}

// AddImage adds a new image node set to the given image
func (vv *Vector) AddImage(fname core.Filename, width, height float32) error { //types:add
	sv := vv.SVG()
	sv.UndoSave("AddImage", string(fname))
	ind := sv.NewElement(svg.ImageType).(*svg.Image)
	ind.Pos.X = 100 // todo: default pos
	ind.Pos.Y = 100 // todo: default pos
	err := ind.OpenImage(string(fname), width, height)
	sv.UpdateView(true)
	vv.ChangeMade()
	return err
}

func (vv *Vector) ModalToolbar() *core.Toolbar {
	return vv.ChildByName("modal-tb", 1).(*core.Toolbar)
}

func (vv *Vector) HBox() *core.Frame {
	return vv.ChildByName("hbox", 2).(*core.Frame)
}

func (vv *Vector) Tools() *core.Toolbar {
	return vv.HBox().ChildByName("tools", 0).(*core.Toolbar)
}

func (vv *Vector) Splits() *core.Splits {
	return vv.HBox().ChildByName("splits", 1).(*core.Splits)
}

func (vv *Vector) LayerTree() *core.Frame {
	return vv.Splits().ChildByName("layer-tree", 0).(*core.Frame)
}

func (vv *Vector) LayerView() *core.Table {
	return vv.LayerTree().ChildByName("layers", 0).(*core.Table)
}

func (vv *Vector) Tree() *Tree {
	return vv.LayerTree().ChildByName("tree-frame", 1).AsTree().Child(0).(*Tree)
}

// SVG returns the [SVG].
func (vv *Vector) SVG() *SVG {
	return vv.Splits().Child(1).(*SVG)
}

// SSVG returns the underlying [svg.SVG].
func (vv *Vector) SSVG() *svg.SVG {
	return vv.SVG().SVG
}

func (vv *Vector) Tabs() *core.Tabs {
	return vv.Splits().ChildByName("tabs", 2).(*core.Tabs)
}

// StatusBar returns the status bar widget
func (vv *Vector) StatusBar() *core.Frame {
	return vv.ChildByName("status-bar", 4).(*core.Frame)
}

// StatusText returns the status bar text widget
func (vv *Vector) StatusText() *core.Text {
	return vv.StatusBar().Child(0).(*core.Text)
}

// PasteAvailFunc is an ActionUpdateFunc that inactivates action if no paste avail
func (vv *Vector) PasteAvailFunc(bt *core.Button) {
	bt.SetEnabled(!vv.Clipboard().IsEmpty())
}

func (vv *Vector) MakeToolbar(p *tree.Plan) {
	tree.Add(p, func(w *core.FuncButton) {
		// TODO(kai): remove Update
		w.SetFunc(vv.UpdateAll).SetText("Update").SetIcon(icons.Update)
	})
	tree.Add(p, func(w *core.Button) {
		w.SetText("New").SetIcon(icons.Add).
			OnClick(func(e events.Event) {
				ndr := vv.NewDrawing(Settings.Size)
				ndr.PromptPhysSize()
			})
	})
	tree.Add(p, func(w *core.Button) {
		w.SetText("Size").SetIcon(icons.FormatSize).SetMenu(func(m *core.Scene) {
			core.NewFuncButton(m).SetFunc(vv.PromptPhysSize).SetText("Set size").SetIcon(icons.FormatSize)
			core.NewFuncButton(m).SetFunc(vv.ResizeToContents).SetIcon(icons.Resize)
		})
	})

	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(vv.OpenDrawing).SetText("Open").SetIcon(icons.Open)
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(vv.SaveDrawing).SetText("Save").SetIcon(icons.Save)
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(vv.SaveDrawingAs).SetText("Save as").SetIcon(icons.SaveAs)
	})

	tree.Add(p, func(w *core.Button) {
		w.SetText("Export").SetIcon(icons.ExportNotes).SetMenu(func(m *core.Scene) {
			core.NewFuncButton(m).SetFunc(vv.ExportPNG).SetIcon(icons.Image)
			core.NewFuncButton(m).SetFunc(vv.ExportPDF).SetIcon(icons.PictureAsPdf)
		})
	})

	tree.Add(p, func(w *core.Separator) {})

	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(vv.Undo).FirstStyler(func(s *styles.Style) {
			s.SetEnabled(vv.EditState.Undos.HasUndoAvail())
		})
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(vv.Redo).FirstStyler(func(s *styles.Style) {
			s.SetEnabled(vv.EditState.Undos.HasRedoAvail())
		})
	})

	tree.Add(p, func(w *core.Separator) {})

	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(vv.DuplicateSelected).SetText("Duplicate").SetIcon(icons.Copy).SetKey(keymap.Duplicate)
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(vv.CopySelected).SetText("Copy").SetIcon(icons.Copy).SetKey(keymap.Copy)
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(vv.CutSelected).SetText("Cut").SetIcon(icons.Cut).SetKey(keymap.Cut)
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(vv.PasteClip).SetText("Paste").SetIcon(icons.Paste).SetKey(keymap.Paste)
	})

	tree.Add(p, func(w *core.Separator) {})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(vv.AddImage).SetIcon(icons.Image)
	})
	tree.Add(p, func(w *core.Separator) {})

	tree.Add(p, func(w *core.Button) {
		w.SetText("Zoom page").SetIcon(icons.ZoomOut)
		w.SetTooltip("Zoom to see the entire page size for drawing")
		w.OnClick(func(e events.Event) {
			sv := vv.SVG()
			sv.ZoomToPage(false)
			sv.UpdateView(true)
		})
	})
	tree.Add(p, func(w *core.Button) {
		w.SetText("Zoom all").SetIcon(icons.ZoomOut)
		w.SetTooltip("Zoom to see all elements")
		w.OnClick(func(e events.Event) {
			sv := vv.SVG()
			sv.ZoomToContents(false)
		})
	})
}

// SetStatus updates the status bar text with the given message, along with other status info
func (vv *Vector) SetStatus(msg string) {
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
	text.UpdateWidget()
	text.NeedsRender()
}

func (vv *Vector) SetTitle() {
	if vv.Filename == "" {
		return
	}
	win := vv.Scene.RenderWindow()
	if win == nil {
		return
	}
	dfnm := fsx.DirAndFile(string(vv.Filename))
	winm := "Cogent Vector • " + dfnm
	win.SetName(winm)
	win.SetTitle(winm)
	vv.Scene.Body.Title = winm
}

// NewDrawing opens a new drawing window
func NewDrawing(sz PhysSize) *Vector {
	ngr := NewVectorWindow("")
	ngr.SetPhysSize(&sz)
	return ngr
}

// NewVectorWindow returns a new VectorWindow loading given file if non-empty
func NewVectorWindow(fnm string) *Vector {
	path := ""
	dfnm := "blank"
	if fnm != "" {
		path, _ = filepath.Abs(fnm)
		dfnm = fsx.DirAndFile(path)
	}
	winm := "Cogent Vector • " + dfnm

	if win, found := core.AllRenderWindows.FindName(winm); found {
		sc := win.MainScene()
		if vv := tree.ChildByType[*Vector](sc.Body); vv != nil {
			if string(vv.Filename) == path {
				win.Raise()
				return vv
			}
		}
	}

	b := core.NewBody(winm).SetTitle(winm)

	vc := NewVector(b)
	b.AddAppBar(vc.MakeToolbar)

	b.OnShow(func(e events.Event) {
		if fnm != "" {
			vc.OpenDrawingFile(core.Filename(path))
		} else {
			vc.EditState.Init(vc)
		}
	})

	b.RunWindow()

	return vc
}

/////////////////////////////////////////////////////////////////////////
//   Controls

// Tab returns the tab with the given name
func (gv *Vector) Tab(name string) *core.Frame {
	return gv.Tabs().TabByName(name)
}

func (vv *Vector) PaintView() *PaintView {
	return vv.Tab("Paint").Child(0).(*PaintView)
}

// UpdateAll updates the display
func (vv *Vector) UpdateAll() { //types:add
	vv.UpdateTabs()
	vv.UpdateTree()
	vv.UpdateDisp()
}

func (vv *Vector) UpdateDisp() {
	sv := vv.SVG()
	sv.UpdateView(true)
}

func (vv *Vector) UpdateTree() {
	tv := vv.Tree()
	tv.ReSync()
}

func (vv *Vector) SetDefaultStyle() {
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

func (vv *Vector) UpdateTabs() {
	// es := &vv.EditState
	// fsel := es.FirstSelectedNode()
	// if fsel != nil {
	// 	sel := fsel.AsNodeBase()
	// 	pv := vv.PaintView()
	// 	pv.Update(&sel.Paint, sel.This)
	// 	txt, istxt := fsel.(*svg.Text)
	// 	if istxt {
	// 		es.Text.SetFromNode(txt)
	// 		txv := vv.Tab("Text").(*core.Form)
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
func (vv *Vector) SelectNodeInSVG(kn tree.Node, mode events.SelectModes) {
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
func (vv *Vector) Undo() string { //types:add
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
func (vv *Vector) Redo() string { //types:add
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
func (vv *Vector) ChangeMade() {
	go vv.AutoSave()
}

/////////////////////////////////////////////////////////////////////////
//   Basic infrastructure

/*
func (gv *Vector) OSFileEvent() {
	gv.ConnectEvent(oswin.OSOpenFilesEvent, core.RegPri, func(recv, send tree.Node, sig int64, d any) {
		ofe := d.(*osevent.OpenFilesEvent)
		for _, fn := range ofe.Files {
			NewVectorWindow(fn)
		}
	})
}
*/

// OpenRecent opens a recently used file
func (vv *Vector) OpenRecent(filename core.Filename) {
	// if string(filename) == VectorResetRecents {
	// 	SavedPaths = nil
	// 	core.StringsAddExtras((*[]string)(&SavedPaths), SavedPathsExtras)
	// } else if string(filename) == VectorEditRecents {
	// 	vv.EditRecents()
	// } else {
	// 	vv.OpenDrawing(filename)
	// }
}

// RecentsEdit opens a dialog editor for deleting from the recents project list
func (vv *Vector) EditRecents() {
	// tmp := make([]string, len(SavedPaths))
	// copy(tmp, SavedPaths)
	// core.StringsRemoveExtras((*[]string)(&tmp), SavedPathsExtras)
	// opts := core.DlgOpts{Title: "Recent Project Paths", Prompt: "Delete paths you no longer use", Ok: true, Cancel: true, NoAdd: true}
	// core.ListDialog(vv.Viewport, &tmp, opts,
	// 	nil, vv, func(recv, send tree.Node, sig int64, data any) {
	// 		if sig == int64(core.DialogAccepted) {
	// 			SavedPaths = nil
	// 			SavedPaths = append(SavedPaths, tmp...)
	// 			core.StringsAddExtras((*[]string)(&SavedPaths), SavedPathsExtras)
	// 		}
	// 	})
}

// SplitsSetView sets split view splitters to given named setting
func (vv *Vector) SplitsSetView(split SplitName) {
	sv := vv.Splits()
	sp, _, ok := AvailableSplits.SplitByName(split)
	if ok {
		sv.SetSplits(sp.Splits...).NeedsLayout()
		Settings.SplitName = split
	}
}

// SplitsSave saves current splitter settings to named splitter settings under
// existing name, and saves to prefs file
func (vv *Vector) SplitsSave(split SplitName) {
	sv := vv.Splits()
	sp, _, ok := AvailableSplits.SplitByName(split)
	if ok {
		sp.SaveSplits(sv.Splits)
		AvailableSplits.SaveSettings()
	}
}

// SplitsSaveAs saves current splitter settings to new named splitter settings, and
// saves to prefs file
func (vv *Vector) SplitsSaveAs(name, desc string) {
	spv := vv.Splits()
	AvailableSplits.Add(name, desc, spv.Splits)
	AvailableSplits.SaveSettings()
}

// SplitsEdit opens the SplitsView editor to customize saved splitter settings
func (vv *Vector) SplitsEdit() {
	SplitsView(&AvailableSplits)
}

// HelpWiki opens wiki page for grid on github
func (vv *Vector) HelpWiki() {
	core.TheApp.OpenURL("https://vector.cogentcore.org")
}

////////////////////////////////////////////////////////////////////////////////////////
//		AutoSave

// AutoSaveFilename returns the autosave filename
func (vv *Vector) AutoSaveFilename() string {
	path, fn := filepath.Split(string(vv.Filename))
	if fn == "" {
		fn = "new_file_" + vv.Name + ".svg"
	}
	asfn := filepath.Join(path, "#"+fn+"#")
	return asfn
}

// AutoSave does the autosave -- safe to call in a separate goroutine
func (vv *Vector) AutoSave() error {
	// if vv.HasFlag(int(VectorAutoSaving)) {
	// 	return nil
	// }
	// vv.SetFlag(int(VectorAutoSaving))
	// asfn := vv.AutoSaveFilename()
	// sv := vv.SVG()
	// err := sv.SaveXML(core.Filename(asfn))
	// if err != nil && err != io.EOF {
	// 	log.Println(err)
	// }
	// vv.ClearFlag(int(VectorAutoSaving))
	// return err
	return nil
}

// AutoSaveDelete deletes any existing autosave file
func (vv *Vector) AutoSaveDelete() {
	asfn := vv.AutoSaveFilename()
	os.Remove(asfn)
}

// AutoSaveCheck checks if an autosave file exists -- logic for dealing with
// it is left to larger app -- call this before opening a file
func (vv *Vector) AutoSaveCheck() bool {
	asfn := vv.AutoSaveFilename()
	if _, err := os.Stat(asfn); os.IsNotExist(err) {
		return false // does not exist
	}
	return true
}
