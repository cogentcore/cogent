// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package canvas

import (
	"fmt"
	"image"

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

// NewSVGElement makes a new SVG element of the given type.
// It uses the current active layer if it is set.
func NewSVGElement[T tree.NodeValue](sv *SVG) *T {
	es := sv.EditState()
	parent := tree.Node(sv.Root())
	if es.CurLayer != "" {
		ly := sv.ChildByName(es.CurLayer, 1)
		if ly != nil {
			parent = ly
		}
	}
	n := tree.New[T](parent)
	sn := any(n).(svg.Node)
	sv.SetSVGName(sn)
	sv.Canvas.PaintSetter().SetProperties(sn)
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
	n := NewSVGElement[T](sv)
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
	sv.UpdateSelSprites()
	es.DragSelStart(start)
	sv.NeedsRender()
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
	n := NewSVGElement[svg.Text](sv)
	tsnm := fmt.Sprintf("tspan%d", sv.SVG.NewUniqueID())
	tspan := svg.NewText(n)
	tspan.SetName(tsnm)
	tspan.Text = "Text"
	tspan.Width = 200

	xfi := sv.Root().Paint.Transform.Inverse()
	pos := math32.FromPoint(start)
	// minsz := float32(20)
	pos.Y += 20 // todo: need the font size..
	pos = xfi.MulVector2AsPoint(pos)
	// sv.Canvas.SetTextPropertiesNode(n, es.Text.TextProperties())
	tspan.Pos = pos
	sz := dv.Abs().Max(math32.Vector2Scalar(minsz / 2))
	sz = xfi.MulVector2AsVector(sz)
	tspan.SetNodeSize(sz)
	tspan.BBoxes(sv.SVG, tspan.ParentTransform(false))

	es.SelectAction(tspan, events.SelectOne, end)
	sv.UpdateSelSprites()
	es.DragSelStart(start)
	sv.NeedsRender()
	return n
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
	n := NewSVGElement[svg.Path](sv)
	xfi := sv.Root().Paint.Transform.Inverse()
	pos := xfi.MulVector2AsPoint(math32.FromPoint(start))
	sz := xfi.MulVector2AsVector(dv)
	fmt.Println(n.Data)
	n.Data = nil
	n.Data.MoveTo(pos.X, pos.Y)
	n.Data.LineTo(sz.X, sz.Y)
	n.BBoxes(sv.SVG, n.ParentTransform(false))

	es.SelectAction(n, events.SelectOne, end)
	sv.UpdateSelSprites()
	sv.EditState().DragSelStart(start)
	sv.NeedsRender()

	es.SelectBBox.Min.X += 1
	es.SelectBBox.Min.Y += 1
	es.DragSelectStartBBox = es.SelectBBox
	es.DragSelectCurrentBBox = es.SelectBBox
	es.DragSelectEffectiveBBox = es.SelectBBox

	// win.SpriteDragging = SpriteName(SpReshapeBBox, SpBBoxDnR, 0)
	return n
}
