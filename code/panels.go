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
func (ge *CodeView) PanelIsOpen(panel int) bool {
	sv := ge.Splits()
	if panel < 0 || panel >= len(sv.Kids) {
		return false
	}
	if sv.Splits[panel] <= 0.01 {
		return false
	}
	return true
}

// CurPanel returns the splitter panel that currently has keyboard focus
func (ge *CodeView) CurPanel() int {
	sv := ge.Splits()
	for i, ski := range sv.Kids {
		_, sk := core.AsWidget(ski)
		if sk.HasStateWithin(states.Focused) {
			return i
		}
	}
	return -1 // nobody
}

// FocusOnPanel moves keyboard focus to given panel -- returns false if nothing at that tab
func (ge *CodeView) FocusOnPanel(panel int) bool {
	sv := ge.Splits()
	switch panel {
	case TextEditor1Index:
		ge.SetActiveTextEditorIndex(0)
	case TextEditor2Index:
		ge.SetActiveTextEditorIndex(1)
	case TabsIndex:
		tv := ge.Tabs()
		ct, _ := tv.CurrentTab()
		if ct != nil {
			ge.Scene.Events.FocusNextFrom(ct)
		} else {
			return false
		}
	default:
		ski, _ := core.AsWidget(sv.Kids[panel])
		ge.Scene.Events.FocusNextFrom(ski)
	}
	ge.NeedsRender()
	return true
}

// FocusNextPanel moves the keyboard focus to the next panel to the right
func (ge *CodeView) FocusNextPanel() { //types:add
	sv := ge.Splits()
	cp := ge.CurPanel()
	cp++
	np := len(sv.Kids)
	if cp >= np {
		cp = 0
	}
	for sv.Splits[cp] <= 0.01 {
		cp++
		if cp >= np {
			cp = 0
		}
	}
	ge.FocusOnPanel(cp)
}

// FocusPrevPanel moves the keyboard focus to the previous panel to the left
func (ge *CodeView) FocusPrevPanel() { //types:add
	sv := ge.Splits()
	cp := ge.CurPanel()
	cp--
	np := len(sv.Kids)
	if cp < 0 {
		cp = np - 1
	}
	for sv.Splits[cp] <= 0.01 {
		cp--
		if cp < 0 {
			cp = np - 1
		}
	}
	ge.FocusOnPanel(cp)
}

// TabByName returns a tab with given name, nil if not found.
func (ge *CodeView) TabByName(name string) core.Widget {
	tv := ge.Tabs()
	return tv.TabByName(name)
}

// SelectTabByName Selects given main tab, and returns all of its contents as well.
func (ge *CodeView) SelectTabByName(name string) core.Widget {
	tv := ge.Tabs()
	if tv == nil {
		return nil
	}
	return tv.SelectTabByName(name)
}

// RecycleTabTextEditor returns a tab with given
// name, first by looking for an existing one, and if not found, making a new
// one with a TextEditor in it.  if sel, then select it.
// returns widget
func (ge *CodeView) RecycleTabTextEditor(name string, sel bool, buf *texteditor.Buffer) *texteditor.Editor {
	tv := ge.Tabs()
	if tv == nil {
		return nil
	}

	fr := tv.RecycleTab(name, sel)
	if fr.HasChildren() {
		return fr.Child(0).(*texteditor.Editor)
	}
	txv := texteditor.NewEditor(fr, fr.Nm)
	if buf != nil {
		txv.SetBuffer(buf)
	}
	ConfigOutputTextEditor(txv)
	tv.Update()
	return txv
}
