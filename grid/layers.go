// Copyright (c) 2021, The Grid Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grid

import (
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// NodeIsLayer returns true if given node is a layer
func NodeIsLayer(kn ki.Ki) bool {
	gm := kit.ToString(kn.Prop("groupmode"))
	return gm == "layer"
}

// LayerIsLocked returns true if layer is locked (insensitive = true)
func LayerIsLocked(kn ki.Ki) bool {
	cp := kit.ToString(kn.Prop("insensitive"))
	return cp == "true"
}

// LayerIsVisible returns true if layer is visible
func LayerIsVisible(kn ki.Ki) bool {
	cp := kit.ToString(kn.Prop("style"))
	return cp != "display:none"
}

// NodeParentLayer returns the parent group that is a layer -- nil if none
func NodeParentLayer(kn ki.Ki) ki.Ki {
	var parLay ki.Ki
	kn.FuncUp(0, kn, func(k ki.Ki, level int, d interface{}) bool {
		if NodeIsLayer(k) {
			parLay = k
			return ki.Break
		}
		return ki.Continue
	})
	return parLay
}

// IsCurLayer returns true if given layer is the current layer
// for creating items
func (gv *GridView) IsCurLayer(lay string) bool {
	return gv.EditState.CurLayer == lay
}

// SetCurLayer sets the current layer for creating items to given one
func (gv *GridView) SetCurLayer(lay string) {
	gv.EditState.CurLayer = lay
	gv.SetStatus("set current layer to: " + lay)
}

// ClearCurLayer clears the current layer for creating items if it
// was set to the given layer name
func (gv *GridView) ClearCurLayer(lay string) {
	if gv.EditState.CurLayer == lay {
		gv.EditState.CurLayer = ""
		gv.SetStatus("clear current layer from: " + lay)
	}
}
