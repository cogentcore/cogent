// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package canvas

import (
	"image"
	"os"
	"path/filepath"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/svg"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// copyTestdata copies the given testdata file to a temporary directory,
// so that saving and autosaving do not write to testdata, and returns
// its path there.
func copyTestdata(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	fn := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(fn, b, 0o644))
	return fn
}

// newCanvas returns a new body with a canvas and its toolbar.
func newCanvas() (*core.Body, *Canvas) {
	b := core.NewBody()
	cv := NewCanvas(b)
	b.AddTopBar(func(bar *core.Frame) {
		core.NewToolbar(bar).Maker(cv.MakeToolbar)
	})
	return b, cv
}

// openDrawing opens the given drawing in the given canvas, keeping the
// current directory, which opening changes to that of the drawing.
func openDrawing(t *testing.T, cv *Canvas, fn string) {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, cv.OpenDrawing(core.Filename(fn)))
	require.NoError(t, os.Chdir(wd))
}

// unlocked is a function for [runCanvas] that is called without the
// lock of the window, as for sending system events.
type unlocked func(b *core.Body, cv *Canvas)

// runCanvas opens a copy of the given testdata drawing in a new canvas
// in a window, and calls each function after the drawing is shown, with
// the lock of the window held for a func(cv *Canvas), and without it for
// an [unlocked] function, waiting for the events after each.
func runCanvas(t *testing.T, name string, funs ...any) *Canvas {
	t.Helper()
	b, cv := newCanvas()
	fn := copyTestdata(t, name)
	lf := []func(){core.Locked(b, func() { openDrawing(t, cv, fn) })}
	for _, f := range funs {
		switch f := f.(type) {
		case func(cv *Canvas):
			lf = append(lf, core.Locked(b, func() { f(cv) }))
		case unlocked:
			lf = append(lf, func() { f(b, cv) })
		default:
			t.Fatalf("runCanvas: invalid function type %T", f)
		}
	}
	lf = append(lf, core.Locked(b, func() { closeSaved(cv) }))
	core.RenderImage(b, lf...)
	return cv
}

// closeSaved marks the drawing of the canvas as saved, so that its window
// closes without the dialog for unsaved changes, which would keep it open.
func closeSaved(cv *Canvas) {
	cv.EditState.Changed = false
}

// events returns the source of the system events of the window of the body.
func sysEvents(b *core.Body) *events.Source {
	return b.Scene.RenderWindow().SystemWindow.Events()
}

// click clicks the left mouse button at the given point in the window.
func click(b *core.Body, pt image.Point, mods ...key.Modifiers) {
	var m key.Modifiers
	for _, mod := range mods {
		m.SetFlag(true, mod)
	}
	ev := sysEvents(b)
	ev.MouseMove(pt)
	ev.MouseButton(events.MouseDown, events.Left, pt, m)
	ev.MouseButton(events.MouseUp, events.Left, pt, m)
}

// center returns the center of the bounding box of the given node, in
// window coordinates.
func center(n svg.Node) image.Point {
	return n.AsNodeBase().BBox.Center().ToPoint()
}

// count returns the number of nodes of the given type in the drawing.
func count[T any](cv *Canvas) int {
	n := 0
	svg.SVGWalkDownNoDefs(cv.SVG.Root(), func(sn svg.Node, nb *svg.NodeBase) bool {
		if _, ok := sn.(T); ok {
			n++
		}
		return true
	})
	return n
}

// node returns the svg node with the given name in the drawing.
func node(t *testing.T, cv *Canvas, name string) svg.Node {
	t.Helper()
	n := cv.SSVG().FindNamedElement(name)
	if n == nil {
		var names []string
		for k := range cv.SVG.Root().All() {
			names = append(names, k.AsTree().Name)
		}
		require.NotNil(t, n, "%s not in %v", name, names)
	}
	return n
}

// selectNodes selects the svg nodes with the given names.
func selectNodes(t *testing.T, cv *Canvas, names ...string) {
	t.Helper()
	es := &cv.EditState
	es.ResetSelected()
	for _, nm := range names {
		es.Select(node(t, cv, nm))
	}
	cv.SVG.UpdateSelect()
	cv.SVG.UpdateView()
}

// TestCanvasRender tests rendering a drawing in the canvas.
func TestCanvasRender(t *testing.T) {
	b, cv := newCanvas()
	fn := copyTestdata(t, "shapes.svg")
	core.AssertRender(t, b, "shapes", core.Locked(b, func() {
		openDrawing(t, cv, fn)
		cv.SVG.Root().SetName("svg") // the default name depends on the test order
		cv.UpdateTree()
		cv.SetStatus("")
	}))
}

// TestOpenAll tests opening each of the drawings in testdata.
func TestOpenAll(t *testing.T) {
	fs, err := filepath.Glob("testdata/*.svg")
	require.NoError(t, err)
	var opens []any
	for _, f := range fs {
		name := filepath.Base(f)
		if name[0] == '#' || name == "tmp.svg" {
			continue
		}
		fn := copyTestdata(t, name)
		opens = append(opens, func(cv *Canvas) {
			openDrawing(t, cv, fn)
			assert.Equal(t, name, filepath.Base(string(cv.Filename)))
			assert.NotEmpty(t, cv.SVG.Root().Children, name)
			cv.SVG.ResizeToContents(true)
			cv.UpdateAll()
		})
	}
	runCanvas(t, "shapes.svg", opens...)
}

// TestSelectArrange tests grouping, ungrouping, and changing the order of
// the selected items, and undoing and redoing those changes.
func TestSelectArrange(t *testing.T) {
	runCanvas(t, "shapes.svg", func(cv *Canvas) {
		root := cv.SVG.Root()
		idx := func(nm string) int { return node(t, cv, nm).AsTree().IndexInParent() }
		selectNodes(t, cv, "rect847", "rect59")
		assert.Len(t, cv.EditState.SelectedNames(), 2)
		n0 := len(root.Children)
		cv.SelectGroup()
		assert.Len(t, root.Children, n0-1, "two items in a group")
		gp := cv.EditState.FirstSelected()
		require.IsType(t, &svg.Group{}, gp)
		assert.Len(t, gp.AsTree().Children, 2)

		cv.SelectUnGroup()
		assert.Len(t, root.Children, n0, "ungrouped")
		assert.Nil(t, cv.SSVG().FindNamedElement(gp.AsTree().Name), "the empty group is removed")

		selectNodes(t, cv, "rect847")
		cv.SelectRaiseTop()
		assert.Equal(t, n0-1, idx("rect847"))
		cv.SelectLowerBottom()
		assert.Equal(t, 0, idx("rect847"))
		cv.SelectRaise()
		assert.Equal(t, 1, idx("rect847"))
		cv.SelectLower()
		assert.Equal(t, 0, idx("rect847"))

		assert.NotEmpty(t, cv.Undo())
		assert.Equal(t, 1, idx("rect847"), "undo of lower")
		assert.NotEmpty(t, cv.Redo())
		assert.Equal(t, 0, idx("rect847"), "redo of lower")
	})
}

// TestSelectTransform tests rotating, flipping, and scaling the selection.
func TestSelectTransform(t *testing.T) {
	runCanvas(t, "shapes.svg", func(cv *Canvas) {
		selectNodes(t, cv, "rect318")
		rc := node(t, cv, "rect318").(*svg.Rect)
		bb := rc.BBox
		cv.SelectRotateRight()
		cv.SVG.UpdateView()
		assert.InDelta(t, bb.Size().X, rc.BBox.Size().Y, 2, "rotated 90 degrees")
		cv.SelectRotateLeft()
		cv.SVG.UpdateView()
		assert.InDelta(t, bb.Size().X, rc.BBox.Size().X, 2, "rotated back")
		cv.SelectFlipHorizontal()
		cv.SelectFlipVertical()
		cv.SVG.UpdateView()
		assert.InDelta(t, bb.Min.X, rc.BBox.Min.X, 2, "flipped in place")
		cv.SelectScale(2, 2)
		cv.SVG.UpdateView()
		assert.InDelta(t, 2*bb.Size().X, rc.BBox.Size().X, 3, "scaled")
	})
}

// TestClipboard tests duplicating, copying, cutting, pasting, and
// deleting the selection.
func TestClipboard(t *testing.T) {
	runCanvas(t, "shapes.svg", func(cv *Canvas) {
		root := cv.SVG.Root()
		n0 := len(root.Children)
		selectNodes(t, cv, "rect59")
		cv.DuplicateSelected()
		assert.Len(t, root.Children, n0+1, "duplicated")
		selectNodes(t, cv, "rect59")
		cv.CopySelected()
		cv.PasteClip()
		assert.Len(t, root.Children, n0+2, "pasted")
		selectNodes(t, cv, "rect59")
		cv.CutSelected()
		assert.Len(t, root.Children, n0+1, "cut")
		assert.Nil(t, cv.SSVG().FindNamedElement("rect59"))
		selectNodes(t, cv, "rect318")
		cv.DeleteSelected()
		assert.Len(t, root.Children, n0, "deleted")
		cv.DeleteItems(node(t, cv, "rect847"))
		assert.Len(t, root.Children, n0-1, "deleted items")

		cv.EditState.ResetSelected()
		cv.DuplicateSelected() // none selected
		cv.CopySelected()
		cv.CutSelected()
		cv.DeleteSelected()
		assert.Len(t, root.Children, n0-1)
	})
}

// TestSelectModes tests the modes of selecting items.
func TestSelectModes(t *testing.T) {
	runCanvas(t, "shapes.svg", func(cv *Canvas) {
		es := &cv.EditState
		a, b := node(t, cv, "rect59"), node(t, cv, "rect318")
		es.ResetSelected()
		es.SelectAction(a, events.SelectOne, image.Point{})
		es.SelectAction(b, events.ExtendOne, image.Point{})
		assert.Equal(t, []string{"rect59", "rect318"}, es.SelectedNames())
		assert.Equal(t, "rect59 rect318", es.SelectedNamesString())
		es.SelectAction(b, events.ExtendContinuous, image.Point{})
		assert.Equal(t, []string{"rect59"}, es.SelectedNames(), "toggled off")
		es.SelectAction(b, events.SelectQuiet, image.Point{})
		es.SelectAction(a, events.UnselectQuiet, image.Point{})
		assert.Equal(t, []string{"rect318"}, es.SelectedNames())
		es.SelectAction(b, events.Unselect, image.Point{})
		assert.False(t, es.HasSelected())
		es.SelectAction(a, events.NoSelect, image.Point{})
		assert.False(t, es.HasSelected())
		es.SelectAction(a, events.SelectOne, image.Point{})
		es.SelectAction(b, events.ExtendOne, image.Point{})
		es.SelectAction(a, events.SelectOne, image.Point{}) // of a selected one
		assert.Equal(t, []string{"rect59"}, es.SelectedNames())
		assert.Contains(t, es.RecentlySelected, b)
		es.StartRecents(image.Pt(10, 10))
		assert.True(t, es.PosInLastSelect(image.Pt(11, 9)))
		assert.False(t, es.PosInLastSelect(image.Pt(30, 9)))

		gp := svg.NewGroup()
		cv.SVG.Root().AddChild(gp)
		es.ResetSelected()
		es.Select(gp)
		tree.MoveToParent(b, gp)
		es.Select(b)
		assert.Equal(t, []string{"rect318"}, es.SelectedNames(), "not both a group and its item")
	})
}
