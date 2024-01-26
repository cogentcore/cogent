// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	"cogentcore.org/cogent"
	"cogentcore.org/cogent/code/cdebug"
	"cogentcore.org/core/fi"
	"cogentcore.org/core/filetree"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/giv"
	"cogentcore.org/core/grows/tomls"
	"cogentcore.org/core/grr"
	"cogentcore.org/core/icons"
)

func init() {
	gi.TheApp.SetName("CogentCode")
	gi.AllSettings = slices.Insert(gi.AllSettings, 1, gi.Settings(Settings))
	DefaultKeyMap = "MacEmacs" // todo
	SetActiveKeyMapName(DefaultKeyMap)
	OpenPaths()
	// OpenIcons()
}

// Settings are the overall Code settings
var Settings = &SettingsData{
	SettingsBase: gi.SettingsBase{
		Name: "Code",
		File: filepath.Join(gi.TheApp.DataDir(), "CogentCode", "settings.toml"),
	},
}

// SettingsData is the data type for the overall user settings for Code.
type SettingsData struct { //gti:add
	gi.SettingsBase

	// file view settings
	Files FileSettings

	// environment variables to set for this app -- if run from the command line, standard shell environment variables are inherited, but on some OS's (Mac), they are not set when run as a gui app
	EnvVars map[string]string

	// key map for code-specific keyboard sequences
	KeyMap KeyMapName

	// if set, the current available set of key maps is saved to your settings directory, and automatically loaded at startup -- this should be set if you are using custom key maps, but it may be safer to keep it <i>OFF</i> if you are <i>not</i> using custom key maps, so that you'll always have the latest compiled-in standard key maps with all the current key functions bound to standard key chords
	SaveKeyMaps bool

	// if set, the current customized set of language options (see Edit Lang Opts) is saved / loaded along with other settings -- if not set, then you always are using the default compiled-in standard set (which will be updated)
	SaveLangOpts bool

	// if set, the current customized set of command parameters (see Edit Cmds) is saved / loaded along with other settings -- if not set, then you always are using the default compiled-in standard set (which will be updated)
	SaveCmds bool
}

// FileSettings contains file view settings
type FileSettings struct { //gti:add

	// if true, then all directories are placed at the top of the tree view -- otherwise everything is alpha sorted
	DirsOnTop bool
}

// todo:
// OpenIcons loads the code icons into the current icon set
// func OpenIcons() error {
// 	err := svg.CurIconSet.OpenIconsFromEmbedDir(icons.Icons, ".")
// 	if err != nil {
// 		log.Println(err)
// 		return err
// 	}
// 	return nil
// }

// Defaults are the defaults for Settings
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

// Apply settings updates things according with settings
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

// VersionInfo returns Code version information
func (se *SettingsData) VersionInfo() string { //gti:add
	vinfo := cogent.Version + " date: " + cogent.VersionDate + " UTC; git commit-1: " + cogent.GitCommit
	return vinfo
}

func (se *SettingsData) ConfigToolbar(tb *gi.Toolbar) {
	giv.NewFuncButton(tb, se.VersionInfo).SetShowReturn(true).SetIcon(icons.Info)
	giv.NewFuncButton(tb, se.EditKeyMaps).SetIcon(icons.Keyboard)
	giv.NewFuncButton(tb, se.EditLangOpts).SetIcon(icons.Subtitles)
	giv.NewFuncButton(tb, se.EditCmds).SetIcon(icons.KeyboardCommandKey)
	giv.NewFuncButton(tb, se.EditSplits).SetIcon(icons.VerticalSplit)
	giv.NewFuncButton(tb, se.EditRegisters).SetIcon(icons.Variables)
}

// EditKeyMaps opens the KeyMapsView editor to create new keymaps / save /
// load from other files, etc.  Current avail keymaps are saved and loaded
// with settings automatically.
func (se *SettingsData) EditKeyMaps() { //gti:add
	se.SaveKeyMaps = true
	KeyMapsView(&AvailKeyMaps)
}

// EditLangOpts opens the LangsView editor to customize options for each type of
// language / data / file type.
func (se *SettingsData) EditLangOpts() { //gti:add
	se.SaveLangOpts = true
	LangsView(&AvailLangs)
}

// EditCmds opens the CmdsView editor to customize commands you can run.
func (se *SettingsData) EditCmds() { //gti:add
	se.SaveCmds = true
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
//   Project Settings

// ProjSettings are the settings for saving for a project. This IS the project file
type ProjSettings struct { //gti:add

	// file view settings
	Files FileSettings

	// editor settings
	Editor gi.EditorSettings `view:"inline"`

	// current named-split config in use for configuring the splitters
	SplitName SplitName

	// the language associated with the most frequently-encountered file
	// extension in the file tree -- can be manually set here as well
	MainLang fi.Known

	// the type of version control system used in this project (git, svn, etc).
	// filters commands available
	VersCtrl filetree.VersCtrlName

	// current project filename for saving / loading specific Code
	// configuration information in a .code file (optional)
	ProjFilename gi.Filename `ext:".code"`

	// root directory for the project. all projects must be organized within
	// a top-level root directory, with all the files therein constituting
	// the scope of the project. By default it is the path for ProjFilename
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
	Debug cdebug.Params

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
}

func (se *ProjSettings) Update() {
	if se.BuildDir != se.ProjRoot {
		if se.BuildTarg == se.ProjRoot {
			se.BuildTarg = se.BuildDir
		}
		if se.RunExec == se.ProjRoot {
			se.RunExec = se.BuildDir
		}
	}
}

// Open open from file
func (se *ProjSettings) Open(filename gi.Filename) error { //gti:add
	err := grr.Log(tomls.Open(se, string(filename)))
	se.VersCtrl = filetree.VersCtrlName(strings.ToLower(string(se.VersCtrl))) // official names are lowercase now
	return err
}

// Save save to file
func (se *ProjSettings) Save(filename gi.Filename) error { //gti:add
	return grr.Log(tomls.Save(se, string(filename)))
}

// RunExecIsExec returns true if the RunExec is actually executable
func (se *ProjSettings) RunExecIsExec() bool {
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

	// SavedPathsFilename is the name of the saved file paths file in Cogent Core prefs directory
	SavedPathsFilename = "code_saved_paths.json"

	// CodeViewResetRecents defines a string that is added as an item to the recents menu
	CodeViewResetRecents = "<i>Reset Recents</i>"

	// CodeViewEditRecents defines a string that is added as an item to the recents menu
	CodeViewEditRecents = "<i>Edit Recents...</i>"

	// SavedPathsExtras are the reset and edit items we add to the recents menu
	SavedPathsExtras = []string{CodeViewResetRecents, CodeViewEditRecents}
)

// SavePaths saves the active SavedPaths to prefs dir
func SavePaths() {
	gi.StringsRemoveExtras((*[]string)(&SavedPaths), SavedPathsExtras)
	pdir := gi.TheApp.AppDataDir()
	pnm := filepath.Join(pdir, SavedPathsFilename)
	SavedPaths.Save(pnm)
	// add back after save
	SavedPaths = append(SavedPaths, SavedPathsExtras...)
}

// OpenPaths loads the active SavedPaths from prefs dir
func OpenPaths() {
	// remove to be sure we don't have duplicate extras
	gi.StringsRemoveExtras((*[]string)(&SavedPaths), SavedPathsExtras)
	pdir := gi.TheApp.AppDataDir()
	pnm := filepath.Join(pdir, SavedPathsFilename)
	SavedPaths.Open(pnm)
	gi.SavedPaths = append(gi.SavedPaths, gi.SavedPathsExtras...)
}
