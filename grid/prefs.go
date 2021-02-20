// Copyright (c) 2021, The Grid Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grid

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/girl"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/svg"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
)

// Preferences for drawing size, etc
type DrawingPrefs struct {
	StdSize  StdSizes `desc:"select a standard size -- this will set units and size"`
	Portrait bool     `desc:"for standard size, use first number as width, second as height"`
	Units    units.Units
	Size     mat32.Vec2 `desc:"drawing size, in Units"`
	Scale    mat32.Vec2 `desc:"drawing scale factor"`
	GridDisp bool       `desc:"turns on the grid display"`
	Grid     int        `desc:"grid spacing, in *integer* units of basic Units"`
}

func (dp *DrawingPrefs) Defaults() {
	dp.StdSize = CustomSize
	dp.Units = units.Pt
	dp.Size.Set(612, 792)
	dp.Scale.Set(1, 1)
	dp.GridDisp = true
	dp.Grid = 12
}

func (dp *DrawingPrefs) Update() {
	if dp.StdSize != CustomSize {
		dp.SetStdSize(dp.StdSize)
	}
}

// SetStdSize sets drawing to a standard size
func (dp *DrawingPrefs) SetStdSize(std StdSizes) error {
	ssv, has := StdSizesMap[std]
	if !has {
		return fmt.Errorf("StdSize: %v not found in StdSizesMap")
	}
	dp.StdSize = std
	dp.Units = ssv.Units
	dp.Size.X = ssv.X
	dp.Size.Y = ssv.Y
	return nil
}

// Preferences is the overall Grid preferences
type Preferences struct {
	Drawing   DrawingPrefs `desc:"default new drawing prefs"`
	Style     girl.Paint   `desc:"default styles"`
	SnapGrid  bool         `desc:"snap positions and sizes to underlying grid"`
	SnapGuide bool         `desc:"snap positions and sizes to line up with other elements"`
	SnapTol   float32      `desc:"proportion tolerance for snapping to grid or any other such guide, e.g., .1 = 10%"`
	SplitName SplitName    `desc:"named-split config in use for configuring the splitters"`
	Changed   bool         `view:"-" changeflag:"+" json:"-" xml:"-" desc:"flag that is set by StructView by virtue of changeflag tag, whenever an edit is made.  Used to drive save menus etc."`
}

var KiT_Preferences = kit.Types.AddType(&Preferences{}, PreferencesProps)

func (pr *Preferences) Defaults() {
	pr.Drawing.Defaults()
	pr.Style.Defaults()
	pr.SnapTol = 0.1
	pr.SnapGrid = true
	pr.SnapGuide = true
}

func (pr *Preferences) Update() {
	pr.Drawing.Update()
	// pr.Style.Update()
}

// Prefs are the overall Grid preferences
var Prefs = Preferences{}

// InitPrefs must be called at startup in mainrun()
func InitPrefs() {
	Prefs.Defaults()
	Prefs.Open()
	OpenPaths()
	svg.CurIconSet.OpenIconsFromAssetDir("../icons", AssetDir, Asset)
	gi.CustomAppMenuFunc = func(m *gi.Menu, win *gi.Window) {
		m.InsertActionAfter("GoGi Preferences...", gi.ActOpts{Label: "Grid Preferences..."},
			win, func(recv, send ki.Ki, sig int64, data interface{}) {
				PrefsView(&Prefs)
			})
	}
}

// PrefsFileName is the name of the preferences file in GoGi prefs directory
var PrefsFileName = "grid_prefs.json"

// Open preferences from GoGi standard prefs directory, and applies them
func (pf *Preferences) Open() error {
	pdir := oswin.TheApp.AppPrefsDir()
	pnm := filepath.Join(pdir, PrefsFileName)
	b, err := ioutil.ReadFile(pnm)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, pf)
	AvailSplits.OpenPrefs()
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
	AvailSplits.SavePrefs()
	pf.Changed = false
	return err
}

// EditSplits opens the SplitsView editor to customize saved splitter settings
func (pf *Preferences) EditSplits() {
	SplitsView(&AvailSplits)
}

// VersionInfo returns Grid version information
func (pf *Preferences) VersionInfo() string {
	vinfo := Version + " date: " + VersionDate + " UTC; git commit-1: " + GitCommit
	return vinfo
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
				"updtfunc": giv.ActionUpdateFunc(func(pfi interface{}, act *gi.Action) {
					pf := pfi.(*Preferences)
					act.SetActiveState(pf.Changed)
				}),
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
			"updtfunc": giv.ActionUpdateFunc(func(pfi interface{}, act *gi.Action) {
				pf := pfi.(*Preferences)
				act.SetActiveStateUpdt(pf.Changed)
			}),
		}},
		{"VersionInfo", ki.Props{
			"desc":        "shows current Grid version information",
			"icon":        "info",
			"show-return": true,
		}},
		{"sep-key", ki.BlankProp{}},
		{"EditSplits", ki.Props{
			"icon": "file-binary",
			"desc": "opens the SplitsView editor of saved named splitter settings.  Current customized settings are saved and loaded with preferences automatically.",
		}},
	},
}

//////////////////////////////////////////////////////////////////////////////////////
//   Saved Projects / Paths

// SavedPaths is a slice of strings that are file paths
var SavedPaths gi.FilePaths

// SavedPathsFileName is the name of the saved file paths file in GoGi prefs directory
var SavedPathsFileName = "grid_saved_paths.json"

// GridViewResetRecents defines a string that is added as an item to the recents menu
var GridViewResetRecents = "<i>Reset Recents</i>"

// GridViewEditRecents defines a string that is added as an item to the recents menu
var GridViewEditRecents = "<i>Edit Recents...</i>"

// SavedPathsExtras are the reset and edit items we add to the recents menu
var SavedPathsExtras = []string{gi.MenuTextSeparator, GridViewResetRecents, GridViewEditRecents}

// SavePaths saves the active SavedPaths to prefs dir
func SavePaths() {
	gi.StringsRemoveExtras((*[]string)(&SavedPaths), SavedPathsExtras)
	pdir := oswin.TheApp.AppPrefsDir()
	pnm := filepath.Join(pdir, SavedPathsFileName)
	SavedPaths.SaveJSON(pnm)
	// add back after save
	gi.StringsAddExtras((*[]string)(&SavedPaths), SavedPathsExtras)
}

// OpenPaths loads the active SavedPaths from prefs dir
func OpenPaths() {
	// remove to be sure we don't have duplicate extras
	gi.StringsRemoveExtras((*[]string)(&SavedPaths), SavedPathsExtras)
	pdir := oswin.TheApp.AppPrefsDir()
	pnm := filepath.Join(pdir, SavedPathsFileName)
	SavedPaths.OpenJSON(pnm)
	gi.StringsAddExtras((*[]string)(&SavedPaths), SavedPathsExtras)
}
