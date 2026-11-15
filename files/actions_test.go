// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestActions tests making a new folder, opening a terminal here and from
// the directory menu items, adding a favorite from them, the editor of the
// favorites, and the directory of the commands.
func TestActions(t *testing.T) {
	root := testRoot(t)
	testSettings(t)
	t.Cleanup(core.TrackWindows())
	b := core.NewBody()
	fl := NewFiles(b).SetRoot(root)
	core.RenderImage(b, core.Locked(b, func() {
		dt := fl.CurrentDirTable()
		assert.Error(t, dt.NewFolder(""))
		require.NoError(t, dt.NewFolder("made"))
		assert.DirExists(t, filepath.Join(root, "made"))
		assert.Error(t, dt.NewFolder("a.go/sub"), "in a file")

		assert.Equal(t, root, commandsDir(dt), "the current directory")
		assert.Equal(t, "", commandsDir(core.NewBody()), "no Files")

		n := fl.Tabs().NumTabs()
		fl.TerminalHere()
		assert.Equal(t, n+1, fl.Tabs().NumTabs())

		m := core.NewScene()
		docs := filepath.Join(root, "docs")
		fl.dirMenuItems(m, docs)
		core.ButtonByText(m, "Terminal here").Send(events.Click)
		assert.Equal(t, n+2, fl.Tabs().NumTabs())
		core.ButtonByText(m, "Add to favorites").Send(events.Click)
		found := false
		for _, f := range core.SystemSettings.FavPaths {
			found = found || f.Path == docs
		}
		assert.True(t, found, "a favorite")
		fl.EditFavorites() // a window, closed at the end
	}))
}

// TestConsoleOutput tests that the console shows the output of the app
// (os.Stdout and os.Stderr), which it still writes to the original ones.
func TestConsoleOutput(t *testing.T) {
	prev := TheConsole
	stdout, stderr := os.Stdout, os.Stderr
	t.Cleanup(func() {
		os.Stdout, os.Stderr = stdout, stderr
		TheConsole = prev
	})
	TheConsole = Console{}
	TheConsole.Init()
	outWrite, errWrite := os.Stdout, os.Stderr
	t.Cleanup(func() { // after the restore above, which runs last
		log.SetOutput(stderr)
		outWrite.Close() // which ends the monitoring
		errWrite.Close()
	})
	fmt.Fprintln(os.Stdout, "to stdout")
	fmt.Fprintln(os.Stderr, "to stderr")
	require.Eventually(t, func() bool {
		s := TheConsole.Lines.String()
		return strings.Contains(s, "to stdout") && strings.Contains(s, "to stderr")
	}, 5*time.Second, 10*time.Millisecond)
	os.Stdout, os.Stderr = stdout, stderr
	TheConsole.Clear()
	assert.Equal(t, "\n", TheConsole.Lines.String())
}
