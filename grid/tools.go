// Copyright (c) 2021, The Grid Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grid

import (
	"github.com/goki/gi/gi"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// Tools are the drawing tools
type Tools int

const (
	SelectTool Tools = iota
	NodeTool
	RectTool
	EllipseTool
	BezierTool
	TextTool
	ToolsN
)

//go:generate stringer -type=Tools

var KiT_Tools = kit.Enums.AddEnum(ToolsN, kit.NotBitFlag, nil)

func (ev Tools) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *Tools) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// ToolDoesBasicSelect returns true if tool should do select for clicks
func ToolDoesBasicSelect(tl Tools) bool {
	return tl != NodeTool
}

// SetTool sets the current active tool
func (gv *GridView) SetTool(tl Tools) {
	tls := gv.Tools()
	updt := tls.UpdateStart()
	for i, ti := range tls.Kids {
		t := ti.(gi.Node2D).AsNode2D()
		t.SetSelectedState(i == int(tl))
	}
	tls.UpdateEnd(updt)
	gv.EditState.Tool = tl
	gv.SetModalToolbar()
	gv.SetStatus("Tool")
	sv := gv.SVG()
	sv.UpdateSelect()
}

// SetModalToolbar sets the current modal toolbar based on tool
func (gv *GridView) SetModalToolbar() {
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

func (gv *GridView) ConfigTools() {
	tb := gv.Tools()

	if tb.HasChildren() {
		return
	}

	tb.Lay = gi.LayoutVert
	tb.SetStretchMaxHeight()
	tb.AddAction(gi.ActOpts{Label: "S", Icon: "arrow", Tooltip: "S, Space: select, move, resize objects"},
		gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			grr.SetTool(SelectTool)
		})
	tb.AddAction(gi.ActOpts{Label: "N", Icon: "tool-node", Tooltip: "N: select, move node points within paths"},
		gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			grr.SetTool(NodeTool)
		})
	tb.AddAction(gi.ActOpts{Label: "R", Icon: "stop", Tooltip: "R: create rectangles and squares"},
		gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			grr.SetTool(RectTool)
		})
	tb.AddAction(gi.ActOpts{Label: "E", Icon: "circlebutton-off", Tooltip: "E: create circles, ellipses, and arcs"},
		gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			grr.SetTool(EllipseTool)
		})
	tb.AddAction(gi.ActOpts{Label: "B", Icon: "color", Tooltip: "B: create bezier curves (straight lines, curves with control points)"},
		gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			grr.SetTool(BezierTool)
		})
	tb.AddAction(gi.ActOpts{Label: "T", Icon: "tool-text", Tooltip: "T: add / edit text"},
		gv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			grr := recv.Embed(KiT_GridView).(*GridView)
			grr.SetTool(TextTool)
		})

	gv.SetTool(SelectTool)
}
