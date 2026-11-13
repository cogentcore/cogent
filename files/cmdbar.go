// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"cogentcore.org/core/cli/commands"
	"cogentcore.org/core/core"
	"cogentcore.org/core/text/terminal"
	"cogentcore.org/core/tree"
)

// theCommands returns the commands of the command bar, which are the
// [commands.Default] ones, shared by the apps, whose choosers run in the
// current directory of the Files of the window they are used from (see
// [commandsDir]).
func theCommands() *commands.Commands {
	cs := commands.Default()
	cs.Dir = commandsDir
	return cs
}

// commandsDir returns the directory that the commands run in from the given
// context widget (see [commands.Commands.Dir]): the current directory of
// the [Files] of its window, or "" if there is none.
func commandsDir(ctx core.Widget) string {
	fl := tree.ParentByType[*Files](ctx)
	if fl == nil && ctx.AsWidget().Scene != nil {
		ctx.AsWidget().Scene.WalkDown(func(n tree.Node) bool {
			if f, ok := n.(*Files); ok {
				fl = f
				return tree.Break
			}
			return fl == nil
		})
	}
	if fl == nil {
		return ""
	}
	return fl.CurrentDir()
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
