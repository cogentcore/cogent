// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vector

import (
	"cogentcore.org/core/events"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
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
func (gv *VectorView) SetTool(tl Tools) {
	es := &gv.EditState
	if es.Tool == tl {
		return
	}
	tls := gv.Tools()
	updt := tls.UpdateStart()
	for i, t := range tls.Kids {
		t.(gi.Widget).AsWidget().SetSelected(i == int(tl))
	}
	tls.UpdateEnd(updt)
	fs := es.FirstSelectedNode()
	if fs != nil {
		switch v := fs.(type) {
		case *svg.Text:
			Prefs.TextStyle.CopyStyleFrom(&v.Paint)
		case *svg.Line:
			Prefs.LineStyle.CopyStyleFrom(&v.Paint)
		case *svg.Path:
			Prefs.PathStyle.CopyStyleFrom(&v.Paint)
		default:
			gg := fs.AsNodeBase()
			Prefs.ShapeStyle.CopyStyleFrom(&gg.Paint)
		}
	}
	es.ResetSelected()
	gv.EditState.Tool = tl
	gv.SetDefaultStyle()
	gv.SetModalToolbar()
	gv.SetStatus("Tool")
	sv := gv.SVG()
	sv.UpdateSelect()
}

// SetModalToolbar sets the current modal toolbar based on tool
func (gv *VectorView) SetModalToolbar() {
	tl := gv.EditState.Tool
	switch tl {
	case NodeTool:
		gv.SetModalNode()
	case TextTool:
		gv.SetModalText()
	default:
		gv.SetModalSelect()
	}
}

func (gv *VectorView) ConfigTools() {
	tb := gv.Tools()

	if tb.HasChildren() {
		return
	}

	tb.Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})
	gi.NewButton(tb).SetIcon(icons.ArrowSelectorTool).SetShortcut("S").
		SetTooltip("Select, move, and resize objects").
		OnClick(func(e events.Event) {
			gv.SetTool(SelectTool)
		})
	gi.NewButton(tb).SetIcon("tool-node").SetShortcut("N").
		SetTooltip("Select and move node points within paths").
		OnClick(func(e events.Event) {
			gv.SetTool(NodeTool)
		})
	gi.NewButton(tb).SetIcon(icons.Rectangle).SetShortcut("R").
		SetTooltip("Create rectangles and squares").
		OnClick(func(e events.Event) {
			gv.SetTool(RectTool)
		})
	gi.NewButton(tb).SetIcon(icons.Circle).SetShortcut("E").
		SetTooltip("Create circles, ellipses, and arcs").
		OnClick(func(e events.Event) {
			gv.SetTool(EllipseTool)
		})
	gi.NewButton(tb).SetIcon(icons.LineCurve).SetShortcut("B").
		SetTooltip("Create bezier curves (straight lines and curves with control points)").
		OnClick(func(e events.Event) {
			gv.SetTool(BezierTool)
		})
	gi.NewButton(tb).SetIcon("tool-text").SetShortcut("T").
		SetTooltip("Add and edit text").
		OnClick(func(e events.Event) {
			gv.SetTool(TextTool)
		})

	gv.SetTool(SelectTool)
}
