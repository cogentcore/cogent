// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	"cogentcore.org/cogent/code/cdebug"
	"cogentcore.org/core/core"
	"cogentcore.org/core/fileinfo"
	"cogentcore.org/core/filetree"
	"cogentcore.org/core/grows/tomls"
	"cogentcore.org/core/grr"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/views"
)

func init() {
	core.TheApp.SetName("Cogent Code")
	core.AllSettings = slices.Insert(core.AllSettings, 1, core.Settings(Settings))
	DefaultKeyMap = "MacEmacs" // todo
	SetActiveKeyMapName(DefaultKeyMap)
	OpenPaths()
	// OpenIcons()
}

// Settings are the overall Code settings
var Settings = &SettingsData{
	SettingsBase: core.SettingsBase{
		Name: "Code",
		File: filepath.Join(core.TheApp.DataDir(), "Cogent Code", "settings.toml"),
	},
}

// SettingsData is the data type for the overall user settings for Code.
type SettingsData struct { //gti:add
	core.SettingsBase

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
	home := core.SystemSettings.User.HomeDir
	texPath := ".:" + home + "/texmf/tex/latex:/Library/TeX/Root/texmf-dist/tex/latex:"
	se.EnvVars = map[string]string{
		"TEXINPUTS":       texPath,
		"BIBINPUTS":       texPath,
		"BSTINPUTS":       texPath,
		"PATH":            home + "/bin:" + home + "/go/bin:/usr/local/bin:/opt/homebrew/bin:/opt/homebrew/shbin:/Library/TeX/texbin:/usr/bin:/bin:/usr/sbin:/sbin",
		"PKG_CONFIG_PATH": "/usr/local/lib/pkgconfig:/opt/homebrew/lib",
	}
}

// Defaults are the defaults for FileSettings
func (se *FileSettings) Defaults() {
	se.DirsOnTop = true
}

func (se *SettingsData) Save() error {
	err := tomls.Save(se, se.Filename())
	if err != nil {
		return err
	}
	if se.SaveKeyMaps {
		AvailableKeyMaps.SaveSettings()
	}
	if se.SaveLangOpts {
		AvailableLangs.SaveSettings()
	}
	if se.SaveCmds {
		CustomCommands.SaveSettings()
	}
	AvailableSplits.SaveSettings()
	AvailableRegisters.SaveSettings()
	return err
}

func (se *SettingsData) Open() error {
	err := tomls.Open(se, se.Filename())
	if err != nil {
		return err
	}
	if se.SaveKeyMaps {
		AvailableKeyMaps.OpenSettings()
	}
	if se.SaveLangOpts {
		AvailableLangs.OpenSettings()
	}
	if se.SaveCmds {
		CustomCommands.OpenSettings()
	}
	AvailableSplits.OpenSettings()
	AvailableRegisters.OpenSettings()
	return err
}

// Apply settings updates things according with settings
func (se *SettingsData) Apply() { //gti:add
	if se.KeyMap != "" {
		SetActiveKeyMapName(se.KeyMap) // fills in missing pieces
	}
	MergeAvailableCmds()
	AvailableLangs.Validate()
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

func (se *SettingsData) ConfigToolbar(tb *core.Toolbar) {
	views.NewFuncButton(tb, se.EditKeyMaps).SetIcon(icons.Keyboard)
	views.NewFuncButton(tb, se.EditLangOpts).SetIcon(icons.Subtitles)
	views.NewFuncButton(tb, se.EditCmds).SetIcon(icons.KeyboardCommandKey)
	views.NewFuncButton(tb, se.EditSplits).SetIcon(icons.VerticalSplit)
	views.NewFuncButton(tb, se.EditRegisters).SetIcon(icons.Variables)
}

// EditKeyMaps opens the KeyMapsView editor to create new keymaps / save /
// load from other files, etc.  Current avail keymaps are saved and loaded
// with settings automatically.
func (se *SettingsData) EditKeyMaps() { //gti:add
	se.SaveKeyMaps = true
	KeyMapsView(&AvailableKeyMaps)
}

// EditLangOpts opens the LangsView editor to customize options for each type of
// language / data / file type.
func (se *SettingsData) EditLangOpts() { //gti:add
	se.SaveLangOpts = true
	LangsView(&AvailableLangs)
}

// EditCmds opens the CmdsView editor to customize commands you can run.
func (se *SettingsData) EditCmds() { //gti:add
	se.SaveCmds = true
	if len(CustomCommands) == 0 {
		exc := &Command{Name: "Example Cmd",
			Desc: "list current dir",
			Lang: fileinfo.Any,
			Cmds: []CmdAndArgs{{Cmd: "ls", Args: []string{"-la"}}},
			Dir:  "{FileDirPath}",
			Wait: CmdNoWait, Focus: CmdNoFocus, Confirm: CmdNoConfirm}

		CustomCommands = append(CustomCommands, exc)
	}
	CmdsView(&CustomCommands)
}

// EditSplits opens the SplitsView editor to customize saved splitter settings
func (se *SettingsData) EditSplits() { //gti:add
	SplitsView(&AvailableSplits)
}

// EditRegisters opens the RegistersView editor to customize saved registers
func (se *SettingsData) EditRegisters() { //gti:add
	RegistersView(&AvailableRegisters)
}

//////////////////////////////////////////////////////////////////////////////////////
//   Project Settings

// ProjSettings are the settings for saving for a project. This IS the project file
type ProjSettings struct { //gti:add

	// file view settings
	Files FileSettings

	// editor settings
	Editor core.EditorSettings `view:"inline"`

	// current named-split config in use for configuring the splitters
	SplitName SplitName

	// the language associated with the most frequently-encountered file
	// extension in the file tree -- can be manually set here as well
	MainLang fileinfo.Known

	// the type of version control system used in this project (git, svn, etc).
	// filters commands available
	VersionControl filetree.VersionControlName

	// current project filename for saving / loading specific Code
	// configuration information in a .code file (optional)
	ProjFilename core.Filename `ext:".code"`

	// root directory for the project. all projects must be organized within
	// a top-level root directory, with all the files therein constituting
	// the scope of the project. By default it is the path for ProjFilename
	ProjRoot core.Filename

	// if true, use Go modules, otherwise use GOPATH -- this sets your effective GO111MODULE environment variable accordingly, dynamically -- updated by toolbar checkbox, dynamically
	GoMod bool

	// command(s) to run for main Build button
	BuildCmds CmdNames

	// build directory for main Build button -- set this to the directory where you want to build the main target for this project -- avail as {BuildDir} in commands
	BuildDir core.Filename

	// build target for main Build button, if relevant for your  BuildCmds
	BuildTarg core.Filename

	// executable to run for this project via main Run button -- called by standard Run Proj command
	RunExec core.Filename

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
func (se *ProjSettings) Open(filename core.Filename) error { //gti:add
	err := grr.Log(tomls.Open(se, string(filename)))
	se.VersionControl = filetree.VersionControlName(strings.ToLower(string(se.VersionControl))) // official names are lowercase now
	return err
}

// Save save to file
func (se *ProjSettings) Save(filename core.Filename) error { //gti:add
	return grr.Log(tomls.Save(se, string(filename)))
}

// RunExecIsExec returns true if the RunExec is actually executable
func (se *ProjSettings) RunExecIsExec() bool {
	fi, err := fileinfo.NewFileInfo(string(se.RunExec))
	if err != nil {
		return false
	}
	return fi.IsExec()
}

//////////////////////////////////////////////////////////////////////////////////////
//   Saved Projects / Paths

var (
	// RecentPaths is a slice of recent file paths
	RecentPaths core.FilePaths

	// SavedPathsFilename is the name of the saved file paths file in Cogent Core prefs directory
	SavedPathsFilename = "code_saved_paths.json"
)

// SavePaths saves the active SavedPaths to prefs dir
func SavePaths() {
	pdir := core.TheApp.AppDataDir()
	pnm := filepath.Join(pdir, SavedPathsFilename)
	RecentPaths.Save(pnm)
}

// OpenPaths loads the active SavedPaths from prefs dir
func OpenPaths() {
	pdir := core.TheApp.AppDataDir()
	pnm := filepath.Join(pdir, SavedPathsFilename)
	RecentPaths.Open(pnm)
}
