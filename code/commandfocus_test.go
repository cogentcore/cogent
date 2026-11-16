// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"runtime"
	"slices"
	"testing"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/styles/states"
	"github.com/stretchr/testify/assert"
)

// TestCommandKeepsEditorFocus tests running a command from the keyboard,
// with the menu search (Control+M) and the dialog of the command in a new
// window, accepted with Command+Return: the dialog window is closed, and
// the editor that it was run from gets the keys typed after it. The
// dialog window stayed open, without a scene, and had the keyboard focus
// of the system.
func TestCommandKeepsEditorFocus(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses ls")
	}
	dir := tempDir(t)
	path := writeFiles(t, dir, map[string]string{"a.txt": "a\n"})
	b, cv := newTestCode(t, dir)
	b.AddTopBar(func(bar *core.Frame) {
		tb := core.NewToolbar(bar)
		tb.Maker(cv.MakeToolbar)
	})
	showed := make(chan struct{}, 1)
	b.OnFinal(events.Show, func(e events.Event) { showed <- struct{}{} })
	b.RunWindow()
	<-showed
	src := b.Scene.RenderWindow().SystemWindow.Events()
	typeText := func(src *events.Source, text string) {
		for _, r := range text {
			src.KeyChord(r, 0, 0)
		}
	}
	var tv *TextEditor
	locked(b, func() {
		cv.ViewFileInIndex(path("a.txt"), 0)
		tv = cv.ActiveEditor()
		tv.SetFocus()
		b.Scene.MenuSearchDialog("Menu search", "")
	})
	waitCode(t, 5*time.Second, "the menu search", func() bool {
		b.AsyncLock()
		defer b.AsyncUnlock()
		return core.TopScene(b) != b.Scene
	})
	typeText(src, "Command > Ls")
	time.Sleep(300 * time.Millisecond) // (the completion of the menu search)
	src.KeyChord(0, key.CodeReturnEnter, 0)
	var dlg *core.Scene
	waitCode(t, 10*time.Second, "the dialog of ls", func() bool {
		for _, w := range core.AllRenderWindows {
			if sc := w.MainScene(); sc != nil && sc != b.Scene && core.ButtonByText(sc, "Run") != nil {
				dlg = sc
				return true
			}
		}
		return false
	})
	dw := dlg.RenderWindow()
	dw.SystemWindow.Events().KeyChord(0, key.CodeReturnEnter, key.Meta)
	waitCode(t, 5*time.Second, "the dialog window closed", func() bool {
		return !slices.Contains(core.AllRenderWindows, dw)
	})
	waitCode(t, 10*time.Second, "the output of ls", func() bool {
		b.AsyncLock()
		defer b.AsyncUnlock()
		return cv.CmdBufs["ls"] != nil
	})
	locked(b, func() { assert.True(t, tv.StateIs(states.Focused)) })
	typeText(src, "y")
	waitCode(t, 5*time.Second, "y typed in the editor", func() bool {
		b.AsyncLock()
		defer b.AsyncUnlock()
		return tv.Lines.String() == "ya\n"
	})
}
