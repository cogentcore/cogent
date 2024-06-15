// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vector

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/svg"
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
func (gv *Vector) SetTool(tl Tools) {
	es := &gv.EditState
	if es.Tool == tl {
		return
	}
	tls := gv.Tools()
	for i, t := range tls.Children {
		t.(core.Widget).AsWidget().SetSelected(i == int(tl))
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
	gv.EditState.Tool = tl
	gv.SetDefaultStyle()
	gv.ModalToolbar().Update()
	gv.SetStatus("Tool")
	sv := gv.SVG()
	sv.UpdateSelect()
}

func (gv *Vector) MakeTools(p *core.Plan) {
	core.Add(p, func(w *core.Button) {
		w.SetIcon(icons.ArrowSelectorTool).SetShortcut("S")
		w.SetTooltip("Select, move, and resize objects")
		w.OnClick(func(e events.Event) {
			gv.SetTool(SelectTool)
		})
	})
	core.Add(p, func(w *core.Button) {
		w.SetIcon("tool-node").SetShortcut("N")
		w.SetTooltip("Select and move node points within paths")
		w.OnClick(func(e events.Event) {
			gv.SetTool(NodeTool)
		})
	})
	core.Add(p, func(w *core.Button) {
		w.SetIcon(icons.Rectangle).SetShortcut("R")
		w.SetTooltip("Create rectangles and squares")
		w.OnClick(func(e events.Event) {
			gv.SetTool(RectTool)
		})
	})
	core.Add(p, func(w *core.Button) {
		w.SetIcon(icons.Circle).SetShortcut("E")
		w.SetTooltip("Create circles, ellipses, and arcs")
		w.OnClick(func(e events.Event) {
			gv.SetTool(EllipseTool)
		})
	})
	core.Add(p, func(w *core.Button) {
		w.SetIcon(icons.LineCurve).SetShortcut("B")
		w.SetTooltip("Create bezier curves (straight lines and curves with control points)")
		w.OnClick(func(e events.Event) {
			gv.SetTool(BezierTool)
		})
	})
	core.Add(p, func(w *core.Button) {
		w.SetIcon("tool-text").SetShortcut("T")
		w.SetTooltip("Add and edit text")
		w.OnClick(func(e events.Event) {
			gv.SetTool(TextTool)
		})
	})
	gv.SetTool(SelectTool)
}
