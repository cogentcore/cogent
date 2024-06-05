// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/texteditor"
)

// PanelIsOpen returns true if the given panel has not been collapsed and is avail
// and visible for displaying something
func (cv *CodeView) PanelIsOpen(panel int) bool {
	sv := cv.Splits()
	if panel < 0 || panel >= len(sv.Children) {
		return false
	}
	if sv.Splits[panel] <= 0.01 {
		return false
	}
	return true
}

// CurPanel returns the splitter panel that currently has keyboard focus
func (cv *CodeView) CurPanel() int {
	sv := cv.Splits()
	for i, ski := range sv.Children {
		_, sk := core.AsWidget(ski)
		if sk.HasStateWithin(states.Focused) {
			return i
		}
	}
	return -1 // nobody
}

// FocusOnPanel moves keyboard focus to given panel -- returns false if nothing at that tab
func (cv *CodeView) FocusOnPanel(panel int) bool {
	sv := cv.Splits()
	switch panel {
	case TextEditor1Index:
		cv.SetActiveTextEditorIndex(0)
	case TextEditor2Index:
		cv.SetActiveTextEditorIndex(1)
	case TabsIndex:
		tv := cv.Tabs()
		ct, _ := tv.CurrentTab()
		if ct != nil {
			cv.Scene.Events.FocusNextFrom(ct)
		} else {
			return false
		}
	default:
		ski, _ := core.AsWidget(sv.Children[panel])
		cv.Scene.Events.FocusNextFrom(ski)
	}
	cv.NeedsRender()
	return true
}

// FocusNextPanel moves the keyboard focus to the next panel to the right
func (cv *CodeView) FocusNextPanel() { //types:add
	sv := cv.Splits()
	cp := cv.CurPanel()
	cp++
	np := len(sv.Children)
	if cp >= np {
		cp = 0
	}
	for sv.Splits[cp] <= 0.01 {
		cp++
		if cp >= np {
			cp = 0
		}
	}
	cv.FocusOnPanel(cp)
}

// FocusPrevPanel moves the keyboard focus to the previous panel to the left
func (cv *CodeView) FocusPrevPanel() { //types:add
	sv := cv.Splits()
	cp := cv.CurPanel()
	cp--
	np := len(sv.Children)
	if cp < 0 {
		cp = np - 1
	}
	for sv.Splits[cp] <= 0.01 {
		cp--
		if cp < 0 {
			cp = np - 1
		}
	}
	cv.FocusOnPanel(cp)
}

// TabByName returns a tab with given name, nil if not found.
func (cv *CodeView) TabByName(name string) core.Widget {
	tv := cv.Tabs()
	return tv.TabByName(name)
}

// SelectTabByName Selects given main tab, and returns all of its contents as well.
func (cv *CodeView) SelectTabByName(name string) core.Widget {
	tv := cv.Tabs()
	if tv == nil {
		return nil
	}
	return tv.SelectTabByName(name)
}

// RecycleTabTextEditor returns a tab with given
// name, first by looking for an existing one, and if not found, making a new
// one with a TextEditor in it.  if sel, then select it.
// returns widget
func (cv *CodeView) RecycleTabTextEditor(name string, sel bool, buf *texteditor.Buffer) *texteditor.Editor {
	tv := cv.Tabs()
	if tv == nil {
		return nil
	}

	fr := tv.RecycleTab(name, sel)
	if fr.HasChildren() {
		return fr.Child(0).(*texteditor.Editor)
	}
	txv := texteditor.NewEditor(fr)
	txv.SetName(fr.Nm)
	if buf != nil {
		txv.SetBuffer(buf)
	}
	ConfigOutputTextEditor(txv)
	tv.Update()
	return txv
}
