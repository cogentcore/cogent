// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"slices"
	"strings"

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

// shortcut is a keyboard shortcut of an action of the main window.
type shortcut struct {
	// name is the name of the method of the action.
	name string

	// chords are the chords of the shortcut, with Command for Meta
	// on the Mac and Control elsewhere (see [key.Chord.PlatformChord]).
	// The first one is shown on the buttons.
	chords []key.Chord

	// fun is the action.
	fun func(a *App)

	// message is whether the action needs a message being read.
	message bool

	// keyFuns are the standard key functions of the shortcut, whose
	// chords in the key map of the user are its chords, as Delete
	// (Control+D in the Emacs key map) and Backspace for deleting. The
	// first one is shown on the buttons, if it has no chords.
	keyFuns []keymap.Functions
}

// shortcuts are the keyboard shortcuts of the actions of the main window,
// which only apply there, not in the compose windows (where Control+O
// opens a line), so they are not in the key maps. Those with standard key
// functions use the chords of the key map of the user; the others, which
// have none, have their own chords.
var shortcuts = []shortcut{
	{"Compose", nil, (*App).Compose, false, []keymap.Functions{keymap.New}},
	{"FocusSearch", nil, (*App).FocusSearch, false, []keymap.Functions{keymap.Find}},
	{"Reply", []key.Chord{"Command+R"}, (*App).Reply, true, nil},
	{"ReplyAll", []key.Chord{"Command+Shift+R"}, (*App).ReplyAll, true, nil},
	{"Forward", []key.Chord{"Command+Shift+F"}, (*App).Forward, true, nil},
	{"File", []key.Chord{"Control+O"}, (*App).File, true, nil},
	{"Tag", []key.Chord{"Control+T"}, (*App).Tag, true, nil},
	{"Delete", nil, (*App).Delete, true, []keymap.Functions{keymap.Delete, keymap.Backspace}},
	{"MarkAsUnread", []key.Chord{"Command+Shift+U"}, (*App).MarkAsUnread, true, nil},
}

// textKey returns whether the shortcut is a key for editing text, as
// Backspace, which is not a shortcut while typing in a text field.
func (sc shortcut) textKey() bool {
	for _, kf := range sc.keyFuns {
		if kf == keymap.Delete || kf == keymap.Backspace {
			return true
		}
	}
	return len(sc.chords) > 0 && !strings.Contains(string(sc.chords[0]), "Command")
}

// shortcutOf returns the chord of the shortcut of the action with the
// given name, as shown on its buttons, or "" if it has none.
func shortcutOf(name string) key.Chord {
	for _, sc := range shortcuts {
		if sc.name != name {
			continue
		}
		if len(sc.chords) > 0 {
			return sc.chords[0]
		}
		if len(sc.keyFuns) > 0 {
			return sc.keyFuns[0].Chord()
		}
	}
	return ""
}

// shortcutFor returns the shortcut with the given chord, or whose key
// functions it is in the key map, if any.
func shortcutFor(kc key.Chord) (shortcut, bool) {
	kf := keymap.Of(kc)
	for _, sc := range shortcuts {
		for _, c := range sc.chords {
			if c.PlatformChord() == kc {
				return sc, true
			}
		}
		if slices.Contains(sc.keyFuns, kf) {
			return sc, true
		}
	}
	return shortcut{}, false
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
	if sc, ok := shortcutFor(kc); ok {
		if sc.message && a.readMessage == nil {
			return
		}
		if a.typing() && sc.textKey() {
			return // a key for the text, such as Backspace
		}
		e.SetHandled()
		sc.fun(a)
	}
}

// typing returns whether the keyboard focus is in a text field of the main
// window, the search field, where only the shortcuts with Command (or
// Control elsewhere) apply, and the other keys, such as Backspace, edit
// the text.
func (a *App) typing() bool {
	return a.searchField != nil && a.searchField.StateIs(states.Focused)
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
// and the list of messages. The focus is quiet, without the highlight of
// the focus from the keyboard, as the selection shows where it is.
func (a *App) focusPanel(w core.Widget) bool {
	if a.mailboxes != nil && a.mailboxes.Parent != nil && a.mailboxes.Parent.AsTree().Parent == w { // the sidebar
		for _, n := range a.mailboxes.GetSelectedNodes() {
			n.AsCoreTree().SetFocusQuiet()
			return true
		}
	}
	if a.list != nil && a.list.Parent == w {
		a.list.SetFocusQuiet() // a read-only list, with key navigation
		return true
	}
	return focusWithin(w)
}

// focusWithin focuses the first focusable widget within the given widget
// (or the widget itself), returning false if there is none.
func focusWithin(w core.Widget) bool {
	for cw, cwb := range w.AsWidget().AllWidgets() {
		if cwb.IsDisplayable() && !cwb.StateIs(states.Disabled) && cwb.AbilityIs(abilities.Focusable) {
			cw.AsWidget().SetFocusQuiet()
			return true
		}
	}
	return false
}
