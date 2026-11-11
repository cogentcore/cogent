// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"path/filepath"
	"testing"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/text/terminal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tabNames returns the names of the tabs of the given tabs.
func tabNames(ts *core.Tabs) []string {
	var ns []string
	for i := range ts.NumTabs() {
		_, tab := ts.TabAtIndex(i)
		ns = append(ns, tab.Name)
	}
	return ns
}

// terminalIn returns the terminal in the given frame of a tab, or nil.
func terminalIn(fr *core.Frame) *terminal.Terminal {
	if fr == nil || !fr.HasChildren() {
		return nil
	}
	tm, _ := fr.Child(0).(*terminal.Terminal)
	return tm
}

// TestTerminalNewTab tests that the new tab button of the command tabs
// makes a new terminal, in the current directory, which gets the focus,
// and that a click on the tab of a terminal gives it the focus.
func TestTerminalNewTab(t *testing.T) {
	root := testRoot(t)
	testSettings(t)
	Settings.Terminal = "/bin/sh"
	t.Cleanup(func() { Settings.Terminal = "" })
	b := core.NewBody()
	fl := NewFiles(b).SetRoot(root)
	b.RenderImage(func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		ts := fl.Tabs()
		require.True(t, ts.NewTabButton)
		require.NotNil(t, ts.NewTabFunc)
		// (as the new tab button does)
		ts.NewTab("New tab")
		idx := ts.NumTabs() - 1
		ts.NewTabFunc(idx)
		ts.SelectTabIndex(idx)
		fr, tab := ts.TabAtIndex(idx)
		tm := terminalIn(fr)
		require.NotNil(t, tm, "a terminal in the new tab")
		assert.Equal(t, terminalTabName(2), tab.Name)
		assert.Equal(t, cleanDir(root), getTerminalDir(tm).dir)
	}, func() {
		b.AsyncLock()
		ts := fl.Tabs()
		fr, _ := ts.TabAtIndex(ts.NumTabs() - 1)
		focused := terminalIn(fr).ContainsFocus()
		// a click on the tab of the default terminal gives it the focus
		dfr := ts.TabByName(TerminalTabName)
		_, dtab := ts.TabAtIndex(dfr.IndexInParent())
		dtab.Send(events.Click)
		b.AsyncUnlock()
		assert.True(t, focused, "the new terminal has the focus")
		assert.Eventually(t, func() bool {
			b.AsyncLock()
			defer b.AsyncUnlock()
			return terminalIn(dfr).ContainsFocus()
		}, 3*time.Second, 20*time.Millisecond, "the terminal of the clicked tab has the focus")
	})
}

// TestFilesTabsMove tests that the directory and terminal tabs of a Files
// can move into those of another (as dragged into another Files window),
// with what goes with them: the label of a directory tab follows its
// directory, and a terminal keeps its working directory.
func TestFilesTabsMove(t *testing.T) {
	root := testRoot(t)
	testSettings(t)
	Settings.Terminal = "/bin/sh"
	t.Cleanup(func() { Settings.Terminal = "" })
	b1, b2 := core.NewBody(), core.NewBody()
	fl1 := InitBody(b1, root)
	fl2 := InitBody(b2, root)
	fl1.Update()
	fl2.Update()
	fl1.OpenDir(root)
	dt := fl1.OpenDir(filepath.Join(root, "docs"))
	fl2.OpenDir(root)
	tm := fl1.recycleTerminal(TerminalTabName, filepath.Join(root, "src"))
	fl1.recycleTerminal(terminalTabName(2), root)

	// a directory tab
	at := fl1.DirTabs().MoveTabTo(1, fl2.DirTabs(), 1)
	require.Equal(t, 1, at)
	assert.Equal(t, 1, fl1.DirTabs().NumTabs())
	assert.Same(t, dt, fl2.DirTables()[1])
	dt.SetDir(filepath.Join(root, "src"))
	_, tab := fl2.DirTabs().TabAtIndex(1)
	assert.Equal(t, "src", tab.Text, "its label follows its directory in the new tabs")

	// a terminal tab
	idx := fl1.Tabs().TabByName(TerminalTabName).IndexInParent()
	fl1.lastTerminal = tm
	at = fl1.Tabs().MoveTabTo(idx, fl2.Tabs(), fl2.Tabs().NumTabs())
	require.GreaterOrEqual(t, at, 0)
	fr, tab := fl2.Tabs().TabAtIndex(at)
	assert.Same(t, tm, terminalIn(fr))
	assert.Equal(t, "src", tab.Text, "labeled by its working directory")
	assert.Same(t, tm, fl2.lastTerminal)
	assert.Nil(t, fl1.lastTerminal, "not the last terminal of the Files it left")
	assert.NotContains(t, tabNames(fl1.Tabs()), TerminalTabName)
}

// TestDirTabTerminal tests that double-clicking on a directory tab opens
// a new terminal in its directory, also once it has moved into the tabs of
// another Files, in which the terminal opens.
func TestDirTabTerminal(t *testing.T) {
	root := testRoot(t)
	testSettings(t)
	Settings.Terminal = "/bin/sh"
	t.Cleanup(func() { Settings.Terminal = "" })
	b1, b2 := core.NewBody(), core.NewBody()
	fl1 := InitBody(b1, root)
	fl2 := InitBody(b2, root)
	fl1.Update()
	fl2.Update()
	fl1.OpenDir(root)
	fl1.OpenDir(filepath.Join(root, "docs"))
	n1, n2 := fl1.Tabs().NumTabs(), fl2.Tabs().NumTabs()
	lastDir := func(fl *Files) string {
		ts := fl.Tabs()
		fr, _ := ts.TabAtIndex(ts.NumTabs() - 1)
		require.NotNil(t, terminalIn(fr))
		return getTerminalDir(terminalIn(fr)).dir
	}
	_, tab := fl1.DirTabs().TabAtIndex(1)
	tab.Send(events.DoubleClick)
	require.Equal(t, n1+1, fl1.Tabs().NumTabs(), "a new terminal")
	assert.Equal(t, cleanDir(filepath.Join(root, "docs")), lastDir(fl1))

	fl1.DirTabs().MoveTabTo(1, fl2.DirTabs(), 0)
	_, tab = fl2.DirTabs().TabAtIndex(0)
	tab.Send(events.DoubleClick)
	assert.Equal(t, n1+1, fl1.Tabs().NumTabs())
	require.Equal(t, n2+1, fl2.Tabs().NumTabs(), "a new terminal in the Files it moved into")
	assert.Equal(t, cleanDir(filepath.Join(root, "docs")), lastDir(fl2))
}

// TestFilesDragToWindow tests that the tabs of Files are dragged into a new
// Files window with the same root.
func TestFilesDragToWindow(t *testing.T) {
	root := testRoot(t)
	testSettings(t)
	b := core.NewBody()
	fl := InitBody(b, root)
	fl.Update()
	for _, ts := range []*core.Tabs{fl.DirTabs(), fl.Tabs()} {
		require.NotNil(t, ts.DragToWindow)
		nb := core.NewBody()
		nt := ts.DragToWindow(nb)
		require.NotNil(t, nt)
		nf, ok := ParentFiles(nt)
		require.True(t, ok)
		assert.Equal(t, fl.Root, nf.Root)
	}
}

// TestDirTableMultiSelect tests that several files can be selected in a
// directory table, as for dragging or deleting them together.
func TestDirTableMultiSelect(t *testing.T) {
	root := testRoot(t)
	testSettings(t)
	b := core.NewBody()
	dt := NewDirTable(b).SetDir(root)
	b.RenderImage(func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		assert.True(t, dt.table.ReadOnlyMultiSelect)
		dt.table.SelectIndexEvent(0, events.SelectOne)
		dt.table.SelectIndexEvent(2, events.ExtendOne)
		var got []string
		for _, fi := range dt.SelectedFiles() {
			got = append(got, fi.Name)
		}
		assert.Equal(t, []string{dt.files[0].Name, dt.files[2].Name}, got)
	})
}
