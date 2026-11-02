// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"os"
	"path/filepath"
	"testing"

	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTestFile(t *testing.T, path, text string) {
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(text), 0o644))
}

func readTestFile(t *testing.T, path string) string {
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(b)
}

// TestDirTableDND tests dropping files on a directory table, as in
// [core.FilePicker], and finding the file of the row being dragged.
func TestDirTableDND(t *testing.T) {
	testSettings(t)
	dir := cleanDir(t.TempDir())
	other := cleanDir(t.TempDir())
	writeTestFile(t, filepath.Join(dir, "a.txt"), "a\n")
	writeTestFile(t, filepath.Join(other, "a.txt"), "other a\n")
	writeTestFile(t, filepath.Join(other, "b.txt"), "b\n")
	writeTestFile(t, filepath.Join(other, "d", "c.txt"), "c\n")
	writeTestFile(t, filepath.Join(other, "m.txt"), "m\n")

	b := core.NewBody()
	b.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(800), units.Dp(600)) })
	dt := NewDirTable(b).SetDir(dir)
	drop := func(mod events.DropMods, paths ...string) {
		var md mimedata.Mimes
		for _, p := range paths {
			md = append(md, mimedata.NewTextData(p))
		}
		de := &events.DragDrop{DropMod: mod, Data: md}
		de.Typ = events.Drop
		dt.drop(de)
	}
	b.AssertRender(t, "dnd", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()

		// copy a file and a directory
		drop(events.DropCopy, filepath.Join(other, "b.txt"), filepath.Join(other, "d"))
		assert.Equal(t, "b\n", readTestFile(t, filepath.Join(dir, "b.txt")))
		assert.Equal(t, "c\n", readTestFile(t, filepath.Join(dir, "d", "c.txt")))
		assert.FileExists(t, filepath.Join(other, "b.txt"))
		assert.Equal(t, []string{"d", "a.txt", "b.txt"}, names(dt))

		// move
		drop(events.DropMove, filepath.Join(other, "m.txt"))
		assert.Equal(t, "m\n", readTestFile(t, filepath.Join(dir, "m.txt")))
		assert.NoFileExists(t, filepath.Join(other, "m.txt"))

		// an existing file is not overwritten without asking
		drop(events.DropCopy, filepath.Join(other, "a.txt"))
		assert.Equal(t, "a\n", readTestFile(t, filepath.Join(dir, "a.txt")))

		// a file in the directory itself is left alone
		drop(events.DropMove, filepath.Join(dir, "a.txt"))
		assert.Equal(t, "a\n", readTestFile(t, filepath.Join(dir, "a.txt")))
	}, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		// the file of a row being dragged is that of the widget there
		dt.table.Update()
		for _, name := range []string{"d", "b.txt"} {
			var w core.Widget
			dt.table.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
				if tf, ok := cw.(*core.TextField); ok && tf.Text() == name {
					w = cw
				}
				return w == nil
			})
			require.NotNil(t, w, name)
			idx, rw := dt.rowAt(w.AsWidget().Geom.TotalBBox.Min.Add(w.AsWidget().Geom.TotalBBox.Size().Div(2)))
			require.GreaterOrEqual(t, idx, 0, name)
			assert.Equal(t, name, dt.files[idx].Name)
			assert.NotNil(t, rw)
		}
		idx, _ := dt.rowAt(dt.Geom.TotalBBox.Max.Sub(dt.Geom.TotalBBox.Size().Div(20)))
		assert.Equal(t, -1, idx, "below the rows")
	})
}

func TestMovePath(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "d", "x.txt"), "x\n")
	require.NoError(t, movePath(filepath.Join(dir, "e"), filepath.Join(dir, "d")))
	assert.Equal(t, "x\n", readTestFile(t, filepath.Join(dir, "e", "x.txt")))
	assert.NoDirExists(t, filepath.Join(dir, "d"))
}
