// Copyright (c) 2021, The Grid Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grid

import (
	"sort"
	"sync"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/gi/svg"
	"github.com/goki/gi/undo"
	"github.com/goki/mat32"
)

// SelState is state for selected nodes
type SelState struct {
	Order    int       `desc:"order item was selected"`
	InitGeom []float32 `desc:"initial geometry, saved when first selected or start dragging -- manipulations restore then transform from there"`
}

// EditState has all the current edit state information
type EditState struct {
	Tool    Tools    `desc:"current tool in use"`
	Action  string   `desc:"current action being performed -- used for undo labeling"`
	UndoMgr undo.Mgr `desc:"undo manager"`

	ActMu         sync.Mutex                `copy:"-" json:"-" xml:"-" view:"-" desc:"action mutex, protecting start / end of actions"`
	Selected      map[svg.NodeSVG]*SelState `copy:"-" json:"-" xml:"-" view:"-" desc:"selected item(s)"`
	SelBBox       mat32.Box2                `desc:"current selection bounding box"`
	DragStartBBox mat32.Box2                `desc:"bbox at start of dragging"`
	DragCurBBox   mat32.Box2                `desc:"current bbox during dragging"`
	ActiveSprites map[Sprites]*gi.Sprite    `copy:"-" json:"-" xml:"-" view:"-" desc:"cached only for active sprites during manipulation"`
}

// InAction reports whether we currently doing an action
func (es *EditState) InAction() bool {
	es.ActMu.Lock()
	defer es.ActMu.Unlock()
	return es.Action != ""
}

// ActStart starts an action, locking the mutex so only one can start
func (es *EditState) ActStart(act string) {
	es.ActMu.Lock()
	es.Action = act
}

// ActUnlock unlocks the action mutex -- after done doing all action starting steps
func (es *EditState) ActUnlock() {
	es.ActMu.Unlock()
}

// ActDone finishes an action, resetting action to ""
func (es *EditState) ActDone() {
	es.ActMu.Lock()
	es.Action = ""
	es.ActMu.Unlock()
}

// HasSelected returns true if there are selected items
func (es *EditState) HasSelected() bool {
	return len(es.Selected) > 0
}

// IsSelected returns the selected status of given slice index
func (es *EditState) IsSelected(itm svg.NodeSVG) bool {
	if _, ok := es.Selected[itm]; ok {
		return true
	}
	return false
}

func (es *EditState) ResetSelected() {
	es.Selected = make(map[svg.NodeSVG]*SelState)
}

// SelectedList returns list of selected items, sorted either ascending or descending
// according to order of selection
func (es *EditState) SelectedList(descendingSort bool) []svg.NodeSVG {
	sls := make([]svg.NodeSVG, len(es.Selected))
	i := 0
	for it := range es.Selected {
		sls[i] = it
		i++
	}
	if descendingSort {
		sort.Slice(sls, func(i, j int) bool {
			return es.Selected[sls[i]].Order > es.Selected[sls[j]].Order
		})
	} else {
		sort.Slice(sls, func(i, j int) bool {
			return es.Selected[sls[i]].Order < es.Selected[sls[j]].Order
		})
	}
	return sls
}

// Select selects given item (if not already selected) -- updates select
// status of index label
func (es *EditState) Select(itm svg.NodeSVG) {
	idx := len(es.Selected)
	ss := &SelState{Order: idx}
	itm.WriteGeom(&ss.InitGeom)
	es.Selected[itm] = ss
}

// Unselect unselects given idx (if selected)
func (es *EditState) Unselect(itm svg.NodeSVG) {
	if es.IsSelected(itm) {
		delete(es.Selected, itm)
	}
}

// SelectedNames returns names of selected items, in order selected
func (es *EditState) SelectedNames() []string {
	sl := es.SelectedList(false)
	ns := len(sl)
	if ns == 0 {
		return nil
	}
	nm := make([]string, ns)
	for i := range sl {
		nm[i] = sl[i].Name()
	}
	return nm
}

// SelectAction is called when a select action has been received (e.g., a
// mouse click) -- translates into selection updates -- gets selection mode
// from mouse event (ExtendContinuous, ExtendOne)
func (es *EditState) SelectAction(itm svg.NodeSVG, mode mouse.SelectModes) {
	if mode == mouse.NoSelect {
		return
	}
	switch mode {
	case mouse.SelectOne:
		if es.IsSelected(itm) {
			if len(es.Selected) > 1 {
				es.ResetSelected()
			}
			es.Select(itm)
		} else {
			es.ResetSelected()
			es.Select(itm)
		}
	case mouse.ExtendContinuous, mouse.ExtendOne:
		if es.IsSelected(itm) {
			es.Unselect(itm)
		} else {
			es.Select(itm)
		}
	case mouse.Unselect:
		es.Unselect(itm)
	case mouse.SelectQuiet:
		es.Select(itm)
	case mouse.UnselectQuiet:
		es.Unselect(itm)
	}
}

////////////////////////////////////////////////////////////////

// UpdateSelBBox updates the current selection bbox surrounding all selected items
func (es *EditState) UpdateSelBBox() {
	if es.ActiveSprites == nil {
		es.ActiveSprites = make(map[Sprites]*gi.Sprite)
	}
	es.SelBBox.SetEmpty()
	if len(es.Selected) == 0 {
		return
	}
	bbox := mat32.Box2{}
	bbox.SetEmpty()
	for itm := range es.Selected {
		bb := mat32.Box2{}
		g := itm.AsSVGNode()
		bb.Min.Set(float32(g.WinBBox.Min.X), float32(g.WinBBox.Min.Y))
		bb.Max.Set(float32(g.WinBBox.Max.X), float32(g.WinBBox.Max.Y))
		bbox.ExpandByBox(bb)
	}
	es.SelBBox = bbox
}

// DragStart captures the current state at start of dragging manipulation
func (es *EditState) DragStart() {
	if len(es.Selected) == 0 {
		return
	}
	es.UpdateSelBBox()
	es.DragStartBBox = es.SelBBox
	es.DragCurBBox = es.SelBBox
	for itm, ss := range es.Selected {
		itm.WriteGeom(&ss.InitGeom)
	}
}
