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
	"cogentcore.org/core/events"
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

// cdDefaultTerminal changes the directory of the default terminal, if it
// has been made, to the project root, as when a project is opened in the
// initial empty window, whose terminal starts in its working directory.
func (cv *Code) cdDefaultTerminal() {
	tv := cv.Tabs()
	if tv == nil || cv.ProjectRoot == "" {
		return
	}
	fr := tv.TabByName(TerminalTabName)
	if fr == nil || !fr.AsTree().HasChildren() {
		return
	}
	tm, ok := fr.AsTree().Child(0).(*terminal.Terminal)
	if !ok {
		return
	}
	if tm.IsRunning() {
		tm.Cd(string(cv.ProjectRoot))
	} else {
		tm.SetDir(string(cv.ProjectRoot))
	}
}

// NewTerminal makes a new terminal tab, in addition to the default one.
// The [keymap.New] key does this in a terminal (see [terminal.KeyFunction]).
func (cv *Code) NewTerminal() { //types:add
	cv.TerminalHere(string(cv.ProjectRoot))
}

// TerminalHere makes a new terminal tab, in addition to the default one,
// in the given directory, as from the context menus of the file tree and
// the editors.
func (cv *Code) TerminalHere(dir string) {
	tv := cv.Tabs()
	if tv == nil {
		return
	}
	n := 2
	for tv.TabByName(terminalTabName(n)) != nil {
		n++
	}
	tm := cv.recycleTerminalIn(terminalTabName(n), dir)
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
// making it in the project root directory if needed, and selects the tab.
func (cv *Code) recycleTerminal(name string) *terminal.Terminal {
	return cv.recycleTerminalIn(name, string(cv.ProjectRoot))
}

// recycleTerminalIn returns the terminal in the tab with the given name,
// making it in the given directory if needed, and selects the tab.
func (cv *Code) recycleTerminalIn(name, dir string) *terminal.Terminal {
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
	tm.SetDir(dir)
	if cmd := strings.Fields(Settings.Terminal.Command); len(cmd) > 0 {
		tm.SetCommand(cmd...)
	}
	tv.Update()
	// a click on the tab moves the keyboard focus to the terminal, once
	// it is shown, as in Cogent Files
	if _, tab := tv.TabAtIndex(fr.IndexInParent()); tab != nil {
		tab.OnClick(func(e events.Event) {
			tm.Defer(func() { tm.SetFocus() })
			tm.NeedsRender()
		})
	}
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
		if tm.ContainsFocus() {
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
		SetShortcut(keymap.PrevTab.Chord())
	core.NewFuncButton(m).SetFunc(cv.NextTab).SetText("Next tab").SetIcon(icons.ArrowForward).
		SetShortcut(keymap.NextTab.Chord())
}

// NextTab selects the next tab in the tabs panel, after the last one
// the first, and focuses it, as the [keymap.NextTab] key (Shift+RightArrow)
// does when the focus is in the tabs panel (see [core.Tabs]).
func (cv *Code) NextTab() { //types:add
	if tv := cv.Tabs(); tv != nil {
		tv.SelectTabDelta(1)
	}
}

// PrevTab selects the previous tab in the tabs panel, before the first
// one the last, and focuses it, as the [keymap.PrevTab] key
// (Shift+LeftArrow) does when the focus is in the tabs panel.
func (cv *Code) PrevTab() { //types:add
	if tv := cv.Tabs(); tv != nil {
		tv.SelectTabDelta(-1)
	}
}

// focusedTabsEditor returns the text editor in the tabs panel that has
// the focus, if any. (Read-only editors are not focusable, so they have
// the focus without the focused state.)
func (cv *Code) focusedTabsEditor() *textcore.Base {
	if cv.CurPanel() != TabsIndex {
		return nil
	}
	var ed *textcore.Base
	core.AsWidget(cv.Splits().Child(TabsIndex)).WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
		if !cwb.ContainsFocus() {
			return tree.Break
		}
		if be, ok := cw.(textcore.BaseEmbedder); ok {
			ed = be.AsBase()
			return tree.Break
		}
		return tree.Continue
	})
	return ed
}

// tabsPanelKeys returns whether the keys of the tabs panel work, as the
// Kill key clearing command output (see [Code.clearOutputTab]): when the
// focus is in the tabs panel, except in editable text and in full-screen
// programs in terminals, which use the keys.
func (cv *Code) tabsPanelKeys() bool {
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
