// Copyright (c) 2021, The Grid Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grid

import (
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

func (gr *GridView) Defaults() {
	gr.Prefs.Defaults()
	// gr.Prefs = Prefs
}

// OpenDrawing opens a new .svg drawing
func (gr *GridView) OpenDrawing(fnm gi.FileName) error {
	path, _ := filepath.Abs(string(fnm))
	gr.FilePath = gi.FileName(path)
	// TheFile.SetText(CurFilename)
	sg := gr.SVG()
	err := sg.OpenXML(path)
	if err != nil && err != io.EOF {
		log.Println(err)
		// return err
	}
	sg.SetNormXForm()
	scx, scy := sg.Pnt.XForm.ExtractScale()
	sg.Scale = 0.5 * (scx + scy)
	sg.Trans.Set(0, 0)
	sg.SetTransform()
	tv := gr.TreeView()
	tv.CloseAll()
	tv.Open()
	return nil
}

// NewDrawing opens a new drawing window
func (gr *GridView) NewDrawing() *GridView {
	_, ngr := NewGridWindow("")
	return ngr
}

// SaveDrawing saves .svg drawing to current filename
func (gr *GridView) SaveDrawing() error {
	fp, fn := filepath.Split(string(gr.FilePath))
	fn = "tmp_" + fn
	fp = filepath.Join(fp, fn)
	sg := gr.SVG()
	err := sg.SaveXML(fp)
	if err != nil && err != io.EOF {
		log.Println(err)
		// return err
	}
	return err
}

// SetTool sets the current active tool
func (gr *GridView) SetTool(tl Tools) {
	gr.EditState.Tool = tl
}

func (gr *GridView) MainToolbar() *gi.ToolBar {
	return gr.ChildByName("main-tb", 0).(*gi.ToolBar)
}

func (gr *GridView) ModalToolbar() *gi.ToolBar {
	return gr.ChildByName("modal-tb", 1).(*gi.ToolBar)
}

func (gr *GridView) HBox() *gi.Layout {
	return gr.ChildByName("hbox", 2).(*gi.Layout)
}

func (gr *GridView) Tools() *gi.ToolBar {
	return gr.HBox().ChildByName("tools", 0).(*gi.ToolBar)
}

func (gr *GridView) SplitView() *gi.SplitView {
	return gr.HBox().ChildByName("splitview", 1).(*gi.SplitView)
}

func (gr *GridView) TreeView() *giv.TreeView {
	return gr.SplitView().ChildByName("tree-frame", 0).Child(0).(*giv.TreeView) // note: name changes
}

func (gr *GridView) SVG() *SVGView {
	return gr.SplitView().Child(1).(*SVGView)
}

func (gr *GridView) Tabs() *gi.TabView {
	return gr.SplitView().ChildByName("tabs", 2).(*gi.TabView)
}

// Config configures entire view -- only runs if no children yet
func (gr *GridView) Config() {
	if gr.HasChildren() {
		return
	}
	updt := gr.UpdateStart()
	gr.Lay = gi.LayoutVert
	gr.SetProp("spacing", gi.StdDialogVSpaceUnits)
	gi.AddNewToolBar(gr, "main-tb")
	gi.AddNewToolBar(gr, "modal-tb")
	hb := gi.AddNewLayout(gr, "hbox", gi.LayoutHoriz)
	hb.SetStretchMax()

	tb := gi.AddNewToolBar(hb, "tools")
	tb.Lay = gi.LayoutVert
	sv := gi.AddNewSplitView(hb, "splitview")
	sv.Dim = mat32.X

	tvfr := gi.AddNewFrame(sv, "tree-frame", gi.LayoutHoriz)
	tvfr.SetStretchMax()
	tvfr.SetReRenderAnchor()
	tv := giv.AddNewTreeView(tvfr, "treeview")
	tv.OpenDepth = 1

	sg := AddNewSVGView(sv, "svg", gr)

	tab := gi.AddNewTabView(sv, "tabs")
	tab.SetStretchMaxWidth()

	tv.SetRootNode(sg)

	tv.TreeViewSig.Connect(gr.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
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

	gr.ConfigMainToolbar()
	gr.ConfigModalToolbar()
	gr.ConfigTools()
	gr.ConfigTabs()

	gr.UpdateEnd(updt)
}

// IsConfiged returns true if the view is fully configured
func (gr *GridView) IsConfiged() bool {
	if !gr.HasChildren() {
		return false
	}
	return true
}

func (gr *GridView) ConfigMainToolbar() {
	tb := gr.MainToolbar()
	tb.SetStretchMaxWidth()
	tb.AddAction(gi.ActOpts{Label: "New", Icon: "new", Tooltip: "create new drawing using default drawing preferences"},
		gr.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			grr.NewDrawing()
		})
	tb.AddAction(gi.ActOpts{Label: "Open", Icon: "file-open", Tooltip: "Open a drawing from .svg file"},
		gr.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			giv.CallMethod(grr, "OpenDrawing", grr.ViewportSafe())
		})
	tb.AddAction(gi.ActOpts{Label: "Save", Icon: "file-save", Tooltip: "Save drawing to .svg file, using current filename"},
		gr.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			grr.SaveDrawing()
		})
}

func (gr *GridView) ConfigModalToolbar() {
	tb := gr.ModalToolbar()
	tb.SetStretchMaxWidth()
}

func (gr *GridView) ConfigTools() {
	tb := gr.Tools()
	tb.Lay = gi.LayoutVert
	tb.SetStretchMaxHeight()
	tb.AddAction(gi.ActOpts{Icon: "arrow", Tooltip: "S, Space: select, move, resize objects"},
		gr.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			grr.SetTool(SelectTool)
		})
	tb.AddAction(gi.ActOpts{Icon: "arrow", Tooltip: "N: select, move node points within paths"},
		gr.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			grr.SetTool(NodeTool)
		})
	tb.AddAction(gi.ActOpts{Icon: "plus", Tooltip: "R: create rectangles and squares"},
		gr.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			grr.SetTool(RectTool)
		})
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

// NewGridWindow returns a new GridWindow loading given file if non-empty
func NewGridWindow(fnm string) (*gi.Window, *GridView) {
	path, _ := filepath.Abs(fnm)
	dfnm := giv.DirAndFile(path)
	winm := "grid-" + dfnm
	wintitle := winm + ": " + path

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
	sv := gv.SVG()
	tv := gv.Tabs()
	tv.NoDeleteTabs = true
	pv := gv.RecycleTab("Paint", KiT_PaintView, false).(*PaintView)
	pv.Config(gv)
	stv := gv.RecycleTab("Obj", giv.KiT_StructView, false).(*giv.StructView)
	stv.SetStruct(sv)
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
