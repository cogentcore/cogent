// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package canvas

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// canvasWindow returns the canvas of the window with the given name,
// waiting for it to be shown.
func canvasWindow(t *testing.T, name string) *Canvas {
	t.Helper()
	var cv *Canvas
	require.Eventually(t, func() bool {
		w := core.AllRenderWindows.FindName(name)
		if w == nil {
			return false
		}
		cv = tree.ChildByType[*Canvas](w.MainScene().Body)
		return cv != nil && cv.SVG != nil
	}, 5*time.Second, 10*time.Millisecond, name)
	return cv
}

// TestNewWindow tests opening a drawing in a new window, again (which
// raises it), and a new drawing.
func TestNewWindow(t *testing.T) {
	defer core.TrackWindows()()
	wd, err := os.Getwd()
	require.NoError(t, err)
	t.Chdir(wd) // restored at the end: opening changes it
	// in a directory with a unique name, as windows are named by the
	// directory and file (and those of t.TempDir are all named 001)
	dir := filepath.Join(t.TempDir(), "new-window")
	require.NoError(t, os.Mkdir(dir, 0o755))
	fn := filepath.Join(dir, "seplays.svg")
	require.NoError(t, os.Rename(copyTestdata(t, "seplays.svg"), fn))
	cv := NewWindow(fn)
	require.Eventually(t, func() bool { // opened when shown
		cv.AsyncLock()
		defer cv.AsyncUnlock()
		return cv.Filename != ""
	}, 5*time.Second, 10*time.Millisecond)
	os.Chdir(wd)
	assert.Same(t, cv, NewWindow(fn), "the same window")
	assert.Same(t, cv, canvasWindow(t, "Cogent Canvas • "+fsx.DirAndFile(fn)))
	core.Locked(cv, func() {
		assert.NotEmpty(t, cv.SVG.Root().Children)
	})()

	sz := Settings.Size
	nd := NewDrawing(sz)
	require.NotNil(t, nd)
	assert.NotSame(t, nd, NewDrawing(sz), "a new drawing is in a new window")
	core.Locked(nd, func() {
		assert.Equal(t, sz.Size, nd.SVG.Root().ViewBox.Size)
		nd.PromptPhysicalSize()
	})()
	os.Chdir(wd)
}

// TestDialogs tests the dialogs of the canvas: the physical size, the
// recent paths, and editing a node.
func TestDialogs(t *testing.T) {
	defer core.TrackWindows()()
	runCanvas(t, "shapes.svg", func(cv *Canvas) {
		cv.PromptPhysicalSize()
	}, func(cv *Canvas) {
		bt := core.ButtonByText(core.TopScene(cv.Scene.Body), "OK")
		require.NotNil(t, bt)
		bt.Send(events.Click)
	}, func(cv *Canvas) {
		cv.EditRecentPaths()
	}, func(cv *Canvas) {
		assert.NotNil(t, core.ButtonByText(core.TopScene(cv.Scene.Body), "OK"))
		cv.SVG.EditNode(node(t, cv, "rect59"))
	})
}

// TestOpenDropped tests opening an svg file dropped on the canvas in a
// new window.
func TestOpenDropped(t *testing.T) {
	defer core.TrackWindows()()
	wd, err := os.Getwd()
	require.NoError(t, err)
	t.Chdir(wd)
	fn := copyTestdata(t, "arrow.svg")
	n := len(core.AllRenderWindows)
	runCanvas(t, "shapes.svg", func(cv *Canvas) {
		md := mimedata.NewText("file://" + fn)
		md = append(md, mimedata.NewText("file:///nonexistent.txt")...)
		cv.HandleEvent(events.NewExternalDrop(events.Drop, events.Left, cv.Geom.ContentBBox.Min, 0, md))
		assert.Greater(t, len(core.AllRenderWindows), n, "a new window")
	})
	os.Chdir(wd)
	_ = fileinfo.TextPlain
}

// TestAlignView tests the buttons of the align tab.
func TestAlignView(t *testing.T) {
	runCanvas(t, "shapes.svg", func(cv *Canvas) {
		cv.tabs.SelectTabByName("Align")
		selectNodes(t, cv, "rect847", "rect59")
	}, func(cv *Canvas) {
		av := cv.Tab("Align").Child(0).(*AlignView)
		ch := tree.ChildByType[*core.Chooser](av.Child(0))
		ch.SetCurrentValue(AlignLast)
		ch.SendChange()
		assert.Equal(t, AlignLast, av.Anchor)
		bt := av.Child(1).AsTree().ChildByName(AlignLeft.String()).(*core.Button)
		bt.Send(events.Click)
		assert.Equal(t, AlignLeft.String(), cv.Undo())
	})
}

// TestDashChooser tests choosing a dash pattern for the selection.
func TestDashChooser(t *testing.T) {
	runCanvas(t, "shapes.svg", func(cv *Canvas) {
		selectNodes(t, cv, "rect59")
		pv := cv.PaintSetter()
		pv.dashes.SetCurrentIndex(2)
		pv.dashes.SendChange()
		assert.NotEmpty(t, prop(t, cv, "rect59", "stroke-dasharray"))
		pv.dashes.SetCurrentIndex(0)
		pv.dashes.SendChange()
		assert.Empty(t, prop(t, cv, "rect59", "stroke-dasharray"))
	})
}
