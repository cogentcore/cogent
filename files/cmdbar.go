// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"path/filepath"
	"sync"

	"cogentcore.org/cogent/files/commands"
	"cogentcore.org/core/core"
	"cogentcore.org/core/text/terminal"
)

// CommandsFile is the file of the commands of the command bar, in the data
// directory of the app (see [theCommands]).
var CommandsFile string

// theCommandsOnce opens the commands once, in [theCommands].
var theCommandsOnce = sync.OnceValue(func() *commands.Commands {
	if CommandsFile == "" {
		CommandsFile = filepath.Join(core.TheApp.AppDataDir(), "commands.json")
	}
	return commands.Open(CommandsFile)
})

// theCommands returns the commands of the command bar, opening them from
// the [CommandsFile] the first time.
func theCommands() *commands.Commands {
	return theCommandsOnce()
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
