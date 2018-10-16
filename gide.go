// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
package gide provides the core Gide editor object.

Derived classes can extend the functionality for specific domains.

*/
package gide

import (
	"bytes"
	"fmt"
	"go/token"
	"html"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/goki/gi"
	"github.com/goki/gi/complete"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/gi/spell"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
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
	MainTabsIdx
	VisTabsIdx
)

// Gide is the core editor and tab viewer framework for the Gide system.  The
// default view has a tree browser of files on the left, editor panels in the
// middle, and a tabbed viewer on the right.
type Gide struct {
	gi.Frame
	ProjRoot          gi.FileName             `desc:"root directory for the project -- all projects must be organized within a top-level root directory, with all the files therein constituting the scope of the project -- by default it is the path for ProjFilename"`
	ProjFilename      gi.FileName             `ext:".gide" desc:"current project filename for saving / loading specific Gide configuration information in a .gide file (optional)"`
	ActiveFilename    gi.FileName             `desc:"filename of the currently-active textview"`
	ActiveLangs       LangNames               `desc:"languages for current active filename"`
	Changed           bool                    `json:"-" desc:"has the root changed?  we receive update signals from root for changes"`
	Files             giv.FileTree            `desc:"all the files in the project directory and subdirectories"`
	ActiveTextViewIdx int                     `json:"-" desc:"index of the currently-active textview -- new files will be viewed in other views if available"`
	OpenNodes         OpenNodes               `json:"-" desc:"list of open nodes, most recent first"`
	CmdBufs           map[string]*giv.TextBuf `json:"-" desc:"the command buffers for commands run in this project"`
	CmdHistory        CmdNames                `json:"-" desc:"history of commands executed in this session"`
	Prefs             ProjPrefs               `desc:"preferences for this project -- this is what is saved in a .gide project file"`
	KeySeq1           key.Chord               `desc:"first key in sequence if needs2 key pressed"`
	UpdtMu            sync.Mutex              `desc:"mutex for protecting overall updates to Gide"`
}

var KiT_Gide = kit.Types.AddType(&Gide{}, GideProps)

// UpdateFiles updates the list of files saved in project
func (ge *Gide) UpdateFiles() {
	ge.Files.OpenPath(string(ge.ProjRoot))
}

func (ge *Gide) IsEmpty() bool {
	return ge.ProjRoot == ""
}

// OpenRecent opens a recently-used file
func (ge *Gide) OpenRecent(filename gi.FileName) {
	ext := strings.ToLower(filepath.Ext(string(filename)))
	if ext == ".gide" {
		ge.OpenProj(filename)
	} else {
		ge.NewProj(filename)
	}
}

// NewProj opens a new pproject at given path, which can either be a specific
// file or a directory containing multiple files of interest -- opens in
// current Gide object if it is empty, or otherwise opens a new window.
func (ge *Gide) NewProj(path gi.FileName) {
	if !ge.IsEmpty() {
		NewGideProj(string(path))
		return
	}
	ge.Defaults()
	root, pnm, fnm, ok := ProjPathParse(string(path))
	if ok {
		os.Chdir(root)
		SavedPaths.AddPath(root, gi.Prefs.SavedPathsMax)
		SavePaths()
		ge.ProjRoot = gi.FileName(root)
		ge.SetName(pnm)
		ge.Prefs.ProjFilename = gi.FileName(filepath.Join(root, pnm+".gide"))
		ge.ProjFilename = ge.Prefs.ProjFilename
		ge.Prefs.ProjRoot = ge.ProjRoot
		ge.UpdateProj()
		ge.GuessMainLang()
		ge.LangDefaults()
		ge.GuessVersCtrl()
		win := ge.ParentWindow()
		if win != nil {
			winm := "gide-" + pnm
			win.SetName(winm)
			win.SetTitle(winm)
		}
		if fnm != "" {
			ge.NextViewFile(gi.FileName(fnm))
		}
	}
}

// SaveProj saves project file containing custom project settings, in a
// standard JSON-formatted file
func (ge *Gide) SaveProj() {
	if ge.Prefs.ProjFilename == "" {
		return
	}
	ge.SaveProjAs(ge.Prefs.ProjFilename)
}

// SaveProjIfExists saves project file containing custom project settings, in a
// standard JSON-formatted file, only if it already exists -- returns true if saved
func (ge *Gide) SaveProjIfExists() bool {
	if ge.Prefs.ProjFilename == "" {
		return false
	}
	if _, err := os.Stat(string(ge.Prefs.ProjFilename)); os.IsNotExist(err) {
		return false // does not exist
	}
	ge.SaveProjAs(ge.Prefs.ProjFilename)
	return true
}

// SaveProjAs saves project custom settings to given filename, in a standard
// JSON-formatted file
func (ge *Gide) SaveProjAs(filename gi.FileName) {
	SavedPaths.AddPath(string(filename), gi.Prefs.SavedPathsMax)
	SavePaths()
	ge.Files.UpdateNewFile(filename)
	ge.Prefs.ProjFilename = filename
	ge.ProjFilename = ge.Prefs.ProjFilename
	ge.GrabPrefs()
	ge.Prefs.SaveJSON(filename)
	ge.Changed = false
	nch := ge.NChangedFiles()
	if nch > 0 {
		gi.ChoiceDialog(ge.Viewport, gi.DlgOpts{Title: "Save Project: There are Unsaved Files",
			Prompt: fmt.Sprintf("In Project: %v There are <b>%v</b> opened files with <b>unsaved changes</b> -- do you want to save all?", ge.Nm, nch)},
			[]string{"Cancel", "Save All"},
			ge.This, func(recv, send ki.Ki, sig int64, data interface{}) {
				switch sig {
				case 0:
				// do nothing, will have returned false already
				case 1:
					ge.SaveAllOpenNodes()
				}
			})
	}
}

// OpenProj opens project and its settings from given filename, in a standard
// JSON-formatted file
func (ge *Gide) OpenProj(filename gi.FileName) {
	if !ge.IsEmpty() {
		OpenGideProj(string(filename))
		return
	}
	ge.Prefs.OpenJSON(filename)
	ge.Prefs.ProjFilename = filename // should already be set but..
	_, pnm, _, ok := ProjPathParse(string(ge.Prefs.ProjRoot))
	if ok {
		os.Chdir(string(ge.Prefs.ProjRoot))
		SavedPaths.AddPath(string(filename), gi.Prefs.SavedPathsMax)
		SavePaths()
		ge.SetName(pnm)
		ge.ApplyPrefs()
		ge.UpdateProj()
		win := ge.ParentWindow()
		if win != nil {
			winm := "gide-" + pnm
			win.SetName(winm)
			win.SetTitle(winm)
		}
	}
}

// UpdateProj does full update to current proj
func (ge *Gide) UpdateProj() {
	mods, updt := ge.StdConfig()
	if !mods {
		updt = ge.UpdateStart()
	}
	ge.UpdateFiles()
	ge.ConfigSplitView()
	ge.ConfigToolbar()
	ge.ConfigStatusBar()
	ge.SetStatus("just updated")
	ge.UpdateEnd(updt)
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
	dir, fn := filepath.Split(path)
	pathIsDir := info.IsDir()
	if pathIsDir {
		root = path
	} else {
		root = dir
		fnm = fn
	}
	_, projnm = filepath.Split(root)
	ok = true
	return
}

// GuessMainLang guesses the main language in the project -- returns true if successful
func (ge *Gide) GuessMainLang() bool {
	ecs := ge.Files.FileExtCounts()
	for _, ec := range ecs {
		ls := LangsForExt(ec.Name)
		if len(ls) == 1 {
			ge.Prefs.MainLang = LangName(ls[0].Name)
			return true
		}
	}
	return false
}

// LangDefaults applies default language settings based on MainLang
func (ge *Gide) LangDefaults() bool {
	ge.Prefs.RunCmds = CmdNames{"Run Proj"}
	ge.Prefs.BuildDir = ge.Prefs.ProjRoot
	ge.Prefs.BuildTarg = ge.Prefs.ProjRoot
	ge.Prefs.RunExec = gi.FileName(filepath.Join(string(ge.Prefs.ProjRoot), ge.Nm))
	if ge.Prefs.MainLang == "" {
		return false
	}
	got := false
	switch ge.Prefs.MainLang {
	case LangName("Go"):
		ge.Prefs.BuildCmds = CmdNames{"Build Go Proj"}
		got = true
	case LangName("LaTeX"):
		ge.Prefs.BuildCmds = CmdNames{"LaTeX PDF"}
		ge.Prefs.RunCmds = CmdNames{"Open Target File"}
		got = true
	}
	return got
}

// GuessVersCtrl guesses the version control system in use
func (ge *Gide) GuessVersCtrl() bool {
	got := false
	for vc, fn := range VersCtrlFiles {
		ftest := filepath.Join(string(ge.Prefs.ProjRoot), fn)
		if _, err := os.Stat(ftest); os.IsNotExist(err) {
			continue
		}
		ge.Prefs.VersCtrl = VersCtrlName(vc)
		got = true
	}
	return got
}

//////////////////////////////////////////////////////////////////////////////////////
//   TextViews

// ActiveTextView returns the currently-active TextView
func (ge *Gide) ActiveTextView() *giv.TextView {
	return ge.TextViewByIndex(ge.ActiveTextViewIdx)
}

// TextViewIndex finds index of given textview (0 or 1)
func (ge *Gide) TextViewIndex(av *giv.TextView) int {
	split := ge.SplitView()
	for i := 0; i < NTextViews; i++ {
		tv := split.KnownChild(TextView1Idx + i).KnownChild(0).Embed(giv.KiT_TextView).(*giv.TextView)
		if tv.This == av.This {
			return i
		}
	}
	return -1 // shouldn't happen
}

// TextViewForFileNode finds a TextView that is viewing given FileNode,
// and its index, or false if none is
func (ge *Gide) TextViewForFileNode(fn *giv.FileNode) (*giv.TextView, int, bool) {
	if fn.Buf == nil {
		return nil, -1, false
	}
	fn.Buf.Hi.Style = ge.Prefs.Editor.HiStyle
	split := ge.SplitView()
	for i := 0; i < NTextViews; i++ {
		tv := split.KnownChild(TextView1Idx + i).KnownChild(0).Embed(giv.KiT_TextView).(*giv.TextView)
		if tv != nil && tv.Buf != nil && tv.Buf.This == fn.Buf.This && ge.PanelIsOpen(i+TextView1Idx) {
			return tv, i, true
		}
	}
	return nil, -1, false
}

// OpenNodeForTextView finds the FileNode that a given TextView is
// viewing, returning its index within OpenNodes list, or false if not found
func (ge *Gide) OpenNodeForTextView(tv *giv.TextView) (*giv.FileNode, int, bool) {
	if tv.Buf == nil {
		return nil, -1, false
	}
	for i, ond := range ge.OpenNodes {
		if ond.Buf == tv.Buf {
			return ond, i, true
		}
	}
	return nil, -1, false
}

// TextViewForFile finds FileNode for file, and returns TextView and index
// that is viewing that FileNode, or false if none is
func (ge *Gide) TextViewForFile(fnm gi.FileName) (*giv.TextView, int, bool) {
	fn, ok := ge.Files.FindFile(string(fnm))
	if !ok {
		return nil, -1, false
	}
	return ge.TextViewForFileNode(fn.This.Embed(giv.KiT_FileNode).(*giv.FileNode))
}

// SetActiveFilename sets the active filename
func (ge *Gide) SetActiveFilename(fname gi.FileName) {
	ge.ActiveFilename = fname
	ge.ActiveLangs = LangNamesForFilename(string(fname))
}

// SetActiveTextView sets the given textview as the active one, and returns its index
func (ge *Gide) SetActiveTextView(av *giv.TextView) int {
	idx := ge.TextViewIndex(av)
	if idx < 0 {
		return -1
	}
	if ge.ActiveTextViewIdx == idx {
		return idx
	}
	ge.ActiveTextViewIdx = idx
	if av.Buf != nil {
		ge.SetActiveFilename(av.Buf.Filename)
	}
	ge.SetStatus("")
	return idx
}

// SetActiveTextViewIdx sets the given view index as the currently-active
// TextView -- returns that textview
func (ge *Gide) SetActiveTextViewIdx(idx int) *giv.TextView {
	if idx < 0 || idx >= NTextViews {
		log.Printf("Gide SetActiveTextViewIdx: text view index out of range: %v\n", idx)
		return nil
	}
	ge.ActiveTextViewIdx = idx
	av := ge.ActiveTextView()
	if av.Buf != nil {
		ge.SetActiveFilename(av.Buf.Filename)
	}
	ge.SetStatus("")
	av.GrabFocus()
	return av
}

// NextTextView returns the next text view available for viewing a file and
// its index -- if the active text view is empty, then it is used, otherwise
// it is the next one (if visible)
func (ge *Gide) NextTextView() (*giv.TextView, int) {
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

// SaveActiveView saves the contents of the currently-active textview
func (ge *Gide) SaveActiveView() {
	tv := ge.ActiveTextView()
	if tv.Buf != nil {
		if tv.Buf.Filename != "" {
			tv.Buf.Save()
			ge.SetStatus("File Saved")
			ge.RunPostCmdsActiveView()
		} else {
			giv.CallMethod(ge, "SaveActiveViewAs", ge.Viewport) // uses fileview
		}
	}
	ge.SaveProjIfExists()
}

// SaveActiveViewAs save with specified filename the contents of the
// currently-active textview
func (ge *Gide) SaveActiveViewAs(filename gi.FileName) {
	tv := ge.ActiveTextView()
	if tv.Buf != nil {
		ofn := tv.Buf.Filename
		obuf := tv.Buf
		tv.Buf.SaveAs(filename)
		ge.SetStatus(fmt.Sprintf("File %v Saved As: %v", ofn, filename))
		ge.RunPostCmdsActiveView()
		obuf.Open(ofn)                   // revert old buffer to old filename
		ge.Files.UpdateNewFile(filename) // new node will have this file!
		fnk, ok := ge.Files.FindFile(string(filename))
		if ok {
			fn := fnk.This.Embed(giv.KiT_FileNode).(*giv.FileNode)
			ge.ViewFileNode(tv, ge.ActiveTextViewIdx, fn)
		}
	}
	ge.SaveProjIfExists()
}

// RevertActiveView revert active view to saved version
func (ge *Gide) RevertActiveView() {
	tv := ge.ActiveTextView()
	if tv.Buf != nil {
		tv.Buf.Hi.Style = ge.Prefs.Editor.HiStyle
		tv.Buf.ReOpen()
	}
}

// CloseActiveView closes the buffer associated with active view
func (ge *Gide) CloseActiveView() bool {
	tv := ge.ActiveTextView()
	ond, idx, got := ge.OpenNodeForTextView(tv)
	if got {
		ond.Buf.Close() // todo: we can't know yet if buffer was closed if has changes!
		ge.OpenNodes.DeleteIdx(idx)
		ond.SetClosed()
		return true
	}
	return false
}

// RunPostCmdsActiveView runs any registered post commands on the active view
// -- returns true if commands were run and file was reverted after that --
// uses MainLang to disambiguate if multiple languages associated with extension.
func (ge *Gide) RunPostCmdsActiveView() bool {
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
func (ge *Gide) RunPostCmdsFileNode(fn *giv.FileNode) bool {
	ls := LangsForFilename(string(fn.Buf.Filename))
	ran := false
	if len(ls) == 1 {
		lr := ls[0]
		if len(lr.PostSaveCmds) > 0 {
			ge.ExecCmdsFileNode(fn, lr.PostSaveCmds, false, true) // no select, yes clear
			ran = true
		}
	} else if len(ls) > 1 {
		hasPosts := false
		for _, lr := range ls {
			if len(lr.PostSaveCmds) > 0 {
				hasPosts = true
				if lr.Name == string(ge.Prefs.MainLang) {
					ge.ExecCmdsFileNode(fn, lr.PostSaveCmds, false, true)
					ran = true
					break
				}
			}
		}
		if hasPosts && !ran {
			ge.SetStatus("File has multiple associated languages and none match main languages of project, cannot run any post commands")
		}
	}
	if ran {
		fn.Buf.ReOpen()
		return true
	}
	return false
}

// AutoSaveCheck checks for an autosave file and prompts user about opening it
// -- returns true if autosave file does exist for a file that currently
// unchanged (means just opened)
func (ge *Gide) AutoSaveCheck(tv *giv.TextView, vidx int, fn *giv.FileNode) bool {
	if strings.HasPrefix(fn.Nm, "#") && strings.HasSuffix(fn.Nm, "#") {
		fn.Buf.Autosave = false
		return false // we are the autosave file
	}
	fn.Buf.Autosave = true
	if tv.IsChanged() || !fn.Buf.AutoSaveCheck() {
		return false
	}
	gi.ChoiceDialog(ge.Viewport, gi.DlgOpts{Title: "Autosave file Exists",
		Prompt: fmt.Sprintf("An auto-save file for file: %v exists -- open it in the other text view (you can then do Save As to replace current file)?  If you don't open it, the next change made will overwrite it with a new one, erasing any changes.", fn.Nm)},
		[]string{"Open", "Ignore and Overwrite"},
		ge.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			switch sig {
			case 0:
				ge.NextViewFile(gi.FileName(fn.Buf.AutoSaveFilename()))
			case 1:
				// do nothing
			}
		})
	return true
}

// ViewFileNode sets the given text view to view file in given node (opens
// buffer if not already opened)
func (ge *Gide) ViewFileNode(tv *giv.TextView, vidx int, fn *giv.FileNode) {
	if fn.IsDir() {
		return
	}
	giv.FileNodeHiStyle = ge.Prefs.Editor.HiStyle // must be set prior to OpenBuf
	if nw, err := fn.OpenBuf(); err == nil {
		if tv.IsChanged() {
			ge.SetStatus(fmt.Sprintf("Note: Changes not yet saved in file: %v", tv.Buf.Filename))
		}
		tv.SetBuf(fn.Buf)
		if nw {
			ge.AutoSaveCheck(tv, vidx, fn)
		}
		ge.OpenNodes.Add(fn)
		fn.SetOpen()
		ge.SetActiveTextViewIdx(vidx)
		ext := filepath.Ext(fn.Name())
		langs := LangsForExt(ext)
		if len(langs) > 0 {
			ln := langs[0].Name
			// todo: completer funcs should be stored in language struct
			switch ln {
			case "Go":
				tv.SetCompleter(tv, CompleteGo, CompleteGoEdit)
			case "Tex":
				tv.SetCompleter(tv, CompleteTex, CompleteTexEdit)
			}
		}
	}
}

// NextViewFileNode sets the next text view to view file in given node (opens
// buffer if not already opened) -- if already being viewed, that is
// activated, returns text view and index
func (ge *Gide) NextViewFileNode(fn *giv.FileNode) (*giv.TextView, int) {
	tv, idx, ok := ge.TextViewForFileNode(fn)
	if ok {
		ge.SetActiveTextViewIdx(idx)
		return tv, idx
	}
	nv, nidx := ge.NextTextView()
	ge.ViewFileNode(nv, nidx, fn)
	return nv, nidx
}

// NextViewFile sets the next text view to view given file name -- include as
// much of name as possible to disambiguate -- will use the first matching --
// if already being viewed, that is activated -- returns textview and its
// index, false if not found
func (ge *Gide) NextViewFile(fnm gi.FileName) (*giv.TextView, int, bool) {
	fnk, ok := ge.Files.FindFile(string(fnm))
	if !ok {
		return nil, -1, false
	}
	fn := fnk.This.Embed(giv.KiT_FileNode).(*giv.FileNode)
	if fn.IsDir() {
		return nil, -1, false
	}
	nv, nidx := ge.NextViewFileNode(fn)
	return nv, nidx, true
}

// ViewFile views file in an existing TextView if it is already viewing that
// file, otherwise opens ViewFileNode in active buffer
func (ge *Gide) ViewFile(fnm gi.FileName) (*giv.TextView, int, bool) {
	fnk, ok := ge.Files.FindFile(string(fnm))
	if !ok {
		return nil, -1, false
	}
	fn := fnk.This.Embed(giv.KiT_FileNode).(*giv.FileNode)
	if fn.IsDir() {
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

// SelectOpenNode pops up a menu to select an open node (aka buffer) to view
// in current active textview
func (ge *Gide) SelectOpenNode() {
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
	gi.StringsChooserPopup(nl, def, tv, func(recv, send ki.Ki, sig int64, data interface{}) {
		ac := send.(*gi.Action)
		idx := ac.Data.(int)
		nb := ge.OpenNodes[idx]
		ge.ViewFileNode(tv, ge.ActiveTextViewIdx, nb)
	})
}

// CloneActiveView sets the next text view to view the same file currently being vieweds
// in the active view. returns text view and index
func (ge *Gide) CloneActiveView() (*giv.TextView, int) {
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
func (ge *Gide) SaveAllOpenNodes() {
	for _, ond := range ge.OpenNodes {
		ond.Buf.Save()
		ge.RunPostCmdsFileNode(ond)
	}
}

// TextViewSig handles all signals from the textviews
func (ge *Gide) TextViewSig(tv *giv.TextView, sig giv.TextViewSignals) {
	ge.SetActiveTextView(tv) // if we're sending signals, we're the active one!
	switch sig {
	case giv.TextViewISearch:
		fallthrough
	case giv.TextViewCursorMoved:
		ge.SetStatus("")
	}
}

//////////////////////////////////////////////////////////////////////////////////////
//   Links

// TextLinkHandler is the Gide handler for text links -- preferred one b/c
// directly connects to correct Gide project
func TextLinkHandler(tl gi.TextLink) bool {
	ftv, _ := tl.Widget.(*giv.TextView)
	gek, ok := tl.Widget.ParentByType(KiT_Gide, true)
	if ok {
		ge := gek.Embed(KiT_Gide).(*Gide)
		ur := tl.URL
		// todo: use net/url package for more systematic parsing
		switch {
		case strings.HasPrefix(ur, "find:///"):
			ge.OpenFindURL(ur, ftv)
		case strings.HasPrefix(ur, "spell:///"):
			ge.OpenSpellURL(ur, ftv)
		case strings.HasPrefix(ur, "file:///"):
			ge.OpenFileURL(ur)
		}
	}
	return true
}

// // URLHandler is the Gide handler for urls --
// func URLHandler(url string) bool {
// 	return true
// }

// OpenFileURL opens given file:/// url
func (ge *Gide) OpenFileURL(ur string) bool {
	up, err := url.Parse(ur)
	if err != nil {
		log.Printf("Gide OpenFileURL parse err: %v\n", err)
		return false
	}
	fpath := up.Path[1:] // has double //
	pos := up.Fragment
	tv, _, ok := ge.NextViewFile(gi.FileName(fpath))
	if !ok {
		_, fnm := filepath.Split(fpath)
		tv, _, ok = ge.NextViewFile(gi.FileName(fnm))
		if !ok {
			gi.PromptDialog(ge.Viewport, gi.DlgOpts{Title: "Couldn't Open File at Link", Prompt: fmt.Sprintf("Could not find or open file path in project: %v", fpath)}, true, false, nil, nil)
			return false
		}
	}
	if pos == "" {
		return true
	}
	// fmt.Printf("pos: %v\n", pos)
	txpos := giv.TextPos{}
	if txpos.FromString(pos) {
		tv.SetCursorShow(txpos)
	}
	return true
}

func init() {
	// gi.URLHandler = URLHandler
	gi.TextLinkHandler = TextLinkHandler
}

//////////////////////////////////////////////////////////////////////////////////////
//   Close / Quit Req

// NChangedFiles returns number of opened files with unsaved changes
func (ge *Gide) NChangedFiles() int {
	return ge.OpenNodes.NChanged()
}

// CurPanel returns the splitter panel that currently has keyboard focus
// CloseWindowReq is called when user tries to close window -- we
// automatically save the project if it already exists (no harm), and prompt
// to save open files -- if this returns true, then it is OK to close --
// otherwise not
func (ge *Gide) CloseWindowReq() bool {
	ge.SaveProjIfExists()
	nch := ge.NChangedFiles()
	if nch == 0 {
		return true
	}
	gi.ChoiceDialog(ge.Viewport, gi.DlgOpts{Title: "Close Project: There are Unsaved Files",
		Prompt: fmt.Sprintf("In Project: %v There are <b>%v</b> opened files with <b>unsaved changes</b> -- do you want to save all or cancel closing this project and review  / save those files first?", ge.Nm, nch)},
		[]string{"Cancel", "Save All", "Close Without Saving"},
		ge.This, func(recv, send ki.Ki, sig int64, data interface{}) {
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
		mfr, ok := win.MainWidget()
		if !ok {
			continue
		}
		gek, ok := mfr.ChildByName("gide", 0)
		if !ok {
			continue
		}
		ge := gek.Embed(KiT_Gide).(*Gide)
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
func (ge *Gide) PanelIsOpen(panel int) bool {
	sv := ge.SplitView()
	if sv == nil {
		return false
	}
	if panel < 0 || panel >= len(sv.Kids) {
		return false
	}
	if sv.Splits[panel] <= 0.01 {
		return false
	}
	return true
}

// CurPanel returns the splitter panel that currently has keyboard focus
func (ge *Gide) CurPanel() int {
	sv := ge.SplitView()
	if sv == nil {
		return -1
	}
	for i, ski := range sv.Kids {
		_, sk := gi.KiToNode2D(ski)
		if sk.ContainsFocus() {
			return i
		}
	}
	return -1 // nobody
}

// FocusOnPanel moves keyboard focus to given panel -- returns false if nothing at that tab
func (ge *Gide) FocusOnPanel(panel int) bool {
	sv := ge.SplitView()
	if sv == nil {
		return false
	}
	win := ge.ParentWindow()
	switch panel {
	case TextView1Idx:
		ge.SetActiveTextViewIdx(0)
	case TextView2Idx:
		ge.SetActiveTextViewIdx(1)
	case MainTabsIdx:
		tv := ge.MainTabs()
		ct, _, has := tv.CurTab()
		if has {
			win.FocusNext(ct)
		} else {
			return false
		}
	case VisTabsIdx:
		tv := ge.VisTabs()
		ct, _, has := tv.CurTab()
		if has {
			win.FocusNext(ct)
		} else {
			return false
		}
	default:
		ski := sv.Kids[panel]
		win.FocusNext(ski)
	}
	return true
}

// FocusNextPanel moves the keyboard focus to the next panel to the right
func (ge *Gide) FocusNextPanel() {
	sv := ge.SplitView()
	if sv == nil {
		return
	}
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
func (ge *Gide) FocusPrevPanel() {
	sv := ge.SplitView()
	if sv == nil {
		return
	}
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

// MainTabByName returns a MainTabs (first set of tabs) tab with given name,
// and its index -- returns false if not found
func (ge *Gide) MainTabByName(label string) (gi.Node2D, int, bool) {
	tv := ge.MainTabs()
	return tv.TabByName(label)
}

// SelectMainTabByName Selects given main tab, and returns all of its contents as well.
func (ge *Gide) SelectMainTabByName(label string) (gi.Node2D, int, bool) {
	tv := ge.MainTabs()
	widg, idx, ok := ge.MainTabByName(label)
	if ok {
		tv.SelectTabIndex(idx)
	}
	return widg, idx, ok
}

// FindOrMakeMainTab returns a MainTabs (first set of tabs) tab with given
// name, first by looking for an existing one, and if not found, making a new
// one with widget of given type.  if sel, then select it.  returns widget and tab index.
func (ge *Gide) FindOrMakeMainTab(label string, typ reflect.Type, sel bool) (gi.Node2D, int) {
	tv := ge.MainTabs()
	widg, idx, ok := ge.MainTabByName(label)
	if ok {
		if sel {
			tv.SelectTabIndex(idx)
		}
		return widg, idx
	}
	widg, idx = tv.AddNewTab(typ, label)
	if sel {
		tv.SelectTabIndex(idx)
	}
	return widg, idx
}

// ConfigOutputTextView configures a command-output textview within given parent layout
func (ge *Gide) ConfigOutputTextView(ly *gi.Layout) *giv.TextView {
	ly.Lay = gi.LayoutVert
	ly.SetStretchMaxWidth()
	ly.SetStretchMaxHeight()
	ly.SetMinPrefWidth(units.NewValue(20, units.Ch))
	ly.SetMinPrefHeight(units.NewValue(10, units.Ch))
	var tv *giv.TextView
	if ly.HasChildren() {
		tv = ly.KnownChild(0).Embed(giv.KiT_TextView).(*giv.TextView)
	} else {
		tv = ly.AddNewChild(giv.KiT_TextView, ly.Nm).(*giv.TextView)
	}

	if ge.Prefs.Editor.WordWrap {
		tv.SetProp("white-space", gi.WhiteSpacePreWrap)
	} else {
		tv.SetProp("white-space", gi.WhiteSpacePre)
	}
	tv.SetProp("tab-size", 8) // std for output
	tv.SetProp("font-family", ge.Prefs.Editor.FontFamily)
	return tv
}

// FindOrMakeMainTabTextView returns a MainTabs (first set of tabs) tab with given
// name, first by looking for an existing one, and if not found, making a new
// one with a Layout and then a TextView in it.  if sel, then select it.
// returns widget and tab index.
func (ge *Gide) FindOrMakeMainTabTextView(label string, sel bool) (*giv.TextView, int) {
	lyk, idx := ge.FindOrMakeMainTab(label, gi.KiT_Layout, sel)
	ly := lyk.Embed(gi.KiT_Layout).(*gi.Layout)
	tv := ge.ConfigOutputTextView(ly)
	return tv, idx
}

// FindOrMakeCmdBuf creates the buffer for command output, or returns
// existing. If clear is true, then any existing buffer is cleared.
// Returns true if new buffer created.
func (ge *Gide) FindOrMakeCmdBuf(cmdNm string, clear bool) (*giv.TextBuf, bool) {
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
	ge.CmdBufs[cmdNm] = buf
	return buf, true
}

// FindOrMakeCmdTab creates the tab to show command output, including making a
// buffer object to save output from the command. returns true if a new buffer
// was created, false if one already existed. if sel, select tab.  if clearBuf, then any
// existing buffer is cleared.  Also returns index of tab.
func (ge *Gide) FindOrMakeCmdTab(cmdNm string, sel bool, clearBuf bool) (*giv.TextBuf, *giv.TextView, int, bool) {
	buf, nw := ge.FindOrMakeCmdBuf(cmdNm, clearBuf)
	ctv, idx := ge.FindOrMakeMainTabTextView(cmdNm, sel)
	ctv.SetInactive()
	ctv.SetBuf(buf)
	return buf, ctv, idx, nw
}

// VisTabByName returns a VisTabs (second set of tabs for visualizations) tab
// with given name, and its index -- returns false if not found
func (ge *Gide) VisTabByName(label string) (gi.Node2D, int, bool) {
	tv := ge.VisTabs()
	if tv == nil {
		return nil, -1, false
	}
	return tv.TabByName(label)
}

//////////////////////////////////////////////////////////////////////////////////////
//    Commands / Tabs

// ExecCmdName executes command of given name -- this is the final common
// pathway for all command invokation except on a node.  if sel, select tab.
// if clearBuf, clear the buffer prior to command
func (ge *Gide) ExecCmdName(cmdNm CmdName, sel bool, clearBuf bool) {
	cmd, _, ok := AvailCmds.CmdByName(cmdNm, true)
	if !ok {
		return
	}
	ge.SetArgVarVals()
	cbuf, _, _, _ := ge.FindOrMakeCmdTab(cmd.Name, sel, clearBuf)
	cmd.Run(ge, cbuf)
}

// ExecCmdNameFileNode executes command of given name on given node
func (ge *Gide) ExecCmdNameFileNode(fn *giv.FileNode, cmdNm CmdName, sel bool, clearBuf bool) {
	cmd, _, ok := AvailCmds.CmdByName(cmdNm, true)
	if !ok {
		return
	}
	SetArgVarVals(&ArgVarVals, string(fn.FPath), &ge.Prefs, nil)
	cbuf, _, _, _ := ge.FindOrMakeCmdTab(cmd.Name, sel, clearBuf)
	cmd.Run(ge, cbuf)
}

// ExecCmd pops up a menu to select a command appropriate for the current
// active text view, and shows output in MainTab with name of command
func (ge *Gide) ExecCmd() {
	tv := ge.ActiveTextView()
	if tv == nil {
		return
	}
	var cmds []string
	if len(ge.ActiveLangs) == 0 {
		cmds = AvailCmds.FilterCmdNames(LangNames{ge.Prefs.MainLang}, ge.Prefs.VersCtrl)
	} else {
		cmds = AvailCmds.FilterCmdNames(ge.ActiveLangs, ge.Prefs.VersCtrl)
	}
	hsz := len(ge.CmdHistory)
	lastCmd := ""
	if hsz > 0 {
		lastCmd = string(ge.CmdHistory[hsz-1])
	}
	gi.StringsChooserPopup(cmds, lastCmd, tv, func(recv, send ki.Ki, sig int64, data interface{}) {
		ac := send.(*gi.Action)
		cmdNm := CmdName(ac.Text)
		ge.CmdHistory.Add(cmdNm)          // only save commands executed via chooser
		ge.ExecCmdName(cmdNm, true, true) // sel, clear
	})
}

// ExecCmdFileNode pops up a menu to select a command appropriate for the given node,
// and shows output in MainTab with name of command
func (ge *Gide) ExecCmdFileNode(fn *giv.FileNode) {
	langs := LangNamesForFilename(fn.Nm)
	cmds := AvailCmds.FilterCmdNames(langs, ge.Prefs.VersCtrl)
	gi.StringsChooserPopup(cmds, "", ge, func(recv, send ki.Ki, sig int64, data interface{}) {
		ac := send.(*gi.Action)
		ge.ExecCmdNameFileNode(fn, CmdName(ac.Text), true, true) // sel, clearbuf
	})
}

// SetArgVarVals sets the ArgVar values for commands, from Gide values
func (ge *Gide) SetArgVarVals() {
	tv := ge.ActiveTextView()
	if tv == nil || tv.Buf == nil {
		SetArgVarVals(&ArgVarVals, "", &ge.Prefs, tv)
	} else {
		SetArgVarVals(&ArgVarVals, string(tv.Buf.Filename), &ge.Prefs, tv)
	}
}

// ExecCmds executes a sequence of commands, sel = select tab, clearBuf = clear buffer
func (ge *Gide) ExecCmds(cmdNms CmdNames, sel bool, clearBuf bool) {
	for _, cmdNm := range cmdNms {
		ge.ExecCmdName(cmdNm, sel, clearBuf)
	}
}

// ExecCmdsFileNode executes a sequence of commands on file node, sel = select tab, clearBuf = clear buffer
func (ge *Gide) ExecCmdsFileNode(fn *giv.FileNode, cmdNms CmdNames, sel bool, clearBuf bool) {
	for _, cmdNm := range cmdNms {
		ge.ExecCmdNameFileNode(fn, cmdNm, sel, clearBuf)
	}
}

// Build runs the BuildCmds set for this project
func (ge *Gide) Build() {
	if len(ge.Prefs.BuildCmds) == 0 {
		gi.PromptDialog(ge.Viewport, gi.DlgOpts{Title: "No BuildCmds Set", Prompt: fmt.Sprintf("You need to set the BuildCmds in the Project Preferences")}, true, false, nil, nil)
		return
	}
	ge.ExecCmds(ge.Prefs.BuildCmds, true, true)
}

// Run runs the RunCmds set for this project
func (ge *Gide) Run() {
	if len(ge.Prefs.RunCmds) == 0 {
		gi.PromptDialog(ge.Viewport, gi.DlgOpts{Title: "No RunCmds Set", Prompt: fmt.Sprintf("You need to set the RunCmds in the Project Preferences")}, true, false, nil, nil)
		return
	}
	ge.ExecCmds(ge.Prefs.RunCmds, true, true)
}

// Commit commits the current changes using relevant VCS tool, and updates the changelog.
// Checks for VCS setting and
func (ge *Gide) Commit() {
	if ge.Prefs.VersCtrl == "" {
		gi.PromptDialog(ge.Viewport, gi.DlgOpts{Title: "No VersCtrl Set", Prompt: fmt.Sprintf("You need to set the VersCtrl in the Project Preferences")}, true, false, nil, nil)
		return
	}

	nch := ge.NChangedFiles()
	if nch > 0 {
		gi.ChoiceDialog(ge.Viewport, gi.DlgOpts{Title: "Commit: There are Unsaved Files",
			Prompt: fmt.Sprintf("In Project: %v There are <b>%v</b> opened files with <b>unsaved changes</b> -- do you want to save all or cancel commit and review  / save those files first?", ge.Nm, nch)},
			[]string{"Cancel", "Save All", "Commit Without Saving"},
			ge.This, func(recv, send ki.Ki, sig int64, data interface{}) {
				switch sig {
				case 0:
					// do nothing, will have bailed already
				case 1:
					ge.SaveAllOpenNodes()
				case 2:
					ge.CommitNoChecks()
				}
			})
	} else {
		ge.CommitNoChecks()
	}
}

// CommitNoChecks does the commit without any further checks for VCS, and unsaved files
func (ge *Gide) CommitNoChecks() {
	cmds := AvailCmds.FilterCmdNames(ge.ActiveLangs, ge.Prefs.VersCtrl)
	cmdnm := ""
	for _, cm := range cmds {
		if strings.Contains(cm, "Commit") {
			cmdnm = cm
			break
		}
	}
	if cmdnm == "" {
		gi.PromptDialog(ge.Viewport, gi.DlgOpts{Title: "No Commit command found", Prompt: fmt.Sprintf("Could not find Commit command in list of avail commands -- this is usually a programmer error -- check preferences settings etc")}, true, false, nil, nil)
		return
	}
	ge.SetArgVarVals() // need to set before setting prompt string below..

	gi.StringPromptDialog(ge.Viewport, "", "Enter commit message here..",
		gi.DlgOpts{Title: "Commit Message", Prompt: "Please enter your commit message here -- this will be recorded along with other information from the commit in the project's ChangeLog, which can be viewed under Proj Prefs menu item -- author information comes from User settings in GoGi Preferences."},
		ge.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			dlg := send.(*gi.Dialog)
			if sig == int64(gi.DialogAccepted) {
				msg := gi.StringPromptDialogValue(dlg)
				ArgVarVals["{PromptString1}"] = msg
				CmdNoUserPrompt = true // don't re-prompt!
				ge.Prefs.ChangeLog.Add(ChangeRec{Date: giv.FileTime(time.Now()), Author: gi.Prefs.User.Name, Email: gi.Prefs.User.Email, Message: msg})
				ge.ExecCmdName(CmdName(cmdnm), true, true) // must be wait
				ge.CommitUpdtLog(cmdnm)
			}
		})
}

// CommitUpdtLog grabs info from buffer in main tabs about the commit, and
// updates the changelog record
func (ge *Gide) CommitUpdtLog(cmdnm string) {
	ctv, _ := ge.FindOrMakeMainTabTextView(cmdnm, false) // don't sel
	if ctv == nil {
		return
	}
	if ctv.Buf == nil {
		return
	}
	// todo: process text!
	ge.SaveProjIfExists()
}

//////////////////////////////////////////////////////////////////////////////////////
//    Find / Replace

// Find does Find / Replace in files, using given options and filters -- opens up a
// main tab with the results and further controls.
func (ge *Gide) Find(find, repl string, ignoreCase bool, langs LangNames, curFileOnly bool) {
	if find == "" {
		return
	}
	ge.Prefs.Find.Find = find
	ge.Prefs.Find.Replace = repl
	ge.Prefs.Find.IgnoreCase = ignoreCase
	ge.Prefs.Find.Langs = langs
	ge.Prefs.Find.CurFile = curFileOnly

	fbuf, _ := ge.FindOrMakeCmdBuf("Find", true)
	fvi, _ := ge.FindOrMakeMainTab("Find", KiT_FindView, true) // sel
	fv := fvi.Embed(KiT_FindView).(*FindView)
	fv.UpdateView(ge, ge.Prefs.Find)
	ftv := fv.TextView()
	ftv.SetInactive()
	ftv.SetBuf(fbuf)

	root := ge.Files.Embed(giv.KiT_FileNode).(*giv.FileNode)

	var res []FileSearchResults
	if curFileOnly {
		tv := ge.ActiveTextView()
		ond, _, got := ge.OpenNodeForTextView(tv)
		if got {
			cnt, matches := tv.Buf.Search([]byte(find), ignoreCase)
			res = append(res, FileSearchResults{ond, cnt, matches})
		}
	} else {
		res = FileTreeSearch(root, find, ignoreCase, langs)
	}

	outlns := make([][]byte, 0, 100)
	outmus := make([][]byte, 0, 100) // markups
	for _, fs := range res {
		fp := fs.Node.Info.Path
		fn := fs.Node.RelPath()
		fbStLn := len(outlns) // find buf start ln
		lstr := fmt.Sprintf(`%v: %v`, fn, fs.Count)
		outlns = append(outlns, []byte(lstr))
		mstr := fmt.Sprintf(`<a href="find:///%v#R%vN%v">%v</a>: %v`, fp, fbStLn, fs.Count, fn, fs.Count)
		outmus = append(outmus, []byte(mstr))
		for _, mt := range fs.Matches {
			ln := mt.Reg.Start.Ln + 1
			ch := mt.Reg.Start.Ch + 1
			ech := mt.Reg.End.Ch + 1
			nomu := bytes.Replace(mt.Text, []byte("<mark>"), nil, -1)
			nomu = bytes.Replace(nomu, []byte("</mark>"), nil, -1)
			nomus := html.EscapeString(string(nomu))
			lstr = fmt.Sprintf(`	%v: %s`, fn, nomus)
			outlns = append(outlns, []byte(lstr))
			mstr = fmt.Sprintf(`	<a href="find:///%v#R%vN%vL%vC%v-L%vC%v">%v</a>: %s`, fp, fbStLn, fs.Count, ln, ch, ln, ech, fn, mt.Text)
			outmus = append(outmus, []byte(mstr))
		}
		outlns = append(outlns, []byte(""))
		outmus = append(outmus, []byte(""))
	}
	ltxt := bytes.Join(outlns, []byte("\n"))
	mtxt := bytes.Join(outmus, []byte("\n"))
	fbuf.AppendTextMarkup(ltxt, mtxt, false, true) // no save undo, yes signal
	ftv.CursorStartDoc()
	ok := ftv.CursorNextLink(false) // no wrap
	if ok {
		ftv.OpenLinkAt(ftv.CursorPos)
		ftv.CursorNextLink(false)
	}
	ge.FocusOnPanel(MainTabsIdx)
}

// Spell checks spelling in files
func (ge *Gide) Spell() {
	fbuf, _ := ge.FindOrMakeCmdBuf("Spell", true)
	svi, _ := ge.FindOrMakeMainTab("Spell", KiT_SpellView, true) // sel
	sv := svi.Embed(KiT_SpellView).(*SpellView)
	sv.UpdateView(ge, ge.Prefs.Spell)
	stv := sv.TextView()
	stv.SetInactive()
	stv.SetBuf(fbuf)

	fp := string(ge.ActiveFilename)
	if fp == "" {
		return
	}

	tv := ge.ActiveTextView()
	spell.NewSpellCheck(tv.Buf.Txt)
	word, suggests := spell.NextUnknownWord()
	sv.SetUnknownAndSuggest(word, suggests)

	//unknowns, err := spell.CheckFile(fp)
	//if err != nil {
	//	gi.PromptDialog(ge.Viewport, gi.DlgOpts{Title: "Dictionaries not found", Prompt: err.Error()}, true, false, nil, nil)
	//	return
	//}
	//outlns := make([][]byte, 0, 100)
	//outmus := make([][]byte, 0, 100) // markups
	//fbStLn := len(outlns)            // find buf start ln
	//lstr := fmt.Sprintf(`%v:`, fn)
	//outlns = append(outlns, []byte(lstr))
	//mstr := fmt.Sprintf(`<a href="spell:///%v#R%v">%v</a>`, fp, fbStLn, fn)
	//outmus = append(outmus, []byte(mstr))
	//for _, u := range unknowns {
	//	lstr := fmt.Sprintf(`%v: %v`, u.LineNo, u.Word)
	//	outlns = append(outlns, []byte(lstr))
	//	mstr = fmt.Sprintf(`	<a href="spell:///%v#R%vL%v-L%v">%v:%v</a>`, fp, fbStLn, u.LineNo, u.LineNo, u.LineNo, u.Word)
	//	outmus = append(outmus, []byte(mstr))
	//}
	//ltxt := bytes.Join(outlns, []byte("\n"))
	//mtxt := bytes.Join(outmus, []byte("\n"))
	//fbuf.AppendTextMarkup(ltxt, mtxt, false, true) // no save undo, yes signal
	//stv.CursorStartDoc()
	ge.FocusOnPanel(MainTabsIdx)
}

// ParseOpenFindURL parses and opens given find:/// url from Find, return text
// region encoded in url, and starting line of results in find buffer, and
// number of results returned -- for parsing all the find results
func (ge *Gide) ParseOpenFindURL(ur string, ftv *giv.TextView) (tv *giv.TextView, reg giv.TextRegion, findBufStLn, findCount int, ok bool) {
	up, err := url.Parse(ur)
	if err != nil {
		log.Printf("FindView OpenFindURL parse err: %v\n", err)
		return
	}
	fpath := up.Path[1:] // has double //
	pos := up.Fragment
	tv, _, ok = ge.NextViewFile(gi.FileName(fpath))
	if !ok {
		gi.PromptDialog(ge.Viewport, gi.DlgOpts{Title: "Couldn't Open File at Link", Prompt: fmt.Sprintf("Could not find or open file path in project: %v", fpath)}, true, false, nil, nil)
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

// HighlightFinds highlights all the find results in ftv buffer
func (ge *Gide) HighlightFinds(tv, ftv *giv.TextView, fbStLn, fCount int, find string) {
	lnka := []byte(`<a href="`)
	lnkasz := len(lnka)

	fb := ftv.Buf

	if len(tv.Highlights) != fCount { // highlight
		hi := make([]giv.TextRegion, fCount)
		for i := 0; i < fCount; i++ {
			fln := fbStLn + 1 + i
			ltxt := fb.Markup[fln]
			fpi := bytes.Index(ltxt, lnka)
			if fpi < 0 {
				continue
			}
			fpi += lnkasz
			epi := fpi + bytes.Index(ltxt[fpi:], []byte(`"`))
			lnk := string(ltxt[fpi:epi])
			iup, err := url.Parse(lnk)
			if err != nil {
				continue
			}
			ireg := giv.TextRegion{}
			lidx := strings.Index(iup.Fragment, "L")
			ireg.FromString(iup.Fragment[lidx:])
			hi[i] = ireg
		}
		tv.Highlights = hi
	}
}

// OpenFindURL opens given find:/// url from Find -- delegates to FindView
func (ge *Gide) OpenFindURL(ur string, ftv *giv.TextView) bool {
	fvk, ok := ftv.ParentByType(KiT_FindView, true)
	if !ok {
		return false
	}
	fv := fvk.(*FindView)
	return fv.OpenFindURL(ur, ftv)
}

// OpenSpellURL opens given spell:/// url from Spell -- delegates to SpellView
func (ge *Gide) OpenSpellURL(ur string, stv *giv.TextView) bool {
	svk, ok := stv.ParentByType(KiT_SpellView, true)
	if !ok {
		return false
	}
	fv := svk.(*SpellView)
	return fv.OpenSpellURL(ur, stv)
}

//////////////////////////////////////////////////////////////////////////////////////
//    Registers

// RegisterSave saves current selection in active text view to register of given name
// returns true if saved
func (ge *Gide) RegisterSave(name string) bool {
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
	if AvailRegisters == nil {
		AvailRegisters = make(Registers, 100)
	}
	AvailRegisters[name] = string(sel.ToBytes())
	AvailRegisters.SavePrefs()
	return true
}

// RegisterPaste pastes register of given name into active text view
// returns true if pasted
func (ge *Gide) RegisterPaste(name RegisterName) bool {
	if name == "" {
		return false
	}
	str, ok := AvailRegisters[string(name)]
	if !ok {
		return false
	}
	tv := ge.ActiveTextView()
	if tv.Buf == nil {
		return false
	}
	tv.InsertAtCursor([]byte(str))
	return true
}

// CommentOut comments-out selected lines in active text view
func (ge *Gide) CommentOut() bool {
	tv := ge.ActiveTextView()
	if tv.Buf == nil {
		return false
	}
	sel := tv.Selection()
	if sel == nil {
		return false
	}
	cmt := []byte("// ")
	ls := LangsForFilename(string(tv.Buf.Filename))
	if len(ls) == 1 {
		cmt = []byte(ls[0].Comment)
	}
	tv.Buf.CommentRegion(sel.Reg.Start.Ln, sel.Reg.End.Ln, cmt, tv.Sty.Text.TabSize)
	tv.SelectReset()
	return true
}

// Indent indents selected lines in active view
func (ge *Gide) Indent() bool {
	tv := ge.ActiveTextView()
	if tv.Buf == nil {
		return false
	}
	sel := tv.Selection()
	if sel == nil {
		return false
	}
	// todo: add indent chars to langs
	// cmt := []byte("// ")
	// ls := LangsForFilename(string(tv.Buf.Filename))
	// if len(ls) == 1 {
	//		cmt = []byte(ls[0].Comment)
	// }
	tv.Buf.AutoIndentRegion(sel.Reg.Start.Ln, sel.Reg.End.Ln, tv.Opts.SpaceIndent, tv.Sty.Text.TabSize,
		giv.DefaultIndentStrings, giv.DefaultUnindentStrings)
	tv.SelectReset()
	return true
}

//////////////////////////////////////////////////////////////////////////////////////
//    StatusBar

// SetStatus updates the statusbar label with given message, along with other status info
func (ge *Gide) SetStatus(msg string) {
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
			if tv.Buf.Changed {
				fnm += "*"
			}
		}
		if tv.ISearchMode {
			msg = fmt.Sprintf("\tISearch: %v (n=%v)\t%v", tv.ISearchString, len(tv.SearchMatches), msg)
		}
	}

	str := fmt.Sprintf("%v\t<b>%v:</b>\t(%v,%v)\t%v", ge.Nm, fnm, ln, ch, msg)
	lbl.SetText(str)
	sb.UpdateEnd(updt)
}

//////////////////////////////////////////////////////////////////////////////////////
//    Defaults, Prefs

// Defaults sets new project defaults based on overall preferences
func (ge *Gide) Defaults() {
	ge.Prefs.Files = Prefs.Files
	ge.Prefs.Editor = Prefs.Editor
	ge.Prefs.Splits = []float32{.1, .3, .3, .3, 0}
	ge.Files.DirsOnTop = ge.Prefs.Files.DirsOnTop
	ge.Files.SetChildType(KiT_FileNode)
}

// GrabPrefs grabs the current project preference settings from various
// places, e.g., prior to saving or editing.
func (ge *Gide) GrabPrefs() {
	sv := ge.SplitView()
	if sv != nil {
		ge.Prefs.Splits = sv.Splits
	}
	ge.Prefs.OpenDirs = ge.Files.OpenDirs
}

// ApplyPrefs applies current project preference settings into places where
// they are used -- only for those done prior to loading
func (ge *Gide) ApplyPrefs() {
	ge.ProjFilename = ge.Prefs.ProjFilename
	ge.ProjRoot = ge.Prefs.ProjRoot
	ge.Files.OpenDirs = ge.Prefs.OpenDirs
	ge.Files.DirsOnTop = ge.Prefs.Files.DirsOnTop
	sv := ge.SplitView()
	if sv != nil {
		for i := 0; i < NTextViews; i++ {
			txly := sv.KnownChild(1 + i).(*gi.Layout)
			txed := txly.KnownChild(0).(*giv.TextView)
			txed.Opts.LineNos = ge.Prefs.Editor.LineNos
			txed.Opts.AutoIndent = true
			txed.Opts.Completion = ge.Prefs.Editor.Completion
		}
	}
}

// ApplyPrefsAction applies current preferences to the project, and updates the project
func (ge *Gide) ApplyPrefsAction() {
	ge.ApplyPrefs()
	ge.SetFullReRender()
	ge.UpdateProj()
}

// ProjPrefs allows editing of project preferences (settings specific to this project)
func (ge *Gide) ProjPrefs() {
	sv, _ := ProjPrefsView(&ge.Prefs)
	// we connect to changes and apply them
	sv.ViewSig.Connect(ge.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		gee, _ := recv.Embed(KiT_Gide).(*Gide)
		gee.ApplyPrefsAction()
	})
}

// SetSplit sets splitter to given named setting
func (ge *Gide) SetSplit(split SplitName) {
	sv := ge.SplitView()
	if sv != nil {
		sp, _, ok := AvailSplits.SplitByName(split)
		if ok {
			sv.SetSplits(sp.Splits...)
		}
	}
}

// SaveSplit saves current splitter settings to named splitter settings, and
// saves to prefs file
func (ge *Gide) SaveSplit(name, desc string) {
	sv := ge.SplitView()
	if sv != nil {
		AvailSplits.Add(name, desc, sv.Splits)
		AvailSplits.SavePrefs()
	}
}

// EditSplits opens the SplitsView editor to customize saved splitter settings
func (ge *Gide) EditSplits() {
	SplitsView(&AvailSplits)
}

//////////////////////////////////////////////////////////////////////////////////////
//   GUI configs

// StdFrameConfig returns a TypeAndNameList for configuring a standard Frame
// -- can modify as desired before calling ConfigChildren on Frame using this
func (ge *Gide) StdFrameConfig() kit.TypeAndNameList {
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_ToolBar, "toolbar")
	config.Add(gi.KiT_SplitView, "splitview")
	config.Add(gi.KiT_Frame, "statusbar")
	return config
}

// StdConfig configures a standard setup of the overall Frame -- returns mods,
// updt from ConfigChildren and does NOT call UpdateEnd
func (ge *Gide) StdConfig() (mods, updt bool) {
	ge.Lay = gi.LayoutVert
	ge.SetProp("spacing", gi.StdDialogVSpaceUnits)
	config := ge.StdFrameConfig()
	mods, updt = ge.ConfigChildren(config, false)
	return
}

// SplitView returns the main SplitView
func (ge *Gide) SplitView() *gi.SplitView {
	svi, ok := ge.ChildByName("splitview", 2)
	if !ok {
		return nil
	}
	return svi.(*gi.SplitView)
}

// FileTree returns the main FileTree
func (ge *Gide) FileTree() *giv.TreeView {
	split := ge.SplitView()
	if split != nil {
		tv := split.KnownChild(FileTreeIdx).KnownChild(0).(*giv.TreeView)
		return tv
	}
	return nil
}

// TextViewByIndex returns the TextView by index (0 or 1), nil if not found
func (ge *Gide) TextViewByIndex(idx int) *giv.TextView {
	if idx < 0 || idx >= NTextViews {
		log.Printf("Gide: text view index out of range: %v\n", idx)
		return nil
	}
	split := ge.SplitView()
	if split != nil {
		svk := split.KnownChild(TextView1Idx + idx).KnownChild(0)
		return svk.Embed(giv.KiT_TextView).(*giv.TextView)
	}
	return nil
}

// MainTabs returns the main TabView
func (ge *Gide) MainTabs() *gi.TabView {
	split := ge.SplitView()
	if split != nil {
		tv := split.KnownChild(MainTabsIdx).Embed(gi.KiT_TabView).(*gi.TabView)
		return tv
	}
	return nil
}

// VisTabs returns the second, visualization TabView
func (ge *Gide) VisTabs() *gi.TabView {
	split := ge.SplitView()
	if split != nil {
		tv := split.KnownChild(VisTabsIdx).Embed(gi.KiT_TabView).(*gi.TabView)
		return tv
	}
	return nil
}

// ToolBar returns the main toolbar
func (ge *Gide) ToolBar() *gi.ToolBar {
	tbi, ok := ge.ChildByName("toolbar", 2)
	if !ok {
		return nil
	}
	return tbi.(*gi.ToolBar)
}

// StatusBar returns the statusbar widget
func (ge *Gide) StatusBar() *gi.Frame {
	tbi, ok := ge.ChildByName("statusbar", 2)
	if !ok {
		return nil
	}
	return tbi.(*gi.Frame)
}

// StatusLabel returns the statusbar label widget
func (ge *Gide) StatusLabel() *gi.Label {
	sb := ge.StatusBar()
	if sb != nil {
		return sb.KnownChild(0).Embed(gi.KiT_Label).(*gi.Label)
	}
	return nil
}

// ConfigStatusBar configures statusbar with label
func (ge *Gide) ConfigStatusBar() {
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
	lbl.SetProp("vertical-align", gi.AlignTop)
	lbl.SetProp("margin", 0)
	lbl.SetProp("padding", 0)
	lbl.SetProp("tab-size", 4)
}

// ConfigToolbar adds a Gide toolbar.
func (ge *Gide) ConfigToolbar() {
	tb := ge.ToolBar()
	if tb.HasChildren() {
		return
	}
	tb.SetStretchMaxWidth()
	giv.ToolBarView(ge, ge.Viewport, tb)
}

// SplitViewConfig returns a TypeAndNameList for configuring the SplitView
func (ge *Gide) SplitViewConfig() kit.TypeAndNameList {
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_Frame, "filetree")
	for i := 0; i < NTextViews; i++ {
		config.Add(gi.KiT_Layout, fmt.Sprintf("textview-%v", i))
	}
	config.Add(gi.KiT_TabView, "main-tabs")
	config.Add(gi.KiT_TabView, "vis-tabs")
	return config
}

var fnFolderProps = ki.Props{
	"icon":     "folder-open",
	"icon-off": "folder",
}

// ConfigSplitView configures the SplitView.
func (ge *Gide) ConfigSplitView() {
	split := ge.SplitView()
	if split == nil {
		return
	}
	split.Dim = gi.X
	//	split.Dim = gi.Y

	config := ge.SplitViewConfig()
	mods, updt := split.ConfigChildren(config, true)
	if mods {
		ftfr := split.KnownChild(0).(*gi.Frame)
		if !ftfr.HasChildren() {
			ft := ftfr.AddNewChild(giv.KiT_FileTreeView, "filetree").(*giv.FileTreeView)
			ft.SetRootNode(&ge.Files)
			ft.TreeViewSig.Connect(ge.This, func(recv, send ki.Ki, sig int64, data interface{}) {
				if data == nil {
					return
				}
				tvn, _ := data.(ki.Ki).Embed(giv.KiT_FileTreeView).(*giv.FileTreeView)
				gee, _ := recv.Embed(KiT_Gide).(*Gide)
				if tvn.SrcNode.Ptr != nil {
					fn := tvn.SrcNode.Ptr.Embed(giv.KiT_FileNode).(*giv.FileNode)
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
		}
		for i := 0; i < NTextViews; i++ {
			txly := split.KnownChild(1 + i).(*gi.Layout)
			txly.SetStretchMaxWidth()
			txly.SetStretchMaxHeight()
			txly.SetMinPrefWidth(units.NewValue(20, units.Ch))
			txly.SetMinPrefHeight(units.NewValue(10, units.Ch))
			if !txly.HasChildren() {
				ted := txly.AddNewChild(giv.KiT_TextView, fmt.Sprintf("textview-%v", i)).(*giv.TextView)
				ted.TextViewSig.Connect(ge.This, func(recv, send ki.Ki, sig int64, data interface{}) {
					gee, _ := recv.Embed(KiT_Gide).(*Gide)
					tee := send.Embed(giv.KiT_TextView).(*giv.TextView)
					gee.TextViewSig(tee, giv.TextViewSignals(sig))
				})
			}
		}
		split.SetSplits(ge.Prefs.Splits...)
		split.UpdateEnd(updt)
	}
	for i := 0; i < NTextViews; i++ {
		txly := split.KnownChild(1 + i).(*gi.Layout)
		txed := txly.KnownChild(0).(*giv.TextView)
		txed.Opts.LineNos = ge.Prefs.Editor.LineNos
		txed.Opts.AutoIndent = true
		txed.Opts.Completion = ge.Prefs.Editor.Completion
		if ge.Prefs.Editor.WordWrap {
			txed.SetProp("white-space", gi.WhiteSpacePreWrap)
		} else {
			txed.SetProp("white-space", gi.WhiteSpacePre)
		}
		txed.SetProp("tab-size", ge.Prefs.Editor.TabSize)
		txed.SetProp("font-family", ge.Prefs.Editor.FontFamily)
	}

	// set some properties always, even if no mods
	split.SetSplits(ge.Prefs.Splits...)
}

// FileNodeSelected is called whenever tree browser has file node selected
func (ge *Gide) FileNodeSelected(fn *giv.FileNode, tvn *giv.FileTreeView) {
	// if fn.IsDir() {
	// } else {
	// }
}

var GideBigFileSize = 10000000 // 10Mb?

// FileNodeOpened is called whenever file node is double-clicked in file tree
func (ge *Gide) FileNodeOpened(fn *giv.FileNode, tvn *giv.FileTreeView) {
	switch {
	case fn.IsDir():
		if !fn.IsOpen() {
			tvn.SetOpen()
			fn.OpenDir()
		}
	case fn.IsExec():
		ge.SetArgVarVals()
		ArgVarVals["{PromptString1}"] = string(fn.FPath)
		CmdNoUserPrompt = true                            // don't re-prompt!
		ge.ExecCmdName(CmdName("Run Prompt"), true, true) // sel, clear
	case strings.HasPrefix(fn.Info.Mime, "image"):
		ge.ExecCmdNameFileNode(fn, CmdName("Open File"), true, true) // sel, clear
	case fn.Info.Mime == "application/pdf":
		ge.ExecCmdNameFileNode(fn, CmdName("Open File"), true, true) // sel, clear
	default:
		if int(fn.Info.Size) > GideBigFileSize {
			gi.ChoiceDialog(ge.Viewport, gi.DlgOpts{Title: "File is relatively large",
				Prompt: fmt.Sprintf("The file: %v is relatively large at: %v -- really open for editing?", fn.Nm, fn.Info.Size)},
				[]string{"Open", "Cancel"},
				ge.This, func(recv, send ki.Ki, sig int64, data interface{}) {
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
}

// FileNodeClosed is called whenever file tree browser node is closed
func (ge *Gide) FileNodeClosed(fn *giv.FileNode, tvn *giv.FileTreeView) {
	if fn.IsDir() {
		if fn.IsOpen() {
			fn.CloseDir()
		}
	}
}

func (ge *Gide) GideKeys(kt *key.ChordEvent) {
	kf := KeyFunNil
	kc := kt.Chord()
	gkf := gi.KeyFun(kc)
	if ge.KeySeq1 != "" {
		kf = KeyFun(ge.KeySeq1, kc)
		if kf == KeyFunNil || kc == "Escape" {
			ge.SetStatus(string(ge.KeySeq1) + " " + string(kc) + " -- aborted")
			kt.SetProcessed() // abort key sequence, don't send esc to anyone else
			ge.KeySeq1 = ""
			return
		}
		ge.SetStatus(string(ge.KeySeq1) + " " + string(kc))
		ge.KeySeq1 = ""
	} else {
		kf = KeyFun(kc, "")
		if kf == KeyFunNeeds2 {
			ge.KeySeq1 = kt.Chord()
			ge.SetStatus(string(ge.KeySeq1))
			return
		}
	}

	switch gkf {
	case gi.KeyFunFind:
		kt.SetProcessed()
		giv.CallMethod(ge, "Find", ge.Viewport)
	}
	if kt.IsProcessed() {
		return
	}
	switch kf {
	case KeyFunNextPanel:
		kt.SetProcessed()
		ge.FocusNextPanel()
	case KeyFunPrevPanel:
		kt.SetProcessed()
		ge.FocusPrevPanel()
	case KeyFunFileOpen:
		kt.SetProcessed()
		giv.CallMethod(ge, "ViewFile", ge.Viewport)
	case KeyFunBufSelect:
		kt.SetProcessed()
		ge.SelectOpenNode()
	case KeyFunBufClone:
		kt.SetProcessed()
		ge.CloneActiveView()
	case KeyFunBufSave:
		kt.SetProcessed()
		ge.SaveActiveView()
	case KeyFunBufSaveAs:
		kt.SetProcessed()
		giv.CallMethod(ge, "SaveActiveViewAs", ge.Viewport)
	case KeyFunBufClose:
		kt.SetProcessed()
		ge.CloseActiveView()
	case KeyFunExecCmd:
		kt.SetProcessed()
		giv.CallMethod(ge, "ExecCmd", ge.Viewport)
	case KeyFunRegSave:
		kt.SetProcessed()
		giv.CallMethod(ge, "RegisterSave", ge.Viewport)
	case KeyFunRegPaste:
		kt.SetProcessed()
		giv.CallMethod(ge, "RegisterPaste", ge.Viewport)
	case KeyFunCommentOut:
		kt.SetProcessed()
		ge.CommentOut()
	case KeyFunIndent:
		kt.SetProcessed()
		ge.Indent()
	case KeyFunSetSplit:
		kt.SetProcessed()
		giv.CallMethod(ge, "SetSplit", ge.Viewport)
	}
}

func (ge *Gide) KeyChordEvent() {
	// need hipri to prevent 2-seq guys from being captured by others
	ge.ConnectEvent(oswin.KeyChordEvent, gi.HiPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		gee := recv.Embed(KiT_Gide).(*Gide)
		kt := d.(*key.ChordEvent)
		gee.GideKeys(kt)
	})
}

func (ge *Gide) Render2D() {
	ge.ToolBar().UpdateActions()
	if win := ge.ParentWindow(); win != nil {
		sv := ge.SplitView()
		if sv != nil {
			win.SetStartFocus(sv.This)
		}
		if !win.IsResizing() {
			win.MainMenuUpdateActives()
		}
	}
	ge.Frame.Render2D()
}

func (ge *Gide) ConnectEvents2D() {
	if ge.HasAnyScroll() {
		ge.LayoutScrollEvents()
	}
	ge.KeyChordEvent()
}

var GideProps = ki.Props{
	"background-color": &gi.Prefs.Colors.Background,
	"color":            &gi.Prefs.Colors.Font,
	"max-width":        -1,
	"max-height":       -1,
	"#title": ki.Props{
		"max-width":        -1,
		"horizontal-align": gi.AlignCenter,
		"vertical-align":   gi.AlignTop,
	},
	"ToolBar": ki.PropSlice{
		{"UpdateFiles", ki.Props{
			"shortcut":        "Command+U",
			"icon":            "update",
			"no-update-after": true,
		}},
		{"ViewFile", ki.Props{
			"label":           "Open",
			"icon":            "file-open",
			"no-update-after": true,
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"default-field": "ActiveFilename",
				}},
			},
		}},
		{"SaveActiveView", ki.Props{
			"label":           "Save",
			"no-update-after": true,
			"icon":            "file-save",
		}},
		{"SaveActiveViewAs", ki.Props{
			"label":           "Save As...",
			"icon":            "file-save",
			"no-update-after": true,
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"default-field": "ActiveFilename",
				}},
			},
		}},
		{"SelectOpenNode", ki.Props{
			"icon":            "file-text",
			"label":           "Edit",
			"no-update-after": true,
		}},
		{"sep-find", ki.BlankProp{}},
		{"Find", ki.Props{
			"label":           "Find...",
			"icon":            "search",
			"desc":            "Find / replace in all open folders in file browser",
			"no-update-after": true,
			"Args": ki.PropSlice{
				{"Search For", ki.Props{
					"default-field": "Prefs.Find.Find",
					"width":         "80",
				}},
				{"Replace With", ki.Props{
					"desc":          "Optional replace string -- replace will be controlled interactively in Find panel, including replace all",
					"default-field": "Prefs.Find.Replace",
					"width":         "80",
				}},
				{"Ignore Case", ki.Props{
					"default-field": "Prefs.Find.IgnoreCase",
				}},
				{"Languages", ki.Props{
					"desc":          "restrict find to files associated with these languages -- leave empty for all files",
					"default-field": "Prefs.Find.Langs",
				}},
				{"Current File Only", ki.Props{
					"desc":          "only look in current active file",
					"default-field": "Prefs.Find.CurFile",
				}},
			},
		}},
		{"Spell", ki.Props{
			"label":           "Spelling...",
			"no-update-after": true,
			"icon":            "spelling",
		}},
		{"sep-file", ki.BlankProp{}},
		{"Build", ki.Props{
			"no-update-after": true,
			"icon":            "terminal",
		}},
		{"Run", ki.Props{
			"no-update-after": true,
			"icon":            "terminal",
		}},
		{"Commit", ki.Props{
			"no-update-after": true,
			"icon":            "star",
		}},
		{"ExecCmd", ki.Props{
			"icon":            "terminal",
			"no-update-after": true, // key for methods that have own selector inside -- update runs before command is executed
		}},
		{"sep-splt", ki.BlankProp{}},
		{"Splits", ki.PropSlice{
			{"SetSplit", ki.Props{
				"label":           "Set",
				"submenu":         &AvailSplitNames,
				"no-update-after": true,
				"Args": ki.PropSlice{
					{"Split Name", ki.Props{}},
				},
			}},
			{"SaveSplit", ki.Props{
				"label":           "Save...",
				"no-update-after": true,
				"Args": ki.PropSlice{
					{"Name", ki.Props{
						"width": "60",
					}},
					{"Desc", ki.Props{
						"width": "60",
					}},
				},
			}},
			{"EditSplits", ki.Props{
				"label":           "Edit...",
				"no-update-after": true,
			}},
		}},
	},
	"MainMenu": ki.PropSlice{
		{"AppMenu", ki.BlankProp{}},
		{"File", ki.PropSlice{
			{"OpenRecent", ki.Props{
				"submenu":         &SavedPaths,
				"no-update-after": true,
				"Args": ki.PropSlice{
					{"File Name", ki.Props{}},
				},
			}},
			{"NewProj", ki.Props{
				"shortcut":        "Command+N",
				"label":           "New Project...",
				"no-update-after": true,
				"Args": ki.PropSlice{
					{"Proj Dir", ki.Props{
						"dirs-only": true, // todo: support
					}},
				},
			}},
			{"OpenProj", ki.Props{
				"shortcut":        "Command+O",
				"label":           "Open Project...",
				"no-update-after": true,
				"Args": ki.PropSlice{
					{"File Name", ki.Props{
						"default-field": "ProjFilename",
						"ext":           ".gide",
					}},
				},
			}},
			{"SaveProj", ki.Props{
				// "shortcut": "Command+S",
				"label":           "Save Project",
				"no-update-after": true,
			}},
			{"SaveProjAs", ki.Props{
				// "shortcut": "Shift+Command+S",
				"label":           "Save Project As...",
				"no-update-after": true,
				"Args": ki.PropSlice{
					{"File Name", ki.Props{
						"default-field": "ProjFilename",
						"ext":           ".gide",
					}},
				},
			}},
			{"sep-af", ki.BlankProp{}},
			{"ViewFile", ki.Props{
				"label":           "Open File...",
				"no-update-after": true,
				// "shortcut": "Command+O",
				"Args": ki.PropSlice{
					{"File Name", ki.Props{}},
				},
			}},
			{"SaveActiveView", ki.Props{
				"label":           "Save File",
				"no-update-after": true,
				// "shortcut": "Command+S", // todo: need gide shortcuts
			}},
			{"SaveActiveViewAs", ki.Props{
				"label":           "Save File As...",
				"no-update-after": true,
				"Args": ki.PropSlice{
					{"File Name", ki.Props{
						"default-field": "ActiveFilename",
					}},
				},
			}},
			{"RevertActiveView", ki.Props{
				"desc":            "Revert active file to last saved version: this will lose all active changes -- are you sure?",
				"confirm":         true,
				"label":           "Revert File...",
				"no-update-after": true,
			}},
			{"sep-prefs", ki.BlankProp{}},
			{"ProjPrefs", ki.Props{
				"label": "Project Prefs...",
				// "shortcut": "Command+S",
			}},
			{"sep-close", ki.BlankProp{}},
			{"Close Window", ki.BlankProp{}},
		}},
		{"Edit", "Copy Cut Paste"},
		{"Window", "Windows"},
	},
	"CallMethods": ki.PropSlice{
		{"NextViewFile", ki.Props{
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"default-field": "ActiveFilename",
				}},
			},
		}},
		{"RegisterSave", ki.Props{
			"Args": ki.PropSlice{
				{"Register Name", ki.Props{
					"default-field": "Prefs.Register",
				}},
			},
		}},
		{"RegisterPaste", ki.Props{
			"Args": ki.PropSlice{
				{"Register Name", ki.Props{
					"default-field": "Prefs.Register",
				}},
			},
		}},
		{"SetSplit", ki.Props{
			"Args": ki.PropSlice{
				{"Split Name", ki.Props{}},
			},
		}},
	},
}

//////////////////////////////////////////////////////////////////////////////////////
//   Project window

func init() {
	gi.CustomAppMenuFunc = func(m *gi.Menu, win *gi.Window) {
		m.InsertActionAfter("GoGi Preferences...", gi.ActOpts{Label: "Gide Preferences..."},
			win, func(recv, send ki.Ki, sig int64, data interface{}) {
				PrefsView(&Prefs)
			})
	}
}

// NewGideProj creates a new Gide window with a new Gide project for given
// path, returning the window and the path
func NewGideProj(path string) (*gi.Window, *Gide) {
	_, projnm, _, _ := ProjPathParse(path)
	return NewGideWindow(path, projnm, true)
}

// OpenGideProj creates a new Gide window opened to given Gide project,
// returning the window and the path
func OpenGideProj(projfile string) (*gi.Window, *Gide) {
	pp := &ProjPrefs{}
	if err := pp.OpenJSON(gi.FileName(projfile)); err != nil {
		gi.PromptDialog(nil, gi.DlgOpts{Title: "Project File Could Not Be Opened", Prompt: fmt.Sprintf("Project file open encountered error: %v", err.Error())}, true, false, nil, nil)
		return nil, nil
	}
	path := string(pp.ProjRoot)
	_, projnm, _, _ := ProjPathParse(path)
	return NewGideWindow(projfile, projnm, false)
}

// NewGideWindow is common code for New / Open GideWindow
func NewGideWindow(path, projnm string, doNew bool) (*gi.Window, *Gide) {
	winm := "gide-" + projnm

	width := 1280
	height := 720

	win := gi.NewWindow2D(winm, winm, width, height, true) // true = pixel sizes

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()
	ge := mfr.AddNewChild(KiT_Gide, "gide").(*Gide)
	ge.Viewport = vp

	if doNew {
		ge.NewProj(gi.FileName(path))
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
				w.Close()
			} else {
				inClosePrompt = false
			}
		}
	})

	// win.OSWin.SetCloseCleanFunc(func(w oswin.Window) {
	// 	fmt.Printf("Doing final Close cleanup here..\n")
	// })

	win.OSWin.SetCloseCleanFunc(func(w oswin.Window) {
		if len(gi.MainWindows) <= 1 {
			go oswin.TheApp.Quit() // once main window is closed, quit
		}
	})

	win.MainMenuUpdated()

	vp.UpdateEndNoSig(updt)

	win.GoStartEventLoop()

	cmd := exec.Command("gocode", "close")
	defer cmd.Run()

	return win, ge
}

// CompleteGocode uses github.com/mdempsky/gocode to do code completion
func CompleteGo(data interface{}, text string, pos token.Position) (matches complete.Completions, seed string) {
	var txbuf *giv.TextBuf
	switch t := data.(type) {
	case *giv.TextView:
		txbuf = t.Buf
	}
	if txbuf == nil {
		log.Printf("complete.Complete: txbuf is nil - can't complete\n")
		return
	}

	seed = complete.SeedGolang(text)
	textbytes := make([]byte, 0, txbuf.NLines*40)
	for _, lr := range txbuf.Lines {
		textbytes = append(textbytes, []byte(string(lr))...)
		textbytes = append(textbytes, '\n')
	}
	results := complete.CompleteGo(textbytes, pos)
	matches = complete.MatchSeedCompletion(results, seed)
	return matches, seed
}

// CompleteGoEdit uses the selected completion to edit the text
func CompleteGoEdit(data interface{}, text string, cursorPos int, selection string, seed string) (s string, delta int) {
	s, delta = complete.EditGoCode(text, cursorPos, selection, seed)
	return s, delta
}

// CompleteTex does completion for tex files
func CompleteTex(data interface{}, text string, pos token.Position) (matches complete.Completions, seed string) {
	var txbuf *giv.TextBuf
	switch t := data.(type) {
	case *giv.TextView:
		txbuf = t.Buf
	}
	if txbuf == nil {
		log.Printf("complete.Complete: txbuf is nil - can't complete\n")
		return
	}

	seed = complete.SeedGolang(text)
	textbytes := make([]byte, 0, txbuf.NLines*40)
	for _, lr := range txbuf.Lines {
		textbytes = append(textbytes, []byte(string(lr))...)
		textbytes = append(textbytes, '\n')
	}
	results := complete.CompleteTex(textbytes, pos)
	matches = complete.MatchSeedCompletion(results, seed)
	return matches, seed
}

// CompleteTexEdit uses the selected completion to edit the text
func CompleteTexEdit(data interface{}, text string, cursorPos int, selection string, seed string) (s string, delta int) {
	s, delta = complete.EditTex(text, cursorPos, selection, seed)
	return s, delta
}
