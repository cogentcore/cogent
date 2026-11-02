// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"strings"

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
	fr.Styler(func(s *styles.Style) {
		s.Overflow.Set(styles.OverflowHidden) // the terminal scrolls itself
	})
	tm := terminal.NewTerminal(fr)
	tm.SetName(name)
	tm.SetDir(dir)
	if cmd := strings.Fields(Settings.Terminal); len(cmd) > 0 {
		tm.SetCommand(cmd...)
	}
	ts.Update()
	fl.lastTerminal = tm
	return tm
}
