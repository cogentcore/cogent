// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package canvas

import (
	"image"
	"sort"
	"strings"
	"sync"

	"cogentcore.org/core/events"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/svg"
	"cogentcore.org/core/tree"
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

	// whether to constrain the current point when dragging
	DragConstrainPoint bool

	// current selection bounding box
	SelectBBox math32.Box2

	// number of current selectbox sprites
	NSelectSprites int

	// last select action position -- continued clicks in same area lead to deeper selection
	LastSelectPos image.Point

	// recently selected item(s) -- within the same selection position
	RecentlySelected map[svg.Node]*SelectedState `copier:"-" json:"-" xml:"-" display:"-"`

	// SelectIsText is true if the current selection is a single text item
	SelectIsText bool

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
	NodeSelect map[int]struct{}

	// Current control being dragged
	CtrlDragIndex int
	CtrlDrag      Sprites

	// the parent [Canvas]
	Canvas *Canvas `copier:"-" json:"-" xml:"-" display:"-"`
}

// Init initializes the edit state -- e.g. after opening a new file
func (es *EditState) Init(cv *Canvas) {
	es.Action = NoAction
	es.ActData = ""
	es.CurLayer = ""
	es.Gradients = nil
	es.Undos.Reset()
	es.Changed = false
	es.Text.Defaults()
	es.Text.Canvas = cv
	es.Canvas = cv
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

// DepthMap returns a map of all nodes and their associated depth count
// counting up from 0 as the deepest, first drawn node.
func (sv *SVG) DepthMap() map[tree.Node]int {
	m := make(map[tree.Node]int)
	depth := 0
	n := tree.Next(sv.This)
	for n != nil {
		m[n] = depth
		depth++
		n = tree.Next(n)
	}
	return m
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
	ss.InitState = svg.BitCloneNode(itm)
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

// UpdateSelectIsText updates the SelectIsText state.
func (es *EditState) UpdateSelectIsText() {
	es.SelectIsText = false
	fsel := es.FirstSelectedNode()
	if fsel == nil {
		return
	}
	if _, ok := fsel.(*svg.Text); ok {
		es.SelectIsText = true
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

////////

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
		bb := g.BBox
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
		ss.InitState = svg.BitCloneNode(itm)
	}
}

////////  Nodes

// DragNodeStart captures the current state at start of node dragging.
// position is starting position.
func (es *EditState) DragNodeStart(pos image.Point) {
	es.DragStartPos = pos
}

func (es *EditState) NodeIsSelected(i int) bool {
	_, ok := es.NodeSelect[i]
	return ok
}

func (es *EditState) SelectNode(i int) {
	es.NodeSelect[i] = struct{}{}
}

func (es *EditState) UnselectNode(i int) {
	delete(es.NodeSelect, i)
}

func (es *EditState) ResetSelectedNodes() {
	es.NodeSelect = make(map[int]struct{})
}

// NodeSelectAction is called when a select action has been received (e.g., a
// mouse click) -- translates into selection updates -- gets selection mode
// from mouse event (ExtendContinuous, ExtendOne)
func (es *EditState) NodeSelectAction(idx int, mode events.SelectModes) {
	if mode == events.NoSelect {
		return
	}
	if es.NodeSelect == nil {
		es.ResetSelectedNodes()
	}
	switch mode {
	case events.SelectOne:
		if len(es.NodeSelect) > 0 {
			es.ResetSelectedNodes()
		}
		es.SelectNode(idx)
	case events.ExtendContinuous, events.ExtendOne:
		if es.NodeIsSelected(idx) {
			es.UnselectNode(idx)
		} else {
			es.SelectNode(idx)
		}
	case events.Unselect:
		es.UnselectNode(idx)
	case events.SelectQuiet:
		es.SelectNode(idx)
	case events.UnselectQuiet:
		es.UnselectNode(idx)
	}
}

////////  Nodes

// DragCtrlStart captures the current state at start of control point dragging.
// position is starting position.
func (es *EditState) DragCtrlStart(pos image.Point, idx int, ptyp Sprites) {
	es.DragStartPos = pos
	es.CtrlDragIndex = idx
	es.CtrlDrag = ptyp
}

////////  SelectedState

// SelectedState is state for selected nodes
type SelectedState struct {

	// order item was selected
	Order int

	// Initial state of the node: copy of node struct.
	InitState svg.Node
}
