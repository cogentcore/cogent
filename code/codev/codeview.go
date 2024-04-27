// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package codev

//go:generate core generate

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"cogentcore.org/cogent/code/code"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/fileinfo"
	"cogentcore.org/core/filetree"
	"cogentcore.org/core/gox/errors"
	"cogentcore.org/core/spell"
	"cogentcore.org/core/texteditor"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/vcs"
	"cogentcore.org/core/views"
)

// CodeView is the core editor and tab viewer framework for the Code system.  The
// default view has a tree browser of files on the left, editor panels in the
// middle, and a tabbed viewer on the right.
type CodeView struct {
	core.Frame

	// root directory for the project -- all projects must be organized within a top-level root directory, with all the files therein constituting the scope of the project -- by default it is the path for ProjectFilename
	ProjectRoot core.Filename

	// current project filename for saving / loading specific Code configuration information in a .code file (optional)
	ProjectFilename core.Filename `ext:".code"`

	// filename of the currently active texteditor
	ActiveFilename core.Filename `set:"-"`

	// language for current active filename
	ActiveLang fileinfo.Known

	// VCS repo for current active filename
	ActiveVCS vcs.Repo `set:"-"`

	// VCS info for current active filename (typically branch or revision) -- for status
	ActiveVCSInfo string `set:"-"`

	// has the root changed?  we receive update signals from root for changes
	Changed bool `set:"-" json:"-"`

	// the last status update message
	StatusMessage string

	// timestamp for when a file was last saved -- provides dirty state for various updates including rebuilding in debugger
	LastSaveTStamp time.Time `set:"-" json:"-"`

	// all the files in the project directory and subdirectories
	Files *filetree.Tree `set:"-" json:"-"`

	// index of the currently active texteditor -- new files will be viewed in other views if available
	ActiveTextEditorIndex int `set:"-" json:"-"`

	// list of open nodes, most recent first
	OpenNodes code.OpenNodes `json:"-"`

	// the command buffers for commands run in this project
	CmdBufs map[string]*texteditor.Buffer `set:"-" json:"-"`

	// history of commands executed in this session
	CmdHistory code.CmdNames `set:"-" json:"-"`

	// currently running commands in this project
	RunningCmds code.CmdRuns `set:"-" json:"-" xml:"-"`

	// current arg var vals
	ArgVals code.ArgVarVals `set:"-" json:"-" xml:"-"`

	// settings for this project -- this is what is saved in a .code project file
	Settings code.ProjectSettings `set:"-"`

	// current debug view
	CurDbg *code.DebugView `set:"-"`

	// first key in sequence if needs2 key pressed
	KeySeq1 key.Chord `set:"-"`

	// mutex for protecting overall updates to CodeView
	UpdateMu sync.Mutex `set:"-"`
}

func init() {
	// core.URLHandler = URLHandler
	// paint.TextLinkHandler = TextLinkHandler
}

func (ge *CodeView) OnInit() {
	ge.Frame.OnInit()
	ge.HandleEvents()
}

func (ge *CodeView) OnAdd() {
	ge.Frame.OnAdd()
	ge.AddCloseDialog()
}

////////////////////////////////////////////////////////
// Code interface

func (ge *CodeView) ProjectSettings() *code.ProjectSettings {
	return &ge.Settings
}

func (ge *CodeView) FileTree() *filetree.Tree {
	return ge.Files
}

func (ge *CodeView) LastSaveTime() time.Time {
	return ge.LastSaveTStamp
}

// VersionControl returns the version control system in effect, using the file tree detected
// version or whatever is set in project settings
func (ge *CodeView) VersionControl() filetree.VersionControlName {
	vc := ge.Settings.VersionControl
	return vc
}

func (ge *CodeView) CmdRuns() *code.CmdRuns {
	return &ge.RunningCmds
}

func (ge *CodeView) CmdHist() *code.CmdNames {
	return &ge.CmdHistory
}

func (ge *CodeView) ArgVarVals() *code.ArgVarVals {
	return &ge.ArgVals
}

func (ge *CodeView) CurOpenNodes() *code.OpenNodes {
	return &ge.OpenNodes
}

func (ge *CodeView) FocusOnTabs() bool {
	return ge.FocusOnPanel(TabsIndex)
}

////////////////////////////////////////////////////////
//  Main project API

// UpdateFiles updates the list of files saved in project
func (ge *CodeView) UpdateFiles() { //types:add
	if ge.Files != nil && ge.ProjectRoot != "" {
		ge.Files.OpenPath(string(ge.ProjectRoot))
		ge.Files.Open()
	}
}

func (ge *CodeView) IsEmpty() bool {
	return ge.ProjectRoot == ""
}

// OpenRecent opens a recently used file
func (ge *CodeView) OpenRecent(filename core.Filename) { //types:add
	ext := strings.ToLower(filepath.Ext(string(filename)))
	if ext == ".code" {
		ge.OpenProject(filename)
	} else {
		ge.OpenPath(filename)
	}
}

// EditRecentPaths opens a dialog editor for editing the recent project paths list
func (ge *CodeView) EditRecentPaths() {
	d := core.NewBody().AddTitle("Recent project paths").
		AddText("You can delete paths you no longer use")
	views.NewSliceView(d).SetSlice(&code.RecentPaths)
	d.AddOKOnly().RunDialog(ge)
}

// OpenFile opens file in an open project if it has the same path as the file
// or in a new window.
func (ge *CodeView) OpenFile(fnm string) { //types:add
	abfn, _ := filepath.Abs(fnm)
	if strings.HasPrefix(abfn, string(ge.ProjectRoot)) {
		ge.ViewFile(core.Filename(abfn))
		return
	}
	for _, win := range core.MainRenderWindows {
		msc := win.MainScene()
		geo := CodeInScene(msc)
		if strings.HasPrefix(abfn, string(geo.ProjectRoot)) {
			geo.ViewFile(core.Filename(abfn))
			return
		}
	}
	// fmt.Printf("open path: %s\n", ge.ProjectRoot)
	ge.OpenPath(core.Filename(abfn))
}

// SetWindowNameTitle sets the window name and title based on current project name
func (ge *CodeView) SetWindowNameTitle() {
	pnm := ge.Name()
	title := "Cogent Code • " + pnm
	ge.Scene.Body.SetTitle(title)
}

// OpenPath creates a new project by opening given path, which can either be a
// specific file or a folder containing multiple files of interest -- opens in
// current CodeView object if it is empty, or otherwise opens a new window.
func (ge *CodeView) OpenPath(path core.Filename) *CodeView { //types:add
	if gproj, has := CheckForProjectAtPath(string(path)); has {
		return ge.OpenProject(core.Filename(gproj))
	}
	if !ge.IsEmpty() {
		return NewCodeProjectPath(string(path))
	}
	ge.Defaults()
	root, pnm, fnm, ok := ProjectPathParse(string(path))
	if ok {
		os.Chdir(root)
		code.RecentPaths.AddPath(root, core.SystemSettings.SavedPathsMax)
		code.SavePaths()
		ge.ProjectRoot = core.Filename(root)
		ge.SetName(pnm)
		ge.Scene.SetName(pnm)
		ge.Settings.ProjectFilename = core.Filename(filepath.Join(root, pnm+".code"))
		ge.ProjectFilename = ge.Settings.ProjectFilename
		ge.Settings.ProjectRoot = ge.ProjectRoot
		ge.GuessMainLang()
		ge.LangDefaults()
		ge.SetWindowNameTitle()
		ge.UpdateFiles()
		ge.SplitsSetView(code.SplitName(code.AvailableSplitNames[0]))
		if fnm != "" {
			ge.NextViewFile(core.Filename(fnm))
		}
	}
	return ge
}

// OpenProject opens .code project file and its settings from given filename, in a standard
// toml-formatted file
func (ge *CodeView) OpenProject(filename core.Filename) *CodeView { //types:add
	if !ge.IsEmpty() {
		return OpenCodeProject(string(filename))
	}
	ge.Defaults()
	if err := ge.Settings.Open(filename); err != nil {
		slog.Error("Project Settings had a loading error", "error", err)
	}
	root, pnm, _, ok := ProjectPathParse(string(filename))
	ge.Settings.ProjectRoot = core.Filename(root)
	if ge.Settings.MainLang == fileinfo.Unknown {
		ge.GuessMainLang()
	}
	ge.Settings.ProjectFilename = filename // should already be set but..
	if ok {
		code.SetGoMod(ge.Settings.GoMod)
		os.Chdir(string(ge.Settings.ProjectRoot))
		ge.ProjectRoot = core.Filename(ge.Settings.ProjectRoot)
		code.RecentPaths.AddPath(string(filename), core.SystemSettings.SavedPathsMax)
		code.SavePaths()
		ge.SetName(pnm)
		ge.Scene.SetName(pnm)
		ge.ApplySettings()
		ge.UpdateFiles()
		ge.SetWindowNameTitle()
	}
	return ge
}

// NewProject creates a new project at given path, making a new folder in that
// path -- all CodeView projects are essentially defined by a path to a folder
// containing files.  If the folder already exists, then use OpenPath.
// Can also specify main language and version control type
func (ge *CodeView) NewProject(path core.Filename, folder string, mainLang fileinfo.Known, VersionControl filetree.VersionControlName) *CodeView { //types:add
	np := filepath.Join(string(path), folder)
	err := os.MkdirAll(np, 0775)
	if err != nil {
		core.MessageDialog(ge, fmt.Sprintf("Could not make folder for project at: %v, err: %v", np, err), "Could not Make Folder")
		return nil
	}
	nge := ge.OpenPath(core.Filename(np))
	nge.Settings.MainLang = mainLang
	if VersionControl != "" {
		nge.Settings.VersionControl = VersionControl
	}
	return nge
}

// NewFile creates a new file in the project
func (ge *CodeView) NewFile(filename string, addToVcs bool) { //types:add
	np := filepath.Join(string(ge.ProjectRoot), filename)
	_, err := os.Create(np)
	if err != nil {
		core.MessageDialog(ge, fmt.Sprintf("Could not make new file at: %v, err: %v", np, err), "Could not Make File")
		return
	}
	ge.Files.UpdatePath(np)
	if addToVcs {
		nfn, ok := ge.Files.FindFile(np)
		if ok {
			nfn.AddToVCS()
		}
	}
}

// SaveProject saves project file containing custom project settings, in a
// standard toml-formatted file
func (ge *CodeView) SaveProject() { //types:add
	if ge.Settings.ProjectFilename == "" {
		return
	}
	ge.SaveProjectAs(ge.Settings.ProjectFilename)
	ge.SaveAllCheck(false, nil) // false = no cancel option
}

// SaveProjectIfExists saves project file containing custom project settings, in a
// standard toml-formatted file, only if it already exists -- returns true if saved
// saveAllFiles indicates if user should be prompted for saving all files
func (ge *CodeView) SaveProjectIfExists(saveAllFiles bool) bool {
	spell.SaveIfLearn()
	if ge.Settings.ProjectFilename == "" {
		return false
	}
	if _, err := os.Stat(string(ge.Settings.ProjectFilename)); os.IsNotExist(err) {
		return false // does not exist
	}
	ge.SaveProjectAs(ge.Settings.ProjectFilename)
	if saveAllFiles {
		ge.SaveAllCheck(false, nil)
	}
	return true
}

// SaveProjectAs saves project custom settings to given filename, in a standard
// toml-formatted file
// saveAllFiles indicates if user should be prompted for saving all files
// returns true if the user was prompted, false otherwise
func (ge *CodeView) SaveProjectAs(filename core.Filename) bool { //types:add
	spell.SaveIfLearn()
	code.RecentPaths.AddPath(string(filename), core.SystemSettings.SavedPathsMax)
	code.SavePaths()
	ge.Settings.ProjectFilename = filename
	ge.ProjectFilename = ge.Settings.ProjectFilename
	ge.GrabSettings()
	ge.Settings.Save(filename)
	ge.Files.UpdatePath(string(filename))
	ge.Changed = false
	return false
}

// SaveAllCheck -- check if any files have not been saved, and prompt to save them
// returns true if there were unsaved files, false otherwise.
// cancelOpt presents an option to cancel current command, in which case function is not called.
// if function is passed, then it is called in all cases except if the user selects cancel.
func (ge *CodeView) SaveAllCheck(cancelOpt bool, fun func()) bool {
	nch := ge.NChangedFiles()
	if nch == 0 {
		if fun != nil {
			fun()
		}
		return false
	}
	d := core.NewBody().AddTitle("There are Unsaved Files").
		AddText(fmt.Sprintf("In Project: %v There are <b>%v</b> opened files with <b>unsaved changes</b> -- do you want to save all?", ge.Nm, nch))
	d.AddBottomBar(func(parent core.Widget) {
		if cancelOpt {
			d.AddCancel(parent).SetText("Cancel Command")
		}
		core.NewButton(parent).SetText("Don't Save").OnClick(func(e events.Event) {
			d.Close()
			if fun != nil {
				fun()
			}
		})
		core.NewButton(parent).SetText("Save All").OnClick(func(e events.Event) {
			d.Close()
			ge.SaveAllOpenNodes()
			if fun != nil {
				fun()
			}
		})
	})
	d.RunDialog(ge)
	return true
}

// ProjectPathParse parses given project path into a root directory (which could
// be the path or just the directory portion of the path, depending in whether
// the path is a directory or not), and a bool if all is good (otherwise error
// message has been reported). projnm is always the last directory of the path.
func ProjectPathParse(path string) (root, projnm, fnm string, ok bool) {
	if path == "" {
		return "", "blank", "", false
	}
	effpath := errors.Log1(filepath.EvalSymlinks(path))
	info, err := os.Lstat(effpath)
	if err != nil {
		emsg := fmt.Errorf("code.ProjectPathParse: Cannot open at given path: %q: Error: %v", effpath, err)
		log.Println(emsg)
		return
	}
	path, _ = filepath.Abs(path)
	dir, fn := filepath.Split(path)
	pathIsDir := info.IsDir()
	if pathIsDir {
		root = path
		projnm = fn
	} else {
		root = filepath.Clean(dir)
		_, projnm = filepath.Split(root)
		fnm = fn
	}
	ok = true
	return
}

// CheckForProjectAtPath checks if there is a .code project at the given path
// returns project path and true if found, otherwise false
func CheckForProjectAtPath(path string) (string, bool) {
	root, pnm, _, ok := ProjectPathParse(path)
	if !ok {
		return "", false
	}
	gproj := filepath.Join(root, pnm+".code")
	if _, err := os.Stat(gproj); os.IsNotExist(err) {
		return "", false // does not exist
	}
	return gproj, true
}

//////////////////////////////////////////////////////////////////////////////////////
//   Close / Quit Req

// NChangedFiles returns number of opened files with unsaved changes
func (ge *CodeView) NChangedFiles() int {
	return ge.OpenNodes.NChanged()
}

// AddCloseDialog adds the close dialog that automatically saves the project
// and prompts the user to save open files when they try to close the scene
// containing this code view.
func (ge *CodeView) AddCloseDialog() {
	ge.WidgetBase.AddCloseDialog(func(d *core.Body) bool {
		ge.SaveProjectIfExists(false) // don't prompt here, as we will do it now..
		nch := ge.NChangedFiles()
		if nch == 0 {
			return false
		}
		d.AddTitle("Unsaved files").
			AddText(fmt.Sprintf("There are %d open files in %s with unsaved changes", nch, ge.Nm))
		d.AddBottomBar(func(parent core.Widget) {
			d.AddOK(parent, "cws").SetText("Close without saving").OnClick(func(e events.Event) {
				ge.Scene.Close()
			})
			core.NewButton(parent, "sa").SetText("Save and close").OnClick(func(e events.Event) {
				ge.SaveAllOpenNodes()
				ge.Scene.Close()
			})
		})
		return true
	})
}

//////////////////////////////////////////////////////////////////////////////////////
//   Project window

// NewCodeProjectPath creates a new CodeView window with a new CodeView project for given
// path, returning the window and the path
func NewCodeProjectPath(path string) *CodeView {
	root, projnm, _, _ := ProjectPathParse(path)
	return NewCodeWindow(path, projnm, root, true)
}

// OpenCodeProject creates a new CodeView window opened to given CodeView project,
// returning the window and the path
func OpenCodeProject(projfile string) *CodeView {
	pp := &code.ProjectSettings{}
	if err := pp.Open(core.Filename(projfile)); err != nil {
		slog.Debug("Project Settings had a loading error", "error", err)
	}
	path := string(pp.ProjectRoot)
	root, projnm, _, _ := ProjectPathParse(path)
	return NewCodeWindow(projfile, projnm, root, false)
}

func CodeInScene(sc *core.Scene) *CodeView {
	gv := sc.Body.ChildByType(CodeViewType, tree.NoEmbeds)
	if gv != nil {
		return gv.(*CodeView)
	}
	return nil
}

// NewCodeWindow is common code for Open CodeWindow from Project or Path
func NewCodeWindow(path, projnm, root string, doPath bool) *CodeView {
	winm := "Cogent Code • " + projnm

	if win, found := core.AllRenderWindows.FindName(winm); found {
		sc := win.MainScene()
		ge := CodeInScene(sc)
		if ge != nil && string(ge.ProjectRoot) == root {
			win.Raise()
			return ge
		}
	}

	b := core.NewBody(winm).SetTitle(winm)

	ge := NewCodeView(b)
	core.TheApp.AppBarConfig = ge.AppBarConfig
	b.AddAppBar(ge.ConfigToolbar)

	b.RunWindow()

	if doPath {
		ge.OpenPath(core.Filename(path))
	} else {
		ge.OpenProject(core.Filename(path))
	}

	return ge
}
