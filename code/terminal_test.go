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
	"sync"
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
		return hasOutputLine(tm, "x") && cv.KeySeq1 == ""
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

	// in the read-only editor of command output, they move among the tabs
	// too, rather than selecting
	b.AsyncLock()
	_, oed, _ := cv.RecycleCmdTab("Output")
	b.AsyncUnlock()
	b.AsyncLock()
	oed.SetFocus()
	b.AsyncUnlock()
	waitFor("the focus in the output", func() bool { return focusedTab() == "Output" && oed.ContainsFocus() })
	src.Key(events.KeyDown, 0, key.CodeLeftArrow, shift)
	waitFor("the second terminal, before the output", func() bool { return cv.focusedTerminal() == tm2 })

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
	assert.Equal(t, []string{"New terminal", "Cd to project", "Previous tab", "Next tab", "Copy", "Select all", "Paste", "Clear scrollback"}, items)
}

// TestTerminalTabFocus tests that a click on the tab of a terminal moves
// the keyboard focus to the terminal, as in Cogent Files.
func TestTerminalTabFocus(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	tmp, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	file := filepath.Join(tmp, "a.txt")
	require.NoError(t, os.WriteFile(file, []byte("a\n"), 0o644))
	prevCmd := Settings.Terminal.Command
	Settings.Terminal.Command = "/bin/sh"
	defer func() { Settings.Terminal.Command = prevCmd }()
	b := core.NewBody()
	b.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(1200), units.Dp(800)) })
	cv := NewCode(b)
	cv.Defaults()
	cv.ProjectRoot = core.Filename(tmp)
	cv.Settings.ProjectRoot = core.Filename(tmp)
	var once sync.Once
	showed := make(chan struct{})
	b.OnFinal(events.Show, func(e events.Event) { once.Do(func() { close(showed) }) })
	b.RunWindow()
	<-showed
	defer func() {
		b.AsyncLock()
		b.Close()
		b.AsyncUnlock()
	}()
	waitFor := func(what string, cond func() bool) bool {
		deadline := time.Now().Add(10 * time.Second)
		for time.Now().Before(deadline) {
			b.AsyncLock()
			ok := cond()
			b.AsyncUnlock()
			if ok {
				return true
			}
			time.Sleep(20 * time.Millisecond)
		}
		t.Errorf("timeout waiting for %s", what)
		return false
	}

	b.AsyncLock()
	cv.NewTerminal()
	tms := cv.Terminals()
	tm := tms[len(tms)-1]
	cv.ViewFileInIndex(file, 0)
	cv.ActiveEditor().SetFocus()
	b.AsyncUnlock()
	if !waitFor("the focus in the editor", func() bool { return !tm.ContainsFocus() }) {
		return
	}
	b.AsyncLock()
	ts := cv.Tabs()
	var tab *core.Tab
	for i := range ts.NumTabs() {
		if fr, tb := ts.TabAtIndex(i); fr != nil && fr.HasChildren() && fr.Child(0) == tm {
			tab = tb
		}
	}
	if tab != nil {
		tab.Send(events.Click)
	}
	b.AsyncUnlock()
	require.NotNil(t, tab, "the tab of the terminal")
	waitFor("the focus in the terminal", func() bool { return tm.ContainsFocus() })

	// moving among the panels (Shift+arrows), to the panel of the tabs
	b.AsyncLock()
	cv.ActiveEditor().SetFocus()
	b.AsyncUnlock()
	waitFor("the focus in the editor", func() bool { return !tm.ContainsFocus() })
	b.AsyncLock()
	for range len(cv.Splits().Children) {
		if tm.ContainsFocus() {
			break
		}
		cv.FocusNextPanel()
	}
	b.AsyncUnlock()
	waitFor("the focus in the terminal, from the panels", func() bool { return tm.ContainsFocus() })

	// moving among the tabs (Shift+arrows), to the tab of a terminal that
	// has not had the focus, and back
	b.AsyncLock()
	cv.NewTerminal()
	tms = cv.Terminals()
	tm3 := tms[len(tms)-1]
	cv.SelectTabByName(tm.Name) // (without the focus in tm3)
	tm.SetFocus()
	b.AsyncUnlock()
	waitFor("the focus in the terminal", func() bool { return tm.ContainsFocus() && !tm3.ContainsFocus() })
	inTab := func(tm *terminal.Terminal) bool {
		ct, _ := cv.Tabs().CurrentTab()
		return ct != nil && ct.AsTree().HasChildren() && ct.AsTree().Child(0) == tm
	}
	// with the keys, in the terminal that has the focus
	src := b.Scene.RenderWindow().SystemWindow.Events()
	shift := key.Modifiers(0)
	shift.SetFlag(true, key.Shift)
	press := func(code key.Codes) {
		src.Key(events.KeyDown, 0, code, shift)
		src.Key(events.KeyUp, 0, code, shift)
	}
	curTab := func() string {
		b.AsyncLock()
		defer b.AsyncUnlock()
		ct, _ := cv.Tabs().CurrentTab()
		return ct.AsTree().Name
	}
	// pressTo presses the key until the tab of the given terminal is
	// selected, waiting for each tab after another
	pressTo := func(code key.Codes, to *terminal.Terminal) {
		for range 10 {
			b.AsyncLock()
			in := inTab(to)
			b.AsyncUnlock()
			if in {
				return
			}
			prev := curTab()
			press(code)
			waitFor("another tab", func() bool {
				ct, _ := cv.Tabs().CurrentTab()
				return ct.AsTree().Name != prev
			})
		}
	}
	pressTo(key.CodeRightArrow, tm3) // (after the other tabs)
	waitFor("the focus in the next terminal, from the tabs", func() bool { return tm3.ContainsFocus() })
	pressTo(key.CodeLeftArrow, tm)
	waitFor("the focus in the terminal, from the tabs", func() bool { return tm.ContainsFocus() })
}

// TestOpenProjectTerminalDir tests that opening a project in the initial
// empty window changes the directory of its default terminal, which starts
// in the working directory, to the project root.
func TestOpenProjectTerminalDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	wd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(wd) // (opening a project changes it)
	tmp, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	proj := filepath.Join(tmp, "proj")
	require.NoError(t, os.Mkdir(proj, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(proj, "a.txt"), []byte("a\n"), 0o644))
	prevCmd := Settings.Terminal.Command
	Settings.Terminal.Command = "/bin/sh"
	defer func() { Settings.Terminal.Command = prevCmd }()
	b := core.NewBody()
	b.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(1200), units.Dp(800)) })
	cv := NewCode(b)
	cv.Defaults()
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
	var tm *terminal.Terminal
	waitFor("the default terminal", func() bool {
		if tms := cv.Terminals(); len(tms) > 0 {
			tm = tms[0]
		}
		return tm != nil && tm.IsRunning() && strings.Contains(terminalText(tm), "$")
	})
	b.AsyncLock()
	require.True(t, cv.IsEmpty())
	ncv := cv.OpenPath(core.Filename(proj))
	b.AsyncUnlock()
	require.Equal(t, cv, ncv, "in the same window")
	waitFor("the cd", func() bool { return strings.Contains(terminalText(tm), "cd ") })
	b.AsyncLock()
	tm.SendText("pwd\r")
	b.AsyncUnlock()
	waitFor("the project root", func() bool { return hasOutputLine(tm, proj) })
}
