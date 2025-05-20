// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package canvas

import (
	"cogentcore.org/cogent/canvas/cicons"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/svg"
	"cogentcore.org/core/tree"
)

// Tools are the drawing tools
type Tools int //enums:enum

const (
	SelectTool Tools = iota
	NodeTool
	RectTool
	EllipseTool
	BezierTool
	TextTool
)

// ToolDoesBasicSelect returns true if tool should do select for clicks
func ToolDoesBasicSelect(tl Tools) bool {
	return tl != NodeTool
}

// SetTool sets the current active tool
func (cv *Canvas) SetTool(tl Tools) {
	es := &cv.EditState
	if es.Tool == tl {
		return
	}
	fs := es.FirstSelectedNode()
	if fs != nil {
		switch v := fs.(type) {
		case *svg.Text:
			Settings.TextStyle.CopyStyleFrom(&v.Paint)
		case *svg.Line:
			Settings.LineStyle.CopyStyleFrom(&v.Paint)
		case *svg.Path:
			Settings.PathStyle.CopyStyleFrom(&v.Paint)
		default:
			gg := fs.AsNodeBase()
			Settings.ShapeStyle.CopyStyleFrom(&gg.Paint)
		}
	}
	es.ResetSelected()
	cv.EditState.Tool = tl
	cv.SetDefaultStyle()
	cv.modalTools.Update()
	cv.SetStatus("Tool")
	cv.Restyle()
	cv.SVG.UpdateSelect()
}

func (cv *Canvas) MakeTools(p *tree.Plan) {
	tree.Add(p, func(w *core.Button) {
		w.SetIcon(icons.ArrowSelectorTool).SetShortcut("S")
		w.SetTooltip("Select, move, and resize objects")
		w.OnClick(func(e events.Event) {
			cv.SetTool(SelectTool)
		})
		w.Styler(func(s *styles.Style) {
			s.SetState(cv.EditState.Tool == SelectTool, states.Selected)
		})
	})
	tree.Add(p, func(w *core.Button) {
		w.SetIcon(cicons.ToolNode).SetShortcut("N")
		w.SetTooltip("Select and move node points within paths")
		w.OnClick(func(e events.Event) {
			cv.SetTool(NodeTool)
		})
		w.Styler(func(s *styles.Style) {
			s.SetState(cv.EditState.Tool == NodeTool, states.Selected)
		})
	})
	tree.Add(p, func(w *core.Button) {
		w.SetIcon(icons.Rectangle).SetShortcut("R")
		w.SetTooltip("Create rectangles and squares")
		w.OnClick(func(e events.Event) {
			cv.SetTool(RectTool)
		})
		w.Styler(func(s *styles.Style) {
			s.SetState(cv.EditState.Tool == RectTool, states.Selected)
		})
	})
	tree.Add(p, func(w *core.Button) {
		w.SetIcon(icons.Circle).SetShortcut("E")
		w.SetTooltip("Create circles, ellipses, and arcs")
		w.OnClick(func(e events.Event) {
			cv.SetTool(EllipseTool)
		})
		w.Styler(func(s *styles.Style) {
			s.SetState(cv.EditState.Tool == EllipseTool, states.Selected)
		})
	})
	tree.Add(p, func(w *core.Button) {
		w.SetIcon(icons.LineCurve).SetShortcut("B")
		w.SetTooltip("Create bezier curves (straight lines and curves with control points)")
		w.OnClick(func(e events.Event) {
			cv.SetTool(BezierTool)
		})
		w.Styler(func(s *styles.Style) {
			s.SetState(cv.EditState.Tool == BezierTool, states.Selected)
		})
	})
	tree.Add(p, func(w *core.Button) {
		w.SetIcon(cicons.ToolText).SetShortcut("T")
		w.SetTooltip("Add and edit text")
		w.OnClick(func(e events.Event) {
			cv.SetTool(TextTool)
		})
		w.Styler(func(s *styles.Style) {
			s.SetState(cv.EditState.Tool == TextTool, states.Selected)
		})
	})
}
