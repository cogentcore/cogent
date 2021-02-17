// Copyright (c) 2021, The Grid Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grid

import (
	"errors"
	"io"
	"log"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/gist"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/dnd"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/gi/svg"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
	"github.com/goki/pi/filecat"
)

// GridView is the Grid SVG vector drawing program: Go-rendered interactive drawing
type GridView struct {
	gi.Frame
	FilePath  gi.FileName `ext:".svg" desc:"full path to current drawing filename"`
	Prefs     Preferences `desc:"current drawing preferences"`
	EditState EditState   `desc:"current edit state"`
}

var KiT_GridView = kit.Types.AddType(&GridView{}, GridViewProps)

// AddNewGridView adds a new editor to given parent node, with given name.
func AddNewGridView(parent ki.Ki, name string) *GridView {
	gv := parent.AddNewChild(KiT_GridView, name).(*GridView)
	return gv
}

func (g *GridView) CopyFieldsFrom(frm interface{}) {
	fr := frm.(*GridView)
	g.Frame.CopyFieldsFrom(&fr.Frame)
	// todo: fill out
}

func (gv *GridView) Defaults() {
	gv.Prefs.Defaults()
	// gr.Prefs = Prefs
}

// OpenDrawing opens a new .svg drawing
func (gv *GridView) OpenDrawing(fnm gi.FileName) error {
	path, _ := filepath.Abs(string(fnm))
	gv.FilePath = gi.FileName(path)
	gv.SetTitle()
	// TheFile.SetText(CurFilename)
	sg := gv.SVG()
	err := sg.OpenXML(path)
	if err != nil && err != io.EOF {
		log.Println(err)
		// return err
	}
	SavedPaths.AddPath(path, gi.Prefs.Params.SavedPathsMax)
	SavePaths()
	gv.EditState.Init()
	gv.EditState.Gradients = sg.Gradients()
	sg.GatherIds() // also ensures uniqueness, key for json saving
	sg.SetNormXForm()
	scx, scy := sg.Pnt.XForm.ExtractScale()
	sg.Scale = 0.5 * (scx + scy)
	sg.Trans.Set(0, 0)
	sg.SetTransform()
	tv := gv.TreeView()
	tv.CloseAll()
	tv.Open()
	gv.SetStatus("Opened: " + path)
	return nil
}

// NewDrawing opens a new drawing window
func (gv *GridView) NewDrawing() *GridView {
	_, ngr := NewGridWindow("")
	return ngr
}

// SaveDrawing saves .svg drawing to current filename
func (gv *GridView) SaveDrawing() error {
	if gv.FilePath == "" {
		giv.CallMethod(gv, "SaveDrawingAs", gv.ViewportSafe())
		return nil
	}
	sg := gv.SVG()
	sg.RemoveOrphanedDefs()
	err := sg.SaveXML(string(gv.FilePath))
	if err != nil && err != io.EOF {
		log.Println(err)
	}
	gv.SetStatus("Saved: " + string(gv.FilePath))
	return err
}

// SaveDrawingAs saves .svg drawing to given filename
func (gv *GridView) SaveDrawingAs(fname gi.FileName) error {
	if fname == "" {
		return errors.New("SaveDrawingAs: filename is empty")
	}
	path, _ := filepath.Abs(string(fname))
	gv.FilePath = gi.FileName(path)
	SavedPaths.AddPath(path, gi.Prefs.Params.SavedPathsMax)
	SavePaths()
	sg := gv.SVG()
	sg.RemoveOrphanedDefs()
	err := sg.SaveXML(path)
	if err != nil && err != io.EOF {
		log.Println(err)
	}
	gv.SetStatus("Saved: " + path)
	return err
}

// SetTool sets the current active tool
func (gv *GridView) SetTool(tl Tools) {
	tls := gv.Tools()
	updt := tls.UpdateStart()
	for i, ti := range tls.Kids {
		t := ti.(gi.Node2D).AsNode2D()
		t.SetSelectedState(i == int(tl))
	}
	tls.UpdateEnd(updt)
	gv.EditState.Tool = tl
	gv.SetStatus("Tool")
}

func (gv *GridView) MainToolbar() *gi.ToolBar {
	return gv.ChildByName("main-tb", 0).(*gi.ToolBar)
}

func (gv *GridView) ModalToolbar() *gi.ToolBar {
	return gv.ChildByName("modal-tb", 1).(*gi.ToolBar)
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

func (gv *GridView) TreeView() *TreeView {
	return gv.SplitView().ChildByName("tree-frame", 0).Child(0).(*TreeView) // note: name changes
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
	gv.SetProp("spacing", gi.StdDialogVSpaceUnits)
	gi.AddNewToolBar(gv, "main-tb")
	gi.AddNewToolBar(gv, "modal-tb")
	hb := gi.AddNewLayout(gv, "hbox", gi.LayoutHoriz)
	hb.SetStretchMax()
	gi.AddNewFrame(gv, "statusbar", gi.LayoutHoriz)

	tb := gi.AddNewToolBar(hb, "tools")
	tb.Lay = gi.LayoutVert
	sv := gi.AddNewSplitView(hb, "splitview")
	sv.Dim = mat32.X

	tvfr := gi.AddNewFrame(sv, "tree-frame", gi.LayoutHoriz)
	tvfr.SetStretchMax()
	tvfr.SetReRenderAnchor()
	tv := AddNewTreeView(tvfr, "treeview")
	tv.GridView = gv
	tv.OpenDepth = 1

	sg := AddNewSVGView(sv, "svg", gv)

	tab := gi.AddNewTabView(sv, "tabs")
	tab.SetStretchMaxWidth()

	tv.SetRootNode(sg)

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

	sv.SetSplits(0.1, 0.65, 0.25)

	gv.ConfigStatusBar()
	gv.ConfigMainToolbar()
	gv.ConfigModalToolbar()
	gv.ConfigTools()
	gv.ConfigTabs()

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

func (gv *GridView) ConfigMainToolbar() {
	tb := gv.MainToolbar()
	tb.SetStretchMaxWidth()
	tb.AddAction(gi.ActOpts{Label: "New", Icon: "new", Tooltip: "create new drawing using default drawing preferences"},
		gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			grr.NewDrawing()
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
	tb.AddSeparator("sep-edit")
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
}

func (gv *GridView) ConfigModalToolbar() {
	tb := gv.ModalToolbar()
	tb.SetStretchMaxWidth()
}

func (gv *GridView) ConfigTools() {
	tb := gv.Tools()
	tb.Lay = gi.LayoutVert
	tb.SetStretchMaxHeight()
	tb.AddAction(gi.ActOpts{Label: "S", Icon: "arrow", Tooltip: "S, Space: select, move, resize objects"},
		gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			grr.SetTool(SelectTool)
		})
	tb.AddAction(gi.ActOpts{Label: "N", Icon: "edit", Tooltip: "N: select, move node points within paths"},
		gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			grr.SetTool(NodeTool)
		})
	tb.AddAction(gi.ActOpts{Label: "R", Icon: "stop", Tooltip: "R: create rectangles and squares"},
		gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			grr.SetTool(RectTool)
		})
	tb.AddAction(gi.ActOpts{Label: "E", Icon: "circlebutton-off", Tooltip: "E: create circles, ellipses, and arcs"},
		gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			grr.SetTool(EllipseTool)
		})
	tb.AddAction(gi.ActOpts{Label: "B", Icon: "color", Tooltip: "B: create bezier curves (straight lines, curves with control points)"},
		gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			grr.SetTool(BezierTool)
		})

	gv.SetTool(SelectTool)
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
	// todo: do this
	// gi.ChoiceDialog(gv.Viewport, gi.DlgOpts{Title: "Close Project: There are Unsaved Files",
	// 	Prompt: fmt.Sprintf("In Project: %v There are <b>%v</b> opened files with <b>unsaved changes</b> -- do you want to save all or cancel closing this project and review  / save those files first?", gv.Nm, nch)},
	// 	[]string{"Cancel", "Save All", "Close Without Saving"},
	// 	gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
	// 		switch sig {
	// 		case 0:
	// 			// do nothing, will have returned false already
	// 		case 1:
	// 			gv.SaveAllOpenNodes()
	// 		case 2:
	// 			gv.ParentWindow().OSWin.Close() // will not be prompted again!
	// 		}
	// 	})
	// return false // not yet
	return true
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
	if gv.FilePath == "" {
		return
	}
	dfnm := giv.DirAndFile(string(gv.FilePath))
	winm := "grid-" + dfnm
	wintitle := "grid: " + dfnm
	win := gv.ParentWindow()
	win.SetName(winm)
	win.SetTitle(wintitle)
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
		if string(gv.FilePath) == path {
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

	if fnm != "" {
		gv.OpenDrawing(gi.FileName(path))
	}

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
	// gv.RecycleTab("Obj", giv.KiT_StructView, false)
}

func (gv *GridView) PaintView() *PaintView {
	return gv.Tab("Paint").(*PaintView)
}

func (gv *GridView) UpdateTabs() {
	// fmt.Printf("updt-tabs\n")
	es := &gv.EditState
	sls := es.SelectedList(false)
	if len(sls) > 0 {
		sel := sls[0].AsSVGNode()
		pv := gv.Tab("Paint").(*PaintView)
		pv.Update(sel)
	}
}

/////////////////////////////////////////////////////////////////////////
//  Actions

// ManipAction manages all the updating etc associated with performing an action
// that includes an ongoing manipulation with a final non-manip update.
// runs given function to actually do the update.
func (gv *GridView) ManipAction(act, data string, manip bool, fun func()) {
	es := &gv.EditState
	sv := gv.SVG()
	updt := false
	sv.SetFullReRender()
	actStart := false
	finalAct := false
	if !manip && es.InAction() {
		finalAct = true
	}
	if manip && !es.InAction() {
		manip = false
		actStart = true
		es.ActStart(act, data)
		es.ActUnlock()
	}
	if !manip {
		if !finalAct {
			sv.UndoSave(act, data)
		}
		updt = sv.UpdateStart()
	}
	fun() // actually do the update
	if !manip {
		sv.UpdateEnd(updt)
		if !actStart {
			es.ActDone()
		}
	} else {
		sv.ManipUpdate()
	}
}

// SetStrokeNode sets the stroke properties of Node
// based on previous and current PaintType
func (gv *GridView) SetStrokeNode(sii svg.NodeSVG, prev, pt PaintTypes, sp string) {
	switch pt {
	case PaintLinear:
		svg.UpdateNodeGradientProp(sii, "stroke", false, sp)
	case PaintRadial:
		svg.UpdateNodeGradientProp(sii, "stroke", true, sp)
	default:
		if prev == PaintLinear || prev == PaintRadial {
			pstr := kit.ToString(sii.Prop("stroke"))
			svg.DeleteNodeGradient(sii, pstr)
		}
		sii.SetProp("stroke", sp)
	}
	gv.UpdateMarkerColors(sii)
}

// SetStroke sets the stroke properties of selected items
// based on previous and current PaintType
func (gv *GridView) SetStroke(prev, pt PaintTypes, sp string) {
	es := &gv.EditState
	sv := gv.SVG()
	sv.UndoSave("SetStroke", sp)
	updt := sv.UpdateStart()
	sv.SetFullReRender()
	for itm := range es.Selected {
		gv.SetStrokeNode(itm, prev, pt, sp)
	}
	sv.UpdateEnd(updt)
}

// SetStrokeWidth sets the stroke width property for selected items
// manip means currently being manipulated -- don't save undo.
func (gv *GridView) SetStrokeWidth(wp string, manip bool) {
	es := &gv.EditState
	sv := gv.SVG()
	updt := false
	if !manip {
		sv.UndoSave("SetStrokeWidth", wp)
		updt = sv.UpdateStart()
		sv.SetFullReRender()
	}
	for itm := range es.Selected {
		g := itm.AsSVGNode()
		if !g.Pnt.StrokeStyle.Color.IsNil() {
			g.SetProp("stroke-width", wp)
		}
	}
	if !manip {
		sv.UpdateEnd(updt)
	} else {
		sv.ManipUpdate()
	}
}

// SetStrokeColor sets the stroke color for selected items.
// manip means currently being manipulated -- don't save undo.
func (gv *GridView) SetStrokeColor(sp string, manip bool) {
	es := &gv.EditState
	gv.ManipAction("SetStrokeColor", sp, manip,
		func() {
			for itm := range es.Selected {
				p := itm.Prop("stroke")
				if p != nil {
					itm.SetProp("stroke", sp)
					gv.UpdateMarkerColors(itm)
				}
			}
		})
}

// SetMarkerNode sets the marker properties of Node.
func (gv *GridView) SetMarkerNode(sii svg.NodeSVG, start, mid, end string, sc, mc, ec MarkerColors) {
	sv := gv.SVG()
	MarkerSetProp(&sv.SVG, sii, "marker-start", start, sc)
	MarkerSetProp(&sv.SVG, sii, "marker-mid", mid, mc)
	MarkerSetProp(&sv.SVG, sii, "marker-end", end, ec)
}

// SetMarkerProps sets the marker props
func (gv *GridView) SetMarkerProps(start, mid, end string, sc, mc, ec MarkerColors) {
	es := &gv.EditState
	sv := gv.SVG()
	sv.UndoSave("SetMarkerProps", start+" "+mid+" "+end)
	updt := sv.UpdateStart()
	sv.SetFullReRender()
	for itm := range es.Selected {
		gv.SetMarkerNode(itm, start, mid, end, sc, mc, ec)
	}
	sv.UpdateEnd(updt)
}

// UpdateMarkerColors updates the marker colors, when setting fill or stroke
func (gv *GridView) UpdateMarkerColors(sii svg.NodeSVG) {
	if sii == nil {
		return
	}
	sv := gv.SVG()
	MarkerUpdateColorProp(&sv.SVG, sii, "marker-start")
	MarkerUpdateColorProp(&sv.SVG, sii, "marker-mid")
	MarkerUpdateColorProp(&sv.SVG, sii, "marker-end")
}

// SetDashNode sets the stroke-dasharray property of Node.
// multiplies dash values by the line width in dots.
func (gv *GridView) SetDashNode(sii svg.NodeSVG, dary []float64) {
	if len(dary) == 0 {
		sii.DeleteProp("stroke-dasharray")
		return
	}
	g := sii.AsSVGNode()
	mary := DashMulWidth(float64(g.Pnt.StrokeStyle.Width.Dots), dary)
	ds := DashString(mary)
	sii.SetProp("stroke-dasharray", ds)
}

// SetDashProps sets the dash props
func (gv *GridView) SetDashProps(dary []float64) {
	es := &gv.EditState
	sv := gv.SVG()
	sv.UndoSave("SetDashProps", "")
	updt := sv.UpdateStart()
	sv.SetFullReRender()
	for itm := range es.Selected {
		gv.SetDashNode(itm, dary)
	}
	sv.UpdateEnd(updt)
}

// SetFillNode sets the fill props of given node
// based on previous and current PaintType
func (gv *GridView) SetFillNode(sii svg.NodeSVG, prev, pt PaintTypes, fp string) {
	switch pt {
	case PaintLinear:
		svg.UpdateNodeGradientProp(sii, "fill", false, fp)
	case PaintRadial:
		svg.UpdateNodeGradientProp(sii, "fill", true, fp)
	default:
		if prev == PaintLinear || prev == PaintRadial {
			pstr := kit.ToString(sii.Prop("fill"))
			svg.DeleteNodeGradient(sii, pstr)
		}
		sii.SetProp("fill", fp)
	}
	gv.UpdateMarkerColors(sii)
}

// SetFill sets the fill props of selected items
// based on previous and current PaintType
func (gv *GridView) SetFill(prev, pt PaintTypes, fp string) {
	es := &gv.EditState
	sv := gv.SVG()
	sv.UndoSave("SetFill", fp)
	updt := sv.UpdateStart()
	sv.SetFullReRender()
	for itm := range es.Selected {
		gv.SetFillNode(itm, prev, pt, fp)
	}
	sv.UpdateEnd(updt)
}

// SetFillColor sets the fill color for selected items
// manip means currently being manipulated -- don't save undo.
func (gv *GridView) SetFillColor(fp string, manip bool) {
	es := &gv.EditState
	gv.ManipAction("SetFillColor", fp, manip,
		func() {
			for itm := range es.Selected {
				p := itm.Prop("fill")
				if p != nil {
					itm.SetProp("fill", fp)
					gv.UpdateMarkerColors(itm)
				}
			}
		})
}

// DefaultGradient returns the default gradient to use for setting stops
func (gv *GridView) DefaultGradient() string {
	es := &gv.EditState
	sv := gv.SVG()
	if len(gv.EditState.Gradients) == 0 {
		es.ConfigDefaultGradient()
		sv.UpdateGradients(es.Gradients)
	}
	return es.Gradients[0].Name
}

// UpdateGradients updates gradients from EditState
func (gv *GridView) UpdateGradients() {
	es := &gv.EditState
	sv := gv.SVG()
	updt := sv.UpdateStart()
	sv.UpdateGradients(es.Gradients)
	sv.UpdateEnd(updt)
}

// IsCurLayer returns true if given layer is the current layer
// for creating items
func (gv *GridView) IsCurLayer(lay string) bool {
	return gv.EditState.CurLayer == lay
}

// SetCurLayer sets the current layer for creating items to given one
func (gv *GridView) SetCurLayer(lay string) {
	gv.EditState.CurLayer = lay
	gv.SetStatus("set current layer to: " + lay)
}

// ClearCurLayer clears the current layer for creating items if it
// was set to the given layer name
func (gv *GridView) ClearCurLayer(lay string) {
	if gv.EditState.CurLayer == lay {
		gv.EditState.CurLayer = ""
		gv.SetStatus("clear current layer from: " + lay)
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
	es.SelectAction(sii, mode)
	sv.UpdateView(false)
}

// SelectNodeInTree selects given node in TreeView
func (gv *GridView) SelectNodeInTree(kn ki.Ki, mode mouse.SelectModes) {
	tv := gv.TreeView()
	tvn := tv.FindSrcNode(kn)
	if tvn != nil {
		tvn.SelectAction(mode)
	}
}

// SelectedAsTreeViews returns the currently-selected items from SVG as TreeView nodes
func (gv *GridView) SelectedAsTreeViews() []*giv.TreeView {
	es := &gv.EditState
	sl := es.SelectedList(false)
	if len(sl) == 0 {
		return nil
	}
	tv := gv.TreeView()
	var tvl []*giv.TreeView
	for _, si := range sl {
		tvn := tv.FindSrcNode(si.This())
		if tvn != nil {
			tvl = append(tvl, tvn)
		}
	}
	return tvl
}

// DuplicateSelected duplicates selected items in SVG view, using TreeView methods
func (gv *GridView) DuplicateSelected() {
	tvl := gv.SelectedAsTreeViews()
	if len(tvl) == 0 {
		gv.SetStatus("Duplicate: no tree items found")
		return
	}
	sv := gv.SVG()
	sv.UndoSave("DuplicateSelected", "")
	updt := sv.UpdateStart()
	sv.SetFullReRender()
	tv := gv.TreeView()
	tvupdt := tv.UpdateStart()
	tv.SetFullReRender()
	for _, tvi := range tvl {
		tvi.SrcDuplicate()
	}
	gv.SetStatus("Duplicated selected items")
	tv.UpdateEnd(tvupdt)
	sv.UpdateEnd(updt)
}

// CopySelected copies selected items in SVG view, using TreeView methods
func (gv *GridView) CopySelected() {
	tvl := gv.SelectedAsTreeViews()
	if len(tvl) == 0 {
		gv.SetStatus("Copy: no tree items found")
		return
	}
	tv := gv.TreeView()
	tv.SetSelectedViews(tvl)
	tvl[0].Copy(true) // operates on first element in selection
	gv.SetStatus("Copied selected items")
}

// CutSelected cuts selected items in SVG view, using TreeView methods
func (gv *GridView) CutSelected() {
	tvl := gv.SelectedAsTreeViews()
	if len(tvl) == 0 {
		gv.SetStatus("Cut: no tree items found")
		return
	}
	sv := gv.SVG()
	sv.UndoSave("CutSelected", "")
	updt := sv.UpdateStart()
	sv.SetFullReRender()
	tv := gv.TreeView()
	tvupdt := tv.UpdateStart()
	tv.SetFullReRender()
	tv.SetSelectedViews(tvl)
	tvl[0].Cut() // operates on first element in selection
	gv.SetStatus("Cut selected items")
	tv.UpdateEnd(tvupdt)
	sv.UpdateEnd(updt)
}

// PasteClip pastes clipboard, using cur layer etc
func (gv *GridView) PasteClip() {
	md := oswin.TheApp.ClipBoard(gv.ParentWindow().OSWin).Read([]string{filecat.DataJson})
	if md == nil {
		return
	}
	es := &gv.EditState
	sv := gv.SVG()
	sv.UndoSave("Paste", "")
	updt := sv.UpdateStart()
	sv.SetFullReRender()
	tv := gv.TreeView()
	tvupdt := tv.UpdateStart()
	tv.SetFullReRender()
	par := tv
	if es.CurLayer != "" {
		ly := tv.ChildByName("tv_"+es.CurLayer, 1)
		if ly != nil {
			par = ly.Embed(KiT_TreeView).(*TreeView)
		}
	}
	par.PasteChildren(md, dnd.DropCopy)
	gv.SetStatus("Pasted items from clipboard")
	tv.UpdateEnd(tvupdt)
	sv.UpdateEnd(updt)
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
	gv.UpdateTabs()
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
	gv.UpdateTabs()
	return act
}

/////////////////////////////////////////////////////////////////////////
//   Basic infrastructure

func (gv *GridView) EditPrefs() {
	giv.StructViewDialog(gv.Viewport, &gv.Prefs, giv.DlgOpts{Title: "Grid Prefs"}, nil, nil)
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
		gv.Prefs.SplitName = split
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
	sv := gv.SplitView()
	AvailSplits.Add(name, desc, sv.Splits)
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

/////////////////////////////////////////////////////////////////////////
//   Props, MainMenu

var GridViewProps = ki.Props{
	"EnumType:Flag":    gi.KiT_VpFlags,
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
				"desc":     "Create a new drawing using current default preferences.",
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
			{"sep-af", ki.BlankProp{}},
			{"EditPrefs", ki.Props{
				"label": "Drawing Prefs...",
			}},
			{"sep-close", ki.BlankProp{}},
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
