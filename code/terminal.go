// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"fmt"
	"image"
	"path/filepath"
	"strings"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/text/terminal"
	"cogentcore.org/core/text/textcore"
	"cogentcore.org/core/tree"
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
// The [keymap.New] key does this in a terminal (see [terminal.KeyFunction]).
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
	tm.AddContextMenu(cv.terminalContextMenu)
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

// terminalContextMenu adds the terminal commands to the context menu
// of a terminal.
func (cv *Code) terminalContextMenu(m *core.Scene, pos image.Point) {
	// the shortcuts of these menu items are shown, but they are not active
	// outside of the menu: the keys are handled in [Code.codeKeys]
	core.NewFuncButton(m).SetFunc(cv.NewTerminal).SetText("New terminal").SetIcon(icons.Add).
		SetShortcut(terminal.ShortcutChord(keymap.New))
	core.NewFuncButton(m).SetFunc(cv.TerminalCdToProject).SetText("Cd to project").SetIcon(icons.FolderOpen)
	core.NewSeparator(m)
	core.NewFuncButton(m).SetFunc(cv.PrevTab).SetText("Previous tab").SetIcon(icons.ArrowBack).
		SetShortcut(prevTabChord)
	core.NewFuncButton(m).SetFunc(cv.NextTab).SetText("Next tab").SetIcon(icons.ArrowForward).
		SetShortcut(nextTabChord)
}

// The keys for moving among the tabs, as in macOS Terminal, which work
// when the focus is in the tabs panel (see [Code.tabKeysActive]). They
// are not menu shortcuts, which would apply in the editors.
const (
	prevTabChord key.Chord = "Shift+LeftArrow"
	nextTabChord key.Chord = "Shift+RightArrow"
)

// NextTab selects the next tab in the tabs panel, after the last one
// the first, and focuses it. Shift+RightArrow does this when the focus
// is in the tabs panel.
func (cv *Code) NextTab() { //types:add
	cv.selectTabDelta(1)
}

// PrevTab selects the previous tab in the tabs panel, before the first
// one the last, and focuses it. Shift+LeftArrow does this when the focus
// is in the tabs panel.
func (cv *Code) PrevTab() { //types:add
	cv.selectTabDelta(-1)
}

// selectTabDelta selects the tab the given number of tabs after the
// current one, cycling around, and focuses it.
func (cv *Code) selectTabDelta(delta int) {
	tv := cv.Tabs()
	if tv == nil || tv.NumTabs() == 0 {
		return
	}
	n := tv.NumTabs()
	_, idx := tv.CurrentTab()
	idx = ((idx+delta)%n + n) % n
	fr := tv.SelectTabIndex(idx)
	if fr != nil && fr.HasChildren() {
		if tm, ok := fr.Child(0).(*terminal.Terminal); ok {
			cv.lastTerminal = tm
			tm.SetFocus()
			return
		}
	}
	cv.FocusOnPanel(TabsIndex)
}

// tabKeysActive returns whether the keys for moving among the tabs
// move among them: when the focus is in the tabs panel, except in
// editable text and in full-screen programs in terminals, which use them.
func (cv *Code) tabKeysActive() bool {
	if cv.CurPanel() != TabsIndex {
		return false
	}
	var foc core.Widget
	core.AsWidget(cv.Splits().Child(TabsIndex)).WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
		if cwb.StateIs(states.Focused) {
			foc = cw
			return tree.Break
		}
		return tree.Continue
	})
	switch w := foc.(type) {
	case *terminal.View:
		return w == w.Terminal.LinesView() && w.Terminal.ActiveView() == w
	case *core.TextField:
		return false
	case textcore.BaseEmbedder:
		return w.AsBase().IsReadOnly()
	}
	return true
}
