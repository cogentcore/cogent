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

// These are the key functions of Cogent Mail for key sequences, in addition
// to the standard [keymap.Functions], as in Cogent Files and Code; the
// other shortcuts are in [SettingsData.Keys].
const (
	// KeyNextPanel moves the keyboard focus to the next panel.
	KeyNextPanel keymap.Functions = keymap.FunctionsN + iota

	// KeyPrevPanel moves the keyboard focus to the previous panel.
	KeyPrevPanel
)

// addPanelKeys adds the key bindings of the key functions of Cogent Mail
// to each of the given key maps: the key sequences of o and p (or Control+O and Control+P) after each of
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
	KeyCompose KeyFunctions = iota
	KeyFocusSearch
	KeyReply
	KeyReplyAll
	KeyForward
	KeyFile
	KeyTag
	KeyDelete
	KeyMarkAsUnread
	KeyStar
	KeyRemoveAttachments
	KeyMarkAsRead
	KeyRefresh
	KeyFocusNextPanel
	KeyFocusPrevPanel
)

// KeyMap is a map of the keyboard shortcuts of the actions of the main
// window, by their chords, with Command for Meta on the Mac and Control
// elsewhere (see [key.Chord.PlatformChord]).
type KeyMap map[key.Chord]KeyFunctions

// DefaultKeys returns the default keyboard shortcuts of the actions of the
// main window, as in Apple Mail, with Command+F for Forward and Command+W
// for Reply all (wide reply), and Command+Alt+F for the search.
func DefaultKeys() KeyMap {
	return KeyMap{
		"Command+N":         KeyCompose,
		"Backspace":         KeyDelete,
		"Delete":            KeyDelete,
		"Control+D":         KeyDelete,
		"Command+X":         KeyRemoveAttachments,
		"Command+Shift+N":   KeyRefresh, // Get new mail in Apple Mail
		"Control+Tab":       KeyFocusNextPanel,
		"Control+Shift+Tab": KeyFocusPrevPanel,
		"Command+Alt+F":     KeyFocusSearch,
		"Command+R":         KeyReply,
		"Command+W":         KeyReplyAll,
		"Command+Shift+R":   KeyReplyAll,
		"Command+F":         KeyForward,
		"Command+Shift+F":   KeyForward,
		"Control+O":         KeyFile,
		"Control+T":         KeyTag,
		"Command+Shift+U":   KeyMarkAsUnread,
		"Command+Shift+L":   KeyStar, // as the flag of Apple Mail
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
		return strings.Compare(string(a), string(b))
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

	// message is whether the action needs a message being read.
	message bool
}

// shortcuts are the actions of the main window with keyboard shortcuts,
// which only apply there, not in the compose windows (where Control+O
// opens a line and Command+F finds), so they are not in the key maps of
// Cogent Core, but in [SettingsData.Keys].
var shortcuts = []shortcut{
	{KeyCompose, (*App).Compose, false},
	{KeyFocusSearch, (*App).FocusSearch, false},
	{KeyReply, (*App).Reply, true},
	{KeyReplyAll, (*App).ReplyAll, true},
	{KeyForward, (*App).Forward, true},
	{KeyFile, (*App).File, true},
	{KeyTag, (*App).Tag, true},
	{KeyDelete, (*App).Delete, true},
	{KeyMarkAsRead, (*App).MarkAsRead, true},
	{KeyMarkAsUnread, (*App).MarkAsUnread, true},
	{KeyStar, (*App).Star, true},
	{KeyRemoveAttachments, (*App).RemoveAttachments, true},
	{KeyRefresh, (*App).Refresh, false},
	{KeyFocusNextPanel, (*App).FocusNextPanel, false},
	{KeyFocusPrevPanel, (*App).FocusPrevPanel, false},
}

// textKeys are the standard key functions of editing text, whose chords
// are not shortcuts while typing in a text field, as Command+X for Cut.
var textKeys = []keymap.Functions{keymap.Delete, keymap.Backspace, keymap.Cut, keymap.Copy,
	keymap.Paste, keymap.SelectAll, keymap.Undo, keymap.Redo, keymap.Kill}

// isTextKey returns whether the given chord (of the platform), of the
// given chord of a shortcut in [SettingsData.Keys], is for editing text, as
// Backspace or Cut, or one without Command, which is not a shortcut while
// typing in a text field.
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
	if sc, c, ok := shortcutFor(kc); ok {
		if a.typing() && isTextKey(kc, c) {
			return // a key for the text, such as Backspace
		}
		e.SetHandled() // also without a message: not its standard function, as Command+W closing the window
		if !sc.message || a.readMessage != nil {
			sc.fun(a)
		}
		return
	}
	switch kf {
	case KeyNextPanel:
		e.SetHandled()
		a.FocusNextPanel()
	case KeyPrevPanel:
		e.SetHandled()
		a.FocusPrevPanel()
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
// false if nothing in it can take the focus: onto the mailbox being shown,
// the list of messages, and the panel of the message being read, which
// then scrolls with the arrow keys, PageUp and PageDown, and Home and End. The focus is quiet, without the highlight of
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
	if w == a.readPanel { // the message, scrolled with the keys
		if a.readMessage == nil {
			return false
		}
		a.readPanel.SetFocusQuiet()
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
