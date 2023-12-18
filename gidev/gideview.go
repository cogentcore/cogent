// Copyright (c) 2023, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gidev

//go:generate goki generate

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"goki.dev/fi"
	"goki.dev/gi/v2/filetree"
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/giv"
	"goki.dev/gi/v2/texteditor"
	"goki.dev/gide/v2/gide"
	"goki.dev/goosi/events"
	"goki.dev/goosi/events/key"
	"goki.dev/grr"
	"goki.dev/ki/v2"
	"goki.dev/spell"
	"goki.dev/vci/v2"
)

// GideView is the core editor and tab viewer framework for the Gide system.  The
// default view has a tree browser of files on the left, editor panels in the
// middle, and a tabbed viewer on the right.
type GideView struct {
	gi.Frame

	// root directory for the project -- all projects must be organized within a top-level root directory, with all the files therein constituting the scope of the project -- by default it is the path for ProjFilename
	ProjRoot gi.FileName

	// current project filename for saving / loading specific Gide configuration information in a .gide file (optional)
	ProjFilename gi.FileName `ext:".gide"`

	// filename of the currently-active textview
	ActiveFilename gi.FileName `set:"-"`

	// language for current active filename
	ActiveLang fi.Known

	// VCS repo for current active filename
	ActiveVCS vci.Repo `set:"-"`

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

	// index of the currently-active textview -- new files will be viewed in other views if available
	ActiveTextEditorIdx int `set:"-" json:"-"`

	// list of open nodes, most recent first
	OpenNodes gide.OpenNodes `json:"-"`

	// the command buffers for commands run in this project
	CmdBufs map[string]*texteditor.Buf `set:"-" json:"-"`

	// history of commands executed in this session
	CmdHistory gide.CmdNames `set:"-" json:"-"`

	// currently running commands in this project
	RunningCmds gide.CmdRuns `set:"-" json:"-" xml:"-"`

	// current arg var vals
	ArgVals gide.ArgVarVals `set:"-" json:"-" xml:"-"`

	// preferences for this project -- this is what is saved in a .gide project file
	Prefs gide.ProjPrefs `set:"-"`

	// current debug view
	CurDbg *gide.DebugView `set:"-"`

	// first key in sequence if needs2 key pressed
	KeySeq1 key.Chord `set:"-"`

	// mutex for protecting overall updates to GideView
	UpdtMu sync.Mutex `set:"-"`
}

func init() {
	// gi.URLHandler = URLHandler
	// paint.TextLinkHandler = TextLinkHandler
}

func (ge *GideView) OnInit() {
	ge.WidgetBase.OnInit()
	ge.Frame.SetStyles()
	ge.HandleEvents()
}

////////////////////////////////////////////////////////
// Gide interface

func (ge *GideView) Scene() *gi.Scene {
	return ge.Sc
}

func (ge *GideView) ProjPrefs() *gide.ProjPrefs {
	return &ge.Prefs
}

func (ge *GideView) FileTree() *filetree.Tree {
	return ge.Files
}

func (ge *GideView) LastSaveTime() time.Time {
	return ge.LastSaveTStamp
}

// VersCtrl returns the version control system in effect, using the file tree detected
// version or whatever is set in project preferences
func (ge *GideView) VersCtrl() filetree.VersCtrlName {
	vc := ge.Prefs.VersCtrl
	return vc
}

func (ge *GideView) CmdRuns() *gide.CmdRuns {
	return &ge.RunningCmds
}

func (ge *GideView) CmdHist() *gide.CmdNames {
	return &ge.CmdHistory
}

func (ge *GideView) ArgVarVals() *gide.ArgVarVals {
	return &ge.ArgVals
}

func (ge *GideView) FocusOnTabs() bool {
	return ge.FocusOnPanel(TabsIdx)
}

////////////////////////////////////////////////////////
//  Main project API

// UpdateFiles updates the list of files saved in project
func (ge *GideView) UpdateFiles() { //gti:add
	if ge.Files != nil && ge.ProjRoot != "" {
		ge.Files.OpenPath(string(ge.ProjRoot))
		ge.Files.Open()
	}
}

func (ge *GideView) IsEmpty() bool {
	return ge.ProjRoot == ""
}

// OpenRecent opens a recently-used file
func (ge *GideView) OpenRecent(filename gi.FileName) { //gti:add
	if string(filename) == gide.GideViewResetRecents {
		gide.SavedPaths = nil
		gi.StringsAddExtras((*[]string)(&gide.SavedPaths), gide.SavedPathsExtras)
	} else if string(filename) == gide.GideViewEditRecents {
		ge.EditRecents()
	} else {
		ext := strings.ToLower(filepath.Ext(string(filename)))
		if ext == ".gide" {
			ge.OpenProj(filename)
		} else {
			ge.OpenPath(filename)
		}
	}
}

// RecentsEdit opens a dialog editor for deleting from the recents project list
func (ge *GideView) EditRecents() {
	tmp := make([]string, len(gide.SavedPaths))
	copy(tmp, gide.SavedPaths)
	gi.StringsRemoveExtras((*[]string)(&tmp), gide.SavedPathsExtras)
	d := gi.NewBody().AddTitle("Recent Project Paths").
		AddText("Delete paths you no longer use")
	giv.NewSliceView(d).SetSlice(tmp)
	d.AddBottomBar(func(pw gi.Widget) {
		d.AddOk(pw).OnClick(func(e events.Event) {
			gide.SavedPaths = tmp
			gi.StringsAddExtras((*[]string)(&gide.SavedPaths), gide.SavedPathsExtras)
		})
	})
	d.NewDialog(ge).Run()
}

// OpenFile opens file in an open project if it has the same path as the file
// or in a new window.
func (ge *GideView) OpenFile(fnm string) { //gti:add
	abfn, _ := filepath.Abs(fnm)
	if strings.HasPrefix(abfn, string(ge.ProjRoot)) {
		ge.ViewFile(gi.FileName(abfn))
		return
	}
	for _, win := range gi.MainRenderWins {
		msc := win.MainScene()
		geo := GideInScene(msc)
		if strings.HasPrefix(abfn, string(geo.ProjRoot)) {
			geo.ViewFile(gi.FileName(abfn))
			return
		}
	}
	// fmt.Printf("open path: %s\n", ge.ProjRoot)
	ge.OpenPath(gi.FileName(abfn))
}

// SetWindowNameTitle sets the window name and title based on current project name
func (ge *GideView) SetWindowNameTitle() {
	win := ge.Sc.RenderWin()
	if win == nil {
		return
	}
	pnm := ge.Name()
	winm := "gide-" + pnm
	win.SetName(winm)
	win.SetTitle(winm + ": " + string(ge.Prefs.ProjRoot))
	tab := ge.Sc.GetTopAppBar()
	if tab != nil {
		aci := tab.ChildByName("app-chooser", 1)
		if aci != nil {
			ac := aci.(*gi.AppChooser)
			ac.ItemsFunc()
			ac.SetCurIndex(0)
		}
	}
}

// OpenPath creates a new project by opening given path, which can either be a
// specific file or a folder containing multiple files of interest -- opens in
// current GideView object if it is empty, or otherwise opens a new window.
func (ge *GideView) OpenPath(path gi.FileName) *GideView { //gti:add
	if gproj, has := CheckForProjAtPath(string(path)); has {
		return ge.OpenProj(gi.FileName(gproj))
	}
	if !ge.IsEmpty() {
		return NewGideProjPath(string(path))
	}
	ge.Defaults()
	root, pnm, fnm, ok := ProjPathParse(string(path))
	if ok {
		os.Chdir(root)
		gide.SavedPaths.AddPath(root, gi.Prefs.Params.SavedPathsMax)
		gide.SavePaths()
		ge.ProjRoot = gi.FileName(root)
		ge.SetName(pnm)
		ge.Sc.SetName(pnm)
		ge.Prefs.ProjFilename = gi.FileName(filepath.Join(root, pnm+".gide"))
		ge.ProjFilename = ge.Prefs.ProjFilename
		ge.Prefs.ProjRoot = ge.ProjRoot
		ge.GuessMainLang()
		ge.LangDefaults()
		ge.SetWindowNameTitle()
		ge.UpdateFiles()
		ge.SplitsSetView(gide.SplitName(gide.AvailSplitNames[0]))
		if fnm != "" {
			ge.NextViewFile(gi.FileName(fnm))
		}
	}
	return ge
}

// OpenProj opens .gide project file and its settings from given filename, in a standard
// toml-formatted file
func (ge *GideView) OpenProj(filename gi.FileName) *GideView { //gti:add
	if !ge.IsEmpty() {
		return OpenGideProj(string(filename))
	}
	ge.Defaults()
	if err := ge.Prefs.Open(filename); err != nil {
		slog.Error("Project Prefs had a loading error", "error", err)
		if ge.Prefs.ProjRoot == "" {
			root, _, _, _ := ProjPathParse(string(filename))
			ge.Prefs.ProjRoot = gi.FileName(root)
			ge.GuessMainLang()
		}
	}
	ge.Prefs.ProjFilename = filename // should already be set but..
	_, pnm, _, ok := ProjPathParse(string(ge.Prefs.ProjRoot))
	if ok {
		gide.SetGoMod(ge.Prefs.GoMod)
		os.Chdir(string(ge.Prefs.ProjRoot))
		ge.ProjRoot = gi.FileName(ge.Prefs.ProjRoot)
		gide.SavedPaths.AddPath(string(filename), gi.Prefs.Params.SavedPathsMax)
		gide.SavePaths()
		ge.SetName(pnm)
		ge.Sc.SetName(pnm)
		ge.ApplyPrefs()
		ge.UpdateFiles()
		ge.SetWindowNameTitle()
	}
	return ge
}

// NewProj creates a new project at given path, making a new folder in that
// path -- all GideView projects are essentially defined by a path to a folder
// containing files.  If the folder already exists, then use OpenPath.
// Can also specify main language and version control type
func (ge *GideView) NewProj(path gi.FileName, folder string, mainLang fi.Known, versCtrl filetree.VersCtrlName) *GideView { //gti:add
	np := filepath.Join(string(path), folder)
	err := os.MkdirAll(np, 0775)
	if err != nil {
		gi.MessageDialog(ge, fmt.Sprintf("Could not make folder for project at: %v, err: %v", np, err), "Could not Make Folder")
		return nil
	}
	nge := ge.OpenPath(gi.FileName(np))
	nge.Prefs.MainLang = mainLang
	if versCtrl != "" {
		nge.Prefs.VersCtrl = versCtrl
	}
	return nge
}

// NewFile creates a new file in the project
func (ge *GideView) NewFile(filename string, addToVcs bool) { //gti:add
	np := filepath.Join(string(ge.ProjRoot), filename)
	_, err := os.Create(np)
	if err != nil {
		gi.MessageDialog(ge, fmt.Sprintf("Could not make new file at: %v, err: %v", np, err), "Could not Make File")
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

// SaveProj saves project file containing custom project settings, in a
// standard toml-formatted file
func (ge *GideView) SaveProj() { //gti:add
	if ge.Prefs.ProjFilename == "" {
		return
	}
	ge.SaveProjAs(ge.Prefs.ProjFilename)
	ge.SaveAllCheck(false, nil) // false = no cancel option
}

// SaveProjIfExists saves project file containing custom project settings, in a
// standard toml-formatted file, only if it already exists -- returns true if saved
// saveAllFiles indicates if user should be prompted for saving all files
func (ge *GideView) SaveProjIfExists(saveAllFiles bool) bool {
	spell.SaveIfLearn()
	if ge.Prefs.ProjFilename == "" {
		return false
	}
	if _, err := os.Stat(string(ge.Prefs.ProjFilename)); os.IsNotExist(err) {
		return false // does not exist
	}
	ge.SaveProjAs(ge.Prefs.ProjFilename)
	if saveAllFiles {
		ge.SaveAllCheck(false, nil)
	}
	return true
}

// SaveProjAs saves project custom settings to given filename, in a standard
// toml-formatted file
// saveAllFiles indicates if user should be prompted for saving all files
// returns true if the user was prompted, false otherwise
func (ge *GideView) SaveProjAs(filename gi.FileName) bool { //gti:add
	spell.SaveIfLearn()
	gide.SavedPaths.AddPath(string(filename), gi.Prefs.Params.SavedPathsMax)
	gide.SavePaths()
	ge.Prefs.ProjFilename = filename
	ge.ProjFilename = ge.Prefs.ProjFilename
	ge.GrabPrefs()
	ge.Prefs.Save(filename)
	ge.Files.UpdatePath(string(filename))
	ge.Changed = false
	return false
}

// SaveAllCheck -- check if any files have not been saved, and prompt to save them
// returns true if there were unsaved files, false otherwise.
// cancelOpt presents an option to cancel current command, in which case function is not called.
// if function is passed, then it is called in all cases except if the user selects cancel.
func (ge *GideView) SaveAllCheck(cancelOpt bool, fun func()) bool {
	nch := ge.NChangedFiles()
	if nch == 0 {
		if fun != nil {
			fun()
		}
		return false
	}
	d := gi.NewBody().AddTitle("There are Unsaved Files").
		AddText(fmt.Sprintf("In Project: %v There are <b>%v</b> opened files with <b>unsaved changes</b> -- do you want to save all?", ge.Nm, nch))
	d.AddBottomBar(func(pw gi.Widget) {
		if cancelOpt {
			d.AddCancel(pw).SetText("Cancel Command")
		}
		gi.NewButton(pw).SetText("Don't Save").OnClick(func(e events.Event) {
			d.Close()
			if fun != nil {
				fun()
			}
		})
		gi.NewButton(pw).SetText("Save All").OnClick(func(e events.Event) {
			d.Close()
			ge.SaveAllOpenNodes()
			if fun != nil {
				fun()
			}
		})
	})
	d.NewDialog(ge).Run()
	return true
}

// ProjPathParse parses given project path into a root directory (which could
// be the path or just the directory portion of the path, depending in whether
// the path is a directory or not), and a bool if all is good (otherwise error
// message has been reported). projnm is always the last directory of the path.
func ProjPathParse(path string) (root, projnm, fnm string, ok bool) {
	if path == "" {
		return "", "blank", "", false
	}
	effpath := grr.Log1(filepath.EvalSymlinks(path))
	info, err := os.Lstat(effpath)
	if err != nil {
		emsg := fmt.Errorf("gide.ProjPathParse: Cannot open at given path: %q: Error: %v", effpath, err)
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

// CheckForProjAtPath checks if there is a .gide project at the given path
// returns project path and true if found, otherwise false
func CheckForProjAtPath(path string) (string, bool) {
	root, pnm, _, ok := ProjPathParse(path)
	if !ok {
		return "", false
	}
	gproj := filepath.Join(root, pnm+".gide")
	if _, err := os.Stat(gproj); os.IsNotExist(err) {
		return "", false // does not exist
	}
	return gproj, true
}

//////////////////////////////////////////////////////////////////////////////////////
//   Close / Quit Req

// NChangedFiles returns number of opened files with unsaved changes
func (ge *GideView) NChangedFiles() int {
	return ge.OpenNodes.NChanged()
}

// CloseWindow actually closes the window
func (ge *GideView) CloseWindow() {
	// todo:
}

// CloseWindowReq is called when user tries to close window -- we
// automatically save the project if it already exists (no harm), and prompt
// to save open files -- if this returns true, then it is OK to close --
// otherwise not
func (ge *GideView) CloseWindowReq() bool {
	ge.SaveProjIfExists(false) // don't prompt here, as we will do it now..
	nch := ge.NChangedFiles()
	if nch == 0 {
		return true
	}
	d := gi.NewBody().AddTitle("Close Project: There are Unsaved Files").
		AddText(fmt.Sprintf("In Project: %v There are <b>%v</b> opened files with <b>unsaved changes</b> -- do you want to save all or cancel closing this project and review  / save those files first?", ge.Nm, nch))
	d.AddBottomBar(func(pw gi.Widget) {
		d.AddCancel(pw)
		gi.NewButton(pw).SetText("Close without saving").OnClick(func(e events.Event) {
			d.Close()
			ge.CloseWindow()
		})
		gi.NewButton(pw).SetText("Save all").OnClick(func(e events.Event) {
			d.Close()
			ge.SaveAllOpenNodes()
		})
	})
	d.NewDialog(ge).Run()
	return false // not yet
}

// QuitReq is called when user tries to quit the app -- we go through all open
// main windows and look for gide windows and call their CloseWindowReq
// functions!
func QuitReq() bool {
	for _, win := range gi.MainRenderWins {
		if !strings.HasPrefix(win.Name, "gide-") {
			continue
		}
		msc := win.MainScene()
		ge := GideInScene(msc)
		if !ge.CloseWindowReq() {
			return false
		}
	}
	return true
}

//////////////////////////////////////////////////////////////////////////////////////
//   Project window

// NewGideProjPath creates a new GideView window with a new GideView project for given
// path, returning the window and the path
func NewGideProjPath(path string) *GideView {
	root, projnm, _, _ := ProjPathParse(path)
	return NewGideWindow(path, projnm, root, true)
}

// OpenGideProj creates a new GideView window opened to given GideView project,
// returning the window and the path
func OpenGideProj(projfile string) *GideView {
	pp := &gide.ProjPrefs{}
	if err := pp.Open(gi.FileName(projfile)); err != nil {
		slog.Debug("Project Prefs had a loading error", "error", err)
	}
	path := string(pp.ProjRoot)
	root, projnm, _, _ := ProjPathParse(path)
	return NewGideWindow(projfile, projnm, root, false)
}

func GideInScene(sc *gi.Scene) *GideView {
	return sc.Body.ChildByType(GideViewType, ki.NoEmbeds).(*GideView)
}

// NewGideWindow is common code for Open GideWindow from Proj or Path
func NewGideWindow(path, projnm, root string, doPath bool) *GideView {
	winm := "gide-" + projnm
	wintitle := winm + ": " + path

	if win, found := gi.AllRenderWins.FindName(winm); found {
		sc := win.MainScene()
		ge := GideInScene(sc)
		if string(ge.ProjRoot) == root {
			win.Raise()
			return ge
		}
	}

	b := gi.NewAppBody("gide").SetTitle(wintitle)
	b.Sc.Nm = winm

	ge := NewGideView(b)
	b.App().AppBarConfig = ge.AppBarConfig
	b.AddAppBar(ge.ConfigToolbar)

	b.App().About = `<code>Gide</code> is a graphical-interface (gi) integrated-development-environment (ide) written in the <b>GoGi</b> graphical interface system, within the <b>GoKi</b> tree framework.  See <a href="https://goki.dev/gide/v2/gide">Gide on GitHub</a> and <a href="https://goki.dev/gide/v2/wiki">Gide wiki</a> for documentation.<br>Gide is based on "projects" which are just directories containing files<br>* Use <code>File/Open Path...</code> to open an existing directory.<br>* Or <code>File/New Project...</code> to create a new directory for a new project<br><br>Version: ` + gide.Prefs.VersionInfo()

	/* todo: window doesn't exist yet -- need a delayed soln
	inClosePrompt := false
	win := ge.Sc.RenderWin()
	win.GoosiWin.SetCloseReqFunc(func(w goosi.Window) {
		if !inClosePrompt {
			inClosePrompt = true
			if ge.CloseWindowReq() {
				win.Close()
			} else {
				inClosePrompt = false
			}
		}
	})
	win.GoosiWin.SetCloseCleanFunc(func(w goosi.Window) {
		if gi.MainRenderWins.Len() <= 1 {
			go goosi.TheApp.Quit() // once main window is closed, quit
		}
	})
	*/

	b.NewWindow().Run()

	if doPath {
		ge.OpenPath(gi.FileName(path))
	} else {
		ge.OpenProj(gi.FileName(path))
	}

	return ge
}
