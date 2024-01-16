// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/goki/gide/v2/gidebug"
	"goki.dev/fi"
	"goki.dev/filetree"
	"goki.dev/gi"
	"goki.dev/grows/tomls"
	"goki.dev/grr"
)

func init() {
	gi.AllSettings = slices.Insert(gi.AllSettings, 1, gi.Settings(Settings))
	DefaultKeyMap = "MacEmacs" // todo
	SetActiveKeyMapName(DefaultKeyMap)
	OpenPaths()
	// OpenIcons()
}

// Settings are the overall Gide settings
var Settings = &SettingsData{
	SettingsBase: gi.SettingsBase{
		Name: "Gide",
		File: filepath.Join("Gide", "settings.toml"),
	},
}

// SettingsData is the data type for the overall user settings for Gide.
type SettingsData struct { //gti:add
	gi.SettingsBase

	// file view settings
	Files FileSettings

	// environment variables to set for this app -- if run from the command line, standard shell environment variables are inherited, but on some OS's (Mac), they are not set when run as a gui app
	EnvVars map[string]string

	// key map for gide-specific keyboard sequences
	KeyMap KeyMapName

	// if set, the current available set of key maps is saved to your preferences directory, and automatically loaded at startup -- this should be set if you are using custom key maps, but it may be safer to keep it <i>OFF</i> if you are <i>not</i> using custom key maps, so that you'll always have the latest compiled-in standard key maps with all the current key functions bound to standard key chords
	SaveKeyMaps bool

	// if set, the current customized set of language options (see Edit Lang Opts) is saved / loaded along with other preferences -- if not set, then you always are using the default compiled-in standard set (which will be updated)
	SaveLangOpts bool

	// if set, the current customized set of command parameters (see Edit Cmds) is saved / loaded along with other preferences -- if not set, then you always are using the default compiled-in standard set (which will be updated)
	SaveCmds bool

	// flag that is set by StructView by virtue of changeflag tag, whenever an edit is made.  Used to drive save menus etc.
	Changed bool `view:"-" changeflag:"+" json:"-" toml:"-" xml:"-"`
}

// FileSettings contains file view settings
type FileSettings struct { //gti:add

	// if true, then all directories are placed at the top of the tree view -- otherwise everything is alpha sorted
	DirsOnTop bool
}

// todo:
// OpenIcons loads the gide icons into the current icon set
// func OpenIcons() error {
// 	err := svg.CurIconSet.OpenIconsFromEmbedDir(icons.Icons, ".")
// 	if err != nil {
// 		log.Println(err)
// 		return err
// 	}
// 	return nil
// }

// Defaults are the defaults for Preferences
func (se *SettingsData) Defaults() {
	se.Files.Defaults()
	se.KeyMap = DefaultKeyMap
	home := gi.SystemSettings.User.HomeDir
	texPath := ".:" + home + "/texmf/tex/latex:/Library/TeX/Root/texmf-dist/tex/latex:"
	se.EnvVars = map[string]string{
		"TEXINPUTS":       texPath,
		"BIBINPUTS":       texPath,
		"BSTINPUTS":       texPath,
		"PATH":            home + "/bin:" + home + "/go/bin:/usr/local/bin:/opt/homebrew/bin:/opt/homebrew/shbin:/Library/TeX/texbin:/usr/bin:/bin:/usr/sbin:/sbin",
		"PKG_CONFIG_PATH": "/usr/local/lib/pkgconfig:/opt/homebrew/lib",
	}
}

// Defaults are the defaults for FilePrefs
func (se *FileSettings) Defaults() {
	se.DirsOnTop = true
}

func (se *SettingsData) Save() error {
	err := tomls.Save(se, se.Filename())
	if err != nil {
		return err
	}
	if se.SaveKeyMaps {
		AvailKeyMaps.SavePrefs()
	}
	if se.SaveLangOpts {
		AvailLangs.SavePrefs()
	}
	if se.SaveCmds {
		CustomCmds.SavePrefs()
	}
	AvailSplits.SavePrefs()
	AvailRegisters.SavePrefs()
	return err
}

func (se *SettingsData) Open() error {
	err := tomls.Open(se, se.Filename())
	if err != nil {
		return err
	}
	if se.SaveKeyMaps {
		AvailKeyMaps.OpenSettings()
	}
	if se.SaveLangOpts {
		AvailLangs.OpenSettings()
	}
	if se.SaveCmds {
		CustomCmds.OpenSettings()
	}
	AvailSplits.OpenSettings()
	AvailRegisters.OpenSettings()
	return err
}

// Apply preferences updates things according with settings
func (se *SettingsData) Apply() { //gti:add
	if se.KeyMap != "" {
		SetActiveKeyMapName(se.KeyMap) // fills in missing pieces
	}
	MergeAvailCmds()
	AvailLangs.Validate()
	se.ApplyEnvVars()
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
func (se *SettingsData) ApplyEnvVars() {
	for k, v := range se.EnvVars {
		os.Setenv(k, v)
	}
}

// VersionInfo returns Gide version information
func (se *SettingsData) VersionInfo() string { //gti:add
	vinfo := Version + " date: " + VersionDate + " UTC; git commit-1: " + GitCommit
	return vinfo
}

// EditKeyMaps opens the KeyMapsView editor to create new keymaps / save /
// load from other files, etc.  Current avail keymaps are saved and loaded
// with preferences automatically.
func (se *SettingsData) EditKeyMaps() { //gti:add
	se.SaveKeyMaps = true
	se.Changed = true
	KeyMapsView(&AvailKeyMaps)
}

// EditLangOpts opens the LangsView editor to customize options for each type of
// language / data / file type.
func (se *SettingsData) EditLangOpts() { //gti:add
	se.SaveLangOpts = true
	se.Changed = true
	LangsView(&AvailLangs)
}

// EditCmds opens the CmdsView editor to customize commands you can run.
func (se *SettingsData) EditCmds() { //gti:add
	se.SaveCmds = true
	se.Changed = true
	if len(CustomCmds) == 0 {
		exc := &Command{Name: "Example Cmd",
			Desc: "list current dir",
			Lang: fi.Any,
			Cmds: []CmdAndArgs{{Cmd: "ls", Args: []string{"-la"}}},
			Dir:  "{FileDirPath}",
			Wait: CmdNoWait, Focus: CmdNoFocus, Confirm: CmdNoConfirm}

		CustomCmds = append(CustomCmds, exc)
	}
	CmdsView(&CustomCmds)
}

// EditSplits opens the SplitsView editor to customize saved splitter settings
func (se *SettingsData) EditSplits() { //gti:add
	SplitsView(&AvailSplits)
}

// EditRegisters opens the RegistersView editor to customize saved registers
func (se *SettingsData) EditRegisters() { //gti:add
	RegistersView(&AvailRegisters)
}

//////////////////////////////////////////////////////////////////////////////////////
//   Project Prefs

// ProjPrefs are the preferences for saving for a project -- this IS the project file
type ProjPrefs struct { //gti:add

	// file view preferences
	Files FileSettings

	// editor preferences
	Editor gi.EditorSettings `view:"inline"`

	// current named-split config in use for configuring the splitters
	SplitName SplitName

	// the language associated with the most frequently-encountered file extension in the file tree -- can be manually set here as well
	MainLang fi.Known

	// the type of version control system used in this project (git, svn, etc) -- filters commands available
	VersCtrl filetree.VersCtrlName

	// current project filename for saving / loading specific Gide configuration information in a .gide file (optional)
	ProjFilename gi.Filename `ext:".gide"`

	// root directory for the project -- all projects must be organized within a top-level root directory, with all the files therein constituting the scope of the project -- by default it is the path for ProjFilename
	ProjRoot gi.Filename

	// if true, use Go modules, otherwise use GOPATH -- this sets your effective GO111MODULE environment variable accordingly, dynamically -- updated by toolbar checkbox, dynamically
	GoMod bool

	// command(s) to run for main Build button
	BuildCmds CmdNames

	// build directory for main Build button -- set this to the directory where you want to build the main target for this project -- avail as {BuildDir} in commands
	BuildDir gi.Filename

	// build target for main Build button, if relevant for your  BuildCmds
	BuildTarg gi.Filename

	// executable to run for this project via main Run button -- called by standard Run Proj command
	RunExec gi.Filename

	// command(s) to run for main Run button (typically Run Proj)
	RunCmds CmdNames

	// custom debugger parameters for this project
	Debug gidebug.Params

	// saved find params
	Find FindParams `view:"-"`

	// saved structure params
	Symbols SymbolsParams `view:"-"`

	// directory properties
	Dirs filetree.DirFlagMap `view:"-"`

	// last register used
	Register RegisterName `view:"-"`

	// current splitter splits
	Splits []float32 `view:"-"`

	// flag that is set by StructView by virtue of changeflag tag, whenever an edit is made.  Used to drive save menus etc.
	Changed bool `view:"-" changeflag:"+" json:"-" toml:"-" xml:"-"`
}

func (se *ProjPrefs) Update() {
	if se.BuildDir != se.ProjRoot {
		if se.BuildTarg == se.ProjRoot {
			se.BuildTarg = se.BuildDir
		}
		if se.RunExec == se.ProjRoot {
			se.RunExec = se.BuildDir
		}
	}
}

// Open open from  file
func (se *ProjPrefs) Open(filename gi.Filename) error { //gti:add
	err := grr.Log(tomls.Open(se, string(filename)))
	se.VersCtrl = filetree.VersCtrlName(strings.ToLower(string(se.VersCtrl))) // official names are lowercase now
	se.Changed = false
	return err
}

// Save save to  file
func (se *ProjPrefs) Save(filename gi.Filename) error { //gti:add
	return grr.Log(tomls.Save(se, string(filename)))
}

// RunExecIsExec returns true if the RunExec is actually executable
func (se *ProjPrefs) RunExecIsExec() bool {
	fi, err := fi.NewFileInfo(string(se.RunExec))
	if err != nil {
		return false
	}
	return fi.IsExec()
}

//////////////////////////////////////////////////////////////////////////////////////
//   Saved Projects / Paths

var (
	// SavedPaths is a slice of strings that are file paths
	SavedPaths gi.FilePaths

	// SavedPathsFilename is the name of the saved file paths file in GoGi prefs directory
	SavedPathsFilename = "gide_saved_paths.json"

	// GideViewResetRecents defines a string that is added as an item to the recents menu
	GideViewResetRecents = "<i>Reset Recents</i>"

	// GideViewEditRecents defines a string that is added as an item to the recents menu
	GideViewEditRecents = "<i>Edit Recents...</i>"

	// SavedPathsExtras are the reset and edit items we add to the recents menu
	SavedPathsExtras = []string{gi.MenuTextSeparator, GideViewResetRecents, GideViewEditRecents}
)

// SavePaths saves the active SavedPaths to prefs dir
func SavePaths() {
	gi.StringsRemoveExtras((*[]string)(&SavedPaths), SavedPathsExtras)
	pdir := AppDataDir()
	pnm := filepath.Join(pdir, SavedPathsFilename)
	SavedPaths.Save(pnm)
	// add back after save
	gi.StringsAddExtras((*[]string)(&SavedPaths), SavedPathsExtras)
}

// OpenPaths loads the active SavedPaths from prefs dir
func OpenPaths() {
	// remove to be sure we don't have duplicate extras
	gi.StringsRemoveExtras((*[]string)(&SavedPaths), SavedPathsExtras)
	pdir := AppDataDir()
	pnm := filepath.Join(pdir, SavedPathsFilename)
	SavedPaths.Open(pnm)
	gi.StringsAddExtras((*[]string)(&SavedPaths), SavedPathsExtras)
}
