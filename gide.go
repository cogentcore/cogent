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

	"github.com/goki/gi"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
)

// Gide is the core editor and tab viewer framework for the Gide system.  The
// default view has a tree browser of files on the left, editor panels in the
// middle, and a tabbed viewer on the right.
type Gide struct {
	gi.Frame
	ProjFilename gi.FileName `desc:"current project filename for saving / loading"`
	ProjRoot     gi.FileName `desc:"root directory for the project -- all projects must be organized within a top-level root directory, with all the files therein constituting the scope of the project -- by default it is the path for ProjFilename"`
	Changed      bool        `json:"-" desc:"has the root changed?  we receive update signals from root for changes"`
	Files        FileNode    `desc:"all the files in the project directory and subdirectories"`
}

var KiT_Gide = kit.Types.AddType(&Gide{}, GideProps)

// UpdateFiles updates the list of files saved in project
func (ge *Gide) UpdateFiles() {
	ge.Files.OpenPath(string(ge.ProjRoot))
}

// NewProj opens a new directory for a project at given directory
func (ge *Gide) NewProj(projDir gi.FileName) {
	ge.ProjRoot = projDir
	ge.UpdateProj()
}

// SaveProj saves project file, in a standard JSON-formatted file
func (ge *Gide) SaveProj() {
	if ge.ProjFilename == "" {
		return
	}
	ge.SaveJSON(string(ge.ProjFilename))
	ge.Changed = false
}

// SaveProjAs saves project to given filename, in a standard JSON-formatted file
func (ge *Gide) SaveProjAs(filename gi.FileName) {
	ge.SaveJSON(string(filename))
	ge.Changed = false
	ge.ProjFilename = filename
	ge.UpdateSig() // notify our editor
}

// OpenProj opens project from given filename, in a standard JSON-formatted file
func (ge *Gide) OpenProj(filename gi.FileName) {
	ge.OpenJSON(string(filename))
	ge.ProjFilename = filename // should already be set but..
	ge.UpdateProj()
	ge.SetFullReRender()
	ge.UpdateSig() // notify our editor
}

// // GetAllUpdates connects to all nodes in the tree to receive notification of changes
// func (ge *Gide) GetAllUpdates(root ki.Ki) {
// 	ge.KiRoot.FuncDownMeFirst(0, ge, func(k ki.Ki, level int, d interface{}) bool {
// 		k.NodeSignal().Connect(ge.This, func(recv, send ki.Ki, sig int64, data interface{}) {
// 			gee := recv.Embed(KiT_Gide).(*Gide)
// 			if !gee.Changed {
// 				fmt.Printf("Gide: Tree changed with signal: %v\n", ki.NodeSignals(sig))
// 				gee.Changed = true
// 			}
// 		})
// 		return true
// 	})
// }

// UpdateProj does full update
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

//////////////////////////////////////////////////////////////////////////////////////
//   Save files

// Save1 saves file viewed in editor 1..
func (ge *Gide) Save1() {
	tv1 := ge.TextView1()
	if tv1.Buf != nil {
		tv1.Buf.Save() // todo: errs..
	}
}

// SaveAs1 save as file viewed in editor 1..
func (ge *Gide) SaveAs1(filename gi.FileName) {
	tv1 := ge.TextView1()
	if tv1.Buf != nil {
		tv1.Buf.SaveAs(filename)
	}
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

// todo: generalize all this..

// TextView1 returns the first main TextView1
func (ge *Gide) TextView1() *giv.TextView {
	split, _ := ge.SplitView()
	if split != nil {
		sv := split.KnownChild(1).KnownChild(0).(*giv.TextView)
		return sv
	}
	return nil
}

// TextView2 returns the first main TextView2
func (ge *Gide) TextView2() *giv.TextView {
	split, _ := ge.SplitView()
	if split != nil {
		sv := split.KnownChild(2).KnownChild(0).(*giv.TextView)
		return sv
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

	if len(split.Kids) == 0 {
		ftfr := split.AddNewChild(gi.KiT_Frame, "filetree-fr").(*gi.Frame)
		ft := ftfr.AddNewChild(giv.KiT_TreeView, "filetree").(*giv.TreeView)
		ft.SetRootNode(&ge.Files)

		// generally need to put text view within its own frame for scrolling
		txfr1 := split.AddNewChild(gi.KiT_Frame, "view-frame-1").(*gi.Frame)
		txfr1.SetStretchMaxWidth()
		txfr1.SetStretchMaxHeight()
		txfr1.SetMinPrefWidth(units.NewValue(20, units.Ch))
		txfr1.SetMinPrefHeight(units.NewValue(10, units.Ch))

		txed1 := txfr1.AddNewChild(giv.KiT_TextView, "textview-1").(*giv.TextView)
		txed1.HiStyle = "emacs"
		txed1.LineNos = true

		// generally need to put text view within its own frame for scrolling
		txfr2 := split.AddNewChild(gi.KiT_Frame, "view-frame-2").(*gi.Frame)
		txfr2.SetStretchMaxWidth()
		txfr2.SetStretchMaxHeight()
		txfr2.SetMinPrefWidth(units.NewValue(20, units.Ch))
		txfr2.SetMinPrefHeight(units.NewValue(10, units.Ch))

		txed2 := txfr2.AddNewChild(giv.KiT_TextView, "textview-2").(*giv.TextView)
		txed2.HiStyle = "emacs"
		txed2.LineNos = true

		// todo: tab view on right

		ft.TreeViewSig.Connect(ge.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			if data == nil {
				return
			}
			tvn, _ := data.(ki.Ki).Embed(giv.KiT_TreeView).(*giv.TreeView)
			gee, _ := recv.Embed(KiT_Gide).(*Gide)
			if sig == int64(giv.TreeViewSelected) {
				fn := tvn.SrcNode.Ptr.(*FileNode)
				if err := fn.OpenBuf(); err == nil {
					gee.TextView1().SetBuf(fn.Buf)
				}
			}
		})
		split.SetSplits(.1, .45, .45) // todo: save splits
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
		{"Save1", ki.Props{
			"icon": "file-save",
		}},
		{"SaveAs1", ki.Props{
			"label": "Save 1 As...",
			"icon":  "file-save",
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					// "default-field": "jFilename",
					// "ext":           ".gide",
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

