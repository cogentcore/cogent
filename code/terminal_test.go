// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"image"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/terminal"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCodeTerminal tests the default terminal tab, making another
// one, changing to the directory of the active file, and that keys
// go to a focused terminal rather than the Code key bindings.
func terminalText(tm *terminal.Terminal) string {
	return tm.LinesView().Lines.String()
}

// hasOutputLine returns whether a line of the terminal text is s, allowing for
// a leading prompt: dash (sh on Ubuntu) echoes typed-ahead input after the prompt.
func hasOutputLine(tm *terminal.Terminal, s string) bool {
	for _, ln := range strings.Split(terminalText(tm), "\n") {
		if strings.TrimPrefix(strings.TrimRight(ln, " "), "$ ") == s {
			return true
		}
	}
	return false
}

func TestCodeTerminal(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	tmp, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	sub := filepath.Join(tmp, "sub")
	require.NoError(t, os.Mkdir(sub, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(sub, "a.txt"), []byte("a\n"), 0o644))
	prevCmd := Settings.Terminal.Command
	Settings.Terminal.Command = "/bin/sh"
	defer func() { Settings.Terminal.Command = prevCmd }()
	b := core.NewBody()
	b.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(1200), units.Dp(800)) })
	cv := NewCode(b)
	cv.Defaults()
	cv.ProjectRoot = core.Filename(tmp)
	cv.Settings.ProjectRoot = core.Filename(tmp)
	showed := make(chan struct{})
	b.OnFinal(events.Show, func(e events.Event) { showed <- struct{}{} })
	b.RunWindow()
	<-showed
	prevMap := keymap.ActiveMapName
	keymap.SetActiveMapName("MacEmacs") // with Control+X and Control+C key sequences
	defer keymap.SetActiveMapName(prevMap)
	defer func() {
		b.AsyncLock()
		b.Close()
		b.AsyncUnlock()
	}()
	waitFor := func(what string, cond func() bool) {
		t.Helper()
		deadline := time.Now().Add(10 * time.Second)
		for time.Now().Before(deadline) {
			b.AsyncLock()
			ok := cond()
			b.AsyncUnlock()
			if ok {
				return
			}
			time.Sleep(20 * time.Millisecond)
		}
		b.AsyncLock()
		defer b.AsyncUnlock()
		for _, tm := range cv.Terminals() {
			t.Logf("%s text:\n%s", tm.Name, terminalText(tm))
		}
		t.Fatalf("timeout waiting for %s", what)
	}

	var tm *terminal.Terminal
	waitFor("the default terminal", func() bool {
		tms := cv.Terminals()
		if len(tms) == 1 {
			tm = tms[0]
		}
		return tm != nil
	})
	b.AsyncLock()
	assert.Equal(t, TerminalTabName, tm.Name)
	cv.ViewFileInIndex(filepath.Join(sub, "a.txt"), 0)
	cv.OpenTerminal()
	b.AsyncUnlock()
	waitFor("the shell", func() bool { return tm.IsRunning() && strings.Contains(terminalText(tm), "$") })

	b.AsyncLock()
	cv.TerminalCdToFile()
	tm.Run("pwd")
	b.AsyncUnlock()
	waitFor("the directory of the file", func() bool { return hasOutputLine(tm, sub) })

	// keys go to the focused terminal: Control+C interrupts the command
	// rather than starting a key sequence
	b.AsyncLock()
	tm.SetFocus()
	tm.Run("sleep 20")
	b.AsyncUnlock()
	time.Sleep(200 * time.Millisecond)
	src := b.Scene.RenderWindow().SystemWindow.Events()
	var ctrl key.Modifiers
	ctrl.SetFlag(true, key.Control)
	src.Key(events.KeyDown, 'C', key.CodeC, ctrl)
	src.Key(events.KeyUp, 'C', key.CodeC, ctrl)
	src.KeyChord('e', key.CodeUnknown, 0)
	src.KeyChord('c', key.CodeUnknown, 0)
	src.KeyChord('h', key.CodeUnknown, 0)
	src.KeyChord('o', key.CodeUnknown, 0)
	src.KeyChord(' ', key.CodeUnknown, 0)
	src.KeyChord('x', key.CodeUnknown, 0)
	src.Key(events.KeyDown, 0, key.CodeReturnEnter, 0)
	waitFor("the interrupt", func() bool {
		return strings.Contains(terminalText(tm), "echo x\nx\n") && cv.KeySeq1 == ""
	})

	// another terminal
	b.AsyncLock()
	cv.NewTerminal()
	b.AsyncUnlock()
	waitFor("the second terminal", func() bool { return len(cv.Terminals()) == 2 })
	b.AsyncLock()
	tm2 := cv.Terminals()[1]
	assert.Equal(t, TerminalTabName+" 2", tm2.Name)
	assert.Equal(t, tm2, cv.CurrentTerminal())
	b.AsyncUnlock()
	waitFor("the second shell", func() bool { return tm2.IsRunning() })

	// Shift+Left/Right move among the tabs, including ones that are not terminals
	b.AsyncLock()
	tm2.SetFocus()
	b.AsyncUnlock()
	var shift key.Modifiers
	shift.SetFlag(true, key.Shift)
	focusedTab := func() string {
		if cv.CurPanel() != TabsIndex {
			return ""
		}
		_, idx := cv.Tabs().CurrentTab()
		_, tab := cv.Tabs().TabAtIndex(idx)
		return tab.Text
	}
	waitFor("the focus in the second terminal", func() bool { return cv.focusedTerminal() == tm2 })
	src.Key(events.KeyDown, 0, key.CodeLeftArrow, shift)
	waitFor("the previous tab", func() bool { return focusedTab() == "Preview" })
	src.Key(events.KeyDown, 0, key.CodeLeftArrow, shift)
	waitFor("the first terminal", func() bool { return focusedTab() == TerminalTabName && cv.focusedTerminal() == tm })
	src.Key(events.KeyDown, 0, key.CodeRightArrow, shift)
	waitFor("the next tab", func() bool { return focusedTab() == "Preview" })
	src.Key(events.KeyDown, 0, key.CodeRightArrow, shift)
	waitFor("the second terminal again", func() bool { return cv.focusedTerminal() == tm2 })

	// the new terminal key in a terminal
	rn, code, mods, err := terminal.ShortcutChord(keymap.New).Chords()[0].Decode()
	require.NoError(t, err)
	src.Key(events.KeyDown, rn, code, mods)
	waitFor("the third terminal", func() bool {
		tms := cv.Terminals()
		return len(tms) == 3 && cv.focusedTerminal() == tms[2]
	})
	b.AsyncLock()
	assert.Equal(t, TerminalTabName+" 3", cv.Terminals()[2].Name)
	b.AsyncUnlock()

	// the terminal commands are in the context menu
	b.AsyncLock()
	defer b.AsyncUnlock()
	m := core.NewScene()
	cms := tm2.LinesView().ContextMenus
	for i := len(cms) - 1; i >= 0; i-- {
		cms[i](m, image.Point{})
	}
	var items []string
	m.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
		if bt := core.AsButton(cw); bt != nil {
			items = append(items, bt.Text)
		}
		return tree.Continue
	})
	assert.Equal(t, []string{"New terminal", "Cd to project", "Previous tab", "Next tab", "Copy", "Paste", "Clear scrollback"}, items)
}
