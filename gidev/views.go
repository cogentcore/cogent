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

	"goki.dev/gi/v2/filetree"
	"goki.dev/gi/v2/gi"
	"goki.dev/gide/v2/gide"
	"goki.dev/gide/v2/gidebug"
	"goki.dev/goosi"
	"goki.dev/pi/v2/filecat"
	"goki.dev/pi/v2/spell"
	"goki.dev/vci/v2"
)

// Find does Find / Replace in files, using given options and filters -- opens up a
// main tab with the results and further controls.
func (ge *GideView) Find(find, repl string, ignoreCase, regExp bool, loc gide.FindLoc, langs []filecat.Supported) { //gti:add
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
	fv := tv.RecycleTabWidget("Find", true, gide.FindViewType).(*gide.FindView)
	fv.Time = time.Now()
	ftv := fv.TextView()
	ftv.SetReadOnly(true)
	ftv.SetBuf(fbuf)

	fv.SaveFindString(find)
	fv.SaveReplString(repl)

	root := filetree.AsNode(ge.Files)

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
func (ge *GideView) Spell() { //gti:add
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
	tv.RecycleTabWidget("Spell", true, gide.SpellViewType)
	tv.UpdateEndLayout(updt)
	ge.FocusOnPanel(TabsIdx)
}

// Symbols displays the Symbols of a file or package
func (ge *GideView) Symbols() { //gti:add
	txv := ge.ActiveTextView()
	if txv == nil || txv.Buf == nil {
		return
	}
	tv := ge.Tabs()
	if tv == nil {
		return
	}
	updt := tv.UpdateStart()
	tv.RecycleTabWidget("Symbols", true, gide.SymbolsViewType)
	tv.UpdateEndLayout(updt)
	ge.FocusOnPanel(TabsIdx)
}

// Debug starts the debugger on the RunExec executable.
func (ge *GideView) Debug() { //gti:add
	tv := ge.Tabs()
	if tv == nil {
		return
	}
	updt := tv.UpdateStart()
	ge.Prefs.Debug.Mode = gidebug.Exec
	exePath := string(ge.Prefs.RunExec)
	exe := filepath.Base(exePath)
	dv := tv.RecycleTabWidget("Debug "+exe, true, gide.DebugViewType).(*gide.DebugView)
	// dv.SetGide(ge, ge.Prefs.MainLang, exePath)
	tv.UpdateEndLayout(updt)
	ge.FocusOnPanel(TabsIdx)
	ge.CurDbg = dv
}

// DebugTest runs the debugger using testing mode in current active textview path
func (ge *GideView) DebugTest() { //gti:add
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
	dv := tv.RecycleTabWidget("Debug "+dir, true, gide.DebugViewType).(*gide.DebugView)
	// dv.SetGide(ge, ge.Prefs.MainLang, tstPath)
	tv.UpdateEndLayout(updt)
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
	dv := tv.RecycleTabWidget("Debug "+exe, true, gide.DebugViewType).(*gide.DebugView)
	// dv.SetGide(ge, ge.Prefs.MainLang, exePath)
	tv.UpdateEndLayout(updt)
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
func (ge *GideView) VCSUpdateAll() {
	ge.Files.UpdateAllVcs()
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
func (ge *GideView) VCSLog(since string) (vci.Log, error) {
	atv := ge.ActiveTextView()
	ond, _, got := ge.OpenNodeForTextView(atv)
	if !got {
		if ge.Files.DirRepo != nil {
			return ge.Files.LogVcs(true, since)
		}
		d := gi.NewBody().AddTitle("No Version Control Repository").
			AddText("No VCS Repository found in current active file or Root path: Open a file in a repository and try again")
		d.AddBottomBar(func(pw gi.Widget) {
			d.AddOk(pw)
		})
		d.NewDialog(ge).Run()
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
	ctv.SetReadOnly(true)
	if ctv.Buf == nil || ctv.Buf != gide.TheConsole.Buf {
		ctv.SetBuf(gide.TheConsole.Buf)
		// todo:
		// gide.TheConsole.Buf.TextBufSig.Connect(ge.This(), func(recv, send ki.Ki, sig int64, data any) {
		// 	ge.SelectTabByName("Console")
		// })
	}
}

// ChooseRunExec selects the executable to run for the project
func (ge *GideView) ChooseRunExec(exePath gi.FileName) { //gti:add
	if exePath != "" {
		ge.Prefs.RunExec = exePath
		ge.Prefs.BuildDir = gi.FileName(filepath.Dir(string(exePath)))
		if !ge.Prefs.RunExecIsExec() {
			d := gi.NewBody().AddTitle("Not Executable").
				AddText(fmt.Sprintf("RunExec file: %v is not exectable", exePath))
			d.AddBottomBar(func(pw gi.Widget) {
				d.AddOk(pw)
			})
			d.NewDialog(ge).Run()
		}
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

// HelpWiki opens wiki page for gide on github
func (ge *GideView) HelpWiki() {
	goosi.TheApp.OpenURL("https://goki.dev/gide/v2/wiki")
}
