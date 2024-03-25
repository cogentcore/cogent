// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vector

//go:generate core generate

import (
	"errors"
	"fmt"
	"image"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"cogentcore.org/core/events"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/giv"
	"cogentcore.org/core/glop/dirs"
	"cogentcore.org/core/grr"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keyfun"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/mat32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/svg"
)

// VectorView is the Vector SVG vector drawing program
type VectorView struct {
	gi.Frame

	// full path to current drawing filename
	Filename gi.Filename `ext:".svg" set:"-"`

	// current edit state
	EditState EditState `set:"-"`
}

func (vv *VectorView) OnInit() {
	vv.Frame.OnInit()
	vv.SetStyles()
	vv.EditState.ConfigDefaultGradient()
}

func (vv *VectorView) OnAdd() {
	vv.Frame.OnAdd()
	vv.AddCloseDialog()
}

func (vv *VectorView) SetStyles() {
	vv.Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})
	vv.OnWidgetAdded(func(w gi.Widget) {
		switch w.PathFrom(vv) {
		case "splits/tabs":
			w.(*gi.Tabs).SetType(gi.FunctionalTabs)
		}
	})
}

// OpenDrawingFile opens a new .svg drawing file -- just the basic opening
func (vv *VectorView) OpenDrawingFile(fnm gi.Filename) error {
	path, _ := filepath.Abs(string(fnm))
	vv.Filename = gi.Filename(path)
	sv := vv.SVG()
	err := grr.Log(sv.SSVG().OpenXML(path))
	// SavedPaths.AddPath(path, gi.Prefs.Params.SavedPathsMax)
	// SavePaths()
	fdir, _ := filepath.Split(path)
	grr.Log(os.Chdir(fdir))
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
func (vv *VectorView) OpenDrawing(fnm gi.Filename) error { //gti:add
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
func (vv *VectorView) PromptPhysSize() { //gti:add
	sv := vv.SVG()
	sz := &PhysSize{}
	sz.SetFromSVG(sv)
	d := gi.NewBody().AddTitle("SVG physical size")
	giv.NewStructView(d).SetStruct(sz)
	d.AddBottomBar(func(pw gi.Widget) {
		d.AddCancel(pw)
		d.AddOk(pw).OnClick(func(e events.Event) {
			vv.SetPhysSize(sz)
			sv.bgVectorEff = -1
			sv.UpdateView(true)
		})
	})
	d.NewDialog(vv).Run()
}

// SetPhysSize sets physical size of drawing
func (vv *VectorView) SetPhysSize(sz *PhysSize) {
	if sz == nil {
		return
	}
	if sz.Size == (mat32.Vec2{}) {
		sz.SetStdSize(Prefs.Size.StdSize)
	}
	sv := vv.SVG()
	sz.SetToSVG(sv)
	sv.SetMetaData()
	sv.ZoomToPage(false)
}

// SaveDrawing saves .svg drawing to current filename
func (vv *VectorView) SaveDrawing() error { //gti:add
	if vv.Filename != "" {
		return vv.SaveDrawingAs(vv.Filename)
	}
	giv.CallFunc(vv, vv.SaveDrawingAs)
	return nil
}

// SaveDrawingAs saves .svg drawing to given filename
func (vv *VectorView) SaveDrawingAs(fname gi.Filename) error { //gti:add
	if fname == "" {
		return errors.New("SaveDrawingAs: filename is empty")
	}
	path, _ := filepath.Abs(string(fname))
	vv.Filename = gi.Filename(path)
	// SavedPaths.AddPath(path, gi.Prefs.Params.SavedPathsMax)
	// SavePaths()
	sv := vv.SVG()
	sv.SSVG().RemoveOrphanedDefs()
	sv.SetMetaData()
	err := sv.SSVG().SaveXML(path)
	if grr.Log(err) == nil {
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
func (vv *VectorView) ExportPNG(width, height float32) error { //gti:add
	path, _ := filepath.Split(string(vv.Filename))
	fnm := filepath.Join(path, "export_png.svg")
	sv := vv.SVG()
	err := sv.SSVG().SaveXML(fnm)
	if grr.Log(err) != nil {
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
func (vv *VectorView) ExportPDF(dpi float32) error { //gti:add
	path, _ := filepath.Split(string(vv.Filename))
	fnm := filepath.Join(path, "export_pdf.svg")
	sv := vv.SVG()
	err := sv.SSVG().SaveXML(fnm)
	if grr.Log(err) != nil {
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
func (vv *VectorView) ResizeToContents() { //gti:add
	sv := vv.SVG()
	sv.ResizeToContents(true)
	sv.UpdateView(true)
}

// AddImage adds a new image node set to the given image
func (vv *VectorView) AddImage(fname gi.Filename, width, height float32) error { //gti:add
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

func (vv *VectorView) ModalToolbarStack() *gi.Layout {
	return vv.ChildByName("modal-tb", 1).(*gi.Layout)
}

// SetModalSelect sets the modal toolbar to be the select one
func (vv *VectorView) SetModalSelect() {
	tbs := vv.ModalToolbarStack()
	vv.UpdateSelectToolbar()
	idx, _ := tbs.Kids.IndexByName("select-tb", 0)
	tbs.StackTop = idx
	tbs.NeedsLayout()
}

// SetModalNode sets the modal toolbar to be the node editing one
func (vv *VectorView) SetModalNode() {
	tbs := vv.ModalToolbarStack()
	vv.UpdateNodeToolbar()
	idx, _ := tbs.Kids.IndexByName("node-tb", 1)
	tbs.StackTop = idx
	tbs.NeedsLayout()
}

// SetModalText sets the modal toolbar to be the text editing one
func (vv *VectorView) SetModalText() {
	tbs := vv.ModalToolbarStack()
	vv.UpdateTextToolbar()
	idx, _ := tbs.Kids.IndexByName("text-tb", 2)
	tbs.StackTop = idx
	tbs.NeedsLayout()
}

func (vv *VectorView) HBox() *gi.Frame {
	return vv.ChildByName("hbox", 2).(*gi.Frame)
}

func (vv *VectorView) Tools() *gi.Toolbar {
	return vv.HBox().ChildByName("tools", 0).(*gi.Toolbar)
}

func (vv *VectorView) Splits() *gi.Splits {
	return vv.HBox().ChildByName("splits", 1).(*gi.Splits)
}

func (vv *VectorView) LayerTree() *gi.Layout {
	return vv.Splits().ChildByName("layer-tree", 0).(*gi.Layout)
}

func (vv *VectorView) LayerView() *giv.TableView {
	return vv.LayerTree().ChildByName("layers", 0).(*giv.TableView)
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

func (vv *VectorView) Tabs() *gi.Tabs {
	return vv.Splits().ChildByName("tabs", 2).(*gi.Tabs)
}

// StatusBar returns the statusbar widget
func (vv *VectorView) StatusBar() *gi.Frame {
	return vv.ChildByName("statusbar", 4).(*gi.Frame)
}

// StatusLabel returns the statusbar label widget
func (vv *VectorView) StatusLabel() *gi.Label {
	return vv.StatusBar().Child(0).(*gi.Label)
}

// Config configures entire view -- only runs if no children yet
func (vv *VectorView) Config() {
	if vv.HasChildren() {
		return
	}
	gi.NewLayout(vv, "modal-tb").Style(func(s *styles.Style) {
		s.Display = styles.Stacked
	})
	hb := gi.NewFrame(vv, "hbox")
	gi.NewFrame(vv, "statusbar").Style(func(s *styles.Style) {
		s.Grow.Set(1, 0)
	})

	gi.NewToolbar(hb, "tools").Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})
	sp := gi.NewSplits(hb, "splits").SetSplits(0.15, 0.60, 0.25)

	tly := gi.NewLayout(sp, "layer-tree").Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})

	giv.NewFuncButton(tly, vv.AddLayer)

	lyv := giv.NewTableView(tly, "layers")

	tvfr := gi.NewFrame(tly, "tree-frame").Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})
	tv := NewTreeView(tvfr, "treeview")
	tv.VectorView = vv
	tv.OpenDepth = 4

	sv := NewSVGView(sp, "svg")
	sv.VectorView = vv

	gi.NewTabs(sp, "tabs")

	tv.SyncTree(sv.Root())

	// tv.TreeViewSig.Connect(vv.This(), func(recv, send ki.Ki, sig int64, data any) {
	// 	gvv := recv.Embed(KiT_VectorView).(*VectorView)
	// 	if data == nil {
	// 		return
	// 	}
	// 	if sig == int64(giv.TreeViewInserted) {
	// 		sn, ok := data.(svg.Node)
	// 		if ok {
	// 			gvv.SVG().NodeEnsureUniqueId(sn)
	// 			svg.CloneNodeGradientProp(sn, "fill")
	// 			svg.CloneNodeGradientProp(sn, "stroke")
	// 		}
	// 		return
	// 	}
	// 	if sig == int64(giv.TreeViewDeleted) {
	// 		sn, ok := data.(svg.Node)
	// 		if ok {
	// 			svg.DeleteNodeGradientProp(sn, "fill")
	// 			svg.DeleteNodeGradientProp(sn, "stroke")
	// 		}
	// 		return
	// 	}
	// 	if sig != int64(giv.TreeViewOpened) {
	// 		return
	// 	}
	// 	tvn, _ := data.(ki.Ki).Embed(KiT_TreeView).(*TreeView)
	// 	_, issvg := tvn.SrcNode.(svg.Node)
	// 	if !issvg {
	// 		return
	// 	}
	// 	if tvn.SrcNode.HasChildren() {
	// 		return
	// 	}
	// 	giv.StructViewDialog(gvv.Viewport, tvn.SrcNode, giv.DlgOpts{Title: "SVG Element View"}, nil, nil)
	// 	// ggv, _ := recv.Embed(KiT_VectorView).(*VectorView)
	// 	// 		stv := ggv.RecycleTab("Obj", giv.KiT_StructView, true).(*giv.StructView)
	// 	// 		stv.SetStruct(tvn.SrcNode)
	// })

	vv.ConfigStatusBar()
	vv.ConfigModalToolbar()
	vv.ConfigTools()
	vv.ConfigTabs()

	vv.SetPhysSize(&Prefs.Size)

	vv.SyncLayers()
	lyv.SetSlice(&vv.EditState.Layers)
	vv.LayerViewSigs(lyv)

	sv.UpdateGradients(vv.EditState.Gradients)
}

// IsConfiged returns true if the view is fully configured
func (vv *VectorView) IsConfiged() bool {
	if !vv.HasChildren() {
		return false
	}
	return true
}

// PasteAvailFunc is an ActionUpdateFunc that inactivates action if no paste avail
func (vv *VectorView) PasteAvailFunc(bt *gi.Button) {
	bt.SetEnabled(!vv.Clipboard().IsEmpty())
}

func (vv *VectorView) ConfigToolbar(tb *gi.Toolbar) {
	// TODO(kai): remove Update
	giv.NewFuncButton(tb, vv.UpdateAll).SetText("Update").SetIcon(icons.Update)
	gi.NewButton(tb).SetText("New").SetIcon(icons.Add).
		OnClick(func(e events.Event) {
			ndr := vv.NewDrawing(Prefs.Size)
			ndr.PromptPhysSize()
		})

	gi.NewButton(tb).SetText("Size").SetIcon(icons.FormatSize).SetMenu(func(m *gi.Scene) {
		giv.NewFuncButton(m, vv.PromptPhysSize).SetText("Set size").
			SetIcon(icons.FormatSize)
		giv.NewFuncButton(m, vv.ResizeToContents).SetIcon(icons.Resize)
	})

	giv.NewFuncButton(tb, vv.OpenDrawing).SetText("Open").SetIcon(icons.Open)
	giv.NewFuncButton(tb, vv.SaveDrawing).SetText("Save").SetIcon(icons.Save)
	giv.NewFuncButton(tb, vv.SaveDrawingAs).SetText("Save as").SetIcon(icons.SaveAs)

	gi.NewButton(tb).SetText("Export").SetIcon(icons.ExportNotes).SetMenu(func(m *gi.Scene) {
		giv.NewFuncButton(m, vv.ExportPNG).SetIcon(icons.Image)
		giv.NewFuncButton(m, vv.ExportPDF).SetIcon(icons.PictureAsPdf)
	})

	gi.NewSeparator(tb)

	giv.NewFuncButton(tb, vv.Undo).StyleFirst(func(s *styles.Style) {
		s.SetEnabled(vv.EditState.UndoMgr.HasUndoAvail())
	})
	giv.NewFuncButton(tb, vv.Redo).StyleFirst(func(s *styles.Style) {
		s.SetEnabled(vv.EditState.UndoMgr.HasRedoAvail())
	})

	gi.NewSeparator(tb)

	giv.NewFuncButton(tb, vv.DuplicateSelected).SetText("Duplicate").SetIcon(icons.Copy).SetKey(keyfun.Duplicate)
	giv.NewFuncButton(tb, vv.CopySelected).SetText("Copy").SetIcon(icons.Copy).SetKey(keyfun.Copy)
	giv.NewFuncButton(tb, vv.CutSelected).SetText("Cut").SetIcon(icons.Cut).SetKey(keyfun.Cut)
	giv.NewFuncButton(tb, vv.PasteClip).SetText("Paste").SetIcon(icons.Paste).SetKey(keyfun.Paste)

	gi.NewSeparator(tb)
	giv.NewFuncButton(tb, vv.AddImage).SetIcon(icons.Image)
	gi.NewSeparator(tb)

	gi.NewButton(tb).SetText("Zoom page").SetIcon(icons.ZoomOut).
		SetTooltip("Zoom to see the entire page size for drawing").
		OnClick(func(e events.Event) {
			sv := vv.SVG()
			sv.ZoomToPage(false)
			sv.UpdateView(true)
		})

	gi.NewButton(tb).SetText("Zoom all").SetIcon(icons.ZoomOut).
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
	gi.NewToolbar(tb, "select-tb")
	gi.NewToolbar(tb, "node-tb")
	gi.NewToolbar(tb, "text-tb")

	vv.ConfigSelectToolbar()
	vv.ConfigNodeToolbar()
	vv.ConfigTextToolbar()
}

// ConfigStatusBar configures statusbar with label
func (vv *VectorView) ConfigStatusBar() {
	sb := vv.StatusBar()
	if sb == nil || sb.HasChildren() {
		return
	}
	gi.NewLabel(sb, "sb-lbl")
}

// SetStatus updates the statusbar label with given message, along with other status info
func (vv *VectorView) SetStatus(msg string) {
	sb := vv.StatusBar()
	if sb == nil {
		return
	}
	lbl := vv.StatusLabel()
	es := &vv.EditState
	str := "<b>" + strings.TrimSuffix(es.Tool.String(), "Tool") + "</b>\t"
	if es.CurLayer != "" {
		str += "Layer: " + es.CurLayer + "\t\t"
	}
	str += msg
	lbl.SetText(str)
}

// AddCloseDialog adds the close dialog that prompts the user to save the
// file when they try to close the scene containing this vector view.
func (vv *VectorView) AddCloseDialog() {
	vv.WidgetBase.AddCloseDialog(func(d *gi.Body) bool {
		if !vv.EditState.Changed {
			return false
		}
		d.AddTitle("Unsaved changes").
			AddText(fmt.Sprintf("There are unsaved changes in %s", dirs.DirAndFile(string(vv.Filename))))
		d.AddBottomBar(func(pw gi.Widget) {
			d.AddOk(pw, "cws").SetText("Close without saving").OnClick(func(e events.Event) {
				vv.Scene.Close()
			})
			d.AddOk(pw, "sa").SetText("Save and close").OnClick(func(e events.Event) {
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
	win := vv.Scene.RenderWin()
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

	if win, found := gi.AllRenderWins.FindName(winm); found {
		sc := win.MainScene()
		if vv, ok := sc.Body.ChildByType(VectorViewType, ki.NoEmbeds).(*VectorView); ok {
			if string(vv.Filename) == path {
				win.Raise()
				return vv
			}
		}
	}

	b := gi.NewBody(winm).SetTitle(winm)

	vv := NewVectorView(b)
	b.AddAppBar(vv.ConfigToolbar)

	b.OnShow(func(e events.Event) {
		if fnm != "" {
			vv.OpenDrawingFile(gi.Filename(path))
		} else {
			vv.EditState.Init(vv)
		}
	})

	b.NewWindow().Run()

	return vv
}

/////////////////////////////////////////////////////////////////////////
//   Controls

// RecycleTab returns the tab with given the name, first by looking for
// an existing one, and if not found, making a new one.
// If sel, then select it.
func (gv *VectorView) RecycleTab(label string, sel bool) *gi.Frame {
	tv := gv.Tabs()
	return tv.RecycleTab(label, sel)
}

// Tab returns the tab with the given label
func (gv *VectorView) Tab(label string) *gi.Frame {
	return gv.Tabs().TabByName(label)
}

func (vv *VectorView) ConfigTabs() {
	pt := vv.RecycleTab("Paint", false)
	NewPaintView(pt).SetVectorView(vv)
	at := vv.RecycleTab("Align", false)
	NewAlignView(at).SetVectorView(vv)
	vv.EditState.Text.Defaults()
	tt := vv.RecycleTab("Text", false)
	giv.NewStructView(tt).SetStruct(&vv.EditState.Text)
}

func (vv *VectorView) PaintView() *PaintView {
	return vv.Tab("Paint").Child(0).(*PaintView)
}

// UpdateAll updates the display
func (vv *VectorView) UpdateAll() { //gti:add
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
	// 	pv.Update(&Prefs.TextStyle, nil)
	// case BezierTool:
	// 	pv.Update(&Prefs.PathStyle, nil)
	// default:
	// 	pv.Update(&Prefs.ShapeStyle, nil)
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
	// 		txv := vv.Tab("Text").(*giv.StructView)
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
func (vv *VectorView) SelectNodeInSVG(kn ki.Ki, mode events.SelectModes) {
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
func (vv *VectorView) Undo() string { //gti:add
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
func (vv *VectorView) Redo() string { //gti:add
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
	gv.ConnectEvent(oswin.OSOpenFilesEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d any) {
		ofe := d.(*osevent.OpenFilesEvent)
		for _, fn := range ofe.Files {
			NewVectorWindow(fn)
		}
	})
}
*/

// OpenRecent opens a recently-used file
func (vv *VectorView) OpenRecent(filename gi.Filename) {
	// if string(filename) == VectorViewResetRecents {
	// 	SavedPaths = nil
	// 	gi.StringsAddExtras((*[]string)(&SavedPaths), SavedPathsExtras)
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
	// gi.StringsRemoveExtras((*[]string)(&tmp), SavedPathsExtras)
	// opts := giv.DlgOpts{Title: "Recent Project Paths", Prompt: "Delete paths you no longer use", Ok: true, Cancel: true, NoAdd: true}
	// giv.SliceViewDialog(vv.Viewport, &tmp, opts,
	// 	nil, vv, func(recv, send ki.Ki, sig int64, data any) {
	// 		if sig == int64(gi.DialogAccepted) {
	// 			SavedPaths = nil
	// 			SavedPaths = append(SavedPaths, tmp...)
	// 			gi.StringsAddExtras((*[]string)(&SavedPaths), SavedPathsExtras)
	// 		}
	// 	})
}

// SplitsSetView sets split view splitters to given named setting
func (vv *VectorView) SplitsSetView(split SplitName) {
	sv := vv.Splits()
	sp, _, ok := AvailSplits.SplitByName(split)
	if ok {
		sv.SetSplits(sp.Splits...).NeedsLayout()
		Prefs.SplitName = split
	}
}

// SplitsSave saves current splitter settings to named splitter settings under
// existing name, and saves to prefs file
func (vv *VectorView) SplitsSave(split SplitName) {
	sv := vv.Splits()
	sp, _, ok := AvailSplits.SplitByName(split)
	if ok {
		sp.SaveSplits(sv.Splits)
		AvailSplits.SavePrefs()
	}
}

// SplitsSaveAs saves current splitter settings to new named splitter settings, and
// saves to prefs file
func (vv *VectorView) SplitsSaveAs(name, desc string) {
	spv := vv.Splits()
	AvailSplits.Add(name, desc, spv.Splits)
	AvailSplits.SavePrefs()
}

// SplitsEdit opens the SplitsView editor to customize saved splitter settings
func (vv *VectorView) SplitsEdit() {
	SplitsView(&AvailSplits)
}

// HelpWiki opens wiki page for grid on github
func (vv *VectorView) HelpWiki() {
	gi.TheApp.OpenURL("https://vector.cogentcore.org")
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
	// err := sv.SaveXML(gi.Filename(asfn))
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
type VectorViewFlags gi.WidgetFlags //enums:bitflag -trim-prefix VectorViewFlag

const (
	// VectorViewAutoSaving means
	VectorViewAutoSaving VectorViewFlags = VectorViewFlags(gi.WidgetFlagsN) + iota
)
