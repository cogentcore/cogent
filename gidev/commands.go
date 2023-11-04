// Copyright (c) 2023, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gidev

import (
	"fmt"
	"strings"

	"goki.dev/gi/v2/filetree"
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/giv"
	"goki.dev/gi/v2/texteditor"
	"goki.dev/gide/v2/gide"
	"goki.dev/goosi/events"
	"goki.dev/pi/v2/filecat"
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
	ctv := ge.RecycleTabTextView(cmdNm, sel)
	if ctv == nil {
		return nil, nil, false
	}
	ctv.SetReadOnly(true)
	ctv.SetBuf(buf)
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

// ExecCmds gets list of available commands for current active file
func ExecCmds(ge *GideView) [][]string {
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
func (ge *GideView) ExecCmdNameActive(cmdNm string) { //gti:add
	tv := ge.ActiveTextView()
	if tv == nil {
		return
	}
	ge.SaveAllCheck(true, func() { // true = cancel option
		ge.ExecCmdName(gide.CmdName(cmdNm), true, true)
	})
}

// CommandFromMenu pops up a menu of commands for given language, with given last command
// selected by default, and runs selected command.
func (ge *GideView) CommandFromMenu(lang filecat.Supported) {
	lastCmd := ""
	hsz := len(ge.CmdHistory)
	if hsz > 0 {
		lastCmd = string(ge.CmdHistory[hsz-1])
	}
	mm := gi.NewScene()
	cmds := gide.AvailCmds.FilterCmdNames(lang, ge.VersCtrl())
	for _, cc := range cmds {
		cc := cc
		n := len(cc)
		if n < 2 {
			continue
		}
		cmdCat := cc[0]
		cb := gi.NewButton(mm).SetText(cmdCat).SetType(gi.ButtonMenu)
		cb.SetMenu(func(m *gi.Scene) {
			for ii := 1; ii < n; ii++ {
				ii := ii
				it := cc[ii]
				cmdNm := gide.CommandName(cmdCat, it)
				b := gi.NewButton(m).SetText(it).OnClick(func(e events.Event) {
					cmd := gide.CmdName(cmdNm)
					ge.CmdHistory.Add(cmd)         // only save commands executed via chooser
					ge.SaveAllCheck(true, func() { // true = cancel option
						ge.ExecCmdName(cmd, true, true) // sel, clear
					})
				})
				if cmdNm == lastCmd {
					b.SetSelected(true)
				}
			}
		})
	}
	tv := ge.ActiveTextView()
	gi.NewMenuFromScene(mm, tv, tv.ContextMenuPos(nil)).Run()
}

// ExecCmd pops up a menu to select a command appropriate for the current
// active text view, and shows output in Tab with name of command
func (ge *GideView) ExecCmd() { //gti:add
	tv := ge.ActiveTextView()
	if tv == nil {
		fmt.Printf("no Active view for ExecCmd\n")
		return
	}
	lang := ge.ActiveLang
	if lang == filecat.NoSupport {
		lang = ge.Prefs.MainLang
	}
	ge.CommandFromMenu(lang)
}

// ExecCmdFileNode pops up a menu to select a command appropriate for the given node,
// and shows output in Tab with name of command
func (ge *GideView) ExecCmdFileNode(fn *filetree.Node) {
	lang := fn.Info.Sup
	ge.CommandFromMenu(lang)
}

// SetArgVarVals sets the ArgVar values for commands, from GideView values
func (ge *GideView) SetArgVarVals() {
	tv := ge.ActiveTextView()
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
		gi.NewDialog(ge).Title("No BuildCmds Set").
			Prompt("You need to set the BuildCmds in the Project Preferences").Modal(true).Ok().Run()
		return
	}
	ge.SaveAllCheck(true, func() { // true = cancel option
		ge.ExecCmds(ge.Prefs.BuildCmds, true, true)
	})
}

// Run runs the RunCmds set for this project
func (ge *GideView) Run() { //gti:add
	if len(ge.Prefs.RunCmds) == 0 {
		gi.NewDialog(ge).Title("No RunCmds Set").
			Prompt("You need to set the RunCmds in the Project Preferences").Modal(true).Ok().Run()
		return
	}
	if ge.Prefs.RunCmds[0] == "Run Proj" && !ge.Prefs.RunExecIsExec() {
		giv.NewSoloFuncButton(ge, ge.ChooseRunExec).CallFunc()
		return
	}
	ge.ExecCmds(ge.Prefs.RunCmds, true, true)
}

// Commit commits the current changes using relevant VCS tool.
// Checks for VCS setting and for unsaved files.
func (ge *GideView) Commit() { //gti:add
	vc := ge.VersCtrl()
	if vc == "" {
		gi.NewDialog(ge).Title("No Version Control System Found").
			Prompt("No version control system detected in file system, or defined in project prefs -- define in project prefs if viewing a sub-directory within a larger repository").Modal(true).Ok().Run()
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
		gi.NewDialog(ge).Title("No Commit command found").
			Prompt("Could not find Commit command in list of avail commands -- this is usually a programmer error -- check preferences settings etc").Modal(true).Ok().Run()
		return
	}
	ge.SetArgVarVals() // need to set before setting prompt string below..

	d := gi.NewDialog(ge).Title("Commit message").
		Prompt("Please enter your commit message here -- remember this is essential front-line documentation.  Author information comes from User settings in GoGi Preferences.").Modal(true)
	tf := gi.NewTextField(d).SetText("").SetPlaceholder("Enter commit message here..")
	d.Cancel().Ok().Run()
	d.OnAccept(func(e events.Event) {
		val := tf.Text()
		ge.ArgVals["{PromptString1}"] = val
		gide.CmdNoUserPrompt = true                     // don't re-prompt!
		ge.ExecCmdName(gide.CmdName(cmdnm), true, true) // must be wait
		ge.SaveProjIfExists(true)                       // saveall
		ge.UpdateFiles()
	})
}
