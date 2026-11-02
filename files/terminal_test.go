// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/text/terminal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTerminalDirCD(t *testing.T) {
	root := testRoot(t)
	home, err := os.UserHomeDir()
	require.NoError(t, err)
	home = cleanDir(home)
	docs, src := filepath.Join(root, "docs"), filepath.Join(root, "src")
	t.Setenv("FILESTEST", src)
	for _, tc := range []struct {
		line, dir string
	}{
		{"cd docs", docs},
		{"cd " + src, src},
		{"cd", home},
		{"cd ~", home},
		{"cd -P docs", docs},
		{`cd "docs"`, docs},
		{"cd $FILESTEST", src},
		{"cd docs && ls", docs},
		{"cd docs; cd ../src", src},
		{"cd docs | cat", root},        // in a subshell
		{"cd missing", root},           // fails
		{"echo cd docs", root},         // not a cd
		{"ls; echo 'a;cd docs'", root}, // quoted
	} {
		td := &terminalDir{dir: root}
		td.cd(tc.line)
		assert.Equal(t, tc.dir, td.dir, tc.line)
	}
	td := &terminalDir{dir: root}
	assert.True(t, td.cd("cd docs"))
	assert.False(t, td.cd("cd ."))
	assert.True(t, td.cd("cd -"))
	assert.Equal(t, root, td.dir)
	assert.True(t, td.cd("cd -"))
	assert.Equal(t, docs, td.dir)
}

// TestTerminalLabels tests that the terminal tabs are labeled with their
// working directories, which follow the cd commands entered by the user.
func TestTerminalLabels(t *testing.T) {
	root := testRoot(t)
	testSettings(t)
	Settings.Terminal = "/bin/sh"
	t.Cleanup(func() { Settings.Terminal = "" })
	b := core.NewBody()
	fl := NewFiles(b).SetRoot(root)
	var tm1 *terminal.Terminal
	labels := func() []string {
		var ls []string
		for i := range fl.Tabs().NumTabs() {
			_, tab := fl.Tabs().TabAtIndex(i)
			ls = append(ls, tab.Text)
		}
		return ls
	}
	waitLabels := func(want ...string) {
		t.Helper()
		var got []string
		for range 300 {
			b.AsyncLock()
			got = labels()
			b.AsyncUnlock()
			if assert.ObjectsAreEqual(append([]string{ConsoleTabName}, want...), got) {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Fatalf("labels: %v, want %v", got, want)
	}
	b.AssertRender(t, "terminals", func() {
		b.AsyncLock()
		tm1 = fl.recycleTerminal(TerminalTabName, root)
		fl.NewTerminal(root)
		b.AsyncUnlock()
		base := filepath.Base(root)
		waitLabels(base, base+" 2")
		b.AsyncLock()
		fl.Tabs().SelectTabByName(TerminalTabName) // typed into the terminal shown
		b.AsyncUnlock()
		time.Sleep(300 * time.Millisecond) // for the prompt

		b.AsyncLock()
		tm1.SendText("cd docs\r")
		b.AsyncUnlock()
		waitLabels("docs", base)

		// not for the commands of a script
		b.AsyncLock()
		tm1.SendText("sh -c 'cd ../src'\r")
		b.AsyncUnlock()
		time.Sleep(300 * time.Millisecond) // for the next prompt
		waitLabels("docs", base)
		b.AsyncLock()
		tm1.SendText("cd ..\r")
		b.AsyncUnlock()
		waitLabels(base, base+" 2")
	})
}
