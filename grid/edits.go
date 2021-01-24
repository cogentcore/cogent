// Copyright (c) 2021, The Grid Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grid

import (
	"sort"

	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/ki/ki"
)

// EditState has all the current edit state information
type EditState struct {
	Tool     Tools         `desc:"current tool in use"`
	Selected map[ki.Ki]int `copy:"-" json:"-" xml:"-" desc:"selected item(s) -- int is order of selection"`
}

// IsSelected returns the selected status of given slice index
func (es *EditState) IsSelected(itm ki.Ki) bool {
	if _, ok := es.Selected[itm]; ok {
		return true
	}
	return false
}

func (es *EditState) ResetSelected() {
	es.Selected = make(map[ki.Ki]int)
}

// SelectedList returns list of selected items, sorted either ascending or descending
// according to order of selection
func (es *EditState) SelectedList(descendingSort bool) []ki.Ki {
	sls := make([]ki.Ki, len(es.Selected))
	i := 0
	for it := range es.Selected {
		sls[i] = it
		i++
	}
	if descendingSort {
		sort.Slice(sls, func(i, j int) bool {
			return es.Selected[sls[i]] > es.Selected[sls[j]]
		})
	} else {
		sort.Slice(sls, func(i, j int) bool {
			return es.Selected[sls[i]] < es.Selected[sls[j]]
		})
	}
	return sls
}

// Select selects given item (if not already selected) -- updates select
// status of index label
func (es *EditState) Select(itm ki.Ki) {
	idx := len(es.Selected)
	es.Selected[itm] = idx
}

// Unselect unselects given idx (if selected)
func (es *EditState) Unselect(itm ki.Ki) {
	if es.IsSelected(itm) {
		delete(es.Selected, itm)
	}
}

// SelectAction is called when a select action has been received (e.g., a
// mouse click) -- translates into selection updates -- gets selection mode
// from mouse event (ExtendContinuous, ExtendOne)
func (es *EditState) SelectAction(itm ki.Ki, mode mouse.SelectModes) {
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
