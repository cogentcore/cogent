// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"log/slog"
	"strings"
	"unicode"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/text/terminal"
)

func init() {
	keymap.AvailableMaps.MergeFrom(StandardKeyMaps)
}

// These are the key functions of Files, in addition to the standard
// [keymap.Functions], as in Cogent Code.
const (
	// KeyNextPanel moves the keyboard focus to the next panel.
	KeyNextPanel keymap.Functions = keymap.FunctionsN + iota

	// KeyPrevPanel moves the keyboard focus to the previous panel.
	KeyPrevPanel
)

// StandardKeyMaps are the key maps of Files, merged into the standard
// [keymap.AvailableMaps], with the same names. Key sequences start with
// a [keymap.MultiA] or [keymap.MultiB] chord, such as Control+X.
var StandardKeyMaps = keymap.Maps{
	{"MacStandard", "Standard Mac KeyMap", keymap.Map{
		"Control+Tab":         KeyNextPanel,
		"Control+Shift+Tab":   KeyPrevPanel,
		"Control+X o":         KeyNextPanel,
		"Control+X Control+O": KeyNextPanel,
		"Control+X p":         KeyPrevPanel,
		"Control+X Control+P": KeyPrevPanel,
	}},
	{"MacEmacs", "Mac with emacs-style navigation -- emacs wins in conflicts", keymap.Map{
		"Control+Tab":         KeyNextPanel,
		"Control+Shift+Tab":   KeyPrevPanel,
		"Control+X o":         KeyNextPanel,
		"Control+X Control+O": KeyNextPanel,
		"Control+X p":         KeyPrevPanel,
		"Control+X Control+P": KeyPrevPanel,
	}},
	{"LinuxEmacs", "Linux with emacs-style navigation -- emacs wins in conflicts", keymap.Map{
		"Control+Tab":         KeyNextPanel,
		"Control+Shift+Tab":   KeyPrevPanel,
		"Control+X o":         KeyNextPanel,
		"Control+X Control+O": KeyNextPanel,
		"Control+X p":         KeyPrevPanel,
		"Control+X Control+P": KeyPrevPanel,
	}},
	{"LinuxStandard", "Standard Linux key map", keymap.Map{
		"Control+Tab":         KeyNextPanel,
		"Control+Shift+Tab":   KeyPrevPanel,
		"Control+E o":         KeyNextPanel,
		"Control+E Control+O": KeyNextPanel,
		"Control+E p":         KeyPrevPanel,
		"Control+E Control+P": KeyPrevPanel,
	}},
	{"WindowsStandard", "Standard Windows key map", keymap.Map{
		"Control+Tab":         KeyNextPanel,
		"Control+Shift+Tab":   KeyPrevPanel,
		"Control+E o":         KeyNextPanel,
		"Control+E Control+O": KeyNextPanel,
		"Control+E p":         KeyPrevPanel,
		"Control+E Control+P": KeyPrevPanel,
	}},
	{"ChromeStd", "Standard chrome-browser and linux-under-chrome bindings", keymap.Map{
		"Control+Tab":         KeyNextPanel,
		"Control+Shift+Tab":   KeyPrevPanel,
		"Control+E o":         KeyNextPanel,
		"Control+E Control+O": KeyNextPanel,
		"Control+E p":         KeyPrevPanel,
		"Control+E Control+P": KeyPrevPanel,
	}},
}

// filesKeys handles the key functions of Files for all of its widgets,
// before they get the keys, including the key sequences.
func (fl *Files) filesKeys(e events.Event) {
	kc := e.KeyChord()
	kf := keymap.Of(kc)
	if core.DebugSettings.KeyEventTrace {
		slog.Info("Files KeyInput", "widget", fl, "keyFunction", kf, "keyChord", kc)
	}
	if tm := fl.focusedTerminal(); tm != nil {
		fl.lastTerminal = tm
		fl.terminalKeys(tm, e, kc, kf)
		return
	}
	if fl.keySeq1 != "" {
		kc2 := string(fl.keySeq1) + " " + string(kc)
		kf2 := keymap.Of(key.Chord(kc2))
		if kf2 == keymap.None || kf == keymap.CancelSelect || kc == "Escape" {
			if core.DebugSettings.KeyEventTrace {
				fmt.Printf("KeyMap sequence: %v aborted\n", fl.keySeq1)
			}
			e.SetHandled() // abort key sequence, don't send esc to anyone else
			fl.keySeq1 = ""
			return
		}
		fl.keySeq1 = ""
		kf = kf2
	} else if kf == keymap.MultiA || kf == keymap.MultiB {
		e.SetHandled()
		fl.keySeq1 = kc
		if core.DebugSettings.KeyEventTrace {
			fmt.Printf("KeyMap sequence needs 2 after: %v\n", fl.keySeq1)
		}
		return
	}
	if fl.keyFunction(kf) {
		e.SetHandled()
	}
}

// keyFunction does the given key function of Files, returning whether
// it is one.
func (fl *Files) keyFunction(kf keymap.Functions) bool {
	switch kf {
	case KeyNextPanel:
		fl.FocusNextPanel()
	case KeyPrevPanel:
		fl.FocusPrevPanel()
	default:
		return false
	}
	return true
}

// terminalFunctions are the key functions of Files that work in a
// terminal (see [Files.terminalKeys]).
var terminalFunctions = map[keymap.Functions]bool{
	KeyNextPanel: true,
	KeyPrevPanel: true,
}

// terminalKeys handles the keys of a focused terminal, which get all of
// the keys, except for those of the [terminalFunctions]. Their single
// chords always work, and their key sequences (Control+X o) at the command
// line, but not in a full-screen program on the alternate screen, which
// gets all of the keys. The first chord of a sequence is held only if it
// starts one of them, so that Control+C interrupts at once, and goes on to
// the terminal before the next chord if that does not complete one, so
// that the sequences of the shell, such as Control+X Control+E, still work.
func (fl *Files) terminalKeys(tm *terminal.Terminal, e events.Event, kc key.Chord, kf keymap.Functions) {
	if fl.keySeq1 != "" {
		seq := fl.keySeq1
		fl.keySeq1 = ""
		if kf2 := keymap.Of(seq + " " + kc); terminalFunctions[kf2] {
			e.SetHandled()
			fl.keyFunction(kf2)
			return
		}
		tm.SendText(chordText(seq)) // this key goes to the terminal after it
		return
	}
	if terminalFunctions[kf] {
		e.SetHandled()
		fl.keyFunction(kf)
		return
	}
	if tm.ActiveView() == tm.LinesView() && startsTerminalSequence(kc) {
		e.SetHandled()
		fl.keySeq1 = kc
	}
}

// startsTerminalSequence returns whether the given chord is the first of
// a key sequence of one of the [terminalFunctions] in the active key map.
func startsTerminalSequence(kc key.Chord) bool {
	prefix := string(kc) + " "
	for ch, kf := range *keymap.ActiveMap {
		if terminalFunctions[kf] && strings.HasPrefix(string(ch), prefix) {
			return true
		}
	}
	return false
}

// chordText returns the control character that the given Control+letter
// chord sends to a terminal, or "" if it is not one.
func chordText(kc key.Chord) string {
	mods, k := key.ModifiersFromString(string(kc))
	var ctrl key.Modifiers
	ctrl.SetFlag(true, key.Control)
	if mods != ctrl || len(k) != 1 {
		return ""
	}
	r := unicode.ToUpper(rune(k[0]))
	if r < 'A' || r > 'Z' {
		return ""
	}
	return string(r - '@')
}

// focusedTerminal returns the terminal that has the keyboard focus, or nil.
func (fl *Files) focusedTerminal() *terminal.Terminal {
	ts := fl.Tabs()
	if ts == nil {
		return nil
	}
	for i := range ts.NumTabs() {
		fr, _ := ts.TabAtIndex(i)
		if fr == nil || !fr.HasChildren() {
			continue
		}
		if tm, ok := fr.Child(0).(*terminal.Terminal); ok && tm.ContainsFocus() {
			return tm
		}
	}
	return nil
}
