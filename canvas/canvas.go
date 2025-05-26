// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package canvas implements a 2D vector graphics editor.
package canvas

//go:generate core generate

import (
	"fmt"
	"image"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/svg"
	"cogentcore.org/core/system"
	"cogentcore.org/core/tree"
)

// Canvas is the main widget of the Cogent Canvas SVG vector graphics program.
type Canvas struct {
	core.Frame

	// full path to current drawing filename
	Filename core.Filename `extension:".svg" set:"-"`

	// current edit state
	EditState EditState `set:"-"`

	SVG        *SVG
	tabs       *core.Tabs
	splits     *core.Splits
	modalTools *core.Toolbar
	tools      *core.Toolbar
	tree       *Tree
	defs       *Tree
	layerTree  *core.Frame
	layers     *core.Table
	statusBar  *core.Frame
}

func (cv *Canvas) Init() {
	cv.Frame.Init()
	// cv.EditState.ConfigDefaultGradient()
	cv.Styler(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Droppable) // external drop
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
	})
	cv.On(events.Drop, func(e events.Event) {
		de := e.(*events.DragDrop)
		md := de.Data.(mimedata.Mimes)
		for _, d := range md {
			if d.Type != fileinfo.TextPlain {
				continue
			}
			path := string(d.Data)
			NewWindow(path)
		}
	})
	cv.AddCloseDialog(func(d *core.Body) bool {
		return false // todo: temporary disable -- need to fix bug
		if !cv.EditState.Changed {
			return false
		}
		d.SetTitle("Unsaved changes")
		core.NewText(d).SetType(core.TextSupporting).SetText(fmt.Sprintf("There are unsaved changes in %s", fsx.DirAndFile(string(cv.Filename))))
		d.AddBottomBar(func(bar *core.Frame) {
			d.AddOK(bar).SetText("Close without saving").OnClick(func(e events.Event) {
				cv.Scene.Close()
			})
			d.AddOK(bar).SetText("Save and close").OnClick(func(e events.Event) {
				cv.SaveDrawing()
				cv.Scene.Close()
			})
		})
		return true
	})

	tree.AddChildAt(cv, "modal-tb", func(w *core.Toolbar) {
		cv.modalTools = w
		w.Styler(func(s *styles.Style) {
			s.Min.Y.Em(2) // keep a consistent height
		})
		tool := cv.EditState.Tool
		w.Maker(func(p *tree.Plan) {
			switch {
			case tool == TextTool || cv.EditState.SelectIsText:
				cv.MakeTextToolbar(p)
			case tool == NodeTool:
				cv.MakeNodeToolbar(p)
			default:
				cv.MakeSelectToolbar(p)
			}
		})
	})

	tree.AddChildAt(cv, "hbox", func(w *core.Frame) {
		w.Styler(func(s *styles.Style) {
			s.Grow.Set(1, 1)
		})
		tree.AddChildAt(w, "tools", func(w *core.Toolbar) {
			cv.tools = w
			w.Styler(func(s *styles.Style) {
				s.Direction = styles.Column
			})
			w.Maker(cv.MakeTools)
		})
		tree.AddChildAt(w, "splits", func(w *core.Splits) {
			w.SetSplits(0.15, 0.60, 0.25)
			tree.AddChildAt(w, "layer-tree", func(w *core.Frame) {
				cv.layerTree = w
				w.Styler(func(s *styles.Style) {
					s.Direction = styles.Column
				})
				tree.AddChild(w, func(w *core.FuncButton) {
					w.SetFunc(cv.AddLayer)
				})
				tree.AddChildAt(w, "layers", func(w *core.Table) {
					w.ShowIndexes = true
					cv.layers = w
					w.Styler(func(s *styles.Style) {
						s.Max.Y.Em(10)
					})
					w.SetSlice(&cv.EditState.Layers)
					w.OnSelect(func(e events.Event) {
						cv.EditState.CurLayer = cv.EditState.Layers[w.SelectedIndex].Name
						cv.tree.Resync()
					})
					w.OnChange(func(e events.Event) {
						cv.SyncLayersToSVG()
						cv.UpdateTree()
					})
				})
				tree.AddChildAt(w, "tree-defs", func(w *core.Frame) {
					w.Styler(func(s *styles.Style) {
						s.Direction = styles.Column
						s.Grow.Set(0, 1)
					})
					tree.AddChildAt(w, "tree", func(w *Tree) {
						cv.defs = w
						w.Canvas = cv
						w.OpenDepth = 4
						w.SyncTree(cv.SVG.SVG.Defs)
					})
				})
				tree.AddChildAt(w, "tree-frame", func(w *core.Frame) {
					w.Styler(func(s *styles.Style) {
						s.Direction = styles.Column
						s.Grow.Set(0, 1)
					})
					tree.AddChildAt(w, "tree", func(w *Tree) {
						cv.tree = w
						w.Canvas = cv
						w.OpenDepth = 4
						w.SyncTree(cv.SVG.Root())
					})
				})
			})
			tree.AddChildAt(w, "svg", func(w *SVG) {
				cv.SVG = w
				w.Canvas = cv
				w.UpdateGradients(cv.EditState.Gradients)
				cv.SetPhysSize(&Settings.Size)
				cv.SyncLayersFromSVG()
			})
			tree.AddChildAt(w, "tabs", func(w *core.Tabs) {
				cv.tabs = w
				w.SetType(core.FunctionalTabs)
				pt, _ := w.NewTab("Paint")
				NewPaintSetter(pt).SetCanvas(cv)
				at, _ := w.NewTab("Align")
				NewAlignView(at).SetCanvas(cv)
				tt, _ := w.NewTab("Text")
				core.NewForm(tt).SetStruct(&cv.EditState.Text).OnChange(func(e events.Event) {
					cv.EditState.Text.Update()
				})
			})
		})
	})
	tree.AddChildAt(cv, "status-bar", func(w *core.Frame) {
		cv.statusBar = w
		w.Styler(func(s *styles.Style) {
			s.Grow.Set(1, 0)
		})
		tree.AddChildAt(w, "status-text", func(w *core.Text) {})
	})
}

// OpenDrawingFile opens a new .svg drawing file -- just the basic opening
func (cv *Canvas) OpenDrawingFile(fnm core.Filename) error {
	path, _ := filepath.Abs(string(fnm))
	cv.Filename = core.Filename(path)
	sv := cv.SVG
	err := errors.Log(sv.SVG.OpenXML(path))
	sv.SVG.GradientFromGradients()
	// SavedPaths.AddPath(path, core.Settings.Params.SavedPathsMax)
	// SavePaths()
	fdir, _ := filepath.Split(path)
	errors.Log(os.Chdir(fdir))
	cv.EditState.Init(cv)
	cv.EditState.Gradients = sv.Gradients()
	sv.SVG.GatherIDs() // also ensures uniqueness, key for json saving
	sv.ReadMetaData()
	sv.SVG.ZoomReset() // todo: not working
	cv.UpdateAll()
	return err
}

// OpenDrawing opens a new .svg drawing
func (cv *Canvas) OpenDrawing(fnm core.Filename) error { //types:add
	err := cv.OpenDrawingFile(fnm)

	sv := cv.SVG
	cv.SetTitle()
	tv := cv.tree
	tv.CloseAll()
	tv.Resync()
	cv.SetStatus("Opened: " + string(cv.Filename))
	tv.CloseAll()
	sv.backgroundGridEff = 0
	// cv.UpdateAll()
	return err
}

// NewDrawing creates a new drawing of the given size
func (cv *Canvas) NewDrawing(sz PhysSize) *Canvas {
	ngr := NewDrawing(sz)
	return ngr
}

// PromptPhysSize prompts for the physical size of the drawing and sets it
func (cv *Canvas) PromptPhysSize() { //types:add
	sv := cv.SVG
	sz := &PhysSize{}
	sz.SetFromSVG(sv)
	d := core.NewBody("SVG physical size")
	core.NewForm(d).SetStruct(sz)
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).OnClick(func(e events.Event) {
			cv.SetPhysSize(sz)
			sv.backgroundGridEff = -1
			sv.UpdateView()
		})
	})
	d.RunDialog(cv)
}

// SetPhysSize sets physical size of drawing
func (cv *Canvas) SetPhysSize(sz *PhysSize) {
	if sz == nil {
		return
	}
	if sz.Size == (math32.Vector2{}) {
		sz.SetStandardSize(Settings.Size.StandardSize)
	}
	sv := cv.SVG
	sz.SetToSVG(sv)
	sv.SetMetaData()
	sv.SVG.ZoomReset()
}

// SaveDrawing saves .svg drawing to current filename
func (cv *Canvas) SaveDrawing() error { //types:add
	if cv.Filename != "" {
		return cv.SaveDrawingAs(cv.Filename)
	}
	core.CallFunc(cv, cv.SaveDrawingAs)
	return nil
}

// SaveDrawingAs saves .svg drawing to given filename
func (cv *Canvas) SaveDrawingAs(fname core.Filename) error { //types:add
	if fname == "" {
		return errors.New("SaveDrawingAs: filename is empty")
	}
	path, _ := filepath.Abs(string(fname))
	cv.Filename = core.Filename(path)
	// SavedPaths.AddPath(path, core.Settings.Params.SavedPathsMax)
	// SavePaths()
	sv := cv.SVG
	sv.SVG.RemoveOrphanedDefs()
	sv.SetMetaData()
	err := sv.SVG.SaveXML(path)
	if errors.Log(err) == nil {
		cv.AutoSaveDelete()
	}
	cv.SetTitle()
	cv.SetStatus("Saved: " + path)
	cv.EditState.Changed = false
	cv.UpdateAll()
	return err
}

// ExportPNG exports drawing to a PNG image (auto-names to same name
// with .png suffix).
// Specify either width or height of resulting image, or nothing for
// physical size as set.  Renders full current page -- do ResizeToContents
// to render just current contents.
func (cv *Canvas) ExportPNG(width, height float32) error { //types:add
	fext := filepath.Ext(string(cv.Filename))
	onm := strings.TrimSuffix(string(cv.Filename), fext) + ".png"
	err := cv.SSVG().SaveImage(onm)
	return err
}

// ExportPDF exports drawing to a PDF file (auto-names to same name
// with .pdf suffix).  Calls inkscape -- needs to be on the PATH.
// specify DPI of resulting image for effects rendering.
// Renders full current page -- do ResizeToContents
// to render just current contents.
func (cv *Canvas) ExportPDF(dpi float32) error { //types:add
	path, _ := filepath.Split(string(cv.Filename))
	fnm := filepath.Join(path, "export_pdf.svg")
	sv := cv.SVG
	err := sv.SVG.SaveXML(fnm)
	if errors.Log(err) != nil {
		return err
	}
	fext := filepath.Ext(string(cv.Filename))
	onm := strings.TrimSuffix(string(cv.Filename), fext) + ".pdf"
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
func (cv *Canvas) ResizeToContents() { //types:add
	sv := cv.SVG
	sv.ResizeToContents(true)
	sv.UpdateView()
}

// AddImage adds a new image node set to the given image
func (cv *Canvas) AddImage(fname core.Filename, width, height float32) error { //types:add
	sv := cv.SVG
	sv.UndoSave("AddImage", string(fname))
	ind := NewSVGElement[svg.Image](sv, false)
	ind.Pos.X = 100 // todo: default pos
	ind.Pos.Y = 100 // todo: default pos
	err := ind.OpenImage(string(fname), width, height)
	cv.ChangeMade()
	sv.UpdateView()
	return err
}

// SSVG returns the underlying [svg.SVG].
func (cv *Canvas) SSVG() *svg.SVG {
	return cv.SVG.SVG
}

// StatusText returns the status bar text widget
func (cv *Canvas) StatusText() *core.Text {
	return cv.statusBar.Child(0).(*core.Text)
}

// PasteAvailFunc is an ActionUpdateFunc that inactivates action if no paste avail
func (cv *Canvas) PasteAvailFunc(bt *core.Button) {
	bt.SetEnabled(!cv.Clipboard().IsEmpty())
}

func (cv *Canvas) MakeToolbar(p *tree.Plan) {
	tree.Add(p, func(w *core.FuncButton) {
		// TODO(kai): remove Update
		w.SetFunc(cv.UpdateAll).SetText("Update").SetIcon(icons.Update)
	})
	tree.Add(p, func(w *core.Button) {
		w.SetText("New").SetIcon(icons.Add).
			OnClick(func(e events.Event) {
				ndr := cv.NewDrawing(Settings.Size)
				ndr.PromptPhysSize()
			})
	})
	tree.Add(p, func(w *core.Button) {
		w.SetText("Size").SetIcon(icons.FormatSize).SetMenu(func(m *core.Scene) {
			core.NewFuncButton(m).SetFunc(cv.PromptPhysSize).SetText("Set size").SetIcon(icons.FormatSize)
			core.NewFuncButton(m).SetFunc(cv.ResizeToContents).SetIcon(icons.Resize)
		})
	})

	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(cv.OpenDrawing).SetText("Open").SetIcon(icons.Open)
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(cv.SaveDrawing).SetText("Save").SetIcon(icons.Save)
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(cv.SaveDrawingAs).SetText("Save as").SetIcon(icons.SaveAs)
	})

	tree.Add(p, func(w *core.Button) {
		w.SetText("Export").SetIcon(icons.ExportNotes).SetMenu(func(m *core.Scene) {
			core.NewFuncButton(m).SetFunc(cv.ExportPNG).SetIcon(icons.Image)
			core.NewFuncButton(m).SetFunc(cv.ExportPDF).SetIcon(icons.PictureAsPdf)
		})
	})

	tree.Add(p, func(w *core.Separator) {})

	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(cv.Undo).SetIcon(icons.Undo)
		w.FirstStyler(func(s *styles.Style) {
			s.SetEnabled(cv.EditState.Undos.HasUndoAvail())
		})
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(cv.Redo).SetIcon(icons.Redo)
		w.FirstStyler(func(s *styles.Style) {
			s.SetEnabled(cv.EditState.Undos.HasRedoAvail())
		})
	})

	tree.Add(p, func(w *core.Separator) {})

	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(cv.DuplicateSelected).SetText("Duplicate").SetIcon(icons.Copy).SetKey(keymap.Duplicate)
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(cv.CopySelected).SetText("Copy").SetIcon(icons.Copy).SetKey(keymap.Copy)
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(cv.CutSelected).SetText("Cut").SetIcon(icons.Cut).SetKey(keymap.Cut)
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(cv.PasteClip).SetText("Paste").SetIcon(icons.Paste).SetKey(keymap.Paste)
	})

	tree.Add(p, func(w *core.Separator) {})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(cv.AddImage).SetIcon(icons.Image)
	})
	tree.Add(p, func(w *core.Separator) {})

	tree.Add(p, func(w *core.Button) {
		w.SetText("Zoom page").SetIcon(icons.ZoomOut)
		w.SetTooltip("Zoom to see the entire page size for drawing")
		w.OnClick(func(e events.Event) {
			sv := cv.SVG
			sv.SVG.ZoomReset()
			sv.UpdateView()
		})
	})
	tree.Add(p, func(w *core.Button) {
		w.SetText("Zoom all").SetIcon(icons.ZoomOut)
		w.SetTooltip("Zoom to see all elements")
		w.OnClick(func(e events.Event) {
			sv := cv.SVG
			sv.SVG.ZoomToContents(sv.Geom.Size.Actual.Content)
			sv.UpdateView()
		})
	})

	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(core.SettingsWindow).SetText("Settings").SetIcon(icons.Settings)
		w.SetTooltip("Canvas and system settings")
	})
}

// SetStatus updates the status bar text with the given message, along with other status info
func (cv *Canvas) SetStatus(msg string) {
	sb := cv.statusBar
	if sb == nil {
		return
	}
	text := cv.StatusText()
	es := &cv.EditState
	str := "<b>" + strings.TrimSuffix(es.Tool.String(), "Tool") + "</b>\t"
	if es.CurLayer != "" {
		str += "Layer: " + es.CurLayer + "\t\t"
	}
	str += msg
	text.SetText(str).UpdateRender()
}

func (cv *Canvas) SetTitle() {
	if cv.Filename == "" {
		return
	}
	dfnm := fsx.DirAndFile(string(cv.Filename))
	cv.Scene.Body.SetTitle("Cogent Canvas • " + dfnm)
}

// NewDrawing opens a new drawing window
func NewDrawing(sz PhysSize) *Canvas {
	ngr := NewWindow("")
	ngr.SetPhysSize(&sz)
	return ngr
}

var openFilesDone = false

// NewWindow returns a new [Canvas] in a new window loading given file if non-empty.
func NewWindow(fnm string) *Canvas {
	path := ""
	dfnm := "blank"
	if fnm != "" {
		path, _ = filepath.Abs(fnm)
		dfnm = fsx.DirAndFile(path)
	}
	appnm := "Cogent Canvas • "
	winm := appnm + dfnm

	if w := core.AllRenderWindows.FindName(winm); w != nil {
		sc := w.MainScene()
		if cv := tree.ChildByType[*Canvas](sc.Body); cv != nil {
			if string(cv.Filename) == path {
				w.Raise()
				return cv
			}
		}
	}

	b := core.NewBody(winm).SetTitle(winm)

	cv := NewCanvas(b)
	b.AddTopBar(func(bar *core.Frame) {
		core.NewToolbar(bar).Maker(cv.MakeToolbar)
	})

	b.OnShow(func(e events.Event) {
		if path != "" {
			cv.OpenDrawingFile(core.Filename(path))
		} else {
			ofn := system.TheApp.OpenFiles()
			if !openFilesDone && len(ofn) > 0 {
				openFilesDone = true
				path, _ = filepath.Abs(ofn[0])
				dfnm = fsx.DirAndFile(path)
				winm = appnm + dfnm
				cv.OpenDrawingFile(core.Filename(path))
				b.SetTitle(winm)
			} else {
				cv.EditState.Init(cv)
			}
		}
	})
	b.Scene.On(events.OSOpenFiles, func(e events.Event) {
		of := e.(*events.OSFiles)
		for _, fn := range of.Files {
			NewWindow(fn)
		}
	})

	b.RunWindow()

	return cv
}

////////   Controls

// Tab returns the tab with the given name
func (gv *Canvas) Tab(name string) *core.Frame {
	return gv.tabs.TabByName(name)
}

func (cv *Canvas) PaintSetter() *PaintSetter {
	return cv.Tab("Paint").Child(0).(*PaintSetter)
}

// UpdateAll updates the display
func (cv *Canvas) UpdateAll() { //types:add
	cv.UpdateTabs()
	cv.UpdateLayers()
	cv.UpdateTree()
	cv.UpdateSVG()
}

func (cv *Canvas) UpdateSVG() {
	cv.SVG.UpdateView()
}

func (cv *Canvas) UpdateTree() {
	cv.defs.Resync()
	cv.tree.Resync()
}

func (cv *Canvas) SetDefaultStyle() {
	// pv := vv.PaintSetter()
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

// UpdateSelectIsText updates the SelectIsText status
func (cv *Canvas) UpdateModalToolbar() {
	cv.EditState.UpdateSelectIsText()
	cv.modalTools.Update()
	// tb := vc.SelectToolbar()
	// tb.NeedsRender()
	// tb.Update()
	// sz := es.DragSelEffBBox.Size()
	// tb.ChildByName("posx", 8).(*core.Spinner).SetValue(es.DragSelEffBBox.Min.X)
	// tb.ChildByName("posy", 9).(*core.Spinner).SetValue(es.DragSelEffBBox.Min.Y)
	// tb.ChildByName("width", 10).(*core.Spinner).SetValue(sz.X)
	// tb.ChildByName("height", 11).(*core.Spinner).SetValue(sz.Y)
}

func (cv *Canvas) UpdateText() {
	cv.Tab("Text").Update()
}

func (cv *Canvas) UpdateTabs() {
	cv.UpdateModalToolbar() // updates SelectIsText
	es := &cv.EditState
	fsel := es.FirstSelectedNode()
	if es.SelectIsText {
		es.Text.SetFromNode(fsel.(*svg.Text))
		return
	}
	if fsel == nil {
		return
	}
	_, idx := cv.tabs.CurrentTab()
	if idx == 2 { // if looking at text, no text selected, go back to paint
		cv.tabs.SelectTabIndex(0)
	}
	sel := fsel.AsNodeBase()
	pv := cv.PaintSetter()
	pv.UpdateFromNode(&sel.Paint, sel)
}

// SelectNodeInSVG selects given svg node in SVG drawing
func (cv *Canvas) SelectNodeInSVG(kn tree.Node, mode events.SelectModes) {
	sii, ok := kn.(svg.Node)
	if !ok {
		return
	}
	sv := cv.SVG
	es := &cv.EditState
	es.SelectAction(sii, mode, image.Point{})
	sv.UpdateView()
}

// Undo undoes the last action
func (cv *Canvas) Undo() string { //types:add
	sv := cv.SVG
	act := sv.Undo()
	if act != "" {
		cv.SetStatus("Undid: " + act)
	} else {
		cv.SetStatus("Undo: no more to undo")
	}
	cv.UpdateAll()
	return act
}

// Redo redoes the previously undone action
func (cv *Canvas) Redo() string { //types:add
	sv := cv.SVG
	act := sv.Redo()
	if act != "" {
		cv.SetStatus("Redid: " + act)
	} else {
		cv.SetStatus("Redo: no more to redo")
	}
	cv.UpdateAll()
	return act
}

// ChangeMade should be called after any change is completed on the drawing.
// Calls autosave.
func (cv *Canvas) ChangeMade() {
	go cv.AutoSave()
}

////////   Basic infrastructure

/*
func (gv *Canvas) OSFileEvent() {
	gv.ConnectEvent(oswin.OSOpenFilesEvent, core.RegPri, func(recv, send tree.Node, sig int64, d any) {
		ofe := d.(*osevent.OpenFilesEvent)
		for _, fn := range ofe.Files {
			NewCanvas(fn)
		}
	})
}
*/

// OpenRecent opens a recently used file
func (cv *Canvas) OpenRecent(filename core.Filename) {
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
func (cv *Canvas) EditRecents() {
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
func (cv *Canvas) SplitsSetView(split SplitName) {
	sv := cv.splits
	sp, _, ok := AvailableSplits.SplitByName(split)
	if ok {
		sv.SetSplits(sp.Splits...).NeedsLayout()
		Settings.SplitName = split
	}
}

// SplitsSave saves current splitter settings to named splitter settings under
// existing name, and saves to prefs file
func (cv *Canvas) SplitsSave(split SplitName) {
	sv := cv.splits
	sp, _, ok := AvailableSplits.SplitByName(split)
	if ok {
		sp.SaveSplits(sv.Splits())
		AvailableSplits.SaveSettings()
	}
}

// SplitsSaveAs saves current splitter settings to new named splitter settings, and
// saves to prefs file
func (cv *Canvas) SplitsSaveAs(name, desc string) {
	spv := cv.splits
	AvailableSplits.Add(name, desc, spv.Splits())
	AvailableSplits.SaveSettings()
}

// SplitsEdit opens the SplitsView editor to customize saved splitter settings
func (cv *Canvas) SplitsEdit() {
	SplitsView(&AvailableSplits)
}

// HelpWiki opens wiki page for grid on github
func (cv *Canvas) HelpWiki() {
	core.TheApp.OpenURL("https://vector.cogentcore.org")
}

////////  AutoSave

// AutoSaveFilename returns the autosave filename
func (cv *Canvas) AutoSaveFilename() string {
	path, fn := filepath.Split(string(cv.Filename))
	if fn == "" {
		fn = "new_file_" + cv.Name + ".svg"
	}
	asfn := filepath.Join(path, "#"+fn+"#")
	return asfn
}

// AutoSave does the autosave -- safe to call in a separate goroutine
func (cv *Canvas) AutoSave() error {
	// if vv.HasFlag(int(VectorAutoSaving)) {
	// 	return nil
	// }
	// vv.SetFlag(int(VectorAutoSaving))
	// asfn := vv.AutoSaveFilename()
	// sv := vv.SVG
	// err := sv.SaveXML(core.Filename(asfn))
	// if err != nil && err != io.EOF {
	// 	log.Println(err)
	// }
	// vv.ClearFlag(int(VectorAutoSaving))
	// return err
	return nil
}

// AutoSaveDelete deletes any existing autosave file
func (cv *Canvas) AutoSaveDelete() {
	asfn := cv.AutoSaveFilename()
	os.Remove(asfn)
}

// AutoSaveCheck checks if an autosave file exists -- logic for dealing with
// it is left to larger app -- call this before opening a file
func (cv *Canvas) AutoSaveCheck() bool {
	asfn := cv.AutoSaveFilename()
	if _, err := os.Stat(asfn); os.IsNotExist(err) {
		return false // does not exist
	}
	return true
}
