// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/textcore"
	"cogentcore.org/core/tree"
)

// PanelIsOpen returns true if the given panel has not been collapsed and is avail
// and visible for displaying something
func (cv *Code) PanelIsOpen(panel int) bool {
	sv := cv.Splits()
	if panel < 0 || panel >= len(sv.Children) {
		return false
	}
	if sv.ChildIsCollapsed(panel) {
		return false
	}
	return true
}

// CurPanel returns the splitter panel that currently has keyboard focus
func (cv *Code) CurPanel() int {
	sv := cv.Splits()
	for i, cn := range sv.Children {
		cw := core.AsWidget(cn)
		if cw.HasStateWithin(states.Focused) {
			return i
		}
	}
	return -1 // nobody
}

// FocusOnPanel moves keyboard focus to given panel -- returns false if nothing at that tab
func (cv *Code) FocusOnPanel(panel int) bool {
	sv := cv.Splits()
	switch panel {
	case TextEditor1Index:
		cv.SetActiveEditorIndex(0)
	case TextEditor2Index:
		cv.SetActiveEditorIndex(1)
	case TabsIndex:
		tv := cv.Tabs()
		ct, idx := tv.CurrentTab()
		if ct == nil {
			return false
		}
		if !focusWithin(ct) { // such as the preview: the tab itself
			_, tab := tv.TabAtIndex(idx)
			tab.SetFocus()
		}
	default:
		if !focusWithin(core.AsWidget(sv.Children[panel])) {
			return false
		}
	}
	cv.NeedsRender()
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

// FocusNextPanel moves the keyboard focus to the next panel to the right
func (cv *Code) FocusNextPanel() { //types:add
	sv := cv.Splits()
	cp := cv.CurPanel()
	cp++
	np := len(sv.Children)
	if cp >= np {
		cp = 0
	}
	for sv.ChildIsCollapsed(cp) {
		cp++
		if cp >= np {
			cp = 0
		}
	}
	cv.FocusOnPanel(cp)
}

// FocusPrevPanel moves the keyboard focus to the previous panel to the left
func (cv *Code) FocusPrevPanel() { //types:add
	sv := cv.Splits()
	cp := cv.CurPanel()
	cp--
	np := len(sv.Children)
	if cp < 0 {
		cp = np - 1
	}
	for sv.ChildIsCollapsed(cp) {
		cp--
		if cp < 0 {
			cp = np - 1
		}
	}
	cv.FocusOnPanel(cp)
}

// TabByName returns a tab with given name, nil if not found.
func (cv *Code) TabByName(name string) core.Widget {
	tv := cv.Tabs()
	return tv.TabByName(name)
}

// SelectTabByName Selects given main tab, and returns all of its contents as well.
func (cv *Code) SelectTabByName(name string) core.Widget {
	tv := cv.Tabs()
	if tv == nil {
		return nil
	}
	return tv.SelectTabByName(name)
}

// RecycleTabTextEditor returns a text editor in a tab with the given name,
// first by looking for an existing one, and if not found, making a new
// one with a text editor in it.
func (cv *Code) RecycleTabTextEditor(name string, buf *lines.Lines) *textcore.Editor {
	tv := cv.Tabs()
	if tv == nil {
		return nil
	}

	fr := tv.RecycleTab(name)
	if fr.HasChildren() {
		return fr.Child(0).(*textcore.Editor)
	}
	txv := textcore.NewEditor(fr)
	txv.SetName(fr.Name)
	fr.Styler(func(s *styles.Style) {
		// critical to not add additional scrollbars: texteditor does it
		s.Overflow.Set(styles.OverflowHidden)
	})
	if buf != nil {
		txv.SetLines(buf)
	}
	ConfigOutputTextEditor(txv)
	tv.Update()
	return txv
}
