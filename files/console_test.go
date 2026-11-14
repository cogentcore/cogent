// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/textcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConsole tests that the console has no line numbers, although the
// editor settings have them, and that the Kill key (Command+K on macOS),
// sent to the window, clears it.
func TestConsole(t *testing.T) {
	root := testRoot(t)
	testSettings(t)
	prevMap := keymap.ActiveMapName
	keymap.SetActiveMapName("MacStandard")
	defer keymap.SetActiveMapName(prevMap)
	prevLines := TheConsole.Lines
	defer func() { TheConsole.Lines = prevLines }()
	TheConsole.Lines = lines.NewLines() // as Init does, without the pipes
	TheConsole.Lines.Settings.LineNumbers = false
	TheConsole.Lines.SetFilename("console-buf")
	TheConsole.Lines.SetReadOnly(true)
	TheConsole.Lines.SetText([]byte("some output\nmore output"))

	b := core.NewBody()
	fl := NewFiles(b).SetRoot(root)
	var ed *textcore.Editor
	meta := key.Modifiers(0)
	meta.SetFlag(true, key.Meta)
	b.RenderImage(func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		fl.OpenConsoleTab()
		fr := fl.Tabs().RecycleTab(ConsoleTabName)
		require.True(t, fr.HasChildren())
		ed = fr.Child(0).(*textcore.Editor)
		ed.SetFocus()
	}, func() {
		b.AsyncLock()
		assert.False(t, TheConsole.Lines.Settings.LineNumbers, "no line numbers")
		assert.Equal(t, 2, TheConsole.Lines.NumLines())
		b.AsyncUnlock()
		src := b.Scene.RenderWindow().SystemWindow.Events()
		src.Key(events.KeyDown, 0, key.CodeK, meta)
		src.Key(events.KeyUp, 0, key.CodeK, meta)
		eventually(t, b, "the console is cleared", func() bool {
			return TheConsole.Lines.String() == "\n" // one empty line
		})
	})
}
