// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"os"
	"path/filepath"
	"sync"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/cli/commands"
	"cogentcore.org/core/core"
	"cogentcore.org/core/text/terminal"
)

// theCommands returns the commands of the command bar, which are the
// [commands.Default] ones, shared by the apps. Those of an earlier version
// of Files, in its own data directory, are copied to their file the first
// time.
func theCommands() *commands.Commands {
	moveCommandsOnce()
	return commands.Default()
}

// moveCommandsOnce copies the files of the commands of an earlier version
// of Files, in its own data directory, to those of the [commands.Default]
// commands, if they do not exist yet, once.
var moveCommandsOnce = sync.OnceFunc(func() {
	if commands.File != "" { // (set, as in tests)
		return
	}
	dir := core.TheApp.CogentCoreDataDir()
	for _, name := range []string{"commands.json", "commands-args.json"} {
		to := filepath.Join(dir, name)
		if _, err := os.Stat(to); err == nil {
			continue
		}
		b, err := os.ReadFile(filepath.Join(core.TheApp.AppDataDir(), name))
		if err != nil {
			continue
		}
		errors.Log(os.MkdirAll(dir, 0o755))
		errors.Log(os.WriteFile(to, b, 0o644))
	}
})

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
