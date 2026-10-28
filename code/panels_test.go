// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/textpos"
	"cogentcore.org/core/tree"
)

// TestCodePanelFocus tests moving the keyboard focus between the panels,
// by clicking into the editors and with [Code.FocusNextPanel] and
// [Code.FocusPrevPanel], with a Markdown file that has a preview.
func TestCodePanelFocus(t *testing.T) {
	tmp := t.TempDir()
	path := func(name string) string { return filepath.Join(tmp, name) }
	os.WriteFile(path("a.md"), []byte("# A\n\nsome text\n"), 0644)
	os.WriteFile(path("b.txt"), []byte("b text\nmore\n"), 0644)
	b := core.NewBody()
	b.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(1200), units.Dp(800)) })
	cv := NewCode(b)
	cv.Defaults()
	cv.ProjectRoot = core.Filename(tmp)
	cv.Settings.ProjectRoot = core.Filename(tmp)
	imagex.UpdateTestImages = true
	defer func() { imagex.UpdateTestImages = false }()
	b.AssertRender(t, "panel-focus", func() {
		prevMap := keymap.ActiveMapName
		keymap.SetActiveMapName("MacEmacs") // with Control+X key sequences
		defer keymap.SetActiveMapName(prevMap)
		b.AsyncLock()
		cv.ViewFileInIndex(path("a.md"), 0)
		cv.ViewFileInIndex(path("b.txt"), 1)
		ed0, ed1 := cv.EditorByIndex(0), cv.EditorByIndex(1)
		b.AsyncUnlock()

		focused := func() string {
			b.AsyncLock()
			defer b.AsyncUnlock()
			switch {
			case ed0.StateIs(states.Focused):
				return "editor 0"
			case ed1.StateIs(states.Focused):
				return "editor 1"
			}
			name := "none"
			cv.Scene.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
				if cwb.StateIs(states.Focused) {
					name = cwb.Path()
					return tree.Break
				}
				return tree.Continue
			})
			return name
		}
		click := func(ed *TextEditor) {
			b.AsyncLock()
			pt := ed.CharStartPos(textpos.Pos{Line: 1, Char: 1}).ToPoint()
			b.AsyncUnlock()
			pt.Y += 4
			src := b.Scene.RenderWindow().SystemWindow.Events()
			src.MouseButton(events.MouseDown, events.Left, pt, 0)
			src.MouseButton(events.MouseUp, events.Left, pt, 0)
		}
		expect := func(what, want string, active int) {
			t.Helper()
			deadline := time.Now().Add(3 * time.Second)
			for focused() != want || cv.ActiveEditorIndex != active {
				if time.Now().After(deadline) {
					t.Errorf("%s: focus %q, active editor %d; want %q, %d", what, focused(), cv.ActiveEditorIndex, want, active)
					return
				}
				time.Sleep(20 * time.Millisecond)
			}
			time.Sleep(300 * time.Millisecond) // stays there
			if f := focused(); f != want {
				t.Errorf("%s: focus moved to %q after", what, f)
			}
		}

		click(ed1)
		expect("click editor 1", "editor 1", 1)
		click(ed0)
		expect("click editor 0", "editor 0", 0)
		click(ed1)
		expect("click editor 1 again", "editor 1", 1)

		b.AsyncLock()
		cv.FocusPrevPanel()
		b.AsyncUnlock()
		expect("prev panel", "editor 0", 0)
		b.AsyncLock()
		cv.FocusNextPanel()
		b.AsyncUnlock()
		expect("next panel", "editor 1", 1)

		// with the keys, which go to the editor first
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
		chord(0, key.CodeTab, ctrlShift)
		expect("Control+Shift+Tab", "editor 0", 0)
		chord(0, key.CodeTab, ctrl)
		expect("Control+Tab", "editor 1", 1)
		chord('x', key.CodeX, ctrl)
		chord('p', key.CodeP, 0)
		expect("Control+X p", "editor 0", 0)
		chord('x', key.CodeX, ctrl)
		chord('o', key.CodeO, 0)
		expect("Control+X o", "editor 1", 1)

		// with the preview tab shown
		b.AsyncLock()
		cv.Tabs().SelectTabByName("Preview")
		b.AsyncUnlock()
		click(ed0)
		expect("click editor 0, preview shown", "editor 0", 0)
		click(ed1)
		expect("click editor 1, preview shown", "editor 1", 1)
		chord(0, key.CodeTab, ctrlShift)
		expect("Control+Shift+Tab, preview shown", "editor 0", 0)
		chord(0, key.CodeTab, ctrl)
		expect("Control+Tab, preview shown", "editor 1", 1)

		// around all of the panels
		panel := func() int {
			b.AsyncLock()
			defer b.AsyncUnlock()
			return cv.CurPanel()
		}
		expectPanel := func(what string, want int) {
			t.Helper()
			deadline := time.Now().Add(3 * time.Second)
			for panel() != want {
				if time.Now().After(deadline) {
					t.Errorf("%s: panel %d (focus %q), want %d", what, panel(), focused(), want)
					return
				}
				time.Sleep(20 * time.Millisecond)
			}
			time.Sleep(300 * time.Millisecond)
			if p := panel(); p != want {
				t.Errorf("%s: panel moved to %d (focus %q)", what, p, focused())
			}
		}
		chord(0, key.CodeTab, ctrl)
		expectPanel("next to tabs", TabsIndex)
		chord(0, key.CodeTab, ctrl)
		expectPanel("next to file tree", FileTreeIndex)
		chord(0, key.CodeTab, ctrl)
		expectPanel("next to editor 0", TextEditor1Index)
		chord(0, key.CodeTab, ctrlShift)
		expectPanel("prev to file tree", FileTreeIndex)
		chord(0, key.CodeTab, ctrlShift)
		expectPanel("prev to tabs", TabsIndex)
		chord(0, key.CodeTab, ctrlShift)
		expectPanel("prev to editor 1", TextEditor2Index)
	})
}
