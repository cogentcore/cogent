// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package canvas

import (
	"image"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/svg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestToolKeys tests choosing the tools with their keys in the drawing.
func TestToolKeys(t *testing.T) {
	runCanvas(t, "shapes.svg", func(cv *Canvas) {
		keys := map[rune]Tools{'b': SelBoxTool, 'n': NodeTool, 'r': RectTool, 'e': EllipseTool, 'd': BezierTool, 't': TextTool, 's': SelectTool}
		for _, k := range "bnredts" {
			core.SendKey(cv.SVG, k, 0)
			assert.Equal(t, keys[k], cv.EditState.Tool, string(k))
		}
		cv.SetTool(SelectTool) // same tool
		cv.SetTool(RectTool)
		core.SendKeyFunction(cv.SVG, keymap.Abort)
		assert.Equal(t, SelectTool, cv.EditState.Tool, "abort goes back to select")
	})
}

// TestDrawShapes tests drawing a rectangle, an ellipse, and text by
// dragging with their tools.
func TestDrawShapes(t *testing.T) {
	var nrect, ntext int
	from, to := image.Pt(600, 800), image.Pt(700, 900)
	drag := func(tool Tools) []any {
		from, to := from, to
		return []any{
			func(cv *Canvas) { cv.SetTool(tool) },
			unlocked(func(b *core.Body, cv *Canvas) { core.DragMouse(b, from, to) }),
		}
	}
	fs := []any{func(cv *Canvas) { nrect, ntext = count[*svg.Rect](cv), count[*svg.Text](cv) }}
	fs = append(fs, drag(RectTool)...)
	fs = append(fs, func(cv *Canvas) {
		assert.Equal(t, nrect+1, count[*svg.Rect](cv), "a new rectangle")
		rc, ok := cv.EditState.FirstSelected().(*svg.Rect)
		require.True(t, ok, "the new rectangle is selected")
		assert.InDelta(t, 100, rc.BBox.Size().X, 2, "the size of the drag")
	})
	fs = append(fs, func(cv *Canvas) {
		assert.Equal(t, "NewElement", cv.Undo(), "one undo for a new element")
		assert.Equal(t, nrect, count[*svg.Rect](cv))
		assert.NotEmpty(t, cv.Redo())
	})
	from, to = image.Pt(800, 800), image.Pt(900, 900)
	fs = append(fs, drag(EllipseTool)...)
	fs = append(fs, func(cv *Canvas) {
		assert.Equal(t, 1, count[*svg.Ellipse](cv), "a new ellipse")
	})
	from, to = image.Pt(600, 600), image.Pt(700, 650)
	fs = append(fs, drag(TextTool)...)
	fs = append(fs, func(cv *Canvas) {
		assert.Equal(t, ntext+1, count[*svg.Text](cv), "a new text")
		assert.Equal(t, "NewText", cv.Undo())
		assert.Equal(t, ntext, count[*svg.Text](cv), "undo of the new text")
	})
	runCanvas(t, "shapes.svg", fs...)
}

// TestDrawBezier tests drawing a path by clicking with the bezier tool.
func TestDrawBezier(t *testing.T) {
	var npath int
	runCanvas(t, "shapes.svg", func(cv *Canvas) {
		npath = count[*svg.Path](cv)
		cv.SetTool(BezierTool)
	}, unlocked(func(b *core.Body, cv *Canvas) {
		click(b, image.Pt(600, 800))
		sysEvents(b).MouseMove(image.Pt(650, 820))
		click(b, image.Pt(700, 800))
		click(b, image.Pt(700, 900), key.Alt)
	}), func(cv *Canvas) {
		assert.Equal(t, npath+1, count[*svg.Path](cv), "a new path")
		require.NotNil(t, cv.EditState.ActivePath)
		assert.Len(t, cv.SVG.PathNodes(cv.EditState.ActivePath), 3)
		core.SendKeyFunction(cv.SVG, keymap.Backspace)
		assert.Len(t, cv.SVG.PathNodes(cv.EditState.ActivePath), 2, "the last node deleted")
		core.SendKeyFunction(cv.SVG, keymap.Enter)
		assert.Nil(t, cv.EditState.ActivePath, "done")
	}, unlocked(func(b *core.Body, cv *Canvas) {
		click(b, image.Pt(800, 800))
		click(b, image.Pt(900, 800))
	}), func(cv *Canvas) {
		core.SendKey(cv.SVG, '\r', key.CodeReturnEnter, key.Alt)
		assert.Nil(t, cv.EditState.ActivePath, "closed")
		assert.Equal(t, npath+2, count[*svg.Path](cv))
	}, unlocked(func(b *core.Body, cv *Canvas) {
		click(b, image.Pt(800, 900))
	}), func(cv *Canvas) {
		core.SendKeyFunction(cv.SVG, keymap.Abort)
		assert.Equal(t, npath+2, count[*svg.Path](cv), "a path of one point is deleted on abort")
		assert.Equal(t, SelectTool, cv.EditState.Tool)
	})
}

// TestSelectMouse tests selecting with the mouse: clicking an item,
// extending the selection, dragging a box around items, and moving and
// resizing the selection by dragging.
func TestSelectMouse(t *testing.T) {
	var p318, p847 image.Point
	var bb318 image.Rectangle
	runCanvas(t, "shapes.svg", func(cv *Canvas) {
		p318 = center(node(t, cv, "rect318"))
		bb318 = node(t, cv, "rect318").AsNodeBase().BBox.ToRect()
		p847 = node(t, cv, "rect847").AsNodeBase().BBox.Min.ToPoint().Add(image.Pt(10, 10))
	}, unlocked(func(b *core.Body, cv *Canvas) {
		click(b, p318)
	}), func(cv *Canvas) {
		assert.Equal(t, []string{"rect318"}, cv.EditState.SelectedNames())
	}, unlocked(func(b *core.Body, cv *Canvas) {
		click(b, p847, key.Shift)
	}), func(cv *Canvas) {
		assert.Len(t, cv.EditState.SelectedNames(), 2, "extended")
	}, unlocked(func(b *core.Body, cv *Canvas) {
		click(b, image.Pt(450, 900)) // off
	}), func(cv *Canvas) {
		assert.False(t, cv.EditState.HasSelected())
	}, unlocked(func(b *core.Body, cv *Canvas) {
		core.DragMouse(b, image.Pt(850, 700), image.Pt(600, 290)) // rubber band
	}), func(cv *Canvas) {
		assert.Contains(t, cv.EditState.SelectedNames(), "rect318", "box selected")
		cv.SetTool(SelBoxTool)
	}, unlocked(func(b *core.Body, cv *Canvas) {
		core.DragMouse(b, image.Pt(850, 700), image.Pt(600, 290))
	}), func(cv *Canvas) {
		assert.Contains(t, cv.EditState.SelectedNames(), "rect318", "box tool selected")
		cv.SetTool(SelectTool)
		selectNodes(t, cv, "rect318")
	}, unlocked(func(b *core.Body, cv *Canvas) {
		core.DragMouse(b, p318, p318.Add(image.Pt(100, 0)))
	}), func(cv *Canvas) {
		bb := node(t, cv, "rect318").AsNodeBase().BBox.ToRect()
		assert.InDelta(t, bb318.Min.X+100, bb.Min.X, 15, "moved")
		bb318 = bb
	}, unlocked(func(b *core.Body, cv *Canvas) {
		pt := bb318.Max.Add(image.Pt(4, 4)) // the reshape handle at the lower right
		core.DragMouse(b, pt, pt.Add(image.Pt(50, 50)))
	}), func(cv *Canvas) {
		bb := node(t, cv, "rect318").AsNodeBase().BBox.ToRect()
		assert.Greater(t, bb.Dx(), bb318.Dx()+20, "resized")
	})
}
