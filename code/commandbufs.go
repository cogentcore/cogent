// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"fmt"
	"strings"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/filetree"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/texteditor"
	"cogentcore.org/core/views"
)

// RecycleCmdBuf creates the buffer for command output, or returns
// existing. If clear is true, then any existing buffer is cleared.
// Returns true if new buffer created.
func (cv *CodeView) RecycleCmdBuf(cmdNm string, clear bool) (*texteditor.Buffer, bool) {
	if cv.CmdBufs == nil {
		cv.CmdBufs = make(map[string]*texteditor.Buffer, 20)
	}
	if buf, has := cv.CmdBufs[cmdNm]; has {
		if clear {
			buf.NewBuffer(0)
		}
		return buf, false
	}
	buf := texteditor.NewBuffer()
	buf.NewBuffer(0)
	cv.CmdBufs[cmdNm] = buf
	buf.Autosave = false
	// buf.Info.Known = fileinfo.Bash
	// buf.Info.Mime = fileinfo.MimeString(fileinfo.Bash)
	// buf.Hi.Lang = "Bash"
	return buf, true
}

// RecycleCmdTab creates the tab to show command output, including making a
// buffer object to save output from the command. returns true if a new buffer
// was created, false if one already existed. if sel, select tab.  if clearBuf, then any
// existing buffer is cleared.  Also returns index of tab.
func (cv *CodeView) RecycleCmdTab(cmdNm string, sel bool, clearBuf bool) (*texteditor.Buffer, *texteditor.Editor, bool) {
	buf, nw := cv.RecycleCmdBuf(cmdNm, clearBuf)
	ctv := cv.RecycleTabTextEditor(cmdNm, sel, buf)
	if ctv == nil {
		return nil, nil, false
	}
	ctv.SetReadOnly(true)
	ctv.SetBuffer(buf)
	ctv.LinkHandler = func(tl *paint.TextLink) {
		cv.OpenFileURL(tl.URL, ctv)
	}
	return buf, ctv, nw
}

// TabDeleted is called when a main tab is deleted -- we cancel any running commmands
func (cv *CodeView) TabDeleted(tabnm string) {
	cv.RunningCmds.KillByName(tabnm)
}

// ExecCmdName executes command of given name -- this is the final common
// pathway for all command invokation except on a node.  if sel, select tab.
// if clearBuf, clear the buffer prior to command
func (cv *CodeView) ExecCmdName(cmdNm CmdName, sel bool, clearBuf bool) {
	cmd, _, ok := AvailableCommands.CmdByName(cmdNm, true)
	if !ok {
		return
	}
	cv.SetArgVarVals()
	cbuf, _, _ := cv.RecycleCmdTab(cmd.Name, sel, clearBuf)
	cmd.Run(cv, cbuf)
}

// ExecCmdNameFileNode executes command of given name on given node
func (cv *CodeView) ExecCmdNameFileNode(fn *filetree.Node, cmdNm CmdName, sel bool, clearBuf bool) {
	cmd, _, ok := AvailableCommands.CmdByName(cmdNm, true)
	if !ok || fn == nil || fn.This() == nil {
		return
	}
	cv.ArgVals.Set(string(fn.FPath), &cv.Settings, nil)
	cbuf, _, _ := cv.RecycleCmdTab(cmd.Name, sel, clearBuf)
	cmd.Run(cv, cbuf)
}

// ExecCmdNameFilename executes command of given name on given file name
func (cv *CodeView) ExecCmdNameFilename(fn string, cmdNm CmdName, sel bool, clearBuf bool) {
	cmd, _, ok := AvailableCommands.CmdByName(cmdNm, true)
	if !ok {
		return
	}
	cv.ArgVals.Set(fn, &cv.Settings, nil)
	cbuf, _, _ := cv.RecycleCmdTab(cmd.Name, sel, clearBuf)
	cmd.Run(cv, cbuf)
}

// ExecCmds gets list of available commands for current active file
func ExecCmds(cv *CodeView) [][]string {
	tv := cv.ActiveTextEditor()
	if tv == nil {
		return nil
	}
	var cmds [][]string

	vc := cv.VersionControl()
	if cv.ActiveLang == fileinfo.Unknown {
		cmds = AvailableCommands.FilterCmdNames(cv.Settings.MainLang, vc)
	} else {
		cmds = AvailableCommands.FilterCmdNames(cv.ActiveLang, vc)
	}
	return cmds
}

// ExecCmdNameActive calls given command on current active texteditor
func (cv *CodeView) ExecCmdNameActive(cmdNm string) { //types:add
	tv := cv.ActiveTextEditor()
	if tv == nil {
		return
	}
	cv.SaveAllCheck(true, func() { // true = cancel option
		cv.ExecCmdName(CmdName(cmdNm), true, true)
	})
}

// CommandFromMenu pops up a menu of commands for given language, with given last command
// selected by default, and runs selected command.
func (cv *CodeView) CommandFromMenu(fn *filetree.Node) {
	tv := cv.ActiveTextEditor()
	core.NewMenu(CommandMenu(fn), tv, tv.ContextMenuPos(nil)).Run()
}

// ExecCmd pops up a menu to select a command appropriate for the current
// active text view, and shows output in Tab with name of command
func (cv *CodeView) ExecCmd() { //types:add
	fn := cv.ActiveFileNode()
	if fn == nil {
		fmt.Printf("no Active File for ExecCmd\n")
		return
	}
	cv.CommandFromMenu(fn)
}

// ExecCmdFileNode pops up a menu to select a command appropriate for the given node,
// and shows output in Tab with name of command
func (cv *CodeView) ExecCmdFileNode(fn *filetree.Node) {
	cv.CommandFromMenu(fn)
}

// SetArgVarVals sets the ArgVar values for commands, from CodeView values
func (cv *CodeView) SetArgVarVals() {
	tv := cv.ActiveTextEditor()
	tve := texteditor.AsEditor(tv)
	if tv == nil || tv.Buffer == nil {
		cv.ArgVals.Set("", &cv.Settings, tve)
	} else {
		cv.ArgVals.Set(string(tv.Buffer.Filename), &cv.Settings, tve)
	}
}

// ExecCmds executes a sequence of commands, sel = select tab, clearBuf = clear buffer
func (cv *CodeView) ExecCmds(cmdNms CmdNames, sel bool, clearBuf bool) {
	for _, cmdNm := range cmdNms {
		cv.ExecCmdName(cmdNm, sel, clearBuf)
	}
}

// ExecCmdsFileNode executes a sequence of commands on file node, sel = select tab, clearBuf = clear buffer
func (cv *CodeView) ExecCmdsFileNode(fn *filetree.Node, cmdNms CmdNames, sel bool, clearBuf bool) {
	for _, cmdNm := range cmdNms {
		cv.ExecCmdNameFileNode(fn, cmdNm, sel, clearBuf)
	}
}

// RunBuild runs the BuildCmds set for this project
func (cv *CodeView) RunBuild() { //types:add
	if len(cv.Settings.BuildCmds) == 0 {
		core.MessageDialog(cv, "You need to set the BuildCmds in the Project Settings", "No BuildCmds Set")
		return
	}
	cv.SaveAllCheck(true, func() { // true = cancel option
		cv.ExecCmds(cv.Settings.BuildCmds, true, true)
	})
}

// Run runs the RunCmds set for this project
func (cv *CodeView) Run() { //types:add
	if len(cv.Settings.RunCmds) == 0 {
		core.MessageDialog(cv, "You need to set the RunCmds in the Project Settings", "No RunCmds Set")
		return
	}
	if cv.Settings.RunCmds[0] == "Run Project" && !cv.Settings.RunExecIsExec() {
		views.CallFunc(cv, cv.ChooseRunExec)
		return
	}
	cv.ExecCmds(cv.Settings.RunCmds, true, true)
}

// Commit commits the current changes using relevant VCS tool.
// Checks for VCS setting and for unsaved files.
func (cv *CodeView) Commit() { //types:add
	vc := cv.VersionControl()
	if vc == "" {
		core.MessageDialog(cv, "No version control system detected in file system, or defined in project prefs -- define in project prefs if viewing a sub-directory within a larger repository", "No Version Control System Found")
		return
	}
	cv.SaveAllCheck(true, func() { // true = cancel option
		cv.CommitNoChecks()
	})
}

// CommitNoChecks does the commit without any further checks for VCS, and unsaved files
func (cv *CodeView) CommitNoChecks() {
	vc := cv.VersionControl()
	cmds := AvailableCommands.FilterCmdNames(cv.ActiveLang, vc)
	cmdnm := ""
	for _, ct := range cmds {
		if len(ct) < 2 {
			continue
		}
		if !filetree.IsVersionControlSystem(ct[0]) {
			continue
		}
		for _, cm := range ct {
			if strings.Contains(cm, "Commit") {
				cmdnm = CommandName(ct[0], cm)
				break
			}
		}
	}
	if cmdnm == "" {
		core.MessageDialog(cv, "Could not find Commit command in list of avail commands -- this is usually a programmer error -- check settings settings etc", "No Commit command found")
		return
	}
	cv.SetArgVarVals() // need to set before setting prompt string below..

	d := core.NewBody().AddTitle("Commit message").
		AddText("Please enter your commit message here. Remember that this is essential documentation. Author information comes from the Cogent Core User Settings.")
	tf := core.NewTextField(d)
	curval, _ := CmdPrompt1Vals["Commit"]
	tf.SetText(curval)
	tf.Styler(func(s *styles.Style) {
		s.Min.X.Ch(100)
	})
	d.AddBottomBar(func(parent core.Widget) {
		d.AddCancel(parent)
		d.AddOK(parent).SetText("Commit").OnClick(func(e events.Event) {
			val := tf.Text()
			cv.ArgVals["{PromptString1}"] = val
			CmdPrompt1Vals["Commit"] = val
			CmdNoUserPrompt = true                     // don't re-prompt!
			cv.ExecCmdName(CmdName(cmdnm), true, true) // must be wait
			cv.SaveProjectIfExists(true)               // saveall
			cv.UpdateFiles()
		})
	})
	d.RunDialog(cv)
}
