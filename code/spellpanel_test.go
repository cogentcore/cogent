// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"slices"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/text/spell"
)

// TestSpellPanel checks that the spell panel steps through the
// misspelled words of an editor, with suggestions, and changes them.
func TestSpellPanel(t *testing.T) {
	b := core.NewBody()
	ed := NewTextEditor(b)
	ed.Lines.SetString("The wrold is “quite” nice.\nAnother speling error here.\n")
	sv := NewSpellPanel(b)
	b.AssertRender(t, "spell-panel", func() {
		sv.Config(nil, ed)
		sv.Update()
		sv.CheckNext()
		if spell.Spell == nil {
			t.Fatal("spell checker not initialized")
		}
		if sv.UnkWord != "wrold" || sv.CurLn != 0 {
			t.Fatalf("first unknown word: %q on line %d", sv.UnkWord, sv.CurLn)
		}
		if !slices.Contains(sv.Suggest, "world") || sv.ChangeText().Text() == "" {
			t.Errorf("suggestions for wrold: %v, change %q", sv.Suggest, sv.ChangeText().Text())
		}
		sv.AcceptSuggestion("world")
		if got := string(ed.Lines.Line(0)); got != "The world is “quite” nice." {
			t.Errorf("after change: %q", got)
		}
		if sv.UnkWord != "speling" || sv.CurLn != 1 {
			t.Fatalf("second unknown word: %q on line %d", sv.UnkWord, sv.CurLn)
		}
		sv.CheckNext() // skip
		if sv.UnkWord != "" {
			t.Errorf("unknown word at end: %q", sv.UnkWord)
		}
	})
}
