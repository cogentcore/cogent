// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCodeTerminal tests the default terminal tab, making another
// one, changing to the directory of the active file, and that keys
// go to a focused terminal rather than the Code key bindings.
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
	prevMap := keymap.ActiveMapName
	keymap.SetActiveMapName("MacEmacs") // with Control+X and Control+C key sequences
	defer keymap.SetActiveMapName(prevMap)

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
		t.Fatalf("timeout waiting for %s", what)
	}
	text := func(tm *terminal.Terminal) string {
		return tm.LinesView().Lines.String()
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
	waitFor("the shell", func() bool { return tm.IsRunning() && strings.Contains(text(tm), "$") })

	b.AsyncLock()
	cv.TerminalCdToFile()
	tm.Run("pwd")
	b.AsyncUnlock()
	waitFor("the directory of the file", func() bool { return strings.Contains(text(tm), "\n"+sub+"\n") })

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
		return strings.Contains(text(tm), "echo x\nx\n") && cv.KeySeq1 == ""
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
}
