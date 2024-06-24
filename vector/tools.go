// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vector

import (
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
func (vc *Vector) SetTool(tl Tools) {
	es := &vc.EditState
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
	vc.EditState.Tool = tl
	vc.SetDefaultStyle()
	vc.ModalToolbar().Update()
	vc.SetStatus("Tool")
	vc.Restyle()
	sv := vc.SVG()
	sv.UpdateSelect()
}

func (vc *Vector) MakeTools(p *tree.Plan) {
	tree.Add(p, func(w *core.Button) {
		w.SetIcon(icons.ArrowSelectorTool).SetShortcut("S")
		w.SetTooltip("Select, move, and resize objects")
		w.OnClick(func(e events.Event) {
			vc.SetTool(SelectTool)
		})
		w.Styler(func(s *styles.Style) {
			s.SetState(vc.EditState.Tool == SelectTool, states.Selected)
		})
	})
	tree.Add(p, func(w *core.Button) {
		w.SetIcon("tool-node").SetShortcut("N")
		w.SetTooltip("Select and move node points within paths")
		w.OnClick(func(e events.Event) {
			vc.SetTool(NodeTool)
		})
		w.Styler(func(s *styles.Style) {
			s.SetState(vc.EditState.Tool == NodeTool, states.Selected)
		})
	})
	tree.Add(p, func(w *core.Button) {
		w.SetIcon(icons.Rectangle).SetShortcut("R")
		w.SetTooltip("Create rectangles and squares")
		w.OnClick(func(e events.Event) {
			vc.SetTool(RectTool)
		})
		w.Styler(func(s *styles.Style) {
			s.SetState(vc.EditState.Tool == RectTool, states.Selected)
		})
	})
	tree.Add(p, func(w *core.Button) {
		w.SetIcon(icons.Circle).SetShortcut("E")
		w.SetTooltip("Create circles, ellipses, and arcs")
		w.OnClick(func(e events.Event) {
			vc.SetTool(EllipseTool)
		})
		w.Styler(func(s *styles.Style) {
			s.SetState(vc.EditState.Tool == EllipseTool, states.Selected)
		})
	})
	tree.Add(p, func(w *core.Button) {
		w.SetIcon(icons.LineCurve).SetShortcut("B")
		w.SetTooltip("Create bezier curves (straight lines and curves with control points)")
		w.OnClick(func(e events.Event) {
			vc.SetTool(BezierTool)
		})
		w.Styler(func(s *styles.Style) {
			s.SetState(vc.EditState.Tool == BezierTool, states.Selected)
		})
	})
	tree.Add(p, func(w *core.Button) {
		w.SetIcon("tool-text").SetShortcut("T")
		w.SetTooltip("Add and edit text")
		w.OnClick(func(e events.Event) {
			vc.SetTool(TextTool)
		})
		w.Styler(func(s *styles.Style) {
			s.SetState(vc.EditState.Tool == TextTool, states.Selected)
		})
	})
}
