// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"slices"

	"cogentcore.org/core/core"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/tree"
)

// panels returns the panels among which the keyboard focus moves, in
// order: the file tree, the favorites, the directory tabs and the command
// tabs, without those that are collapsed.
func (fl *Files) panels() []core.Widget {
	sv, lt := fl.Splits(), fl.LeftSplits()
	if sv == nil || lt == nil {
		return nil
	}
	var ps []core.Widget
	if !sv.ChildIsCollapsed(FileTreeIndex) {
		for i, c := range lt.Children {
			if !lt.ChildIsCollapsed(i) {
				ps = append(ps, c.(core.Widget))
			}
		}
	}
	for i := DirTabsIndex; i < len(sv.Children); i++ {
		if !sv.ChildIsCollapsed(i) {
			ps = append(ps, sv.Child(i).(core.Widget))
		}
	}
	return ps
}

// currentPanel returns the index in the panels of the one that has the
// keyboard focus, or -1 if none does.
func (fl *Files) currentPanel(ps []core.Widget) int {
	return slices.IndexFunc(ps, func(w core.Widget) bool {
		return w.AsWidget().ContainsFocus()
	})
}

// FocusNextPanel moves the keyboard focus to the next panel, after the
// last one the first.
func (fl *Files) FocusNextPanel() { //types:add
	fl.focusPanelDelta(1)
}

// FocusPrevPanel moves the keyboard focus to the previous panel, before
// the first one the last.
func (fl *Files) FocusPrevPanel() { //types:add
	fl.focusPanelDelta(-1)
}

// focusPanelDelta moves the keyboard focus by the given number of panels,
// going on to the next one in that direction that can take the focus.
func (fl *Files) focusPanelDelta(delta int) {
	ps := fl.panels()
	np := len(ps)
	if np == 0 {
		return
	}
	cp := fl.currentPanel(ps)
	if cp < 0 && delta < 0 {
		cp = 0
	}
	for range np {
		cp = (cp + delta + np) % np
		if focusPanel(ps[cp]) {
			fl.NeedsRender()
			return
		}
	}
}

// focusPanel moves the keyboard focus into the given panel, returning
// false if nothing in it can take the focus: for tabs, into the current
// tab, or else onto its tab button.
func focusPanel(w core.Widget) bool {
	ts, ok := w.(*core.Tabs)
	if !ok {
		return focusWithin(w)
	}
	ct, idx := ts.CurrentTab()
	if ct == nil {
		return false
	}
	if !focusWithin(ct) {
		_, tab := ts.TabAtIndex(idx)
		tab.SetFocus()
	}
	return true
}

// focusWithin focuses the first focusable widget within the given widget
// (or the widget itself), returning false if there is none. This is unlike
// [core.Events.FocusNextFrom], which goes on to the widgets after it.
func focusWithin(w core.Widget) bool {
	var foc core.Widget
	w.AsWidget().WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
		if foc != nil {
			return tree.Break
		}
		if cwb.IsDisplayable() && !cwb.StateIs(states.Disabled) && cwb.AbilityIs(abilities.Focusable) {
			foc = cw
			return tree.Break
		}
		return tree.Continue
	})
	if foc == nil {
		return false
	}
	foc.AsWidget().SetFocus()
	return true
}
