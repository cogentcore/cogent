// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"slices"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
)

func init() {
	addPanelKeys(keymap.AvailableMaps)
}

// These are the key functions of Cogent Mail, in addition to the standard
// [keymap.Functions], as in Cogent Files and Code.
const (
	// KeyNextPanel moves the keyboard focus to the next panel.
	KeyNextPanel keymap.Functions = keymap.FunctionsN + iota

	// KeyPrevPanel moves the keyboard focus to the previous panel.
	KeyPrevPanel
)

// addPanelKeys adds the key bindings of the key functions of Cogent Mail
// to each of the given key maps: Control+Tab and Control+Shift+Tab, and
// the key sequences of o and p (or Control+O and Control+P) after each of
// the chords that start key sequences in it ([keymap.MultiA] and
// [keymap.MultiB], such as Control+X on the Mac).
func addPanelKeys(maps keymap.Maps) {
	for _, m := range maps {
		var starts []key.Chord
		for c, f := range m.Map {
			if f == keymap.MultiA || f == keymap.MultiB {
				starts = append(starts, c)
			}
		}
		m.Map["Control+Tab"] = KeyNextPanel
		m.Map["Control+Shift+Tab"] = KeyPrevPanel
		for _, c := range starts {
			m.Map[c+" o"] = KeyNextPanel
			m.Map[c+" Control+O"] = KeyNextPanel
			m.Map[c+" p"] = KeyPrevPanel
			m.Map[c+" Control+P"] = KeyPrevPanel
		}
	}
}

// mailChords are the single key chords of the actions on messages,
// which only apply in the main window, not in the compose windows (where
// Control+O opens a line), so they are not in the key maps.
var mailChords = map[key.Chord]func(a *App){
	"Control+O": (*App).File,
	"Control+T": (*App).Tag,
}

// mailKeys handles the keys of Cogent Mail for all of the widgets of the
// app, before they get the keys, including the key sequences.
func (a *App) mailKeys(e events.Event) {
	kc := e.KeyChord()
	kf := keymap.Of(kc)
	if a.keySeq1 != "" {
		kf2 := keymap.Of(a.keySeq1 + " " + kc)
		a.keySeq1 = ""
		e.SetHandled() // a key sequence, done or aborted
		switch kf2 {
		case KeyNextPanel:
			a.FocusNextPanel()
		case KeyPrevPanel:
			a.FocusPrevPanel()
		}
		return
	}
	if kf == keymap.MultiA || kf == keymap.MultiB {
		e.SetHandled()
		a.keySeq1 = kc
		return
	}
	switch kf {
	case KeyNextPanel:
		e.SetHandled()
		a.FocusNextPanel()
		return
	case KeyPrevPanel:
		e.SetHandled()
		a.FocusPrevPanel()
		return
	}
	if f, ok := mailChords[kc]; ok {
		e.SetHandled()
		f(a)
	}
}

// panels returns the panels among which the keyboard focus moves, in
// order: the mailboxes, the list of messages, and the message being read.
func (a *App) panels() []core.Widget {
	if a.splits == nil {
		return nil
	}
	var ps []core.Widget
	for _, c := range a.splits.Children {
		ps = append(ps, c.(core.Widget))
	}
	return ps
}

// FocusNextPanel moves the keyboard focus to the next panel, after the
// last one the first.
func (a *App) FocusNextPanel() { //types:add
	a.focusPanelDelta(1)
}

// FocusPrevPanel moves the keyboard focus to the previous panel, before
// the first one the last.
func (a *App) FocusPrevPanel() { //types:add
	a.focusPanelDelta(-1)
}

// focusPanelDelta moves the keyboard focus by the given number of panels,
// going on to the next one in that direction that can take the focus.
func (a *App) focusPanelDelta(delta int) {
	ps := a.panels()
	np := len(ps)
	if np == 0 {
		return
	}
	cp := slices.IndexFunc(ps, func(w core.Widget) bool {
		return w.AsWidget().ContainsFocus()
	})
	if cp < 0 && delta < 0 {
		cp = 0
	}
	for range np {
		cp = (cp + delta + np) % np
		if a.focusPanel(ps[cp]) {
			a.NeedsRender()
			return
		}
	}
}

// focusPanel moves the keyboard focus into the given panel, returning
// false if nothing in it can take the focus: onto the mailbox being shown
// and the selected message, if any.
func (a *App) focusPanel(w core.Widget) bool {
	if w == core.Widget(a.mailboxes) {
		for _, n := range a.mailboxes.GetSelectedNodes() {
			n.AsCoreTree().SetFocus()
			return true
		}
	}
	if a.list != nil && a.list.Parent == w {
		a.list.SetFocus() // a read-only list, with key navigation
		return true
	}
	return focusWithin(w)
}

// focusWithin focuses the first focusable widget within the given widget
// (or the widget itself), returning false if there is none.
func focusWithin(w core.Widget) bool {
	for cw, cwb := range w.AsWidget().AllWidgets() {
		if cwb.IsDisplayable() && !cwb.StateIs(states.Disabled) && cwb.AbilityIs(abilities.Focusable) {
			cw.AsWidget().SetFocus()
			return true
		}
	}
	return false
}
