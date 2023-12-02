// Copyright (c) 2023, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gidev

import (
	"fmt"
	"strings"

	"goki.dev/fi"
	"goki.dev/gi/v2/filetree"
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/giv"
	"goki.dev/gi/v2/texteditor"
	"goki.dev/gide/v2/gide"
	"goki.dev/girl/paint"
	"goki.dev/girl/styles"
	"goki.dev/goosi/events"
)

// RecycleCmdBuf creates the buffer for command output, or returns
// existing. If clear is true, then any existing buffer is cleared.
// Returns true if new buffer created.
func (ge *GideView) RecycleCmdBuf(cmdNm string, clear bool) (*texteditor.Buf, bool) {
	if ge.CmdBufs == nil {
		ge.CmdBufs = make(map[string]*texteditor.Buf, 20)
	}
	if buf, has := ge.CmdBufs[cmdNm]; has {
		if clear {
			buf.NewBuf(0)
		}
		return buf, false
	}
	buf := texteditor.NewBuf()
	buf.NewBuf(0)
	ge.CmdBufs[cmdNm] = buf
	buf.Autosave = false
	return buf, true
}

// RecycleCmdTab creates the tab to show command output, including making a
// buffer object to save output from the command. returns true if a new buffer
// was created, false if one already existed. if sel, select tab.  if clearBuf, then any
// existing buffer is cleared.  Also returns index of tab.
func (ge *GideView) RecycleCmdTab(cmdNm string, sel bool, clearBuf bool) (*texteditor.Buf, *texteditor.Editor, bool) {
	buf, nw := ge.RecycleCmdBuf(cmdNm, clearBuf)
	ctv := ge.RecycleTabTextEditor(cmdNm, sel)
	if ctv == nil {
		return nil, nil, false
	}
	ctv.SetReadOnly(true)
	ctv.SetBuf(buf)
	ctv.LinkHandler = func(tl *paint.TextLink) {
		ge.OpenFileURL(tl.URL, ctv)
	}
	return buf, ctv, nw
}

// TabDeleted is called when a main tab is deleted -- we cancel any running commmands
func (ge *GideView) TabDeleted(tabnm string) {
	ge.RunningCmds.KillByName(tabnm)
}

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
func (ge *GideView) ExecCmdNameFileNode(fn *filetree.Node, cmdNm gide.CmdName, sel bool, clearBuf bool) {
	cmd, _, ok := gide.AvailCmds.CmdByName(cmdNm, true)
	if !ok || fn == nil || fn.This() == nil {
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

// ExecCmds gets list of available commands for current active file
func ExecCmds(ge *GideView) [][]string {
	tv := ge.ActiveTextEditor()
	if tv == nil {
		return nil
	}
	var cmds [][]string

	vc := ge.VersCtrl()
	if ge.ActiveLang == fi.NoSupport {
		cmds = gide.AvailCmds.FilterCmdNames(ge.Prefs.MainLang, vc)
	} else {
		cmds = gide.AvailCmds.FilterCmdNames(ge.ActiveLang, vc)
	}
	return cmds
}

// ExecCmdNameActive calls given command on current active textview
func (ge *GideView) ExecCmdNameActive(cmdNm string) { //gti:add
	tv := ge.ActiveTextEditor()
	if tv == nil {
		return
	}
	ge.SaveAllCheck(true, func() { // true = cancel option
		ge.ExecCmdName(gide.CmdName(cmdNm), true, true)
	})
}

// CommandFromMenu pops up a menu of commands for given language, with given last command
// selected by default, and runs selected command.
func (ge *GideView) CommandFromMenu(fn *filetree.Node) {
	tv := ge.ActiveTextEditor()
	gi.NewMenu(gide.CommandMenu(fn), tv, tv.ContextMenuPos(nil)).Run()
}

// ExecCmd pops up a menu to select a command appropriate for the current
// active text view, and shows output in Tab with name of command
func (ge *GideView) ExecCmd() { //gti:add
	fn := ge.ActiveFileNode()
	if fn == nil {
		fmt.Printf("no Active File for ExecCmd\n")
		return
	}
	ge.CommandFromMenu(fn)
}

// ExecCmdFileNode pops up a menu to select a command appropriate for the given node,
// and shows output in Tab with name of command
func (ge *GideView) ExecCmdFileNode(fn *filetree.Node) {
	ge.CommandFromMenu(fn)
}

// SetArgVarVals sets the ArgVar values for commands, from GideView values
func (ge *GideView) SetArgVarVals() {
	tv := ge.ActiveTextEditor()
	tve := texteditor.AsEditor(tv)
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
func (ge *GideView) ExecCmdsFileNode(fn *filetree.Node, cmdNms gide.CmdNames, sel bool, clearBuf bool) {
	for _, cmdNm := range cmdNms {
		ge.ExecCmdNameFileNode(fn, cmdNm, sel, clearBuf)
	}
}

// Build runs the BuildCmds set for this project
func (ge *GideView) Build() { //gti:add
	if len(ge.Prefs.BuildCmds) == 0 {
		gi.NewBody().AddTitle("No BuildCmds Set").
			AddText("You need to set the BuildCmds in the Project Preferences").
			AddOkOnly().NewDialog(ge).Run()
		return
	}
	ge.SaveAllCheck(true, func() { // true = cancel option
		ge.ExecCmds(ge.Prefs.BuildCmds, true, true)
	})
}

// Run runs the RunCmds set for this project
func (ge *GideView) Run() { //gti:add
	if len(ge.Prefs.RunCmds) == 0 {
		gi.NewBody().AddTitle("No RunCmds Set").
			AddText("You need to set the RunCmds in the Project Preferences").
			AddOkOnly().NewDialog(ge).Run()
		return
	}
	if ge.Prefs.RunCmds[0] == "Run Proj" && !ge.Prefs.RunExecIsExec() {
		giv.CallFunc(ge, ge.ChooseRunExec)
		return
	}
	ge.ExecCmds(ge.Prefs.RunCmds, true, true)
}

// Commit commits the current changes using relevant VCS tool.
// Checks for VCS setting and for unsaved files.
func (ge *GideView) Commit() { //gti:add
	vc := ge.VersCtrl()
	if vc == "" {
		gi.NewBody().AddTitle("No Version Control System Found").
			AddText("No version control system detected in file system, or defined in project prefs -- define in project prefs if viewing a sub-directory within a larger repository").
			AddOkOnly().NewDialog(ge).Run()
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
		if !filetree.IsVersCtrlSystem(ct[0]) {
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
		gi.NewBody().AddTitle("No Commit command found").
			AddText("Could not find Commit command in list of avail commands -- this is usually a programmer error -- check preferences settings etc").
			AddOkOnly().NewDialog(ge).Run()
		return
	}
	ge.SetArgVarVals() // need to set before setting prompt string below..

	d := gi.NewBody().AddTitle("Commit message").
		AddText("Please enter your commit message here -- remember this is essential front-line documentation.  Author information comes from User settings in GoGi Preferences.")
	tf := gi.NewTextField(d).SetText("").SetPlaceholder("Enter commit message here..")
	tf.Style(func(s *styles.Style) {
		s.Min.X.Ch(200)
	})
	d.AddBottomBar(func(pw gi.Widget) {
		d.AddCancel(pw)
		d.AddOk(pw).SetText("Replace All").OnClick(func(e events.Event) {
			val := tf.Text()
			ge.ArgVals["{PromptString1}"] = val
			gide.CmdNoUserPrompt = true                     // don't re-prompt!
			ge.ExecCmdName(gide.CmdName(cmdnm), true, true) // must be wait
			ge.SaveProjIfExists(true)                       // saveall
			ge.UpdateFiles()
		})
	})
	d.NewDialog(ge).SetModal(false).Run()
}
