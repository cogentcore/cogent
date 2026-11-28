// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package photos

import (
	"slices"
	"strings"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/tree"
)

// The keyboard shortcuts are as in Cogent Mail: those of the actions of
// the main window are in [SettingsData.Keys], which can be changed, and
// the key sequences for moving among the panels are in the key maps of
// Cogent Core, as Control+X o and Control+X p on the Mac.

func init() {
	addPanelKeys(keymap.AvailableMaps)
}

// These are the key functions of Cogent Photos for key sequences, in
// addition to the standard [keymap.Functions], as in Cogent Mail.
const (
	// KeyNextPanel moves the keyboard focus to the next panel.
	KeyNextPanel keymap.Functions = keymap.FunctionsN + iota

	// KeyPrevPanel moves the keyboard focus to the previous panel.
	KeyPrevPanel
)

// addPanelKeys adds the key bindings of the key functions of Cogent Photos
// to each of the given key maps: the key sequences of o and p (or
// Control+O and Control+P) after each of the chords that start key
// sequences in it ([keymap.MultiA] and [keymap.MultiB], such as Control+X
// on the Mac).
func addPanelKeys(maps keymap.Maps) {
	for _, m := range maps {
		var starts []key.Chord
		for c, f := range m.Map {
			if f == keymap.MultiA || f == keymap.MultiB {
				starts = append(starts, c)
			}
		}
		for _, c := range starts {
			m.Map[c+" o"] = KeyNextPanel
			m.Map[c+" Control+O"] = KeyNextPanel
			m.Map[c+" p"] = KeyPrevPanel
			m.Map[c+" Control+P"] = KeyPrevPanel
		}
	}
}

// KeyFunctions are the actions of the main window that have keyboard
// shortcuts, in [SettingsData.Keys], named as their methods.
type KeyFunctions int32 //enums:enum -trim-prefix Key

const (
	KeyTag KeyFunctions = iota
	KeyRotateLeft
	KeyRotateRight
	KeyFocusSearch
	KeyFocusNextPanel
	KeyFocusPrevPanel
)

// KeyMap is a map of the keyboard shortcuts of the actions of the main
// window, by their chords, with Command for Meta on the Mac and Control
// elsewhere (see [key.Chord.PlatformChord]).
type KeyMap map[key.Chord]KeyFunctions

// DefaultKeys returns the default keyboard shortcuts of the actions of the
// main window, with Control+T and Control+O for tagging, as for tagging
// and filing in Cogent Mail.
func DefaultKeys() KeyMap {
	return KeyMap{
		"Control+T":         KeyTag,
		"Control+O":         KeyTag,
		"Command+L":         KeyRotateLeft,
		"Command+R":         KeyRotateRight,
		"Command+Alt+F":     KeyFocusSearch,
		"Control+Tab":       KeyFocusNextPanel,
		"Control+Shift+Tab": KeyFocusPrevPanel,
	}
}

// addDefaultKeys adds the default keyboard shortcuts ([DefaultKeys]) of the
// actions that are not in [SettingsData.KeysKnown] to [SettingsData.Keys],
// for chords that are free: all of them at first, and those of new actions
// after.
func (sd *SettingsData) addDefaultKeys() {
	if sd.Keys == nil {
		sd.Keys = KeyMap{}
	}
	for c, f := range DefaultKeys() {
		if _, used := sd.Keys[c]; !used && !slices.Contains(sd.KeysKnown, f) {
			sd.Keys[c] = f
		}
	}
	sd.KeysKnown = KeyFunctionsValues()
}

// keys returns the keyboard shortcuts, [DefaultKeys] before they are set
// (see [SettingsData.addDefaultKeys]).
func (sd *SettingsData) keys() KeyMap {
	if sd.Keys == nil {
		return DefaultKeys()
	}
	return sd.Keys
}

// chords returns the chords of the given action in the key map, the
// shortest first, as shown on the buttons.
func (km KeyMap) chords(kf KeyFunctions) []key.Chord {
	var res []key.Chord
	for c, f := range km {
		if f == kf {
			res = append(res, c)
		}
	}
	slices.SortFunc(res, func(a, b key.Chord) int {
		if len(a) != len(b) {
			return len(a) - len(b)
		}
		return strings.Compare(string(b), string(a)) // Control+T before Control+O
	})
	return res
}

// shortcut is an action of the main window, which can have keyboard
// shortcuts in [SettingsData.Keys].
type shortcut struct {
	// kf is the action, whose chords are in [SettingsData.Keys].
	kf KeyFunctions

	// fun is the action.
	fun func(a *App)
}

// shortcuts are the actions of the main window with keyboard shortcuts,
// which only apply there, so they are not in the key maps of Cogent Core,
// but in [SettingsData.Keys].
var shortcuts = []shortcut{
	{KeyTag, (*App).Tag},
	{KeyRotateLeft, (*App).RotateLeft},
	{KeyRotateRight, (*App).RotateRight},
	{KeyFocusSearch, (*App).FocusSearch},
	{KeyFocusNextPanel, (*App).FocusNextPanel},
	{KeyFocusPrevPanel, (*App).FocusPrevPanel},
}

// textKeys are the standard key functions of editing text, whose chords
// are not shortcuts while typing in a text field, as Command+X for Cut.
var textKeys = []keymap.Functions{keymap.Delete, keymap.Backspace, keymap.Cut, keymap.Copy,
	keymap.Paste, keymap.SelectAll, keymap.Undo, keymap.Redo, keymap.Kill}

// isTextKey returns whether the given chord (of the platform), of the
// given chord of a shortcut in [SettingsData.Keys], is for editing text,
// as Backspace or Cut, or one without Command, which is not a shortcut
// while typing in a text field.
func isTextKey(kc, c key.Chord) bool {
	return slices.Contains(textKeys, keymap.Of(kc)) || !strings.Contains(string(c), "Command")
}

// shortcutOf returns the chord of the shortcut of the action with the
// given name, as shown on its buttons, or "" if it has none.
func shortcutOf(name string) key.Chord {
	for _, sc := range shortcuts {
		if sc.kf.String() == name {
			if cs := Settings.keys().chords(sc.kf); len(cs) > 0 {
				return cs[0]
			}
		}
	}
	return ""
}

// shortcutFor returns the shortcut with the given chord (of the platform)
// in [SettingsData.Keys], and its chord there, if any.
func shortcutFor(kc key.Chord) (sc shortcut, chord key.Chord, ok bool) {
	for c, f := range Settings.keys() {
		if c.PlatformChord() == kc {
			for _, sc := range shortcuts {
				if sc.kf == f {
					return sc, c, true
				}
			}
		}
	}
	return shortcut{}, "", false
}

// appKeys handles the keys of Cogent Photos for all of the widgets of the
// app, before they get the keys, including the key sequences.
func (a *App) appKeys(e events.Event) {
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
	if sc, c, ok := shortcutFor(kc); ok {
		if a.typing() && isTextKey(kc, c) {
			return // a key for the text, such as Backspace
		}
		e.SetHandled()
		sc.fun(a)
		return
	}
	switch kf {
	case keymap.Find, keymap.Search:
		e.SetHandled()
		a.FocusSearch()
	case keymap.Delete, keymap.Backspace:
		if a.grid.StateIs(states.Focused) || a.viewer.StateIs(states.Focused) {
			e.SetHandled()
			if a.shown == trashKey {
				return
			}
			a.Trash()
		}
	case keymap.ZoomIn:
		e.SetHandled()
		a.zoomThumbs(1.25)
	case keymap.ZoomOut:
		e.SetHandled()
		a.zoomThumbs(0.8)
	}
}

// typing returns whether the keyboard focus is in a text field of the main
// window, the search field, where only the shortcuts with Command (or
// Control elsewhere) apply, and the other keys, such as Backspace, edit
// the text.
func (a *App) typing() bool {
	return a.search != nil && a.search.StateIs(states.Focused)
}

// FocusSearch moves the keyboard focus to the search field.
func (a *App) FocusSearch() { //types:add
	a.search.SetFocus()
}

// panels returns the panels among which the keyboard focus moves, in
// order: the tags, the photos, and the photo shown.
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
// false if nothing in it can take the focus: onto the tree of tags, the
// grid of the photos of the tab shown, and the photo shown. The focus is
// quiet, without the highlight of the focus from the keyboard, as the
// selection shows where it is.
func (a *App) focusPanel(w core.Widget) bool {
	wb := w.AsWidget()
	switch {
	case a.tags != nil && isWithin(a.tags, w):
		a.tags.SetFocusQuiet() // a read-only tree table, with key navigation
		return true
	case a.grid != nil && isWithin(a.grid, w):
		a.grid.SetFocusQuiet()
		return true
	case a.viewer != nil && isWithin(a.viewer, w):
		if a.viewer.Photo == nil {
			return false
		}
		a.viewer.SetFocusQuiet()
		return true
	}
	for cw, cwb := range wb.AllWidgets() {
		if cwb.IsDisplayable() && !cwb.StateIs(states.Disabled) && cwb.AbilityIs(abilities.Focusable) {
			cw.AsWidget().SetFocusQuiet()
			return true
		}
	}
	return false
}

// isWithin returns whether the given node is the given widget, or in it.
func isWithin(n tree.Node, w core.Widget) bool {
	for ; n != nil; n = n.AsTree().Parent {
		if n == tree.Node(w) {
			return true
		}
	}
	return false
}
