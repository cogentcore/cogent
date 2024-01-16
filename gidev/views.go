// Copyright (c) 2023, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gidev

import (
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"regexp"
	"time"

	"github.com/goki/gide/v2/gide"
	"github.com/goki/gide/v2/gidebug"
	"goki.dev/events"
	"goki.dev/fi"
	"goki.dev/filetree"
	"goki.dev/gi"
	"goki.dev/giv"
	"goki.dev/goosi"
	"goki.dev/spell"
	"goki.dev/vci"
)

// ConfigFindButton configures the Find FuncButton with current params
func (ge *GideView) ConfigFindButton(fb *giv.FuncButton) *giv.FuncButton {
	fb.Args[0].SetValue(ge.Prefs.Find.Find)
	fb.Args[0].SetTag("width", "80")
	fb.Args[1].SetValue(ge.Prefs.Find.Replace)
	fb.Args[1].SetTag("width", "80")
	fb.Args[2].SetValue(ge.Prefs.Find.IgnoreCase)
	fb.Args[3].SetValue(ge.Prefs.Find.Regexp)
	fb.Args[4].SetValue(ge.Prefs.Find.Loc)
	fb.Args[5].SetValue(ge.Prefs.Find.Langs)
	return fb
}

func (ge *GideView) CallFind(ctx gi.Widget) {
	ge.ConfigFindButton(giv.NewSoloFuncButton(ctx, ge.Find)).CallFunc()
}

// Find does Find / Replace in files, using given options and filters -- opens up a
// main tab with the results and further controls.
func (ge *GideView) Find(find string, repl string, ignoreCase bool, regExp bool, loc gide.FindLoc, langs []fi.Known) { //gti:add
	if find == "" {
		return
	}
	ge.Prefs.Find.IgnoreCase = ignoreCase
	ge.Prefs.Find.Regexp = regExp
	ge.Prefs.Find.Langs = langs
	ge.Prefs.Find.Loc = loc

	tv := ge.Tabs()
	if tv == nil {
		return
	}
	updt := tv.UpdateStart()
	defer tv.UpdateEndLayout(updt)

	fbuf, _ := ge.RecycleCmdBuf("Find", true)
	fv := tv.RecycleTabWidget("Find", true, gide.FindViewType).(*gide.FindView)
	fv.Time = time.Now()
	ftv := fv.TextEditor()
	ftv.SetBuf(fbuf)

	fv.SaveFindString(find)
	fv.SaveReplString(repl)
	fv.UpdateFromParams()
	fv.Update()
	root := filetree.AsNode(ge.Files)

	atv := ge.ActiveTextEditor()
	ond, _, got := ge.OpenNodeForTextEditor(atv)
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
	ge.FocusOnPanel(TabsIdx)
}

// Spell checks spelling in active text view
func (ge *GideView) Spell() { //gti:add
	txv := ge.ActiveTextEditor()
	if txv == nil || txv.Buf == nil {
		return
	}
	spell.OpenCheck() // make sure latest file opened
	tv := ge.Tabs()
	if tv == nil {
		return
	}
	updt := tv.UpdateStart()
	defer tv.UpdateEndLayout(updt)

	sv := tv.RecycleTabWidget("Spell", true, gide.SpellViewType).(*gide.SpellView)
	sv.ConfigSpellView(ge, txv)
	sv.Update()
	ge.FocusOnPanel(TabsIdx)
}

// Symbols displays the Symbols of a file or package
func (ge *GideView) Symbols() { //gti:add
	txv := ge.ActiveTextEditor()
	if txv == nil || txv.Buf == nil {
		return
	}
	tv := ge.Tabs()
	if tv == nil {
		return
	}
	updt := tv.UpdateStart()
	defer tv.UpdateEndLayout(updt)

	sv := tv.RecycleTabWidget("Symbols", true, gide.SymbolsViewType).(*gide.SymbolsView)
	sv.ConfigSymbolsView(ge, ge.ProjPrefs().Symbols)
	sv.Update()
	ge.FocusOnPanel(TabsIdx)
}

// Debug starts the debugger on the RunExec executable.
func (ge *GideView) Debug() { //gti:add
	tv := ge.Tabs()
	if tv == nil {
		return
	}
	updt := tv.UpdateStart()
	defer tv.UpdateEndLayout(updt)

	ge.Prefs.Debug.Mode = gidebug.Exec
	exePath := string(ge.Prefs.RunExec)
	exe := filepath.Base(exePath)
	dv := tv.RecycleTabWidget("Debug "+exe, true, gide.DebugViewType).(*gide.DebugView)
	dv.ConfigDebugView(ge, fi.Go, exePath)
	dv.Update()
	ge.FocusOnPanel(TabsIdx)
	ge.CurDbg = dv
}

// DebugTest runs the debugger using testing mode in current active textview path
func (ge *GideView) DebugTest() { //gti:add
	txv := ge.ActiveTextEditor()
	if txv == nil || txv.Buf == nil {
		return
	}
	tv := ge.Tabs()
	if tv == nil {
		return
	}
	updt := tv.UpdateStart()
	defer tv.UpdateEndLayout(updt)

	ge.Prefs.Debug.Mode = gidebug.Test
	tstPath := string(txv.Buf.Filename)
	dir := filepath.Base(filepath.Dir(tstPath))
	dv := tv.RecycleTabWidget("Debug "+dir, true, gide.DebugViewType).(*gide.DebugView)
	dv.ConfigDebugView(ge, fi.Go, tstPath)
	dv.Update()
	ge.FocusOnPanel(TabsIdx)
	ge.CurDbg = dv
}

// DebugAttach runs the debugger by attaching to an already-running process.
// pid is the process id to attach to.
func (ge *GideView) DebugAttach(pid uint64) { //gti:add
	tv := ge.Tabs()
	if tv == nil {
		return
	}
	updt := tv.UpdateStart()
	defer tv.UpdateEndLayout(updt)

	ge.Prefs.Debug.Mode = gidebug.Attach
	ge.Prefs.Debug.PID = pid
	exePath := string(ge.Prefs.RunExec)
	exe := filepath.Base(exePath)
	dv := tv.RecycleTabWidget("Debug "+exe, true, gide.DebugViewType).(*gide.DebugView)
	dv.ConfigDebugView(ge, fi.Go, exePath)
	dv.Update()
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

// VCSUpdateAll does an Update (e.g., Pull) on all VCS repositories within
// the open tree nodes in FileTree.
func (ge *GideView) VCSUpdateAll() { //gti:add
	ge.Files.UpdateAllVCS()
	ge.Files.UpdateAll()
}

// VCSLog shows the VCS log of commits for this file, optionally with a
// since date qualifier: If since is non-empty, it should be
// a date-like expression that the VCS will understand, such as
// 1/1/2020, yesterday, last year, etc.  SVN only understands a
// number as a maximum number of items to return.
// If allFiles is true, then the log will show revisions for all files, not just
// this one.
// Returns the Log and also shows it in a VCSLogView which supports further actions.
func (ge *GideView) VCSLog(since string) (vci.Log, error) { //gti:add
	atv := ge.ActiveTextEditor()
	ond, _, got := ge.OpenNodeForTextEditor(atv)
	if !got {
		if ge.Files.DirRepo != nil {
			return ge.Files.LogVCS(true, since)
		}
		gi.MessageDialog(atv, "No VCS Repository found in current active file or Root path: Open a file in a repository and try again", "No Version Control Repository")
		return nil, errors.New("No VCS Repository found in current active file or Root path")
	}
	return ond.LogVCS(true, since)
}

// OpenConsoleTab opens a main tab displaying console output (stdout, stderr)
func (ge *GideView) OpenConsoleTab() { //gti:add
	ctv := ge.RecycleTabTextEditor("Console", true)
	if ctv == nil {
		return
	}
	ctv.SetReadOnly(true)
	if ctv.Buf == nil || ctv.Buf != gide.TheConsole.Buf {
		ctv.SetBuf(gide.TheConsole.Buf)
		ctv.OnChange(func(e events.Event) {
			ge.SelectTabByLabel("Console")
		})
	}
}

// ChooseRunExec selects the executable to run for the project
func (ge *GideView) ChooseRunExec(exePath gi.Filename) { //gti:add
	if exePath != "" {
		ge.Prefs.RunExec = exePath
		ge.Prefs.BuildDir = gi.Filename(filepath.Dir(string(exePath)))
		if !ge.Prefs.RunExecIsExec() {
			gi.MessageDialog(ge, fmt.Sprintf("RunExec file: %v is not exectable", exePath), "Not Executable")
		}
	}
}

//////////////////////////////////////////////////////////////////////////////////////
//    StatusBar

// SetStatus sets the current status update message for the StatusBar next time it renders
func (ge *GideView) SetStatus(msg string) {
	ge.StatusMessage = msg
	ge.UpdateTextButtons()
}

// UpdateStatusLabel updates the statusbar label, called for each render!
func (ge *GideView) UpdateStatusLabel() {
	sb := ge.StatusBar()
	if sb == nil {
		return
	}
	lbl := ge.StatusLabel()
	fnm := ""
	ln := 0
	ch := 0
	tv := ge.ActiveTextEditor()
	msg := ""
	if tv != nil {
		ln = tv.CursorPos.Ln + 1
		ch = tv.CursorPos.Ch
		if tv.Buf != nil {
			fnm = ge.Files.RelPath(tv.Buf.Filename)
			if tv.Buf.IsNotSaved() {
				fnm += "*"
			}
			if tv.Buf.Info.Known != fi.Unknown {
				fnm += " (" + tv.Buf.Info.Known.String() + ")"
			}
		}
		if tv.ISearch.On {
			msg = fmt.Sprintf("\tISearch: %v (n=%v)\t%v", tv.ISearch.Find, len(tv.ISearch.Matches), ge.StatusMessage)
		}
		if tv.QReplace.On {
			msg = fmt.Sprintf("\tQReplace: %v -> %v (n=%v)\t%v", tv.QReplace.Find, tv.QReplace.Replace, len(tv.QReplace.Matches), ge.StatusMessage)
		}
	}

	str := fmt.Sprintf("%s\t%s\t<b>%s:</b>\t(%d,%d)\t%s", ge.Nm, ge.ActiveVCSInfo, fnm, ln, ch, msg)
	lbl.SetTextUpdate(str)
}

// HelpWiki opens wiki page for gide on github
func (ge *GideView) HelpWiki() { //gti:add
	goosi.TheApp.OpenURL("https://goki.dev/gide/")
}
