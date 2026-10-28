// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"image"
	"testing"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/terminal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCodeOutputClear tests clearing a command output tab with the
// [keymap.Kill] key.
func TestCodeOutputClear(t *testing.T) {
	b := core.NewBody()
	b.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(1200), units.Dp(800)) })
	cv := NewCode(b)
	cv.Defaults()
	tmp := t.TempDir()
	cv.ProjectRoot = core.Filename(tmp)
	cv.Settings.ProjectRoot = core.Filename(tmp)
	showed := make(chan struct{})
	b.OnFinal(events.Show, func(e events.Event) { showed <- struct{}{} })
	b.RunWindow()
	<-showed
	prevMap := keymap.ActiveMapName
	keymap.SetActiveMapName("MacEmacs") // with Control+X key sequences
	defer keymap.SetActiveMapName(prevMap)
	defer func() {
		b.AsyncLock()
		b.Close()
		b.AsyncUnlock()
	}()
	waitFor := func(what string, cond func() bool) {
		t.Helper()
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			b.AsyncLock()
			ok := cond()
			b.AsyncUnlock()
			if ok {
				return
			}
			time.Sleep(20 * time.Millisecond)
		}
		t.Fatalf("timeout waiting for %s", what)
	}

	b.AsyncLock()
	buf, ed, _ := cv.RecycleCmdTab("Output")
	write := func() {
		txt := []rune("some output")
		buf.AppendTextMarkup([][]rune{txt}, []rich.Text{rich.NewText(buf.FontStyle(), txt)})
	}
	write()
	ed.SetFocus()
	b.AsyncUnlock()
	hasOutput := func() bool { return buf.NumLines() > 1 || len(buf.Line(0)) > 0 }
	waitFor("the output tab", hasOutput)
	time.Sleep(100 * time.Millisecond)

	src := b.Scene.RenderWindow().SystemWindow.Events()
	send := func(ch key.Chord) {
		rn, code, mods, err := ch.Decode()
		require.NoError(t, err)
		src.Key(events.KeyDown, rn, code, mods)
	}

	// not in a key sequence
	send("Control+X")
	send("Control+K")
	time.Sleep(200 * time.Millisecond)
	b.AsyncLock()
	assert.True(t, hasOutput(), "not cleared in a key sequence")
	b.AsyncUnlock()

	// the key that clears terminals
	send(terminal.ShortcutChord(keymap.Kill).Chords()[0])
	waitFor("the output to be cleared", func() bool { return !hasOutput() })

	// the usual binding
	b.AsyncLock()
	write()
	b.AsyncUnlock()
	send("Control+K")
	waitFor("the output to be cleared again", func() bool { return !hasOutput() })

	// with the focus on the tab button, after moving to the tabs panel
	b.AsyncLock()
	write()
	cv.FocusOnPanel(TabsIndex)
	b.AsyncUnlock()
	waitFor("the focus in the tabs panel", func() bool { return cv.CurPanel() == TabsIndex })
	send(terminal.ShortcutChord(keymap.Kill).Chords()[0])
	waitFor("the output to be cleared from the tab button", func() bool { return !hasOutput() })

	// the keys are shown in the context menu
	b.AsyncLock()
	defer b.AsyncUnlock()
	m := core.NewScene()
	for _, cm := range ed.ContextMenus {
		cm(m, image.Point{})
	}
	shortcuts := map[string]key.Chord{}
	for _, c := range m.Children {
		if bt := core.AsButton(c); bt != nil {
			shortcuts[bt.Text] = bt.Shortcut
		}
	}
	assert.Equal(t, keymap.Copy.Chord(), shortcuts["Copy"])
	assert.Equal(t, terminal.ShortcutChord(keymap.Kill), shortcuts["Clear"])
}
