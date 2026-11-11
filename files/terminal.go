// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/text/terminal"
)

// TerminalTabName is the name of the tab of the default terminal.
const TerminalTabName = "Terminal"

// NewTerminal makes a new terminal tab, in addition to the default one,
// running in the given directory, and selects it.
func (fl *Files) NewTerminal(dir string) *terminal.Terminal {
	ts := fl.Tabs()
	if ts == nil {
		return nil
	}
	n := 2
	for ts.TabByName(terminalTabName(n)) != nil {
		n++
	}
	tm := fl.recycleTerminal(terminalTabName(n), dir)
	if tm != nil {
		tm.SetFocus()
	}
	return tm
}

// terminalTabName returns the name of the n-th terminal tab.
func terminalTabName(n int) string {
	if n <= 1 {
		return TerminalTabName
	}
	return fmt.Sprintf("%s %d", TerminalTabName, n)
}

// recycleTerminal returns the terminal in the tab with the given name,
// making it if needed, running in the given directory, and selects the tab.
// Terminals run the user's shell, or the command in the [Settings].
func (fl *Files) recycleTerminal(name, dir string) *terminal.Terminal {
	ts := fl.Tabs()
	if ts == nil {
		return nil
	}
	fr := ts.RecycleTab(name)
	if fr.HasChildren() {
		if tm, ok := fr.Child(0).(*terminal.Terminal); ok {
			fl.lastTerminal = tm
			return tm
		}
	}
	return fl.initTerminal(fr, name, dir)
}

// initTerminal makes a terminal with the given name in the given frame of a
// tab of the command tabs, running in the given directory.
func (fl *Files) initTerminal(fr *core.Frame, name, dir string) *terminal.Terminal {
	ts := fl.Tabs()
	fr.Styler(func(s *styles.Style) {
		s.Overflow.Set(styles.OverflowHidden) // the terminal scrolls itself
	})
	tm := terminal.NewTerminal(fr)
	tm.SetName(name)
	tm.SetDir(dir)
	if cmd := strings.Fields(Settings.Terminal); len(cmd) > 0 {
		tm.SetCommand(cmd...)
	}
	td := &terminalDir{dir: cleanDir(dir)}
	setTerminalDir(tm, td)
	tm.OnUserCommand(func(cmd string) {
		// (of the Files that it is in then, as it can be dragged to another)
		if td.cd(cmd) {
			if f, ok := ParentFiles(tm); ok {
				f.updateTerminalLabels()
			}
		}
	})
	ts.Update()
	fl.focusOnTab(fr)
	fl.updateTerminalLabels()
	fl.lastTerminal = tm
	return tm
}

// focusOnTab makes a click on the tab of the given frame of the command
// tabs move the keyboard focus to the terminal in it, once it is shown.
func (fl *Files) focusOnTab(fr *core.Frame) {
	ts := fl.Tabs()
	if ts == nil {
		return
	}
	_, tab := ts.TabAtIndex(fr.IndexInParent())
	if tab == nil {
		return
	}
	tab.OnClick(func(e events.Event) {
		if !fr.HasChildren() {
			return
		}
		if tm, ok := fr.Child(0).(*terminal.Terminal); ok {
			tm.Defer(func() { tm.SetFocus() }) // (after it is shown)
			tm.NeedsRender()
		}
	})
}

// terminalDirs are the working directories of the terminals, for the
// labels of their tabs, of all of the Files windows, as a terminal can be
// dragged from one to another.
var terminalDirs struct {
	sync.Mutex
	dirs map[*terminal.Terminal]*terminalDir
}

// setTerminalDir sets the working directory of the given terminal.
func setTerminalDir(tm *terminal.Terminal, td *terminalDir) {
	terminalDirs.Lock()
	defer terminalDirs.Unlock()
	if terminalDirs.dirs == nil {
		terminalDirs.dirs = map[*terminal.Terminal]*terminalDir{}
	}
	terminalDirs.dirs[tm] = td
}

// getTerminalDir returns the working directory of the given terminal, or nil.
func getTerminalDir(tm *terminal.Terminal) *terminalDir {
	terminalDirs.Lock()
	defer terminalDirs.Unlock()
	return terminalDirs.dirs[tm]
}

// terminalDir is the working directory of a terminal, which follows the
// cd commands that the user enters in it (see [terminalDir.cd]).
type terminalDir struct {

	// dir is the working directory.
	dir string

	// prev is the previous working directory, for cd -.
	prev string
}

// cd updates the directory for the cd commands of the given command line
// entered by the user, returning whether it changed. The commands of the
// line are split at ; and &&, and a command in a pipe, which runs in a
// subshell, does not change the directory of the shell, nor do those that
// scripts run, which are not entered by the user.
func (td *terminalDir) cd(line string) bool {
	changed := false
	for _, cmd := range splitCommands(line) {
		args := shellFields(cmd)
		if len(args) == 0 || args[0] != "cd" || slices.Contains(args, "|") {
			continue
		}
		args = slices.DeleteFunc(args[1:], func(a string) bool { // options, as -P
			return len(a) > 1 && strings.HasPrefix(a, "-")
		})
		var dir string
		switch {
		case len(args) == 0 || args[0] == "~":
			dir, _ = os.UserHomeDir()
		case args[0] == "-":
			dir = td.prev
		default:
			dir = os.ExpandEnv(args[0])
			if rest, ok := strings.CutPrefix(dir, "~/"); ok {
				if home, err := os.UserHomeDir(); err == nil {
					dir = filepath.Join(home, rest)
				}
			}
			if !filepath.IsAbs(dir) {
				dir = filepath.Join(td.dir, dir)
			}
		}
		if dir == "" {
			continue
		}
		if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
			continue // as the cd fails
		}
		dir = cleanDir(dir)
		if dir != td.dir {
			td.prev, td.dir = td.dir, dir
			changed = true
		}
	}
	return changed
}

// splitCommands splits the given command line into its commands
// separated by ; and &&, outside of quotes.
func splitCommands(line string) []string {
	var cmds []string
	var quote rune
	st := 0
	rs := []rune(line)
	for i := 0; i < len(rs); i++ {
		r := rs[i]
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			}
		case r == '\'':
			quote = r
		case r == '"':
			quote = r
		case r == ';':
			cmds = append(cmds, string(rs[st:i]))
			st = i + 1
		case r == '&' && i+1 < len(rs) && rs[i+1] == '&':
			cmds = append(cmds, string(rs[st:i]))
			i++
			st = i + 1
		}
	}
	return append(cmds, string(rs[st:]))
}

// shellFields splits the given command into its words, as the shell does
// with quotes and backslashes, with a | as a word of its own.
func shellFields(cmd string) []string {
	var words []string
	var cur strings.Builder
	inWord := false
	var quote rune
	rs := []rune(cmd)
	for i := 0; i < len(rs); i++ {
		r := rs[i]
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote = r
			inWord = true
		case r == '\\' && i+1 < len(rs):
			i++
			cur.WriteRune(rs[i])
			inWord = true
		case r == ' ' || r == '\t' || r == '|':
			if inWord {
				words = append(words, cur.String())
				cur.Reset()
				inWord = false
			}
			if r == '|' {
				words = append(words, "|")
			}
		default:
			cur.WriteRune(r)
			inWord = true
		}
	}
	if inWord {
		words = append(words, cur.String())
	}
	return words
}

// updateTerminalLabels labels the terminal tabs with the names of their
// working directories, numbered when several have the same name, with
// their full paths as their tooltips.
func (fl *Files) updateTerminalLabels() {
	ts := fl.Tabs()
	if ts == nil {
		return
	}
	count := map[string]int{}
	for i := range ts.NumTabs() {
		fr, tab := ts.TabAtIndex(i)
		if fr == nil || !fr.HasChildren() {
			continue
		}
		tm, ok := fr.Child(0).(*terminal.Terminal)
		td := getTerminalDir(tm)
		if !ok || td == nil {
			continue
		}
		dir := td.dir
		label := dirLabel(dir)
		count[label]++
		if n := count[label]; n > 1 {
			label = fmt.Sprintf("%s %d", label, n)
		}
		tab.SetText(label).SetTooltip(dir)
		tab.Update()
	}
}
