// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package canvas

import (
	"fmt"
	"image"
	"maps"

	"cogentcore.org/core/events"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/svg"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/types"
)

// SetSVGName sets the name of the element to standard type + id name
func (sv *SVG) SetSVGName(el svg.Node) {
	nwid := sv.SVG.NewUniqueID()
	nwnm := fmt.Sprintf("%s%d", el.SVGName(), nwid)
	el.AsTree().SetName(nwnm)
}

// NewParent returns a parent for creating a new SVG element.
// It uses the current selected group in SVG, or node selected in tree
// if useTree is true, or active layer if it is set.
// If nothing else, it returns the SVG Root node.
func (sv *SVG) NewParent(useTree bool) tree.Node {
	es := sv.EditState()
	cv := sv.Canvas
	var sl []svg.Node
	if useTree {
		sl = cv.AnySelectedNodes()
	} else {
		sl = cv.EditState.SelectedList(false)
	}
	if len(sl) == 1 {
		if gp, isgp := sl[0].(*svg.Group); isgp {
			return gp.This
		}
	}
	if es.CurLayer != "" {
		ly := sv.Root().ChildByName(es.CurLayer, 1)
		if ly != nil {
			return ly
		}
	}
	return sv.Root().This
}

// NewSVGElement makes a new SVG element of the given type.
// see [Canvas.NewParent] for the parent where it is made.
func NewSVGElement[T tree.NodeValue](sv *SVG, useTree bool) *T {
	parent := sv.NewParent(useTree)
	n := tree.New[T](parent)
	sn := any(n).(svg.Node)
	sv.SetSVGName(sn)
	if _, isgp := sn.(*svg.Group); !isgp {
		sv.Canvas.PaintSetter().SetProperties(sn)
	}
	sv.Canvas.UpdateTree()
	return n
}

// NewSVGElementDrag makes a new SVG element of the given type during the drag operation.
func NewSVGElementDrag[T tree.NodeValue](sv *SVG, start, end image.Point) *T {
	minsz := float32(10)
	es := sv.EditState()
	dv := math32.FromPoint(end.Sub(start))
	if !es.InAction() && math32.Abs(dv.X) < minsz && math32.Abs(dv.Y) < minsz {
		// fmt.Println("dv under min:", dv, minsz)
		return nil
	}
	sv.ManipStart(NewElement, types.For[T]().IDName)
	n := NewSVGElement[T](sv, false)
	sn := any(n).(svg.Node)
	snb := sn.AsNodeBase()

	xfi := sv.Root().Paint.Transform.Inverse()
	pos := xfi.MulVector2AsPoint(math32.FromPoint(start))
	sn.SetNodePos(pos)
	sz := dv.Abs().Max(math32.Vector2Scalar(minsz / 2))
	sz = xfi.MulVector2AsVector(sz)
	sn.SetNodeSize(sz)
	sn.BBoxes(sv.SVG, snb.ParentTransform(false))

	es.SelectAction(sn, events.SelectOne, end)
	sv.UpdateView()
	es.DragSelStart(start)
	return n
}

// NewText makes a new Text element with embedded tspan
func (sv *SVG) NewText(start, end image.Point) svg.Node {
	minsz := float32(10)
	es := sv.EditState()
	dv := math32.FromPoint(end.Sub(start))
	if !es.InAction() && math32.Abs(dv.X) < minsz && math32.Abs(dv.Y) < minsz {
		// fmt.Println("dv under min:", dv, minsz)
		return nil
	}

	sv.ManipStart(NewText, "")
	tspan := NewSVGElement[svg.Text](sv, false)
	tspan.Text = "Text"

	xfi := sv.Root().Paint.Transform.Inverse()
	pos := math32.FromPoint(start)
	// minsz := float32(20)
	pos.Y += 20 // todo: need the font size..
	pos = xfi.MulVector2AsPoint(pos)
	tspan.Properties = maps.Clone(es.Text.TextProperties())
	tspan.Pos = pos
	sz := dv.Abs().Max(math32.Vector2Scalar(minsz / 2))
	sz = xfi.MulVector2AsVector(sz)
	tspan.SetNodeSize(sz)
	tspan.BBoxes(sv.SVG, tspan.ParentTransform(false))

	es.SelectAction(tspan, events.SelectOne, end)
	sv.UpdateView()
	es.DragSelStart(start)
	return tspan
}

// NewPath makes a new SVG Path element during the drag operation
func (sv *SVG) NewPath(start, end image.Point) *svg.Path {
	minsz := float32(10)
	es := sv.EditState()
	dv := math32.FromPoint(end.Sub(start))
	if !es.InAction() && math32.Abs(dv.X) < minsz && math32.Abs(dv.Y) < minsz {
		return nil
	}
	sv.ManipStart(NewPath, "")
	n := NewSVGElement[svg.Path](sv, false)
	xfi := sv.Root().Paint.Transform.Inverse()
	pos := xfi.MulVector2AsPoint(math32.FromPoint(start))
	sz := xfi.MulVector2AsVector(dv)
	fmt.Println(n.Data)
	n.Data = nil
	n.Data.MoveTo(pos.X, pos.Y)
	n.Data.LineTo(pos.X+sz.X, pos.Y+sz.Y)
	n.BBoxes(sv.SVG, n.ParentTransform(false))

	es.SelectAction(n, events.SelectOne, end)
	sv.UpdateView()
	sv.EditState().DragSelStart(start)

	es.SelectBBox.Min.X += 1
	es.SelectBBox.Min.Y += 1
	es.DragStartBBox = es.SelectBBox
	es.DragBBox = es.SelectBBox
	es.DragSnapBBox = es.SelectBBox

	// win.SpriteDragging = SpriteName(SpReshapeBBox, SpDnR, 0)
	return n
}
