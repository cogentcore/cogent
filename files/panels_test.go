// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"testing"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
)

// TestPanelFocus tests moving the keyboard focus around the panels with
// the keys of [Files.FocusNextPanel] and [Files.FocusPrevPanel], including
// the Control+X key sequences, which the terminal does not get.
func TestPanelFocus(t *testing.T) {
	root := testRoot(t)
	testSettings(t, root)
	b := core.NewBody()
	b.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(1200), units.Dp(800)) })
	fl := NewFiles(b).SetRoot(root)
	b.AssertRender(t, "panel-focus", func() {
		prevMap := keymap.ActiveMapName
		keymap.SetActiveMapName("MacEmacs") // with Control+X key sequences
		defer keymap.SetActiveMapName(prevMap)

		// the panels: the file tree, the favorites, the directory tabs, and
		// the command tabs, with the terminal selected
		const (
			fileTree = iota
			favorites
			dirTabs
			tabs
		)
		panel := func() int {
			b.AsyncLock()
			defer b.AsyncUnlock()
			return fl.currentPanel(fl.panels())
		}
		expect := func(what string, want int) {
			t.Helper()
			deadline := time.Now().Add(3 * time.Second)
			for panel() != want {
				if time.Now().After(deadline) {
					t.Errorf("%s: panel %d, want %d", what, panel(), want)
					return
				}
				time.Sleep(20 * time.Millisecond)
			}
			time.Sleep(200 * time.Millisecond) // stays there
			if p := panel(); p != want {
				t.Errorf("%s: panel moved to %d", what, p)
			}
		}
		src := b.Scene.RenderWindow().SystemWindow.Events()
		chord := func(rn rune, code key.Codes, mods key.Modifiers) {
			src.Key(events.KeyDown, rn, code, mods)
			if mods == 0 { // a character: the KeyChord is from text input
				src.KeyChord(rn, code, mods)
			}
			src.Key(events.KeyUp, rn, code, mods)
		}
		ctrl := key.Modifiers(0)
		ctrl.SetFlag(true, key.Control)
		ctrlShift := ctrl
		ctrlShift.SetFlag(true, key.Shift)

		b.AsyncLock()
		fl.FocusNextPanel() // from no focus
		b.AsyncUnlock()
		expect("first panel", fileTree)
		chord(0, key.CodeTab, ctrl)
		expect("Control+Tab", favorites)
		chord('x', key.CodeX, ctrl)
		chord('o', key.CodeO, 0)
		expect("Control+X o", dirTabs)
		chord('x', key.CodeX, ctrl)
		chord('o', key.CodeO, ctrl)
		expect("Control+X Control+O", tabs)
		b.AsyncLock()
		tm := fl.focusedTerminal()
		b.AsyncUnlock()
		if tm == nil {
			t.Error("the terminal does not have the focus")
		}
		chord(0, key.CodeTab, ctrl)
		expect("Control+Tab from the terminal, around", fileTree)
		chord(0, key.CodeTab, ctrlShift)
		expect("Control+Shift+Tab, around", tabs)
		chord(0, key.CodeTab, ctrlShift)
		expect("Control+Shift+Tab from the terminal", dirTabs)
		chord('x', key.CodeX, ctrl)
		chord('p', key.CodeP, 0)
		expect("Control+X p", favorites)
		chord('x', key.CodeX, ctrl)
		chord(0, key.CodeEscape, 0)
		chord('p', key.CodeP, 0)
		expect("Control+X Escape aborts", favorites)

		// a collapsed panel is skipped
		b.AsyncLock()
		fl.LeftSplits().SetSplits(1, 0)
		fl.FocusPrevPanel()
		b.AsyncUnlock()
		expect("prev, favorites collapsed", tabs-1) // the panels are one fewer
	})
}
