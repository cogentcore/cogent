// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"log/slog"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles/states"
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
		// all keys go to the terminal, except for moving to another panel
		fl.lastTerminal = tm
		fl.keySeq1 = ""
		switch kf {
		case KeyNextPanel:
			e.SetHandled()
			fl.FocusNextPanel()
		case KeyPrevPanel:
			e.SetHandled()
			fl.FocusPrevPanel()
		}
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
	switch kf {
	case KeyNextPanel:
		e.SetHandled()
		fl.FocusNextPanel()
	case KeyPrevPanel:
		e.SetHandled()
		fl.FocusPrevPanel()
	}
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
		if tm, ok := fr.Child(0).(*terminal.Terminal); ok && tm.HasStateWithin(states.Focused) {
			return tm
		}
	}
	return nil
}
