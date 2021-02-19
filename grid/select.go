// Copyright (c) 2021, The Grid Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grid

import (
	"image"

	"github.com/goki/gi/svg"
	"github.com/goki/ki/ki"
)

// SelectWithinBBox returns a list of all nodes whose WinBBox is fully contained
// within the given BBox. SVG version excludes layer groups.
func (sv *SVGView) SelectWithinBBox(bbox image.Rectangle, leavesOnly bool) []svg.NodeSVG {
	var rval []svg.NodeSVG
	sv.FuncDownMeFirst(0, sv.This(), func(k ki.Ki, level int, d interface{}) bool {
		if k == sv.This() {
			return ki.Continue
		}
		if k.IsDeleted() || k.IsDestroyed() {
			return ki.Break
		}
		if leavesOnly && k.HasChildren() {
			return ki.Continue
		}
		if k == sv.Defs.This() || NodeIsMetaData(k) {
			return ki.Break
		}
		if NodeIsLayer(k) {
			return ki.Continue
		}
		sii, issvg := k.(svg.NodeSVG)
		if !issvg {
			return ki.Continue
		}
		sg := sii.AsSVGNode()
		if sg.WinBBoxInBBox(bbox) {
			rval = append(rval, sii)
		}
		return ki.Continue
	})
	return rval
}

// SelectContainsPoint finds the first node whose WinBBox contains the given
// point -- nil if none.  If leavesOnly is set then only nodes that have no
// nodes (leaves, terminal nodes) will be considered.
// Any leaf nodes that are within the current edit selection are also
// excluded, as are layer groups.
func (sv *SVGView) SelectContainsPoint(pt image.Point, leavesOnly bool) svg.NodeSVG {
	es := sv.EditState()
	var rval svg.NodeSVG
	sv.FuncDownMeFirst(0, sv.This(), func(k ki.Ki, level int, d interface{}) bool {
		if k == sv.This() {
			return ki.Continue
		}
		if k.IsDeleted() || k.IsDestroyed() {
			return ki.Break
		}
		if leavesOnly && k.HasChildren() {
			return ki.Continue
		}
		if k == sv.Defs.This() || NodeIsMetaData(k) {
			return ki.Break
		}
		if NodeIsLayer(k) {
			return ki.Continue
		}
		sii, issvg := k.(svg.NodeSVG)
		if !issvg {
			return ki.Continue
		}
		if _, issel := es.Selected[sii]; issel {
			return ki.Continue
		}
		sg := sii.AsSVGNode()
		if sg.PosInWinBBox(pt) {
			rval = sii
			return ki.Break
		}
		return ki.Continue
	})
	return rval
}
