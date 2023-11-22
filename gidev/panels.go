// Copyright (c) 2023, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gidev

import (
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/texteditor"
	"goki.dev/gide/v2/gide"
	"goki.dev/girl/states"
)

// PanelIsOpen returns true if the given panel has not been collapsed and is avail
// and visible for displaying something
func (ge *GideView) PanelIsOpen(panel int) bool {
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
func (ge *GideView) CurPanel() int {
	sv := ge.Splits()
	for i, ski := range sv.Kids {
		_, sk := gi.AsWidget(ski)
		if sk.HasStateWithin(states.Focused) {
			return i
		}
	}
	return -1 // nobody
}

// FocusOnPanel moves keyboard focus to given panel -- returns false if nothing at that tab
func (ge *GideView) FocusOnPanel(panel int) bool {
	wupdt := ge.UpdateStart()
	defer ge.UpdateEndRender(wupdt)

	sv := ge.Splits()
	switch panel {
	case TextView1Idx:
		ge.SetActiveTextViewIdx(0)
	case TextView2Idx:
		ge.SetActiveTextViewIdx(1)
	case TabsIdx:
		tv := ge.Tabs()
		ct, _, has := tv.CurTab()
		if has {
			ge.Sc.EventMgr.FocusNextFrom(ct)
		} else {
			return false
		}
	default:
		ski, _ := gi.AsWidget(sv.Kids[panel])
		ge.Sc.EventMgr.FocusNextFrom(ski)
	}
	return true
}

// FocusNextPanel moves the keyboard focus to the next panel to the right
func (ge *GideView) FocusNextPanel() {
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
func (ge *GideView) FocusPrevPanel() {
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

// TabByLabel returns a tab with given name, nil if not found.
func (ge *GideView) TabByLabel(label string) gi.Widget {
	tv := ge.Tabs()
	return tv.TabByLabel(label)
}

// TabByLabelTry returns a tab with given name, error if not found.
func (ge *GideView) TabByLabelTry(label string) (gi.Widget, error) {
	tv := ge.Tabs()
	return tv.TabByLabelTry(label)
}

// SelectTabByLabel Selects given main tab, and returns all of its contents as well.
func (ge *GideView) SelectTabByLabel(label string) gi.Widget {
	tv := ge.Tabs()
	if tv == nil {
		return nil
	}
	return tv.SelectTabByLabel(label)
}

// RecycleTabTextView returns a tab with given
// name, first by looking for an existing one, and if not found, making a new
// one with a TextView in it.  if sel, then select it.
// returns widget
func (ge *GideView) RecycleTabTextView(label string, sel bool) *texteditor.Editor {
	tv := ge.Tabs()
	if tv == nil {
		return nil
	}
	updt := tv.UpdateStart()
	fr := tv.RecycleTab(label, sel)
	if fr.HasChildren() {
		return fr.Child(0).(*texteditor.Editor)
	}
	txv := texteditor.NewEditor(fr, fr.Nm)
	gide.ConfigOutputTextView(txv)
	tv.Update()
	tv.UpdateEndLayout(updt)
	return txv
}
