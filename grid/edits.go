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
	Text      TextStyle   `desc:"current text styling info"`
	UndoMgr   undo.Mgr    `desc:"undo manager"`

	ActMu            sync.Mutex                `copy:"-" json:"-" xml:"-" view:"-" desc:"action mutex, protecting start / end of actions"`
	Selected         map[svg.NodeSVG]*SelState `copy:"-" json:"-" xml:"-" view:"-" desc:"selected item(s)"`
	SelNoDrag        bool                      `desc:"selection just happened on press, and no drag happened in between"`
	NewTextMade      bool                      `desc:"true if a new text item was made while dragging"`
	DragStartPos     image.Point               `desc:"point where dragging started, mouse coords"`
	DragCurPos       image.Point               `desc:"current dragging position, mouse coords"`
	SelBBox          mat32.Box2                `desc:"current selection bounding box"`
	NSelSprites      int                       `desc:"number of current selectbox sprites"`
	LastSelPos       image.Point               `desc:"last select action position -- continued clicks in same area lead to deeper selection"`
	RecentlySelected map[svg.NodeSVG]*SelState `copy:"-" json:"-" xml:"-" view:"-" desc:"recently selected item(s) -- within the same selection position"`
	DragSelStartBBox mat32.Box2                `desc:"bbox at start of dragging"`
	DragSelCurBBox   mat32.Box2                `desc:"current bbox during dragging -- non-snapped version"`
	DragSelEffBBox   mat32.Box2                `desc:"current effective bbox during dragging -- snapped version"`
	AlignPts         [BBoxPointsN][]mat32.Vec2 `desc:"potential points of alignment for dragging"`
	NNodeSprites     int                       `desc:"number of current node sprites in use"`
	ActivePath       *svg.Path                 `desc:"currently manipulating path object"`
	PathNodes        []*PathNode               `desc:"current path node points"`
	PathSel          map[int]struct{}          `desc:"selected path nodes"`
	PathCmds         []int                     `desc:"current path command indexes within PathNodes -- where the commands start"`
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

// ResetSelected resets the selection list, including recents
func (es *EditState) ResetSelected() {
	es.NewSelected()
	es.StartRecents(image.ZP)
}

// NewSelected makes a new Selected list
func (es *EditState) NewSelected() {
	es.Selected = make(map[svg.NodeSVG]*SelState)
}

// SelectedList returns list of selected items, sorted either ascending or descending
// according to order of selection
func (es *EditState) SelectedList(descendingSort bool) []svg.NodeSVG {
	sls := make([]svg.NodeSVG, 0, len(es.Selected))
	for it := range es.Selected {
		if it == nil || it.This() == nil || it.IsDeleted() || it.IsDestroyed() {
			delete(es.Selected, it)
			continue
		}
		sls = append(sls, it)
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

// FirstSelectedNode returns the first selected node, that is not a Group
// (recurses into groups)
func (es *EditState) FirstSelectedNode() svg.NodeSVG {
	if !es.HasSelected() {
		return nil
	}
	sls := es.SelectedList(true)
	for _, sl := range sls {
		fsl := svg.FirstNonGroupNode(sl.This())
		if fsl != nil {
			return fsl.(svg.NodeSVG)
		}
	}
	return nil
}

// FirstSelectedPath returns the first selected Path, that is not a Group
// (recurses into groups)
func (es *EditState) FirstSelectedPath() *svg.Path {
	fsn := es.FirstSelectedNode()
	if fsn == nil {
		return nil
	}
	path, ok := fsn.(*svg.Path)
	if !ok {
		return nil
	}
	return path
}

// Select selects given item (if not already selected) -- updates select
// status of index label
func (es *EditState) Select(itm svg.NodeSVG) {
	idx := len(es.Selected)
	ss := &SelState{Order: idx}
	itm.WriteGeom(&ss.InitGeom)
	if es.Selected == nil {
		es.NewSelected()
	}
	es.Selected[itm] = ss
	es.SanitizeSelected()
}

// Unselect unselects given idx (if selected)
func (es *EditState) Unselect(itm svg.NodeSVG) {
	if es.IsSelected(itm) {
		ss := es.Selected[itm]
		es.RecentlySelected[itm] = ss
		delete(es.Selected, itm)
	}
}

// SanitizeSelected ensures that the current selected list makes
// sense.  E.g., it prevents having a group and a child both in
// the selected list (removes the parent group).
func (es *EditState) SanitizeSelected() {
	for k := range es.Selected {
		if pg, has := k.Parent().(*svg.Group); has {
			pgi := pg.This().(svg.NodeSVG)
			if _, issel := es.Selected[pgi]; issel {
				delete(es.Selected, pgi)
			}
		}
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
func (es *EditState) SelectAction(itm svg.NodeSVG, mode mouse.SelectModes, pos image.Point) {
	if mode == mouse.NoSelect {
		return
	}
	if !es.HasSelected() || !es.PosInLastSel(pos) {
		es.StartRecents(pos)
	}
	switch mode {
	case mouse.SelectOne:
		if es.IsSelected(itm) {
			if len(es.Selected) > 1 {
				es.SelectedToRecents()
			}
			es.Select(itm)
		} else {
			es.SelectedToRecents()
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

func (es *EditState) SelectedToRecents() {
	for k, v := range es.Selected {
		es.RecentlySelected[k] = v
		delete(es.Selected, k)
	}
}

func (es *EditState) NewRecents() {
	es.RecentlySelected = make(map[svg.NodeSVG]*SelState)
}

// StartRecents starts a new list of recently-selected items
func (es *EditState) StartRecents(pos image.Point) {
	es.NewRecents()
	es.LastSelPos = pos
}

// PosInLastSel returns true if position is within tolerance of
// last selection point
func (es *EditState) PosInLastSel(pos image.Point) bool {
	tol := image.Point{Prefs.SnapTol, Prefs.SnapTol}
	bb := image.Rectangle{Min: es.LastSelPos.Sub(tol), Max: es.LastSelPos.Add(tol)}
	return pos.In(bb)
}

////////////////////////////////////////////////////////////////

// UpdateSelBBox updates the current selection bbox surrounding all selected items
func (es *EditState) UpdateSelBBox() {
	es.SelBBox.SetEmpty()
	if len(es.Selected) == 0 {
		return
	}
	bbox := mat32.Box2{}
	bbox.SetEmpty()
	for itm := range es.Selected {
		g := itm.AsSVGNode()
		bb := mat32.Box2{}
		bb.SetFromRect(g.WinBBox)
		bbox.ExpandByBox(bb)
	}
	es.SelBBox = bbox
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
	es.DragSelEffBBox = es.SelBBox
	for itm, ss := range es.Selected {
		itm.WriteGeom(&ss.InitGeom)
	}
}

// DragNodeStart captures the current state at start of node dragging.
// position is starting position.
func (es *EditState) DragNodeStart(pos image.Point) {
	es.DragStartPos = pos
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
