// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package canvas

import (
	"image"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/events"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/svg"
)

// dragOut starts a drag of the selected items, when a move of them
// leaves the window, so that they can be dropped in other windows
// (and other Cogent Canvas apps), as for copy and paste. The move is
// undone: a drop back on this drawing copies the items there, or moves
// them with Shift (see [events.DefaultModBits]).
// It returns false if the mouse is still within the window.
func (sv *SVG) dragOut(e events.Event) bool {
	if sv.Events().InWindow(e.WindowPos()) {
		return false
	}
	es := sv.EditState()
	sl := es.SelectedList(false)
	tvl := sv.Canvas.ItemsAsTrees(sl...)
	if len(tvl) == 0 {
		return false
	}
	var md mimedata.Mimes
	for _, tr := range tvl {
		tr.MimeData(&md)
	}

	// undo the move, which is now a drag
	sprites := sv.SpritesLock()
	for itm, ss := range es.Selected {
		svg.BitCopyFrom(itm, ss.InitState)
	}
	sv.InactivateSprites(SpAlignMatch)
	es.DragReset()
	sprites.Unlock()
	es.ActDone()
	sv.UpdateView()

	sv.dragItems = sl
	sv.Events().DragStart(sv, md, e)
	sv.Events().SetDragImage(sv.selectionImage())
	return true
}

// selectionImage returns a semi-transparent image of the selected
// items, for dragging them.
func (sv *SVG) selectionImage() image.Image {
	if sv.Scene.Renderer == nil {
		return nil
	}
	scimg, ok := sv.Scene.Renderer.Image().(*image.RGBA)
	if !ok {
		return nil
	}
	es := sv.EditState()
	es.UpdateSelectBBox()
	r := es.SelectBBox.ToRect().Intersect(sv.Geom.ContentBBox)
	if r.Empty() {
		return nil
	}
	return gradient.ApplyOpacity(scimg.SubImage(r), 0.5)
}

// handleDND handles drops on the drawing: items dragged from a drawing
// (see [SVG.dragOut]) are added at the drop position, and files are opened.
func (sv *SVG) handleDND() {
	sv.On(events.Drop, func(e events.Event) {
		de := e.(*events.DragDrop)
		e.SetHandled()
		md, _ := de.Data.(mimedata.Mimes)
		if md.HasType(fileinfo.DataJson) {
			sv.dropItems(de, md)
			return
		}
		sv.Canvas.openDropped(de)
	})
	sv.On(events.DropDeleteSource, func(e events.Event) {
		if len(sv.dragItems) > 0 {
			sv.Canvas.DeleteItems(sv.dragItems...)
		}
		sv.dragItems = nil
	})
}

// dropItems adds the items in the given mime data from [SVG.dragOut]
// with the top left of their bounding box at the drop position,
// and selects them.
func (sv *SVG) dropItems(de *events.DragDrop, md mimedata.Mimes) {
	cv := sv.Canvas
	es := sv.EditState()
	sv.UndoSave("Drop", "")
	parent := sv.NewParent(false)
	pn := cv.tree.FindSyncNode(parent)
	if pn == nil {
		return
	}
	n0 := parent.AsTree().NumChildren()
	pn.PasteChildren(md, events.DropCopy)
	kids := parent.AsTree().Children[n0:]
	if len(kids) == 0 {
		return
	}
	sv.SVG.UpdateBBoxes()
	bbox := math32.Box2{}
	bbox.SetEmpty()
	for _, k := range kids {
		bbox.ExpandByBox(k.(svg.Node).AsNodeBase().BBox)
	}
	del := math32.FromPoint(de.Pos()).Sub(bbox.Min)
	es.ResetSelected()
	for _, k := range kids {
		itm := k.(svg.Node)
		xf := itm.AsNodeBase().DeltaTransform(del, math32.Vec2(1, 1), 0, bbox.Min)
		itm.ApplyTransform(sv.SVG, xf)
		es.Select(itm)
	}
	cv.SetStatus("Dropped items")
	cv.ChangeMade()
	cv.UpdateSVG()
	sv.UpdateSelect()
	if de.Source != nil {
		sv.Events().DropFinalize(de) // deletes the source items for a move
	}
}
