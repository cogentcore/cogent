// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
package gide provides the core Gide editor object.

Derived classes can extend the functionality for specific domains.

*/
package gide

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"github.com/goki/gi"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/key"
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
	ProjRoot          gi.FileName  `desc:"root directory for the project -- all projects must be organized within a top-level root directory, with all the files therein constituting the scope of the project -- by default it is the path for ProjFilename"`
	ProjFilename      gi.FileName  `ext:".gide" desc:"current project filename for saving / loading specific Gide configuration information in a .gide file (optional)"`
	ActiveFilename    gi.FileName  `desc:"filename of the currently-active textview"`
	Changed           bool         `json:"-" desc:"has the root changed?  we receive update signals from root for changes"`
	Files             giv.FileTree `desc:"all the files in the project directory and subdirectories"`
	ActiveTextViewIdx int          `json:"-" desc:"index of the currently-active textview -- new files will be viewed in other views if available"`
	Prefs             ProjPrefs    `desc:"preferences for this project -- this is what is saved in a .gide project file"`
	KeySeq1           key.Chord    `desc:"first key in sequence if needs2 key pressed"`
	UpdtMu            sync.Mutex   `desc:"mutex for protecting overall updates to Gide"`
}

var KiT_Gide = kit.Types.AddType(&Gide{}, GideProps)

// UpdateFiles updates the list of files saved in project
func (ge *Gide) UpdateFiles() {
	ge.Files.OpenPath(string(ge.ProjRoot))
}

// IsEmpty returns true if given Gide project is empty -- has not been set to a valid path
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
		ge.GuessMainLang()
		ge.UpdateProj()
		win := ge.ParentWindow()
		if win != nil {
			winm := "gide-" + pnm
			win.SetName(winm)
			win.SetTitle(winm)
		}
		if fnm != "" {
			ge.ViewFile(gi.FileName(fnm))
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
	ge.UpdateSig()
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
	ge.UpdateFiles()
	ge.ConfigSplitView()
	ge.ConfigToolbar()
	ge.ConfigStatusBar()
	ge.SetStatus("just updated")
	if mods {
		ge.UpdateEnd(updt)
	}
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

// SetActiveTextView sets the given textview as the active one, and returns its index
func (ge *Gide) SetActiveTextView(av *giv.TextView) int {
	idx := ge.TextViewIndex(av)
	if idx < 0 {
		return -1
	}
	ge.ActiveTextViewIdx = idx
	if av.Buf != nil {
		ge.ActiveFilename = av.Buf.Filename
	}
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
		ge.ActiveFilename = av.Buf.Filename
	}
	av.GrabFocus()
	return av
}

// NextTextView returns the next text view available for viewing a file and
// its index -- if the active text view is empty, then it is used, otherwise
// it is the next one
func (ge *Gide) NextTextView() (*giv.TextView, int) {
	av := ge.TextViewByIndex(ge.ActiveTextViewIdx)
	if av.Buf == nil {
		return av, ge.ActiveTextViewIdx
	}
	nxt := (ge.ActiveTextViewIdx + 1) % NTextViews
	return ge.TextViewByIndex(nxt), nxt
}

// SaveActiveView saves the contents of the currently-active textview
func (ge *Gide) SaveActiveView() {
	tv := ge.ActiveTextView()
	if tv.Buf != nil {
		if tv.Buf.Filename != "" {
			tv.Buf.Save()
			ge.ActiveViewRunPostCmds()
		} else {
			giv.CallMethod(ge, "SaveActiveViewAs", ge.Viewport) // uses fileview
		}
	}
}

// SaveActiveViewAs save with specified filename the contents of the
// currently-active textview
func (ge *Gide) SaveActiveViewAs(filename gi.FileName) {
	tv := ge.ActiveTextView()
	if tv.Buf != nil {
		tv.Buf.SaveAs(filename)
		ge.Files.UpdateNewFile(filename)
		ge.ActiveViewRunPostCmds()
	}
}

// ActiveViewRunPostCmds runs any registered post commands on the active view
// -- returns true if commands were run and file was reverted after that --
// uses MainLang to disambiguate if multiple languages associated with extension.
func (ge *Gide) ActiveViewRunPostCmds() bool {
	tv := ge.ActiveTextView()
	ran := false
	if tv.Buf == nil || tv.Buf.Filename == "" {
		return false
	}

	ls := LangsForFilename(string(tv.Buf.Filename))
	if len(ls) == 1 {
		lr := ls[0]
		if len(lr.PostSaveCmds) > 0 {
			ge.ExecCmds(lr.PostSaveCmds)
			ran = true
		}
	} else if len(ls) > 1 {
		hasPosts := false
		for _, lr := range ls {
			if len(lr.PostSaveCmds) > 0 {
				hasPosts = true
				if lr.Name == string(ge.Prefs.MainLang) {
					ge.ExecCmds(lr.PostSaveCmds)
					ran = true
					break
				}
			}
		}
		if hasPosts && !ran {
			ge.SetStatus("File has multiple associated languages and none match main language of project, cannot run any post commands")
		}
	}
	if ran {
		tv.Buf.ReOpen()
		return true
	}
	return false
}

// ViewFileNode sets the next text view to view file in given node (opens
// buffer if not already opened)
func (ge *Gide) ViewFileNode(fn *FileNode) {
	if err := fn.OpenBuf(); err == nil {
		nv, nidx := ge.NextTextView()
		if nv.Buf != nil && nv.Buf.Edited { // todo: save current changes?
			fmt.Printf("Changes not saved in file: %v before switching view there to new file\n", nv.Buf.Filename)
		}
		nv.SetBuf(fn.Buf)
		ge.SetActiveTextViewIdx(nidx)
	}
}

// ViewFile sets the next text view to view given file name -- include as much
// of name as possible to disambiguate -- will use the first matching --
// returns false if not found
func (ge *Gide) ViewFile(fnm gi.FileName) bool {
	fn, ok := ge.Files.FindFile(string(fnm))
	if !ok {
		return false
	}
	ge.ViewFileNode(fn.This.(*FileNode))
	return true
}

// SelectBuf selects an open buffer to view in current active textview
func (ge *Gide) SelectBuf() {
	// todo: simple quick popup menu selector of all open buffers -- need a separate list of those!
}

// TextViewSig handles all signals from the textviews
func (ge *Gide) TextViewSig(tv *giv.TextView, sig giv.TextViewSignals) {
	ge.SetActiveTextView(tv) // if we're sending signals, we're the active one!
	switch sig {
	case giv.TextViewCursorMoved:
		ge.SetStatus("")
	}
}

//////////////////////////////////////////////////////////////////////////////////////
//   Panels

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
	if cp == TextView1Idx {
		ge.SetActiveTextViewIdx(0)
	} else if cp == TextView2Idx {
		ge.SetActiveTextViewIdx(1)
	} else {
		ski := sv.Kids[cp]
		win := ge.ParentWindow()
		win.FocusNext(ski)
	}
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
	ski := sv.Kids[cp]
	win := ge.ParentWindow()
	win.FocusNext(ski)
}

//////////////////////////////////////////////////////////////////////////////////////
//    Tabs

// MainTabByName returns a MainTabs (first set of tabs) tab with given name,
// and its index -- returns false if not found
func (ge *Gide) MainTabByName(label string) (gi.Node2D, int, bool) {
	tv := ge.MainTabs()
	return tv.TabByName(label)
}

// FindOrMakeMainTab returns a MainTabs (first set of tabs) tab with given
// name, first by looking for an existing one, and if not found, making a new
// one with widget of given type.  returns widget and tab index.
func (ge *Gide) FindOrMakeMainTab(label string, typ reflect.Type) (gi.Node2D, int) {
	tv := ge.MainTabs()
	widg, idx, ok := ge.MainTabByName(label)
	if ok {
		tv.SelectTabIndex(idx)
		return widg, idx
	}
	widg, idx = tv.AddNewTab(typ, label)
	tv.SelectTabIndex(idx)
	return widg, idx
}

// FindOrMakeMainTabTextView returns a MainTabs (first set of tabs) tab with given
// name, first by looking for an existing one, and if not found, making a new
// one with a TextView in it.  returns widget and tab index.
func (ge *Gide) FindOrMakeMainTabTextView(label string) (*giv.TextView, int) {
	tvk, idx := ge.FindOrMakeMainTab(label, giv.KiT_TextView)
	tv := tvk.Embed(giv.KiT_TextView).(*giv.TextView)
	tv.SetProp("word-wrap", ge.Prefs.Editor.WordWrap)
	tv.SetProp("tab-size", 8) // std for output
	tv.SetProp("font-family", ge.Prefs.Editor.FontFamily)
	return tv, idx
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

// ExecCmd executes given command and shows output in MainTab with name of command
func (ge *Gide) ExecCmd(cmdNm CmdName) {
	cmd, _, ok := AvailCmds.CmdByName(cmdNm)
	if !ok {
		return
	}
	av := ge.ActiveTextView()
	if av == nil { // todo: could just issue warning
		return
	}
	cmd.MakeBuf(true)
	tv, _ := ge.FindOrMakeMainTabTextView(cmd.Name)
	tv.SetBuf(cmd.Buf)
	SetArgVarVals(&ArgVarVals, string(av.Buf.Filename), string(ge.ProjRoot), av)
	cmd.Run(ge)
}

// ExecCmds executes a sequence of commands
func (ge *Gide) ExecCmds(cmdNms CmdNames) {
	for _, cmdNm := range cmdNms {
		ge.ExecCmd(cmdNm)
	}
}

// ExecCmdFileNode executes given command on given file node
func (ge *Gide) ExecCmdFileNode(cmdNm CmdName, fn *FileNode) {
	cmd, _, ok := AvailCmds.CmdByName(cmdNm)
	if !ok {
		return
	}
	tv, _ := ge.FindOrMakeMainTabTextView(cmd.Name)
	cmd.MakeBuf(true)
	tv.SetBuf(cmd.Buf)
	SetArgVarVals(&ArgVarVals, string(fn.FPath), string(ge.ProjRoot), nil)
	cmd.Run(ge)
}

//////////////////////////////////////////////////////////////////////////////////////
//    StatusBar

// SetStatus updates the statusbar label with given message, along with other status info
func (ge *Gide) SetStatus(msg string) {
	sb := ge.StatusBar()
	if sb == nil {
		return
	}
	ge.UpdtMu.Lock()
	defer ge.UpdtMu.Unlock()

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
			if tv.Buf.Edited {
				fnm += "*"
			}
		}
	}

	str := fmt.Sprintf("%v   <b>%v:</b>   (%v,%v)    %v", ge.Nm, fnm, ln, ch, msg)
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
func (ge *Gide) MainTabs() *giv.TabView {
	split := ge.SplitView()
	if split != nil {
		tv := split.KnownChild(MainTabsIdx).Embed(giv.KiT_TabView).(*giv.TabView)
		return tv
	}
	return nil
}

// VisTabs returns the second, visualization TabView
func (ge *Gide) VisTabs() *giv.TabView {
	split := ge.SplitView()
	if split != nil {
		tv := split.KnownChild(VisTabsIdx).Embed(giv.KiT_TabView).(*giv.TabView)
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
	config.Add(giv.KiT_TabView, "main-tabs")
	config.Add(giv.KiT_TabView, "vis-tabs")
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
				fn := tvn.SrcNode.Ptr.Embed(KiT_FileNode).(*FileNode)
				switch sig {
				case int64(giv.TreeViewSelected):
					gee.FileNodeSelected(fn, tvn)
				case int64(giv.TreeViewOpened):
					gee.FileNodeOpened(fn, tvn)
				case int64(giv.TreeViewClosed):
					gee.FileNodeClosed(fn, tvn)
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

		// tabs := split.KnownChild(len(*split.Children()) - 1).(*giv.TabView)
		// if !tabs.HasChildren() {
		// 	lbl1 := tabs.AddNewTab(gi.KiT_Label, "Label1").(*gi.Label)
		// 	lbl1.SetText("this is the contents of the first tab")
		// 	lbl1.SetProp("word-wrap", true)

		// 	lbl2 := tabs.AddNewTab(gi.KiT_Label, "Label2").(*gi.Label)
		// 	lbl2.SetText("this is the contents of the second tab")
		// 	lbl2.SetProp("word-wrap", true)
		// 	tabs.SelectTabIndex(0)
		// }
		split.SetSplits(ge.Prefs.Splits...)
		split.UpdateEnd(updt)
	}
	for i := 0; i < NTextViews; i++ {
		txly := split.KnownChild(1 + i).(*gi.Layout)
		txed := txly.KnownChild(0).(*giv.TextView)
		txed.HiStyle = ge.Prefs.Editor.HiStyle
		txed.Opts.LineNos = ge.Prefs.Editor.LineNos
		txed.Opts.AutoIndent = true
		txed.SetProp("word-wrap", ge.Prefs.Editor.WordWrap)
		txed.SetProp("tab-size", ge.Prefs.Editor.TabSize)
		txed.SetProp("font-family", ge.Prefs.Editor.FontFamily)
	}

	// set some properties always, even if no mods
	split.SetSplits(ge.Prefs.Splits...)
}

func (ge *Gide) FileNodeSelected(fn *FileNode, tvn *giv.FileTreeView) {
	// if fn.IsDir() {
	// } else {
	// }
}

func (ge *Gide) FileNodeOpened(fn *FileNode, tvn *giv.FileTreeView) {
	if fn.IsDir() {
		if !fn.IsOpen() {
			tvn.SetOpen()
			fn.OpenDir()
		}
	} else {
		ge.ViewFileNode(fn.This.(*FileNode))
	}
}

func (ge *Gide) FileNodeClosed(fn *FileNode, tvn *giv.FileTreeView) {
	if fn.IsDir() {
		if fn.IsOpen() {
			fn.CloseDir()
		}
	}
}

func (ge *Gide) GideKeys(kt *key.ChordEvent) {
	kf := KeyFunNil
	kc := kt.Chord()
	if ge.KeySeq1 != "" {
		kf = KeyFun(ge.KeySeq1, kc)
		if kf == KeyFunNil && kc == "Escape" {
			ge.SetStatus(string(ge.KeySeq1) + " " + string(kc) + " -- aborted")
			kt.SetProcessed() // abort key sequence, don't send esc to anyone else
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
		ge.SelectBuf()
	case KeyFunBufSave:
		kt.SetProcessed()
		ge.SaveActiveView()
	case KeyFunExecCmd:
		kt.SetProcessed()
		giv.CallMethod(ge, "ExecCmd", ge.Viewport)
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
			"shortcut": "Command+U",
			"icon":     "update",
		}},
		{"SaveActiveView", ki.Props{
			"label": "Save",
			"icon":  "file-save",
		}},
		{"SaveActiveViewAs", ki.Props{
			"label": "Save As...",
			"icon":  "file-save",
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"default-field": "ActiveFilename",
				}},
			},
		}},
		{"ExecCmd", ki.Props{
			"Args": ki.PropSlice{
				{"Command", ki.Props{}},
			},
		}},
	},
	"MainMenu": ki.PropSlice{
		{"AppMenu", ki.BlankProp{}},
		{"File", ki.PropSlice{
			{"OpenRecent", ki.Props{
				"submenu": &SavedPaths,
				"Args": ki.PropSlice{
					{"File Name", ki.Props{}},
				},
			}},
			{"NewProj", ki.Props{
				"shortcut":        "Command+N",
				"no-update-after": true,
				"Args": ki.PropSlice{
					{"Proj Dir", ki.Props{
						"dirs-only": true, // todo: support
					}},
				},
			}},
			{"OpenProj", ki.Props{
				"shortcut": "Command+O",
				"Args": ki.PropSlice{
					{"File Name", ki.Props{
						"default-field": "ProjFilename",
						"ext":           ".gide",
					}},
				},
			}},
			{"SaveProj", ki.Props{
				// "shortcut": "Command+S",
			}},
			{"SaveProjAs", ki.Props{
				// "shortcut": "Shift+Command+S",
				"label": "Save Proj As...",
				"Args": ki.PropSlice{
					{"File Name", ki.Props{
						"default-field": "ProjFilename",
						"ext":           ".gide",
					}},
				},
			}},
			{"sep-prefs", ki.BlankProp{}},
			{"ProjPrefs", ki.Props{
				// "shortcut": "Command+S",
			}},
			{"sep-close", ki.BlankProp{}},
			{"Close Window", ki.BlankProp{}},
		}},
		{"Edit", "Copy Cut Paste"},
		{"Window", "Windows"},
	},
	"CallMethods": ki.PropSlice{
		{"ViewFile", ki.Props{
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"default-field": "ActiveFilename",
				}},
			},
		}},
	},
}

//////////////////////////////////////////////////////////////////////////////////////
//   Project window

func init() {
	gi.CustomAppMenuFunc = func(m *gi.Menu, win *gi.Window) {
		m.InsertActionAfter("GoGi Preferences", gi.ActOpts{Label: "Gide Preferences"},
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
			if ge.Changed {
				gi.ChoiceDialog(vp, gi.DlgOpts{Title: "Close Without Saving?",
					Prompt: "Do you want to save your changes?  If so, Cancel and then Save"},
					[]string{"Close Without Saving", "Cancel"},
					win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
						switch sig {
						case 0:
							w.Close()
						case 1:
							inClosePrompt = false
							// default is to do nothing, i.e., cancel
						}
					})
			} else {
				w.Close()
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
	return win, ge
}
