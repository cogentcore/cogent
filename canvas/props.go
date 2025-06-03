// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package canvas

import (
	"strings"

	"cogentcore.org/core/svg"
	"cogentcore.org/core/tree"
)

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
	if (es.Tool == NodeTool || es.Tool == BezierTool) && es.ActivePath != nil {
		setPropsNode(es.ActivePath, fun)
	} else {
		for itm := range es.Selected {
			setPropsNode(itm, fun)
		}
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
	if (es.Tool == NodeTool || es.Tool == BezierTool) && es.ActivePath != nil {
		setPropsNode(es.ActivePath, fun)
	} else {
		for nd := range es.Selected {
			setPropsNode(nd, fun)
		}
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

////////  SVG misc utils

// DistributeProps distributes properties into leaf nodes
// from groups. Putting properties on groups is not good
// for editing.
func (sv *SVG) DistributeProps() {
	gotSome := false
	root := sv.Root()

	exclude := func(k string) bool {
		if k == "transform" || strings.Contains(k, "groupmode") || strings.Contains(k, "display:inline") || strings.Contains(k, "xmlns:") || strings.Contains(k, "xlink:") {
			return true
		}
		return false
	}

	svg.SVGWalkDownNoDefs(root, func(n svg.Node, nb *svg.NodeBase) bool {
		if n == sv.Root().This {
			return tree.Continue
		}
		if nb.HasChildren() {
			return tree.Continue
		}
		if _, istxt := n.(*svg.Text); istxt { // no text
			return tree.Break
		}
		if nb.Properties == nil {
			nb.Properties = make(map[string]any)
		}
		par := nb.Parent.(svg.Node).AsNodeBase()
		if par.This == root.This || NodeIsLayer(par) {
			return tree.Continue
		}
		for {
			if par.Properties != nil {
				for k, v := range par.Properties {
					if exclude(k) {
						continue
					}
					if _, has := nb.Properties[k]; !has {
						gotSome = true
						nb.Properties[k] = v
					}
				}
			}
			if par.Parent == nil || par.Parent == sv.Root().This {
				break
			}
			par = par.Parent.(svg.Node).AsNodeBase()
		}
		return tree.Continue
	})
	if !gotSome {
		return
	}

	// then get rid of properties on groups
	svg.SVGWalkDownNoDefs(sv.Root(), func(n svg.Node, nb *svg.NodeBase) bool {
		if n == sv.Root().This {
			return tree.Continue
		}
		if !nb.HasChildren() {
			return tree.Break
		}
		if NodeIsLayer(n) {
			return tree.Continue
		}
		if _, istxt := n.(*svg.Text); istxt { // no text
			return tree.Break
		}
		if nb.Properties == nil {
			return tree.Continue
		}
		for k := range nb.Properties {
			if exclude(k) {
				continue
			}
			delete(nb.Properties, k)
		}
		return tree.Continue
	})
}

// UngroupSingletons moves leaf nodes that are all by self in a group
// out of the group.
func (sv *SVG) UngroupSingletons() {
	var singles []svg.Node
	svg.SVGWalkDownNoDefs(sv.Root(), func(n svg.Node, nb *svg.NodeBase) bool {
		if n == sv.Root().This {
			return tree.Continue
		}
		if nb.HasChildren() {
			return tree.Continue
		}
		if txt, istxt := n.(*svg.Text); istxt { // no tspans
			if txt.Text != "" {
				if _, istxt := txt.Parent.(*svg.Text); istxt {
					return tree.Break
				}
			}
		}
		par := nb.Parent.(svg.Node).AsNodeBase()
		if par.NumChildren() != 1 || par.Parent == nil {
			return tree.Continue
		}
		singles = append(singles, n)
		return tree.Continue
	})
	if len(singles) == 0 {
		return
	}
	sv.SVG.Style() // ensure all styles are set
	for _, n := range singles {
		nb := n.AsNodeBase()
		par := nb.Parent.(svg.Node).AsNodeBase()
		parPar := par.Parent
		tree.MoveToParent(nb.This, parPar)
		if !par.Paint.Transform.IsIdentity() {
			nb.ApplyTransform(sv.SVG, par.Paint.Transform)
		}
		ppn := parPar.(svg.Node).AsNodeBase()
		ppn.DeleteChild(par.This)
	}
}

// RemoveEmptyGroups removes groups that have no children.
func (sv *SVG) RemoveEmptyGroups() {
	var empties []svg.Node
	svg.SVGWalkDownNoDefs(sv.Root(), func(n svg.Node, nb *svg.NodeBase) bool {
		if n == sv.Root().This {
			return tree.Continue
		}
		if nb.HasChildren() {
			return tree.Continue
		}
		if _, isgp := n.(*svg.Group); isgp {
			empties = append(empties, n)
		}
		return tree.Continue
	})
	if len(empties) == 0 {
		return
	}
	for _, n := range empties {
		nb := n.AsNodeBase()
		par := nb.Parent.(svg.Node).AsNodeBase()
		par.DeleteChild(n)
	}
}
