// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package canvas

type Actions int32 //enums:enum

const (
	NoAction Actions = iota
	Move
	Reshape
	Rotate
	BoxSelect
	SetStrokeColor
	SetFillColor
	NewElement
	NewText
	NewPath
	NodeMove
	CtrlMove
)

// ActionHelpMap contains a set of help strings for different actions.
var ActionHelpMap = map[Actions]string{
	Move:     "<b>Alt</b> = move without snapping, <b>Ctrl</b> = constrain to axis with smallest delta",
	Reshape:  "<b>Alt</b> = rotate, <b>Ctrl</b> = constrain to axis with smallest delta",
	NodeMove: "<b>Alt</b> = only move node, not control points, <b>Ctrl</b> = constrain to axis with smallest delta",
	CtrlMove: "<b>Ctrl</b> = constrain to axis with smallest delta",
}
