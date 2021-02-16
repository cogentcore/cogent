// Copyright (c) 2021, The Grid Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grid

import "github.com/goki/ki/kit"

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

var KiT_Tools = kit.Enums.AddEnumAltLower(ToolsN, kit.NotBitFlag, nil, "")

func (ev Tools) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *Tools) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }
