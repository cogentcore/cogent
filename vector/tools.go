// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vector

import (
	"cogentcore.org/core/events"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/ki"
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
		SetTooltip("select, move, and resize objects").
		OnClick(func(e events.Event) {
			gv.SetTool(SelectTool)
		})
	tb.AddAction(gi.ActOpts{Label: "N", Icon: "tool-node", Tooltip: "N: select, move node points within paths"},
		gv.This(), func(recv, send ki.Ki, sig int64, data any) {
			grr := recv.Embed(KiT_VectorView).(*VectorView)
			grr.SetTool(NodeTool)
		})
	tb.AddAction(gi.ActOpts{Label: "R", Icon: "stop", Tooltip: "R: create rectangles and squares"},
		gv.This(), func(recv, send ki.Ki, sig int64, data any) {
			grr := recv.Embed(KiT_VectorView).(*VectorView)
			grr.SetTool(RectTool)
		})
	tb.AddAction(gi.ActOpts{Label: "E", Icon: "circlebutton-off", Tooltip: "E: create circles, ellipses, and arcs"},
		gv.This(), func(recv, send ki.Ki, sig int64, data any) {
			grr := recv.Embed(KiT_VectorView).(*VectorView)
			grr.SetTool(EllipseTool)
		})
	tb.AddAction(gi.ActOpts{Label: "B", Icon: "color", Tooltip: "B: create bezier curves (straight lines, curves with control points)"},
		gv.This(), func(recv, send ki.Ki, sig int64, data any) {
			grr := recv.Embed(KiT_VectorView).(*VectorView)
			grr.SetTool(BezierTool)
		})
	tb.AddAction(gi.ActOpts{Label: "T", Icon: "tool-text", Tooltip: "T: add / edit text"},
		gv.This(), func(recv, send ki.Ki, sig int64, data any) {
			grr := recv.Embed(KiT_VectorView).(*VectorView)
			grr.SetTool(TextTool)
		})

	gv.SetTool(SelectTool)
}
