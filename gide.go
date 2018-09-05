// Copyright (c) 2018, The gide / GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
package gide provides the core Gide editor object.

Derived classes can extend the functionality for specific domains.

*/
package gide

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/goki/gi"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
)

// Gide is the core editor and tab viewer framework for the Gide system.  The
// default view has a tree browser of files on the left, editor panels in the
// middle, and a tabbed viewer on the right.
type Gide struct {
	gi.Frame
	ProjFilename      gi.FileName `desc:"current project filename for saving / loading specific Gide configuration information in a .gide file (optional)"`
	ProjRoot          gi.FileName `desc:"root directory for the project -- all projects must be organized within a top-level root directory, with all the files therein constituting the scope of the project -- by default it is the path for ProjFilename"`
	ActiveFilename    gi.FileName `desc:"filename of the currently-active textview"`
	Changed           bool        `json:"-" desc:"has the root changed?  we receive update signals from root for changes"`
	Files             FileNode    `desc:"all the files in the project directory and subdirectories"`
	NTextViews        int         `xml:"n-text-views" desc:"number of textviews available for editing files (default 2) -- configurable with n-text-views property"`
	ActiveTextViewIdx int         `json:"-" desc:"index of the currently-active textview -- new files will be viewed in other views if available"`
}

var KiT_Gide = kit.Types.AddType(&Gide{}, GideProps)

// UpdateFiles updates the list of files saved in project
func (ge *Gide) UpdateFiles() {
	ge.Files.OpenPath(string(ge.ProjRoot))
}

// IsEmpty returns true if given Gide project is empty -- has not been set to a valide path
func (ge *Gide) IsEmpty() bool {
	return ge.ProjRoot == ""
}

// NewProj opens a new pproject at given path, which can either be a specific
// file or a directory containing multiple files of interest -- opens in
// current Gide object if it is empty, or otherwise opens a new window.
func (ge *Gide) NewProj(path gi.FileName) {
	if !ge.IsEmpty() {
		NewGideProj(string(path))
		return
	}
	ge.Defaults()
	root, pnm, fnm, ok := ProjPathParse(string(path))
	if ok {
		ge.ProjRoot = gi.FileName(root)
		ge.SetName(pnm)
		ge.ProjFilename = gi.FileName(pnm + ".gide") // default filename
		win := ge.ParentWindow()
		if win != nil {
			winm := "gide-" + pnm
			win.SetName(winm)
			win.OSWin.SetName(winm)
			win.OSWin.SetTitle(winm)
		}
		ge.UpdateProj()
		if fnm != "" {
			ge.ViewFile(fnm)
		}
	}
}

// SaveProj saves project file containing custom project settings, in a
// standard JSON-formatted file
func (ge *Gide) SaveProj() {
	if ge.ProjFilename == "" {
		return
	}
	ge.SaveJSON(string(ge.ProjFilename))
	ge.Changed = false
}

// SaveProjAs saves project custom settings to given filename, in a standard
// JSON-formatted file
func (ge *Gide) SaveProjAs(filename gi.FileName) {
	ge.SaveJSON(string(filename))
	ge.Changed = false
	ge.ProjFilename = filename
	ge.UpdateSig() // notify our editor
}

// OpenProj opens project and its settings from given filename, in a standard
// JSON-formatted file
func (ge *Gide) OpenProj(filename gi.FileName) {
	ge.OpenJSON(string(filename))
	ge.ProjFilename = filename // should already be set but..
	ge.UpdateProj()
	ge.SetFullReRender()
	ge.UpdateSig() // notify our editor
}

// UpdateProj does full update to current proj
func (ge *Gide) UpdateProj() {
	mods, updt := ge.StdConfig()
	ge.SetTitle(fmt.Sprintf("Gide of: %v", ge.ProjRoot)) // todo: get rid of title
	ge.UpdateFiles()
	ge.ConfigSplitView()
	ge.ConfigToolbar()
	if mods {
		ge.UpdateEnd(updt)
	}
}

// ProjPathParse parses given project path into a root directory (which could
// be the path or just the directory portion of the path, depending in whether
// the path is a directory or not), and a bool if all is good (otherwise error
// message has been reported). projnm is always the last directory of the path.
func ProjPathParse(path string) (root, projnm, fnm string, ok bool) {
	info, err := os.Lstat(path)
	if err != nil {
		emsg := fmt.Errorf("gide.ProjPathParse: Cannot open at given path: %q: Error: %v", path, err)
		log.Println(emsg)
		return
	}
	dir, fn := filepath.Split(path)
	pathIsDir := info.IsDir()
	if pathIsDir {
		root = path
	} else {
		root = dir
		fnm = fn
	}
	_, projnm = filepath.Split(root)
	ok = true
	return
}

//////////////////////////////////////////////////////////////////////////////////////
//   TextViews

// ActiveTextView returns the currently-active TextView
func (ge *Gide) ActiveTextView() *giv.TextView {
	return ge.TextViewByIndex(ge.ActiveTextViewIdx)
}

// SetActiveTextView sets the given view index as the currently-active
// TextView -- returns that textview
func (ge *Gide) SetActiveTextView(idx int) *giv.TextView {
	if idx < 0 || idx >= ge.NTextViews {
		log.Printf("Gide SetActiveTextView: text view index out of range: %v\n", idx)
		return nil
	}
	ge.ActiveTextViewIdx = idx
	av := ge.ActiveTextView()
	if av.Buf != nil {
		ge.ActiveFilename = av.Buf.Filename
	}
	av.GrabFocus()
	return av
}

// NextTextView returns the next text view available for viewing a file and
// its index -- if the active text view is empty, then it is used, otherwise
// it is the next one
func (ge *Gide) NextTextView() (*giv.TextView, int) {
	av := ge.TextViewByIndex(ge.ActiveTextViewIdx)
	if av.Buf == nil {
		return av, ge.ActiveTextViewIdx
	}
	nxt := (ge.ActiveTextViewIdx + 1) % ge.NTextViews
	return ge.TextViewByIndex(nxt), nxt
}

// SaveActiveView saves the contents of the currently-active textview
func (ge *Gide) SaveActiveView() {
	tv := ge.ActiveTextView()
	if tv.Buf != nil {
		tv.Buf.Save() // todo: errs..
	}
}

// SaveActiveViewAs save with specified filename the contents of the
// currently-active textview
func (ge *Gide) SaveActiveViewAs(filename gi.FileName) {
	tv := ge.ActiveTextView()
	if tv.Buf != nil {
		tv.Buf.SaveAs(filename)
	}
}

// ViewFileNode sets the next text view to view file in given node (opens
// buffer if not already opened)
func (ge *Gide) ViewFileNode(fn *FileNode) {
	if err := fn.OpenBuf(); err == nil {
		nv, nidx := ge.NextTextView()
		if nv.Buf != nil && nv.Buf.Edited { // todo: save current changes?
			fmt.Printf("Changes not saved in file: %v before switching view there to new file\n", nv.Buf.Filename)
		}
		nv.SetBuf(fn.Buf)
		ge.SetActiveTextView(nidx)
	}
}

// ViewFile sets the next text view to view given file name -- include as much
// of name as possible to disambiguate -- will use the first matching --
// returns false if not found
func (ge *Gide) ViewFile(fnm string) bool {
	fn, ok := ge.Files.FindFile(fnm)
	if !ok {
		return false
	}
	ge.ViewFileNode(fn)
	return true
}

//////////////////////////////////////////////////////////////////////////////////////
//    Defaults, Prefs

// todo: add a prefs obj with everything

func (ge *Gide) Defaults() {
	ge.NTextViews = 2
}

//////////////////////////////////////////////////////////////////////////////////////
//   GUI configs

// StdFrameConfig returns a TypeAndNameList for configuring a standard Frame
// -- can modify as desired before calling ConfigChildren on Frame using this
func (ge *Gide) StdFrameConfig() kit.TypeAndNameList {
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_Label, "title")
	config.Add(gi.KiT_ToolBar, "toolbar")
	config.Add(gi.KiT_SplitView, "splitview")
	return config
}

// StdConfig configures a standard setup of the overall Frame -- returns mods,
// updt from ConfigChildren and does NOT call UpdateEnd
func (ge *Gide) StdConfig() (mods, updt bool) {
	ge.Lay = gi.LayoutVert
	ge.SetProp("spacing", gi.StdDialogVSpaceUnits)
	config := ge.StdFrameConfig()
	mods, updt = ge.ConfigChildren(config, false)
	return
}

// SetTitle sets the optional title and updates the Title label
func (ge *Gide) SetTitle(title string) {
	lab, _ := ge.TitleWidget()
	if lab != nil {
		lab.Text = title
	}
}

// Title returns the title label widget, and its index, within frame -- nil,
// -1 if not found
func (ge *Gide) TitleWidget() (*gi.Label, int) {
	idx, ok := ge.Children().IndexByName("title", 0)
	if !ok {
		return nil, -1
	}
	return ge.KnownChild(idx).(*gi.Label), idx
}

// SplitView returns the main SplitView
func (ge *Gide) SplitView() (*gi.SplitView, int) {
	idx, ok := ge.Children().IndexByName("splitview", 2)
	if !ok {
		return nil, -1
	}
	return ge.KnownChild(idx).(*gi.SplitView), idx
}

// FileTree returns the main FileTree
func (ge *Gide) FileTree() *giv.TreeView {
	split, _ := ge.SplitView()
	if split != nil {
		tv := split.KnownChild(0).KnownChild(0).(*giv.TreeView)
		return tv
	}
	return nil
}

// TextViewByIndex returns the TextView by index, nil if not found
func (ge *Gide) TextViewByIndex(idx int) *giv.TextView {
	if idx < 0 || idx >= ge.NTextViews {
		log.Printf("Gide: text view index out of range: %v\n", idx)
		return nil
	}
	split, _ := ge.SplitView()
	stidx := 1 // 0 = file browser -- could be collapsed but always there.
	if split != nil {
		svk := split.KnownChild(stidx + idx).KnownChild(0)
		if !svk.TypeEmbeds(giv.KiT_TextView) {
			log.Printf("Gide: text view not at index: %v\n", idx)
			return nil
		}
		return svk.(*giv.TextView)
	}
	return nil
}

// ToolBar returns the toolbar widget
func (ge *Gide) ToolBar() *gi.ToolBar {
	idx, ok := ge.Children().IndexByName("toolbar", 1)
	if !ok {
		return nil
	}
	return ge.KnownChild(idx).(*gi.ToolBar)
}

// ConfigToolbar adds a Gide toolbar.
func (ge *Gide) ConfigToolbar() {
	tb := ge.ToolBar()
	if tb.HasChildren() {
		return
	}
	tb.SetStretchMaxWidth()
	giv.ToolBarView(ge, ge.Viewport, tb)
}

// SplitViewConfig returns a TypeAndNameList for configuring the SplitView
func (ge *Gide) SplitViewConfig() kit.TypeAndNameList {
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_Frame, "filetree-fr")
	for i := 0; i < ge.NTextViews; i++ {
		config.Add(gi.KiT_Layout, fmt.Sprintf("textview-lay-%v", i))
	}
	// todo: tab view
	return config
}

var fnFolderProps = ki.Props{
	"icon":     "folder-open",
	"icon-off": "folder",
}

// ConfigSplitView configures the SplitView.
func (ge *Gide) ConfigSplitView() {
	split, _ := ge.SplitView()
	if split == nil {
		return
	}
	split.Dim = gi.X

	// todo: gide prefs for these
	split.SetProp("word-wrap", true)
	split.SetProp("tab-size", 4)
	split.SetProp("font-family", "Go Mono")

	config := ge.SplitViewConfig()
	mods, updt := split.ConfigChildren(config, true)
	if mods {
		ftfr := split.KnownChild(0).(*gi.Frame)
		ft := ftfr.AddNewChild(giv.KiT_TreeView, "filetree").(*giv.TreeView)
		ft.SetRootNode(&ge.Files)
		// set icons -- todo need an update routine for this
		ft.FuncDownMeFirst(0, ft, func(k ki.Ki, level int, d interface{}) bool {
			if k.TypeEmbeds(giv.KiT_TreeView) {
				tvn := k.(*giv.TreeView)
				fn := tvn.SrcNode.Ptr.(*FileNode)
				if fn.IsDir() {
					tvn.SetProp("#branch", fnFolderProps)
				} else {
					tvn.Icon = fn.Ic
				}
				return true
			} else {
				return false
			}
		})
		for i := 0; i < ge.NTextViews; i++ {
			txly := split.KnownChild(1 + i).(*gi.Layout)
			txly.SetStretchMaxWidth()
			txly.SetStretchMaxHeight()
			txly.SetMinPrefWidth(units.NewValue(20, units.Ch))
			txly.SetMinPrefHeight(units.NewValue(10, units.Ch))

			txed := txly.AddNewChild(giv.KiT_TextView, fmt.Sprintf("textview-%v", i)).(*giv.TextView)
			txed.HiStyle = "emacs" // todo prefs
			txed.LineNos = true    // todo prefs
		}

		// todo: tab view on right

		ft.TreeViewSig.Connect(ge.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			if data == nil {
				return
			}
			tvn, _ := data.(ki.Ki).Embed(giv.KiT_TreeView).(*giv.TreeView)
			gee, _ := recv.Embed(KiT_Gide).(*Gide)
			if sig == int64(giv.TreeViewSelected) {
				fn := tvn.SrcNode.Ptr.(*FileNode)
				gee.ViewFileNode(fn)
			}
		})
		split.SetSplits(.1, .45, .45) // todo: save splits -- that goes in .gide proj file
		split.UpdateEnd(updt)
	}
}

// this is not necc and resets root pointer
// func (ge *Gide) Style2D() {
// 	if ge.Viewport != nil && ge.Viewport.DoingFullRender {
// 		ge.UpdateFromRoot()
// 	}
// 	ge.Frame.Style2D()
// }

func (ge *Gide) Render2D() {
	ge.ToolBar().UpdateActions()
	if win := ge.ParentWindow(); win != nil {
		if !win.IsResizing() {
			win.MainMenuUpdateActives()
		}
	}
	ge.Frame.Render2D()
}

var GideProps = ki.Props{
	"background-color": &gi.Prefs.Colors.Background,
	"color":            &gi.Prefs.Colors.Font,
	"max-width":        -1,
	"max-height":       -1,
	"#title": ki.Props{
		"max-width":        -1,
		"horizontal-align": gi.AlignCenter,
		"vertical-align":   gi.AlignTop,
	},
	"ToolBar": ki.PropSlice{
		{"UpdateFiles", ki.Props{
			"shortcut": "Command+U",
			"icon":     "update",
		}},
		{"SaveActiveView", ki.Props{
			"label": "Save",
			"icon":  "file-save",
		}},
		{"SaveActiveViewAs", ki.Props{
			"label": "Save As...",
			"icon":  "file-save",
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"default-field": "ActiveFilename",
				}},
			},
		}},
	},
	"MainMenu": ki.PropSlice{
		{"AppMenu", ki.BlankProp{}},
		{"File", ki.PropSlice{
			{"NewProj", ki.Props{
				"shortcut": "Command+N",
				"Args": ki.PropSlice{
					{"Proj Dir", ki.Props{
						"dirs-only": true, // todo: support
					}},
				},
			}},
			{"OpenProj", ki.Props{
				"shortcut": "Command+O",
				"Args": ki.PropSlice{
					{"File Name", ki.Props{
						"default-field": "ProjFilename",
						"ext":           ".gide",
					}},
				},
			}},
			{"SaveProj", ki.Props{
				// "shortcut": "Command+S",
			}},
			{"SaveProjAs", ki.Props{
				// "shortcut": "Shift+Command+S",
				"label": "Save Proj As...",
				"Args": ki.PropSlice{
					{"File Name", ki.Props{
						"default-field": "ProjFilename",
						"ext":           ".gide",
					}},
				},
			}},
			{"sep-close", ki.BlankProp{}},
			{"Close Window", ki.BlankProp{}},
		}},
		{"Edit", "Copy Cut Paste"},
		{"Window", "Windows"},
	},
}

//////////////////////////////////////////////////////////////////////////////////////
//   Project window

// NewGideProj creates a new Gide window with a new Gide project for given
// path, returning the window and the path
func NewGideProj(path string) (*gi.Window, *Gide) {

	_, projnm, _, _ := ProjPathParse(path)
	winm := "gide-" + projnm

	width := 1280
	height := 720

	win := gi.NewWindow2D(winm, winm, width, height, true) // true = pixel sizes

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()

	ge := mfr.AddNewChild(KiT_Gide, "gide").(*Gide)
	ge.Viewport = vp

	ge.NewProj(gi.FileName(path))

	mmen := win.MainMenu
	giv.MainMenuView(ge, win, mmen)

	tb := ge.ToolBar()
	if tb != nil {
		if asave, ok := tb.FindActionByName("Save"); ok {
			asave.UpdateFunc = func(act *gi.Action) {
				act.SetActiveStateUpdt(ge.ActiveFilename != "")
			}
		}
		tb.UpdateActions()
	}

	inClosePrompt := false
	win.OSWin.SetCloseReqFunc(func(w oswin.Window) {
		if !inClosePrompt {
			inClosePrompt = true
			// if ge.Changed {
			gi.ChoiceDialog(vp, gi.DlgOpts{Title: "Close Without Saving?",
				Prompt: "Do you want to save your changes?  If so, Cancel and then Save"},
				[]string{"Close Without Saving", "Cancel"},
				win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
					switch sig {
					case 0:
						w.Close()
					case 1:
						// default is to do nothing, i.e., cancel
					}
				})
			// } else {
			// 	w.Close()
			// }
		}
	})

	inQuitPrompt := false
	oswin.TheApp.SetQuitReqFunc(func() {
		if !inQuitPrompt {
			inQuitPrompt = true
			gi.PromptDialog(vp, gi.DlgOpts{Title: "Really Quit?",
				Prompt: "Are you <i>sure</i> you want to quit?"}, true, true,
				win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
					if sig == int64(gi.DialogAccepted) {
						oswin.TheApp.Quit()
					} else {
						inQuitPrompt = false
					}
				})
		}
	})

	// win.OSWin.SetCloseCleanFunc(func(w oswin.Window) {
	// 	fmt.Printf("Doing final Close cleanup here..\n")
	// })

	// win.OSWin.SetCloseCleanFunc(func(w oswin.Window) {
	// 	// go oswin.TheApp.Quit() // once main window is closed, quit
	// })

	win.MainMenuUpdated()

	vp.UpdateEndNoSig(updt)

	win.GoStartEventLoop()
	return win, ge
}
