// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"os"
	"path/filepath"
	"strings"

	"cogentcore.org/cogent/code/cdebug"
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/iox/tomlx"
	"cogentcore.org/core/base/vcs"
	"cogentcore.org/core/cli/commands"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/filetree"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/text/text"
	"cogentcore.org/core/tree"
)

func init() {
	core.TheApp.SetName("Cogent Code")
	core.AddAppSettings(Settings)
	OpenPaths()
	// OpenIcons()
}

// Settings are the overall Code settings
var Settings = &SettingsData{
	Name: "Code",
	File: filepath.Join(core.TheApp.DataDir(), "Cogent Code", "settings.toml"),
}

// LangBackends specifies which system provides the language services:
// diagnostics, completions, hover docs, navigation, and symbols.
type LangBackends int32 //enums:enum -trim-prefix LangBackend

const (
	// LangBackendLSP uses external language servers (gopls etc),
	// which run as separate processes and provide full IDE-level
	// language features for any file with an available server,
	// except for languages set to use the parser in the Languages
	// settings (Markdown and TeX by default)
	LangBackendLSP LangBackends = iota

	// LangBackendParser uses the built-in interactive parse system,
	// which runs in-process
	LangBackendParser
)

// SettingsData is the data type for the overall user settings for Code.
type SettingsData struct { //types:add
	core.SettingsBase

	// file picker settings
	Files FileSettings

	// Terminal has the settings for terminals.
	Terminal TerminalSettings

	// if set, the current customized set of language options (see Edit Lang Opts) is saved / loaded along with other settings -- if not set, then you always are using the default compiled-in standard set (which will be updated)
	SaveLangOpts bool

	// executable paths for language servers, keyed by server id (gopls, typescript-language-server, pyright-langserver, pylsp, rust-analyzer, clangd, bash-language-server, vscode-json-language-server) -- paths or names to search in PATH if not an absolute path -- empty by default, meaning just search the PATH for the standard executable
	LSExecs map[string]string `display:"-"`
}

// FileSettings contains file picker settings
type FileSettings struct { //types:add

	// if true, then all directories are placed at the top of the tree -- otherwise everything is alpha sorted
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
	if se.SaveLangOpts {
		AvailableLanguages.SaveSettings()
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
	if se.SaveLangOpts {
		AvailableLanguages.OpenSettings()
	}
	AvailableSplits.OpenSettings()
	AvailableRegisters.OpenSettings()
	return err
}

// Apply settings updates things according with settings
func (se *SettingsData) Apply() { //types:add
}

// SetGoMod applies the given gomod setting, setting the GO111MODULE env var
func SetGoMod(gomod bool) {
	if gomod {
		os.Setenv("GO111MODULE", "on")
	} else {
		os.Setenv("GO111MODULE", "off")
	}
}

func (se *SettingsData) MakeToolbar(p *tree.Plan) {
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(se.EditLangOpts).SetIcon(icons.Subtitles)
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(se.EditCmds).SetIcon(icons.KeyboardCommandKey)
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(se.EditSplits).SetIcon(icons.VerticalSplit)
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(se.EditRegisters).SetIcon(icons.Variables)
	})
}

// EditLangOpts opens the LangsView editor to customize options for each type of
// language / data / file type.
func (se *SettingsData) EditLangOpts() { //types:add
	se.SaveLangOpts = true
	LanguagesView(&AvailableLanguages)
}

// EditCmds opens the editor of the commands of the Command menu, which
// are shared by the apps (see [commands.Default]), with their macros.
func (se *SettingsData) EditCmds() { //types:add
	commands.Default().Edit(nil)
}

// EditSplits opens the SplitsView editor to customize saved splitter settings
func (se *SettingsData) EditSplits() { //types:add
	SplitsView(&AvailableSplits)
}

// EditRegisters opens the RegistersView editor to customize saved registers
func (se *SettingsData) EditRegisters() { //types:add
	RegistersView(&AvailableRegisters)
}

////////   Project Settings

// ProjectSettings are the settings for saving for a project. This IS the project file
type ProjectSettings struct { //types:add

	// file picker settings
	Files FileSettings

	// editor settings
	Editor text.EditorSettings `display:"inline"`

	// current named-split config in use for configuring the splitters
	SplitName SplitName

	// the language associated with the most frequently encountered file
	// extension in the file tree -- can be manually set here as well
	MainLang fileinfo.Known

	// backend that provides language services (diagnostics, completion, hover, navigation, symbols) for this project -- LSP uses external language servers such as gopls, except for languages set to use the parser in the Languages settings (Markdown and TeX by default)
	LangBackend LangBackends

	// if set, language servers that support inlay hints (e.g. gopls) are configured to return them, and they are shown inline in the editor, such as parameter names and variable types; running servers pick the change up immediately through dynamic reconfiguration
	InlayHints bool

	// problem messages to ignore: the problems (diagnostics) from language servers whose message contains a key whose value is true are not shown, in the Problems tab or in the editor; commonly ignored Go warnings are listed, disabled, so that they are easy to enable
	IgnoreProblems map[string]bool

	// if set, info and hint problems from language servers (e.g., gopls modernize suggestions) are shown in the editor and the Problems tab, in addition to errors and warnings
	ShowHints bool

	// the type of version control system used in this project (git, svn, etc).
	// filters commands available
	VersionControl vcs.Types

	// current project filename for saving / loading specific Code
	// configuration information in a .code file (optional)
	ProjectFilename core.Filename `extension:".code"`

	// root directory for the project. all projects must be organized within
	// a top-level root directory, with all the files therein constituting
	// the scope of the project. By default it is the path for ProjectFilename
	ProjectRoot core.Filename

	// if true, use Go modules, otherwise use GOPATH -- this sets your effective GO111MODULE environment variable accordingly, dynamically -- updated by toolbar checkbox, dynamically
	GoMod bool

	// BuildCommand is the command line that the Build button runs, in the
	// directory of the file of the active editor (or the project root),
	// with its output in a tab. If it is empty, it is the default
	// for the MainLang: go build -v for Go, latexmk -pdf for TeX, and
	// otherwise make.
	BuildCommand string

	// RunExec is the executable that the debugger runs, and the Run button
	// runs by default for a project that is not in Go or TeX.
	RunExec core.Filename

	// RunCommand is the command line that the Run button runs, in the
	// directory of the file of the active editor (or the project root),
	// with its output in a tab. If it is empty, it is the default
	// for the MainLang: core run for Go, and otherwise the RunExec.
	RunCommand string

	// custom debugger parameters for this project
	Debug cdebug.Params

	// saved find params
	Find FindParams `display:"-"`

	// saved structure params
	Symbols SymbolsParams `display:"-"`

	// saved scope of the problems shown in the Problems tab
	ProblemsLoc Locations `display:"-"`

	// directory properties
	Dirs filetree.DirFlagMap `display:"-"`

	// last register used
	Register RegisterName `display:"-"`

	// current splitter splits
	Splits [4]float32 `display:"-"`

	// current tabUnder setting for splits
	TabsUnder bool `display:"-"`
}

// StandardIgnoreProblems are commonly ignored Go warnings, which are
// added to [ProjectSettings.IgnoreProblems], disabled, so that they are
// easy to enable.
var StandardIgnoreProblems = []string{
	"struct literal uses unkeyed fields",
	"copies lock value",
	"passes lock by value",
	"non-constant format string",
	"error strings should not be capitalized",
	"error strings should not end with punctuation",
	"should not use dot imports",
	"methods on the same type should have the same receiver name",
	"receiver name should be",
	"could use tagged switch",
	"could remove embedded field",
	"should merge variable declaration with assignment",
	"unused parameter",

	// hints, shown with [ProjectSettings.ShowHints]:
	"can be modernized",
	"interface{} can be replaced by any",
	"can be simplified using",
	"ReplaceAll instead",
	"could apply De Morgan's law",
	"could lift into loop condition",
	"could merge conditional assignment",
	"could eliminate this type assertion",
}

// addStandardIgnores adds the [StandardIgnoreProblems] that are not in
// IgnoreProblems to it, disabled.
func (se *ProjectSettings) addStandardIgnores() {
	if se.IgnoreProblems == nil {
		se.IgnoreProblems = make(map[string]bool)
	}
	for _, ig := range StandardIgnoreProblems {
		if _, has := se.IgnoreProblems[ig]; !has {
			se.IgnoreProblems[ig] = false
		}
	}
}

// Open open from file
func (se *ProjectSettings) Open(filename core.Filename) error { //types:add
	err := errors.Log(tomlx.Open(se, string(filename)))
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

////////   Saved Projects / Paths

var (
	// RecentPaths is a slice of recent file paths
	RecentPaths core.FilePaths

	// SavedPathsFilename is the name of the saved file paths file in Cogent Code data directory
	SavedPathsFilename = "saved-paths.json"
)

// SavePaths saves the active SavedPaths to settings dir
func SavePaths() {
	pdir := core.TheApp.AppDataDir()
	pnm := filepath.Join(pdir, SavedPathsFilename)
	RecentPaths.Save(pnm)
}

// OpenPaths loads the active SavedPaths from settings dir
func OpenPaths() {
	pdir := core.TheApp.AppDataDir()
	pnm := filepath.Join(pdir, SavedPathsFilename)
	RecentPaths.Open(pnm)
}

//////// Code

// Defaults sets new project defaults based on overall settings
func (cv *Code) Defaults() {
	cv.Settings.VersionControl = vcs.NoVCS
	cv.Settings.Files = Settings.Files
	cv.Settings.Editor = core.SystemSettings.Editor
	cv.Settings.Splits = [4]float32{.1, .5, .5, .3}
	cv.Settings.TabsUnder = true
	cv.Settings.Debug = cdebug.DefaultParams
}

// GrabSettings grabs the current project preference settings from various
// places, e.g., prior to saving or editing.
func (cv *Code) GrabSettings() {
	sv := cv.Splits()
	copy(cv.Settings.Splits[:], sv.Splits())
	cv.Settings.TabsUnder = len(sv.Tiles) == 2
	cv.Settings.Dirs.Copy(&cv.Files.Dirs)
}

// ApplySettings applies current project preference settings into places where
// they are used -- only for those done prior to loading
func (cv *Code) ApplySettings() {
	cv.ProjectFilename = cv.Settings.ProjectFilename
	cv.ProjectRoot = cv.Settings.ProjectRoot
	if cv.Files != nil {
		cv.Files.Dirs.Copy(&cv.Settings.Dirs)
		cv.Files.DirsOnTop = cv.Settings.Files.DirsOnTop
	}
	if len(cv.Children) > 0 {
		for i := range NTextEditors {
			tv := cv.EditorByIndex(i)
			if tv.Lines != nil {
				cv.ConfigLines(tv.Lines)
			}
		}
		for _, ln := range cv.OpenFiles.Values {
			cv.ConfigLines(ln)
		}
		cv.ApplySplitsSettings(cv.Splits())
	}
	UpdateParserUse(cv)
	cv.Settings.addStandardIgnores()
	if cv.langSvc != nil {
		cv.langSvc.setProblemFilter(cv.Settings.IgnoreProblems, cv.Settings.ShowHints)
		cv.langSvc.applyInlayHints(cv.Settings.InlayHints)
	}
	// core.UpdateAll() // note: this is not necessary and is very slow
}

func (cv *Code) ApplySplitsSettings(sv *core.Splits) {
	if cv.Settings.TabsUnder {
		sv.SetTiles(core.TileSpan, core.TileSecondLong)
	} else {
		sv.SetTiles(core.TileSpan, core.TileSpan, core.TileSpan, core.TileSpan)
	}
	sv.SetSplits(cv.Settings.Splits[:]...)
	sv.Update()
}

// ApplySettingsAction applies current settings to the project, and updates the project
func (cv *Code) ApplySettingsAction() {
	cv.ApplySettings()
	cv.SplitsSetView(cv.Settings.SplitName)
	cv.SetStatus("Applied settings")
}

// EditProjectSettings allows editing of project settings (settings specific to this project)
func (cv *Code) EditProjectSettings() { //types:add
	cv.Settings.addStandardIgnores()
	sv := ProjectSettingsEditor(&cv.Settings)
	if sv != nil {
		sv.OnChange(func(e events.Event) {
			cv.ApplySettingsAction()
		})
	}
}

func (cv *Code) CallSplitsSetView(ctx core.Widget) {
	fb := core.NewSoloFuncButton(ctx).SetFunc(cv.SplitsSetView)
	fb.Args[0].SetValue(cv.Settings.SplitName)
	fb.CallFunc()
}

// SplitsSetView sets split view splitters to given named setting
func (cv *Code) SplitsSetView(split SplitName) { //types:add
	sv := cv.Splits()
	sp, _, ok := AvailableSplits.SplitByName(split)
	if ok {
		cv.Settings.Splits = sp.Splits
		cv.Settings.TabsUnder = sp.TabsUnder
		cv.Settings.SplitName = split
		cv.ApplySplitsSettings(sv)
		if !cv.PanelIsOpen(cv.ActiveEditorIndex + TextEditor1Index) {
			cv.SetActiveEditorIndex((cv.ActiveEditorIndex + 1) % 2)
		}
	}
}

// SplitsSave saves current splitter settings to named splitter settings under
// existing name, and saves to settings file
func (cv *Code) SplitsSave(split SplitName) { //types:add
	sv := cv.Splits()
	sp, _, ok := AvailableSplits.SplitByName(split)
	if ok {
		sp.SaveSplits(sv.Splits())
		AvailableSplits.SaveSettings()
	}
}

// SplitsSaveAs saves current splitter settings to new named splitter settings, and
// saves to settings file
func (cv *Code) SplitsSaveAs(name, desc string) { //types:add
	sv := cv.Splits()
	tabUnder := len(sv.Tiles) == 2
	AvailableSplits.Add(name, desc, sv.Splits(), tabUnder)
	AvailableSplits.SaveSettings()
}

// SplitsEdit opens the SplitsView editor to customize saved splitter settings
func (cv *Code) SplitsEdit() { //types:add
	SplitsView(&AvailableSplits)
}

// LangDefaults applies default language settings based on MainLang: the
// run executable (the build and run commands default
// to those of the MainLang when they are empty).
func (cv *Code) LangDefaults() {
	cv.Settings.RunExec = core.Filename(filepath.Join(string(cv.Settings.ProjectRoot), cv.Name))
	cv.guessVersionControl()
}

// buildCommand returns the [ProjectSettings.BuildCommand], or else the
// default for the MainLang.
func (se *ProjectSettings) buildCommand() string {
	if se.BuildCommand != "" {
		return se.BuildCommand
	}
	switch se.MainLang {
	case fileinfo.Go:
		return "go build -v"
	case fileinfo.TeX:
		return "latexmk -pdf"
	}
	return "make"
}

// runCommand returns the [ProjectSettings.RunCommand], or else the default
// for the MainLang: core run for Go, nothing for TeX, and otherwise the
// RunExec, if it is executable.
func (se *ProjectSettings) runCommand() string {
	if se.RunCommand != "" {
		return se.RunCommand
	}
	switch se.MainLang {
	case fileinfo.Go:
		return "core run"
	case fileinfo.TeX:
		return ""
	}
	if se.RunExec != "" && se.RunExecIsExec() {
		return shellQuote(string(se.RunExec))
	}
	return ""
}

// shellQuote returns the given string quoted for a POSIX shell, if needed.
func shellQuote(s string) string {
	if s != "" && !strings.ContainsFunc(s, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("-_./:=,@%+", r))
	}) {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// guessVersionControl sets the version control system, if not set, to that
// of the repository containing the project root, or else the first one within
// it in the file tree, which is not built when a project is first opened:
// this is called again after it is (see [Code.UpdateFiles]).
func (cv *Code) guessVersionControl() {
	if cv.Settings.VersionControl != vcs.NoVCS {
		return
	}
	if _, typ := vcs.DetectRepoRoot(string(cv.Settings.ProjectRoot)); typ != vcs.NoVCS {
		cv.Settings.VersionControl = typ
		return
	}
	if cv.Files == nil {
		return
	}
	if repo, _ := cv.Files.FirstVCS(); repo != nil {
		cv.Settings.VersionControl = repo.Type()
	}
}

// GuessMainLang guesses the main language in the project -- returns true if successful
func (cv *Code) GuessMainLang() bool {
	ecsc := cv.Files.FileExtensionCounts(fileinfo.Code)
	ecsd := cv.Files.FileExtensionCounts(fileinfo.Doc)
	ecs := append(ecsc, ecsd...)
	filetree.NodeNameCountSort(ecs)
	for _, ec := range ecs {
		ls := fileinfo.ExtKnown(ec.Name)
		if ls != fileinfo.Unknown {
			cv.Settings.MainLang = ls
			return true
		}
	}
	return false
}
