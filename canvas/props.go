// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package canvas

import "cogentcore.org/core/svg"

// setPropsNode sets a paint property on given node,
// using given setter function. Handles iterating over groups.
func setPropsNode(nd svg.Node, fun func(g svg.Node)) {
	if gp, isgp := nd.(*svg.Group); isgp {
		for _, kid := range gp.Children {
			setPropsNode(kid.(svg.Node), fun)
		}
		return
	}
	fun(nd)
}

// setPropsOnSelected sets paint property on selected nodes,
// using given setter function.
func (cv *Canvas) setPropsOnSelected(actName, val string, fun func(g svg.Node)) {
	es := &cv.EditState
	cv.SVG.UndoSave(actName, val)
	for itm := range es.Selected {
		setPropsNode(itm, fun)
	}
	cv.ChangeMade()
	cv.SVG.NeedsRender()
}

// setPropsOnSelectedInput sets paint property from a slider-based input that
// sends continuous [events.Input] events, followed by a final [events.Change]
// event, which should have the final = true flag set. This uses the
// [Action] framework to manage the undo saving dynamics involved.
func (cv *Canvas) setPropsOnSelectedInput(act Actions, data string, final bool, fun func(nd svg.Node)) {
	es := &cv.EditState
	sv := cv.SVG
	actStart := false
	finalAct := false
	if final && es.InAction() {
		finalAct = true
	}
	if !final && !es.InAction() {
		final = true
		actStart = true
		es.ActStart(act, data)
		es.ActUnlock()
	}
	if final {
		if !finalAct { // was already saved earlier otherwise
			sv.UndoSave(act.String(), data)
		}
	}
	for nd := range es.Selected {
		setPropsNode(nd, fun)
	}
	if final {
		if !actStart {
			es.ActDone()
			cv.ChangeMade()
			sv.NeedsRender()
		}
	} else {
		sv.NeedsRender()
	}
}
