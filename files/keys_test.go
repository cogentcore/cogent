// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"image"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/filetree"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// eventually waits for the given condition, under the lock of the given
// body, reporting the given failure if it does not happen.
func eventually(t *testing.T, b *core.Body, what string, cond func() bool) {
	t.Helper()
	assert.Eventually(t, func() bool {
		b.AsyncLock()
		defer b.AsyncUnlock()
		return cond()
	}, 3*time.Second, 20*time.Millisecond, what)
}

// TestDirTableDeleteKey tests that the delete shortcut of the platform
// (Command+Backspace on macOS, as in the Finder), sent to the window as
// it is when the table of the files has the focus, deletes the selected
// file, once accepted in the dialog.
func TestDirTableDeleteKey(t *testing.T) {
	root := testRoot(t)
	testSettings(t)
	prevMap := keymap.ActiveMapName
	keymap.SetActiveMapName("MacStandard")
	defer keymap.SetActiveMapName(prevMap)
	b := core.NewBody()
	dt := NewDirTable(b).SetDir(root)
	meta := key.Modifiers(0)
	meta.SetFlag(true, key.Meta)
	b.RenderImage(func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		idx := slices.IndexFunc(dt.files, func(fi *fileinfo.FileInfo) bool { return fi.Name == "b.txt" })
		require.GreaterOrEqual(t, idx, 0)
		dt.table.SelectIndexEvent(idx, events.SelectOne)
		dt.table.SetFocus()
	}, func() {
		src := b.Scene.RenderWindow().SystemWindow.Events()
		src.Key(events.KeyDown, 0, key.CodeBackspace, meta)
		src.Key(events.KeyUp, 0, key.CodeBackspace, meta)
		time.Sleep(300 * time.Millisecond)                    // (for the dialog to be shown)
		src.Key(events.KeyDown, 0, key.CodeReturnEnter, meta) // (accepts it)
		src.Key(events.KeyUp, 0, key.CodeReturnEnter, meta)
		assert.Eventually(t, func() bool {
			_, err := os.Stat(filepath.Join(root, "b.txt"))
			return os.IsNotExist(err)
		}, 3*time.Second, 20*time.Millisecond, "the file is deleted")
	})
}

// TestDirTabDoubleClick tests that double-clicking on a directory tab with
// the mouse (two clicks sent to the window) opens a terminal in its
// directory.
func TestDirTabDoubleClick(t *testing.T) {
	root := testRoot(t)
	testSettings(t)
	Settings.Terminal = "/bin/sh"
	t.Cleanup(func() { Settings.Terminal = "" })
	b := core.NewBody()
	b.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(1200), units.Dp(700)) })
	fl := InitBody(b, root)
	n := 0
	b.RenderImage(func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		fl.OpenDir(filepath.Join(root, "docs"))
		n = fl.Tabs().NumTabs()
	}, func() {
		b.AsyncLock()
		_, tab := fl.DirTabs().TabAtIndex(fl.DirTabs().NumTabs() - 1)
		bb := tab.Geom.TotalBBox
		b.AsyncUnlock()
		require.False(t, bb.Empty())
		pt := bb.Min.Add(bb.Size().Div(2))
		src := b.Scene.RenderWindow().SystemWindow.Events()
		for range 2 {
			src.MouseButton(events.MouseDown, events.Left, pt, 0)
			src.MouseButton(events.MouseUp, events.Left, pt, 0)
		}
		eventually(t, b, "a new terminal", func() bool { return fl.Tabs().NumTabs() == n+1 })
		b.AsyncLock()
		defer b.AsyncUnlock()
		if fl.Tabs().NumTabs() == n+1 {
			fr, _ := fl.Tabs().TabAtIndex(n)
			assert.Equal(t, cleanDir(filepath.Join(root, "docs")), getTerminalDir(terminalIn(fr)).dir)
		}
	})
}

// TestDirTableOpenApp tests that opening an app bundle (a directory named
// *.app, on macOS) in the table of the files opens it as an app, unless
// hidden files are shown, in which case its files are shown.
func TestDirTableOpenApp(t *testing.T) {
	root := testRoot(t)
	testSettings(t)
	app := filepath.Join(root, "Tool.app")
	require.NoError(t, os.Mkdir(app, 0750))
	prevBundles, prevOpen, prevHidden := filetree.AppBundles, openFile, Settings.ShowHidden
	t.Cleanup(func() { filetree.AppBundles, openFile, Settings.ShowHidden = prevBundles, prevOpen, prevHidden })
	var opened []string
	filetree.AppBundles = true
	openFile = func(path string) { opened = append(opened, path) }
	b := core.NewBody()
	dt := NewDirTable(b).SetDir(root)
	b.RenderImage(func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		selectApp := func() {
			idx := slices.IndexFunc(dt.files, func(fi *fileinfo.FileInfo) bool { return fi.Name == "Tool.app" })
			require.GreaterOrEqual(t, idx, 0)
			dt.table.SelectIndexEvent(idx, events.SelectOne)
		}
		selectApp()
		dt.openSelected()
		assert.Equal(t, []string{app}, opened)
		assert.Equal(t, root, dt.Dir)

		Settings.ShowHidden = true
		selectApp()
		dt.openSelected()
		assert.Len(t, opened, 1)
		assert.Equal(t, app, dt.Dir)
	})
}

// TestDirTableDeleteShortcut tests that the Delete item of the context
// menu of the table of the files shows the delete shortcut of the keymap.
func TestDirTableDeleteShortcut(t *testing.T) {
	root := testRoot(t)
	testSettings(t)
	b := core.NewBody()
	dt := NewDirTable(b).SetDir(root)
	b.RenderImage(func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		idx := slices.IndexFunc(dt.files, func(fi *fileinfo.FileInfo) bool { return fi.Name == "b.txt" })
		require.GreaterOrEqual(t, idx, 0)
		dt.table.SelectIndexEvent(idx, events.SelectOne)
		m := core.NewScene()
		dt.contextMenu(m, image.Point{})
		var del *core.Button
		m.WalkDown(func(n tree.Node) bool {
			if bt, ok := n.(*core.Button); ok && bt.Text == "Delete" {
				del = bt
			}
			return tree.Continue
		})
		require.NotNil(t, del)
		assert.NotEmpty(t, del.Shortcut)
		assert.Equal(t, keymap.Delete.Chord(), del.Shortcut)
	})
}

// TestNewWindow tests that a new Files window, made from another one by
// the New window item of the Window menu or the New key (see
// [core.NewWindowFunc]), has its root.
func TestNewWindow(t *testing.T) {
	root := testRoot(t)
	testSettings(t)
	b := core.NewBody()
	InitBody(b, filepath.Join(root, "docs"))
	require.NotNil(t, core.NewWindowFunc)
	nb := core.NewWindowFunc(b.Scene)
	require.NotNil(t, nb)
	nf := tree.ChildByType[*Files](nb)
	require.NotNil(t, nf)
	assert.Equal(t, filepath.Join(root, "docs"), nf.Root)
}

// TestTabKeys tests that the keys for the next and previous tabs
// (Shift+RightArrow and Shift+LeftArrow), sent to the window, move among
// the terminal tabs from a terminal, and among the directory tabs from the
// table of the files.
func TestTabKeys(t *testing.T) {
	root := testRoot(t)
	testSettings(t)
	Settings.Terminal = "/bin/sh"
	t.Cleanup(func() { Settings.Terminal = "" })
	b := core.NewBody()
	b.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(1200), units.Dp(700)) })
	fl := InitBody(b, root)
	prevMap := keymap.ActiveMapName
	keymap.SetActiveMapName("MacStandard")
	defer keymap.SetActiveMapName(prevMap)
	var shift key.Modifiers
	shift.SetFlag(true, key.Shift)
	current := func(ts *core.Tabs) int {
		b.AsyncLock()
		defer b.AsyncUnlock()
		_, idx := ts.CurrentTab()
		return idx
	}
	b.RenderImage(func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		fl.OpenDir(filepath.Join(root, "docs"))
		fl.NewTerminal(root)
	}, func() {
		src := b.Scene.RenderWindow().SystemWindow.Events()
		press := func(code key.Codes) {
			src.Key(events.KeyDown, 0, code, shift)
			src.Key(events.KeyUp, 0, code, shift)
			time.Sleep(200 * time.Millisecond)
		}
		tabs := fl.Tabs()
		n := tabs.NumTabs()
		require.GreaterOrEqual(t, n, 2)
		b.AsyncLock()
		fr, idx := tabs.CurrentTab()
		tm := terminalIn(fr.(*core.Frame))
		require.NotNil(t, tm)
		tm.SetFocus()
		b.AsyncUnlock()
		press(key.CodeRightArrow)
		assert.Equal(t, (idx+1)%n, current(tabs), "the next terminal tab, from a terminal")
		press(key.CodeLeftArrow)
		assert.Equal(t, idx, current(tabs), "the previous terminal tab")

		dirs := fl.DirTabs()
		require.GreaterOrEqual(t, dirs.NumTabs(), 2)
		b.AsyncLock()
		_, di := dirs.CurrentTab()
		dt := fl.DirTables()[di]
		dt.FocusTable()
		b.AsyncUnlock()
		eventually(t, b, "the table is focused", func() bool { return dt.table.ContainsFocus() })
		press(key.CodeRightArrow)
		assert.Equal(t, (di+1)%dirs.NumTabs(), current(dirs), "the next directory tab, from the table")
	})
}
