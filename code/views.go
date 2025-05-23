// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"errors"
	"fmt"
	"path/filepath"

	"cogentcore.org/cogent/code/cdebug"
	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/base/vcs"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/text/spell"
)

// ConfigFindButton configures the Find FuncButton with current params
func (cv *Code) ConfigFindButton(fb *core.FuncButton) *core.FuncButton {
	fb.Args[0].SetValue(cv.Settings.Find.Find)
	fb.Args[0].SetTag(`width:"80"`)
	fb.Args[1].SetValue(cv.Settings.Find.Replace)
	fb.Args[1].SetTag(`width:"80"`)
	fb.Args[2].SetValue(cv.Settings.Find.IgnoreCase)
	fb.Args[3].SetValue(cv.Settings.Find.Regexp)
	fb.Args[4].SetValue(cv.Settings.Find.Loc)
	fb.Args[5].SetValue(cv.Settings.Find.Languages)
	return fb
}

// Spell checks spelling in active text view
func (cv *Code) Spell() { //types:add
	txv := cv.ActiveEditor()
	if txv == nil || txv.Lines == nil {
		return
	}
	spell.Spell.OpenUserCheck() // make sure latest file opened
	tv := cv.Tabs()
	if tv == nil {
		return
	}

	sv := core.RecycleTabWidget[SpellPanel](tv, "Spell")
	sv.Config(cv, txv)
	sv.Update()
	cv.FocusOnPanel(TabsIndex)
}

// Symbols displays the Symbols of a file or package
func (cv *Code) Symbols() { //types:add
	txv := cv.ActiveEditor()
	if txv == nil || txv.Lines == nil {
		return
	}
	tv := cv.Tabs()
	if tv == nil {
		return
	}

	sv := core.RecycleTabWidget[SymbolsPanel](tv, "Symbols")
	sv.Config(cv, cv.Settings.Symbols)
	sv.Update()
	cv.FocusOnPanel(TabsIndex)
}

// Debug starts the debugger on the RunExec executable.
func (cv *Code) Debug() { //types:add
	tv := cv.Tabs()
	if tv == nil {
		return
	}

	cv.Settings.Debug.Mode = cdebug.Exec
	exePath := string(cv.Settings.RunExec)
	exe := filepath.Base(exePath)
	dv := core.RecycleTabWidget[DebugPanel](tv, "Debug "+exe)
	dv.Config(cv, fileinfo.Go, exePath)
	cv.FocusOnPanel(TabsIndex)
	dv.Update()
	dv.Start()
	cv.CurDbg = dv
}

// DebugTest runs the debugger using testing mode in current active texteditor path.
// testName specifies which test(s) to run according to the standard go test -run
// specification.
func (cv *Code) DebugTest(testName string) { //types:add
	txv := cv.ActiveEditor()
	if txv == nil || txv.Lines == nil {
		return
	}
	tv := cv.Tabs()
	if tv == nil {
		return
	}

	cv.Settings.Debug.Mode = cdebug.Test
	cv.Settings.Debug.TestName = testName
	tstPath := txv.Lines.Filename()
	dir := filepath.Base(filepath.Dir(tstPath))
	dv := core.RecycleTabWidget[DebugPanel](tv, "Debug "+dir)
	dv.Config(cv, fileinfo.Go, tstPath)
	cv.FocusOnPanel(TabsIndex)
	dv.Update()
	dv.Start()
	cv.CurDbg = dv
}

// DebugAttach runs the debugger by attaching to an already-running process.
// pid is the process id to attach to.
func (cv *Code) DebugAttach(pid uint64) { //types:add
	tv := cv.Tabs()
	if tv == nil {
		return
	}

	cv.Settings.Debug.Mode = cdebug.Attach
	cv.Settings.Debug.PID = pid
	exePath := string(cv.Settings.RunExec)
	exe := filepath.Base(exePath)
	dv := core.RecycleTabWidget[DebugPanel](tv, "Debug "+exe)
	dv.Config(cv, fileinfo.Go, exePath)
	cv.FocusOnPanel(TabsIndex)
	dv.Update()
	dv.Start()
	cv.CurDbg = dv
}

// CurDebug returns the current debug view
func (cv *Code) CurDebug() *DebugPanel {
	return cv.CurDbg
}

// ClearDebug clears the current debugger setting -- no more debugger active.
func (cv *Code) ClearDebug() {
	cv.CurDbg = nil
}

// VCSUpdateAll does an Update (e.g., Pull) on all VCS repositories within
// the open tree nodes in FileTree.
func (cv *Code) VCSUpdateAll() { //types:add
	cv.Files.UpdateAllVCS()
	cv.Files.Update()
}

// VCSLog shows the VCS log of commits in this project,
// in an interactive browser from which any revisions can be
// compared and diffs browsed.
func (cv *Code) VCSLog() (vcs.Log, error) { //types:add
	since := ""
	atv := cv.ActiveEditor()
	ond := cv.FileNodeForFile(atv.Lines.Filename())
	if ond == nil {
		if cv.Files.DirRepo != nil {
			return cv.Files.LogVCS(true, since)
		}
		core.MessageDialog(atv, "No VCS Repository found in current active file or Root path: Open a file in a repository and try again", "No Version Control Repository")
		return nil, errors.New("No VCS Repository found in current active file or Root path")
	}
	return ond.LogVCS(true, since)
}

// OpenConsoleTab opens a main tab displaying console output (stdout, stderr)
func (cv *Code) OpenConsoleTab() { //types:add
	ctv := cv.RecycleTabTextEditor("Console", nil)
	if ctv == nil {
		return
	}
	ctv.SetReadOnly(true)
	if ctv.Lines == nil || ctv.Lines != TheConsole.Lines {
		ctv.SetLines(TheConsole.Lines)
		ctv.OnChange(func(e events.Event) {
			cv.SelectTabByName("Console")
		})
	}
}

// updatePreviewPanel updates the [PreviewPanel], making it if it doesn't exist yet.
func (cv *Code) updatePreviewPanel() {
	ts := cv.Tabs()
	_, ptab := ts.CurrentTab()
	pp := core.RecycleTabWidget[PreviewPanel](ts, "Preview")
	curIsPreview := false
	if ptab >= 0 {
		_, ctab := ts.CurrentTab()
		curIsPreview = ctab == ptab
		ts.SelectTabIndex(ptab) // we stay at the previous tab
	} else {
		curIsPreview = true
	}
	pp.code = cv
	if !curIsPreview { // not visible don't update
		return
	}
	pp.Update()
}

// ChooseRunExec selects the executable to run for the project
func (cv *Code) ChooseRunExec(exePath core.Filename) { //types:add
	if exePath != "" {
		cv.Settings.RunExec = exePath
		cv.Settings.BuildDir = core.Filename(filepath.Dir(string(exePath)))
		if !cv.Settings.RunExecIsExec() {
			core.MessageDialog(cv, fmt.Sprintf("RunExec file: %v is not exectable", exePath), "Not Executable")
		}
	}
}

//////////////////////////////////////////////////////////////////////////////////////
//    StatusBar

// SetStatus sets the current status update message for the StatusBar next time it renders
func (cv *Code) SetStatus(msg string) {
	cv.StatusMessage = msg
	cv.UpdateStatusText()
	cv.UpdateTextButtons()
}

// UpdateStatusText updates the status bar text with current data.
func (cv *Code) UpdateStatusText() {
	sb := cv.StatusBar()
	if sb == nil {
		return
	}
	text := cv.StatusText()
	fnm := ""
	ln := 0
	ch := 0
	tv := cv.ActiveEditor()
	msg := ""
	if tv != nil {
		ln = tv.CursorPos.Line + 1
		ch = tv.CursorPos.Char
		if tv.Lines != nil {
			fnm = cv.Files.RelativePathFrom(fsx.Filename(tv.Lines.Filename()))
			if tv.Lines.IsNotSaved() {
				fnm += "*"
			}
			if tv.Lines.FileInfo().Known != fileinfo.Unknown {
				fnm += " (" + tv.Lines.FileInfo().Known.String() + ")"
			}
		}
		if tv.ISearch.On {
			msg = fmt.Sprintf("\tISearch: %v (n=%v)", tv.ISearch.Find, len(tv.ISearch.Matches))
		}
		if tv.QReplace.On {
			msg = fmt.Sprintf("\tQReplace: %v -> %v (n=%v)", tv.QReplace.Find, tv.QReplace.Replace, len(tv.QReplace.Matches))
		}
	}

	str := fmt.Sprintf("%s\t%s\t<b>%s:</b>\t(%d,%d)\t%s", cv.Name, cv.ActiveVCSInfo, fnm, ln, ch, msg)
	text.SetText(str).UpdateRender()
}

// HelpWiki opens wiki page for code on github
func (cv *Code) HelpWiki() { //types:add
	core.TheApp.OpenURL("https://cogentcore.org/code")
}
