// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package commands provides a GUI for calling command line tools, such as
// git, whose many options are hard to remember. The help output of each
// command is parsed into a [Cmd], with its flags and subcommands, which is
// cached (see [Commands]). The user organizes the commands into categories,
// edited in a [core.TreeTable] (see [Commands.Edit]), which are shown in a
// command bar like a bookmarks bar (see [Commands.Bar]). Clicking on a
// command shows a form of its flags and arguments, which builds the command
// line to run, in a terminal for example (see [Commands.RunDialog]).
package commands

import (
	"strings"
)

// Cmd is a command line command, such as "git commit", with the
// information parsed from its help output (see [Parse]).
type Cmd struct {

	// Cmd is the command, with any subcommands: "git commit".
	Cmd string

	// Doc is the documentation of the command, if found.
	Doc string `json:",omitempty"`

	// Usage is the usage synopsis of the command, if found.
	Usage string `json:",omitempty"`

	// Flags are the flags of the command.
	Flags []*Flag `json:",omitempty"`

	// Cmds are the subcommands of the command, with their documentation,
	// which are parsed separately, when they are used.
	Cmds []*Cmd `json:",omitempty"`

	// Source is the help command whose output was parsed: "git commit -h".
	Source string `json:",omitempty"`
}

// Flag is a flag of a [Cmd].
type Flag struct {

	// Name is the canonical (longest) name of the flag, with its dashes:
	// "--message".
	Name string

	// Names are all of the names of the flag, with their dashes, shortest
	// first: "-m", "--message".
	Names []string

	// Value is the hint for the value of the flag, such as "<message>",
	// or empty for a flag without a value, which is on or off.
	Value string `json:",omitempty"`

	// Optional is whether the value is optional, as in "--name[=<value>]":
	// it is attached to the flag, with = for long names.
	Optional bool `json:",omitempty"`

	// Equals is whether the value is attached with =, as in "--name=<value>".
	Equals bool `json:",omitempty"`

	// Negatable is whether the flag has a negated form, as in "--[no-]name".
	Negatable bool `json:",omitempty"`

	// Doc is the documentation of the flag.
	Doc string `json:",omitempty"`
}

// IsBool returns whether the flag has no value: it is on or off.
func (fl *Flag) IsBool() bool {
	return fl.Value == ""
}

// Label returns the label of the flag: its names and value hint.
func (fl *Flag) Label() string {
	s := strings.Join(fl.Names, ", ")
	if fl.Value != "" {
		s += " " + fl.Value
	}
	return s
}

// Arg returns the command line argument(s) for the flag with the given
// value, which is quoted for a shell as needed.
func (fl *Flag) Arg(value string) string {
	if fl.IsBool() {
		return fl.Name
	}
	q := shellQuote(value)
	long := strings.HasPrefix(fl.Name, "--")
	switch {
	case fl.Equals || (fl.Optional && long):
		return fl.Name + "=" + q
	case fl.Optional:
		return fl.Name + q
	}
	return fl.Name + " " + q
}

// shellQuote returns the given string quoted for a POSIX shell, if it has
// any characters that need it.
func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	if !strings.ContainsFunc(s, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("-_./:=,@%+", r))
	}) {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
