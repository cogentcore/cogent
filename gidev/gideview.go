// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package gidev implements the GideView editor, using all the elements
// from the gide interface.  Having it in a separate package
// allows GideView to also include other packages that tap into
// the gide interface, such as the GoPi interactive parser.
package gidev

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/girl"
	"github.com/goki/gi/gist"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/giv/textbuf"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/gi/oswin/mimedata"
	"github.com/goki/gi/oswin/osevent"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
	"github.com/goki/pi/complete"
	"github.com/goki/pi/filecat"
	"github.com/goki/pi/lex"
	"github.com/goki/pi/parse"
	"github.com/goki/pi/pi"
	"github.com/goki/pi/spell"
	"github.com/goki/vci"
	"goki.dev/gide/gide"
	"goki.dev/gide/gidebug"
)

// NTextViews is the number of text views to create -- to keep things simple
// and consistent (e.g., splitter settings always have the same number of
// values), we fix this degree of freedom, and have flexibility in the
// splitter settings for what to actually show.
const NTextViews = 2

// These are then the fixed indices of the different elements in the splitview
const (
	FileTreeIdx = iota
	TextView1Idx
	TextView2Idx
	TabsIdx
)

// GideView is the core editor and tab viewer framework for the Gide system.  The
// default view has a tree browser of files on the left, editor panels in the
// middle, and a tabbed viewer on the right.
type GideView struct {
	gi.Frame

	// root directory for the project -- all projects must be organized within a top-level root directory, with all the files therein constituting the scope of the project -- by default it is the path for ProjFilename
	ProjRoot gi.FileName `desc:"root directory for the project -- all projects must be organized within a top-level root directory, with all the files therein constituting the scope of the project -- by default it is the path for ProjFilename"`

	// current project filename for saving / loading specific Gide configuration information in a .gide file (optional)
	ProjFilename gi.FileName `ext:".gide" desc:"current project filename for saving / loading specific Gide configuration information in a .gide file (optional)"`

	// filename of the currently-active textview
	ActiveFilename gi.FileName `desc:"filename of the currently-active textview"`

	// language for current active filename
	ActiveLang filecat.Supported `desc:"language for current active filename"`

	// VCS repo for current active filename
	ActiveVCS vci.Repo `desc:"VCS repo for current active filename"`

	// VCS info for current active filename (typically branch or revision) -- for status
	ActiveVCSInfo string `desc:"VCS info for current active filename (typically branch or revision) -- for status"`

	// has the root changed?  we receive update signals from root for changes
	Changed bool `json:"-" desc:"has the root changed?  we receive update signals from root for changes"`

	// timestamp for when a file was last saved -- provides dirty state for various updates including rebuilding in debugger
	LastSaveTStamp time.Time `json:"-" desc:"timestamp for when a file was last saved -- provides dirty state for various updates including rebuilding in debugger"`

	// all the files in the project directory and subdirectories
	Files giv.FileTree `desc:"all the files in the project directory and subdirectories"`

	// the files tree view
	FilesView *gide.FileTreeView `json:"-" desc:"the files tree view"`

	// index of the currently-active textview -- new files will be viewed in other views if available
	ActiveTextViewIdx int `json:"-" desc:"index of the currently-active textview -- new files will be viewed in other views if available"`

	// list of open nodes, most recent first
	OpenNodes gide.OpenNodes `json:"-" desc:"list of open nodes, most recent first"`

	// the command buffers for commands run in this project
	CmdBufs map[string]*giv.TextBuf `json:"-" desc:"the command buffers for commands run in this project"`

	// history of commands executed in this session
	CmdHistory gide.CmdNames `json:"-" desc:"history of commands executed in this session"`

	// currently running commands in this project
	RunningCmds gide.CmdRuns `json:"-" xml:"-" desc:"currently running commands in this project"`

	// current arg var vals
	ArgVals gide.ArgVarVals `json:"-" xml:"-" desc:"current arg var vals"`

	// preferences for this project -- this is what is saved in a .gide project file
	Prefs gide.ProjPrefs `desc:"preferences for this project -- this is what is saved in a .gide project file"`

	// current debug view
	CurDbg *gide.DebugView `desc:"current debug view"`

	// first key in sequence if needs2 key pressed
	KeySeq1 key.Chord `desc:"first key in sequence if needs2 key pressed"`

	// mutex for protecting overall updates to GideView
	UpdtMu sync.Mutex `desc:"mutex for protecting overall updates to GideView"`
}

var KiT_GideView = kit.Types.AddType(&GideView{}, nil)

func init() {
	kit.Types.SetProps(KiT_GideView, GideViewProps)
	// gi.URLHandler = URLHandler
	girl.TextLinkHandler = TextLinkHandler
}

////////////////////////////////////////////////////////
// Gide interface

func (ge *GideView) VPort() *gi.Viewport2D {
	return ge.Viewport
}

func (ge *GideView) ProjPrefs() *gide.ProjPrefs {
	return &ge.Prefs
}

func (ge *GideView) FileTree() *giv.FileTree {
	return &ge.Files
}

func (ge *GideView) LastSaveTime() time.Time {
	return ge.LastSaveTStamp
}

// VersCtrl returns the version control system in effect, using the file tree detected
// version or whatever is set in project preferences
func (ge *GideView) VersCtrl() giv.VersCtrlName {
	vc := ge.Prefs.VersCtrl
	return vc
}

func (ge *GideView) CmdRuns() *gide.CmdRuns {
	return &ge.RunningCmds
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
func (ge *GideView) UpdateFiles() {
	ge.Files.OpenPath(string(ge.ProjRoot))
	if ge.FilesView != nil {
		ge.FilesView.ReSync()
	}
}

func (ge *GideView) IsEmpty() bool {
	return ge.ProjRoot == ""
}

// OpenRecent opens a recently-used file
func (ge *GideView) OpenRecent(filename gi.FileName) {
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
	opts := giv.DlgOpts{Title: "Recent Project Paths", Prompt: "Delete paths you no longer use", Ok: true, Cancel: true, NoAdd: true}
	giv.SliceViewDialog(ge.Viewport, &tmp, opts,
		nil, ge, func(recv, send ki.Ki, sig int64, data any) {
			if sig == int64(gi.DialogAccepted) {
				gide.SavedPaths = nil
				gide.SavedPaths = append(gide.SavedPaths, tmp...)
				gi.StringsAddExtras((*[]string)(&gide.SavedPaths), gide.SavedPathsExtras)
			}
		})
}

// OpenFile opens file in an open project if it has the same path as the file
// or in a new window.
func (ge *GideView) OpenFile(fnm string) {
	abfn, _ := filepath.Abs(fnm)
	if strings.HasPrefix(abfn, string(ge.ProjRoot)) {
		ge.ViewFile(gi.FileName(abfn))
		return
	}
	for _, w := range gi.MainWindows {
		mfr, err := w.MainFrame()
		if err != nil || mfr.NumChildren() == 0 {
			continue
		}
		gevi := mfr.Child(0).Embed(KiT_GideView)
		if gevi == nil {
			continue
		}
		geo := gevi.(*GideView)
		if strings.HasPrefix(abfn, string(geo.ProjRoot)) {
			geo.ViewFile(gi.FileName(abfn))
			return
		}
	}
	// fmt.Printf("open path: %s\n", ge.ProjRoot)
	ge.OpenPath(gi.FileName(abfn))
}

// OpenPath creates a new project by opening given path, which can either be a
// specific file or a folder containing multiple files of interest -- opens in
// current GideView object if it is empty, or otherwise opens a new window.
func (ge *GideView) OpenPath(path gi.FileName) (*gi.Window, *GideView) {
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
		ge.Prefs.ProjFilename = gi.FileName(filepath.Join(root, pnm+".gide"))
		ge.ProjFilename = ge.Prefs.ProjFilename
		ge.Prefs.ProjRoot = ge.ProjRoot
		ge.Config()
		ge.GuessMainLang()
		ge.LangDefaults()
		win := ge.ParentWindow()
		if win != nil {
			winm := "gide-" + pnm
			win.SetName(winm)
			win.SetTitle(winm + ": " + root)
		}
		if fnm != "" {
			ge.NextViewFile(gi.FileName(fnm))
		}
	}
	return ge.ParentWindow(), ge
}

// OpenProj opens .gide project file and its settings from given filename, in a standard
// JSON-formatted file
func (ge *GideView) OpenProj(filename gi.FileName) (*gi.Window, *GideView) {
	if !ge.IsEmpty() {
		return OpenGideProj(string(filename))
	}
	ge.Defaults()
	ge.Prefs.OpenJSON(filename)
	ge.Prefs.ProjFilename = filename // should already be set but..
	_, pnm, _, ok := ProjPathParse(string(ge.Prefs.ProjRoot))
	if ok {
		gide.SetGoMod(ge.Prefs.GoMod)
		os.Chdir(string(ge.Prefs.ProjRoot))
		gide.SavedPaths.AddPath(string(filename), gi.Prefs.Params.SavedPathsMax)
		gide.SavePaths()
		ge.SetName(pnm)
		ge.ApplyPrefs()
		ge.Config()
		win := ge.ParentWindow()
		if win != nil {
			winm := "gide-" + pnm
			win.SetName(winm)
			win.SetTitle(winm + ": " + string(ge.Prefs.ProjRoot))
		}
	}
	return ge.ParentWindow(), ge
}

// NewProj creates a new project at given path, making a new folder in that
// path -- all GideView projects are essentially defined by a path to a folder
// containing files.  If the folder already exists, then use OpenPath.
// Can also specify main language and version control type
func (ge *GideView) NewProj(path gi.FileName, folder string, mainLang filecat.Supported, versCtrl giv.VersCtrlName) (*gi.Window, *GideView) {
	np := filepath.Join(string(path), folder)
	err := os.MkdirAll(np, 0775)
	if err != nil {
		gi.PromptDialog(ge.Viewport, gi.DlgOpts{Title: "Couldn't Make Folder", Prompt: fmt.Sprintf("Could not make folder for project at: %v, err: %v", np, err)}, gi.AddOk, gi.NoCancel, nil, nil)
		return nil, nil
	}
	win, nge := ge.OpenPath(gi.FileName(np))
	nge.Prefs.MainLang = mainLang
	if versCtrl != "" {
		nge.Prefs.VersCtrl = versCtrl
	}
	return win, nge
}

// NewFile creates a new file in the project
func (ge *GideView) NewFile(filename string, addToVcs bool) {
	np := filepath.Join(string(ge.ProjRoot), filename)
	_, err := os.Create(np)
	if err != nil {
		gi.PromptDialog(ge.Viewport, gi.DlgOpts{Title: "Couldn't Make File", Prompt: fmt.Sprintf("Could not make new file at: %v, err: %v", np, err)}, gi.AddOk, gi.NoCancel, nil, nil)
		return
	}
	ge.Files.UpdateNewFile(np)
	if addToVcs {
		nfn, ok := ge.Files.FindFile(np)
		if ok {
			nfn.AddToVcs()
		}
	}
}

// SaveProj saves project file containing custom project settings, in a
// standard JSON-formatted file
func (ge *GideView) SaveProj() {
	if ge.Prefs.ProjFilename == "" {
		return
	}
	ge.SaveProjAs(ge.Prefs.ProjFilename, true) // save all files
}

// SaveProjIfExists saves project file containing custom project settings, in a
// standard JSON-formatted file, only if it already exists -- returns true if saved
// saveAllFiles indicates if user should be prompted for saving all files
func (ge *GideView) SaveProjIfExists(saveAllFiles bool) bool {
	spell.SaveIfLearn()
	if ge.Prefs.ProjFilename == "" {
		return false
	}
	if _, err := os.Stat(string(ge.Prefs.ProjFilename)); os.IsNotExist(err) {
		return false // does not exist
	}
	ge.SaveProjAs(ge.Prefs.ProjFilename, saveAllFiles)
	return true
}

// SaveProjAs saves project custom settings to given filename, in a standard
// JSON-formatted file
// saveAllFiles indicates if user should be prompted for saving all files
// returns true if the user was prompted, false otherwise
func (ge *GideView) SaveProjAs(filename gi.FileName, saveAllFiles bool) bool {
	spell.SaveIfLearn()
	gide.SavedPaths.AddPath(string(filename), gi.Prefs.Params.SavedPathsMax)
	gide.SavePaths()
	// ge.Files.UpdateNewFile(string(filename))
	ge.Prefs.ProjFilename = filename
	ge.ProjFilename = ge.Prefs.ProjFilename
	ge.GrabPrefs()
	ge.Prefs.SaveJSON(filename)
	ge.Changed = false
	if saveAllFiles {
		return ge.SaveAllCheck(false, nil) // false = no cancel option
	}
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
	opts := []string{"Save All", "Don't Save"}
	if cancelOpt {
		opts = []string{"Save All", "Don't Save", "Cancel Command"}
	}
	gi.ChoiceDialog(ge.Viewport, gi.DlgOpts{Title: "There are Unsaved Files",
		Prompt: fmt.Sprintf("In Project: %v There are <b>%v</b> opened files with <b>unsaved changes</b> -- do you want to save all?", ge.Nm, nch)}, opts,
		ge.This(), func(recv, send ki.Ki, sig int64, data any) {
			if sig != 2 {
				if sig == 0 {
					ge.SaveAllOpenNodes()
				}
				if fun != nil {
					fun()
				}
			}
		})
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
	info, err := os.Lstat(path)
	if err != nil {
		emsg := fmt.Errorf("gide.ProjPathParse: Cannot open at given path: %q: Error: %v", path, err)
		log.Println(emsg)
		return
	}
	path, _ = filepath.Abs(path)
	dir, fn := filepath.Split(path)
	pathIsDir := info.IsDir()
	if pathIsDir {
		root = path
	} else {
		root = filepath.Clean(dir)
		fnm = fn
	}
	_, projnm = filepath.Split(root)
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

// GuessMainLang guesses the main language in the project -- returns true if successful
func (ge *GideView) GuessMainLang() bool {
	ecsc := ge.Files.FileExtCounts(filecat.Code)
	ecsd := ge.Files.FileExtCounts(filecat.Doc)
	ecs := append(ecsc, ecsd...)
	giv.FileNodeNameCountSort(ecs)
	for _, ec := range ecs {
		ls := filecat.ExtSupported(ec.Name)
		if ls != filecat.NoSupport {
			ge.Prefs.MainLang = ls
			return true
		}
	}
	return false
}

// LangDefaults applies default language settings based on MainLang
func (ge *GideView) LangDefaults() {
	ge.Prefs.RunCmds = gide.CmdNames{"Build: Run Proj"}
	ge.Prefs.BuildDir = ge.Prefs.ProjRoot
	ge.Prefs.BuildTarg = ge.Prefs.ProjRoot
	ge.Prefs.RunExec = gi.FileName(filepath.Join(string(ge.Prefs.ProjRoot), ge.Nm))
	if len(ge.Prefs.BuildCmds) == 0 {
		switch ge.Prefs.MainLang {
		case filecat.Go:
			ge.Prefs.BuildCmds = gide.CmdNames{"Go: Build Proj"}
		case filecat.TeX:
			ge.Prefs.BuildCmds = gide.CmdNames{"LaTeX: LaTeX PDF"}
			ge.Prefs.RunCmds = gide.CmdNames{"File: Open Target"}
		default:
			ge.Prefs.BuildCmds = gide.CmdNames{"Build: Make"}
		}
	}
	if ge.Prefs.VersCtrl == "" {
		repo, _ := ge.Files.FirstVCS()
		if repo != nil {
			ge.Prefs.VersCtrl = giv.VersCtrlName(repo.Vcs())
		}
	}
}

//////////////////////////////////////////////////////////////////////////////////////
//   TextViews

// ConfigTextBuf configures the text buf according to prefs
func (ge *GideView) ConfigTextBuf(tb *giv.TextBuf) {
	tb.SetHiStyle(gi.Prefs.Colors.HiStyle)
	tb.Opts.EditorPrefs = ge.Prefs.Editor
	tb.ConfigSupported()
	if tb.Complete != nil {
		tb.Complete.LookupFunc = ge.LookupFun
	}

	// these are now set in std textbuf..
	// tb.SetSpellCorrect(tb, giv.SpellCorrectEdit)                    // always set -- option can override
	// tb.SetCompleter(&tb.PiState, pi.CompletePi, giv.CompleteGoEdit) // todo: need pi edit too..
}

// ActiveTextView returns the currently-active TextView
func (ge *GideView) ActiveTextView() *gide.TextView {
	//	fmt.Printf("stdout: active text view idx: %v\n", ge.ActiveTextViewIdx)
	return ge.TextViewByIndex(ge.ActiveTextViewIdx)
}

// ActiveFileNode returns the file node for the active file -- nil if none
func (ge *GideView) ActiveFileNode() *giv.FileNode {
	return ge.FileNodeForFile(string(ge.ActiveFilename), false)
}

// TextViewIndex finds index of given textview (0 or 1)
func (ge *GideView) TextViewIndex(av *gide.TextView) int {
	for i := 0; i < NTextViews; i++ {
		tv := ge.TextViewByIndex(i)
		if tv.This() == av.This() {
			return i
		}
	}
	return -1 // shouldn't happen
}

// TextViewForFileNode finds a TextView that is viewing given FileNode,
// and its index, or false if none is
func (ge *GideView) TextViewForFileNode(fn *giv.FileNode) (*gide.TextView, int, bool) {
	if fn.Buf == nil {
		return nil, -1, false
	}
	ge.ConfigTextBuf(fn.Buf)
	for i := 0; i < NTextViews; i++ {
		tv := ge.TextViewByIndex(i)
		if tv != nil && tv.Buf != nil && tv.Buf.This() == fn.Buf.This() && ge.PanelIsOpen(i+TextView1Idx) {
			return tv, i, true
		}
	}
	return nil, -1, false
}

// OpenNodeForTextView finds the FileNode that a given TextView is
// viewing, returning its index within OpenNodes list, or false if not found
func (ge *GideView) OpenNodeForTextView(tv *gide.TextView) (*giv.FileNode, int, bool) {
	if tv.Buf == nil {
		return nil, -1, false
	}
	for i, ond := range ge.OpenNodes {
		tve := tv.Embed(giv.KiT_TextView).(*giv.TextView)
		if ond.Buf == tve.Buf {
			return ond, i, true
		}
	}
	return nil, -1, false
}

// TextViewForFile finds FileNode for file, and returns TextView and index
// that is viewing that FileNode, or false if none is
func (ge *GideView) TextViewForFile(fnm gi.FileName) (*gide.TextView, int, bool) {
	fn, ok := ge.Files.FindFile(string(fnm))
	if !ok {
		return nil, -1, false
	}
	return ge.TextViewForFileNode(fn.This().Embed(giv.KiT_FileNode).(*giv.FileNode))
}

// SetActiveFileInfo sets the active file info from textbuf
func (ge *GideView) SetActiveFileInfo(buf *giv.TextBuf) {
	ge.ActiveFilename = buf.Filename
	ge.ActiveLang = buf.Info.Sup
	fn := ge.FileNodeForFile(string(ge.ActiveFilename), false)
	ge.ActiveVCSInfo = ""
	ge.ActiveVCS = nil
	if fn != nil {
		repo, _ := fn.Repo()
		if repo != nil {
			ge.ActiveVCS = repo
			cur, err := repo.Current()
			if err == nil {
				ge.ActiveVCSInfo = fmt.Sprintf("%s: <i>%s</i>", repo.Vcs(), cur)
			}
		}
	}
}

// SetActiveTextView sets the given textview as the active one, and returns its index
func (ge *GideView) SetActiveTextView(av *gide.TextView) int {
	wupdt := ge.TopUpdateStart()
	defer ge.TopUpdateEnd(wupdt)

	idx := ge.TextViewIndex(av)
	if idx < 0 {
		return -1
	}
	if ge.ActiveTextViewIdx == idx {
		return idx
	}
	ge.ActiveTextViewIdx = idx
	if av.Buf != nil {
		ge.SetActiveFileInfo(av.Buf)
	}
	ge.SetStatus("")
	return idx
}

// SetActiveTextViewIdx sets the given view index as the currently-active
// TextView -- returns that textview
func (ge *GideView) SetActiveTextViewIdx(idx int) *gide.TextView {
	wupdt := ge.TopUpdateStart()
	defer ge.TopUpdateEnd(wupdt)

	if idx < 0 || idx >= NTextViews {
		log.Printf("GideView SetActiveTextViewIdx: text view index out of range: %v\n", idx)
		return nil
	}
	ge.ActiveTextViewIdx = idx
	av := ge.ActiveTextView()
	if av.Buf != nil {
		ge.SetActiveFileInfo(av.Buf)
		av.Buf.FileModCheck()
	}
	ge.SetStatus("")
	av.GrabFocus()
	return av
}

// NextTextView returns the next text view available for viewing a file and
// its index -- if the active text view is empty, then it is used, otherwise
// it is the next one (if visible)
func (ge *GideView) NextTextView() (*gide.TextView, int) {
	av := ge.TextViewByIndex(ge.ActiveTextViewIdx)
	if av.Buf == nil {
		return av, ge.ActiveTextViewIdx
	}
	nxt := (ge.ActiveTextViewIdx + 1) % NTextViews
	if !ge.PanelIsOpen(nxt + TextView1Idx) {
		return av, ge.ActiveTextViewIdx
	}
	return ge.TextViewByIndex(nxt), nxt
}

// SwapTextViews switches the buffers for the two open textviews
// only operates if both panels are open
func (ge *GideView) SwapTextViews() bool {
	if !ge.PanelIsOpen(TextView1Idx) || !ge.PanelIsOpen(TextView1Idx+1) {
		return false
	}
	wupdt := ge.TopUpdateStart()
	defer ge.TopUpdateEnd(wupdt)

	tva := ge.TextViewByIndex(0)
	tvb := ge.TextViewByIndex(1)
	bufa := tva.Buf
	bufb := tvb.Buf
	tva.SetBuf(bufb)
	tvb.SetBuf(bufa)
	ge.SetStatus("swapped buffers")
	return true
}

///////////////////////////////////////////////////////////////////////
//  File Actions

// SaveActiveView saves the contents of the currently-active textview
func (ge *GideView) SaveActiveView() {
	tv := ge.ActiveTextView()
	if tv.Buf != nil {
		ge.LastSaveTStamp = time.Now()
		if tv.Buf.Filename != "" {
			tv.Buf.Save()
			ge.SetStatus("File Saved")
			fnm := string(tv.Buf.Filename)
			updt := ge.FilesView.UpdateStart()
			ge.FilesView.SetFullReRender()
			fpath, _ := filepath.Split(fnm)
			ge.Files.UpdateNewFile(fpath) // update everything in dir -- will have removed autosave
			ge.FilesView.UpdateEnd(updt)
			ge.RunPostCmdsActiveView()
		} else {
			giv.CallMethod(ge, "SaveActiveViewAs", ge.Viewport) // uses fileview
		}
	}
	ge.SaveProjIfExists(false) // no saveall
}

// SaveActiveViewAs save with specified filename the contents of the
// currently-active textview
func (ge *GideView) SaveActiveViewAs(filename gi.FileName) {
	tv := ge.ActiveTextView()
	if tv.Buf != nil {
		ge.LastSaveTStamp = time.Now()
		ofn := tv.Buf.Filename
		tv.Buf.SaveAsFunc(filename, func(canceled bool) {
			if canceled {
				ge.SetStatus(fmt.Sprintf("File %v NOT Saved As: %v", ofn, filename))
				return
			}
			ge.SetStatus(fmt.Sprintf("File %v Saved As: %v", ofn, filename))
			// ge.RunPostCmdsActiveView() // doesn't make sense..
			ge.Files.UpdateNewFile(string(filename)) // update everything in dir -- will have removed autosave
			fnk, ok := ge.Files.FindFile(string(filename))
			if ok {
				fn := fnk.This().Embed(giv.KiT_FileNode).(*giv.FileNode)
				if fn.Buf != nil {
					fn.Buf.Revert()
				}
				ge.ViewFileNode(tv, ge.ActiveTextViewIdx, fn)
			}
		})
	}
	ge.SaveProjIfExists(false) // no saveall
}

// RevertActiveView revert active view to saved version
func (ge *GideView) RevertActiveView() {
	tv := ge.ActiveTextView()
	if tv.Buf != nil {
		ge.ConfigTextBuf(tv.Buf)
		tv.Buf.Revert()
		tv.Buf.Undos.Reset() // key implication of revert
		fpath, _ := filepath.Split(string(tv.Buf.Filename))
		ge.Files.UpdateNewFile(fpath) // update everything in dir -- will have removed autosave
	}
}

// CloseActiveView closes the buffer associated with active view
func (ge *GideView) CloseActiveView() {
	tv := ge.ActiveTextView()
	ond, _, got := ge.OpenNodeForTextView(tv)
	if got {
		ond.Buf.Close(func(canceled bool) {
			if canceled {
				ge.SetStatus(fmt.Sprintf("File %v NOT closed", ond.FPath))
				return
			}
			ge.SetStatus(fmt.Sprintf("File %v closed", ond.FPath))
		})
	}
}

// RunPostCmdsActiveView runs any registered post commands on the active view
// -- returns true if commands were run and file was reverted after that --
// uses MainLang to disambiguate if multiple languages associated with extension.
func (ge *GideView) RunPostCmdsActiveView() bool {
	tv := ge.ActiveTextView()
	ond, _, got := ge.OpenNodeForTextView(tv)
	if got {
		return ge.RunPostCmdsFileNode(ond)
	}
	return false
}

// RunPostCmdsFileNode runs any registered post commands on the given file node
// -- returns true if commands were run and file was reverted after that --
// uses MainLang to disambiguate if multiple languages associated with extension.
func (ge *GideView) RunPostCmdsFileNode(fn *giv.FileNode) bool {
	lang := fn.Info.Sup
	if lopt, has := gide.AvailLangs[lang]; has {
		if len(lopt.PostSaveCmds) > 0 {
			ge.ExecCmdsFileNode(fn, lopt.PostSaveCmds, false, true) // no select, yes clear
			fn.Buf.Revert()
			return true
		}
	}
	return false
}

// AutoSaveCheck checks for an autosave file and prompts user about opening it
// -- returns true if autosave file does exist for a file that currently
// unchanged (means just opened)
func (ge *GideView) AutoSaveCheck(tv *gide.TextView, vidx int, fn *giv.FileNode) bool {
	if strings.HasPrefix(fn.Nm, "#") && strings.HasSuffix(fn.Nm, "#") {
		fn.Buf.Autosave = false
		return false // we are the autosave file
	}
	fn.Buf.Autosave = true
	if tv.IsChanged() || !fn.Buf.AutoSaveCheck() {
		return false
	}
	ge.DiffFileNode(fn, gi.FileName(fn.Buf.AutoSaveFilename()))
	gi.ChoiceDialog(ge.Viewport, gi.DlgOpts{Title: "Autosave file Exists",
		Prompt: fmt.Sprintf("An auto-save file for file: %v exists -- open it in the other text view (you can then do Save As to replace current file)?  If you don't open it, the next change made will overwrite it with a new one, erasing any changes.", fn.Nm)},
		[]string{"Open Autosave File", "Ignore and Overwrite Autosave File"},
		ge.This(), func(recv, send ki.Ki, sig int64, data any) {
			switch sig {
			case 0:
				ge.NextViewFile(gi.FileName(fn.Buf.AutoSaveFilename()))
			case 1:
				fn.Buf.AutoSaveDelete()
				ge.Files.UpdateNewFile(fn.Buf.AutoSaveFilename()) // will update dir
			}
		})
	return true
}

// OpenFileNode opens file for file node -- returns new bool and error
func (ge *GideView) OpenFileNode(fn *giv.FileNode) (bool, error) {
	if fn.IsDir() {
		return false, fmt.Errorf("cannot open directory: %v", fn.FPath)
	}
	giv.FileNodeHiStyle = gi.Prefs.Colors.HiStyle // must be set prior to OpenBuf
	nw, err := fn.OpenBuf()
	if err == nil {
		ge.ConfigTextBuf(fn.Buf)
		ge.OpenNodes.Add(fn)
		fn.SetOpen()
		// updt := ge.FilesView.UpdateStart()
		// ge.FilesView.SetFullReRender()
		fn.UpdateNode()
		// ge.FilesView.UpdateEnd(updt)
	}
	return nw, err
}

// ViewFileNode sets the given text view to view file in given node (opens
// buffer if not already opened)
func (ge *GideView) ViewFileNode(tv *gide.TextView, vidx int, fn *giv.FileNode) {
	if fn.IsDir() {
		return
	}
	wupdt := ge.TopUpdateStart()
	defer ge.TopUpdateEnd(wupdt)

	if tv.IsChanged() {
		ge.SetStatus(fmt.Sprintf("Note: Changes not yet saved in file: %v", tv.Buf.Filename))
	}
	nw, err := ge.OpenFileNode(fn)
	if err == nil {
		tv.StyleTextView() // make sure
		tv.SetBuf(fn.Buf)
		if nw {
			ge.AutoSaveCheck(tv, vidx, fn)
		}
		ge.SetActiveTextViewIdx(vidx) // this calls FileModCheck
	}
}

// NextViewFileNode sets the next text view to view file in given node (opens
// buffer if not already opened) -- if already being viewed, that is
// activated, returns text view and index
func (ge *GideView) NextViewFileNode(fn *giv.FileNode) (*gide.TextView, int) {
	wupdt := ge.TopUpdateStart()
	defer ge.TopUpdateEnd(wupdt)

	tv, idx, ok := ge.TextViewForFileNode(fn)
	if ok {
		ge.SetActiveTextViewIdx(idx)
		return tv, idx
	}
	nv, nidx := ge.NextTextView()
	ge.ViewFileNode(nv, nidx, fn)
	return nv, nidx
}

// FileNodeForFile returns file node for given file path
// add: if not found in existing tree and external files, then if add is true,
// it is added to the ExtFiles list.
func (ge *GideView) FileNodeForFile(fpath string, add bool) *giv.FileNode {
	fnk, ok := ge.Files.FindFile(fpath)
	if !ok {
		if !add {
			return nil
		}
		if strings.HasSuffix(fpath, "/") {
			log.Printf("GideView: attempt to add dir to external files: %v\n", fpath)
			return nil
		}
		efn, err := ge.Files.AddExtFile(fpath)
		if err != nil {
			log.Printf("GideView: cannot add external file: %v\n", err)
			return nil
		}
		return efn
	}
	fn := fnk.This().Embed(giv.KiT_FileNode).(*giv.FileNode)
	if fn.IsDir() {
		return nil
	}
	return fn
}

// TextBufForFile returns TextBuf for given file path.
// add: if not found in existing tree and external files, then if add is true,
// it is added to the ExtFiles list.
func (ge *GideView) TextBufForFile(fpath string, add bool) *giv.TextBuf {
	fn := ge.FileNodeForFile(fpath, add)
	if fn == nil {
		return nil
	}
	_, err := ge.OpenFileNode(fn)
	if err == nil {
		return fn.Buf
	}
	return nil
}

// NextViewFile sets the next text view to view given file name -- include as
// much of name as possible to disambiguate -- will use the first matching --
// if already being viewed, that is activated -- returns textview and its
// index, false if not found
func (ge *GideView) NextViewFile(fnm gi.FileName) (*gide.TextView, int, bool) {
	fn := ge.FileNodeForFile(string(fnm), true)
	if fn == nil {
		return nil, -1, false
	}
	nv, nidx := ge.NextViewFileNode(fn)
	return nv, nidx, true
}

// ViewFile views file in an existing TextView if it is already viewing that
// file, otherwise opens ViewFileNode in active buffer
func (ge *GideView) ViewFile(fnm gi.FileName) (*gide.TextView, int, bool) {
	wupdt := ge.TopUpdateStart()
	defer ge.TopUpdateEnd(wupdt)

	fn := ge.FileNodeForFile(string(fnm), true)
	if fn == nil {
		return nil, -1, false
	}
	tv, idx, ok := ge.TextViewForFileNode(fn)
	if ok {
		ge.SetActiveTextViewIdx(idx)
		return tv, idx, ok
	}
	tv = ge.ActiveTextView()
	idx = ge.ActiveTextViewIdx
	ge.ViewFileNode(tv, idx, fn)
	return tv, idx, true
}

// ViewFileInIdx views file in given text view index
func (ge *GideView) ViewFileInIdx(fnm gi.FileName, idx int) (*gide.TextView, int, bool) {
	wupdt := ge.TopUpdateStart()
	defer ge.TopUpdateEnd(wupdt)

	fn := ge.FileNodeForFile(string(fnm), true)
	if fn == nil {
		return nil, -1, false
	}
	tv := ge.TextViewByIndex(idx)
	ge.ViewFileNode(tv, idx, fn)
	return tv, idx, true
}

// LinkViewFileNode opens the file node in the 2nd textview, which is next to
// the tabs where links are clicked, if it is not collapsed -- else 1st
func (ge *GideView) LinkViewFileNode(fn *giv.FileNode) (*gide.TextView, int) {
	wupdt := ge.TopUpdateStart()
	defer ge.TopUpdateEnd(wupdt)

	if ge.PanelIsOpen(TextView2Idx) {
		ge.SetActiveTextViewIdx(1)
	} else {
		ge.SetActiveTextViewIdx(0)
	}
	tv := ge.ActiveTextView()
	idx := ge.ActiveTextViewIdx
	ge.ViewFileNode(tv, idx, fn)
	return tv, idx
}

// LinkViewFile opens the file in the 2nd textview, which is next to
// the tabs where links are clicked, if it is not collapsed -- else 1st
func (ge *GideView) LinkViewFile(fnm gi.FileName) (*gide.TextView, int, bool) {
	fn := ge.FileNodeForFile(string(fnm), true)
	if fn == nil {
		return nil, -1, false
	}
	tv, idx, ok := ge.TextViewForFileNode(fn)
	if ok {
		if idx == 1 {
			return tv, idx, true
		}
		if ge.SwapTextViews() {
			return ge.TextViewByIndex(1), 1, true
		}
	}
	nv, nidx := ge.LinkViewFileNode(fn)
	return nv, nidx, true
}

// ShowFile shows given file name at given line, returning TextView showing it
// or error if not found.
func (ge *GideView) ShowFile(fname string, ln int) (*gide.TextView, error) {
	tv, _, ok := ge.LinkViewFile(gi.FileName(fname))
	if ok {
		tv.SetCursorShow(lex.Pos{Ln: ln - 1})
		return tv, nil
	}
	return nil, fmt.Errorf("ShowFile: file named: %v not found\n", fname)
}

// GideViewOpenNodes gets list of open nodes for submenu-func
func GideViewOpenNodes(it any, vp *gi.Viewport2D) []string {
	ge, ok := it.(ki.Ki).Embed(KiT_GideView).(*GideView)
	if !ok {
		return nil
	}
	return ge.OpenNodes.Strings()
}

// ViewOpenNodeName views given open node (by name) in active view
func (ge *GideView) ViewOpenNodeName(name string) {
	nb := ge.OpenNodes.ByStringName(name)
	if nb == nil {
		return
	}
	tv := ge.ActiveTextView()
	ge.ViewFileNode(tv, ge.ActiveTextViewIdx, nb)
}

// SelectOpenNode pops up a menu to select an open node (aka buffer) to view
// in current active textview
func (ge *GideView) SelectOpenNode() {
	if len(ge.OpenNodes) == 0 {
		ge.SetStatus("No open nodes to choose from")
		return
	}
	nl := ge.OpenNodes.Strings()
	tv := ge.ActiveTextView() // nl[0] is always currently viewed
	def := nl[0]
	if len(nl) > 1 {
		def = nl[1]
	}
	gi.StringsChooserPopup(nl, def, tv, func(recv, send ki.Ki, sig int64, data any) {
		ac := send.(*gi.Action)
		idx := ac.Data.(int)
		nb := ge.OpenNodes[idx]
		ge.ViewFileNode(tv, ge.ActiveTextViewIdx, nb)
	})
}

// CloneActiveView sets the next text view to view the same file currently being vieweds
// in the active view. returns text view and index
func (ge *GideView) CloneActiveView() (*gide.TextView, int) {
	tv := ge.ActiveTextView()
	if tv == nil {
		return nil, -1
	}
	ond, _, got := ge.OpenNodeForTextView(tv)
	if got {
		nv, nidx := ge.NextTextView()
		ge.ViewFileNode(nv, nidx, ond)
		return nv, nidx
	}
	return nil, -1
}

// SaveAllOpenNodes saves all of the open filenodes to their current file names
func (ge *GideView) SaveAllOpenNodes() {
	for _, ond := range ge.OpenNodes {
		if ond.Buf == nil {
			continue
		}
		if ond.Buf.IsChanged() {
			ond.Buf.Save()
			ge.RunPostCmdsFileNode(ond)
		}
	}
}

// SaveAll saves all of the open filenodes to their current file names
// and saves the project state if it has been saved before (i.e., the .gide file exists)
func (ge *GideView) SaveAll() {
	ge.SaveAllOpenNodes()
	ge.SaveProjIfExists(false)
}

// CloseOpenNodes closes any nodes with open views (including those in directories under nodes).
// called prior to rename.
func (ge *GideView) CloseOpenNodes(nodes []*gide.FileNode) {
	nn := len(ge.OpenNodes)
	for ni := nn - 1; ni >= 0; ni-- {
		ond := ge.OpenNodes[ni]
		if ond.Buf == nil {
			continue
		}
		path := string(ond.Buf.Filename)
		for _, cnd := range nodes {
			if strings.HasPrefix(path, string(cnd.FPath)) {
				ond.Buf.Close(func(canceled bool) {
					if canceled {
						ge.SetStatus(fmt.Sprintf("File %v NOT closed -- recommended as file name changed!", ond.FPath))
						return
					}
					ge.SetStatus(fmt.Sprintf("File %v closed due to file name change", ond.FPath))
				})
				break // out of inner node loop
			}
		}
	}
}

// TextViewSig handles all signals from the textviews
func (ge *GideView) TextViewSig(tv *gide.TextView, sig giv.TextViewSignals) {
	ge.SetActiveTextView(tv) // if we're sending signals, we're the active one!
	switch sig {
	case giv.TextViewCursorMoved:
		ge.SetStatus("") // this really doesn't make any noticeable diff in perf
	case giv.TextViewISearch, giv.TextViewQReplace:
		ge.SetStatus("")
	}
}

// DiffFiles shows the differences between two given files
// in side-by-side DiffView and in the console as a context diff.
// It opens the files as file nodes and uses existing contents if open already.
func (ge *GideView) DiffFiles(fnmA, fnmB gi.FileName) {
	fna := ge.FileNodeForFile(string(fnmA), true)
	if fna == nil {
		return
	}
	if fna.Buf == nil {
		ge.OpenFileNode(fna)
	}
	if fna.Buf == nil {
		return
	}
	ge.DiffFileNode(fna, fnmB)
}

// DiffFileNode shows the differences between given file node as the A file,
// and another given file as the B file,
// in side-by-side DiffView and in the console as a context diff.
func (ge *GideView) DiffFileNode(fna *giv.FileNode, fnmB gi.FileName) {
	fnb := ge.FileNodeForFile(string(fnmB), true)
	if fnb == nil {
		return
	}
	if fnb.Buf == nil {
		ge.OpenFileNode(fnb)
	}
	if fnb.Buf == nil {
		return
	}
	dif := fna.Buf.DiffBufsUnified(fnb.Buf, 3)
	cbuf, _, _ := ge.RecycleCmdTab("Diffs", true, true)
	cbuf.SetText(dif)
	cbuf.AutoScrollViews()

	astr := fna.Buf.Strings(false)
	bstr := fnb.Buf.Strings(false)

	giv.DiffViewDialog(ge.Viewport, astr, bstr, string(fna.Buf.Filename), string(fnb.Buf.Filename), "", "", giv.DlgOpts{Title: "Diff File View:"})
}

// CountWords counts number of words (and lines) in active file
// returns a string report thereof.
func (ge *GideView) CountWords() string {
	av := ge.ActiveTextView()
	if av.Buf == nil || av.Buf.NLines <= 0 {
		return "empty"
	}
	av.Buf.LinesMu.RLock()
	defer av.Buf.LinesMu.RUnlock()
	ll := av.Buf.NLines - 1
	reg := textbuf.NewRegion(0, 0, ll, len(av.Buf.Lines[ll]))
	words, lines := textbuf.CountWordsLinesRegion(av.Buf.Lines, reg)
	return fmt.Sprintf("File: %s  Words: %d   Lines: %d\n", giv.DirAndFile(string(av.Buf.Filename)), words, lines)
}

// CountWordsRegion counts number of words (and lines) in selected region in file
// if no selection, returns numbers for entire file.
func (ge *GideView) CountWordsRegion() string {
	av := ge.ActiveTextView()
	if av.Buf == nil || av.Buf.NLines <= 0 {
		return "empty"
	}
	if !av.HasSelection() {
		return ge.CountWords()
	}
	av.Buf.LinesMu.RLock()
	defer av.Buf.LinesMu.RUnlock()
	sel := av.Selection()
	words, lines := textbuf.CountWordsLinesRegion(av.Buf.Lines, sel.Reg)
	return fmt.Sprintf("File: %s  Words: %d   Lines: %d\n", giv.DirAndFile(string(av.Buf.Filename)), words, lines)
}

//////////////////////////////////////////////////////////////////////////////////////
//   Links

// TextLinkHandler is the GideView handler for text links -- preferred one b/c
// directly connects to correct GideView project
func TextLinkHandler(tl girl.TextLink) bool {
	ftv, _ := tl.Widget.Embed(giv.KiT_TextView).(*giv.TextView)
	gek := tl.Widget.ParentByType(KiT_GideView, true)
	if gek != nil {
		ge := gek.Embed(KiT_GideView).(*GideView)
		ur := tl.URL
		// todo: use net/url package for more systematic parsing
		switch {
		case strings.HasPrefix(ur, "find:///"):
			ge.OpenFindURL(ur, ftv)
		case strings.HasPrefix(ur, "file:///"):
			ge.OpenFileURL(ur, ftv)
		default:
			oswin.TheApp.OpenURL(ur)
		}
	} else {
		oswin.TheApp.OpenURL(tl.URL)
	}
	return true
}

// // URLHandler is the GideView handler for urls --
// func URLHandler(url string) bool {
// 	return true
// }

// OpenFileURL opens given file:/// url
func (ge *GideView) OpenFileURL(ur string, ftv *giv.TextView) bool {
	up, err := url.Parse(ur)
	if err != nil {
		log.Printf("GideView OpenFileURL parse err: %v\n", err)
		return false
	}
	fpath := up.Path[1:] // has double //
	cdpath := ""
	if ftv != nil && ftv.Buf != nil { // get cd path for non-pathed fnames
		cdln := ftv.Buf.BytesLine(0)
		if bytes.HasPrefix(cdln, []byte("cd ")) {
			fmidx := bytes.Index(cdln, []byte(" (from: "))
			if fmidx > 0 {
				cdpath = string(cdln[3:fmidx])
				dr, _ := filepath.Split(fpath)
				if dr == "" || !filepath.IsAbs(dr) {
					fpath = filepath.Join(cdpath, fpath)
				}
			}
		}
	}
	pos := up.Fragment
	tv, _, ok := ge.LinkViewFile(gi.FileName(fpath))
	if !ok {
		_, fnm := filepath.Split(fpath)
		tv, _, ok = ge.LinkViewFile(gi.FileName(fnm))
		if !ok {
			gi.PromptDialog(ge.Viewport, gi.DlgOpts{Title: "Couldn't Open File at Link", Prompt: fmt.Sprintf("Could not find or open file path in project: %v", fpath)}, gi.AddOk, gi.NoCancel, nil, nil)
			return false
		}
	}
	if pos == "" {
		return true
	}
	// fmt.Printf("pos: %v\n", pos)
	txpos := lex.Pos{}
	if txpos.FromString(pos) {
		reg := textbuf.Region{Start: txpos, End: lex.Pos{Ln: txpos.Ln, Ch: txpos.Ch + 4}}
		// todo: need some way of tagging the time stamp for adjusting!
		// reg = tv.Buf.AdjustReg(reg)
		txpos = reg.Start
		tv.HighlightRegion(reg)
		tv.SetCursorShow(txpos)
	}
	return true
}

//////////////////////////////////////////////////////////////////////////////////////
//   Close / Quit Req

// NChangedFiles returns number of opened files with unsaved changes
func (ge *GideView) NChangedFiles() int {
	return ge.OpenNodes.NChanged()
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
	gi.ChoiceDialog(ge.Viewport, gi.DlgOpts{Title: "Close Project: There are Unsaved Files",
		Prompt: fmt.Sprintf("In Project: %v There are <b>%v</b> opened files with <b>unsaved changes</b> -- do you want to save all or cancel closing this project and review  / save those files first?", ge.Nm, nch)},
		[]string{"Cancel", "Save All", "Close Without Saving"},
		ge.This(), func(recv, send ki.Ki, sig int64, data any) {
			switch sig {
			case 0:
				// do nothing, will have returned false already
			case 1:
				ge.SaveAllOpenNodes()
			case 2:
				ge.ParentWindow().OSWin.Close() // will not be prompted again!
			}
		})
	return false // not yet
}

// QuitReq is called when user tries to quit the app -- we go through all open
// main windows and look for gide windows and call their CloseWindowReq
// functions!
func QuitReq() bool {
	for _, win := range gi.MainWindows {
		if !strings.HasPrefix(win.Nm, "gide-") {
			continue
		}
		mfr, err := win.MainWidget()
		if err != nil {
			continue
		}
		gek := mfr.ChildByName("gide", 0)
		if gek == nil {
			continue
		}
		ge := gek.Embed(KiT_GideView).(*GideView)
		if !ge.CloseWindowReq() {
			return false
		}
	}
	return true
}

//////////////////////////////////////////////////////////////////////////////////////
//   Panels

// PanelIsOpen returns true if the given panel has not been collapsed and is avail
// and visible for displaying something
func (ge *GideView) PanelIsOpen(panel int) bool {
	sv := ge.SplitView()
	if panel < 0 || panel >= len(sv.Kids) {
		return false
	}
	if sv.Splits[panel] <= 0.01 {
		return false
	}
	return true
}

// CurPanel returns the splitter panel that currently has keyboard focus
func (ge *GideView) CurPanel() int {
	sv := ge.SplitView()
	for i, ski := range sv.Kids {
		_, sk := gi.KiToNode2D(ski)
		if sk.ContainsFocus() {
			return i
		}
	}
	return -1 // nobody
}

// FocusOnPanel moves keyboard focus to given panel -- returns false if nothing at that tab
func (ge *GideView) FocusOnPanel(panel int) bool {
	wupdt := ge.TopUpdateStart()
	defer ge.TopUpdateEnd(wupdt)

	sv := ge.SplitView()
	win := ge.ParentWindow()
	switch panel {
	case TextView1Idx:
		ge.SetActiveTextViewIdx(0)
	case TextView2Idx:
		ge.SetActiveTextViewIdx(1)
	case TabsIdx:
		tv := ge.Tabs()
		ct, _, has := tv.CurTab()
		if has {
			win.EventMgr.FocusNext(ct)
		} else {
			return false
		}
	default:
		ski := sv.Kids[panel]
		win.EventMgr.FocusNext(ski)
	}
	return true
}

// FocusNextPanel moves the keyboard focus to the next panel to the right
func (ge *GideView) FocusNextPanel() {
	sv := ge.SplitView()
	cp := ge.CurPanel()
	cp++
	np := len(sv.Kids)
	if cp >= np {
		cp = 0
	}
	for sv.Splits[cp] <= 0.01 {
		cp++
		if cp >= np {
			cp = 0
		}
	}
	ge.FocusOnPanel(cp)
}

// FocusPrevPanel moves the keyboard focus to the previous panel to the left
func (ge *GideView) FocusPrevPanel() {
	sv := ge.SplitView()
	cp := ge.CurPanel()
	cp--
	np := len(sv.Kids)
	if cp < 0 {
		cp = np - 1
	}
	for sv.Splits[cp] <= 0.01 {
		cp--
		if cp < 0 {
			cp = np - 1
		}
	}
	ge.FocusOnPanel(cp)
}

//////////////////////////////////////////////////////////////////////////////////////
//    Tabs

// TabByName returns a tab with given name, nil if not found.
func (ge *GideView) TabByName(label string) gi.Node2D {
	tv := ge.Tabs()
	return tv.TabByName(label)
}

// TabByNameTry returns a tab with given name, error if not found.
func (ge *GideView) TabByNameTry(label string) (gi.Node2D, error) {
	tv := ge.Tabs()
	return tv.TabByNameTry(label)
}

// SelectTabByName Selects given main tab, and returns all of its contents as well.
func (ge *GideView) SelectTabByName(label string) gi.Node2D {
	tv := ge.Tabs()
	if tv == nil {
		return nil
	}
	return tv.SelectTabByName(label)
}

// RecycleTabTextView returns a tab with given
// name, first by looking for an existing one, and if not found, making a new
// one with a Layout and then a TextView in it.  if sel, then select it.
// returns widget
func (ge *GideView) RecycleTabTextView(label string, sel bool) *giv.TextView {
	tv := ge.Tabs()
	if tv == nil {
		return nil
	}
	updt := tv.UpdateStart()
	tv.SetFullReRender()
	ly := tv.RecycleTab(label, gi.KiT_Layout, sel).Embed(gi.KiT_Layout).(*gi.Layout)
	txv := gide.ConfigOutputTextView(ly)
	tv.UpdateEnd(updt)
	return txv
}

// RecycleCmdBuf creates the buffer for command output, or returns
// existing. If clear is true, then any existing buffer is cleared.
// Returns true if new buffer created.
func (ge *GideView) RecycleCmdBuf(cmdNm string, clear bool) (*giv.TextBuf, bool) {
	if ge.CmdBufs == nil {
		ge.CmdBufs = make(map[string]*giv.TextBuf, 20)
	}
	if buf, has := ge.CmdBufs[cmdNm]; has {
		if clear {
			buf.New(0)
		}
		return buf, false
	}
	buf := &giv.TextBuf{}
	buf.InitName(buf, cmdNm+"-buf")
	buf.New(0)
	ge.CmdBufs[cmdNm] = buf
	buf.Autosave = false
	return buf, true
}

// RecycleCmdTab creates the tab to show command output, including making a
// buffer object to save output from the command. returns true if a new buffer
// was created, false if one already existed. if sel, select tab.  if clearBuf, then any
// existing buffer is cleared.  Also returns index of tab.
func (ge *GideView) RecycleCmdTab(cmdNm string, sel bool, clearBuf bool) (*giv.TextBuf, *giv.TextView, bool) {
	buf, nw := ge.RecycleCmdBuf(cmdNm, clearBuf)
	ctv := ge.RecycleTabTextView(cmdNm, sel)
	if ctv == nil {
		return nil, nil, false
	}
	ctv.SetInactive()
	ctv.SetBuf(buf)
	return buf, ctv, nw
}

// TabDeleted is called when a main tab is deleted -- we cancel any running commmands
func (ge *GideView) TabDeleted(tabnm string) {
	ge.RunningCmds.KillByName(tabnm)
}

//////////////////////////////////////////////////////////////////////////////////////
//    Commands / Tabs

// ExecCmdName executes command of given name -- this is the final common
// pathway for all command invokation except on a node.  if sel, select tab.
// if clearBuf, clear the buffer prior to command
func (ge *GideView) ExecCmdName(cmdNm gide.CmdName, sel bool, clearBuf bool) {
	cmd, _, ok := gide.AvailCmds.CmdByName(cmdNm, true)
	if !ok {
		return
	}
	ge.SetArgVarVals()
	cbuf, _, _ := ge.RecycleCmdTab(cmd.Name, sel, clearBuf)
	cmd.Run(ge, cbuf)
}

// ExecCmdNameFileNode executes command of given name on given node
func (ge *GideView) ExecCmdNameFileNode(fn *giv.FileNode, cmdNm gide.CmdName, sel bool, clearBuf bool) {
	cmd, _, ok := gide.AvailCmds.CmdByName(cmdNm, true)
	if !ok {
		return
	}
	ge.ArgVals.Set(string(fn.FPath), &ge.Prefs, nil)
	cbuf, _, _ := ge.RecycleCmdTab(cmd.Name, sel, clearBuf)
	cmd.Run(ge, cbuf)
}

// ExecCmdNameFileName executes command of given name on given file name
func (ge *GideView) ExecCmdNameFileName(fn string, cmdNm gide.CmdName, sel bool, clearBuf bool) {
	cmd, _, ok := gide.AvailCmds.CmdByName(cmdNm, true)
	if !ok {
		return
	}
	ge.ArgVals.Set(fn, &ge.Prefs, nil)
	cbuf, _, _ := ge.RecycleCmdTab(cmd.Name, sel, clearBuf)
	cmd.Run(ge, cbuf)
}

// ExecCmds gets list of available commands for current active file, as a submenu-func
func ExecCmds(it any, vp *gi.Viewport2D) [][]string {
	ge, ok := it.(ki.Ki).Embed(KiT_GideView).(*GideView)
	if !ok {
		return nil
	}
	tv := ge.ActiveTextView()
	if tv == nil {
		return nil
	}
	var cmds [][]string

	vc := ge.VersCtrl()
	if ge.ActiveLang == filecat.NoSupport {
		cmds = gide.AvailCmds.FilterCmdNames(ge.Prefs.MainLang, vc)
	} else {
		cmds = gide.AvailCmds.FilterCmdNames(ge.ActiveLang, vc)
	}
	return cmds
}

// ExecCmdNameActive calls given command on current active textview
func (ge *GideView) ExecCmdNameActive(cmdNm string) {
	tv := ge.ActiveTextView()
	if tv == nil {
		return
	}
	ge.SaveAllCheck(true, func() { // true = cancel option
		ge.ExecCmdName(gide.CmdName(cmdNm), true, true)
	})
}

// ExecCmd pops up a menu to select a command appropriate for the current
// active text view, and shows output in Tab with name of command
func (ge *GideView) ExecCmd() {
	tv := ge.ActiveTextView()
	if tv == nil {
		fmt.Printf("no Active view for ExecCmd\n")
		return
	}
	var cmds [][]string
	vc := ge.VersCtrl()
	if ge.ActiveLang == filecat.NoSupport {
		cmds = gide.AvailCmds.FilterCmdNames(ge.Prefs.MainLang, vc)
	} else {
		cmds = gide.AvailCmds.FilterCmdNames(ge.ActiveLang, vc)
	}
	hsz := len(ge.CmdHistory)
	lastCmd := ""
	if hsz > 0 {
		lastCmd = string(ge.CmdHistory[hsz-1])
	}
	gi.SubStringsChooserPopup(cmds, lastCmd, tv, func(recv, send ki.Ki, sig int64, data any) {
		didx := data.([]int)
		si := didx[0]
		ii := didx[1]
		cmdCat := cmds[si][0]
		cmdNm := gide.CmdName(gide.CommandName(cmdCat, cmds[si][ii]))
		ge.CmdHistory.Add(cmdNm)       // only save commands executed via chooser
		ge.SaveAllCheck(true, func() { // true = cancel option
			ge.ExecCmdName(cmdNm, true, true) // sel, clear
		})
	})
}

// ExecCmdFileNode pops up a menu to select a command appropriate for the given node,
// and shows output in Tab with name of command
func (ge *GideView) ExecCmdFileNode(fn *giv.FileNode) {
	lang := fn.Info.Sup
	vc := ge.VersCtrl()
	cmds := gide.AvailCmds.FilterCmdNames(lang, vc)
	gi.SubStringsChooserPopup(cmds, "", ge, func(recv, send ki.Ki, sig int64, data any) {
		didx := data.([]int)
		si := didx[0]
		ii := didx[1]
		cmdCat := cmds[si][0]
		cmdNm := gide.CmdName(gide.CommandName(cmdCat, cmds[si][ii]))
		ge.ExecCmdNameFileNode(fn, cmdNm, true, true) // sel, clearbuf
	})
}

// SetArgVarVals sets the ArgVar values for commands, from GideView values
func (ge *GideView) SetArgVarVals() {
	tv := ge.ActiveTextView()
	tve := tv.Embed(giv.KiT_TextView).(*giv.TextView)
	if tv == nil || tv.Buf == nil {
		ge.ArgVals.Set("", &ge.Prefs, tve)
	} else {
		ge.ArgVals.Set(string(tv.Buf.Filename), &ge.Prefs, tve)
	}
}

// ExecCmds executes a sequence of commands, sel = select tab, clearBuf = clear buffer
func (ge *GideView) ExecCmds(cmdNms gide.CmdNames, sel bool, clearBuf bool) {
	for _, cmdNm := range cmdNms {
		ge.ExecCmdName(cmdNm, sel, clearBuf)
	}
}

// ExecCmdsFileNode executes a sequence of commands on file node, sel = select tab, clearBuf = clear buffer
func (ge *GideView) ExecCmdsFileNode(fn *giv.FileNode, cmdNms gide.CmdNames, sel bool, clearBuf bool) {
	for _, cmdNm := range cmdNms {
		ge.ExecCmdNameFileNode(fn, cmdNm, sel, clearBuf)
	}
}

// Build runs the BuildCmds set for this project
func (ge *GideView) Build() {
	if len(ge.Prefs.BuildCmds) == 0 {
		gi.PromptDialog(ge.Viewport, gi.DlgOpts{Title: "No BuildCmds Set", Prompt: fmt.Sprintf("You need to set the BuildCmds in the Project Preferences")}, gi.AddOk, gi.NoCancel, nil, nil)
		return
	}
	ge.SaveAllCheck(true, func() { // true = cancel option
		ge.ExecCmds(ge.Prefs.BuildCmds, true, true)
	})
}

// Run runs the RunCmds set for this project
func (ge *GideView) Run() {
	if len(ge.Prefs.RunCmds) == 0 {
		gi.PromptDialog(ge.Viewport, gi.DlgOpts{Title: "No RunCmds Set", Prompt: fmt.Sprintf("You need to set the RunCmds in the Project Preferences")}, gi.AddOk, gi.NoCancel, nil, nil)
		return
	}
	if ge.Prefs.RunCmds[0] == "Run Proj" && !ge.Prefs.RunExecIsExec() {
		giv.CallMethod(ge, "ChooseRunExec", ge.Viewport)
		return
	}
	ge.ExecCmds(ge.Prefs.RunCmds, true, true)
}

// Commit commits the current changes using relevant VCS tool.
// Checks for VCS setting and for unsaved files.
func (ge *GideView) Commit() {
	vc := ge.VersCtrl()
	if vc == "" {
		gi.PromptDialog(ge.Viewport, gi.DlgOpts{Title: "No Version Control System Found", Prompt: fmt.Sprintf("No version control system detected in file system, or defined in project prefs -- define in project prefs if viewing a sub-directory within a larger repository")}, gi.AddOk, gi.NoCancel, nil, nil)
		return
	}
	ge.SaveAllCheck(true, func() { // true = cancel option
		ge.CommitNoChecks()
	})
}

// CommitNoChecks does the commit without any further checks for VCS, and unsaved files
func (ge *GideView) CommitNoChecks() {
	vc := ge.VersCtrl()
	cmds := gide.AvailCmds.FilterCmdNames(ge.ActiveLang, vc)
	cmdnm := ""
	for _, ct := range cmds {
		if len(ct) < 2 {
			continue
		}
		if !giv.IsVersCtrlSystem(ct[0]) {
			continue
		}
		for _, cm := range ct {
			if strings.Contains(cm, "Commit") {
				cmdnm = gide.CommandName(ct[0], cm)
				break
			}
		}
	}
	if cmdnm == "" {
		gi.PromptDialog(ge.Viewport, gi.DlgOpts{Title: "No Commit command found", Prompt: fmt.Sprintf("Could not find Commit command in list of avail commands -- this is usually a programmer error -- check preferences settings etc")}, gi.AddOk, gi.NoCancel, nil, nil)
		return
	}
	ge.SetArgVarVals() // need to set before setting prompt string below..

	gi.StringPromptDialog(ge.Viewport, "", "Enter commit message here..",
		gi.DlgOpts{Title: "Commit Message", Prompt: "Please enter your commit message here -- remember this is essential front-line documentation.  Author information comes from User settings in GoGi Preferences."},
		ge.This(), func(recv, send ki.Ki, sig int64, data any) {
			dlg := send.(*gi.Dialog)
			if sig == int64(gi.DialogAccepted) {
				msg := gi.StringPromptDialogValue(dlg)
				ge.ArgVals["{PromptString1}"] = msg
				gide.CmdNoUserPrompt = true                     // don't re-prompt!
				ge.ExecCmdName(gide.CmdName(cmdnm), true, true) // must be wait
				ge.SaveProjIfExists(true)                       // saveall
				ge.UpdateFiles()
			}
		})
}

// VCSUpdateAll does an Update (e.g., Pull) on all VCS repositories within
// the open tree nodes in FileTree.
func (ge *GideView) VCSUpdateAll() {
	updt := ge.FilesView.UpdateStart()
	ge.FilesView.SetFullReRender()
	ge.Files.UpdateAllVcs()
	ge.FilesView.UpdateEnd(updt)
}

// VCSLog shows the VCS log of commits for this file, optionally with a
// since date qualifier: If since is non-empty, it should be
// a date-like expression that the VCS will understand, such as
// 1/1/2020, yesterday, last year, etc.  SVN only understands a
// number as a maximum number of items to return.
// If allFiles is true, then the log will show revisions for all files, not just
// this one.
// Returns the Log and also shows it in a VCSLogView which supports further actions.
func (ge *GideView) VCSLog(since string) (vci.Log, error) {
	atv := ge.ActiveTextView()
	ond, _, got := ge.OpenNodeForTextView(atv)
	if !got {
		if ge.Files.DirRepo != nil {
			return ge.Files.LogVcs(true, since)
		}
		gi.PromptDialog(ge.Viewport, gi.DlgOpts{Title: "No VCS Repository", Prompt: "No VCS Repository found in current active file or Root path: Open a file in a repository and try again"}, gi.AddOk, gi.NoCancel, nil, nil)
		return nil, errors.New("No VCS Repository found in current active file or Root path")
	}
	return ond.LogVcs(true, since)
}

// OpenConsoleTab opens a main tab displaying console output (stdout, stderr)
func (ge *GideView) OpenConsoleTab() {
	ctv := ge.RecycleTabTextView("Console", true)
	if ctv == nil {
		return
	}
	ctv.SetInactive()
	if ctv.Buf == nil || ctv.Buf != gide.TheConsole.Buf {
		ctv.SetBuf(gide.TheConsole.Buf)
		gide.TheConsole.Buf.TextBufSig.Connect(ge.This(), func(recv, send ki.Ki, sig int64, data any) {
			gee, _ := recv.Embed(KiT_GideView).(*GideView)
			gee.SelectTabByName("Console")
		})
	}
}

//////////////////////////////////////////////////////////////////////////////////////
//    TextView functions

// CursorToHistPrev moves cursor to previous position on history list --
// returns true if moved
func (ge *GideView) CursorToHistPrev() bool {
	tv := ge.ActiveTextView()
	return tv.CursorToHistPrev()
}

// CursorToHistNext moves cursor to next position on history list --
// returns true if moved
func (ge *GideView) CursorToHistNext() bool {
	tv := ge.ActiveTextView()
	return tv.CursorToHistNext()
}

// LookupFun is the completion system Lookup function that makes a custom
// textview dialog that has option to edit resulting file.
func (ge *GideView) LookupFun(data any, text string, posLn, posCh int) (ld complete.Lookup) {
	sfs := data.(*pi.FileStates)
	if sfs == nil {
		log.Printf("LookupFun: data is nil not FileStates or is nil - can't lookup\n")
		return ld
	}
	lp, err := pi.LangSupport.Props(sfs.Sup)
	if err != nil {
		log.Printf("LookupFun: %v\n", err)
		return ld
	}
	if lp.Lang == nil {
		return ld
	}

	// note: must have this set to ture to allow viewing of AST
	// must set it in pi/parse directly -- so it is changed in the fileparse too
	parse.GuiActive = true // note: this is key for debugging -- runs slower but makes the tree unique

	ld = lp.Lang.Lookup(sfs, text, lex.Pos{posLn, posCh})
	if len(ld.Text) > 0 {
		giv.TextViewDialog(nil, ld.Text, giv.DlgOpts{Title: "Lookup: " + text, Data: text})
		return ld
	}
	if ld.Filename == "" {
		return ld
	}

	txt, err := textbuf.FileBytes(ld.Filename)
	if err != nil {
		return ld
	}
	if ld.StLine > 0 {
		lns := bytes.Split(txt, []byte("\n"))
		comLn, comSt, comEd := textbuf.SupportedComments(ld.Filename)
		ld.StLine = textbuf.PreCommentStart(lns, ld.StLine, comLn, comSt, comEd, 10) // just go back 10 max
	}

	prmpt := ""
	if ld.EdLine > ld.StLine {
		prmpt = fmt.Sprintf("%v [%d -- %d]", ld.Filename, ld.StLine, ld.EdLine)
	} else {
		prmpt = fmt.Sprintf("%v:%d", ld.Filename, ld.StLine)
	}
	opts := giv.DlgOpts{Title: "Lookup: " + text, Prompt: prmpt}

	dlg, recyc := gi.RecycleStdDialog(prmpt, opts.ToGiOpts(), gi.NoOk, gi.NoCancel)
	if recyc {
		return ld
	}
	frame := dlg.Frame()
	_, prIdx := dlg.PromptWidget(frame)

	tb := &giv.TextBuf{}
	tb.InitName(tb, "text-view-dialog-buf")
	tb.Filename = gi.FileName(ld.Filename)
	tb.Hi.Style = gi.Prefs.Colors.HiStyle
	tb.Opts.LineNos = ge.Prefs.Editor.LineNos
	tb.Stat() // update markup

	tlv := frame.InsertNewChild(gi.KiT_Layout, prIdx+1, "text-lay").(*gi.Layout)
	tlv.SetProp("width", units.NewCh(80))
	tlv.SetProp("height", units.NewEm(40))
	tlv.SetStretchMax()
	tv := giv.AddNewTextView(tlv, "text-view")
	tv.Viewport = dlg.Embed(gi.KiT_Viewport2D).(*gi.Viewport2D)
	tv.SetInactive()
	tv.SetProp("font-family", gi.Prefs.MonoFont)
	tv.SetBuf(tb)
	tv.ScrollToCursorPos = lex.Pos{Ln: ld.StLine}
	tv.ScrollToCursorOnRender = true

	tb.SetText(txt) // calls remarkup

	bbox, _ := dlg.ButtonBox(frame)
	if bbox == nil {
		bbox = dlg.AddButtonBox(frame)
	}
	ofb := gi.AddNewButton(bbox, "open-file")
	ofb.SetText("Open File")
	ofb.SetIcon("file-open")
	ofb.ButtonSig.Connect(dlg.This(), func(recv, send ki.Ki, sig int64, data any) {
		if sig == int64(gi.ButtonClicked) {
			ge.ViewFile(gi.FileName(ld.Filename))
			dlg.Close()
		}
	})
	cpb := gi.AddNewButton(bbox, "copy-to-clip")
	cpb.SetText("Copy To Clipboard")
	cpb.SetIcon("copy")
	cpb.ButtonSig.Connect(dlg.This(), func(recv, send ki.Ki, sig int64, data any) {
		if sig == int64(gi.ButtonClicked) {
			ddlg := recv.Embed(gi.KiT_Dialog).(*gi.Dialog)
			oswin.TheApp.ClipBoard(ddlg.Win.OSWin).Write(mimedata.NewTextBytes(txt))
		}
	})
	dlg.UpdateEndNoSig(true) // going to be shown
	dlg.Open(0, 0, ge.Viewport, nil)
	return ld
}

//////////////////////////////////////////////////////////////////////////////////////
//    Find / Replace

// Find does Find / Replace in files, using given options and filters -- opens up a
// main tab with the results and further controls.
func (ge *GideView) Find(find, repl string, ignoreCase, regExp bool, loc gide.FindLoc, langs []filecat.Supported) {
	if find == "" {
		return
	}
	ge.Prefs.Find.IgnoreCase = ignoreCase
	ge.Prefs.Find.Langs = langs
	ge.Prefs.Find.Loc = loc

	tv := ge.Tabs()
	if tv == nil {
		return
	}
	updt := tv.UpdateStart()
	fbuf, _ := ge.RecycleCmdBuf("Find", true)
	fv := tv.RecycleTab("Find", gide.KiT_FindView, true).Embed(gide.KiT_FindView).(*gide.FindView)
	fv.Config(ge)
	fv.Time = time.Now()
	ftv := fv.TextView()
	ftv.SetInactive()
	ftv.SetBuf(fbuf)

	fv.SaveFindString(find)
	fv.SaveReplString(repl)

	root := ge.Files.Embed(giv.KiT_FileNode).(*giv.FileNode)

	atv := ge.ActiveTextView()
	ond, _, got := ge.OpenNodeForTextView(atv)
	adir := ""
	if got {
		adir, _ = filepath.Split(string(ond.FPath))
	}

	var res []gide.FileSearchResults
	if loc == gide.FindLocFile {
		if got {
			if regExp {
				re, err := regexp.Compile(find)
				if err != nil {
					log.Println(err)
				} else {
					cnt, matches := atv.Buf.SearchRegexp(re)
					res = append(res, gide.FileSearchResults{ond, cnt, matches})
				}
			} else {
				cnt, matches := atv.Buf.Search([]byte(find), ignoreCase, false)
				res = append(res, gide.FileSearchResults{ond, cnt, matches})
			}
		}
	} else {
		res = gide.FileTreeSearch(root, find, ignoreCase, regExp, loc, adir, langs)
	}
	fv.ShowResults(res)
	tv.UpdateEnd(updt)
	ge.FocusOnPanel(TabsIdx)
}

// Spell checks spelling in active text view
func (ge *GideView) Spell() {
	txv := ge.ActiveTextView()
	if txv == nil || txv.Buf == nil {
		return
	}
	spell.OpenCheck() // make sure latest file opened
	tv := ge.Tabs()
	if tv == nil {
		return
	}
	updt := tv.UpdateStart()
	sv := tv.RecycleTab("Spell", gide.KiT_SpellView, true).Embed(gide.KiT_SpellView).(*gide.SpellView)
	sv.Config(ge, txv)
	tv.UpdateEnd(updt)
	ge.FocusOnPanel(TabsIdx)
}

// Symbols displays the Symbols of a file or package
func (ge *GideView) Symbols() {
	txv := ge.ActiveTextView()
	if txv == nil || txv.Buf == nil {
		return
	}
	tv := ge.Tabs()
	if tv == nil {
		return
	}
	updt := tv.UpdateStart()
	sv := tv.RecycleTab("Symbols", gide.KiT_SymbolsView, true).Embed(gide.KiT_SymbolsView).(*gide.SymbolsView)
	sv.Config(ge, ge.Prefs.Symbols)
	tv.UpdateEnd(updt)
	ge.FocusOnPanel(TabsIdx)
}

// Debug starts the debugger on the RunExec executable.
func (ge *GideView) Debug() {
	tv := ge.Tabs()
	if tv == nil {
		return
	}
	updt := tv.UpdateStart()
	ge.Prefs.Debug.Mode = gidebug.Exec
	exePath := string(ge.Prefs.RunExec)
	exe := filepath.Base(exePath)
	dv := tv.RecycleTab("Debug "+exe, gide.KiT_DebugView, true).Embed(gide.KiT_DebugView).(*gide.DebugView)
	dv.Config(ge, ge.Prefs.MainLang, exePath)
	tv.UpdateEnd(updt)
	ge.FocusOnPanel(TabsIdx)
	ge.CurDbg = dv
}

// DebugTest runs the debugger using testing mode in current active textview path
func (ge *GideView) DebugTest() {
	txv := ge.ActiveTextView()
	if txv == nil || txv.Buf == nil {
		return
	}
	tv := ge.Tabs()
	if tv == nil {
		return
	}
	updt := tv.UpdateStart()
	ge.Prefs.Debug.Mode = gidebug.Test
	tstPath := string(txv.Buf.Filename)
	dir := filepath.Base(filepath.Dir(tstPath))
	dv := tv.RecycleTab("Debug "+dir, gide.KiT_DebugView, true).Embed(gide.KiT_DebugView).(*gide.DebugView)
	dv.Config(ge, ge.Prefs.MainLang, tstPath)
	tv.UpdateEnd(updt)
	ge.FocusOnPanel(TabsIdx)
	ge.CurDbg = dv
}

// DebugAttach runs the debugger by attaching to an already-running process.
// pid is the process id to attach to.
func (ge *GideView) DebugAttach(pid uint64) {
	tv := ge.Tabs()
	if tv == nil {
		return
	}
	updt := tv.UpdateStart()
	ge.Prefs.Debug.Mode = gidebug.Attach
	ge.Prefs.Debug.PID = pid
	exePath := string(ge.Prefs.RunExec)
	exe := filepath.Base(exePath)
	dv := tv.RecycleTab("Debug "+exe, gide.KiT_DebugView, true).Embed(gide.KiT_DebugView).(*gide.DebugView)
	dv.Config(ge, ge.Prefs.MainLang, exePath)
	tv.UpdateEnd(updt)
	ge.FocusOnPanel(TabsIdx)
	ge.CurDbg = dv
}

// CurDebug returns the current debug view
func (ge *GideView) CurDebug() *gide.DebugView {
	return ge.CurDbg
}

// ClearDebug clears the current debugger setting -- no more debugger active.
func (ge *GideView) ClearDebug() {
	ge.CurDbg = nil
}

// ChooseRunExec selects the executable to run for the project
func (ge *GideView) ChooseRunExec(exePath gi.FileName) {
	if exePath != "" {
		ge.Prefs.RunExec = exePath
		ge.Prefs.BuildDir = gi.FileName(filepath.Dir(string(exePath)))
		if !ge.Prefs.RunExecIsExec() {
			gi.PromptDialog(ge.Viewport, gi.DlgOpts{Title: "Not Executable", Prompt: fmt.Sprintf("RunExec file: %v is not exectable", exePath)}, gi.AddOk, gi.NoCancel, nil, nil)
		}
	}
}

// ParseOpenFindURL parses and opens given find:/// url from Find, return text
// region encoded in url, and starting line of results in find buffer, and
// number of results returned -- for parsing all the find results
func (ge *GideView) ParseOpenFindURL(ur string, ftv *giv.TextView) (tv *gide.TextView, reg textbuf.Region, findBufStLn, findCount int, ok bool) {
	up, err := url.Parse(ur)
	if err != nil {
		log.Printf("FindView OpenFindURL parse err: %v\n", err)
		return
	}
	fpath := up.Path[1:] // has double //
	pos := up.Fragment
	tv, _, ok = ge.LinkViewFile(gi.FileName(fpath))
	if !ok {
		gi.PromptDialog(ge.Viewport, gi.DlgOpts{Title: "Couldn't Open File at Link", Prompt: fmt.Sprintf("Could not find or open file path in project: %v", fpath)}, gi.AddOk, gi.NoCancel, nil, nil)
		return
	}
	if pos == "" {
		return
	}

	lidx := strings.Index(pos, "L")
	if lidx > 0 {
		reg.FromString(pos[lidx:])
		pos = pos[:lidx]
	}
	fmt.Sscanf(pos, "R%dN%d", &findBufStLn, &findCount)
	return
}

// OpenFindURL opens given find:/// url from Find -- delegates to FindView
func (ge *GideView) OpenFindURL(ur string, ftv *giv.TextView) bool {
	fvk := ftv.ParentByType(gide.KiT_FindView, true)
	if fvk == nil {
		return false
	}
	fv := fvk.(*gide.FindView)
	return fv.OpenFindURL(ur, ftv)
}

// ReplaceInActive does query-replace in active file only
func (ge *GideView) ReplaceInActive() {
	tv := ge.ActiveTextView()
	tv.QReplacePrompt()
}

func (ge *GideView) OpenFileAtRegion(filename gi.FileName, tr textbuf.Region) (tv *gide.TextView, ok bool) {
	tv, _, ok = ge.LinkViewFile(filename)
	if tv != nil {
		tv.UpdateStart()
		tv.Highlights = tv.Highlights[:0]
		tv.Highlights = append(tv.Highlights, tr)
		tv.UpdateEnd(true)
		tv.RefreshIfNeeded()
		tv.SetCursorShow(tr.Start)
		tv.GrabFocus()
		return tv, true

	}
	return nil, false
}

//////////////////////////////////////////////////////////////////////////////////////
//    Rects, Registers

// CutRect cuts rectangle in active text view
func (ge *GideView) CutRect() {
	tv := ge.ActiveTextView()
	if tv.Buf == nil {
		return
	}
	tv.CutRect()
}

// CopyRect copies rectangle in active text view
func (ge *GideView) CopyRect() {
	tv := ge.ActiveTextView()
	if tv.Buf == nil {
		return
	}
	tv.CopyRect(true)
}

// PasteRect cuts rectangle in active text view
func (ge *GideView) PasteRect() {
	tv := ge.ActiveTextView()
	if tv.Buf == nil {
		return
	}
	tv.PasteRect()
}

// RegisterCopy saves current selection in active text view to register of given name
// returns true if saved
func (ge *GideView) RegisterCopy(name string) bool {
	if name == "" {
		return false
	}
	tv := ge.ActiveTextView()
	if tv.Buf == nil {
		return false
	}
	sel := tv.Selection()
	if sel == nil {
		return false
	}
	if gide.AvailRegisters == nil {
		gide.AvailRegisters = make(gide.Registers, 100)
	}
	gide.AvailRegisters[name] = string(sel.ToBytes())
	gide.AvailRegisters.SavePrefs()
	ge.Prefs.Register = gide.RegisterName(name)
	tv.SelectReset()
	return true
}

// RegisterPaste pastes register of given name into active text view
// returns true if pasted
func (ge *GideView) RegisterPaste(name gide.RegisterName) bool {
	if name == "" {
		return false
	}
	str, ok := gide.AvailRegisters[string(name)]
	if !ok {
		return false
	}
	tv := ge.ActiveTextView()
	if tv.Buf == nil {
		return false
	}
	tv.InsertAtCursor([]byte(str))
	ge.Prefs.Register = name
	return true
}

// CommentOut comments-out selected lines in active text view
// and uncomments if already commented
// If multiple lines are selected and any line is uncommented all will be commented
func (ge *GideView) CommentOut() bool {
	tv := ge.ActiveTextView()
	if tv.Buf == nil {
		return false
	}
	sel := tv.Selection()
	var stl, etl int
	if sel == nil {
		stl = tv.CursorPos.Ln
		etl = stl + 1
	} else {
		stl = sel.Reg.Start.Ln
		etl = sel.Reg.End.Ln
	}
	tv.Buf.CommentRegion(stl, etl)
	tv.SelectReset()
	return true
}

// Indent indents selected lines in active view
func (ge *GideView) Indent() bool {
	tv := ge.ActiveTextView()
	if tv.Buf == nil {
		return false
	}
	sel := tv.Selection()
	if sel == nil {
		return false
	}
	tv.Buf.AutoIndentRegion(sel.Reg.Start.Ln, sel.Reg.End.Ln)
	tv.SelectReset()
	return true
}

// ReCase replaces currently selected text in current active view with given case
func (ge *GideView) ReCase(c textbuf.Cases) string {
	tv := ge.ActiveTextView()
	if tv.Buf == nil {
		return ""
	}
	return tv.ReCaseSelection(c)
}

// JoinParaLines merges sequences of lines with hard returns forming paragraphs,
// separated by blank lines, into a single line per paragraph,
// for given selected region (full text if no selection)
func (ge *GideView) JoinParaLines() {
	tv := ge.ActiveTextView()
	if tv.Buf == nil {
		return
	}
	if tv.HasSelection() {
		tv.Buf.JoinParaLines(tv.SelectReg.Start.Ln, tv.SelectReg.End.Ln)
	} else {
		tv.Buf.JoinParaLines(0, tv.NLines-1)
	}
}

// TabsToSpaces converts tabs to spaces
// for given selected region (full text if no selection)
func (ge *GideView) TabsToSpaces() {
	tv := ge.ActiveTextView()
	if tv.Buf == nil {
		return
	}
	if tv.HasSelection() {
		tv.Buf.TabsToSpacesRegion(tv.SelectReg.Start.Ln, tv.SelectReg.End.Ln)
	} else {
		tv.Buf.TabsToSpacesRegion(0, tv.NLines-1)
	}
}

// SpacesToTabs converts spaces to tabs
// for given selected region (full text if no selection)
func (ge *GideView) SpacesToTabs() {
	tv := ge.ActiveTextView()
	if tv.Buf == nil {
		return
	}
	if tv.HasSelection() {
		tv.Buf.SpacesToTabsRegion(tv.SelectReg.Start.Ln, tv.SelectReg.End.Ln)
	} else {
		tv.Buf.SpacesToTabsRegion(0, tv.NLines-1)
	}
}

//////////////////////////////////////////////////////////////////////////////////////
//    StatusBar

// SetStatus updates the statusbar label with given message, along with other status info
func (ge *GideView) SetStatus(msg string) {
	sb := ge.StatusBar()
	if sb == nil {
		return
	}
	// ge.UpdtMu.Lock()
	// defer ge.UpdtMu.Unlock()

	updt := sb.UpdateStart()
	lbl := ge.StatusLabel()
	fnm := ""
	ln := 0
	ch := 0
	tv := ge.ActiveTextView()
	if tv != nil {
		ln = tv.CursorPos.Ln + 1
		ch = tv.CursorPos.Ch
		if tv.Buf != nil {
			fnm = ge.Files.RelPath(tv.Buf.Filename)
			if tv.Buf.IsChanged() {
				fnm += "*"
			}
			if tv.Buf.Info.Sup != filecat.NoSupport {
				fnm += " (" + tv.Buf.Info.Sup.String() + ")"
			}
		}
		if tv.ISearch.On {
			msg = fmt.Sprintf("\tISearch: %v (n=%v)\t%v", tv.ISearch.Find, len(tv.ISearch.Matches), msg)
		}
		if tv.QReplace.On {
			msg = fmt.Sprintf("\tQReplace: %v -> %v (n=%v)\t%v", tv.QReplace.Find, tv.QReplace.Replace, len(tv.QReplace.Matches), msg)
		}
	}

	str := fmt.Sprintf("%s\t%s\t<b>%s:</b>\t(%d,%d)\t%s", ge.Nm, ge.ActiveVCSInfo, fnm, ln, ch, msg)
	lbl.SetText(str)
	sb.UpdateEnd(updt)
	ge.UpdateTextButtons()
}

//////////////////////////////////////////////////////////////////////////////////////
//    Defaults, Prefs

// Defaults sets new project defaults based on overall preferences
func (ge *GideView) Defaults() {
	ge.Prefs.Files = gide.Prefs.Files
	ge.Prefs.Editor = gi.Prefs.Editor
	ge.Prefs.Splits = []float32{.1, .325, .325, .25}
	ge.Prefs.Debug = gidebug.DefaultParams
	ge.Files.DirsOnTop = ge.Prefs.Files.DirsOnTop
	ge.Files.NodeType = gide.KiT_FileNode
}

// GrabPrefs grabs the current project preference settings from various
// places, e.g., prior to saving or editing.
func (ge *GideView) GrabPrefs() {
	sv := ge.SplitView()
	ge.Prefs.Splits = sv.Splits
	ge.Prefs.Dirs = ge.Files.Dirs
}

// ApplyPrefs applies current project preference settings into places where
// they are used -- only for those done prior to loading
func (ge *GideView) ApplyPrefs() {
	ge.ProjFilename = ge.Prefs.ProjFilename
	ge.ProjRoot = ge.Prefs.ProjRoot
	ge.Files.Dirs = ge.Prefs.Dirs
	ge.Files.DirsOnTop = ge.Prefs.Files.DirsOnTop
	if len(ge.Kids) > 0 {
		for i := 0; i < NTextViews; i++ {
			tv := ge.TextViewByIndex(i)
			if tv.Buf != nil {
				ge.ConfigTextBuf(tv.Buf)
			}
		}
		for _, ond := range ge.OpenNodes {
			if ond.Buf != nil {
				ge.ConfigTextBuf(ond.Buf)
			}
		}
	}
}

// ApplyPrefsAction applies current preferences to the project, and updates the project
func (ge *GideView) ApplyPrefsAction() {
	ge.ApplyPrefs()
	ge.SetFullReRender()
	ge.ConfigTextViews()
	ge.SplitsSetView(ge.Prefs.SplitName)
	ge.SetStatus("Applied prefs")
}

// EditProjPrefs allows editing of project preferences (settings specific to this project)
func (ge *GideView) EditProjPrefs() {
	sv, _ := gide.ProjPrefsView(&ge.Prefs)
	// we connect to changes and apply them
	sv.ViewSig.Connect(ge.This(), func(recv, send ki.Ki, sig int64, data any) {
		gee, _ := recv.Embed(KiT_GideView).(*GideView)
		gee.ApplyPrefsAction()
	})
}

// SplitsSetView sets split view splitters to given named setting
func (ge *GideView) SplitsSetView(split gide.SplitName) {
	sv := ge.SplitView()
	sp, _, ok := gide.AvailSplits.SplitByName(split)
	if ok {
		sv.SetSplitsAction(sp.Splits...)
		ge.Prefs.SplitName = split
		if !ge.PanelIsOpen(ge.ActiveTextViewIdx + TextView1Idx) {
			ge.SetActiveTextViewIdx((ge.ActiveTextViewIdx + 1) % 2)
		}
	}
}

// SplitsSave saves current splitter settings to named splitter settings under
// existing name, and saves to prefs file
func (ge *GideView) SplitsSave(split gide.SplitName) {
	sv := ge.SplitView()
	sp, _, ok := gide.AvailSplits.SplitByName(split)
	if ok {
		sp.SaveSplits(sv.Splits)
		gide.AvailSplits.SavePrefs()
	}
}

// SplitsSaveAs saves current splitter settings to new named splitter settings, and
// saves to prefs file
func (ge *GideView) SplitsSaveAs(name, desc string) {
	sv := ge.SplitView()
	gide.AvailSplits.Add(name, desc, sv.Splits)
	gide.AvailSplits.SavePrefs()
}

// SplitsEdit opens the SplitsView editor to customize saved splitter settings
func (ge *GideView) SplitsEdit() {
	gide.SplitsView(&gide.AvailSplits)
}

// HelpWiki opens wiki page for gide on github
func (ge *GideView) HelpWiki() {
	oswin.TheApp.OpenURL("https://goki.dev/gide/wiki")
}

//////////////////////////////////////////////////////////////////////////////////////
//   GUI configs

// Config configures the view
func (ge *GideView) Config() {
	if ge.HasChildren() {
		return
	}
	updt := ge.UpdateStart()
	ge.Lay = gi.LayoutVert
	// ge.SetProp("spacing", gi.StdDialogVSpaceUnits)
	gi.AddNewToolBar(ge, "toolbar")
	gi.AddNewSplitView(ge, "splitview")
	gi.AddNewFrame(ge, "statusbar", gi.LayoutHoriz)

	ge.UpdateFiles()
	ge.ConfigSplitView()
	ge.ConfigToolbar()
	ge.ConfigStatusBar()

	ge.SetStatus("just updated")

	ge.OpenConsoleTab()

	ge.UpdateEnd(updt)
}

// IsConfiged returns true if the view is fully configured
func (ge *GideView) IsConfiged() bool {
	if !ge.HasChildren() {
		return false
	}
	sv := ge.SplitView()
	if !sv.HasChildren() {
		return false
	}
	return true
}

// SplitView returns the main SplitView
func (ge *GideView) SplitView() *gi.SplitView {
	spi := ge.ChildByName("splitview", 2)
	if spi == nil {
		return nil
	}
	return spi.(*gi.SplitView)
}

// FileTree returns the main FileTreeView
func (ge *GideView) FileTreeView() *gide.FileTreeView {
	return ge.SplitView().Child(FileTreeIdx).Child(0).(*gide.FileTreeView)
}

// TextViewByIndex returns the TextView by index (0 or 1), nil if not found
func (ge *GideView) TextViewByIndex(idx int) *gide.TextView {
	split := ge.SplitView()
	svk := split.Child(TextView1Idx + idx).Child(1).Child(0)
	return svk.Embed(gide.KiT_TextView).(*gide.TextView)
}

// TextViewButtonByIndex returns the top textview menu button by index (0 or 1)
func (ge *GideView) TextViewButtonByIndex(idx int) *gi.MenuButton {
	split := ge.SplitView()
	svk := split.Child(TextView1Idx + idx).Child(0).Child(0)
	return svk.Embed(gi.KiT_MenuButton).(*gi.MenuButton)
}

// Tabs returns the main TabView
func (ge *GideView) Tabs() *gi.TabView {
	split := ge.SplitView()
	if split == nil {
		return nil
	}
	tv := split.Child(TabsIdx).Embed(gi.KiT_TabView).(*gi.TabView)
	return tv
}

// ToolBar returns the main toolbar
func (ge *GideView) ToolBar() *gi.ToolBar {
	tbk := ge.ChildByName("toolbar", 2)
	if tbk == nil {
		return nil
	}
	return tbk.(*gi.ToolBar)
}

// StatusBar returns the statusbar widget
func (ge *GideView) StatusBar() *gi.Frame {
	if ge.This() == nil || ge.IsDeleted() || ge.IsDestroyed() || !ge.HasChildren() {
		return nil
	}
	return ge.ChildByName("statusbar", 2).(*gi.Frame)
}

// StatusLabel returns the statusbar label widget
func (ge *GideView) StatusLabel() *gi.Label {
	return ge.StatusBar().Child(0).Embed(gi.KiT_Label).(*gi.Label)
}

// ConfigStatusBar configures statusbar with label
func (ge *GideView) ConfigStatusBar() {
	sb := ge.StatusBar()
	if sb == nil || sb.HasChildren() {
		return
	}
	sb.SetStretchMaxWidth()
	sb.SetMinPrefHeight(units.NewValue(1.2, units.Em))
	sb.SetProp("overflow", "hidden") // no scrollbars!
	sb.SetProp("margin", 0)
	sb.SetProp("padding", 0)
	lbl := sb.AddNewChild(gi.KiT_Label, "sb-lbl").(*gi.Label)
	lbl.SetStretchMaxWidth()
	lbl.SetMinPrefHeight(units.NewValue(1, units.Em))
	lbl.SetProp("vertical-align", gist.AlignTop)
	lbl.SetProp("margin", 0)
	lbl.SetProp("padding", 0)
	lbl.SetProp("tab-size", 4)
}

// ConfigToolbar adds a GideView toolbar.
func (ge *GideView) ConfigToolbar() {
	tb := ge.ToolBar()
	if tb.HasChildren() {
		return
	}
	tb.SetStretchMaxWidth()
	giv.ToolBarView(ge, ge.Viewport, tb)
	tb.AddSeparator("sepmod")
	sm := tb.AddNewChild(gi.KiT_CheckBox, "go-mod").(*gi.CheckBox)
	sm.SetChecked(ge.Prefs.GoMod)
	sm.SetText("Go Mod")
	sm.Tooltip = "Toggles the use of go modules -- saved with project -- if off, uses old school GOPATH mode"
	sm.ButtonSig.Connect(ge.This(), func(recv, send ki.Ki, sig int64, data any) {
		if sig == int64(gi.ButtonToggled) {
			cb := send.(*gi.CheckBox)
			ge.Prefs.GoMod = cb.IsChecked()
			gide.SetGoMod(ge.Prefs.GoMod)
		}
	})
}

var fnFolderProps = ki.Props{
	"icon":     "folder-open",
	"icon-off": "folder",
}

// ConfigSplitView configures the SplitView.
func (ge *GideView) ConfigSplitView() {
	split := ge.SplitView()
	split.Dim = mat32.X
	if split.HasChildren() {
		return
	}
	updt := split.UpdateStart()
	ftfr := gi.AddNewFrame(split, "filetree", gi.LayoutVert)
	ftfr.SetReRenderAnchor()
	ft := ftfr.AddNewChild(gide.KiT_FileTreeView, "filetree").(*gide.FileTreeView)
	ft.SetFlag(int(giv.TreeViewFlagUpdtRoot)) // filetree needs this
	ft.OpenDepth = 4
	ge.FilesView = ft
	ft.SetRootNode(&ge.Files)
	ft.TreeViewSig.Connect(ge.This(), func(recv, send ki.Ki, sig int64, data any) {
		if data == nil {
			return
		}
		tvn, _ := data.(ki.Ki).Embed(gide.KiT_FileTreeView).(*gide.FileTreeView)
		gee, _ := recv.Embed(KiT_GideView).(*GideView)
		if tvn.SrcNode != nil {
			fn := tvn.SrcNode.Embed(giv.KiT_FileNode).(*giv.FileNode)
			switch sig {
			case int64(giv.TreeViewSelected):
				gee.FileNodeSelected(fn, tvn)
			case int64(giv.TreeViewOpened):
				gee.FileNodeOpened(fn, tvn)
			case int64(giv.TreeViewClosed):
				gee.FileNodeClosed(fn, tvn)
			}
		}
	})

	for i := 0; i < NTextViews; i++ {
		txnm := fmt.Sprintf("%d", i)
		txly := gi.AddNewLayout(split, "textlay-"+txnm, gi.LayoutVert)
		txly.SetStretchMaxWidth()
		txly.SetStretchMaxHeight()
		txly.SetReRenderAnchor() // anchor here: SplitView will only anchor Frame, but we just have layout

		// need to sandbox the button in its own layer to isolate FullReRender issues
		txbly := gi.AddNewLayout(txly, "butlay-"+txnm, gi.LayoutVert)
		txbly.SetProp("spacing", units.NewEm(0))
		txbly.SetStretchMaxWidth()
		txbly.SetReRenderAnchor() // anchor here!

		txbut := gi.AddNewMenuButton(txbly, "textbut-"+txnm)
		txbut.SetStretchMaxWidth()
		txbut.SetText("textview: " + txnm)
		txbut.MakeMenuFunc = ge.TextViewButtonMenu
		txbut.ButtonSig.Connect(ge.This(), func(recv, send ki.Ki, sig int64, data any) {
			if sig == int64(gi.ButtonClicked) {
				gee, _ := recv.Embed(KiT_GideView).(*GideView)
				idx := 0
				nm := send.Name()
				nln := len(nm)
				if nm[nln-1] == '1' {
					idx = 1
				}
				gee.SetActiveTextViewIdx(idx)
			}
		})

		txily := gi.AddNewLayout(txly, "textilay-"+txnm, gi.LayoutVert)
		txily.SetStretchMaxWidth()
		txily.SetStretchMaxHeight()
		txily.SetMinPrefWidth(units.NewCh(80))
		txily.SetMinPrefHeight(units.NewEm(40))

		ted := gide.AddNewTextView(txily, "textview-"+txnm)
		ted.TextViewSig.Connect(ge.This(), func(recv, send ki.Ki, sig int64, data any) {
			gee, _ := recv.Embed(KiT_GideView).(*GideView)
			tee := send.Embed(gide.KiT_TextView).(*gide.TextView)
			gee.TextViewSig(tee, giv.TextViewSignals(sig))
		})
	}

	ge.ConfigTextViews()
	ge.UpdateTextButtons()

	mtab := gi.AddNewTabView(split, "tabs")
	mtab.TabViewSig.Connect(ge.This(), func(recv, send ki.Ki, sig int64, data any) {
		gee, _ := recv.Embed(KiT_GideView).(*GideView)
		tvsig := gi.TabViewSignals(sig)
		switch tvsig {
		case gi.TabDeleted:
			gee.TabDeleted(data.(string))
			if data == "Find" {
				ge.ActiveTextView().ClearHighlights()
			}
		}
	})

	split.SetSplits(ge.Prefs.Splits...)
	split.UpdateEnd(updt)
}

// ConfigTextViews configures text views according to current settings
func (ge *GideView) ConfigTextViews() {
	for i := 0; i < NTextViews; i++ {
		tv := ge.TextViewByIndex(i)
		if ge.Prefs.Editor.WordWrap {
			tv.SetProp("white-space", gist.WhiteSpacePreWrap)
		} else {
			tv.SetProp("white-space", gist.WhiteSpacePre)
		}
		tv.SetProp("tab-size", ge.Prefs.Editor.TabSize)
		tv.SetProp("font-family", gi.Prefs.MonoFont)
	}
}

// UpdateTextButtons updates textview menu buttons
// is called by SetStatus and is generally under cover of TopUpdateStart / End
// doesn't do anything unless a change is required -- safe to call frequently.
func (ge *GideView) UpdateTextButtons() {
	ati := ge.ActiveTextViewIdx
	for i := 0; i < NTextViews; i++ {
		tv := ge.TextViewByIndex(i)
		mb := ge.TextViewButtonByIndex(i)
		txnm := "<no file>"
		if tv.Buf != nil {
			txnm = giv.DirAndFile(string(tv.Buf.Filename))
			if tv.Buf.IsChanged() {
				txnm += " <b>*</b>"
			}
		}
		sel := ati == i
		if mb.Text != txnm || sel != mb.IsSelected() {
			updt := mb.UpdateStart()
			mb.SetText(txnm)
			mb.SetSelectedState(sel)
			mb.UpdateEnd(updt)
		}
	}
}

func (ge *GideView) TextViewButtonMenu(obj ki.Ki, m *gi.Menu) {
	idx := 0
	nm := obj.Name()
	nln := len(nm)
	if nm[nln-1] == '1' {
		idx = 1
	}
	opn := ge.OpenNodes.Strings()
	*m = gi.Menu{}

	m.AddAction(gi.ActOpts{Label: "Open File..."}, ge.This(),
		func(recv, send ki.Ki, sig int64, data any) {
			giv.CallMethod(ge, "ViewFile", ge.Viewport)
		})

	m.AddSeparator("file-sep")

	tv := ge.TextViewByIndex(idx)
	for i, n := range opn {
		m.AddAction(gi.ActOpts{Label: n, Data: i}, ge.This(),
			func(recv, send ki.Ki, sig int64, data any) {
				ac := send.(*gi.Action)
				gidx := ac.Data.(int)
				nb := ge.OpenNodes[gidx]
				ge.ViewFileNode(tv, idx, nb)
			})
	}
}

// FileNodeSelected is called whenever tree browser has file node selected
func (ge *GideView) FileNodeSelected(fn *giv.FileNode, tvn *gide.FileTreeView) {
	// if fn.IsDir() {
	// } else {
	// }
}

// CatNoEdit are the files to NOT edit from categories: Doc, Data
var CatNoEdit = map[filecat.Supported]bool{
	filecat.Rtf:          true,
	filecat.MSWord:       true,
	filecat.OpenText:     true,
	filecat.OpenPres:     true,
	filecat.MSPowerpoint: true,
	filecat.EBook:        true,
	filecat.EPub:         true,
}

// FileNodeOpened is called whenever file node is double-clicked in file tree
func (ge *GideView) FileNodeOpened(fn *giv.FileNode, tvn *gide.FileTreeView) {
	// todo: could add all these options in LangOpts
	switch fn.Info.Cat {
	case filecat.Folder:
		// if !fn.IsOpen() {
		tvn.SetOpen()
		fn.OpenDir()
		// }
		return
	case filecat.Exe:
		// this uses exe path for cd to this path!
		ge.SetArgVarVals()
		ge.ArgVals["{PromptString1}"] = string(fn.FPath)
		gide.CmdNoUserPrompt = true // don't re-prompt!
		cmd, _, ok := gide.AvailCmds.CmdByName(gide.CmdName("Build: Run Prompt"), true)
		if ok {
			ge.ArgVals.Set(string(fn.FPath), &ge.Prefs, nil)
			cbuf, _, _ := ge.RecycleCmdTab(cmd.Name, true, true)
			cmd.Run(ge, cbuf)
		}
		return
	case filecat.Font, filecat.Video, filecat.Model, filecat.Audio, filecat.Sheet, filecat.Bin,
		filecat.Archive, filecat.Image:
		ge.ExecCmdNameFileNode(fn, gide.CmdName("File: Open"), true, true) // sel, clear
		return
	}

	edit := true
	switch fn.Info.Cat {
	case filecat.Code:
		edit = true
	case filecat.Text:
		edit = true
	default:
		if _, noed := CatNoEdit[fn.Info.Sup]; noed {
			edit = false
		}
	}
	if !edit {
		ge.ExecCmdNameFileNode(fn, gide.CmdName("File: Open"), true, true) // sel, clear
		return
	}
	// program, document, data
	if int(fn.Info.Size) > gi.Prefs.Params.BigFileSize {
		gi.ChoiceDialog(ge.Viewport, gi.DlgOpts{Title: "File is relatively large",
			Prompt: fmt.Sprintf("The file: %v is relatively large at: %v -- really open for editing?", fn.Nm, fn.Info.Size)},
			[]string{"Open", "Cancel"},
			ge.This(), func(recv, send ki.Ki, sig int64, data any) {
				switch sig {
				case 0:
					ge.NextViewFileNode(fn)
				case 1:
					// do nothing
				}
			})
	} else {
		ge.NextViewFileNode(fn)
	}

}

// FileNodeClosed is called whenever file tree browser node is closed
func (ge *GideView) FileNodeClosed(fn *giv.FileNode, tvn *gide.FileTreeView) {
	if fn.IsDir() {
		if fn.IsOpen() {
			// fmt.Printf("FileNodeClosed, was open: %s\n", fn.FPath)
			fn.CloseDir()
		}
	}
}

func (ge *GideView) GideViewKeys(kt *key.ChordEvent) {
	gide.SetGoMod(ge.Prefs.GoMod)
	var kf gide.KeyFuns
	kc := kt.Chord()
	if gi.KeyEventTrace {
		fmt.Printf("GideView KeyInput: %v\n", ge.Path())
	}
	gkf := gi.KeyFun(kc)
	if ge.KeySeq1 != "" {
		kf = gide.KeyFun(ge.KeySeq1, kc)
		seqstr := string(ge.KeySeq1) + " " + string(kc)
		if kf == gide.KeyFunNil || kc == "Escape" {
			if gi.KeyEventTrace {
				fmt.Printf("gide.KeyFun sequence: %v aborted\n", seqstr)
			}
			ge.SetStatus(seqstr + " -- aborted")
			kt.SetProcessed() // abort key sequence, don't send esc to anyone else
			ge.KeySeq1 = ""
			return
		}
		ge.SetStatus(seqstr)
		ge.KeySeq1 = ""
		gkf = gi.KeyFunNil // override!
	} else {
		kf = gide.KeyFun(kc, "")
		if kf == gide.KeyFunNeeds2 {
			kt.SetProcessed()
			tv := ge.ActiveTextView()
			if tv != nil {
				tv.CancelComplete()
			}
			ge.KeySeq1 = kt.Chord()
			ge.SetStatus(string(ge.KeySeq1))
			if gi.KeyEventTrace {
				fmt.Printf("gide.KeyFun sequence needs 2 after: %v\n", ge.KeySeq1)
			}
			return
		} else if kf != gide.KeyFunNil {
			if gi.KeyEventTrace {
				fmt.Printf("gide.KeyFun got in one: %v = %v\n", ge.KeySeq1, kf)
			}
			gkf = gi.KeyFunNil // override!
		}
	}

	switch gkf {
	case gi.KeyFunFind:
		kt.SetProcessed()
		tv := ge.ActiveTextView()
		if tv != nil && tv.HasSelection() {
			ge.Prefs.Find.Find = string(tv.Selection().ToBytes())
		}
		giv.CallMethod(ge, "Find", ge.Viewport)
	}
	if kt.IsProcessed() {
		return
	}
	switch kf {
	case gide.KeyFunNextPanel:
		kt.SetProcessed()
		ge.FocusNextPanel()
	case gide.KeyFunPrevPanel:
		kt.SetProcessed()
		ge.FocusPrevPanel()
	case gide.KeyFunFileOpen:
		kt.SetProcessed()
		giv.CallMethod(ge, "ViewFile", ge.Viewport)
	case gide.KeyFunBufSelect:
		kt.SetProcessed()
		ge.SelectOpenNode()
	case gide.KeyFunBufClone:
		kt.SetProcessed()
		ge.CloneActiveView()
	case gide.KeyFunBufSave:
		kt.SetProcessed()
		ge.SaveActiveView()
	case gide.KeyFunBufSaveAs:
		kt.SetProcessed()
		giv.CallMethod(ge, "SaveActiveViewAs", ge.Viewport)
	case gide.KeyFunBufClose:
		kt.SetProcessed()
		ge.CloseActiveView()
	case gide.KeyFunExecCmd:
		kt.SetProcessed()
		giv.CallMethod(ge, "ExecCmd", ge.Viewport)
	case gide.KeyFunRectCut:
		kt.SetProcessed()
		ge.CutRect()
	case gide.KeyFunRectCopy:
		kt.SetProcessed()
		ge.CopyRect()
	case gide.KeyFunRectPaste:
		kt.SetProcessed()
		ge.PasteRect()
	case gide.KeyFunRegCopy:
		kt.SetProcessed()
		giv.CallMethod(ge, "RegisterCopy", ge.Viewport)
	case gide.KeyFunRegPaste:
		kt.SetProcessed()
		giv.CallMethod(ge, "RegisterPaste", ge.Viewport)
	case gide.KeyFunCommentOut:
		kt.SetProcessed()
		ge.CommentOut()
	case gide.KeyFunIndent:
		kt.SetProcessed()
		ge.Indent()
	case gide.KeyFunJump:
		kt.SetProcessed()
		tv := ge.ActiveTextView()
		if tv != nil {
			tv.JumpToLinePrompt()
		}
		ge.Indent()
	case gide.KeyFunSetSplit:
		kt.SetProcessed()
		giv.CallMethod(ge, "SplitsSetView", ge.Viewport)
	case gide.KeyFunBuildProj:
		kt.SetProcessed()
		ge.Build()
	case gide.KeyFunRunProj:
		kt.SetProcessed()
		ge.Run()
	}
}

func (ge *GideView) KeyChordEvent() {
	// need hipri to prevent 2-seq guys from being captured by others
	ge.ConnectEvent(oswin.KeyChordEvent, gi.HiPri, func(recv, send ki.Ki, sig int64, d any) {
		gee := recv.Embed(KiT_GideView).(*GideView)
		kt := d.(*key.ChordEvent)
		gee.GideViewKeys(kt)
	})
}

func (ge *GideView) MouseEvent() {
	ge.ConnectEvent(oswin.MouseEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d any) {
		gee := recv.Embed(KiT_GideView).(*GideView)
		gide.SetGoMod(gee.Prefs.GoMod)
	})
}

func (ge *GideView) OSFileEvent() {
	ge.ConnectEvent(oswin.OSOpenFilesEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d any) {
		gee := recv.Embed(KiT_GideView).(*GideView)
		ofe := d.(*osevent.OpenFilesEvent)
		for _, fn := range ofe.Files {
			gee.OpenFile(fn)
		}
	})
}

func (ge *GideView) Render2D() {
	if len(ge.Kids) > 0 {
		ge.ToolBar().UpdateActions()
		if win := ge.ParentWindow(); win != nil {
			sv := ge.SplitView()
			win.EventMgr.SetStartFocus(sv.This())
			if !win.IsResizing() {
				win.MainMenuUpdateActives()
			}
		}
	}
	ge.Frame.Render2D()
}

func (ge *GideView) ConnectEvents2D() {
	if ge.HasAnyScroll() {
		ge.LayoutScrollEvents()
	}
	ge.KeyChordEvent()
	ge.MouseEvent()
	ge.OSFileEvent()
}

// GideViewInactiveEmptyFunc is an ActionUpdateFunc that inactivates action if project is empty
var GideViewInactiveEmptyFunc = giv.ActionUpdateFunc(func(gei any, act *gi.Action) {
	ge := gei.(ki.Ki).Embed(KiT_GideView).(*GideView)
	if !ge.IsConfiged() {
		return
	}
	act.SetInactiveState(ge.IsEmpty())
})

// GideViewInactiveTextViewFunc is an ActionUpdateFunc that inactivates action there is no active text view
var GideViewInactiveTextViewFunc = giv.ActionUpdateFunc(func(gei any, act *gi.Action) {
	ge := gei.(ki.Ki).Embed(KiT_GideView).(*GideView)
	if !ge.IsConfiged() {
		return
	}
	act.SetInactiveState(ge.ActiveTextView().Buf == nil)
})

// GideViewInactiveTextSelectionFunc is an ActionUpdateFunc that inactivates action there is no active text view
var GideViewInactiveTextSelectionFunc = giv.ActionUpdateFunc(func(gei any, act *gi.Action) {
	ge := gei.(ki.Ki).Embed(KiT_GideView).(*GideView)
	if !ge.IsConfiged() {
		return
	}
	if ge.ActiveTextView() != nil && ge.ActiveTextView().Buf != nil {
		act.SetActiveState(ge.ActiveTextView().HasSelection())
	} else {
		act.SetActiveState(false)
	}
})

var GideViewProps = ki.Props{
	"EnumType:Flag":    gi.KiT_NodeFlags,
	"background-color": &gi.Prefs.Colors.Background,
	"color":            &gi.Prefs.Colors.Font,
	"max-width":        -1,
	"max-height":       -1,
	"#title": ki.Props{
		"max-width":        -1,
		"horizontal-align": gist.AlignCenter,
		"vertical-align":   gist.AlignTop,
	},
	"MethViewNoUpdateAfter": true, // no update after is default for everything
	"ToolBar": ki.PropSlice{
		{"UpdateFiles", ki.Props{
			"shortcut": "Command+U",
			"desc":     "update file browser list of files",
			"icon":     "update",
		}},
		{"NextViewFile", ki.Props{
			"label": "Open...",
			"icon":  "file-open",
			"desc":  "open a file in current active text view",
			"shortcut-func": giv.ShortcutFunc(func(gei any, act *gi.Action) key.Chord {
				return key.Chord(gide.ChordForFun(gide.KeyFunFileOpen).String())
			}),
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"default-field": "ActiveFilename",
				}},
			},
		}},
		{"SaveActiveView", ki.Props{
			"label": "Save",
			"desc":  "save active text view file to its current filename",
			"icon":  "file-save",
			"shortcut-func": giv.ShortcutFunc(func(gei any, act *gi.Action) key.Chord {
				return key.Chord(gide.ChordForFun(gide.KeyFunBufSave).String())
			}),
		}},
		{"SaveActiveViewAs", ki.Props{
			"label": "Save As...",
			"icon":  "file-save",
			"desc":  "save active text view file to a new filename",
			"shortcut-func": giv.ShortcutFunc(func(gei any, act *gi.Action) key.Chord {
				return key.Chord(gide.ChordForFun(gide.KeyFunBufSaveAs).String())
			}),
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"default-field": "ActiveFilename",
				}},
			},
		}},
		{"SaveAll", ki.Props{
			"icon": "file-save",
			"desc": "save all open files (if modified) and the current project prefs (if .gide file exists, from prior Save Proj As..)",
		}},
		{"ViewOpenNodeName", ki.Props{
			"icon":         "file-text",
			"label":        "Edit",
			"desc":         "select an open file to view in active text view",
			"submenu-func": giv.SubMenuFunc(GideViewOpenNodes),
			"shortcut-func": giv.ShortcutFunc(func(gei any, act *gi.Action) key.Chord {
				return key.Chord(gide.ChordForFun(gide.KeyFunBufSelect).String())
			}),
			"Args": ki.PropSlice{
				{"Node Name", ki.Props{}},
			},
		}},
		{"sep-find", ki.BlankProp{}},
		{"CursorToHistPrev", ki.Props{
			"icon":     "wedge-left",
			"shortcut": gi.KeyFunHistPrev,
			"label":    "",
			"desc":     "move cursor to previous location in active text view",
		}},
		{"CursorToHistNext", ki.Props{
			"icon":     "wedge-right",
			"shortcut": gi.KeyFunHistNext,
			"label":    "",
			"desc":     "move cursor to next location in active text view",
		}},
		{"Find", ki.Props{
			"label":    "Find...",
			"icon":     "search",
			"desc":     "Find / replace in all open folders in file browser",
			"shortcut": gi.KeyFunFind,
			"Args": ki.PropSlice{
				{"Search For", ki.Props{
					"default-field": "Prefs.Find.Find",
					"history-field": "Prefs.Find.FindHist",
					"width":         80,
				}},
				{"Replace With", ki.Props{
					"desc":          "Optional replace string -- replace will be controlled interactively in Find panel, including replace all",
					"default-field": "Prefs.Find.Replace",
					"history-field": "Prefs.Find.ReplHist",
					"width":         80,
				}},
				{"Ignore Case", ki.Props{
					"default-field": "Prefs.Find.IgnoreCase",
				}},
				{"Regexp", ki.Props{
					"default-field": "Prefs.Find.Regexp",
				}},
				{"Location", ki.Props{
					"desc":          "location to find in",
					"default-field": "Prefs.Find.Loc",
				}},
				{"Languages", ki.Props{
					"desc":          "restrict find to files associated with these languages -- leave empty for all files",
					"default-field": "Prefs.Find.Langs",
				}},
			},
		}},
		{"Symbols", ki.Props{
			"icon": "structure",
		}},
		{"Spell", ki.Props{
			"label": "Spelling",
			"icon":  "spelling",
		}},
		{"sep-file", ki.BlankProp{}},
		{"Build", ki.Props{
			"icon": "terminal",
			"desc": "build the project -- command(s) specified in Project Prefs",
			"shortcut-func": giv.ShortcutFunc(func(gei any, act *gi.Action) key.Chord {
				return key.Chord(gide.ChordForFun(gide.KeyFunBuildProj).String())
			}),
		}},
		{"Run", ki.Props{
			"icon": "terminal",
			"desc": "run the project -- command(s) specified in Project Prefs",
			"shortcut-func": giv.ShortcutFunc(func(gei any, act *gi.Action) key.Chord {
				return key.Chord(gide.ChordForFun(gide.KeyFunRunProj).String())
			}),
		}},
		{"Debug", ki.Props{
			"icon": "terminal",
			"desc": "debug currently selected executable (context menu on executable, select Set Run Exec) -- if none selected, prompts to select one",
		}},
		{"DebugTest", ki.Props{
			"icon": "terminal",
			"desc": "debug test in current active view directory",
		}},
		{"sep-exe", ki.BlankProp{}},
		{"Commit", ki.Props{
			"icon": "star",
		}},
		{"ExecCmdNameActive", ki.Props{
			"icon":            "terminal",
			"label":           "Exec Cmd",
			"desc":            "execute given command on active file / directory / project",
			"subsubmenu-func": giv.SubSubMenuFunc(ExecCmds),
			"shortcut-func": giv.ShortcutFunc(func(gei any, act *gi.Action) key.Chord {
				return key.Chord(gide.ChordForFun(gide.KeyFunExecCmd).String())
			}),
			"Args": ki.PropSlice{
				{"Cmd Name", ki.Props{}},
			},
		}},
		{"sep-splt", ki.BlankProp{}},
		{"Splits", ki.PropSlice{
			{"SplitsSetView", ki.Props{
				"label":   "Set View",
				"submenu": &gide.AvailSplitNames,
				"Args": ki.PropSlice{
					{"Split Name", ki.Props{
						"default-field": "Prefs.SplitName",
					}},
				},
			}},
			{"SplitsSaveAs", ki.Props{
				"label": "Save As...",
				"desc":  "save current splitter values to a new named split configuration",
				"Args": ki.PropSlice{
					{"Name", ki.Props{
						"width": 60,
					}},
					{"Desc", ki.Props{
						"width": 60,
					}},
				},
			}},
			{"SplitsSave", ki.Props{
				"label":   "Save",
				"submenu": &gide.AvailSplitNames,
				"Args": ki.PropSlice{
					{"Split Name", ki.Props{
						"default-field": "Prefs.SplitName",
					}},
				},
			}},
			{"SplitsEdit", ki.Props{
				"label": "Edit...",
			}},
		}},
	},
	"MainMenu": ki.PropSlice{
		{"AppMenu", ki.BlankProp{}},
		{"File", ki.PropSlice{
			{"OpenRecent", ki.Props{
				"submenu": &gide.SavedPaths,
				"Args": ki.PropSlice{
					{"File Name", ki.Props{}},
				},
			}},
			{"OpenProj", ki.Props{
				"shortcut": gi.KeyFunMenuOpen,
				"label":    "Open Project...",
				"desc":     "open a gide project -- can be a .gide file or just a file or directory (projects are just directories with relevant files)",
				"Args": ki.PropSlice{
					{"File Name", ki.Props{
						"default-field": "ProjFilename",
						"ext":           ".gide",
					}},
				},
			}},
			{"OpenPath", ki.Props{
				"shortcut": gi.KeyFunMenuOpenAlt1,
				"label":    "Open Path...",
				"desc":     "open a gide project for a file or directory (projects are just directories with relevant files)",
				"Args": ki.PropSlice{
					{"Path", ki.Props{}},
				},
			}},
			{"New", ki.PropSlice{
				{"NewProj", ki.Props{
					"shortcut": gi.KeyFunMenuNew,
					"label":    "New Project...",
					"desc":     "Create a new project -- select a path for the parent folder, and a folder name for the new project -- all GideView projects are basically folders with files.  You can also specify the main language and {version control system for the project.  For other options, do <code>Proj Prefs</code> in the File menu of the new project.",
					"Args": ki.PropSlice{
						{"Parent Folder", ki.Props{
							"dirs-only":     true, // todo: support
							"default-field": "ProjRoot",
						}},
						{"Folder", ki.Props{
							"width": 60,
						}},
						{"Main Lang", ki.Props{}},
						{"Version Ctrl", ki.Props{}},
					},
				}},
				{"NewFile", ki.Props{
					"shortcut": gi.KeyFunMenuNewAlt1,
					"label":    "New File...",
					"desc":     "Create a new file in project -- to create in sub-folders, use context menu on folder in file browser",
					"Args": ki.PropSlice{
						{"File Name", ki.Props{
							"width": 60,
						}},
						{"Add To Version Control", ki.Props{}},
					},
				}},
			}},
			{"SaveProj", ki.Props{
				"shortcut": gi.KeyFunMenuSave,
				"label":    "Save Project",
				"updtfunc": GideViewInactiveEmptyFunc,
			}},
			{"SaveProjAs", ki.Props{
				"shortcut": gi.KeyFunMenuSaveAs,
				"label":    "Save Project As...",
				"desc":     "Save project to given file name -- this is the .gide file containing preferences and current settings -- also saves all open files -- once saved, further saving is automatic",
				"updtfunc": GideViewInactiveEmptyFunc,
				"Args": ki.PropSlice{
					{"File Name", ki.Props{
						"default-field": "ProjFilename",
						"ext":           ".gide",
					}},
					{"SaveAll", ki.Props{
						"value": false,
					}},
				},
			}},
			{"SaveAll", ki.Props{}},
			{"sep-af", ki.BlankProp{}},
			{"ViewFile", ki.Props{
				"label": "Open File...",
				"shortcut-func": func(gei any, act *gi.Action) key.Chord {
					return key.Chord(gide.ChordForFun(gide.KeyFunFileOpen).String())
				},
				"updtfunc": GideViewInactiveEmptyFunc,
				"Args": ki.PropSlice{
					{"File Name", ki.Props{
						"default-field": "ActiveFilename",
					}},
				},
			}},
			{"SaveActiveView", ki.Props{
				"label": "Save File",
				"shortcut-func": giv.ShortcutFunc(func(gei any, act *gi.Action) key.Chord {
					return key.Chord(gide.ChordForFun(gide.KeyFunBufSave).String())
				}),
				"updtfunc": GideViewInactiveEmptyFunc,
			}},
			{"SaveActiveViewAs", ki.Props{
				"label":    "Save File As...",
				"updtfunc": GideViewInactiveEmptyFunc,
				"desc":     "save active text view file to a new filename",
				"shortcut-func": giv.ShortcutFunc(func(gei any, act *gi.Action) key.Chord {
					return key.Chord(gide.ChordForFun(gide.KeyFunBufSaveAs).String())
				}),
				"Args": ki.PropSlice{
					{"File Name", ki.Props{
						"default-field": "ActiveFilename",
					}},
				},
			}},
			{"RevertActiveView", ki.Props{
				"desc":     "Revert active file to last saved version: this will lose all active changes -- are you sure?",
				"confirm":  true,
				"label":    "Revert File...",
				"updtfunc": GideViewInactiveEmptyFunc,
			}},
			{"CloseActiveView", ki.Props{
				"label":    "Close File",
				"updtfunc": GideViewInactiveEmptyFunc,
				"shortcut-func": giv.ShortcutFunc(func(gei any, act *gi.Action) key.Chord {
					return key.Chord(gide.ChordForFun(gide.KeyFunBufClose).String())
				}),
			}},
			{"sep-prefs", ki.BlankProp{}},
			{"EditProjPrefs", ki.Props{
				"label":    "Project Prefs...",
				"updtfunc": GideViewInactiveEmptyFunc,
			}},
			{"sep-close", ki.BlankProp{}},
			{"Close Window", ki.BlankProp{}},
		}},
		{"Edit", ki.PropSlice{
			{"Copy", ki.Props{
				"keyfun":   gi.KeyFunCopy,
				"updtfunc": GideViewInactiveTextSelectionFunc,
			}},
			{"Cut", ki.Props{
				"keyfun":   gi.KeyFunCut,
				"updtfunc": GideViewInactiveTextSelectionFunc,
			}},
			{"Paste", ki.Props{
				"keyfun": gi.KeyFunPaste,
			}},
			{"Paste History...", ki.Props{
				"keyfun": gi.KeyFunPasteHist,
			}},
			{"Registers", ki.PropSlice{
				{"RegisterCopy", ki.Props{
					"label": "Copy...",
					"desc":  "save currently-selected text to a named register, which can be pasted later -- persistent across sessions as well",
					"shortcut-func": giv.ShortcutFunc(func(gei any, act *gi.Action) key.Chord {
						return key.Chord(gide.ChordForFun(gide.KeyFunRegCopy).String())
					}),
					"updtfunc": GideViewInactiveEmptyFunc,
					"Args": ki.PropSlice{
						{"Register Name", ki.Props{
							"default": "", // override memory of last
						}},
					},
				}},
				{"RegisterPaste", ki.Props{
					"label": "Paste...",
					"desc":  "paste text from named register",
					"shortcut-func": giv.ShortcutFunc(func(gei any, act *gi.Action) key.Chord {
						return key.Chord(gide.ChordForFun(gide.KeyFunRegPaste).String())
					}),
					"updtfunc": GideViewInactiveEmptyFunc,
					"Args": ki.PropSlice{
						{"Register Name", ki.Props{
							"default-field": "Prefs.Register",
						}},
					},
				}},
			}},
			{"sep-undo", ki.BlankProp{}},
			{"Undo", ki.Props{
				"keyfun": gi.KeyFunUndo,
			}},
			{"Redo", ki.Props{
				"keyfun": gi.KeyFunRedo,
			}},
			{"sep-find", ki.BlankProp{}},
			{"Find", ki.Props{
				"label":    "Find...",
				"shortcut": gi.KeyFunFind,
				"desc":     "Find / replace in all open folders in file browser",
				"updtfunc": GideViewInactiveEmptyFunc,
				"Args": ki.PropSlice{
					{"Search For", ki.Props{
						"default-field": "Prefs.Find.Find",
						"history-field": "Prefs.Find.FindHist",
						"width":         80,
					}},
					{"Replace With", ki.Props{
						"desc":          "Optional replace string -- replace will be controlled interactively in Find panel, including replace all",
						"default-field": "Prefs.Find.Replace",
						"history-field": "Prefs.Find.ReplHist",
						"width":         80,
					}},
					{"Ignore Case", ki.Props{
						"default-field": "Prefs.Find.IgnoreCase",
					}},
					{"Regexp", ki.Props{
						"default-field": "Prefs.Find.Regexp",
					}},
					{"Location", ki.Props{
						"desc":          "location to find in",
						"default-field": "Prefs.Find.Loc",
					}},
					{"Languages", ki.Props{
						"desc":          "restrict find to files associated with these languages -- leave empty for all files",
						"default-field": "Prefs.Find.Langs",
					}},
				},
			}},
			{"ReplaceInActive", ki.Props{
				"label":    "Replace In Active...",
				"shortcut": gi.KeyFunReplace,
				"desc":     "query-replace in current active text view only (use Find for multi-file)",
				"updtfunc": GideViewInactiveEmptyFunc,
			}},
			{"Spell", ki.Props{
				"label":    "Spelling...",
				"updtfunc": GideViewInactiveEmptyFunc,
			}},
			{"ShowCompletions", ki.Props{
				"keyfun":   gi.KeyFunComplete,
				"updtfunc": GideViewInactiveEmptyFunc,
			}},
			{"LookupSymbol", ki.Props{
				"keyfun":   gi.KeyFunLookup,
				"updtfunc": GideViewInactiveEmptyFunc,
			}},
			{"sep-adv", ki.BlankProp{}},
			{"CommentOut", ki.Props{
				"shortcut-func": giv.ShortcutFunc(func(gei any, act *gi.Action) key.Chord {
					return key.Chord(gide.ChordForFun(gide.KeyFunCommentOut).String())
				}),
				"updtfunc": GideViewInactiveEmptyFunc,
			}},
			{"Indent", ki.Props{
				"shortcut-func": giv.ShortcutFunc(func(gei any, act *gi.Action) key.Chord {
					return key.Chord(gide.ChordForFun(gide.KeyFunIndent).String())
				}),
				"updtfunc": GideViewInactiveEmptyFunc,
			}},
			{"sep-xform", ki.BlankProp{}},
			{"ReCase", ki.Props{
				"desc":     "replace currently-selected text with text of given case",
				"updtfunc": GideViewInactiveEmptyFunc,
				"Args": ki.PropSlice{
					{"To Case", ki.Props{}},
				},
			}},
			{"JoinParaLines", ki.Props{
				"desc":     "merges sequences of lines with hard returns forming paragraphs, separated by blank lines, into a single line per paragraph, for given selected region (full text if no selection)",
				"confirm":  true,
				"updtfunc": GideViewInactiveEmptyFunc,
			}},
			{"TabsToSpaces", ki.Props{
				"desc":     "converts tabs to spaces for given selected region (full text if no selection)",
				"confirm":  true,
				"updtfunc": GideViewInactiveEmptyFunc,
			}},
			{"SpacesToTabs", ki.Props{
				"desc":     "converts spaces to tabs for given selected region (full text if no selection)",
				"confirm":  true,
				"updtfunc": GideViewInactiveEmptyFunc,
			}},
		}},
		{"View", ki.PropSlice{
			{"Panels", ki.PropSlice{
				{"FocusNextPanel", ki.Props{
					"label": "Focus Next",
					"shortcut-func": giv.ShortcutFunc(func(gei any, act *gi.Action) key.Chord {
						return key.Chord(gide.ChordForFun(gide.KeyFunNextPanel).String())
					}),
					"updtfunc": GideViewInactiveEmptyFunc,
				}},
				{"FocusPrevPanel", ki.Props{
					"label": "Focus Prev",
					"shortcut-func": giv.ShortcutFunc(func(gei any, act *gi.Action) key.Chord {
						return key.Chord(gide.ChordForFun(gide.KeyFunPrevPanel).String())
					}),
					"updtfunc": GideViewInactiveEmptyFunc,
				}},
				{"CloneActiveView", ki.Props{
					"label": "Clone Active",
					"shortcut-func": giv.ShortcutFunc(func(gei any, act *gi.Action) key.Chord {
						return key.Chord(gide.ChordForFun(gide.KeyFunBufClone).String())
					}),
					"updtfunc": GideViewInactiveEmptyFunc,
				}},
			}},
			{"Splits", ki.PropSlice{
				{"SplitsSetView", ki.Props{
					"label":    "Set View",
					"submenu":  &gide.AvailSplitNames,
					"updtfunc": GideViewInactiveEmptyFunc,
					"Args": ki.PropSlice{
						{"Split Name", ki.Props{}},
					},
				}},
				{"SplitsSaveAs", ki.Props{
					"label":    "Save As...",
					"desc":     "save current splitter values to a new named split configuration",
					"updtfunc": GideViewInactiveEmptyFunc,
					"Args": ki.PropSlice{
						{"Name", ki.Props{
							"width": 60,
						}},
						{"Desc", ki.Props{
							"width": 60,
						}},
					},
				}},
				{"SplitsSave", ki.Props{
					"label":    "Save",
					"submenu":  &gide.AvailSplitNames,
					"updtfunc": GideViewInactiveEmptyFunc,
					"Args": ki.PropSlice{
						{"Split Name", ki.Props{}},
					},
				}},
				{"SplitsEdit", ki.Props{
					"updtfunc": GideViewInactiveEmptyFunc,
					"label":    "Edit...",
				}},
			}},
			{"OpenConsoleTab", ki.Props{
				"updtfunc": GideViewInactiveEmptyFunc,
			}},
		}},
		{"Navigate", ki.PropSlice{
			{"Cursor", ki.PropSlice{
				{"Back", ki.Props{
					"keyfun": gi.KeyFunHistPrev,
				}},
				{"Forward", ki.Props{
					"keyfun": gi.KeyFunHistNext,
				}},
				{"Jump To Line", ki.Props{
					"keyfun": gi.KeyFunJump,
				}},
			}},
		}},
		{"Command", ki.PropSlice{
			{"Build", ki.Props{
				"updtfunc": GideViewInactiveEmptyFunc,
				"shortcut-func": giv.ShortcutFunc(func(gei any, act *gi.Action) key.Chord {
					return key.Chord(gide.ChordForFun(gide.KeyFunBuildProj).String())
				}),
			}},
			{"Run", ki.Props{
				"updtfunc": GideViewInactiveEmptyFunc,
				"shortcut-func": giv.ShortcutFunc(func(gei any, act *gi.Action) key.Chord {
					return key.Chord(gide.ChordForFun(gide.KeyFunRunProj).String())
				}),
			}},
			{"Debug", ki.Props{}},
			{"DebugTest", ki.Props{}},
			{"DebugAttach", ki.Props{
				"desc": "attach to an already running process: enter the process PID",
				"Args": ki.PropSlice{
					{"Process PID", ki.Props{}},
				},
			}},
			{"ChooseRunExec", ki.Props{
				"desc": "choose the executable to run for this project using the Run button",
				"Args": ki.PropSlice{
					{"RunExec", ki.Props{
						"default-field": "Prefs.RunExec",
					}},
				},
			}},
			{"sep-run", ki.BlankProp{}},
			{"Commit", ki.Props{
				"updtfunc": GideViewInactiveEmptyFunc,
			}},
			{"VCSLog", ki.Props{
				"label":    "VCS Log View",
				"desc":     "shows the VCS log of commits to repository associated with active file, optionally with a since date qualifier: If since is non-empty, it should be a date-like expression that the VCS will understand, such as 1/1/2020, yesterday, last year, etc (SVN only supports a max number of entries).",
				"updtfunc": GideViewInactiveEmptyFunc,
				"Args": ki.PropSlice{
					{"Since Date", ki.Props{}},
				},
			}},
			{"VCSUpdateAll", ki.Props{
				"label":    "VCS Update All",
				"updtfunc": GideViewInactiveEmptyFunc,
			}},
			{"sep-cmd", ki.BlankProp{}},
			{"ExecCmdNameActive", ki.Props{
				"label":           "Exec Cmd",
				"subsubmenu-func": giv.SubSubMenuFunc(ExecCmds),
				"updtfunc":        GideViewInactiveEmptyFunc,
				"Args": ki.PropSlice{
					{"Cmd Name", ki.Props{}},
				},
			}},
			{"DiffFiles", ki.Props{
				"updtfunc": GideViewInactiveEmptyFunc,
				"Args": ki.PropSlice{
					{"File Name 1", ki.Props{}},
					{"File Name 2", ki.Props{}},
				},
			}},
			{"sep-cmd", ki.BlankProp{}},
			{"CountWords", ki.Props{
				"updtfunc":    GideViewInactiveEmptyFunc,
				"show-return": true,
			}},
			{"CountWordsRegion", ki.Props{
				"updtfunc":    GideViewInactiveEmptyFunc,
				"show-return": true,
			}},
		}},
		{"Window", "Windows"},
		{"Help", ki.PropSlice{
			{"HelpWiki", ki.Props{}},
		}},
	},
	"CallMethods": ki.PropSlice{
		{"NextViewFile", ki.Props{
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"default-field": "ActiveFilename",
				}},
			},
		}},
		{"SplitsSetView", ki.Props{
			"Args": ki.PropSlice{
				{"Split Name", ki.Props{}},
			},
		}},
		{"ExecCmd", ki.Props{}},
		{"ChooseRunExec", ki.Props{
			"Args": ki.PropSlice{
				{"Exec File Name", ki.Props{}},
			},
		}},
	},
}

//////////////////////////////////////////////////////////////////////////////////////
//   Project window

// NewGideProjPath creates a new GideView window with a new GideView project for given
// path, returning the window and the path
func NewGideProjPath(path string) (*gi.Window, *GideView) {
	root, projnm, _, _ := ProjPathParse(path)
	return NewGideWindow(path, projnm, root, true)
}

// OpenGideProj creates a new GideView window opened to given GideView project,
// returning the window and the path
func OpenGideProj(projfile string) (*gi.Window, *GideView) {
	pp := &gide.ProjPrefs{}
	if err := pp.OpenJSON(gi.FileName(projfile)); err != nil {
		gi.PromptDialog(nil, gi.DlgOpts{Title: "Project File Could Not Be Opened", Prompt: fmt.Sprintf("Project file open encountered error: %v", err.Error())}, gi.AddOk, gi.NoCancel, nil, nil)
		return nil, nil
	}
	path := string(pp.ProjRoot)
	root, projnm, _, _ := ProjPathParse(path)
	return NewGideWindow(projfile, projnm, root, false)
}

// NewGideWindow is common code for Open GideWindow from Proj or Path
func NewGideWindow(path, projnm, root string, doPath bool) (*gi.Window, *GideView) {
	winm := "gide-" + projnm
	wintitle := winm + ": " + path

	if win, found := gi.AllWindows.FindName(winm); found {
		mfr := win.SetMainFrame()
		ge := mfr.Child(0).Embed(KiT_GideView).(*GideView)
		if string(ge.ProjRoot) == root {
			win.OSWin.Raise()
			return win, ge
		}
	}

	width := 1600
	height := 1280
	sc := oswin.TheApp.Screen(0)
	if sc != nil {
		scsz := sc.Geometry.Size()
		width = int(.9 * float64(scsz.X))
		height = int(.8 * float64(scsz.Y))
	}

	win := gi.NewMainWindow(winm, wintitle, width, height)

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()
	ge := mfr.AddNewChild(KiT_GideView, "gide").(*GideView)
	ge.Viewport = vp

	if doPath {
		ge.OpenPath(gi.FileName(path))
	} else {
		ge.OpenProj(gi.FileName(path))
	}

	mmen := win.MainMenu
	giv.MainMenuView(ge, win, mmen)

	inClosePrompt := false
	win.OSWin.SetCloseReqFunc(func(w oswin.Window) {
		if !inClosePrompt {
			inClosePrompt = true
			if ge.CloseWindowReq() {
				win.Close()
			} else {
				inClosePrompt = false
			}
		}
	})

	win.OSWin.SetCloseCleanFunc(func(w oswin.Window) {
		if gi.MainWindows.Len() <= 1 {
			go oswin.TheApp.Quit() // once main window is closed, quit
		}
	})

	win.MainMenuUpdated()

	vp.UpdateEndNoSig(updt)

	win.GoStartEventLoop()

	return win, ge
}
