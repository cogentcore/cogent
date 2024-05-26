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
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/iox/tomlx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/filetree"
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
type SettingsData struct { //types:add
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
type FileSettings struct { //types:add

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
	err := tomlx.Save(se, se.Filename())
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
	err := tomlx.Open(se, se.Filename())
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
func (se *SettingsData) Apply() { //types:add
	if se.KeyMap != "" {
		SetActiveKeyMapName(se.KeyMap) // fills in missing pieces
	}
	MergeAvailableCmds()
	AvailableLangs.Validate()
	se.ApplyEnvVars()
}

// SetGoMod applies the given gomod setting, setting the GO111MODULE env var
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

func (se *SettingsData) MakeToolbar(p *core.Plan) {
	core.Add(p, func(w *views.FuncButton) {
		w.SetFunc(se.EditKeyMaps).SetIcon(icons.Keyboard)
	})
	core.Add(p, func(w *views.FuncButton) {
		w.SetFunc(se.EditLangOpts).SetIcon(icons.Subtitles)
	})
	core.Add(p, func(w *views.FuncButton) {
		w.SetFunc(se.EditCmds).SetIcon(icons.KeyboardCommandKey)
	})
	core.Add(p, func(w *views.FuncButton) {
		w.SetFunc(se.EditSplits).SetIcon(icons.VerticalSplit)
	})
	core.Add(p, func(w *views.FuncButton) {
		w.SetFunc(se.EditRegisters).SetIcon(icons.Variables)
	})
}

// EditKeyMaps opens the KeyMapsView editor to create new keymaps / save /
// load from other files, etc.  Current avail keymaps are saved and loaded
// with settings automatically.
func (se *SettingsData) EditKeyMaps() { //types:add
	se.SaveKeyMaps = true
	KeyMapsView(&AvailableKeyMaps)
}

// EditLangOpts opens the LangsView editor to customize options for each type of
// language / data / file type.
func (se *SettingsData) EditLangOpts() { //types:add
	se.SaveLangOpts = true
	LangsView(&AvailableLangs)
}

// EditCmds opens the CmdsView editor to customize commands you can run.
func (se *SettingsData) EditCmds() { //types:add
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
func (se *SettingsData) EditSplits() { //types:add
	SplitsView(&AvailableSplits)
}

// EditRegisters opens the RegistersView editor to customize saved registers
func (se *SettingsData) EditRegisters() { //types:add
	RegistersView(&AvailableRegisters)
}

//////////////////////////////////////////////////////////////////////////////////////
//   Project Settings

// ProjectSettings are the settings for saving for a project. This IS the project file
type ProjectSettings struct { //types:add

	// file view settings
	Files FileSettings

	// editor settings
	Editor core.EditorSettings `view:"inline"`

	// current named-split config in use for configuring the splitters
	SplitName SplitName

	// the language associated with the most frequently encountered file
	// extension in the file tree -- can be manually set here as well
	MainLang fileinfo.Known

	// the type of version control system used in this project (git, svn, etc).
	// filters commands available
	VersionControl filetree.VersionControlName

	// current project filename for saving / loading specific Code
	// configuration information in a .code file (optional)
	ProjectFilename core.Filename `ext:".code"`

	// root directory for the project. all projects must be organized within
	// a top-level root directory, with all the files therein constituting
	// the scope of the project. By default it is the path for ProjectFilename
	ProjectRoot core.Filename

	// if true, use Go modules, otherwise use GOPATH -- this sets your effective GO111MODULE environment variable accordingly, dynamically -- updated by toolbar checkbox, dynamically
	GoMod bool

	// command(s) to run for main Build button
	BuildCmds CmdNames

	// build directory for main Build button -- set this to the directory where you want to build the main target for this project -- avail as {BuildDir} in commands
	BuildDir core.Filename

	// build target for main Build button, if relevant for your  BuildCmds
	BuildTarg core.Filename

	// executable to run for this project via main Run button -- called by standard Run Project command
	RunExec core.Filename

	// command(s) to run for main Run button (typically Run Project)
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

func (se *ProjectSettings) Update() {
	if se.BuildDir != se.ProjectRoot {
		if se.BuildTarg == se.ProjectRoot {
			se.BuildTarg = se.BuildDir
		}
		if se.RunExec == se.ProjectRoot {
			se.RunExec = se.BuildDir
		}
	}
}

// Open open from file
func (se *ProjectSettings) Open(filename core.Filename) error { //types:add
	err := errors.Log(tomlx.Open(se, string(filename)))
	se.VersionControl = filetree.VersionControlName(strings.ToLower(string(se.VersionControl))) // official names are lowercase now
	return err
}

// Save save to file
func (se *ProjectSettings) Save(filename core.Filename) error { //types:add
	return errors.Log(tomlx.Save(se, string(filename)))
}

// RunExecIsExec returns true if the RunExec is actually executable
func (se *ProjectSettings) RunExecIsExec() bool {
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

	// SavedPathsFilename is the name of the saved file paths file in Cogent Code data directory
	SavedPathsFilename = "saved-paths.json"
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

////////////////////////////////////////////////
// CodeView

// Defaults sets new project defaults based on overall settings
func (ge *CodeView) Defaults() {
	ge.Settings.Files = Settings.Files
	ge.Settings.Editor = core.SystemSettings.Editor
	ge.Settings.Splits = []float32{.1, .325, .325, .25}
	ge.Settings.Debug = cdebug.DefaultParams
}

// GrabSettings grabs the current project preference settings from various
// places, e.g., prior to saving or editing.
func (ge *CodeView) GrabSettings() {
	sv := ge.Splits()
	ge.Settings.Splits = sv.Splits
	ge.Settings.Dirs = ge.Files.Dirs
}

// ApplySettings applies current project preference settings into places where
// they are used -- only for those done prior to loading
func (ge *CodeView) ApplySettings() {
	ge.ProjectFilename = ge.Settings.ProjectFilename
	ge.ProjectRoot = ge.Settings.ProjectRoot
	if ge.Files != nil {
		ge.Files.Dirs = ge.Settings.Dirs
		ge.Files.DirsOnTop = ge.Settings.Files.DirsOnTop
	}
	if len(ge.Kids) > 0 {
		for i := 0; i < NTextEditors; i++ {
			tv := ge.TextEditorByIndex(i)
			if tv.Buffer != nil {
				ge.ConfigTextBuffer(tv.Buffer)
			}
		}
		for _, ond := range ge.OpenNodes {
			if ond.Buffer != nil {
				ge.ConfigTextBuffer(ond.Buffer)
			}
		}
		ge.Splits().SetSplits(ge.Settings.Splits...)
	}
	core.UpdateAll() // drives full rebuild
}

// ApplySettingsAction applies current settings to the project, and updates the project
func (ge *CodeView) ApplySettingsAction() {
	ge.ApplySettings()
	ge.SplitsSetView(ge.Settings.SplitName)
	ge.SetStatus("Applied prefs")
}

// EditProjectSettings allows editing of project settings (settings specific to this project)
func (ge *CodeView) EditProjectSettings() { //types:add
	sv := ProjectSettingsView(&ge.Settings)
	if sv != nil {
		sv.OnChange(func(e events.Event) {
			ge.ApplySettingsAction()
		})
	}
}

func (ge *CodeView) CallSplitsSetView(ctx core.Widget) {
	fb := views.NewSoloFuncButton(ctx, ge.SplitsSetView)
	fb.Args[0].SetValue(ge.Settings.SplitName)
	fb.CallFunc()
}

// SplitsSetView sets split view splitters to given named setting
func (ge *CodeView) SplitsSetView(split SplitName) { //types:add
	sv := ge.Splits()
	sp, _, ok := AvailableSplits.SplitByName(split)
	if ok {
		sv.SetSplits(sp.Splits...).NeedsLayout()
		ge.Settings.SplitName = split
		if !ge.PanelIsOpen(ge.ActiveTextEditorIndex + TextEditor1Index) {
			ge.SetActiveTextEditorIndex((ge.ActiveTextEditorIndex + 1) % 2)
		}
	}
}

// SplitsSave saves current splitter settings to named splitter settings under
// existing name, and saves to prefs file
func (ge *CodeView) SplitsSave(split SplitName) { //types:add
	sv := ge.Splits()
	sp, _, ok := AvailableSplits.SplitByName(split)
	if ok {
		sp.SaveSplits(sv.Splits)
		AvailableSplits.SaveSettings()
	}
}

// SplitsSaveAs saves current splitter settings to new named splitter settings, and
// saves to prefs file
func (ge *CodeView) SplitsSaveAs(name, desc string) { //types:add
	sv := ge.Splits()
	AvailableSplits.Add(name, desc, sv.Splits)
	AvailableSplits.SaveSettings()
}

// SplitsEdit opens the SplitsView editor to customize saved splitter settings
func (ge *CodeView) SplitsEdit() { //types:add
	SplitsView(&AvailableSplits)
}

// LangDefaults applies default language settings based on MainLang
func (ge *CodeView) LangDefaults() {
	ge.Settings.RunCmds = CmdNames{"Build: Run Project"}
	ge.Settings.BuildDir = ge.Settings.ProjectRoot
	ge.Settings.BuildTarg = ge.Settings.ProjectRoot
	ge.Settings.RunExec = core.Filename(filepath.Join(string(ge.Settings.ProjectRoot), ge.Nm))
	if len(ge.Settings.BuildCmds) == 0 {
		switch ge.Settings.MainLang {
		case fileinfo.Go:
			ge.Settings.BuildCmds = CmdNames{"Go: Build Project"}
		case fileinfo.TeX:
			ge.Settings.BuildCmds = CmdNames{"LaTeX: LaTeX PDF"}
			ge.Settings.RunCmds = CmdNames{"File: Open Target"}
		default:
			ge.Settings.BuildCmds = CmdNames{"Build: Make"}
		}
	}
	if ge.Settings.VersionControl == "" {
		repo, _ := ge.Files.FirstVCS()
		if repo != nil {
			ge.Settings.VersionControl = filetree.VersionControlName(repo.Vcs())
		}
	}
}

// GuessMainLang guesses the main language in the project -- returns true if successful
func (ge *CodeView) GuessMainLang() bool {
	ecsc := ge.Files.FileExtCounts(fileinfo.Code)
	ecsd := ge.Files.FileExtCounts(fileinfo.Doc)
	ecs := append(ecsc, ecsd...)
	filetree.NodeNameCountSort(ecs)
	for _, ec := range ecs {
		ls := fileinfo.ExtKnown(ec.Name)
		if ls != fileinfo.Unknown {
			ge.Settings.MainLang = ls
			return true
		}
	}
	return false
}
