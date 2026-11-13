// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"os"
	"path/filepath"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/terminal"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testCommands are the commands of the command bar in the tests, with their
// help already parsed, so that no help commands are run.
const testCommands = `{
	"Version": 9,
	"Entries": [
		{"Command": "git"},
		{"Command": "ls"}
	],
	"Parsed": {
		"git": {"Cmd": "git", "SubCmds": [
			{"Name": "status", "Cmd": "git status", "Source": "git status -h", "Flags": [{"Name": "--short", "Names": ["-s", "--short"]}]},
			{"Name": "commit", "Cmd": "git commit"}
		]},
		"ls": {"Cmd": "ls", "SubCmds": [
			{"Name": "Main", "Cmd": "ls", "Source": "man ls", "Flags": [{"Name": "-l", "Names": ["-l"]}]}
		]}
	}
}`

func init() {
	// (not the commands of the user)
	dir, err := os.MkdirTemp("", "files-test")
	if err != nil {
		panic(err)
	}
	CommandsFile = filepath.Join(dir, "commands.json")
	if err := os.WriteFile(CommandsFile, []byte(testCommands), 0o644); err != nil {
		panic(err)
	}
}

// TestCommandBar tests that the command bar has the top-level commands and
// categories, and that a command line runs in the last terminal used, or
// else in a new terminal.
func TestCommandBar(t *testing.T) {
	root := testRoot(t)
	testSettings(t)
	Settings.Terminal = "/bin/sh"
	t.Cleanup(func() { Settings.Terminal = "" })
	var ran []string
	var ranIn []*terminal.Terminal
	prevRun := runInTerminal
	runInTerminal = func(tm *terminal.Terminal, cmdline string) {
		ran = append(ran, cmdline)
		ranIn = append(ranIn, tm)
	}
	t.Cleanup(func() { runInTerminal = prevRun })
	b := core.NewBody()
	b.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(1200), units.Dp(700)) })
	fl := InitBody(b, root)
	b.AssertRender(t, "command-bar", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		var tb *core.Toolbar
		b.Scene.WalkDown(func(n tree.Node) bool {
			if w, ok := n.(*core.Toolbar); ok && w.Name == "commands" {
				tb = w
			}
			return tb == nil
		})
		require.NotNil(t, tb, "the command bar")
		var labels []string
		for _, kid := range tb.Children[:2] {
			labels = append(labels, kid.(*core.Button).Text)
		}
		assert.Equal(t, []string{"Git", "Ls"}, labels)
		assert.NotNil(t, tb.Child(0).(*core.Button).Menu, "the menu of the subcommands of git")
		assert.Nil(t, tb.Child(1).(*core.Button).Menu, "ls has none")
		assert.NotNil(t, tb.ChildByName("edit-commands"), "the edit button")

		// the default terminal is the last one used at the start
		n := fl.Tabs().NumTabs()
		fl.RunCommand("git status --short")
		require.Len(t, ran, 1)
		assert.Equal(t, "git status --short", ran[0])
		assert.Same(t, fl.lastTerminal, ranIn[0])
		assert.Equal(t, n, fl.Tabs().NumTabs(), "in the last terminal")

		fl.lastTerminal = nil
		fl.RunCommand("ls -l")
		require.Len(t, ran, 2)
		assert.NotSame(t, ranIn[0], ranIn[1], "in a new terminal")
		assert.Equal(t, n+1, fl.Tabs().NumTabs())
	})
}
