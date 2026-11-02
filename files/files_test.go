// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/base/iox/tomlx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	// In tests, the types of package main have its import path instead of
	// main, under which typegen registers them, so they are added under it.
	pkg := reflect.TypeFor[Files]().PkgPath()
	for name, typ := range types.Types {
		if strings.HasPrefix(name, "main.") {
			types.Types[pkg+strings.TrimPrefix(name, "main")] = typ
		}
	}
}

// testSettings makes the tests use a settings file of their own,
// and the default settings, with the given favorite paths.
func testSettings(t *testing.T, favs ...string) {
	imagex.UpdateTestImages = true // the images are for viewing the layout
	t.Cleanup(func() { imagex.UpdateTestImages = false })
	file, splits := Settings.File, Settings.Splits
	sys := core.SystemSettings.FavPaths
	t.Cleanup(func() {
		Settings.File, Settings.Splits = file, splits
		Settings.SplitValues, Settings.TreeSplits = nil, nil
		core.SystemSettings.FavPaths = sys
	})
	Settings.File = filepath.Join(t.TempDir(), "settings.toml")
	Settings.Splits, Settings.SplitValues, Settings.TreeSplits = "Wide", nil, nil
	core.SystemSettings.FavPaths = core.SystemSettings.FavPaths[:0:0]
	for _, f := range favs {
		item := reflect.New(reflect.TypeOf(core.SystemSettings.FavPaths).Elem()).Elem()
		item.FieldByName("Name").SetString(filepath.Base(f))
		item.FieldByName("Path").SetString(f)
		fv := reflect.ValueOf(&core.SystemSettings.FavPaths).Elem()
		fv.Set(reflect.Append(fv, item))
	}
}

// testRoot makes a directory with files and directories for the tests.
func testRoot(t *testing.T) string {
	root := cleanDir(t.TempDir()) // as in the file tree
	for _, d := range []string{"docs", "src", ".hidden"} {
		require.NoError(t, os.Mkdir(filepath.Join(root, d), 0750))
	}
	for _, f := range []string{"b.txt", "a.go", ".dotfile", "docs/readme.md"} {
		require.NoError(t, os.WriteFile(filepath.Join(root, f), []byte(f+"\n"), 0644))
	}
	return root
}

// names returns the names of the files of the given table.
func names(dt *DirTable) []string {
	var nms []string
	for _, fi := range dt.files {
		nms = append(nms, fi.Name)
	}
	return nms
}

// TestFiles tests the layout of the panels, the directory tabs, opened by
// double-clicking a directory in the file tree, and the navigation in them.
func TestFiles(t *testing.T) {
	root := testRoot(t)
	testSettings(t, filepath.Join(root, "docs"))
	b := core.NewBody()
	b.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(1200), units.Dp(700)) })
	fl := NewFiles(b).SetRoot(root)
	b.AssertRender(t, "wide", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		assert.Equal(t, 3, fl.Splits().NumChildren())
		require.Equal(t, 1, fl.DirTabs().NumTabs())
		dt := fl.CurrentDirTable()
		assert.Equal(t, root, dt.Dir)
		assert.Equal(t, root, dt.Child(0).(*core.Text).Text)
		assert.Equal(t, []string{"docs", "src", "a.go", "b.txt"}, names(dt))
		_, terminal := fl.Tabs().TabAtIndex(1)
		assert.Equal(t, TerminalTabName, terminal.Text)

		// double-clicking docs in the file tree opens it in a new tab
		var docs *FileNode
		fl.Tree.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
			if fn, ok := cw.(*FileNode); ok && fn.Name == "docs" {
				docs = fn
			}
			return docs == nil
		})
		require.NotNil(t, docs)
		docs.Parts.Send(events.DoubleClick)
		require.Equal(t, 2, fl.DirTabs().NumTabs())
		dt = fl.CurrentDirTable()
		assert.Equal(t, filepath.Join(root, "docs"), dt.Dir)
		assert.Equal(t, []string{"readme.md"}, names(dt))
		_, tab := fl.DirTabs().TabAtIndex(1)
		assert.Equal(t, "docs", tab.Text)

		// and again selects its tab
		fl.DirTabs().SelectTabIndex(0)
		docs.Parts.Send(events.DoubleClick)
		assert.Equal(t, 2, fl.DirTabs().NumTabs())
		assert.Equal(t, dt, fl.CurrentDirTable())

		// opening a directory in the table shows it in the same tab
		fl.DirUp()
		assert.Equal(t, root, dt.Dir)
		assert.Equal(t, filepath.Base(root), tab.Text)
		dt.table.SelectedIndex = 1 // src
		dt.openSelected()
		assert.Equal(t, filepath.Join(root, "src"), dt.Dir)
		assert.Equal(t, "src", tab.Text)
		assert.Equal(t, dt.Dir, dt.Child(0).(*core.Text).Text)

		// back in docs, for the image
		fl.DirUp()
		dt.table.SelectedIndex = 0 // docs
		dt.openSelected()
	})

	Settings.Splits = "Compact"
	b = core.NewBody()
	b.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(1000), units.Dp(700)) })
	fl = NewFiles(b).SetRoot(root)
	b.AssertRender(t, "compact")
}

// TestFavorites tests that selecting a favorite path makes it the root
// of the file tree.
func TestFavorites(t *testing.T) {
	root := testRoot(t)
	docs := filepath.Join(root, "docs")
	testSettings(t, root, docs)
	b := core.NewBody()
	b.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(1200), units.Dp(700)) })
	fl := NewFiles(b).SetRoot(root)
	b.AssertRender(t, "favorites", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		fv := fl.Favorites()
		require.NotNil(t, fv)
		assert.Equal(t, 2, fv.SliceSize)
		fv.SelectIndexEvent(1, events.SelectOne)
		assert.Equal(t, docs, fl.Root)
		assert.Equal(t, docs, string(fl.Tree.Filepath))
	})
}

// TestSplitsSettings tests that the proportions of the splits are saved
// in the settings when they are moved, and restored when the app starts,
// and that they are reset when a layout is chosen.
func TestSplitsSettings(t *testing.T) {
	root := testRoot(t)
	testSettings(t)
	b := core.NewBody()
	fl := NewFiles(b).SetRoot(root)
	b.AssertRender(t, "splits-moved", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		fl.Splits().SetSplits(.3, .4, .3) // as when moved
		fl.LeftSplits().SetSplits(.6, .4)
		fl.saveSplits()
	})
	saved, err := os.ReadFile(Settings.File)
	require.NoError(t, err)
	assert.Contains(t, string(saved), "SplitValues")

	// restored at startup: from the file, in a new window
	Settings.SplitValues, Settings.TreeSplits = nil, nil
	require.NoError(t, tomlx.Open(Settings, Settings.File))
	b = core.NewBody()
	fl = NewFiles(b).SetRoot(root)
	b.AssertRender(t, "splits-restored", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		assert.InDeltaSlice(t, []float32{.3, .4, .3}, fl.Splits().Splits(), .01)
		assert.InDeltaSlice(t, []float32{.6, .4}, fl.LeftSplits().Splits(), .01)

		// choosing a layout resets them to its own
		fl.setSplits("Compact")
		assert.Equal(t, SplitName("Compact"), Settings.Splits)
		assert.InDeltaSlice(t, []float32{.2, .6, .4}, fl.Splits().Splits(), .01)
	})
}
