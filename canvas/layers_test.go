// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package canvas

import (
	"image"
	"testing"

	"cogentcore.org/core/events"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/svg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// treeNode returns the tree node of the canvas for the given svg node.
func treeNode(t *testing.T, cv *Canvas, name string) *Tree {
	t.Helper()
	tn := cv.tree.FindSyncNode(node(t, cv, name))
	require.NotNil(t, tn, name)
	return tn.This.(*Tree)
}

// TestLayers tests the layers of a drawing: the list of them, setting the
// current one, locking them, hiding them, and adding them.
func TestLayers(t *testing.T) {
	runCanvas(t, "seplays.svg", func(cv *Canvas) {
		es := &cv.EditState
		require.Len(t, es.Layers, 2)
		assert.Equal(t, "LayerBG", es.Layers[0].Name)
		assert.True(t, es.Layers[0].Vis)
		assert.Equal(t, 1, cv.FirstLayerIndex(), "after the metadata")
		assert.True(t, NodeIsLayer(node(t, cv, "Layer1")))
		assert.Equal(t, node(t, cv, "Layer1"), NodeParentLayer(node(t, cv, "ellipse318")))
		assert.Nil(t, NodeParentLayer(cv.SVG.Root()))

		bg := treeNode(t, cv, "LayerBG")
		bg.LayerSetCurrent()
		assert.Equal(t, "LayerBG", es.CurLayer)
		assert.True(t, bg.LayerIsCurrent())
		treeNode(t, cv, "Layer1").LayerSetCurrent()
		assert.Equal(t, "Layer1", es.CurLayer, "the current layer changes")

		l1 := treeNode(t, cv, "Layer1")
		l1.LayerToggleLock()
		assert.True(t, LayerIsLocked(l1.SyncNode))
		assert.Equal(t, "LayerBG", es.CurLayer, "a locked layer is not current")
		ebb := node(t, cv, "ellipse318").AsNodeBase().BBox // outside of the rectangle
		assert.Nil(t, cv.SVG.SelectContainsPoint(image.Pt(int(ebb.Center().X), int(ebb.Max.Y)-5), false, false), "a locked layer can not be selected")
		l1.LayerToggleVis()
		assert.False(t, LayerIsVisible(l1.SyncNode))
		l1.LayerSetCurrent()
		assert.False(t, LayerIsLocked(l1.SyncNode), "the current layer is unlocked")
		assert.True(t, LayerIsVisible(l1.SyncNode), "and visible")
		l1.LayerToggleVis()
		l1.LayerToggleVis()
		assert.True(t, LayerIsVisible(l1.SyncNode))

		es.Layers[1].Vis = false
		es.Layers[1].Lock = true
		cv.SyncLayers()
		assert.False(t, LayerIsVisible(node(t, cv, "Layer1")), "from the layer list")
		assert.True(t, LayerIsLocked(node(t, cv, "Layer1")))

		cv.AddLayer()
		assert.Len(t, es.Layers, 3)
		assert.Equal(t, "Layer2", es.CurLayer)
		cv.SVG.NewParent(false)
		el := NewSVGElement[svg.Rect](cv.SVG, false)
		assert.Equal(t, "Layer2", el.AsTree().Parent.AsTree().Name, "new elements go in the current layer")
	})
}

// TestAddLayerFirst tests adding a layer to a drawing without layers,
// which puts its items in a new layer.
func TestAddLayerFirst(t *testing.T) {
	runCanvas(t, "shapes.svg", func(cv *Canvas) {
		es := &cv.EditState
		assert.Empty(t, es.Layers)
		cv.AddLayer()
		require.Len(t, es.Layers, 2)
		assert.Equal(t, "Layer1", es.CurLayer)
		assert.True(t, NodeIsLayer(node(t, cv, "rect847").AsTree().Parent), "the items are in the layer")
	})
}

// TestLayersUnique tests making the names of layers unique.
func TestLayersUnique(t *testing.T) {
	ly := Layers{{Name: "a"}, {Name: "b"}, {Name: "a"}}
	ly.UniqueNames()
	assert.Equal(t, "Layer0", ly[0].Name)
	assert.Equal(t, "b", ly[1].Name)
	assert.Equal(t, "Layer2", ly[2].Name)
	assert.Equal(t, 1, ly.LayerIndexByName("b"))
	assert.Equal(t, -1, ly.LayerIndexByName("x"))
}

// TestAlign tests aligning the selection in each way.
func TestAlign(t *testing.T) {
	runCanvas(t, "shapes.svg", func(cv *Canvas) {
		sv := cv.SVG
		bb := func(nm string) math32.Box2 { return node(t, cv, nm).AsNodeBase().BBox }
		sel := func() {
			cv.EditState.ResetSelected()
			cv.EditState.Select(node(t, cv, "rect847")) // first: the anchor
			cv.EditState.Select(node(t, cv, "rect59"))
			sv.UpdateView()
		}
		for _, al := range AlignsValues() {
			sel()
			anc := bb("rect847")
			sv.Align(AlignFirst, al)
			sv.UpdateView()
			b := bb("rect59")
			switch al {
			case AlignLeft:
				assert.InDelta(t, anc.Min.X, b.Min.X, 1, al.String())
			case AlignRight:
				assert.InDelta(t, anc.Max.X, b.Max.X, 1, al.String())
			case AlignCenter:
				assert.InDelta(t, anc.Center().X, b.Center().X, 1, al.String())
			case AlignTop:
				assert.InDelta(t, anc.Min.Y, b.Min.Y, 1, al.String())
			case AlignBottom:
				assert.InDelta(t, anc.Max.Y, b.Max.Y, 1, al.String())
			case AlignMiddle:
				assert.InDelta(t, anc.Center().Y, b.Center().Y, 1, al.String())
			case AlignRightAnchor:
				assert.InDelta(t, anc.Min.X, b.Max.X, 1, al.String())
			case AlignLeftAnchor:
				assert.InDelta(t, anc.Max.X, b.Min.X, 1, al.String())
			case AlignBottomAnchor:
				assert.InDelta(t, anc.Min.Y, b.Max.Y, 1, al.String())
			case AlignTopAnchor:
				assert.InDelta(t, anc.Max.Y, b.Min.Y, 1, al.String())
			}
			assert.Equal(t, anc, bb("rect847"), "the anchor does not move")
			cv.Undo()
		}
		sel()
		last := bb("rect59")
		sv.Align(AlignLast, AlignLeft)
		sv.UpdateView()
		assert.InDelta(t, last.Min.X, bb("rect847").Min.X, 1, "the last selected is the anchor")
		sel()
		sv.Align(AlignDrawing, AlignLeft)
		sv.UpdateView()
		root := sv.Root()
		assert.InDelta(t, root.Paint.Transform.X0, bb("rect59").Min.X, 1, "the left of the page")
		sv.Align(AlignDrawing, AlignBottom)
		sv.UpdateView()
		assert.InDelta(t, root.Paint.Transform.MulPoint(root.ViewBox.Size).Y, bb("rect59").Max.Y, 1, "the bottom of the page")
		sel()
		sv.Align(AlignSelectBox, AlignTop)
		cv.EditState.ResetSelected()
		sv.Align(AlignFirst, AlignTop) // none selected
	})
}

// TestTree tests the functions of the tree of the drawing.
func TestTree(t *testing.T) {
	runCanvas(t, "shapes.svg", func(cv *Canvas) {
		tn := treeNode(t, cv, "rect59")
		tn.SelectSVG()
		assert.Equal(t, []string{"rect59"}, cv.EditState.SelectedNames())
		cv.SelectNodeInTree(node(t, cv, "rect318"), events.SelectOne)
		assert.Len(t, cv.AnySelectedNodes(), 1)
		assert.Equal(t, "rect318", cv.AnySelectedNodes()[0].AsTree().Name, "the tree selection first")
		n0 := len(cv.SVG.Root().Children)
		treeNode(t, cv, "rect318").NewGroup()
		assert.Len(t, cv.SVG.Root().Children, n0+1)
		assert.True(t, NodeIsMetaData(cv.SVG.Root().Child(0)))
		cv.UpdateAll()
	})
}
