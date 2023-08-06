// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/svg"
	"github.com/goki/gide/gidebug"
	"github.com/goki/gide/icons"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/pi/filecat"
)

// FilePrefs contains file view preferences
type FilePrefs struct {

	// if true, then all directories are placed at the top of the tree view -- otherwise everything is alpha sorted
	DirsOnTop bool `desc:"if true, then all directories are placed at the top of the tree view -- otherwise everything is alpha sorted"`
}

// Preferences are the overall user preferences for Gide.
type Preferences struct {

	// file view preferences
	Files FilePrefs `desc:"file view preferences"`

	// environment variables to set for this app -- if run from the command line, standard shell environment variables are inherited, but on some OS's (Mac), they are not set when run as a gui app
	EnvVars map[string]string `desc:"environment variables to set for this app -- if run from the command line, standard shell environment variables are inherited, but on some OS's (Mac), they are not set when run as a gui app"`

	// key map for gide-specific keyboard sequences
	KeyMap KeyMapName `desc:"key map for gide-specific keyboard sequences"`

	// if set, the current available set of key maps is saved to your preferences directory, and automatically loaded at startup -- this should be set if you are using custom key maps, but it may be safer to keep it <i>OFF</i> if you are <i>not</i> using custom key maps, so that you'll always have the latest compiled-in standard key maps with all the current key functions bound to standard key chords
	SaveKeyMaps bool `desc:"if set, the current available set of key maps is saved to your preferences directory, and automatically loaded at startup -- this should be set if you are using custom key maps, but it may be safer to keep it <i>OFF</i> if you are <i>not</i> using custom key maps, so that you'll always have the latest compiled-in standard key maps with all the current key functions bound to standard key chords"`

	// if set, the current customized set of language options (see Edit Lang Opts) is saved / loaded along with other preferences -- if not set, then you always are using the default compiled-in standard set (which will be updated)
	SaveLangOpts bool `desc:"if set, the current customized set of language options (see Edit Lang Opts) is saved / loaded along with other preferences -- if not set, then you always are using the default compiled-in standard set (which will be updated)"`

	// if set, the current customized set of command parameters (see Edit Cmds) is saved / loaded along with other preferences -- if not set, then you always are using the default compiled-in standard set (which will be updated)
	SaveCmds bool `desc:"if set, the current customized set of command parameters (see Edit Cmds) is saved / loaded along with other preferences -- if not set, then you always are using the default compiled-in standard set (which will be updated)"`

	// [view: -] flag that is set by StructView by virtue of changeflag tag, whenever an edit is made.  Used to drive save menus etc.
	Changed bool `view:"-" changeflag:"+" json:"-" xml:"-" desc:"flag that is set by StructView by virtue of changeflag tag, whenever an edit is made.  Used to drive save menus etc."`
}

var KiT_Preferences = kit.Types.AddType(&Preferences{}, PreferencesProps)

// Prefs are the overall Gide preferences
var Prefs = Preferences{}

// OpenIcons loads the gide icons into the current icon set
func OpenIcons() error {
	err := svg.CurIconSet.OpenIconsFromEmbedDir(icons.Icons, ".")
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

// InitPrefs must be called at startup in mainrun()
func InitPrefs() {
	DefaultKeyMap = "MacEmacs" // todo
	SetActiveKeyMapName(DefaultKeyMap)
	Prefs.Defaults()
	Prefs.Open()
	OpenPaths()
	OpenIcons()
	// TheConsole.Init() // must do this manually
	gi.CustomAppMenuFunc = func(m *gi.Menu, win *gi.Window) {
		m.InsertActionAfter("GoGi Preferences...", gi.ActOpts{Label: "Gide Preferences..."},
			win, func(recv, send ki.Ki, sig int64, data any) {
				PrefsView(&Prefs)
			})
	}
}

// Defaults are the defaults for FilePrefs
func (pf *FilePrefs) Defaults() {
	pf.DirsOnTop = true
}

// Defaults are the defaults for Preferences
func (pf *Preferences) Defaults() {
	pf.Files.Defaults()
	pf.KeyMap = DefaultKeyMap
	home := gi.Prefs.User.HomeDir
	texPath := ".:" + home + "/texmf/tex/latex:/Library/TeX/Root/texmf-dist/tex/latex:"
	pf.EnvVars = map[string]string{
		"TEXINPUTS":       texPath,
		"BIBINPUTS":       texPath,
		"BSTINPUTS":       texPath,
		"PATH":            home + "/bin:" + home + "/go/bin:/usr/local/bin:/opt/homebrew/bin:/opt/homebrew/shbin:/Library/TeX/texbin:/usr/bin:/bin:/usr/sbin:/sbin",
		"PKG_CONFIG_PATH": "/usr/local/lib/pkgconfig:/opt/homebrew/lib",
	}
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
	pf.ApplyEnvVars()
}

// ApplyGoMod applies the given gomod setting, setting the GO111MODULE env var
func SetGoMod(gomod bool) {
	if gomod {
		os.Setenv("GO111MODULE", "on")
	} else {
		os.Setenv("GO111MODULE", "off")
	}
}

// ApplyEnvVars applies environment variables set in EnvVars
func (pf *Preferences) ApplyEnvVars() {
	for k, v := range pf.EnvVars {
		os.Setenv(k, v)
	}
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
	if pf.SaveLangOpts {
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
	if pf.SaveLangOpts {
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

// VersionInfo returns Gide version information
func (pf *Preferences) VersionInfo() string {
	vinfo := Version + " date: " + VersionDate + " UTC; git commit-1: " + GitCommit
	return vinfo
}

// EditKeyMaps opens the KeyMapsView editor to create new keymaps / save /
// load from other files, etc.  Current avail keymaps are saved and loaded
// with preferences automatically.
func (pf *Preferences) EditKeyMaps() {
	pf.SaveKeyMaps = true
	pf.Changed = true
	KeyMapsView(&AvailKeyMaps)
}

// EditLangOpts opens the LangsView editor to customize options for each type of
// language / data / file type.
func (pf *Preferences) EditLangOpts() {
	pf.SaveLangOpts = true
	pf.Changed = true
	LangsView(&AvailLangs)
}

// EditCmds opens the CmdsView editor to customize commands you can run.
func (pf *Preferences) EditCmds() {
	pf.SaveCmds = true
	pf.Changed = true
	if len(CustomCmds) == 0 {
		exc := &Command{Name: "Example Cmd",
			Desc: "list current dir",
			Lang: filecat.Any,
			Cmds: []CmdAndArgs{{Cmd: "ls", Args: []string{"-la"}}},
			Dir:  "{FileDirPath}",
			Wait: CmdNoWait, Focus: CmdNoFocus, Confirm: CmdNoConfirm}

		CustomCmds = append(CustomCmds, exc)
	}
	CmdsView(&CustomCmds)
}

// EditSplits opens the SplitsView editor to customize saved splitter settings
func (pf *Preferences) EditSplits() {
	SplitsView(&AvailSplits)
}

// EditRegisters opens the RegistersView editor to customize saved registers
func (pf *Preferences) EditRegisters() {
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
				"updtfunc": giv.ActionUpdateFunc(func(pfi any, act *gi.Action) {
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
		{"Apply", ki.Props{
			"desc": "Applies current prefs settings so they affect actual functionality.",
			"icon": "update",
		}},
		{"Save", ki.Props{
			"desc": "Saves current preferences to standard prefs.json file, which is auto-loaded at startup.",
			"icon": "file-save",
			"updtfunc": giv.ActionUpdateFunc(func(pfi any, act *gi.Action) {
				pf := pfi.(*Preferences)
				act.SetActiveStateUpdt(pf.Changed)
			}),
		}},
		{"VersionInfo", ki.Props{
			"desc":        "shows current Gide version information",
			"icon":        "info",
			"show-return": true,
		}},
		{"sep-key", ki.BlankProp{}},
		{"EditKeyMaps", ki.Props{
			"icon": "keyboard",
			"desc": "opens the KeyMapsView editor to create new keymaps / save / load from other files, etc.  Current keymaps are saved and loaded with preferences automatically if SaveKeyMaps is clicked (will be turned on automatically if you open this editor).",
		}},
		{"EditLangOpts", ki.Props{
			"icon": "file-text",
			"desc": "opens the LangsView editor to customize options different language / data / file types.  Current customized settings are saved and loaded with preferences automatically if SaveLangOpts is clicked (will be turned on automatically if you open this editor).",
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

	// file view preferences
	Files FilePrefs `desc:"file view preferences"`

	// [view: inline] editor preferences
	Editor gi.EditorPrefs `view:"inline" desc:"editor preferences"`

	// current named-split config in use for configuring the splitters
	SplitName SplitName `desc:"current named-split config in use for configuring the splitters"`

	// the language associated with the most frequently-encountered file extension in the file tree -- can be manually set here as well
	MainLang filecat.Supported `desc:"the language associated with the most frequently-encountered file extension in the file tree -- can be manually set here as well"`

	// the type of version control system used in this project (git, svn, etc) -- filters commands available
	VersCtrl giv.VersCtrlName `desc:"the type of version control system used in this project (git, svn, etc) -- filters commands available"`

	// current project filename for saving / loading specific Gide configuration information in a .gide file (optional)
	ProjFilename gi.FileName `ext:".gide" desc:"current project filename for saving / loading specific Gide configuration information in a .gide file (optional)"`

	// root directory for the project -- all projects must be organized within a top-level root directory, with all the files therein constituting the scope of the project -- by default it is the path for ProjFilename
	ProjRoot gi.FileName `desc:"root directory for the project -- all projects must be organized within a top-level root directory, with all the files therein constituting the scope of the project -- by default it is the path for ProjFilename"`

	// if true, use Go modules, otherwise use GOPATH -- this sets your effective GO111MODULE environment variable accordingly, dynamically -- updated by toolbar checkbox, dynamically
	GoMod bool `desc:"if true, use Go modules, otherwise use GOPATH -- this sets your effective GO111MODULE environment variable accordingly, dynamically -- updated by toolbar checkbox, dynamically"`

	// command(s) to run for main Build button
	BuildCmds CmdNames `desc:"command(s) to run for main Build button"`

	// build directory for main Build button -- set this to the directory where you want to build the main target for this project -- avail as {BuildDir} in commands
	BuildDir gi.FileName `desc:"build directory for main Build button -- set this to the directory where you want to build the main target for this project -- avail as {BuildDir} in commands"`

	// build target for main Build button, if relevant for your  BuildCmds
	BuildTarg gi.FileName `desc:"build target for main Build button, if relevant for your  BuildCmds"`

	// executable to run for this project via main Run button -- called by standard Run Proj command
	RunExec gi.FileName `desc:"executable to run for this project via main Run button -- called by standard Run Proj command"`

	// command(s) to run for main Run button (typically Run Proj)
	RunCmds CmdNames `desc:"command(s) to run for main Run button (typically Run Proj)"`

	// custom debugger parameters for this project
	Debug gidebug.Params `desc:"custom debugger parameters for this project"`

	// [view: -] saved find params
	Find FindParams `view:"-" desc:"saved find params"`

	// [view: -] saved structure params
	Symbols SymbolsParams `view:"-" desc:"saved structure params"`

	// [view: -] directory properties
	Dirs giv.DirFlagMap `view:"-" desc:"directory properties"`

	// [view: -] last register used
	Register RegisterName `view:"-" desc:"last register used"`

	// [view: -] current splitter splits
	Splits []float32 `view:"-" desc:"current splitter splits"`

	// [view: -] flag that is set by StructView by virtue of changeflag tag, whenever an edit is made.  Used to drive save menus etc.
	Changed bool `view:"-" changeflag:"+" json:"-" xml:"-" desc:"flag that is set by StructView by virtue of changeflag tag, whenever an edit is made.  Used to drive save menus etc."`
}

var KiT_ProjPrefs = kit.Types.AddType(&ProjPrefs{}, ProjPrefsProps)

func (pf *ProjPrefs) Update() {
	if pf.BuildDir != pf.ProjRoot {
		if pf.BuildTarg == pf.ProjRoot {
			pf.BuildTarg = pf.BuildDir
		}
		if pf.RunExec == pf.ProjRoot {
			pf.RunExec = pf.BuildDir
		}
	}
}

// OpenJSON open from JSON file
func (pf *ProjPrefs) OpenJSON(filename gi.FileName) error {
	b, err := ioutil.ReadFile(string(filename))
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, pf)
	pf.VersCtrl = giv.VersCtrlName(strings.ToLower(string(pf.VersCtrl))) // official names are lowercase now
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

// RunExecIsExec returns true if the RunExec is actually executable
func (pf *ProjPrefs) RunExecIsExec() bool {
	fi, err := giv.NewFileInfo(string(pf.RunExec))
	if err != nil {
		return false
	}
	return fi.IsExec()
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

// SavedPaths is a slice of strings that are file paths
var SavedPaths gi.FilePaths

// SavedPathsFileName is the name of the saved file paths file in GoGi prefs directory
var SavedPathsFileName = "gide_saved_paths.json"

// GideViewResetRecents defines a string that is added as an item to the recents menu
var GideViewResetRecents = "<i>Reset Recents</i>"

// GideViewEditRecents defines a string that is added as an item to the recents menu
var GideViewEditRecents = "<i>Edit Recents...</i>"

// SavedPathsExtras are the reset and edit items we add to the recents menu
var SavedPathsExtras = []string{gi.MenuTextSeparator, GideViewResetRecents, GideViewEditRecents}

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
