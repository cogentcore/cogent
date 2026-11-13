// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"image"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/cli/commands"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/terminal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	// (not the commands of the user, with only ls to parse)
	dir, err := os.MkdirTemp("", "code-commands")
	if err != nil {
		panic(err)
	}
	commands.File = filepath.Join(dir, "commands.json")
	if err := os.WriteFile(commands.File, []byte(`{"Version": 2, "StandardVersion": 2, "Entries": [{"Command": "ls"}]}`), 0o644); err != nil {
		panic(err)
	}
}

func TestCommandTabName(t *testing.T) {
	for cmdline, name := range map[string]string{
		"git commit --all --message 'the fix'": "git commit",
		"go test -v ./...":                     "go test",
		"go mod tidy":                          "go mod tidy",
		"git push origin HEAD":                 "git push origin",
		"ls":                                   "ls",
		"make":                                 "make",
		"core run web":                         "core run web",
		"/bin/sh '/a/Cogent Core/scripts/run/git/gitprune'": "gitprune",
		"/bin/sh /a/scripts/run/commands/psgrep":            "psgrep",
		"":                                                  "command",
	} {
		assert.Equal(t, name, commandTabName(cmdline), cmdline)
	}
}

// TestRunCommandLine tests running the command line of a command: in the
// directory of the file of the active editor, with its output in its tab,
// or else in the terminal that has the focus. It also tests the Command
// menu, with the commands of the apps and the other commands of Code.
func TestRunCommandLine(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	tmp, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	sub := filepath.Join(tmp, "sub")
	require.NoError(t, os.Mkdir(sub, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(sub, "a.txt"), []byte("a\n"), 0o644))
	prevCmd := Settings.Terminal.Command
	Settings.Terminal.Command = "/bin/sh"
	defer func() { Settings.Terminal.Command = prevCmd }()
	b := core.NewBody()
	b.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(1200), units.Dp(800)) })
	cv := NewCode(b)
	cv.Defaults()
	cv.ProjectRoot = core.Filename(tmp)
	cv.Settings.ProjectRoot = core.Filename(tmp)
	showed := make(chan struct{})
	b.OnFinal(events.Show, func(e events.Event) { showed <- struct{}{} })
	b.RunWindow()
	<-showed
	defer func() {
		b.AsyncLock()
		b.Close()
		b.AsyncUnlock()
	}()
	waitFor := func(what string, cond func() bool) {
		t.Helper()
		deadline := time.Now().Add(10 * time.Second)
		for time.Now().Before(deadline) {
			b.AsyncLock()
			ok := cond()
			b.AsyncUnlock()
			if ok {
				return
			}
			time.Sleep(20 * time.Millisecond)
		}
		t.Fatalf("timeout waiting for %s", what)
	}

	// in the directory of the file of the active editor, with its output
	b.AsyncLock()
	cv.ViewFileInIndex(filepath.Join(sub, "a.txt"), 0)
	cv.ActiveEditor().SetFocus()
	cv.RunCommandLine("pwd")
	b.AsyncUnlock()
	waitFor("the output of pwd", func() bool {
		buf := cv.CmdBufs["pwd"]
		return buf != nil && strings.Contains(buf.String(), sub+"\n") && strings.Contains(buf.String(), "pwd successful")
	})
	b.AsyncLock()
	assert.True(t, strings.HasPrefix(cv.CmdBufs["pwd"].String(), sub+"$ pwd\n"+sub+"\n"), "the directory and command line first: %q", cv.CmdBufs["pwd"].String())
	b.AsyncUnlock()
	b.AsyncLock()
	ct, _ := cv.Tabs().CurrentTab()
	assert.Equal(t, sub, cv.buildDir(), "build in the directory of the active editor")
	cv.Settings.BuildDir = core.Filename(tmp)
	assert.Equal(t, sub, cv.buildDir(), "the project root is the default")
	cv.Settings.BuildDir = core.Filename(filepath.Join(tmp, "build"))
	assert.Equal(t, filepath.Join(tmp, "build"), cv.buildDir(), "set")
	cv.Settings.BuildDir = ""
	b.AsyncUnlock()
	require.NotNil(t, ct)
	assert.Equal(t, "pwd", ct.AsTree().Name, "its tab is shown")

	// in the terminal that has the focus
	var tm *terminal.Terminal
	waitFor("the default terminal", func() bool {
		if tms := cv.Terminals(); len(tms) > 0 {
			tm = tms[0]
		}
		return tm != nil && tm.IsRunning() && strings.Contains(terminalText(tm), "$")
	})
	b.AsyncLock()
	cv.OpenTerminal()
	tm.SetFocus()
	b.AsyncUnlock()
	waitFor("the focus in the terminal", func() bool { return cv.focusedTerminal() == tm })
	b.AsyncLock()
	cv.RunCommandLine("echo from-command")
	b.AsyncUnlock()
	waitFor("the command in the terminal", func() bool { return hasOutputLine(tm, "from-command") })
	b.AsyncLock()
	_, inBuf := cv.CmdBufs["echo"]
	b.AsyncUnlock()
	assert.False(t, inBuf, "not in an output tab")

	// the Command menu
	b.AsyncLock()
	defer b.AsyncUnlock()
	m := core.NewScene()
	cv.commandsMenu(m)
	var texts []string
	for _, kid := range m.Children {
		if bt, ok := kid.(*core.Button); ok {
			texts = append(texts, bt.Text)
		}
	}
	assert.Equal(t, []string{"Ls", "Edit commands", "Edit choosers"}, texts, "the commands of the apps")
}

// TestBuildRunCommands tests the default build and run command lines of
// projects, by main language, and the settings that set them.
func TestBuildRunCommands(t *testing.T) {
	se := &ProjectSettings{ProjectRoot: "/proj", MainLang: fileinfo.Go}
	assert.Equal(t, "go build -v", se.buildCommand())
	assert.Equal(t, "core run", se.runCommand())
	assert.Equal(t, "", se.buildDir(), "that of the active editor")
	se.BuildCommand, se.RunCommand, se.BuildDir = "go build ./cmd/x", "./x -v", "/proj/cmd/x"
	assert.Equal(t, "go build ./cmd/x", se.buildCommand())
	assert.Equal(t, "./x -v", se.runCommand())
	assert.Equal(t, "/proj/cmd/x", se.buildDir())

	se = &ProjectSettings{MainLang: fileinfo.TeX}
	assert.Equal(t, "latexmk -pdf", se.buildCommand())
	assert.Equal(t, "", se.runCommand())

	exe := filepath.Join(t.TempDir(), "my prog")
	require.NoError(t, os.WriteFile(exe, []byte("#!/bin/sh\n"), 0o755))
	se = &ProjectSettings{MainLang: fileinfo.C, RunExec: core.Filename(exe)}
	assert.Equal(t, "make", se.buildCommand())
	assert.Equal(t, "'"+exe+"'", se.runCommand(), "the RunExec, quoted")
}

// TestOpenOldProjectSettings tests that the project settings of a project
// file from before the build and run command lines open, without the old
// names of commands.
func TestOpenOldProjectSettings(t *testing.T) {
	file := filepath.Join(t.TempDir(), "old.code")
	old := "ProjectRoot = \"/proj\"\nBuildCmds = [\"Go: Build Dir\"]\nBuildTarg = \"/proj\"\nRunCmds = [\"Core: Run\"]\nRunExec = \"/proj/x\"\n"
	require.NoError(t, os.WriteFile(file, []byte(old), 0o644))
	se := &ProjectSettings{}
	require.NoError(t, se.Open(core.Filename(file)))
	assert.Equal(t, core.Filename("/proj"), se.ProjectRoot)
	assert.Equal(t, core.Filename("/proj/x"), se.RunExec)
	assert.Equal(t, "", se.BuildCommand, "the default")
}

// TestTerminalHere tests the Terminal here items of the context menus of
// the editors and the file tree, which open a new terminal in the
// directory of the file, or in the directory.
func TestTerminalHere(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	tmp, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	sub := filepath.Join(tmp, "sub")
	require.NoError(t, os.Mkdir(sub, 0o755))
	file := filepath.Join(sub, "a.txt")
	require.NoError(t, os.WriteFile(file, []byte("a\n"), 0o644))
	prevCmd := Settings.Terminal.Command
	Settings.Terminal.Command = "/bin/sh"
	defer func() { Settings.Terminal.Command = prevCmd }()
	b := core.NewBody()
	b.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(1200), units.Dp(800)) })
	cv := NewCode(b)
	cv.Defaults()
	cv.ProjectRoot = core.Filename(tmp)
	cv.Settings.ProjectRoot = core.Filename(tmp)
	showed := make(chan struct{})
	b.OnFinal(events.Show, func(e events.Event) { showed <- struct{}{} })
	b.RunWindow()
	<-showed
	defer func() {
		b.AsyncLock()
		b.Close()
		b.AsyncUnlock()
	}()
	menuItem := func(menu func(m *core.Scene, pos image.Point), text string) *core.Button {
		m := core.NewScene()
		menu(m, image.Point{})
		for _, kid := range m.Children {
			if bt, ok := kid.(*core.Button); ok && bt.Text == text {
				return bt
			}
		}
		return nil
	}
	pwdIn := func(tm *terminal.Terminal, dir string) {
		t.Helper()
		deadline := time.Now().Add(10 * time.Second)
		sent := false
		for time.Now().Before(deadline) {
			b.AsyncLock()
			ready := tm.IsRunning() && strings.Contains(terminalText(tm), "$")
			if ready && !sent {
				tm.SendText("pwd\r")
				sent = true
			}
			ok := sent && hasOutputLine(tm, dir)
			b.AsyncUnlock()
			if ok {
				return
			}
			time.Sleep(20 * time.Millisecond)
		}
		t.Fatalf("the terminal is not in %s", dir)
	}
	newest := func() *terminal.Terminal {
		b.AsyncLock()
		defer b.AsyncUnlock()
		tms := cv.Terminals()
		return tms[len(tms)-1]
	}

	// the editor of a file: its directory
	b.AsyncLock()
	cv.ViewFileInIndex(file, 0)
	var bt *core.Button
	if ed := cv.ActiveEditor(); ed != nil {
		bt = menuItem(ed.ContextMenu, "Terminal here")
	}
	if bt != nil {
		bt.Send(events.Click)
	}
	b.AsyncUnlock()
	require.NotNil(t, bt, "in the context menu of the editor")
	tm := newest()
	assert.NotEqual(t, TerminalTabName, tm.Name, "a new terminal")
	pwdIn(tm, sub)

	// the file tree: a directory itself, and the directory of a file
	for path, dir := range map[string]string{sub: sub, file: sub} {
		b.AsyncLock()
		var bt *core.Button
		if cv.Files.Filepath == "" {
			cv.Files.OpenPath(tmp)
		}
		if n, ok := cv.Files.FindFile(path); ok {
			bt = menuItem(n.This.(*FileNode).ContextMenu, "Terminal here")
		}
		if bt != nil {
			bt.Send(events.Click)
		}
		b.AsyncUnlock()
		require.NotNil(t, bt, "in the context menu of the file tree: %s", path)
		pwdIn(newest(), dir)
	}
}
