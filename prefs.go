// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"encoding/json"
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
	HiStyle    giv.HiStyleName `desc:"highilighting style / theme"`
	FontFamily gi.FontName     `desc:"monospaced font family for editor"`
	TabSize    int             `desc:"size of a tab, in chars"`
	WordWrap   bool            `desc:"wrap lines at word boundaries -- otherwise long lines scroll off the end"`
	LineNos    bool            `desc:"show line numbers"`
	Completion bool            `desc:"use the completion system to suggest options while typing"`
}

// Preferences are the overall user preferences for Gide.
type Preferences struct {
	Files       FilePrefs   `desc:"file view preferences"`
	Editor      EditorPrefs `desc:"editor preferences"`
	KeyMap      KeyMapName  `desc:"key map for gide-specific keyboard sequences"`
	SaveKeyMaps bool        `desc:"if set, the current available set of key maps is saved to your preferences directory, and automatically loaded at startup -- this should be set if you are using custom key maps, but it may be safer to keep it <i>OFF</i> if you are <i>not</i> using custom key maps, so that you'll always have the latest compiled-in standard key maps with all the current key functions bound to standard key chords"`
	SaveLangs   bool        `desc:"if set, the current customized set of language parameters (see Edit Langs) is saved / loaded along with other preferences -- if not set, then you always are using the default compiled-in standard set (which will be updated)"`
	SaveCmds    bool        `desc:"if set, the current customized set of command parameters (see Edit Cmds) is saved / loaded along with other preferences -- if not set, then you always are using the default compiled-in standard set (which will be updated)"`
	Changed     bool        `view:"-" changeflag:"+" json:"-" xml:"-" desc:"flag that is set by StructView by virtue of changeflag tag, whenever an edit is made.  Used to drive save menus etc."`
}

var KiT_Preferences = kit.Types.AddType(&Preferences{}, PreferencesProps)

// Prefs are the overall Gide preferences
var Prefs = Preferences{}

// InitPrefs must be called at startup in mainrun()
func InitPrefs() {
	DefaultKeyMap = "MacEmacs" // todo
	SetActiveKeyMapName(DefaultKeyMap)
	Prefs.Defaults()
	Prefs.Open()
	OpenPaths()
}

func (pf *FilePrefs) Defaults() {
	pf.DirsOnTop = true
}

func (pf *EditorPrefs) Defaults() {
	pf.HiStyle = "emacs"
	pf.FontFamily = "Go Mono"
	pf.TabSize = 4
	pf.WordWrap = true
	pf.LineNos = true
	pf.Completion = true
}

func (pf *Preferences) Defaults() {
	pf.Files.Defaults()
	pf.Editor.Defaults()
	pf.KeyMap = DefaultKeyMap
}

// PrefsFileName is the name of the preferences file in GoGi prefs directory
var PrefsFileName = "gide_prefs.json"

// Apply preferences updates things according with settings
func (pf *Preferences) Apply() {
	if pf.KeyMap != "" {
		SetActiveKeyMapName(pf.KeyMap) // fills in missing pieces
	}
	MergeAvailCmds()
	AvailLangs.Validate()
}

// Open preferences from GoGi standard prefs directory, and applies them
func (pf *Preferences) Open() error {
	pdir := oswin.TheApp.AppPrefsDir()
	pnm := filepath.Join(pdir, PrefsFileName)
	b, err := ioutil.ReadFile(pnm)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, pf)
	if pf.SaveKeyMaps {
		AvailKeyMaps.OpenPrefs()
	}
	if pf.SaveLangs {
		AvailLangs.OpenPrefs()
	}
	if pf.SaveCmds {
		CustomCmds.OpenPrefs()
	}
	AvailSplits.OpenPrefs()
	AvailRegisters.OpenPrefs()
	pf.Apply()
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
	if pf.SaveKeyMaps {
		AvailKeyMaps.SavePrefs()
	}
	if pf.SaveLangs {
		AvailLangs.SavePrefs()
	}
	if pf.SaveCmds {
		CustomCmds.SavePrefs()
	}
	AvailSplits.SavePrefs()
	AvailRegisters.SavePrefs()
	pf.Changed = false
	return err
}

// EditKeyMaps opens the KeyMapsView editor to create new keymaps / save /
// load from other files, etc.  Current avail keymaps are saved and loaded
// with preferences automatically.
func (pf *Preferences) EditKeyMaps() {
	pf.SaveKeyMaps = true
	pf.Changed = true
	KeyMapsView(&AvailKeyMaps)
}

// EditLangs opens the LangsView editor to customize settings for each type of
// language / data / file type.
func (pf *Preferences) EditLangs() {
	pf.SaveLangs = true
	pf.Changed = true
	LangsView(&AvailLangs)
}

// EditCmds opens the CmdsView editor to customize commands you can run.
func (pf *Preferences) EditCmds() {
	pf.SaveCmds = true
	pf.Changed = true
	CmdsView(&CustomCmds)
}

// EditSplits opens the SplitsView editor to customize saved splitter settings
func (pf *Preferences) EditSplits() {
	pf.Changed = true
	SplitsView(&AvailSplits)
}

// EditRegisters opens the RegistersView editor to customize saved registers
func (pf *Preferences) EditRegisters() {
	pf.Changed = true
	RegistersView(&AvailRegisters)
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
		{"sep-key", ki.BlankProp{}},
		{"EditKeyMaps", ki.Props{
			"icon": "keyboard",
			"desc": "opens the KeyMapsView editor to create new keymaps / save / load from other files, etc.  Current keymaps are saved and loaded with preferences automatically if SaveKeyMaps is clicked (will be turned on automatically if you open this editor).",
		}},
		{"EditLangs", ki.Props{
			"icon": "file-text",
			"desc": "opens the LangsView editor to customize settings for each type of language / data / file type.  Current customized settings are saved and loaded with preferences automatically if SaveLangs is clicked (will be turned on automatically if you open this editor).",
		}},
		{"EditCmds", ki.Props{
			"icon": "file-binary",
			"desc": "opens the CmdsView editor to add custom commands you can run, in addition to standard commands built into the system.  Current customized settings are saved and loaded with preferences automatically if SaveCmds is clicked (will be turned on automatically if you open this editor).",
		}},
		{"EditSplits", ki.Props{
			"icon": "file-binary",
			"desc": "opens the SplitsView editor of saved named splitter settings.  Current customized settings are saved and loaded with preferences automatically.",
		}},
		{"EditRegisters", ki.Props{
			"icon": "file-binary",
			"desc": "opens the RegistersView editor of saved named text registers.  Current values are saved and loaded with preferences automatically.",
		}},
	},
}

//////////////////////////////////////////////////////////////////////////////////////
//   Project Prefs

// ProjPrefs are the preferences for saving for a project -- this IS the project file
type ProjPrefs struct {
	Files        FilePrefs      `desc:"file view preferences"`
	Editor       EditorPrefs    `desc:"editor preferences"`
	MainLang     LangName       `desc:"the language associated with the most frequently-encountered file extension in the file tree -- can be manually set here as well"`
	VersCtrl     VersCtrlName   `desc:"the type of version control system used in this project (git, svn, etc) -- filters commands available"`
	ChangeLog    ChangeLog      `desc:"log of version control commits made through Gide by current author -- use appropriate VCS log command to see full set of changes"`
	ProjFilename gi.FileName    `ext:".gide" desc:"current project filename for saving / loading specific Gide configuration information in a .gide file (optional)"`
	ProjRoot     gi.FileName    `desc:"root directory for the project -- all projects must be organized within a top-level root directory, with all the files therein constituting the scope of the project -- by default it is the path for ProjFilename"`
	BuildCmds    CmdNames       `desc:"command(s) to run for main Build button"`
	BuildDir     gi.FileName    `desc:"build directory for main Build button -- set this to the directory where you want to build the main target for this project -- avail as {BuildDir} in commands"`
	BuildTarg    gi.FileName    `desc:"build target for main Build button, if relevant for your  BuildCmds"`
	RunExec      gi.FileName    `desc:"executable to run for this project via main Run button -- called by standard Run Proj command"`
	RunCmds      CmdNames       `desc:"command(s) to run for main Run button (typically Run Proj)"`
	Find         FindParams     `view:"-" desc:"saved find params"`
	Spell        SpellParams    `view:"-" desc:"saved spell params"`
	OpenDirs     giv.OpenDirMap `view:"-" desc:"open directories"`
	Register     RegisterName   `view:"-" desc:"last register used"`
	Splits       []float32      `view:"-" desc:"current splitter splits"`
	Changed      bool           `view:"-" changeflag:"+" json:"-" xml:"-" desc:"flag that is set by StructView by virtue of changeflag tag, whenever an edit is made.  Used to drive save menus etc."`
}

var KiT_ProjPrefs = kit.Types.AddType(&ProjPrefs{}, ProjPrefsProps)

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

// ProjPrefsProps define the ToolBar and MenuBar for StructView, e.g.,
// giv.PrefsView -- don't have a save option as that would save to regular prefs
var ProjPrefsProps = ki.Props{
	"MainMenu": ki.PropSlice{
		{"AppMenu", ki.BlankProp{}},
		{"File", ki.PropSlice{
			{"Close Window", ki.BlankProp{}},
		}},
		{"Edit", "Copy Cut Paste"},
		{"Window", "Windows"},
	},
	// "ToolBar": ki.PropSlice{},
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
