// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"cogentcore.org/core/cli/commands"
	"cogentcore.org/core/text/terminal"
)

// theCommands returns the commands of the command bar, which are the
// [commands.Default] ones, shared by the apps.
func theCommands() *commands.Commands {
	return commands.Default()
}

// RunCommand runs the given command line in the last terminal used, or in
// a new terminal in the current directory if there is none, selecting its
// tab, as if the command line was typed in it.
func (fl *Files) RunCommand(cmdline string) {
	tm := fl.lastTerminal
	if tm == nil || tm.This == nil || !fl.hasTerminal(tm) {
		tm = fl.NewTerminal(fl.CurrentDir())
		if tm == nil {
			return
		}
	} else if ts := fl.Tabs(); ts != nil {
		ts.SelectTabByName(tm.Parent.AsTree().Name)
	}
	fl.lastTerminal = tm
	tm.SetFocus()
	runInTerminal(tm, cmdline)
}

// runInTerminal sends the given command line to the given terminal, with a
// Return to run it.
var runInTerminal = func(tm *terminal.Terminal, cmdline string) {
	tm.SendText(cmdline + "\r")
}
