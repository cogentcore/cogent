// Copyright (c) 2021, The Grid Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grid

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/gist"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/cursor"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/gi/oswin/mouse"
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
	FilePath      gi.FileName `ext:".svg" desc:"full path to current drawing filename"`
	Tool          Tools       `desc:"current tool in use"`
	Prefs         Preferences `desc:"current drawing preferences"`
	Trans         mat32.Vec2  `desc:"view translation offset (from dragging)"`
	Scale         float32     `desc:"view scaling (from zooming)"`
	SetDragCursor bool        `view:"-" desc:"has dragging cursor been set yet?"`
}

var KiT_GridView = kit.Types.AddType(&GridView{}, GridViewProps)

// AddNewGridView adds a new editor to given parent node, with given name.
func AddNewGridView(parent ki.Ki, name string) *GridView {
	return parent.AddNewChild(KiT_GridView, name).(*GridView)
}

func (g *GridView) CopyFieldsFrom(frm interface{}) {
	fr := frm.(*GridView)
	g.Frame.CopyFieldsFrom(&fr.Frame)
	// todo: fill out
	// g.Trans = fr.Trans
	// g.Scale = fr.Scale
	// g.SetDragCursor = fr.SetDragCursor
}

// OpenDrawing opens a new .svg drawing
func (gr *GridView) OpenDrawing(fnm gi.FileName) error {
	path, _ := filepath.Abs(string(fnm))
	gr.FilePath = gi.FileName(path)
	// TheFile.SetText(CurFilename)
	sg := gr.SVG()
	updt := gr.UpdateStart()
	gr.SetFullReRender()
	err := sg.OpenXML(path)
	if err != nil {
		log.Println(err)
		return err
	}
	// SetZoom(TheSVG.ParentWindow().LogicalDPI() / 96.0)
	// SetTrans(0, 0)
	tv := gr.TreeView()
	tv.SetRootNode(sg)
	gr.UpdateEnd(updt)
	return nil
}

// NewDrawing opens a new drawing window
func (gr *GridView) NewDrawing() *GridView {
	return nil
}

// SaveDrawing saves .svg drawing to current filename
func (gr *GridView) SaveDrawing() error {
	return nil
}

// SetTool sets the current active tool
func (gr *GridView) SetTool(tl Tools) {
	gr.Tool = tl
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
	return gr.SplitView().ChildByName("tree-frame", 0).ChildByName("treeview", 0).(*giv.TreeView)
}

func (gr *GridView) SVG() *svg.SVG {
	return gr.SplitView().Child(1).(*svg.SVG)
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
	tv := giv.AddNewTreeView(tvfr, "treeview")

	sg := svg.AddNewSVG(sv, "svg")
	// sg.Scale = 1
	sg.Fill = true
	sg.SetProp("background-color", "white")
	sg.SetProp("width", units.NewPx(480))
	sg.SetProp("height", units.NewPx(240))
	sg.SetStretchMaxWidth()
	sg.SetStretchMaxHeight()

	tab := gi.AddNewTabView(sv, "tabs")
	tab.SetStretchMaxWidth()

	tv.SetRootNode(sg)

	sv.SetSplits(0.1, 0.7, 0.2)

	gr.ConfigMainToolbar()
	gr.ConfigModalToolbar()
	gr.ConfigTools()

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

// GridViewEvents handles svg editing events
func (gv *GridView) GridViewEvents() {
	gv.ConnectEvent(oswin.MouseDragEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.DragEvent)
		me.SetProcessed()
		ssvg := recv.Embed(KiT_GridView).(*GridView)
		if ssvg.IsDragging() {
			if !ssvg.SetDragCursor {
				oswin.TheApp.Cursor(ssvg.ParentWindow().OSWin).Push(cursor.HandOpen)
				ssvg.SetDragCursor = true
			}
			del := me.Where.Sub(me.From)
			ssvg.Trans.X += float32(del.X)
			ssvg.Trans.Y += float32(del.Y)
			ssvg.SetTransform()
			ssvg.SetFullReRender()
			ssvg.UpdateSig()
		} else {
			if ssvg.SetDragCursor {
				oswin.TheApp.Cursor(ssvg.ParentWindow().OSWin).Pop()
				ssvg.SetDragCursor = false
			}
		}

	})
	gv.ConnectEvent(oswin.MouseScrollEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.ScrollEvent)
		me.SetProcessed()
		ssvg := recv.Embed(KiT_GridView).(*GridView)
		if ssvg.SetDragCursor {
			oswin.TheApp.Cursor(ssvg.ParentWindow().OSWin).Pop()
			ssvg.SetDragCursor = false
		}
		ssvg.InitScale()
		ssvg.Scale += float32(me.NonZeroDelta(false)) / 20
		if ssvg.Scale <= 0 {
			ssvg.Scale = 0.01
		}
		ssvg.SetTransform()
		ssvg.SetFullReRender()
		ssvg.UpdateSig()
	})
	gv.ConnectEvent(oswin.MouseEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.Event)
		ssvg := recv.Embed(KiT_GridView).(*GridView)
		if ssvg.SetDragCursor {
			oswin.TheApp.Cursor(ssvg.ParentWindow().OSWin).Pop()
			ssvg.SetDragCursor = false
		}
		obj := ssvg.FirstContainingPoint(me.Where, true)
		if me.Action == mouse.Release && me.Button == mouse.Right {
			me.SetProcessed()
			if obj != nil {
				giv.StructViewDialog(ssvg.Viewport, obj, giv.DlgOpts{Title: "SVG Element View"}, nil, nil)
			}
		}
	})
	gv.ConnectEvent(oswin.MouseHoverEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.HoverEvent)
		me.SetProcessed()
		ssvg := recv.Embed(KiT_GridView).(*GridView)
		obj := ssvg.FirstContainingPoint(me.Where, true)
		if obj != nil {
			pos := me.Where
			ttxt := fmt.Sprintf("element name: %v -- use right mouse click to edit", obj.Name())
			gi.PopupTooltip(obj.Name(), pos.X, pos.Y, gv.ViewportSafe(), ttxt)
		}
	})
}

func (gv *GridView) ConnectEvents2D() {
	gv.GridViewEvents()
}

// InitScale ensures that Scale is initialized and non-zero
func (gv *GridView) InitScale() {
	if gv.Scale == 0 {
		mvp := gv.ViewportSafe()
		if mvp != nil {
			gv.Scale = gv.ParentWindow().LogicalDPI() / 96.0
		} else {
			gv.Scale = 1
		}
	}
}

// SetTransform sets the transform based on Trans and Scale values
func (gv *GridView) SetTransform() {
	gv.InitScale()
	gv.SetProp("transform", fmt.Sprintf("translate(%v,%v) scale(%v,%v)", gv.Trans.X, gv.Trans.Y, gv.Scale, gv.Scale))
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
			{"OpenConsoleTab", ki.Props{}},
		}},
		{"Command", ki.PropSlice{
			{"Build", ki.Props{
				"shortcut-func": giv.ShortcutFunc(func(gei interface{}, act *gi.Action) key.Chord {
					return key.Chord(gide.ChordForFun(gide.KeyFunBuildProj).String())
				}),
			}},
			{"Run", ki.Props{
				"shortcut-func": giv.ShortcutFunc(func(gei interface{}, act *gi.Action) key.Chord {
					return key.Chord(gide.ChordForFun(gide.KeyFunRunProj).String())
				}),
			}},
		}},
		{"Window", "Windows"},
		{"Help", ki.PropSlice{
			{"HelpWiki", ki.Props{}},
		}},
	},
}
