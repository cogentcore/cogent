// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

//go:generate core generate

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/vcs"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/filetree"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/spell"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/texteditor"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/views"
)

//go:embed icons/*.svg
var Icons embed.FS

func init() {
	icons.AddFS(errors.Log1(fs.Sub(Icons, "icons")))
}

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
	OpenNodes OpenNodes `json:"-"`

	// the command buffers for commands run in this project
	CmdBufs map[string]*texteditor.Buffer `set:"-" json:"-"`

	// history of commands executed in this session
	CmdHistory CmdNames `set:"-" json:"-"`

	// currently running commands in this project
	RunningCmds CmdRuns `set:"-" json:"-" xml:"-"`

	// current arg var vals
	ArgVals ArgVarVals `set:"-" json:"-" xml:"-"`

	// settings for this project -- this is what is saved in a .code project file
	Settings ProjectSettings `set:"-"`

	// current debug view
	CurDbg *DebugView `set:"-"`

	// first key in sequence if needs2 key pressed
	KeySeq1 key.Chord `set:"-"`

	// mutex for protecting overall updates to CodeView
	UpdateMu sync.Mutex `set:"-"`
}

func init() {
	// core.URLHandler = URLHandler
	// paint.TextLinkHandler = TextLinkHandler
}

func (cv *CodeView) OnInit() {
	cv.Frame.OnInit()
	cv.Style(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
	})

	cv.AddCloseDialog()
	cv.OnFirst(events.KeyChord, cv.codeViewKeys)
	cv.On(events.OSOpenFiles, func(e events.Event) {
		ofe := e.(*events.OSFiles)
		for _, fn := range ofe.Files {
			cv.OpenFile(fn)
		}
	})
	cv.OnShow(func(e events.Event) {
		cv.OpenConsoleTab()
		cv.UpdateFiles()
	})

	core.AddChildAt(cv, "splits", func(w *core.Splits) {
		w.SetSplits(cv.Settings.Splits...)
		core.AddChildAt(w, "filetree", func(w *core.Frame) {
			w.Style(func(s *styles.Style) {
				s.Direction = styles.Column
				s.Overflow.Set(styles.OverflowAuto)
			})
			core.AddChildAt(w, "filetree", func(w *filetree.Tree) {
				w.OpenDepth = 4
				cv.Files = w
				w.FileNodeType = FileNodeType

				w.OnSelect(func(e events.Event) {
					e.SetHandled()
					sn := cv.SelectedFileNode()
					if sn != nil {
						cv.FileNodeSelected(sn)
					}
				})
			})
		})
		w.Maker(func(p *core.Plan) {
			for i := 0; i < NTextEditors; i++ {
				cv.makeTextEditor(p, i)
			}
		})
		core.AddChildAt(w, "tabs", func(w *core.Tabs) {
			w.SetType(core.FunctionalTabs)
			w.Style(func(s *styles.Style) {
				s.Grow.Set(1, 1)
			})
		})
	})
	core.AddChildAt(cv, "statusbar", func(w *core.Frame) {
		w.Style(func(s *styles.Style) {
			s.Grow.Set(1, 0)
			s.Min.Y.Em(1.0)
			s.Padding.Set(units.Dp(4))
		})
		core.AddChildAt(w, "sb-text", func(w *core.Text) {
			w.SetText("Welcome to Cogent Code!" + strings.Repeat(" ", 80))
			w.Style(func(s *styles.Style) {
				s.Min.X.Ch(100)
				s.Min.Y.Em(1.0)
				s.Text.TabSize = 4
			})
		})
	})
	// todo: need this still:
	// mtab.OnChange(func(e events.Event) {
	// todo: need to monitor deleted
	// gee.TabDeleted(data.(string))
	// if data == "Find" {
	// 	ge.ActiveTextEditor().ClearHighlights()
	// }
	// })
}

func (cv *CodeView) makeTextEditor(p *core.Plan, i int) {
	txnm := fmt.Sprintf("%d", i)
	core.AddAt(p, "textframe-"+txnm, func(w *core.Frame) {
		w.Style(func(s *styles.Style) {
			s.Direction = styles.Column
			s.Grow.Set(1, 1)
		})
		core.AddChildAt(w, "textbut-"+txnm, func(w *core.Button) {
			w.SetText("texteditor: " + txnm)
			w.Type = core.ButtonAction
			w.Style(func(s *styles.Style) {
				s.Grow.Set(1, 0)
			})
			w.Menu = func(m *core.Scene) {
				cv.TextEditorButtonMenu(i, m)
			}
			w.OnClick(func(e events.Event) {
				cv.SetActiveTextEditorIndex(i)
			})
			// todo: update
			// ge.UpdateTextButtons()
		})
		core.AddChildAt(w, "texteditor-"+txnm, func(w *TextEditor) {
			w.Code = cv
			ConfigEditorTextEditor(&w.Editor)
			w.OnFocus(func(e events.Event) {
				cv.ActiveTextEditorIndex = i
			})
			// get updates on cursor movement and qreplace
			w.OnInput(func(e events.Event) {
				cv.UpdateStatusText()
			})
		})
	})
}

// ParentCode returns the Code parent of given node
func ParentCode(tn tree.Node) (*CodeView, bool) {
	var res *CodeView
	tn.WalkUp(func(n tree.Node) bool {
		if c, ok := n.This().(*CodeView); ok {
			res = c
			return false
		}
		return true
	})
	return res, res != nil
}

// NTextEditors is the number of text views to create -- to keep things simple
// and consistent (e.g., splitter settings always have the same number of
// values), we fix this degree of freedom, and have flexibility in the
// splitter settings for what to actually show.
const NTextEditors = 2

// These are then the fixed indices of the different elements in the splitview
const (
	FileTreeIndex = iota
	TextEditor1Index
	TextEditor2Index
	TabsIndex
)

// Splits returns the main Splits
func (cv *CodeView) Splits() *core.Splits {
	return cv.ChildByName("splits", 2).(*core.Splits)
}

// TextEditorButtonByIndex returns the top texteditor menu button by index (0 or 1)
func (cv *CodeView) TextEditorButtonByIndex(idx int) *core.Button {
	return cv.Splits().Child(TextEditor1Index + idx).Child(0).(*core.Button)
}

// TextEditorByIndex returns the TextEditor by index (0 or 1), nil if not found
func (cv *CodeView) TextEditorByIndex(idx int) *TextEditor {
	return cv.Splits().Child(TextEditor1Index + idx).Child(1).(*TextEditor)
}

// Tabs returns the main TabView
func (cv *CodeView) Tabs() *core.Tabs {
	return cv.Splits().Child(TabsIndex).(*core.Tabs)
}

// StatusBar returns the statusbar widget
func (cv *CodeView) StatusBar() *core.Frame {
	if cv.This() == nil || !cv.HasChildren() {
		return nil
	}
	return cv.ChildByName("statusbar", 2).(*core.Frame)
}

// StatusText returns the status bar text widget
func (cv *CodeView) StatusText() *core.Text {
	return cv.StatusBar().Child(0).(*core.Text)
}

// SelectedFileNode returns currently selected file tree node as a *filetree.Node
// could be nil.
func (cv *CodeView) SelectedFileNode() *filetree.Node {
	n := len(cv.Files.SelectedNodes)
	if n == 0 {
		return nil
	}
	return filetree.AsNode(cv.Files.SelectedNodes[n-1].This())
}

// VersionControl returns the version control system in effect, using the file tree detected
// version or whatever is set in project settings
func (cv *CodeView) VersionControl() filetree.VersionControlName {
	vc := cv.Settings.VersionControl
	return vc
}

func (cv *CodeView) FocusOnTabs() bool {
	return cv.FocusOnPanel(TabsIndex)
}

////////////////////////////////////////////////////////
//  Main project API

// UpdateFiles updates the list of files saved in project
func (cv *CodeView) UpdateFiles() { //types:add
	if cv.Files != nil && cv.ProjectRoot != "" {
		cv.Files.OpenPath(string(cv.ProjectRoot))
		cv.Files.Open()
	}
}

func (cv *CodeView) IsEmpty() bool {
	return cv.ProjectRoot == ""
}

// OpenRecent opens a recently used file
func (cv *CodeView) OpenRecent(filename core.Filename) { //types:add
	ext := strings.ToLower(filepath.Ext(string(filename)))
	if ext == ".code" {
		cv.OpenProject(filename)
	} else {
		cv.OpenPath(filename)
	}
}

// EditRecentPaths opens a dialog editor for editing the recent project paths list
func (cv *CodeView) EditRecentPaths() {
	d := core.NewBody().AddTitle("Recent project paths").
		AddText("You can delete paths you no longer use")
	views.NewSliceView(d).SetSlice(&RecentPaths)
	d.AddOKOnly().RunDialog(cv)
}

// OpenFile opens file in an open project if it has the same path as the file
// or in a new window.
func (cv *CodeView) OpenFile(fnm string) { //types:add
	abfn, _ := filepath.Abs(fnm)
	if strings.HasPrefix(abfn, string(cv.ProjectRoot)) {
		cv.ViewFile(core.Filename(abfn))
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
	cv.OpenPath(core.Filename(abfn))
}

// SetWindowNameTitle sets the window name and title based on current project name
func (cv *CodeView) SetWindowNameTitle() {
	pnm := cv.Name()
	title := "Cogent Code • " + pnm
	cv.Scene.Body.SetTitle(title)
}

// OpenPath creates a new project by opening given path, which can either be a
// specific file or a folder containing multiple files of interest -- opens in
// current CodeView object if it is empty, or otherwise opens a new window.
func (cv *CodeView) OpenPath(path core.Filename) *CodeView { //types:add
	if gproj, has := CheckForProjectAtPath(string(path)); has {
		return cv.OpenProject(core.Filename(gproj))
	}
	if !cv.IsEmpty() {
		return NewCodeProjectPath(string(path))
	}
	cv.Defaults()
	root, pnm, fnm, ok := ProjectPathParse(string(path))
	if ok {
		os.Chdir(root)
		RecentPaths.AddPath(root, core.SystemSettings.SavedPathsMax)
		SavePaths()
		cv.ProjectRoot = core.Filename(root)
		cv.SetName(pnm)
		cv.Scene.SetName(pnm)
		cv.Settings.ProjectFilename = core.Filename(filepath.Join(root, pnm+".code"))
		cv.ProjectFilename = cv.Settings.ProjectFilename
		cv.Settings.ProjectRoot = cv.ProjectRoot
		cv.GuessMainLang()
		cv.LangDefaults()
		cv.SetWindowNameTitle()
		cv.UpdateFiles()
		cv.SplitsSetView(SplitName(AvailableSplitNames[0]))
		if fnm != "" {
			cv.NextViewFile(core.Filename(fnm))
		}
	}
	return cv
}

// OpenProject opens .code project file and its settings from given filename, in a standard
// toml-formatted file
func (cv *CodeView) OpenProject(filename core.Filename) *CodeView { //types:add
	if !cv.IsEmpty() {
		return OpenCodeProject(string(filename))
	}
	cv.Defaults()
	if err := cv.Settings.Open(filename); err != nil {
		slog.Error("Project Settings had a loading error", "error", err)
	}
	root, pnm, _, ok := ProjectPathParse(string(filename))
	cv.Settings.ProjectRoot = core.Filename(root)
	if cv.Settings.MainLang == fileinfo.Unknown {
		cv.GuessMainLang()
	}
	cv.Settings.ProjectFilename = filename // should already be set but..
	if ok {
		SetGoMod(cv.Settings.GoMod)
		os.Chdir(string(cv.Settings.ProjectRoot))
		cv.ProjectRoot = core.Filename(cv.Settings.ProjectRoot)
		RecentPaths.AddPath(string(filename), core.SystemSettings.SavedPathsMax)
		SavePaths()
		cv.SetName(pnm)
		cv.Scene.SetName(pnm)
		cv.ApplySettings()
		cv.UpdateFiles()
		cv.SetWindowNameTitle()
	}
	return cv
}

// NewProject creates a new project at given path, making a new folder in that
// path -- all CodeView projects are essentially defined by a path to a folder
// containing files.  If the folder already exists, then use OpenPath.
// Can also specify main language and version control type
func (cv *CodeView) NewProject(path core.Filename, folder string, mainLang fileinfo.Known, VersionControl filetree.VersionControlName) *CodeView { //types:add
	np := filepath.Join(string(path), folder)
	err := os.MkdirAll(np, 0775)
	if err != nil {
		core.MessageDialog(cv, fmt.Sprintf("Could not make folder for project at: %v, err: %v", np, err), "Could not Make Folder")
		return nil
	}
	nge := cv.OpenPath(core.Filename(np))
	nge.Settings.MainLang = mainLang
	if VersionControl != "" {
		nge.Settings.VersionControl = VersionControl
	}
	return nge
}

// NewFile creates a new file in the project
func (cv *CodeView) NewFile(filename string, addToVcs bool) { //types:add
	np := filepath.Join(string(cv.ProjectRoot), filename)
	_, err := os.Create(np)
	if err != nil {
		core.MessageDialog(cv, fmt.Sprintf("Could not make new file at: %v, err: %v", np, err), "Could not Make File")
		return
	}
	cv.Files.UpdatePath(np)
	if addToVcs {
		nfn, ok := cv.Files.FindFile(np)
		if ok {
			nfn.AddToVCS()
		}
	}
}

// SaveProject saves project file containing custom project settings, in a
// standard toml-formatted file
func (cv *CodeView) SaveProject() { //types:add
	if cv.Settings.ProjectFilename == "" {
		return
	}
	cv.SaveProjectAs(cv.Settings.ProjectFilename)
	cv.SaveAllCheck(false, nil) // false = no cancel option
}

// SaveProjectIfExists saves project file containing custom project settings, in a
// standard toml-formatted file, only if it already exists -- returns true if saved
// saveAllFiles indicates if user should be prompted for saving all files
func (cv *CodeView) SaveProjectIfExists(saveAllFiles bool) bool {
	spell.SaveIfLearn()
	if cv.Settings.ProjectFilename == "" {
		return false
	}
	if _, err := os.Stat(string(cv.Settings.ProjectFilename)); os.IsNotExist(err) {
		return false // does not exist
	}
	cv.SaveProjectAs(cv.Settings.ProjectFilename)
	if saveAllFiles {
		cv.SaveAllCheck(false, nil)
	}
	return true
}

// SaveProjectAs saves project custom settings to given filename, in a standard
// toml-formatted file
// saveAllFiles indicates if user should be prompted for saving all files
// returns true if the user was prompted, false otherwise
func (cv *CodeView) SaveProjectAs(filename core.Filename) bool { //types:add
	spell.SaveIfLearn()
	RecentPaths.AddPath(string(filename), core.SystemSettings.SavedPathsMax)
	SavePaths()
	cv.Settings.ProjectFilename = filename
	cv.ProjectFilename = cv.Settings.ProjectFilename
	cv.GrabSettings()
	cv.Settings.Save(filename)
	cv.Files.UpdatePath(string(filename))
	cv.Changed = false
	return false
}

// SaveAllCheck -- check if any files have not been saved, and prompt to save them
// returns true if there were unsaved files, false otherwise.
// cancelOpt presents an option to cancel current command, in which case function is not called.
// if function is passed, then it is called in all cases except if the user selects cancel.
func (cv *CodeView) SaveAllCheck(cancelOpt bool, fun func()) bool {
	nch := cv.NChangedFiles()
	if nch == 0 {
		if fun != nil {
			fun()
		}
		return false
	}
	d := core.NewBody().AddTitle("There are Unsaved Files").
		AddText(fmt.Sprintf("In Project: %v There are <b>%v</b> opened files with <b>unsaved changes</b> -- do you want to save all?", cv.Nm, nch))
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
			cv.SaveAllOpenNodes()
			if fun != nil {
				fun()
			}
		})
	})
	d.RunDialog(cv)
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
		emsg := fmt.Errorf("ProjectPathParse: Cannot open at given path: %q: Error: %v", effpath, err)
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
func (cv *CodeView) NChangedFiles() int {
	return cv.OpenNodes.NChanged()
}

// AddCloseDialog adds the close dialog that automatically saves the project
// and prompts the user to save open files when they try to close the scene
// containing this code view.
func (cv *CodeView) AddCloseDialog() {
	cv.WidgetBase.AddCloseDialog(func(d *core.Body) bool {
		cv.SaveProjectIfExists(false) // don't prompt here, as we will do it now..
		nch := cv.NChangedFiles()
		if nch == 0 {
			return false
		}
		d.AddTitle("Unsaved files").
			AddText(fmt.Sprintf("There are %d open files in %s with unsaved changes", nch, cv.Nm))
		d.AddBottomBar(func(parent core.Widget) {
			d.AddOK(parent).SetText("Close without saving").OnClick(func(e events.Event) {
				cv.Scene.Close()
			})
			core.NewButton(parent).SetText("Save and close").OnClick(func(e events.Event) {
				cv.SaveAllOpenNodes()
				cv.Scene.Close()
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
	pp := &ProjectSettings{}
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
		cv := CodeInScene(sc)
		if cv != nil && string(cv.ProjectRoot) == root {
			win.Raise()
			return cv
		}
	}

	b := core.NewBody(winm).SetTitle(winm)

	cv := NewCodeView(b)
	core.TheApp.AppBarConfig = cv.AppBarConfig
	b.AddAppBar(cv.MakeToolbar)

	b.RunWindow()

	if doPath {
		cv.OpenPath(core.Filename(path))
	} else {
		cv.OpenProject(core.Filename(path))
	}

	return cv
}
