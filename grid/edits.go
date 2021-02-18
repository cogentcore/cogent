// Copyright (c) 2021, The Grid Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grid

import (
	"fmt"
	"image"
	"sort"
	"strings"
	"sync"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/gist"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/gi/svg"
	"github.com/goki/gi/undo"
	"github.com/goki/mat32"
	"github.com/srwiley/rasterx"
)

// EditState has all the current edit state information
type EditState struct {
	Tool      Tools       `desc:"current tool in use"`
	Action    string      `desc:"current action being performed -- used for undo labeling"`
	ActData   string      `desc:"action data set at start of action"`
	CurLayer  string      `desc:"current layer -- where new objects are inserted"`
	Gradients []*Gradient `desc:"current shared gradients, referenced by obj-specific gradients"`
	UndoMgr   undo.Mgr    `desc:"undo manager"`

	ActMu            sync.Mutex                `copy:"-" json:"-" xml:"-" view:"-" desc:"action mutex, protecting start / end of actions"`
	Selected         map[svg.NodeSVG]*SelState `copy:"-" json:"-" xml:"-" view:"-" desc:"selected item(s)"`
	DragStartPos     image.Point               `desc:"point where dragging started, mouse coords"`
	DragCurPos       image.Point               `desc:"current dragging position, mouse coords"`
	SelBBox          mat32.Box2                `desc:"current selection bounding box"`
	DragSelStartBBox mat32.Box2                `desc:"bbox at start of dragging"`
	DragSelCurBBox   mat32.Box2                `desc:"current bbox during dragging"`
	ActiveSprites    map[Sprites]*gi.Sprite    `copy:"-" json:"-" xml:"-" view:"-" desc:"cached only for active sprites during manipulation"`
}

// Init initializes the edit state -- e.g. after opening a new file
func (es *EditState) Init() {
	es.Action = ""
	es.ActData = ""
	es.CurLayer = ""
	es.Gradients = nil
	es.UndoMgr.Reset()
}

// InAction reports whether we currently doing an action
func (es *EditState) InAction() bool {
	es.ActMu.Lock()
	defer es.ActMu.Unlock()
	return es.Action != ""
}

// ActStart starts an action, locking the mutex so only one can start
func (es *EditState) ActStart(act, data string) {
	es.ActMu.Lock()
	es.Action = act
	es.ActData = data
}

// ActUnlock unlocks the action mutex -- after done doing all action starting steps
func (es *EditState) ActUnlock() {
	es.ActMu.Unlock()
}

// ActDone finishes an action, resetting action to ""
func (es *EditState) ActDone() {
	es.ActMu.Lock()
	es.Action = ""
	es.ActData = ""
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

// SelectedNamesString returns names of selected items as a
// space-separated single string.  If over 256 chars long, then
// truncated.
func (es *EditState) SelectedNamesString() string {
	sl := strings.Join(es.SelectedNames(), " ")
	if len(sl) >= 256 {
		sl = sl[:255]
	}
	return sl
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

func (es *EditState) EnsureActiveSprites() {
	if es.ActiveSprites == nil {
		es.ActiveSprites = make(map[Sprites]*gi.Sprite)
	}
}

// UpdateSelBBox updates the current selection bbox surrounding all selected items
func (es *EditState) UpdateSelBBox() {
	es.EnsureActiveSprites()
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

// DragStart starts the drag if it hasn't yet started
// returns the starting pos, and true if is start
func (es *EditState) DragStart(pos image.Point) (image.Point, bool) {
	if es.DragStartPos == image.ZP {
		es.DragStartPos = pos
		return pos, true
	}
	return es.DragStartPos, false
}

// DragReset resets drag state information
func (es *EditState) DragReset() {
	es.DragStartPos = image.ZP
}

// DragSelStart captures the current state at start of dragging manipulation
// with selected items. position is starting position.
func (es *EditState) DragSelStart(pos image.Point) {
	es.DragStartPos = pos
	if len(es.Selected) == 0 {
		return
	}
	es.UpdateSelBBox()
	es.DragSelStartBBox = es.SelBBox
	es.DragSelCurBBox = es.SelBBox
	for itm, ss := range es.Selected {
		itm.WriteGeom(&ss.InitGeom)
	}
}

//////////////////////////////////////////////////////
//  Other Types

// SelState is state for selected nodes
type SelState struct {
	Order    int       `desc:"order item was selected"`
	InitGeom []float32 `desc:"initial geometry, saved when first selected or start dragging -- manipulations restore then transform from there"`
}

// GradStop represents a single gradient stop
type GradStop struct {
	Color   gist.Color `desc:"color -- alpha is ignored -- set opacity separately"`
	Opacity float64    `desc:"opacity determines how opaque color is - used instead of alpha in color"`
	Offset  float64    `desc:"offset position along the gradient vector: 0 = start, 1 = nominal end"`
}

// Gradient represents a single gradient that defines stops (referenced in StopName of other gradients)
type Gradient struct {
	Ic    gi.IconName `inactive:"+" tableview:"no-header" desc:"icon of gradient -- generated to display each gradient"`
	Id    string      `inactive:"+" width:"6" desc:"name of gradient (id)"`
	Name  string      `view:"-" desc:"full name of gradient as SVG element"`
	Stops []*GradStop `desc:"gradient stops"`
}

// Updates our gradient from svg gradient
func (gr *Gradient) UpdateFromGrad(g *gi.Gradient) {
	_, id := svg.SplitNameIdDig(g.Nm)
	gr.Id = fmt.Sprintf("%d", id)
	gr.Name = g.Nm
	if g.Grad.Gradient == nil {
		gr.Stops = nil
		return
	}
	gr.Ic = "stop" // todo manage separate list of gradient icons
	xgr := g.Grad.Gradient
	nst := len(xgr.Stops)
	if len(gr.Stops) != nst || gr.Stops == nil {
		gr.Stops = make([]*GradStop, nst)
	}
	for i, xst := range xgr.Stops {
		gst := gr.Stops[i]
		if gr.Stops[i] == nil {
			gst = &GradStop{}
		}
		gst.Color.SetColor(xst.StopColor)
		gst.Opacity = xst.Opacity
		gst.Offset = xst.Offset
		gr.Stops[i] = gst
	}
}

// Updates svg gradient from our gradient
func (gr *Gradient) UpdateGrad(g *gi.Gradient) {
	_, id := svg.SplitNameIdDig(g.Nm) // we always need to sync to id & name though
	gr.Id = fmt.Sprintf("%d", id)
	gr.Name = g.Nm
	gr.Ic = "stop" // todo manage separate list of gradient icons -- update
	if g.Grad.Gradient == nil {
		if strings.HasPrefix(gr.Name, "radial") {
			g.Grad.NewRadialGradient()
		} else {
			g.Grad.NewLinearGradient()
		}
	}
	xgr := g.Grad.Gradient
	if gr.Stops == nil {
		gr.Stops = make([]*GradStop, 1)
		gr.Stops[0] = &GradStop{}
	}
	nst := len(gr.Stops)
	if len(xgr.Stops) != nst {
		xgr.Stops = make([]rasterx.GradStop, nst)
	}
	for i, gst := range gr.Stops {
		xst := &xgr.Stops[i]
		xst.StopColor = gst.Color
		xst.Opacity = gst.Opacity
		xst.Offset = gst.Offset
	}
}

// ConfigDefaultGradient configures a new default gradient
func (es *EditState) ConfigDefaultGradient() {
	es.Gradients = make([]*Gradient, 1)
	gr := &Gradient{}
	es.Gradients[0] = gr
	gr.Stops = make([]*GradStop, 2)
	st1 := &GradStop{Opacity: 1, Offset: 0}
	st1.Color.SetName("blue")
	st2 := &GradStop{Opacity: 1, Offset: 1}
	st2.Color.SetName("white")
	gr.Stops[0] = st1
	gr.Stops[1] = st2
}
