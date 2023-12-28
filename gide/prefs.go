// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"os"
	"path/filepath"
	"strings"

	"goki.dev/fi"
	"goki.dev/gi/v2/filetree"
	"goki.dev/gi/v2/gi"
	"goki.dev/gide/v2/gidebug"
	"goki.dev/grows/tomls"
	"goki.dev/grr"
)

// FilePrefs contains file view preferences
type FilePrefs struct { //gti:add

	// if true, then all directories are placed at the top of the tree view -- otherwise everything is alpha sorted
	DirsOnTop bool
}

// Preferences are the overall user preferences for Gide.
type Preferences struct { //gti:add

	// file view preferences
	Files FilePrefs

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

// Prefs are the overall Gide preferences
var Prefs = Preferences{}

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

// InitPrefs must be called at startup in mainrun()
func InitPrefs() {
	DefaultKeyMap = "MacEmacs" // todo
	SetActiveKeyMapName(DefaultKeyMap)
	Prefs.Defaults()
	Prefs.Open()
	OpenPaths()
	// OpenIcons()
}

// Defaults are the defaults for FilePrefs
func (pf *FilePrefs) Defaults() {
	pf.DirsOnTop = true
}

// Defaults are the defaults for Preferences
func (pf *Preferences) Defaults() {
	pf.Files.Defaults()
	pf.KeyMap = DefaultKeyMap
	home := gi.GeneralSettings.User.HomeDir
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
var PrefsFileName = "gide_prefs.toml"

// Apply preferences updates things according with settings
func (pf *Preferences) Apply() { //gti:add
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
func (pf *Preferences) Open() error { //gti:add
	pdir := AppDataDir()
	pnm := filepath.Join(pdir, PrefsFileName)
	err := grr.Log(tomls.Open(pf, pnm))
	if err != nil {
		return err
	}
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
func (pf *Preferences) Save() error { //gti:add
	pdir := AppDataDir()
	pnm := filepath.Join(pdir, PrefsFileName)
	err := grr.Log(tomls.Save(pf, pnm))
	if err != nil {
		return err
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
func (pf *Preferences) VersionInfo() string { //gti:add
	vinfo := Version + " date: " + VersionDate + " UTC; git commit-1: " + GitCommit
	return vinfo
}

// EditKeyMaps opens the KeyMapsView editor to create new keymaps / save /
// load from other files, etc.  Current avail keymaps are saved and loaded
// with preferences automatically.
func (pf *Preferences) EditKeyMaps() { //gti:add
	pf.SaveKeyMaps = true
	pf.Changed = true
	KeyMapsView(&AvailKeyMaps)
}

// EditLangOpts opens the LangsView editor to customize options for each type of
// language / data / file type.
func (pf *Preferences) EditLangOpts() { //gti:add
	pf.SaveLangOpts = true
	pf.Changed = true
	LangsView(&AvailLangs)
}

// EditCmds opens the CmdsView editor to customize commands you can run.
func (pf *Preferences) EditCmds() { //gti:add
	pf.SaveCmds = true
	pf.Changed = true
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
func (pf *Preferences) EditSplits() { //gti:add
	SplitsView(&AvailSplits)
}

// EditRegisters opens the RegistersView editor to customize saved registers
func (pf *Preferences) EditRegisters() { //gti:add
	RegistersView(&AvailRegisters)
}

//////////////////////////////////////////////////////////////////////////////////////
//   Project Prefs

// ProjPrefs are the preferences for saving for a project -- this IS the project file
type ProjPrefs struct { //gti:add

	// file view preferences
	Files FilePrefs

	// editor preferences
	Editor gi.EditorSettings `view:"inline"`

	// current named-split config in use for configuring the splitters
	SplitName SplitName

	// the language associated with the most frequently-encountered file extension in the file tree -- can be manually set here as well
	MainLang fi.Known

	// the type of version control system used in this project (git, svn, etc) -- filters commands available
	VersCtrl filetree.VersCtrlName

	// current project filename for saving / loading specific Gide configuration information in a .gide file (optional)
	ProjFilename gi.FileName `ext:".gide"`

	// root directory for the project -- all projects must be organized within a top-level root directory, with all the files therein constituting the scope of the project -- by default it is the path for ProjFilename
	ProjRoot gi.FileName

	// if true, use Go modules, otherwise use GOPATH -- this sets your effective GO111MODULE environment variable accordingly, dynamically -- updated by toolbar checkbox, dynamically
	GoMod bool

	// command(s) to run for main Build button
	BuildCmds CmdNames

	// build directory for main Build button -- set this to the directory where you want to build the main target for this project -- avail as {BuildDir} in commands
	BuildDir gi.FileName

	// build target for main Build button, if relevant for your  BuildCmds
	BuildTarg gi.FileName

	// executable to run for this project via main Run button -- called by standard Run Proj command
	RunExec gi.FileName

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

// Open open from  file
func (pf *ProjPrefs) Open(filename gi.FileName) error { //gti:add
	err := grr.Log(tomls.Open(pf, string(filename)))
	pf.VersCtrl = filetree.VersCtrlName(strings.ToLower(string(pf.VersCtrl))) // official names are lowercase now
	pf.Changed = false
	return err
}

// Save save to  file
func (pf *ProjPrefs) Save(filename gi.FileName) error { //gti:add
	return grr.Log(tomls.Save(pf, string(filename)))
}

// RunExecIsExec returns true if the RunExec is actually executable
func (pf *ProjPrefs) RunExecIsExec() bool {
	fi, err := fi.NewFileInfo(string(pf.RunExec))
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

	// SavedPathsFileName is the name of the saved file paths file in GoGi prefs directory
	SavedPathsFileName = "gide_saved_paths.json"

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
	pnm := filepath.Join(pdir, SavedPathsFileName)
	SavedPaths.Save(pnm)
	// add back after save
	gi.StringsAddExtras((*[]string)(&SavedPaths), SavedPathsExtras)
}

// OpenPaths loads the active SavedPaths from prefs dir
func OpenPaths() {
	// remove to be sure we don't have duplicate extras
	gi.StringsRemoveExtras((*[]string)(&SavedPaths), SavedPathsExtras)
	pdir := AppDataDir()
	pnm := filepath.Join(pdir, SavedPathsFileName)
	SavedPaths.Open(pnm)
	gi.StringsAddExtras((*[]string)(&SavedPaths), SavedPathsExtras)
}
