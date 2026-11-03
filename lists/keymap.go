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
)

func init() {
	keymap.AvailableMaps.MergeFrom(StandardKeyMaps)
}

// These are the key functions of Lists, in addition to the standard
// [keymap.Functions], as in Cogent Code and Files.
const (
	// KeyNextPanel moves the keyboard focus to the next panel.
	KeyNextPanel keymap.Functions = keymap.FunctionsN + iota

	// KeyPrevPanel moves the keyboard focus to the previous panel.
	KeyPrevPanel
)

// StandardKeyMaps are the key maps of Lists, merged into the standard
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

// listsKeys handles the key functions of Lists for all of its widgets,
// before they get the keys, including the key sequences.
func (ls *Lists) listsKeys(e events.Event) {
	kc := e.KeyChord()
	kf := keymap.Of(kc)
	if core.DebugSettings.KeyEventTrace {
		slog.Info("Lists KeyInput", "widget", ls, "keyFunction", kf, "keyChord", kc)
	}
	if ls.keySeq1 != "" {
		kc2 := string(ls.keySeq1) + " " + string(kc)
		kf2 := keymap.Of(key.Chord(kc2))
		if kf2 == keymap.None || kf == keymap.CancelSelect || kc == "Escape" {
			if core.DebugSettings.KeyEventTrace {
				fmt.Printf("KeyMap sequence: %v aborted\n", ls.keySeq1)
			}
			e.SetHandled() // abort key sequence, don't send esc to anyone else
			ls.keySeq1 = ""
			return
		}
		ls.keySeq1 = ""
		kf = kf2
	} else if kf == keymap.MultiA || kf == keymap.MultiB {
		e.SetHandled()
		ls.keySeq1 = kc
		if core.DebugSettings.KeyEventTrace {
			fmt.Printf("KeyMap sequence needs 2 after: %v\n", ls.keySeq1)
		}
		return
	}
	switch kf {
	case KeyNextPanel, KeyPrevPanel:
		e.SetHandled()
		ls.FocusOtherPanel()
	}
}

// FocusOtherPanel moves the keyboard focus to the other panel: from the
// tree of lists to the table of the current list, if it has one, and from
// the table to the node of the current list in the tree of lists.
func (ls *Lists) FocusOtherPanel() { //types:add
	if ls.tree == nil {
		return
	}
	inTree := ls.tree.ContainsFocus()
	if inTree && ls.Current != nil && !ls.Current.IsGroup() && ls.table != nil {
		ls.focusTable()
		return
	}
	if inTree {
		return
	}
	// on the node of the current list, or else the selected one
	switch sel := ls.tree.GetSelectedNodes(); {
	case ls.Current != nil && ls.tree.FindSyncNode(ls.Current) != nil:
		ls.tree.FindSyncNode(ls.Current).SetFocus()
	case len(sel) > 0:
		sel[len(sel)-1].AsCoreTree().SetFocus()
	default:
		ls.tree.SetFocus()
	}
	ls.NeedsRender()
}
