// Copyright (c) 2021, The Grid Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grid

import (
	"errors"
	"fmt"
	"image"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/gist"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/gi/oswin/osevent"
	"github.com/goki/gi/svg"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
)

// GridView is the Grid SVG vector drawing program: Go-rendered interactive drawing
type GridView struct {
	gi.Frame
	Filename  gi.FileName `ext:".svg" desc:"full path to current drawing filename"`
	EditState EditState   `desc:"current edit state"`
}

var KiT_GridView = kit.Types.AddType(&GridView{}, GridViewProps)

// AddNewGridView adds a new editor to given parent node, with given name.
func AddNewGridView(parent ki.Ki, name string) *GridView {
	gv := parent.AddNewChild(KiT_GridView, name).(*GridView)
	gv.Defaults()
	return gv
}

func (g *GridView) CopyFieldsFrom(frm interface{}) {
	fr := frm.(*GridView)
	g.Frame.CopyFieldsFrom(&fr.Frame)
	// todo: fill out
}

func (gv *GridView) Defaults() {
	es := &gv.EditState
	es.ConfigDefaultGradient()
}

// OpenDrawingFile opens a new .svg drawing file -- just the basic opening
func (gv *GridView) OpenDrawingFile(fnm gi.FileName) error {
	path, _ := filepath.Abs(string(fnm))
	gv.Filename = gi.FileName(path)
	sv := gv.SVG()
	err := sv.OpenXML(gi.FileName(path))
	if err != nil && err != io.EOF {
		log.Println(err)
		// return err
	}
	SavedPaths.AddPath(path, gi.Prefs.Params.SavedPathsMax)
	SavePaths()
	fdir, _ := filepath.Split(path)
	os.Chdir(fdir)
	gv.EditState.Init()
	gv.UpdateLayerView()

	gv.EditState.Gradients = sv.Gradients()
	sv.GatherIds() // also ensures uniqueness, key for json saving
	sv.ZoomToContents(false)
	sv.ReadMetaData()
	sv.SetTransform()
	return err
}

// OpenDrawing opens a new .svg drawing
func (gv *GridView) OpenDrawing(fnm gi.FileName) error {
	wupdt := gv.TopUpdateStart()
	defer gv.TopUpdateEnd(wupdt)
	updt := gv.UpdateStart()
	gv.SetFullReRender()

	gv.Defaults()
	err := gv.OpenDrawingFile(fnm)

	sv := gv.SVG()
	gv.SetTitle()
	tv := gv.TreeView()
	tv.CloseAll()
	tv.ReSync()
	gv.SetStatus("Opened: " + string(gv.Filename))
	gv.UpdateEnd(updt)
	tv.CloseAll()
	sv.bgGridEff = 0
	sv.UpdateView(true)
	return err
}

// NewDrawing opens a new drawing window
func (gv *GridView) NewDrawing(sz PhysSize) *GridView {
	ngr := NewDrawing(sz)
	return ngr
}

// PromptPhysSize prompts for physical size of drawing and sets it
func (gv *GridView) PromptPhysSize() {
	sv := gv.SVG()
	sz := &PhysSize{}
	sz.SetFromSVG(sv)
	giv.StructViewDialog(gv.Viewport, sz, giv.DlgOpts{Title: "SVG Physical Size", Ok: true, Cancel: true}, gv.This(),
		func(recv, send ki.Ki, sig int64, d interface{}) {
			if sig == int64(gi.DialogAccepted) {
				gv.SetPhysSize(sz)
				sv.bgGridEff = -1
				sv.UpdateView(true)
			}
		})
}

// SetPhysSize sets physical size of drawing
func (gv *GridView) SetPhysSize(sz *PhysSize) {
	if sz == nil {
		return
	}
	if sz.Size.IsNil() {
		sz.SetStdSize(Prefs.Size.StdSize)
	}
	sv := gv.SVG()
	sz.SetToSVG(sv)
	sv.SetMetaData()
	sv.ZoomToPage(false)
}

// SaveDrawing saves .svg drawing to current filename
func (gv *GridView) SaveDrawing() error {
	if gv.Filename == "" {
		giv.CallMethod(gv, "SaveDrawingAs", gv.ViewportSafe())
		return nil
	}
	sv := gv.SVG()
	sv.RemoveOrphanedDefs()
	sv.SetMetaData()
	err := sv.SaveXML(gv.Filename)
	if err != nil && err != io.EOF {
		log.Println(err)
	} else {
		gv.AutoSaveDelete()
	}
	gv.SetStatus("Saved: " + string(gv.Filename))
	gv.EditState.Changed = false
	return err
}

// SaveDrawingAs saves .svg drawing to given filename
func (gv *GridView) SaveDrawingAs(fname gi.FileName) error {
	if fname == "" {
		return errors.New("SaveDrawingAs: filename is empty")
	}
	path, _ := filepath.Abs(string(fname))
	gv.Filename = gi.FileName(path)
	SavedPaths.AddPath(path, gi.Prefs.Params.SavedPathsMax)
	SavePaths()
	sv := gv.SVG()
	sv.RemoveOrphanedDefs()
	sv.SetMetaData()
	err := sv.SaveXML(gi.FileName(path))
	if err != nil && err != io.EOF {
		log.Println(err)
	} else {
		gv.AutoSaveDelete()
	}
	gv.SetTitle()
	gv.SetStatus("Saved: " + path)
	gv.EditState.Changed = false
	return err
}

// ExportPNG exports drawing to a PNG image (auto-names to same name
// with .png suffix).  Calls cairosvg (e.g., pip3 install cairosvg).
// specify either width or height of resulting image, or nothing for
// physical size as set.  Renders full current page -- do ResizeToContents
// to render just current contents.
func (gv *GridView) ExportPNG(width, height float32) error {
	path, _ := filepath.Split(string(gv.Filename))
	fnm := filepath.Join(path, "export_png.svg")
	sv := gv.SVG()
	err := sv.SaveXML(gi.FileName(fnm))
	if err != nil && err != io.EOF {
		log.Println(err)
		return err
	}
	fext := filepath.Ext(string(gv.Filename))
	onm := strings.TrimSuffix(string(gv.Filename), fext) + ".png"
	cstr := "cairosvg"
	args := []string{"-o", onm}
	if width > 0 {
		args = append(args, "--output-width")
		args = append(args, fmt.Sprintf("%g", width))
	}
	if height > 0 {
		args = append(args, "--output-height")
		args = append(args, fmt.Sprintf("%g", height))
	}
	args = append(args, fnm)
	cmd := exec.Command(cstr, args...)
	fmt.Printf("executing command: %s %v\n", cstr, args)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(out))
	}
	os.Remove(fnm)
	return err
}

// ExportPDF exports drawing to a PDF file (auto-names to same name
// with .pdf suffix).  Calls cairosvg (e.g., pip3 install cairosvg).
// specify DPI of resulting image for effects rendering.
// Renders full current page -- do ResizeToContents
// to render just current contents.
func (gv *GridView) ExportPDF(dpi float32) error {
	path, _ := filepath.Split(string(gv.Filename))
	fnm := filepath.Join(path, "export_pdf.svg")
	sv := gv.SVG()
	err := sv.SaveXML(gi.FileName(fnm))
	if err != nil && err != io.EOF {
		log.Println(err)
		return err
	}
	fext := filepath.Ext(string(gv.Filename))
	onm := strings.TrimSuffix(string(gv.Filename), fext) + ".pdf"
	cstr := "cairosvg"
	args := []string{"-o", onm}
	if dpi > 0 {
		args = append(args, "--dpi")
		args = append(args, fmt.Sprintf("%g", dpi))
	}
	args = append(args, fnm)
	cmd := exec.Command(cstr, args...)
	fmt.Printf("executing command: %s %v\n", cstr, args)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(out))
	}
	os.Remove(fnm)
	return err
}

// ResizeToContents resizes the drawing to just fit the current contents,
// including moving everything to start at upper-left corner,
// preserving the current grid offset, so grid snapping
// is preserved.
func (gv *GridView) ResizeToContents() {
	sv := gv.SVG()
	sv.ResizeToContents(true)
	sv.UpdateView(true)
}

// AddImage adds a new image node set to given image
func (gv *GridView) AddImage(fname gi.FileName, width, height float32) error {
	sv := gv.SVG()
	sv.UndoSave("AddImage", string(fname))
	ind := sv.NewEl(svg.KiT_Image).(*svg.Image)
	ind.Pos.X = 100 // todo: default pos
	ind.Pos.Y = 100 // todo: default pos
	err := ind.OpenImage(fname, width, height)
	sv.UpdateView(true)
	gv.ChangeMade()
	return err
}

//////////////////////////////////////////////////////////////////////////
//  GUI Config

func (gv *GridView) MainToolbar() *gi.ToolBar {
	return gv.ChildByName("main-tb", 0).(*gi.ToolBar)
}

func (gv *GridView) ModalToolbarStack() *gi.Layout {
	return gv.ChildByName("modal-tb", 1).(*gi.Layout)
}

// SetModalSelect sets the modal toolbar to be the select one
func (gv *GridView) SetModalSelect() {
	tbs := gv.ModalToolbarStack()
	updt := tbs.UpdateStart()
	tbs.SetFullReRender()
	gv.UpdateSelectToolbar()
	idx, _ := tbs.Kids.IndexByName("select-tb", 0)
	tbs.StackTop = idx
	tbs.UpdateEnd(updt)
}

// SetModalNode sets the modal toolbar to be the node editing one
func (gv *GridView) SetModalNode() {
	tbs := gv.ModalToolbarStack()
	updt := tbs.UpdateStart()
	tbs.SetFullReRender()
	gv.UpdateNodeToolbar()
	idx, _ := tbs.Kids.IndexByName("node-tb", 1)
	tbs.StackTop = idx
	tbs.UpdateEnd(updt)
}

// SetModalText sets the modal toolbar to be the text editing one
func (gv *GridView) SetModalText() {
	tbs := gv.ModalToolbarStack()
	updt := tbs.UpdateStart()
	tbs.SetFullReRender()
	gv.UpdateTextToolbar()
	idx, _ := tbs.Kids.IndexByName("text-tb", 2)
	tbs.StackTop = idx
	tbs.UpdateEnd(updt)
}

func (gv *GridView) HBox() *gi.Layout {
	return gv.ChildByName("hbox", 2).(*gi.Layout)
}

func (gv *GridView) Tools() *gi.ToolBar {
	return gv.HBox().ChildByName("tools", 0).(*gi.ToolBar)
}

func (gv *GridView) SplitView() *gi.SplitView {
	return gv.HBox().ChildByName("splitview", 1).(*gi.SplitView)
}

func (gv *GridView) LayerTree() *gi.Layout {
	return gv.SplitView().ChildByName("layer-tree", 0).(*gi.Layout)
}

func (gv *GridView) LayerView() *giv.TableView {
	return gv.LayerTree().ChildByName("layers", 0).(*giv.TableView)
}

func (gv *GridView) TreeView() *TreeView {
	return gv.LayerTree().ChildByName("tree-frame", 1).Child(0).(*TreeView)
}

func (gv *GridView) SVG() *SVGView {
	return gv.SplitView().Child(1).(*SVGView)
}

func (gv *GridView) Tabs() *gi.TabView {
	return gv.SplitView().ChildByName("tabs", 2).(*gi.TabView)
}

// StatusBar returns the statusbar widget
func (gv *GridView) StatusBar() *gi.Frame {
	return gv.ChildByName("statusbar", 4).(*gi.Frame)
}

// StatusLabel returns the statusbar label widget
func (gv *GridView) StatusLabel() *gi.Label {
	return gv.StatusBar().Child(0).Embed(gi.KiT_Label).(*gi.Label)
}

// Config configures entire view -- only runs if no children yet
func (gv *GridView) Config() {
	if gv.HasChildren() {
		return
	}
	updt := gv.UpdateStart()
	gv.Lay = gi.LayoutVert
	// gv.SetProp("spacing", gi.StdDialogVSpaceUnits)
	gi.AddNewToolBar(gv, "main-tb")
	gi.AddNewLayout(gv, "modal-tb", gi.LayoutStacked)
	hb := gi.AddNewLayout(gv, "hbox", gi.LayoutHoriz)
	hb.SetStretchMax()
	gi.AddNewFrame(gv, "statusbar", gi.LayoutHoriz)

	tb := gi.AddNewToolBar(hb, "tools")
	tb.Lay = gi.LayoutVert
	spv := gi.AddNewSplitView(hb, "splitview")
	spv.Dim = mat32.X

	tly := gi.AddNewLayout(spv, "layer-tree", gi.LayoutVert)
	tly.SetStretchMax()

	nly := gi.AddNewButton(tly, "add-layer")
	nly.SetText("Add Layer")
	nly.OnClicked(func() {
		gv.AddLayer()
	})

	lyv := giv.AddNewTableView(tly, "layers")
	lyv.SetMinPrefHeight(units.NewEm(6))
	lyv.SetStretchMax()

	tvfr := gi.AddNewFrame(tly, "tree-frame", gi.LayoutVert)
	tvfr.SetMinPrefHeight(units.NewEm(12))
	tvfr.SetStretchMax()
	tvfr.SetReRenderAnchor()
	tv := AddNewTreeView(tvfr, "treeview")
	tv.GridView = gv
	tv.OpenDepth = 4

	sv := AddNewSVGView(spv, "svg", gv)

	tab := gi.AddNewTabView(spv, "tabs")
	tab.SetStretchMaxWidth()

	tv.SetRootNode(sv)

	tv.TreeViewSig.Connect(gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		gvv := recv.Embed(KiT_GridView).(*GridView)
		if data == nil {
			return
		}
		if sig == int64(giv.TreeViewInserted) {
			sn, ok := data.(svg.NodeSVG)
			if ok {
				gvv.SVG().NodeEnsureUniqueId(sn)
				svg.CloneNodeGradientProp(sn, "fill")
				svg.CloneNodeGradientProp(sn, "stroke")
			}
			return
		}
		if sig == int64(giv.TreeViewDeleted) {
			sn, ok := data.(svg.NodeSVG)
			if ok {
				svg.DeleteNodeGradientProp(sn, "fill")
				svg.DeleteNodeGradientProp(sn, "stroke")
			}
			return
		}
		if sig != int64(giv.TreeViewOpened) {
			return
		}
		tvn, _ := data.(ki.Ki).Embed(KiT_TreeView).(*TreeView)
		_, issvg := tvn.SrcNode.(svg.NodeSVG)
		if !issvg {
			return
		}
		if tvn.SrcNode.HasChildren() {
			return
		}
		giv.StructViewDialog(gvv.Viewport, tvn.SrcNode, giv.DlgOpts{Title: "SVG Element View"}, nil, nil)
		// ggv, _ := recv.Embed(KiT_GridView).(*GridView)
		// 		stv := ggv.RecycleTab("Obj", giv.KiT_StructView, true).(*giv.StructView)
		// 		stv.SetStruct(tvn.SrcNode)
	})

	spv.SetSplits(0.15, 0.60, 0.25)

	gv.ConfigStatusBar()
	gv.ConfigMainToolbar()
	gv.ConfigModalToolbar()
	gv.ConfigTools()
	gv.ConfigTabs()

	gv.SetPhysSize(&Prefs.Size)

	gv.SyncLayers()
	lyv.SetSlice(&gv.EditState.Layers)
	gv.LayerViewSigs(lyv)

	sv.UpdateGradients(gv.EditState.Gradients)

	gv.UpdateEnd(updt)
}

// IsConfiged returns true if the view is fully configured
func (gv *GridView) IsConfiged() bool {
	if !gv.HasChildren() {
		return false
	}
	return true
}

// UndoAvailFunc is an ActionUpdateFunc that inactivates action if no more undos
func (gv *GridView) UndoAvailFunc(act *gi.Action) {
	es := &gv.EditState
	act.SetInactiveState(!es.UndoMgr.HasUndoAvail())
}

// RedoAvailFunc is an ActionUpdateFunc that inactivates action if no more redos
func (gv *GridView) RedoAvailFunc(act *gi.Action) {
	es := &gv.EditState
	act.SetInactiveState(!es.UndoMgr.HasRedoAvail())
}

// PasteAvailFunc is an ActionUpdateFunc that inactivates action if no paste avail
func (gv *GridView) PasteAvailFunc(act *gi.Action) {
	empty := oswin.TheApp.ClipBoard(gv.ParentWindow().OSWin).IsEmpty()
	act.SetInactiveState(empty)
}

func (gv *GridView) ConfigMainToolbar() {
	tb := gv.MainToolbar()
	tb.SetStretchMaxWidth()
	tb.AddAction(gi.ActOpts{Label: "Updt", Icon: "update", Tooltip: "update display -- should not be needed but sometimes, while still under development..."},
		gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			grr.UpdateAll()
		})
	tb.AddAction(gi.ActOpts{Label: "New", Icon: "new", Tooltip: "create new drawing of specified size"},
		gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			ndr := grr.NewDrawing(Prefs.Size)
			ndr.PromptPhysSize()
		})
	szmen := tb.AddAction(gi.ActOpts{Label: "Size", Icon: "gear"}, nil, nil)
	szmen.Menu.AddAction(gi.ActOpts{Label: "Set Size...", Icon: "gear", Tooltip: "set size and grid spacing of drawing"},
		gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			grr.PromptPhysSize()
		})
	szmen.Menu.AddAction(gi.ActOpts{Label: "Resize To Contents", Icon: "gear", Tooltip: "resizes the drawing to fit the current contents, moving everything to upper-left corner while preserving grid alignment"},
		gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			grr.ResizeToContents()
		})
	tb.AddAction(gi.ActOpts{Label: "Open...", Icon: "file-open", Tooltip: "Open a drawing from .svg file"},
		gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			giv.CallMethod(grr, "OpenDrawing", grr.ViewportSafe())
		})
	tb.AddAction(gi.ActOpts{Label: "Save", Icon: "file-save", Tooltip: "Save drawing to .svg file, using current filename (if empty, prompts)"},
		gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			grr.SaveDrawing()
		})
	tb.AddAction(gi.ActOpts{Label: "Save As...", Icon: "file-save", Tooltip: "Save drawing to a new .svg file"},
		gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			giv.CallMethod(grr, "SaveDrawingAs", grr.ViewportSafe())
		})
	expmen := tb.AddAction(gi.ActOpts{Label: "Export", Icon: "file-save"}, nil, nil)
	expmen.Menu.AddAction(gi.ActOpts{Label: "Export PNG", Icon: "file-image", Tooltip: "Export drawing to a .png file -- requires cairosvg.org to be installed"},
		gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			giv.CallMethod(grr, "ExportPNG", grr.ViewportSafe())
		})
	expmen.Menu.AddAction(gi.ActOpts{Label: "Export PDF", Icon: "file-pdf", Tooltip: "Export drawing to a .pdf  file -- requires cairosvg.org to be installed"},
		gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			giv.CallMethod(grr, "ExportPDF", grr.ViewportSafe())
		})

	tb.AddSeparator("sep-undo")
	tb.AddAction(gi.ActOpts{Label: "Undo", Icon: "rotate-left", Tooltip: "Undo last action", UpdateFunc: gv.UndoAvailFunc},
		gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			grr.Undo()
		})
	tb.AddAction(gi.ActOpts{Label: "Redo", Icon: "rotate-right", Tooltip: "Redo last undo action", UpdateFunc: gv.RedoAvailFunc},
		gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			grr.Redo()
		})
	tb.AddSeparator("sep-edit")
	tb.AddAction(gi.ActOpts{Label: "Duplicate", Icon: "documents", Tooltip: "Duplicate current selection -- original items will remain selected", UpdateFunc: gv.SelectedEnableFunc},
		gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			grr.DuplicateSelected()
		})
	tb.AddAction(gi.ActOpts{Label: "Copy", Icon: "copy", Tooltip: "Copy current selection to clipboard", UpdateFunc: gv.SelectedEnableFunc},
		gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			grr.CopySelected()
		})
	tb.AddAction(gi.ActOpts{Label: "Cut", Icon: "cut", Tooltip: "Cut current selection -- delete and copy to clipboard", UpdateFunc: gv.SelectedEnableFunc},
		gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			grr.CutSelected()
		})
	tb.AddAction(gi.ActOpts{Label: "Paste", Icon: "paste", Tooltip: "Paste clipboard contents", UpdateFunc: gv.PasteAvailFunc},
		gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			grr.PasteClip()
		})
	tb.AddSeparator("sep-import")
	tb.AddAction(gi.ActOpts{Label: "Add Image...", Icon: "file-image", Tooltip: "add an image from a file"},
		gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			giv.CallMethod(grr, "AddImage", grr.ViewportSafe())
		})
	tb.AddSeparator("sep-view")
	tb.AddAction(gi.ActOpts{Label: "Zoom Page", Icon: "zoom-out", Tooltip: "zoom to see entire page size for drawing"},
		gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			svvv := grr.SVG()
			svvv.ZoomToPage(false)
			svvv.UpdateView(true)
		})
	tb.AddAction(gi.ActOpts{Label: "Zoom All", Icon: "zoom-out", Tooltip: "zoom to see entire contents"},
		gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			svvv := grr.SVG()
			svvv.ZoomToContents(false)
			svvv.UpdateView(true)
		})
}

func (gv *GridView) ConfigModalToolbar() {
	tb := gv.ModalToolbarStack()
	if tb == nil || tb.HasChildren() {
		return
	}
	tb.SetStretchMaxWidth()
	gi.AddNewToolBar(tb, "select-tb")
	gi.AddNewToolBar(tb, "node-tb")
	gi.AddNewToolBar(tb, "text-tb")

	gv.ConfigSelectToolbar()
	gv.ConfigNodeToolbar()
	gv.ConfigTextToolbar()
}

// ConfigStatusBar configures statusbar with label
func (gv *GridView) ConfigStatusBar() {
	sb := gv.StatusBar()
	if sb == nil || sb.HasChildren() {
		return
	}
	sb.SetStretchMaxWidth()
	sb.SetMinPrefHeight(units.NewValue(1.2, units.Em))
	sb.SetProp("overflow", "hidden") // no scrollbars!
	sb.SetProp("margin", 0)
	sb.SetProp("padding", 0)
	lbl := sb.AddNewChild(gi.KiT_Label, "sb-lbl").(*gi.Label)
	lbl.SetStretchMaxWidth()
	lbl.SetMinPrefHeight(units.NewValue(1, units.Em))
	lbl.SetProp("vertical-align", gist.AlignTop)
	lbl.SetProp("margin", 0)
	lbl.SetProp("padding", 0)
	lbl.SetProp("tab-size", 4)
}

// SetStatus updates the statusbar label with given message, along with other status info
func (gv *GridView) SetStatus(msg string) {
	sb := gv.StatusBar()
	if sb == nil {
		return
	}
	updt := sb.UpdateStart()
	lbl := gv.StatusLabel()
	es := &gv.EditState
	str := "<b>" + strings.TrimSuffix(es.Tool.String(), "Tool") + "</b>\t"
	if es.CurLayer != "" {
		str += "Layer: " + es.CurLayer + "\t\t"
	}
	str += msg
	lbl.SetText(str)
	sb.UpdateEnd(updt)
}

// CloseWindowReq is called when user tries to close window -- we
// automatically save the project if it already exists (no harm), and prompt
// to save open files -- if this returns true, then it is OK to close --
// otherwise not
func (gv *GridView) CloseWindowReq() bool {
	if !gv.EditState.Changed {
		return true
	}
	gi.ChoiceDialog(gv.Viewport, gi.DlgOpts{Title: "Close Drawing: There are Unsaved Changes",
		Prompt: fmt.Sprintf("In Drawing: %v There are <b>unsaved changes</b> -- do you want to save or cancel closing this drawing?", giv.DirAndFile(string(gv.Filename)))},
		[]string{"Cancel", "Save", "Close Without Saving"},
		gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			switch sig {
			case 0:
				// do nothing, will have returned false already
			case 1:
				gv.SaveDrawing()
			case 2:
				gv.ParentWindow().OSWin.Close() // will not be prompted again!
			}
		})
	return false // not yet
}

// QuitReq is called when user tries to quit the app -- we go through all open
// main windows and look for grid windows and call their CloseWindowReq
// functions!
func QuitReq() bool {
	for _, win := range gi.MainWindows {
		if !strings.HasPrefix(win.Nm, "grid-") {
			continue
		}
		mfr, err := win.MainWidget()
		if err != nil {
			continue
		}
		gek := mfr.ChildByName("grid", 0)
		if gek == nil {
			continue
		}
		gv := gek.Embed(KiT_GridView).(*GridView)
		if !gv.CloseWindowReq() {
			return false
		}
	}
	return true
}

func (gv *GridView) SetTitle() {
	if gv.Filename == "" {
		return
	}
	dfnm := giv.DirAndFile(string(gv.Filename))
	winm := "grid-" + dfnm
	wintitle := "grid: " + dfnm
	win := gv.ParentWindow()
	win.SetName(winm)
	win.SetTitle(wintitle)
}

// NewDrawing opens a new drawing window
func NewDrawing(sz PhysSize) *GridView {
	_, ngr := NewGridWindow("")
	ngr.SetPhysSize(&sz)
	return ngr
}

// NewGridWindow returns a new GridWindow loading given file if non-empty
func NewGridWindow(fnm string) (*gi.Window, *GridView) {
	path := ""
	dfnm := ""
	if fnm != "" {
		path, _ = filepath.Abs(fnm)
		dfnm = giv.DirAndFile(path)
	}
	winm := "grid-" + dfnm
	wintitle := "grid: " + dfnm

	if win, found := gi.AllWindows.FindName(winm); found {
		mfr := win.SetMainFrame()
		gv := mfr.Child(0).Embed(KiT_GridView).(*GridView)
		if string(gv.Filename) == path {
			win.OSWin.Raise()
			return win, gv
		}
	}

	width := 1600
	height := 1280
	sc := oswin.TheApp.Screen(0)
	if sc != nil {
		scsz := sc.Geometry.Size()
		width = int(.9 * float64(scsz.X))
		height = int(.8 * float64(scsz.Y))
	}

	win := gi.NewMainWindow(winm, wintitle, width, height)

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()
	gv := AddNewGridView(mfr, "gridview")
	gv.Viewport = vp
	gv.Defaults()
	gv.Config()

	mmen := win.MainMenu
	giv.MainMenuView(gv, win, mmen)

	inClosePrompt := false
	win.OSWin.SetCloseReqFunc(func(w oswin.Window) {
		if !inClosePrompt {
			inClosePrompt = true
			if gv.CloseWindowReq() {
				win.Close()
			} else {
				inClosePrompt = false
			}
		}
	})

	win.OSWin.SetCloseCleanFunc(func(w oswin.Window) {
		if gi.MainWindows.Len() <= 1 {
			go oswin.TheApp.Quit() // once main window is closed, quit
		}
	})

	win.MainMenuUpdated()

	vp.UpdateEndNoSig(updt)

	win.GoStartEventLoop()

	if fnm != "" {
		gv.OpenDrawingFile(gi.FileName(path))
	}

	return win, gv
}

/////////////////////////////////////////////////////////////////////////
//   Controls

// RecycleTab returns a tab with given name, first by looking for an existing one,
// and if not found, making a new one with widget of given type.
// If sel, then select it.  returns widget for tab.
func (gv *GridView) RecycleTab(label string, typ reflect.Type, sel bool) gi.Node2D {
	tv := gv.Tabs()
	return tv.RecycleTab(label, typ, sel)
}

// Tab returns tab with given label
func (gv *GridView) Tab(label string) gi.Node2D {
	tv := gv.Tabs()
	return tv.TabByName(label)
}

func (gv *GridView) ConfigTabs() {
	tv := gv.Tabs()
	tv.NoDeleteTabs = true
	pv := gv.RecycleTab("Paint", KiT_PaintView, false).(*PaintView)
	pv.Config(gv)
	av := gv.RecycleTab("Align", KiT_AlignView, false).(*AlignView)
	av.Config(gv)
	gv.EditState.Text.Defaults()
	txv := gv.RecycleTab("Text", giv.KiT_StructView, false).(*giv.StructView)
	txv.SetStruct(&gv.EditState.Text)
}

func (gv *GridView) PaintView() *PaintView {
	return gv.Tab("Paint").(*PaintView)
}

func (gv *GridView) UpdateAll() {
	gv.UpdateTabs()
	gv.UpdateTreeView()
	gv.UpdateDisp()
}

func (gv *GridView) UpdateDisp() {
	sv := gv.SVG()
	sv.UpdateView(true)
}

func (gv *GridView) UpdateTreeView() {
	tv := gv.TreeView()
	tv.ReSync()
}

func (gv *GridView) SetDefaultStyle() {
	pv := gv.Tab("Paint").(*PaintView)
	es := &gv.EditState
	switch es.Tool {
	case TextTool:
		pv.Update(&Prefs.TextStyle, nil)
	case BezierTool:
		pv.Update(&Prefs.PathStyle, nil)
	default:
		pv.Update(&Prefs.ShapeStyle, nil)
	}
}

func (gv *GridView) UpdateTabs() {
	// fmt.Printf("updt-tabs\n")
	es := &gv.EditState
	fsel := es.FirstSelectedNode()
	if fsel != nil {
		sel := fsel.AsSVGNode()
		pv := gv.Tab("Paint").(*PaintView)
		pv.Update(&sel.Pnt, sel.This())
		txt, istxt := fsel.(*svg.Text)
		if istxt {
			es.Text.SetFromNode(txt)
			txv := gv.Tab("Text").(*giv.StructView)
			txv.UpdateFields()
			// todo: only show text toolbar on double-click
			// gv.SetModalText()
			// gv.UpdateTextToolbar()
		} else {
			gv.SetModalToolbar()
		}
	}
}

// SelectNodeInSVG selects given svg node in SVG drawing
func (gv *GridView) SelectNodeInSVG(kn ki.Ki, mode mouse.SelectModes) {
	sii, ok := kn.(svg.NodeSVG)
	if !ok {
		return
	}
	sv := gv.SVG()
	es := &gv.EditState
	es.SelectAction(sii, mode, image.ZP)
	sv.UpdateView(false)
}

// Undo undoes one step, returning name of action that was undone
func (gv *GridView) Undo() string {
	sv := gv.SVG()
	act := sv.Undo()
	if act != "" {
		gv.SetStatus("Undid: " + act)
	} else {
		gv.SetStatus("Undo: no more to undo")
	}
	gv.UpdateAll()
	return act
}

// Redo redoes one step, returning name of action that was redone
func (gv *GridView) Redo() string {
	sv := gv.SVG()
	act := sv.Redo()
	if act != "" {
		gv.SetStatus("Redid: " + act)
	} else {
		gv.SetStatus("Redo: no more to redo")
	}
	gv.UpdateAll()
	return act
}

// ChangeMade should be called after any change is completed on the drawing.
// Calls autosave.
func (gv *GridView) ChangeMade() {
	go gv.AutoSave()
}

/////////////////////////////////////////////////////////////////////////
//   Basic infrastructure

func (gv *GridView) ConnectEvents2D() {
	gv.OSFileEvent()
}

func (gv *GridView) OSFileEvent() {
	gv.ConnectEvent(oswin.OSOpenFilesEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		ofe := d.(*osevent.OpenFilesEvent)
		for _, fn := range ofe.Files {
			NewGridWindow(fn)
		}
	})
}

// OpenRecent opens a recently-used file
func (gv *GridView) OpenRecent(filename gi.FileName) {
	if string(filename) == GridViewResetRecents {
		SavedPaths = nil
		gi.StringsAddExtras((*[]string)(&SavedPaths), SavedPathsExtras)
	} else if string(filename) == GridViewEditRecents {
		gv.EditRecents()
	} else {
		gv.OpenDrawing(filename)
	}
}

// RecentsEdit opens a dialog editor for deleting from the recents project list
func (gv *GridView) EditRecents() {
	tmp := make([]string, len(SavedPaths))
	copy(tmp, SavedPaths)
	gi.StringsRemoveExtras((*[]string)(&tmp), SavedPathsExtras)
	opts := giv.DlgOpts{Title: "Recent Project Paths", Prompt: "Delete paths you no longer use", Ok: true, Cancel: true, NoAdd: true}
	giv.SliceViewDialog(gv.Viewport, &tmp, opts,
		nil, gv, func(recv, send ki.Ki, sig int64, data interface{}) {
			if sig == int64(gi.DialogAccepted) {
				SavedPaths = nil
				SavedPaths = append(SavedPaths, tmp...)
				gi.StringsAddExtras((*[]string)(&SavedPaths), SavedPathsExtras)
			}
		})
}

// SplitsSetView sets split view splitters to given named setting
func (gv *GridView) SplitsSetView(split SplitName) {
	sv := gv.SplitView()
	sp, _, ok := AvailSplits.SplitByName(split)
	if ok {
		sv.SetSplitsAction(sp.Splits...)
		Prefs.SplitName = split
	}
}

// SplitsSave saves current splitter settings to named splitter settings under
// existing name, and saves to prefs file
func (gv *GridView) SplitsSave(split SplitName) {
	sv := gv.SplitView()
	sp, _, ok := AvailSplits.SplitByName(split)
	if ok {
		sp.SaveSplits(sv.Splits)
		AvailSplits.SavePrefs()
	}
}

// SplitsSaveAs saves current splitter settings to new named splitter settings, and
// saves to prefs file
func (gv *GridView) SplitsSaveAs(name, desc string) {
	spv := gv.SplitView()
	AvailSplits.Add(name, desc, spv.Splits)
	AvailSplits.SavePrefs()
}

// SplitsEdit opens the SplitsView editor to customize saved splitter settings
func (gv *GridView) SplitsEdit() {
	SplitsView(&AvailSplits)
}

// HelpWiki opens wiki page for grid on github
func (gv *GridView) HelpWiki() {
	oswin.TheApp.OpenURL("https://github.com/goki/grid/wiki")
}

////////////////////////////////////////////////////////////////////////////////////////
//		AutoSave

// AutoSaveFilename returns the autosave filename
func (gv *GridView) AutoSaveFilename() string {
	path, fn := filepath.Split(string(gv.Filename))
	if fn == "" {
		fn = "new_file_" + gv.Nm + ".svg"
	}
	asfn := filepath.Join(path, "#"+fn+"#")
	return asfn
}

// AutoSave does the autosave -- safe to call in a separate goroutine
func (gv *GridView) AutoSave() error {
	if gv.HasFlag(int(GridViewAutoSaving)) {
		return nil
	}
	gv.SetFlag(int(GridViewAutoSaving))
	asfn := gv.AutoSaveFilename()
	sv := gv.SVG()
	err := sv.SaveXML(gi.FileName(asfn))
	if err != nil && err != io.EOF {
		log.Println(err)
	}
	gv.ClearFlag(int(GridViewAutoSaving))
	return err
}

// AutoSaveDelete deletes any existing autosave file
func (gv *GridView) AutoSaveDelete() {
	asfn := gv.AutoSaveFilename()
	os.Remove(asfn)
}

// AutoSaveCheck checks if an autosave file exists -- logic for dealing with
// it is left to larger app -- call this before opening a file
func (gv *GridView) AutoSaveCheck() bool {
	asfn := gv.AutoSaveFilename()
	if _, err := os.Stat(asfn); os.IsNotExist(err) {
		return false // does not exist
	}
	return true
}

/////////////////////////////////////////////////////////////////////////
//   Props, MainMenu

// GridViewFlags extend NodeBase NodeFlags to hold viewport state
type GridViewFlags int

//go:generate stringer -type=GridViewFlags

var KiT_GridViewFlags = kit.Enums.AddEnumExt(gi.KiT_NodeFlags, GridViewFlagsN, kit.BitFlag, nil)

const (
	// VpFlagPopup means viewport is a popup (menu or dialog) -- does not obey
	// parent bounds (otherwise does)
	GridViewAutoSaving GridViewFlags = GridViewFlags(gi.NodeFlagsN) + iota

	GridViewFlagsN
)

/////////////////////////////////////////////////////////////////////////
//   Props, MainMenu

var GridViewProps = ki.Props{
	"EnumType:Flag":    gi.KiT_NodeFlags,
	"background-color": &gi.Prefs.Colors.Background,
	"color":            &gi.Prefs.Colors.Font,
	"max-width":        -1,
	"max-height":       -1,
	"#title": ki.Props{
		"max-width":        -1,
		"horizontal-align": gist.AlignCenter,
		"vertical-align":   gist.AlignTop,
	},
	"MainMenu": ki.PropSlice{
		{"AppMenu", ki.BlankProp{}},
		{"File", ki.PropSlice{
			{"OpenRecent", ki.Props{
				"submenu": &SavedPaths,
				"Args": ki.PropSlice{
					{"File Name", ki.Props{}},
				},
			}},
			{"OpenDrawing", ki.Props{
				"shortcut": gi.KeyFunMenuOpen,
				"label":    "Open SVG...",
				"desc":     "open an SVG drawing",
				"Args": ki.PropSlice{
					{"File Name", ki.Props{
						"ext": ".svg",
					}},
				},
			}},
			{"NewDrawing", ki.Props{
				"shortcut": gi.KeyFunMenuNew,
				"label":    "New",
				"desc":     "Create a new drawing of given physical size (size units are used for ViewBox).",
				"Args": ki.PropSlice{
					{"Physical Size", ki.Props{
						"default": Prefs.Size,
					}},
				},
			}},
			{"SaveDrawing", ki.Props{
				"shortcut": gi.KeyFunMenuSave,
				"label":    "Save Drawing",
			}},
			{"SaveDrawingAs", ki.Props{
				"shortcut": gi.KeyFunMenuSaveAs,
				"label":    "Save As...",
				"desc":     "Save drawing to given svg file name",
				"Args": ki.PropSlice{
					{"File Name", ki.Props{
						"ext": ".svg",
					}},
				},
			}},
			{"sep-file", ki.BlankProp{}},
			{"PromptPhysSize", ki.Props{
				"label": "Set Size",
				"desc":  "sets the physical size (size units are used for ViewBox)",
			}},
			{"ResizeToContents", ki.Props{
				"label": "Resize To Contents",
				"desc":  "resizes the drawing to fit the current contents, moving everything to upper-left corner while preserving grid alignment",
			}},
			{"sep-exp", ki.BlankProp{}},
			{"ExportPNG", ki.Props{
				"desc": "Export drawing as a PNG image file (uses cairosvg -- must install!) -- specify either width or height in pixels as non-zero, or both 0 to use physical size.  Renders full page -- do Resize To Contents to only render contents.",
				"Args": ki.PropSlice{
					{"Width", ki.Props{
						"default": 1280,
					}},
					{"Height", ki.Props{
						"default": 0,
					}},
				},
			}},
			{"ExportPDF", ki.Props{
				"desc": "Export drawing as a PDF file (uses cairosvg -- must install!), at given specified DPI (only relevant for rendered effects).  Renders full page -- do Resize To Contents to only render contents.",
				"Args": ki.PropSlice{
					{"DPI", ki.Props{
						"default": 300,
					}},
				},
			}},
			{"sep-imp", ki.BlankProp{}},
			{"AddImage", ki.Props{
				"label": "Add Image...",
				"desc":  "Add a new Image node with given image file for this image node, rescaling to given size -- use 0, 0 to use native image size.",
				"Args": ki.PropSlice{
					{"File Name", ki.Props{
						"default-field": "Filename",
						"ext":           ".png,.jpg,.jpeg",
					}},
					{"Width", ki.Props{}},
					{"Height", ki.Props{}},
				},
			}},
			{"sep-af", ki.BlankProp{}},
			{"Close Window", ki.BlankProp{}},
		}},
		{"Edit", ki.PropSlice{
			{"Duplicate", ki.Props{
				"keyfun": gi.KeyFunDuplicate,
				// "updtfunc": GridViewInactiveTextSelectionFunc,
			}},
			{"Copy", ki.Props{
				"keyfun": gi.KeyFunCopy,
				// "updtfunc": GridViewInactiveTextSelectionFunc,
			}},
			{"Cut", ki.Props{
				"keyfun": gi.KeyFunCut,
				// "updtfunc": GridViewInactiveTextSelectionFunc,
			}},
			{"Paste", ki.Props{
				"keyfun": gi.KeyFunPaste,
			}},
			{"sep-undo", ki.BlankProp{}},
			{"Undo", ki.Props{
				"keyfun": gi.KeyFunUndo,
			}},
			{"Redo", ki.Props{
				"keyfun": gi.KeyFunRedo,
			}},
		}},
		{"View", ki.PropSlice{
			{"Splits", ki.PropSlice{
				{"SplitsSetView", ki.Props{
					"label":   "Set View",
					"submenu": &AvailSplitNames,
					"Args": ki.PropSlice{
						{"Split Name", ki.Props{}},
					},
				}},
				{"SplitsSaveAs", ki.Props{
					"label": "Save As...",
					"desc":  "save current splitter values to a new named split configuration",
					"Args": ki.PropSlice{
						{"Name", ki.Props{
							"width": 60,
						}},
						{"Desc", ki.Props{
							"width": 60,
						}},
					},
				}},
				{"SplitsSave", ki.Props{
					"label":   "Save",
					"submenu": &AvailSplitNames,
					"Args": ki.PropSlice{
						{"Split Name", ki.Props{}},
					},
				}},
				{"SplitsEdit", ki.Props{
					"label": "Edit...",
				}},
			}},
		}},
		{"Window", "Windows"},
		{"Help", ki.PropSlice{
			{"HelpWiki", ki.Props{}},
		}},
	},
}
