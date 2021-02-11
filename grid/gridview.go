// Copyright (c) 2021, The Grid Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grid

import (
	"errors"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/gist"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/gi/svg"
	"github.com/goki/gi/units"
	"github.com/goki/gide/gide"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
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
	// sg.GatherIds() // also ensures uniqueness, key for json saving
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
	sg := gv.SVG()
	err := sg.SaveXML(path)
	if err != nil && err != io.EOF {
		log.Println(err)
	}
	gv.SetStatus("Saved: " + path)
	return err
}

// SetTool sets the current active tool
func (gv *GridView) SetTool(tl Tools) {
	gv.EditState.Tool = tl
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

func (gv *GridView) TreeView() *giv.TreeView {
	return gv.SplitView().ChildByName("tree-frame", 0).Child(0).(*giv.TreeView) // note: name changes
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
	tv := giv.AddNewTreeView(tvfr, "treeview")
	tv.OpenDepth = 1

	sg := AddNewSVGView(sv, "svg", gv)

	tab := gi.AddNewTabView(sv, "tabs")
	tab.SetStretchMaxWidth()

	tv.SetRootNode(sg)

	tv.TreeViewSig.Connect(gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if data == nil || sig != int64(giv.TreeViewSelected) {
			return
		}
		tvn, _ := data.(ki.Ki).Embed(giv.KiT_TreeView).(*giv.TreeView)
		_, isgp := tvn.SrcNode.(*svg.Group)
		if !isgp {
			ggv, _ := recv.Embed(KiT_GridView).(*GridView)
			stv := ggv.RecycleTab("Obj", giv.KiT_StructView, true).(*giv.StructView)
			stv.SetStruct(tvn.SrcNode)
		}
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
	es := gv.EditState
	act.SetInactiveState(!es.UndoMgr.HasUndoAvail())
}

// RedoAvailFunc is an ActionUpdateFunc that inactivates action if no more redos
func (gv *GridView) RedoAvailFunc(act *gi.Action) {
	es := gv.EditState
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
	tb.AddAction(gi.ActOpts{Label: "Undo", Icon: "undo", Tooltip: "Undo last action", UpdateFunc: gv.UndoAvailFunc},
		gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			grr.Undo()
		})
	tb.AddAction(gi.ActOpts{Label: "Redo", Icon: "redo", Tooltip: "Redo last undo action", UpdateFunc: gv.RedoAvailFunc},
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
	tb.AddAction(gi.ActOpts{Icon: "arrow", Tooltip: "S, Space: select, move, resize objects"},
		gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			grr.SetTool(SelectTool)
		})
	tb.AddAction(gi.ActOpts{Icon: "arrow", Tooltip: "N: select, move node points within paths"},
		gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			grr.SetTool(NodeTool)
		})
	tb.AddAction(gi.ActOpts{Icon: "plus", Tooltip: "R: create rectangles and squares"},
		gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			grr.SetTool(RectTool)
		})
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
	lbl.SetText(msg)
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
// main windows and look for gide windows and call their CloseWindowReq
// functions!
func QuitReq() bool {
	for _, win := range gi.MainWindows {
		if !strings.HasPrefix(win.Nm, "gide-") {
			continue
		}
		mfr, err := win.MainWidget()
		if err != nil {
			continue
		}
		gek := mfr.ChildByName("gide", 0)
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
	gv.RecycleTab("Obj", giv.KiT_StructView, false)
}

func (gv *GridView) UpdateTabs() {
	// fmt.Printf("updt-tabs\n")
	es := gv.EditState
	sls := es.SelectedList(false)
	if len(sls) > 0 {
		pnt := &(sls[0].AsSVGNode().Pnt)
		pv := gv.Tab("Paint").(*PaintView)
		pv.Update(pnt)
	}
}

/////////////////////////////////////////////////////////////////////////
//  Actions

// SetStrokeOn sets the stroke on or not
func (gv *GridView) SetStrokeOn(on bool, clr gist.Color) {
	es := gv.EditState
	sv := gv.SVG()
	updt := sv.UpdateStart()
	sv.SetFullReRender()
	for itm := range es.Selected {
		g := itm.AsSVGNode()
		hp := g.Prop("stroke")
		if hp == nil {
			if !on {
				g.SetProp("stroke", "none")
			} else {
				g.SetProp("stroke", clr.HexString())
			}
		} else {
			if !on {
				g.SetProp("stroke", "none")
			} else {
				if hps, ok := hp.(string); ok {
					if hps == "none" {
						g.SetProp("stroke", clr.HexString())
					}
				}
			}
		}
	}
	sv.UpdateEnd(updt)
}

// SetStrokeWidth sets the stroke width for selected items
func (gv *GridView) SetStrokeWidth(wd float32) { // todo: add units
	es := gv.EditState
	sv := gv.SVG()
	updt := sv.UpdateStart()
	sv.SetFullReRender()
	for itm := range es.Selected {
		g := itm.AsSVGNode()
		if !g.Pnt.StrokeStyle.Color.IsNil() {
			g.SetProp("stroke-width", fmt.Sprintf("%gpx", wd))
		}
	}
	sv.UpdateEnd(updt)
}

// SetStrokeColor sets the stroke color for selected items
func (gv *GridView) SetStrokeColor(clr gist.Color) {
	es := gv.EditState
	sv := gv.SVG()
	updt := sv.UpdateStart()
	sv.SetFullReRender()
	for itm := range es.Selected {
		g := itm.AsSVGNode()
		if !g.Pnt.StrokeStyle.Color.IsNil() {
			g.SetProp("stroke", clr.HexString())
		}
	}
	sv.UpdateEnd(updt)
}

// SetFillOn sets the stroke on or not
func (gv *GridView) SetFillOn(on bool, clr gist.Color) {
	es := gv.EditState
	sv := gv.SVG()
	updt := sv.UpdateStart()
	sv.SetFullReRender()
	for itm := range es.Selected {
		g := itm.AsSVGNode()
		hp := g.Prop("fill")
		if hp == nil {
			if !on {
				g.SetProp("fill", "none")
			} else {
				g.SetProp("fill", clr.HexString())
			}
		} else {
			if !on {
				g.SetProp("fill", "none")
			} else {
				if hps, ok := hp.(string); ok {
					if hps == "none" {
						g.SetProp("fill", clr.HexString())
					}
				}
			}
		}
	}
	sv.UpdateEnd(updt)
}

// SetFillColor sets the fill color for selected items
func (gv *GridView) SetFillColor(clr gist.Color) {
	es := gv.EditState
	sv := gv.SVG()
	updt := sv.UpdateStart()
	sv.SetFullReRender()
	for itm := range es.Selected {
		g := itm.AsSVGNode()
		if !g.Pnt.FillStyle.Color.IsNil() {
			g.SetProp("fill", clr.HexString())
		}
	}
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
	return act
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
				// "submenu": &gide.SavedPaths,
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
			{"Paste History...", ki.Props{
				"keyfun": gi.KeyFunPasteHist,
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
			{"Panels", ki.PropSlice{
				{"FocusNextPanel", ki.Props{
					"label": "Focus Next",
					"shortcut-func": giv.ShortcutFunc(func(gei interface{}, act *gi.Action) key.Chord {
						return key.Chord(gide.ChordForFun(gide.KeyFunNextPanel).String())
					}),
				}},
				{"FocusPrevPanel", ki.Props{
					"label": "Focus Prev",
					"shortcut-func": giv.ShortcutFunc(func(gei interface{}, act *gi.Action) key.Chord {
						return key.Chord(gide.ChordForFun(gide.KeyFunPrevPanel).String())
					}),
				}},
				{"CloneActiveView", ki.Props{
					"label": "Clone Active",
					"shortcut-func": giv.ShortcutFunc(func(gei interface{}, act *gi.Action) key.Chord {
						return key.Chord(gide.ChordForFun(gide.KeyFunBufClone).String())
					}),
				}},
			}},
			{"Splits", ki.PropSlice{
				{"SplitsSetView", ki.Props{
					"label":   "Set View",
					"submenu": &gide.AvailSplitNames,
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
					"submenu": &gide.AvailSplitNames,
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
