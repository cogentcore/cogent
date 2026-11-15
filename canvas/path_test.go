// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package canvas

import (
	"image"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/svg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// editPath makes the path with the given name the one edited with the
// node tool.
func editPath(t *testing.T, cv *Canvas, name string) *svg.Path {
	t.Helper()
	cv.SetTool(NodeTool)
	p := node(t, cv, name).(*svg.Path)
	es := &cv.EditState
	es.ResetSelectedNodes()
	es.ActivePath = p
	cv.SVG.UpdateView()
	return p
}

// cmds returns the commands of the nodes of the edited path.
func cmds(cv *Canvas) []Sprites {
	var cs []Sprites
	for _, pn := range cv.EditState.PathNodes {
		cs = append(cs, pn.Cmd)
	}
	return cs
}

// TestPathNodes tests inserting, replacing, and deleting the nodes of a
// path.
func TestPathNodes(t *testing.T) {
	runCanvas(t, "node_paths.svg", func(cv *Canvas) {
		editPath(t, cv, "path236")
		es := &cv.EditState
		require.Equal(t, []Sprites{SpMoveTo, SpLineTo, SpCubeTo, SpCubeTo, SpLineTo}, cmds(cv))
		assert.Equal(t, 8, es.PathNodes[1].EndIndex(), "the command and its points and command")

		cv.InsertLineNode() // at the end
		cv.SVG.UpdateView()
		assert.Equal(t, SpLineTo, cmds(cv)[5])
		cv.InsertCubicNode()
		cv.SVG.UpdateView()
		assert.Equal(t, SpCubeTo, cmds(cv)[6])

		es.NodeSelectAction(1, events.SelectOne)
		assert.True(t, es.HasNodeSelected())
		assert.Equal(t, []int{1}, es.NodeSelectedList())
		cv.InsertBreak() // after the selected one
		assert.Equal(t, SpMoveTo, cmds(cv)[2])
		cv.NodeReplaceCubic()
		assert.Equal(t, SpCubeTo, cmds(cv)[1], "replaced")
		n := len(cmds(cv))
		cv.NodeDelete()
		assert.Len(t, cmds(cv), n-1, "deleted")
		cv.SVG.NodeDeleteLast()
		assert.Len(t, cmds(cv), n-2)

		es.NodeSelectAction(0, events.SelectOne)
		es.NodeSelectAction(1, events.ExtendOne)
		assert.Len(t, es.NodeSelectedList(), 2)
		es.NodeSelectAction(1, events.ExtendOne)
		assert.Len(t, es.NodeSelectedList(), 1, "unselected")
		es.NodeSelectAction(0, events.Unselect)
		assert.False(t, es.HasNodeSelected())
		cv.NodeDelete() // none selected
		cv.NodeReplaceCubic()

		cv.SetTool(SelectTool)
		cv.InsertLineNode() // no path
	})
}

// TestPathQuad tests the nodes of a path with quadratic curves.
func TestPathQuad(t *testing.T) {
	runCanvas(t, "node_paths.svg", func(cv *Canvas) {
		editPath(t, cv, "path346")
		assert.Contains(t, cmds(cv), SpQuadTo)
		assert.Contains(t, cmds(cv), SpClose)
		cv.EditState.NodeSelectAction(2, events.SelectOne) // a quad
		cv.SVG.UpdateView()
		cv.NodeInsert(SpQuadTo)
		assert.Equal(t, SpQuadTo, cmds(cv)[3])
	})
}

// TestPathDrag tests moving a node and a control point of a path by
// dragging them with the node tool.
func TestPathDrag(t *testing.T) {
	var p *svg.Path
	var end, cp image.Point
	var orig []float32
	runCanvas(t, "node_paths.svg", func(cv *Canvas) {
		p = editPath(t, cv, "path236")
		cv.EditState.NodeSelectAction(2, events.SelectOne) // the first curve
		cv.SVG.UpdateView()
		end = cv.EditState.PathNodes[1].TEnd.ToPoint()
		cp = cv.EditState.PathNodes[2].TCp2.ToPoint()
		orig = p.Data.Clone()
	}, unlocked(func(b *core.Body, cv *Canvas) {
		core.DragMouse(b, cp, cp.Add(image.Pt(30, 30)))
	}), func(cv *Canvas) {
		assert.NotEqual(t, orig, []float32(p.Data), "the control point moved")
		pn := cv.EditState.PathNodes[2]
		assert.InDelta(t, float32(cp.X+30), pn.TCp2.X, 10)
		orig = p.Data.Clone()
		cv.EditState.ResetSelectedNodes()
		cv.SVG.UpdateView()
	}, unlocked(func(b *core.Body, cv *Canvas) {
		core.DragMouse(b, end, end.Add(image.Pt(40, 0)))
	}), func(cv *Canvas) {
		pn := cv.EditState.PathNodes[1]
		assert.InDelta(t, float32(end.X+40), pn.TEnd.X, 10, "the node moved")
		assert.Equal(t, "NodeMove", cv.Undo())
		cv.SVG.UpdateView()
		assert.InDeltaSlice(t, orig, []float32(node(t, cv, "path236").(*svg.Path).Data), 0.001, "undo")
	})
}

// TestSelectPath tests selecting a path with the node tool by clicking it.
func TestSelectPath(t *testing.T) {
	var pt image.Point
	runCanvas(t, "node_paths.svg", func(cv *Canvas) {
		cv.SetTool(NodeTool)
		// within the ring of the closed path of a Q (an open path has no
		// area, so only a click exactly on its line selects it)
		p := node(t, cv, "path346").(*svg.Path)
		pt = p.ParentTransform(true).MulPoint(math32.Vec2(167, 135)).ToPoint()
	}, unlocked(func(b *core.Body, cv *Canvas) {
		click(b, pt)
	}), func(cv *Canvas) {
		require.NotNil(t, cv.EditState.ActivePath)
		assert.Equal(t, "path346", cv.EditState.ActivePath.Name)
		assert.Contains(t, cmds(cv), SpQuadTo)
	})
}
