// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"

	"github.com/goki/gi"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/oswin"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
)

// FilePrefs contains file view preferences
type FilePrefs struct {
	DirsOnTop bool `desc:"if true, then all directories are placed at the top of the tree view -- otherwise everything is alpha sorted"`
}

// EditorPrefs contains editor preferences
type EditorPrefs struct {
	NViews     int             `desc:"number of textviews available for editing files (default 2)"`
	HiStyle    giv.HiStyleName `desc:"highilighting style / theme"`
	FontFamily gi.FontName     `desc:"monospaced font family for editor"`
	TabSize    int             `desc:"size of a tab, in chars"`
	WordWrap   bool            `desc:"wrap lines at word boundaries -- otherwise long lines scroll off the end"`
	LineNos    bool            `desc:"show line numbers"`
}

// Preferences are the overall user preferences for Gide.
type Preferences struct {
	Files   FilePrefs   `desc:"file view preferences"`
	Editor  EditorPrefs `desc:"editor preferences"`
	Changed bool        `view:"-" changeflag:"+" json:"-" xml:"-" desc:"flag that is set by StructView by virtue of changeflag tag, whenever an edit is made.  Used to drive save menus etc."`
}

var KiT_Preferences = kit.Types.AddType(&Preferences{}, PreferencesProps)

// Prefs are the overall Gide preferences
var Prefs = Preferences{}

// InitPrefs must be called at startup in mainrun()
func InitPrefs() {
	Prefs.Defaults()
	Prefs.Open()
	OpenPaths()
}

func (pf *FilePrefs) Defaults() {
	pf.DirsOnTop = true
}

func (pf *EditorPrefs) Defaults() {
	pf.NViews = 2
	pf.HiStyle = "emacs"
	pf.FontFamily = "Go Mono"
	pf.TabSize = 4
	pf.WordWrap = true
	pf.LineNos = true
}

func (pf *Preferences) Defaults() {
	pf.Files.Defaults()
	pf.Editor.Defaults()
}

// PrefsFileName is the name of the preferences file in GoGi prefs directory
var PrefsFileName = "gide_prefs.json"

// Open preferences from GoGi standard prefs directory
func (pf *Preferences) Open() error {
	pdir := oswin.TheApp.AppPrefsDir()
	pnm := filepath.Join(pdir, PrefsFileName)
	b, err := ioutil.ReadFile(pnm)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, pf)
	pf.Changed = false
	return err
}

// Save Preferences to GoGi standard prefs directory
func (pf *Preferences) Save() error {
	pdir := oswin.TheApp.AppPrefsDir()
	pnm := filepath.Join(pdir, PrefsFileName)
	b, err := json.MarshalIndent(pf, "", "  ")
	if err != nil {
		log.Println(err)
		return err
	}
	err = ioutil.WriteFile(pnm, b, 0644)
	if err != nil {
		log.Println(err)
	}
	pf.Changed = false
	return err
}

// PreferencesProps define the ToolBar and MenuBar for StructView, e.g., giv.PrefsView
var PreferencesProps = ki.Props{
	"MainMenu": ki.PropSlice{
		{"AppMenu", ki.BlankProp{}},
		{"File", ki.PropSlice{
			{"Open", ki.Props{
				"shortcut": "Command+O",
			}},
			{"Save", ki.Props{
				"shortcut": "Command+S",
				"updtfunc": func(pfi interface{}, act *gi.Action) {
					pf := pfi.(*Preferences)
					act.SetActiveState(pf.Changed)
				},
			}},
			{"sep-close", ki.BlankProp{}},
			{"Close Window", ki.BlankProp{}},
		}},
		{"Edit", "Copy Cut Paste"},
		{"Window", "Windows"},
	},
	"ToolBar": ki.PropSlice{
		{"Save", ki.Props{
			"desc": "Saves current preferences to standard prefs.json file, which is auto-loaded at startup.",
			"icon": "file-save",
			"updtfunc": func(pfi interface{}, act *gi.Action) {
				pf := pfi.(*Preferences)
				act.SetActiveStateUpdt(pf.Changed)
			},
		}},
	},
}

//////////////////////////////////////////////////////////////////////////////////////
//   Project Prefs

// ProjPrefs are the preferences for saving for a project -- this IS the project file
type ProjPrefs struct {
	Preferences
	ProjFilename gi.FileName    `view:"-" ext:".gide" desc:"current project filename for saving / loading specific Gide configuration information in a .gide file (optional)"`
	ProjRoot     gi.FileName    `view:"-" desc:"root directory for the project -- all projects must be organized within a top-level root directory, with all the files therein constituting the scope of the project -- by default it is the path for ProjFilename"`
	OpenDirs     giv.OpenDirMap `view:"-" desc:"open directories"`
	Splits       []float32      `view:"-" desc:"splitter splits"`
}

// OpenJSON open from JSON file
func (pf *ProjPrefs) OpenJSON(filename gi.FileName) error {
	b, err := ioutil.ReadFile(string(filename))
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, pf)
	pf.Changed = false
	return err
}

// SaveJSON save to JSON file
func (pf *ProjPrefs) SaveJSON(filename gi.FileName) error {
	b, err := json.MarshalIndent(pf, "", "  ")
	if err != nil {
		log.Println(err)
		return err
	}
	err = ioutil.WriteFile(string(filename), b, 0644)
	if err != nil {
		log.Println(err)
	}
	pf.Changed = false
	return err
}

// PrefsView opens a view of user preferences, returns structview and window
func PrefsView(pf *Preferences) (*giv.StructView, *gi.Window) {
	winm := "gide-prefs"
	// if w, ok := gi.MainWindows.FindName(winm); ok {
	// 	w.OSWin.Raise()
	// 	return
	// }

	width := 800
	height := 800
	win := gi.NewWindow2D(winm, "Gide Preferences", width, height, true)

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()
	mfr.Lay = gi.LayoutVert

	sv := mfr.AddNewChild(giv.KiT_StructView, "sv").(*giv.StructView)
	sv.Viewport = vp
	sv.SetStruct(pf, nil)
	sv.SetStretchMaxWidth()
	sv.SetStretchMaxHeight()

	mmen := win.MainMenu
	giv.MainMenuView(pf, win, mmen)

	inClosePrompt := false
	win.OSWin.SetCloseReqFunc(func(w oswin.Window) {
		if pf.Changed {
			if !inClosePrompt {
				gi.ChoiceDialog(vp, gi.DlgOpts{Title: "Save Prefs Before Closing?",
					Prompt: "Do you want to save any changes to preferences before closing?"},
					[]string{"Save and Close", "Discard and Close", "Cancel"},
					win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
						switch sig {
						case 0:
							pf.Save()
							fmt.Println("Preferences Saved to prefs.json")
							w.Close()
						case 1:
							pf.Open() // if we don't do this, then it actually remains in edited state
							w.Close()
						case 2:
							inClosePrompt = false
							// default is to do nothing, i.e., cancel
						}
					})
			}
		} else {
			w.Close()
		}
	})

	win.MainMenuUpdated()

	vp.UpdateEndNoSig(updt)
	win.GoStartEventLoop()
	return sv, win
}

//////////////////////////////////////////////////////////////////////////////////////
//   Saved Projects / Paths

var SavedPaths gi.FilePaths

// SavedPathsFileName is the name of the saved file paths file in GoGi prefs directory
var SavedPathsFileName = "gide_saved_paths.json"

// SavePaths saves the active SavedPaths to prefs dir
func SavePaths() {
	pdir := oswin.TheApp.AppPrefsDir()
	pnm := filepath.Join(pdir, SavedPathsFileName)
	SavedPaths.SaveJSON(pnm)
}

// OpenPaths loads the active SavedPaths from prefs dir
func OpenPaths() {
	pdir := oswin.TheApp.AppPrefsDir()
	pnm := filepath.Join(pdir, SavedPathsFileName)
	SavedPaths.OpenJSON(pnm)
}
