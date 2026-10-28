// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"fmt"
	"path/filepath"
	"strings"

	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/text/terminal"
)

// TerminalTabName is the name of the tab of the default terminal.
const TerminalTabName = "Terminal"

// TerminalSettings are the settings for terminals.
type TerminalSettings struct { //types:add

	// Command is the command to run in terminals, with its arguments,
	// such as goal; the default is the user's shell.
	Command string
}

// OpenTerminal shows the default terminal tab, making it if needed.
// Terminals run the shell (or the command in the terminal settings)
// in the project root directory.
func (cv *Code) OpenTerminal() { //types:add
	tm := cv.recycleTerminal(TerminalTabName)
	if tm != nil {
		tm.SetFocus()
	}
}

// NewTerminal makes a new terminal tab, in addition to the default one.
func (cv *Code) NewTerminal() { //types:add
	tv := cv.Tabs()
	if tv == nil {
		return
	}
	n := 2
	for tv.TabByName(terminalTabName(n)) != nil {
		n++
	}
	tm := cv.recycleTerminal(terminalTabName(n))
	if tm != nil {
		tm.SetFocus()
	}
}

// terminalTabName returns the name of the n-th terminal tab.
func terminalTabName(n int) string {
	if n <= 1 {
		return TerminalTabName
	}
	return fmt.Sprintf("%s %d", TerminalTabName, n)
}

// recycleTerminal returns the terminal in the tab with the given name,
// making it if needed, and selects the tab.
func (cv *Code) recycleTerminal(name string) *terminal.Terminal {
	tv := cv.Tabs()
	if tv == nil {
		return nil
	}
	fr := tv.RecycleTab(name)
	if fr.HasChildren() {
		if tm, ok := fr.Child(0).(*terminal.Terminal); ok {
			cv.lastTerminal = tm
			return tm
		}
	}
	fr.Styler(func(s *styles.Style) {
		s.Overflow.Set(styles.OverflowHidden) // the terminal scrolls itself
	})
	tm := terminal.NewTerminal(fr)
	tm.SetName(name)
	tm.SetDir(string(cv.ProjectRoot))
	if cmd := strings.Fields(Settings.Terminal.Command); len(cmd) > 0 {
		tm.SetCommand(cmd...)
	}
	tv.Update()
	cv.lastTerminal = tm
	return tm
}

// Terminals returns the terminals in the tabs.
func (cv *Code) Terminals() []*terminal.Terminal {
	tv := cv.Tabs()
	if tv == nil {
		return nil
	}
	var tms []*terminal.Terminal
	for i := range tv.NumTabs() {
		fr, _ := tv.TabAtIndex(i)
		if fr != nil && fr.HasChildren() {
			if tm, ok := fr.Child(0).(*terminal.Terminal); ok {
				tms = append(tms, tm)
			}
		}
	}
	return tms
}

// focusedTerminal returns the terminal that has the keyboard focus, or nil.
func (cv *Code) focusedTerminal() *terminal.Terminal {
	for _, tm := range cv.Terminals() {
		if tm.HasStateWithin(states.Focused) {
			return tm
		}
	}
	return nil
}

// CurrentTerminal returns the terminal for terminal commands: the one in
// the selected tab, if it is a terminal, or else the last one used, or
// else the default one, which is made if needed.
func (cv *Code) CurrentTerminal() *terminal.Terminal {
	tv := cv.Tabs()
	if tv == nil {
		return nil
	}
	if ct, _ := tv.CurrentTab(); ct != nil && ct.AsTree().HasChildren() {
		if tm, ok := ct.AsTree().Child(0).(*terminal.Terminal); ok {
			cv.lastTerminal = tm
			return tm
		}
	}
	if tm := cv.lastTerminal; tm != nil && tm.This != nil {
		cv.recycleTerminal(tm.Name) // selects it
		return tm
	}
	return cv.recycleTerminal(TerminalTabName)
}

// TerminalCdToFile changes the directory of the shell in the current
// terminal to the directory of the file in the active editor.
func (cv *Code) TerminalCdToFile() { //types:add
	tv := cv.ActiveEditor()
	if tv == nil || tv.Lines == nil || tv.Lines.Filename() == "" {
		core.MessageSnackbar(cv, "No file in the active editor")
		return
	}
	cv.terminalCd(filepath.Dir(tv.Lines.Filename()))
}

// TerminalCdToProject changes the directory of the shell in the current
// terminal to the project root directory.
func (cv *Code) TerminalCdToProject() { //types:add
	cv.terminalCd(string(cv.ProjectRoot))
}

// terminalCd changes the directory of the current terminal to the given one.
func (cv *Code) terminalCd(dir string) {
	tm := cv.CurrentTerminal()
	if tm == nil {
		return
	}
	tm.Cd(dir)
	tm.SetFocus()
}
