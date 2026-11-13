// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"image"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unicode"

	"cogentcore.org/core/cli/commands"
	"cogentcore.org/core/core"
	"cogentcore.org/core/filetree"
	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/textcore"
)

// commandsMenu adds the commands to the given menu: those shared by the
// apps (see [commands.Default]), with their macros, run with
// [Code.RunCommandLine].
func (cv *Code) commandsMenu(m *core.Scene) {
	commands.Default().Menu(m, cv.RunCommandLine)
}

// nodeCommandsMenu returns a function that adds the commands to a menu,
// as [Code.commandsMenu], to run in the directory of the given node of
// the file tree (that of a file), with their output in a tab.
func (cv *Code) nodeCommandsMenu(fn *filetree.Node) func(m *core.Scene, pos image.Point) {
	dir := string(fn.Filepath)
	if !fn.IsDir() {
		dir = filepath.Dir(dir)
	}
	return func(m *core.Scene, pos image.Point) {
		commands.Default().Menu(m, func(cmdline string) {
			cv.SaveAllCheck(true, func() {
				cv.runCommandOutput(cmdline, dir, nil)
			})
		})
	}
}

// RunCommandLine runs the given command line, as from the commands (see
// [commands.Default]): in the terminal that has the keyboard focus, if any,
// as if it was typed there, or else, after saving the files, in the
// directory of the file of the active editor (or the project root), with
// its output in a tab named for the command (see [commandTabName]).
func (cv *Code) RunCommandLine(cmdline string) {
	if tm := cv.focusedTerminal(); tm != nil {
		tm.SendText(cmdline + "\r")
		return
	}
	cv.SaveAllCheck(true, func() {
		cv.runCommandOutput(cmdline, cv.commandDir(), nil)
	})
}

// commandDir returns the directory that commands run in: that of the file
// of the active editor, or else the project root.
func (cv *Code) commandDir() string {
	if tv := cv.ActiveEditor(); tv != nil && tv.Lines != nil && tv.Lines.Filename() != "" {
		return filepath.Dir(string(tv.Lines.Filename()))
	}
	if cv.ProjectRoot != "" {
		return string(cv.ProjectRoot)
	}
	dir, _ := os.Getwd()
	return dir
}

// commandTabName returns the name of the tab of the output of the given
// command line: its command and subcommands, such as "git commit", up to
// its first argument that is not a subcommand (a flag, path, or value), or
// the name of the file of a script run with sh, such as that of a script
// of the commands (see [commands.Commands.ScriptCommandLine]).
func commandTabName(cmdline string) string {
	if file, ok := strings.CutPrefix(cmdline, "/bin/sh "); ok {
		if n := len(file); n > 2 && file[0] == '\'' && file[n-1] == '\'' {
			file = file[1 : n-1] // (a quoted path, of one word)
		}
		if !strings.ContainsAny(file, "' \t") || strings.HasPrefix(cmdline, "/bin/sh '") {
			return filepath.Base(file)
		}
	}
	f := strings.Fields(cmdline)
	if len(f) == 0 {
		return "command"
	}
	isSubcommand := func(w string) bool {
		return w[0] != '-' && !strings.ContainsFunc(w, func(r rune) bool {
			return !(unicode.IsLower(r) || unicode.IsDigit(r) || r == '-')
		})
	}
	words := f[:1]
	for _, w := range f[1:] {
		if len(words) == 3 || !isSubcommand(w) {
			break
		}
		words = append(words, w)
	}
	return strings.Join(words, " ")
}

// shellCommand returns the command to run the given command line with the
// shell, in the given directory.
func shellCommand(cmdline, dir string) *exec.Cmd {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", cmdline)
	} else {
		cmd = exec.Command("/bin/sh", "-c", cmdline)
	}
	cmd.Dir = dir
	return cmd
}

// runCommandOutput runs the given command line in the given directory,
// with its output in its tab (see [commandTabName]), in which it is killed
// if the tab is closed, followed by its status, and then calls the given
// function, if it is not nil (with the lock of cv).
func (cv *Code) runCommandOutput(cmdline, dir string, done func()) {
	name := commandTabName(cmdline)
	buf, _, _ := cv.RecycleCmdTab(name)
	if buf == nil {
		return
	}
	cv.SelectTabByName(name)
	cmd := shellCommand(cmdline, dir)
	cv.RunningCmds.AddCmd(name, cmdline, cmd)
	lexer := ""
	if strings.Contains(name, "diff") {
		lexer = "diff"
	}
	markup := func(buf *lines.Lines, out []rune) rich.Text { return MarkupCmdOutput(buf, out, lexer) }
	go func() {
		stdout, err := cmd.StdoutPipe()
		if err == nil {
			cmd.Stderr = cmd.Stdout
			if err = cmd.Start(); err == nil {
				obuf := textcore.OutputBuffer{}
				obuf.SetOutput(stdout).SetLines(buf).SetMarkupFunc(markup)
				obuf.MonitorOutput()
				err = cmd.Wait()
			}
		}
		if !cv.AsyncLockIfShown() {
			return
		}
		defer cv.AsyncUnlock()
		cv.RunningCmds.DeleteByName(name)
		cv.commandStatus(name, buf, cmdline, err)
		if done != nil {
			done()
		}
	}()
}

// commandStatus appends the status of the given command line, run with the
// given error, to the given buffer of its output in the tab of the given
// name, which is selected if it failed, and shows it in the status bar.
func (cv *Code) commandStatus(name string, buf *lines.Lines, cmdline string, err error) {
	sty := buf.FontStyle()
	bold := *sty
	bold.SetWeight(rich.Bold)
	status := rich.NewText(sty, []rune(cmdline))
	when := []rune(" at: " + time.Now().Format("Mon Jan  2 15:04:05 MST 2006"))
	switch ee, ok := err.(*exec.ExitError); {
	case err == nil:
		status.AddSpan(&bold, []rune(" successful")).AddSpan(sty, when)
	case ok:
		status.AddSpan(&bold, []rune(" failed")).AddSpan(sty, when).AddSpan(sty, []rune(" with error: "+ee.Error()))
	default:
		status.AddSpan(&bold, []rune(" exec error")).AddSpan(sty, when).AddSpan(sty, []rune(" error: "+err.Error()))
	}
	buf.SetReadOnly(true)
	buf.AppendTextMarkup([][]rune{{}, status.Join()}, []rich.Text{rich.NewText(sty, nil), status})
	if err != nil {
		cv.SelectTabByName(name)
	}
	cv.SetStatus(string(status.Join()))
}
