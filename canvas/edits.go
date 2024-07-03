// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package canvas

import (
	"image"
	"image/color"
	"sort"
	"strings"
	"sync"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/svg"
	"cogentcore.org/core/undo"
)

// EditState has all the current edit state information
type EditState struct {

	// current tool in use
	Tool Tools

	// current action being performed, for undo labeling
	Action Actions

	// action data set at start of action
	ActData string

	// list of layers
	Layers Layers

	// current layer -- where new objects are inserted
	CurLayer string

	// current shared gradients, referenced by obj-specific gradients
	Gradients []*Gradient

	// current text styling info
	Text TextStyle

	// the undo manager
	Undos undo.Stack

	// contents have changed
	Changed bool `display:"inactive"`

	// action mutex, protecting start / end of actions
	ActMu sync.Mutex `copier:"-" json:"-" xml:"-" display:"-"`

	// selected item(s)
	Selected map[svg.Node]*SelectedState `copier:"-" json:"-" xml:"-" display:"-"`

	// selection just happened on press, and no drag happened in between
	SelectNoDrag bool

	// true if a new text item was made while dragging
	NewTextMade bool

	// point where dragging started, mouse coords
	DragStartPos image.Point

	// current dragging position, mouse coords
	DragCurPos image.Point

	// current selection bounding box
	SelectBBox math32.Box2

	// number of current selectbox sprites
	NSelectSprites int

	// last select action position -- continued clicks in same area lead to deeper selection
	LastSelectPos image.Point

	// recently selected item(s) -- within the same selection position
	RecentlySelected map[svg.Node]*SelectedState `copier:"-" json:"-" xml:"-" display:"-"`

	// bbox at start of dragging
	DragSelectStartBBox math32.Box2

	// current bbox during dragging -- non-snapped version
	DragSelectCurrentBBox math32.Box2

	// current effective bbox during dragging -- snapped version
	DragSelectEffectiveBBox math32.Box2

	// potential points of alignment for dragging
	AlignPts [BBoxPointsN][]math32.Vector2

	// number of current node sprites in use
	NNodeSprites int

	// currently manipulating path object
	ActivePath *svg.Path

	// current path node points
	PathNodes []*PathNode

	// selected path nodes
	PathSelect map[int]struct{}

	// current path command indexes within PathNodes -- where the commands start
	PathCommands []int

	// the parent [Canvas]
	Canvas *Canvas `copier:"-" json:"-" xml:"-" display:"-"`
}

// Init initializes the edit state -- e.g. after opening a new file
func (es *EditState) Init(vv *Canvas) {
	es.Action = NoAction
	es.ActData = ""
	es.CurLayer = ""
	es.Gradients = nil
	es.Undos.Reset()
	es.Changed = false
	es.Canvas = vv
}

// InAction reports whether we currently doing an action
func (es *EditState) InAction() bool {
	es.ActMu.Lock()
	defer es.ActMu.Unlock()
	return es.Action != NoAction
}

// ActStart starts an action, locking the mutex so only one can start
func (es *EditState) ActStart(act Actions, data string) {
	es.ActMu.Lock()
	es.Action = act
	es.ActData = data
}

// ActUnlock unlocks the action mutex -- after done doing all action starting steps
func (es *EditState) ActUnlock() {
	es.ActMu.Unlock()
}

// ActDone finishes an action, resetting action
func (es *EditState) ActDone() {
	es.ActMu.Lock()
	es.Action = NoAction
	es.ActData = ""
	es.ActMu.Unlock()
}

// HasSelected returns true if there are selected items
func (es *EditState) HasSelected() bool {
	return len(es.Selected) > 0
}

// IsSelected returns the selected status of given slice index
func (es *EditState) IsSelected(itm svg.Node) bool {
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
	es.Selected = make(map[svg.Node]*SelectedState)
}

// SelectedList returns list of selected items, sorted either ascending or descending
// according to order of selection
func (es *EditState) SelectedList(descendingSort bool) []svg.Node {
	sls := make([]svg.Node, 0, len(es.Selected))
	for it := range es.Selected {
		if it == nil || it.AsTree().This == nil {
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

// SelectedListDepth returns list of selected items, sorted either
// ascending or descending according to depth:
// ascending = deepest first, descending = highest first
func (es *EditState) SelectedListDepth(sv *SVG, descendingSort bool) []svg.Node {
	dm := sv.DepthMap()
	sls := make([]svg.Node, 0, len(es.Selected))
	for it := range es.Selected {
		if it == nil || it.AsTree().This == nil {
			delete(es.Selected, it)
			continue
		}
		sls = append(sls, it)
	}
	if descendingSort {
		sort.Slice(sls, func(i, j int) bool { // TODO: use slices.SortFunc
			return dm[sls[i]] > dm[sls[j]]
		})
	} else {
		sort.Slice(sls, func(i, j int) bool {
			return dm[sls[i]] < dm[sls[j]]
		})
	}
	return sls
}

// FirstSelectedNode returns the first selected node, that is not a Group
// (recurses into groups)
func (es *EditState) FirstSelectedNode() svg.Node {
	if !es.HasSelected() {
		return nil
	}
	sls := es.SelectedList(true)
	for _, sl := range sls {
		fsl := svg.FirstNonGroupNode(sl.(svg.Node))
		if fsl != nil {
			return fsl.(svg.Node)
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
// status of index text
func (es *EditState) Select(itm svg.Node) {
	idx := len(es.Selected)
	ss := &SelectedState{Order: idx}
	itm.WriteGeom(es.Canvas.SSVG(), &ss.InitGeom)
	if es.Selected == nil {
		es.NewSelected()
	}
	es.Selected[itm] = ss
	es.SanitizeSelected()
}

// Unselect unselects given idx (if selected)
func (es *EditState) Unselect(itm svg.Node) {
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
	for n := range es.Selected {
		if pg, has := n.AsTree().Parent.(*svg.Group); has {
			pgi := pg.This.(svg.Node)
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
		nm[i] = sl[i].AsTree().Name
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
func (es *EditState) SelectAction(n svg.Node, mode events.SelectModes, pos image.Point) {
	if mode == events.NoSelect {
		return
	}
	if !es.HasSelected() || !es.PosInLastSelect(pos) {
		es.StartRecents(pos)
	}
	switch mode {
	case events.SelectOne:
		if es.IsSelected(n) {
			if len(es.Selected) > 1 {
				es.SelectedToRecents()
			}
			es.Select(n)
		} else {
			es.SelectedToRecents()
			es.Select(n)
		}
	case events.ExtendContinuous, events.ExtendOne:
		if es.IsSelected(n) {
			es.Unselect(n)
		} else {
			es.Select(n)
		}
	case events.Unselect:
		es.Unselect(n)
	case events.SelectQuiet:
		es.Select(n)
	case events.UnselectQuiet:
		es.Unselect(n)
	}
}

func (es *EditState) SelectedToRecents() {
	for k, v := range es.Selected {
		es.RecentlySelected[k] = v
		delete(es.Selected, k)
	}
}

func (es *EditState) NewRecents() {
	es.RecentlySelected = make(map[svg.Node]*SelectedState)
}

// StartRecents starts a new list of recently selected items
func (es *EditState) StartRecents(pos image.Point) {
	es.NewRecents()
	es.LastSelectPos = pos
}

// PosInLastSelect returns true if position is within tolerance of
// last selection point
func (es *EditState) PosInLastSelect(pos image.Point) bool {
	tol := image.Point{Settings.SnapTol, Settings.SnapTol}
	bb := image.Rectangle{Min: es.LastSelectPos.Sub(tol), Max: es.LastSelectPos.Add(tol)}
	return pos.In(bb)
}

////////////////////////////////////////////////////////////////

// UpdateSelectBBox updates the current selection bbox surrounding all selected items
func (es *EditState) UpdateSelectBBox() {
	es.SelectBBox.SetEmpty()
	if len(es.Selected) == 0 {
		return
	}
	bbox := math32.Box2{}
	bbox.SetEmpty()
	for itm := range es.Selected {
		g := itm.AsNodeBase()
		bb := math32.Box2{}
		bb.SetFromRect(g.BBox)
		bbox.ExpandByBox(bb)
	}
	es.SelectBBox = bbox
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
	es.UpdateSelectBBox()
	es.DragSelectStartBBox = es.SelectBBox
	es.DragSelectCurrentBBox = es.SelectBBox
	es.DragSelectEffectiveBBox = es.SelectBBox
	for itm, ss := range es.Selected {
		itm.WriteGeom(es.Canvas.SSVG(), &ss.InitGeom)
	}
}

// DragNodeStart captures the current state at start of node dragging.
// position is starting position.
func (es *EditState) DragNodeStart(pos image.Point) {
	es.DragStartPos = pos
}

//////////////////////////////////////////////////////
//  Other Types

// SelectedState is state for selected nodes
type SelectedState struct {

	// order item was selected
	Order int

	// initial geometry, saved when first selected or start dragging -- manipulations restore then transform from there
	InitGeom []float32
}

// GradStop represents a single gradient stop
type GradStop struct {

	// color -- alpha is ignored -- set opacity separately
	Color color.Color

	// opacity determines how opaque color is - used instead of alpha in color
	Opacity float64

	// offset position along the gradient vector: 0 = start, 1 = nominal end
	Offset float64
}

// Gradient represents a single gradient that defines stops (referenced in StopName of other gradients)
type Gradient struct {

	// icon of gradient -- generated to display each gradient
	Ic core.SVG `edit:"-" table:"no-header" width:"5"`

	// name of gradient (id)
	Id string `edit:"-" width:"6"`

	// full name of gradient as SVG element
	Name string `display:"-"`

	// gradient stops
	Stops []*GradStop
}

/*
// Updates our gradient from svg gradient
func (gr *Gradient) UpdateFromGrad(g *core.Gradient) {
	_, id := svg.SplitNameIDDig(g.Nm)
	gr.Id = fmt.Sprintf("%d", id)
	gr.Name = g.Nm
	if g.Grad.Gradient == nil {
		gr.Stops = nil
		return
	}
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
	gr.UpdateIcon()
}
*/

// todo: update grad to sane vals for offs etc

/*
// Updates svg gradient from our gradient
func (gr *Gradient) UpdateGrad(g *core.Gradient) {
	_, id := svg.SplitNameIDDig(g.Nm) // we always need to sync to id & name though
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
		gr.ConfigDefaultGradientStops()
	}
	nst := len(gr.Stops)
	if len(xgr.Stops) != nst {
		xgr.Stops = make([]rasterx.GradStop, nst)
	}
	all0 := true
	for _, gst := range gr.Stops {
		if gst.Offset != 0 {
			all0 = false
		}
	}
	if all0 {
		for i, gst := range gr.Stops {
			gst.Offset = float64(i)
		}
	}

	for i, gst := range gr.Stops {
		xst := &xgr.Stops[i]
		xst.StopColor = gst.Color
		xst.Opacity = gst.Opacity
		xst.Offset = gst.Offset
	}
	gr.UpdateIcon()
}
*/

// ConfigDefaultGradient configures a new default gradient
func (es *EditState) ConfigDefaultGradient() {
	es.Gradients = make([]*Gradient, 1)
	gr := &Gradient{}
	es.Gradients[0] = gr
	// gr.ConfigDefaultGradientStops()
	gr.UpdateIcon()
}

/*
// ConfigDefaultGradientStops configures a new default gradient stops
func (gr *Gradient) ConfigDefaultGradientStops() {
	gr.Stops = make([]*GradStop, 2)
	st1 := &GradStop{Opacity: 1, Offset: 0}
	st1.Color.SetName("white")
	st2 := &GradStop{Opacity: 1, Offset: 1}
	st2.Color.SetName("blue")
	gr.Stops[0] = st1
	gr.Stops[1] = st2
}
*/

// UpdateIcon updates icon
func (gr *Gradient) UpdateIcon() {
	/*
		nm := fmt.Sprintf("grid_grad_%s", gr.Name)
		ici, err := core.TheIcons.IconByName(nm)
		var ic *svg.Icon
		if err != nil {
			ic = &svg.Icon{}
			ic.InitName(ic, nm)
			ic.ViewBox.Size = math32.Vec2(1, 1)
			ic.SetProp("width", units.NewCh(5))
			svg.CurIconSet[nm] = ic
		} else {
			ic = ici.(*svg.Icon)
		}
		nst := len(gr.Stops)
		if ic.NumChildren() != nst {
			config := tree.Config
			for i := range gr.Stops {
				config.Add(svg.RectType, fmt.Sprintf("%d", i))
			}
			ic.ConfigChildren(config)

		}

		px := 0.9 / float32(nst)
		for i, gst := range gr.Stops {
			bx := ic.Child(i).(*svg.Rect)
			bx.Pos.X = 0.05 + float32(i)*px
			bx.Pos.Y = 0.05
			bx.Size.X = px
			bx.Size.Y = 0.9
			bx.SetProp("stroke-width", units.NewPx(0))
			bx.SetProp("stroke", "none")
			bx.SetProp("fill", gst.Color.HexString())
		}
		gr.Ic = icons.Icon(nm)
	*/
}
