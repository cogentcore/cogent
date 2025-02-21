// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"fmt"
	"strings"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/vcs"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/filetree"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/textcore"
)

// RecycleCmdBuf creates the buffer for command output, or returns
// existing. Returns true if new buffer created.
func (cv *Code) RecycleCmdBuf(cmdName string) (*lines.Lines, bool) {
	if cv.CmdBufs == nil {
		cv.CmdBufs = make(map[string]*lines.Lines, 20)
	}
	if buf, has := cv.CmdBufs[cmdName]; has {
		buf.SetText(nil)
		return buf, false
	}
	buf := lines.NewLines()
	cv.CmdBufs[cmdName] = buf
	buf.Autosave = false
	buf.SetReadOnly(true)
	// note: critical to NOT set this, otherwise overwrites our native markup
	// buf.SetLanguage(fileinfo.Bash)
	return buf, true
}

// RecycleCmdTab creates the tab to show command output, including making a
// buffer object to save output from the command. Returns true if a new buffer
// was created, false if one already existed.
func (cv *Code) RecycleCmdTab(cmdName string) (*lines.Lines, *textcore.Editor, bool) {
	buf, nw := cv.RecycleCmdBuf(cmdName)
	ctv := cv.RecycleTabTextEditor(cmdName, buf)
	if ctv == nil {
		return nil, nil, false
	}
	ctv.SetReadOnly(true)
	ctv.SetLines(buf)
	ctv.LinkHandler = func(tl *rich.Hyperlink) {
		cv.OpenFileURL(tl.URL, ctv)
	}
	return buf, ctv, nw
}

// TabDeleted is called when a main tab is deleted -- we cancel any running commands
func (cv *Code) TabDeleted(tabnm string) {
	cv.RunningCmds.KillByName(tabnm)
}

// ExecCmdName executes command of given name; this is the final common
// pathway for all command invocation except on a node.
func (cv *Code) ExecCmdName(cmdName CmdName) {
	cmd, _, ok := AvailableCommands.CmdByName(cmdName, true)
	if !ok {
		return
	}
	cv.SetArgVarVals()
	cbuf, _, _ := cv.RecycleCmdTab(cmd.Name)
	cmd.Run(cv, cbuf)
}

// ExecCmdNameFileNode executes command of given name on given node
func (cv *Code) ExecCmdNameFileNode(fn *filetree.Node, cmdNm CmdName) {
	cmd, _, ok := AvailableCommands.CmdByName(cmdNm, true)
	if !ok || fn == nil || fn.This == nil {
		return
	}
	cv.ArgVals.Set(string(fn.Filepath), &cv.Settings, nil)
	cbuf, _, _ := cv.RecycleCmdTab(cmd.Name)
	cmd.Run(cv, cbuf)
}

// ExecCmdNameFilename executes command of given name on given file name
func (cv *Code) ExecCmdNameFilename(fn string, cmdNm CmdName) {
	cmd, _, ok := AvailableCommands.CmdByName(cmdNm, true)
	if !ok {
		return
	}
	cv.ArgVals.Set(fn, &cv.Settings, nil)
	cbuf, _, _ := cv.RecycleCmdTab(cmd.Name)
	cmd.Run(cv, cbuf)
}

// ExecCmds gets list of available commands for current active file
func ExecCmds(cv *Code) [][]string {
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
func (cv *Code) ExecCmdNameActive(cmdName string) { //types:add
	tv := cv.ActiveTextEditor()
	if tv == nil {
		return
	}
	cv.SaveAllCheck(true, func() { // true = cancel option
		cv.ExecCmdName(CmdName(cmdName))
	})
}

// CommandFromMenu pops up a menu of commands for given language, with given last command
// selected by default, and runs selected command.
func (cv *Code) CommandFromMenu(fn *filetree.Node) {
	tv := cv.ActiveTextEditor()
	core.NewMenu(CommandMenu(fn), tv, tv.ContextMenuPos(nil)).Run()
}

// ExecCmd pops up a menu to select a command appropriate for the current
// active text view, and shows output in Tab with name of command
func (cv *Code) ExecCmd() { //types:add
	fn := cv.ActiveFileNode()
	if fn == nil {
		fmt.Printf("no Active File for ExecCmd\n")
		return
	}
	cv.CommandFromMenu(fn)
}

// ExecCmdFileNode pops up a menu to select a command appropriate for the given node,
// and shows output in Tab with name of command
func (cv *Code) ExecCmdFileNode(fn *filetree.Node) {
	cv.CommandFromMenu(fn)
}

// SetArgVarVals sets the ArgVar values for commands, from Code values
func (cv *Code) SetArgVarVals() {
	tv := cv.ActiveTextEditor()
	tve := textcore.AsEditor(tv)
	if tv == nil || tv.Lines == nil {
		cv.ArgVals.Set("", &cv.Settings, tve)
	} else {
		cv.ArgVals.Set(tv.Lines.Filename(), &cv.Settings, tve)
	}
}

// ExecCmds executes a sequence of commands.
func (cv *Code) ExecCmds(cmdNms CmdNames) {
	for _, cmdNm := range cmdNms {
		cv.ExecCmdName(cmdNm)
	}
}

// ExecCmdsFileNode executes a sequence of commands on file node
func (cv *Code) ExecCmdsFileNode(fn *filetree.Node, cmdNames CmdNames) {
	for _, cmdNm := range cmdNames {
		cv.ExecCmdNameFileNode(fn, cmdNm)
	}
}

// RunBuild runs the BuildCmds set for this project
func (cv *Code) RunBuild() { //types:add
	if len(cv.Settings.BuildCmds) == 0 {
		core.MessageDialog(cv, "You need to set the BuildCmds in the Project Settings", "No BuildCmds Set")
		return
	}
	cv.SaveAllCheck(true, func() {
		cv.ExecCmds(cv.Settings.BuildCmds)
	})
}

// Run runs the RunCmds set for this project
func (cv *Code) Run() { //types:add
	if len(cv.Settings.RunCmds) == 0 {
		core.MessageDialog(cv, "You need to set the RunCmds in the Project Settings", "No RunCmds Set")
		return
	}
	if cv.Settings.RunCmds[0] == "Run Project" && !cv.Settings.RunExecIsExec() {
		core.CallFunc(cv, cv.ChooseRunExec)
		return
	}
	cv.ExecCmds(cv.Settings.RunCmds)
}

// Commit commits the current changes using relevant VCS tool.
// Checks for VCS setting and for unsaved files.
func (cv *Code) Commit() { //types:add
	vc := cv.VersionControl()
	if vc == vcs.NoVCS {
		core.MessageDialog(cv, "No version control system detected in file system, or defined in project prefs -- define in project prefs if viewing a sub-directory within a larger repository", "No Version Control System Found")
		return
	}
	cv.SaveAllCheck(true, func() { // true = cancel option
		cv.CommitNoChecks()
	})
}

// CommitNoChecks does the commit without any further checks for VCS, and unsaved files
func (cv *Code) CommitNoChecks() {
	vc := cv.VersionControl()
	cmds := AvailableCommands.FilterCmdNames(cv.ActiveLang, vc)
	cmdnm := ""
	for _, ct := range cmds {
		if len(ct) < 2 {
			continue
		}
		var vcstype vcs.Types
		if vcstype.SetString(ct[0]) != nil {
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

	d := core.NewBody("Commit message")
	core.NewText(d).SetType(core.TextSupporting).SetText("Please enter your commit message here. Remember that this is essential documentation. Author information comes from the Cogent Core User Settings.")
	tf := core.NewTextField(d)
	curval, _ := CmdPrompt1Vals["Commit"]
	tf.SetText(curval)
	tf.Styler(func(s *styles.Style) {
		s.Min.X.Ch(100)
	})
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).SetText("Commit").OnClick(func(e events.Event) {
			val := tf.Text()
			cv.ArgVals["{PromptString1}"] = val
			CmdPrompt1Vals["Commit"] = val
			CmdNoUserPrompt = true         // don't re-prompt!
			cv.ExecCmdName(CmdName(cmdnm)) // must be wait
			cv.SaveProjectIfExists(true)   // saveall
			cv.UpdateFiles()
		})
	})
	d.RunDialog(cv)
}
