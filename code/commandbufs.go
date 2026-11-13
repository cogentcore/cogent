// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"image"
	"os/exec"
	"slices"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/vcs"
	"cogentcore.org/core/cli/commands"
	"cogentcore.org/core/core"
	"cogentcore.org/core/text/highlighting"
	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/textcore"
	"github.com/alecthomas/chroma/v3/lexers"
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
	buf.SetHighlighting(core.AppearanceSettings.Highlighting)
	// note: critical to NOT set this, otherwise overwrites our native markup
	// buf.SetLanguage(fileinfo.Bash)
	return buf, true
}

// todo: Markup colors are baked in when output is generated -- no remarkup possible!

// CmdBuffsReMarkup runs ReMarkup on all command bufs.
func (cv *Code) CmdBuffsReMarkup() {
	for _, ln := range cv.CmdBufs {
		ln.SetHighlighting(core.AppearanceSettings.Highlighting)
		ln.ReMarkup()
	}
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
	ctv.AutoscrollOnInput = true
	ctv.SetReadOnly(true)
	ctv.SetLines(buf)
	ctv.HighlightsReset()
	ctv.LinkHandler = func(tl *rich.Hyperlink) {
		cv.OpenFileURL(tl.URL, ctv)
	}
	return buf, ctv, nw
}

// TabDeleted is called when a main tab is deleted -- we cancel any running commands
func (cv *Code) TabDeleted(tabnm string) {
	cv.RunningCmds.KillByName(tabnm)
}

// CmdRun is a command that is running, with its output in a tab.
type CmdRun struct {

	// Name is the name of the tab of its output.
	Name string

	// CmdStr is its command line.
	CmdStr string

	// Exec is its process.
	Exec *exec.Cmd
}

// Kill kills the process.
func (cm *CmdRun) Kill() {
	if cm.Exec.Process != nil {
		cm.Exec.Process.Kill()
	}
}

// CmdRuns is a list of running commands.
type CmdRuns []*CmdRun

// AddCmd adds a running command with the given name (of its tab), command
// line, and process.
func (rc *CmdRuns) AddCmd(name, cmdstr string, ex *exec.Cmd) {
	*rc = append(*rc, &CmdRun{Name: name, CmdStr: cmdstr, Exec: ex})
}

// ByName returns the running command of the given name, and its index,
// which is -1 if there is none.
func (rc *CmdRuns) ByName(name string) (*CmdRun, int) {
	for i, cm := range *rc {
		if cm.Name == name {
			return cm, i
		}
	}
	return nil, -1
}

// DeleteByName removes the running command of the given name from the
// list, returning whether there was one.
func (rc *CmdRuns) DeleteByName(name string) bool {
	_, idx := rc.ByName(name)
	if idx >= 0 {
		*rc = slices.Delete(*rc, idx, idx+1)
	}
	return idx >= 0
}

// KillByName kills the running command of the given name, and removes it
// from the list, returning whether there was one.
func (rc *CmdRuns) KillByName(name string) bool {
	cm, idx := rc.ByName(name)
	if idx >= 0 {
		cm.Kill()
		*rc = slices.Delete(*rc, idx, idx+1)
	}
	return idx >= 0
}

// MarkupCmdOutput applies links to the first element in command output line
// if it looks like a file name / position, and runs markup using given lexer
// name if provided (default is "bash")
func MarkupCmdOutput(buf *lines.Lines, out []rune, lexName string) rich.Text {
	if len(out) == 0 {
		return rich.NewPlainText(nil)
	}
	clex := lexers.Get("bash")
	if lexName != "" {
		nl := lexers.Get(lexName)
		if nl != nil {
			clex = nl
		}
	}
	sout := string(out)
	ctags, _ := highlighting.ChromaTagsLine(clex, sout)
	mu := highlighting.MarkupLineRich(buf.Highlighter.Style, buf.FontStyle(), out, ctags, nil)
	mu = highlighting.MarkupPathsAsLinks(out, mu, 2) // first 2 fields
	return mu
}

// ExecCmd pops up the menu of the commands (as the Command menu), shared
// by the apps, with their macros (see [Code.RunCommandLine]).
func (cv *Code) ExecCmd() { //types:add
	var ctx core.Widget = cv
	if tv := cv.ActiveEditor(); tv != nil {
		ctx = tv
	}
	core.NewMenu(func(m *core.Scene, pos image.Point) { cv.commandsMenu(m) }, ctx, ctx.AsWidget().ContextMenuPos(nil)).Run()
}

// RunBuild runs the [ProjectSettings.BuildCommand] of the project in its
// build directory, after saving the files, with its output in a tab.
func (cv *Code) RunBuild() { //types:add
	cmdline := cv.Settings.buildCommand()
	if cmdline == "" {
		core.MessageDialog(cv, "You need to set the BuildCommand in the Project Settings", "No BuildCommand set")
		return
	}
	cv.SaveAllCheck(true, func() {
		cv.runCommandOutput(cmdline, cv.Settings.buildDir(), nil)
	})
}

// Run runs the [ProjectSettings.RunCommand] of the project in its build
// directory, with its output in a tab: by default, for a project that is
// not in Go or TeX, its RunExec executable, which is chosen if it is not
// executable.
func (cv *Code) Run() { //types:add
	cmdline := cv.Settings.runCommand()
	if cmdline == "" {
		if cv.Settings.MainLang != fileinfo.Go && cv.Settings.MainLang != fileinfo.TeX {
			core.CallFunc(cv, cv.ChooseRunExec)
			return
		}
		core.MessageDialog(cv, "You need to set the RunCommand in the Project Settings", "No RunCommand set")
		return
	}
	cv.runCommandOutput(cmdline, cv.Settings.buildDir(), nil)
}

// Commit commits the current changes, after saving the files, with the
// git commit macro of the commands (see [commands.StandardMacros]), which
// prompts for the message, and then updates the files.
func (cv *Code) Commit() { //types:add
	if cv.VersionControl() != vcs.Git {
		core.MessageDialog(cv, "Commit is for git repositories: no git repository was found in the project, or defined in its settings. Use the Command menu for others.", "No git repository")
		return
	}
	cv.SaveAllCheck(true, func() { // true = cancel option
		cv.CommitNoChecks()
	})
}

// CommitNoChecks commits with the git commit macro of the commands (or the
// form of git commit, if there is none), without checking for unsaved files.
func (cv *Code) CommitNoChecks() {
	cs := commands.Default()
	run := func(cmdline string) {
		cv.runCommandOutput(cmdline, cv.commandDir(), func() {
			cv.SaveProjectIfExists(true)
			cv.UpdateFiles()
		})
	}
	if mc := cs.Macro("git", "commit", ""); mc != nil {
		cs.RunMacro(cv, "git", mc, run)
		return
	}
	cs.RunDialog(cv, "git commit", run)
}
