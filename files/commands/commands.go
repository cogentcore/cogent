// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package commands

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"sync"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/core"
)

// Entry is a command, or a category of commands, in the [Commands], which
// are edited in a [core.TreeTable] (see [Commands.Edit]). The top-level
// entries are the buttons of the command bar (see [Commands.Bar]): those
// of categories have menus of their entries, and those of commands with
// subcommands, such as git, have menus of their subcommands.
type Entry struct {

	// Name is the name of the command or category, which is shown for it.
	// The default for a command is its last word: "Git" for "git", and
	// "Commit" for "git commit".
	Name string

	// Command is the command, such as "git", or a subcommand, such as
	// "git commit", which is empty for a category.
	Command string `width:"30"`

	// Children are the entries in the category.
	Children []*Entry `display:"-"`

	// Closed is whether the category is closed in the editor.
	Closed bool `display:"-" treetable:"closed"`
}

// IsCategory returns whether the entry is a category of commands: it has
// children, or no command.
func (en *Entry) IsCategory() bool {
	return len(en.Children) > 0 || en.Command == ""
}

// Label returns the text shown for the entry: its name, or else the last
// word of its command.
func (en *Entry) Label() string {
	if en.Name != "" {
		return en.Name
	}
	return cmp.Or(label(en.Command), "Commands")
}

// version is the version of the format of the file of the [Commands]:
// the parsed commands of other versions are parsed again.
const version = 2

// Commands are the commands of the command bar, organized into categories
// by the user, with the cache of the parsed help of commands, which are
// saved in a JSON file (see [Open]), and the arguments last used for them,
// which are saved in another (see [Args]).
type Commands struct {

	// Version is the version of the format of the file.
	Version int

	// Entries are the top-level entries, which are the buttons of the
	// command bar.
	Entries []*Entry

	// Parsed are the parsed commands, by command, such as "git", which are
	// parsed when they are first used (see [Commands.SubCmd]).
	Parsed map[string]*Cmd

	// Args are the arguments last used for the subcommands, which are saved
	// in their own file.
	Args *Args `json:"-"`

	// file is the file the commands are saved in, which other processes
	// (apps) may share.
	file sharedFile

	// mu protects the commands and the saving of the file.
	mu sync.Mutex

	// views are the command bars and the editor of the commands, which are
	// updated when the commands change (see [Commands.Changed]).
	views struct {
		sync.Mutex
		bars   []*core.Toolbar
		editor *core.TreeTable
	}
}

// Open returns the commands saved in the given JSON file, or the
// [Standard] ones if it does not exist yet, which are saved in it when
// they change, with the arguments last used for them, saved in the file of
// the same name ending in -args.json (see [OpenArgs]). The files may be
// shared by other processes (apps): they are read again when another
// process changes them. The help of the commands that have not been parsed
// yet is parsed in the background (see [Commands.ParseAll]).
func Open(filename string) *Commands {
	cs := &Commands{file: sharedFile{path: filename}, Parsed: map[string]*Cmd{}}
	cs.mu.Lock()
	cs.syncLocked()
	if cs.Version < 2 { // (the first version had categories of git subcommands)
		cs.Entries = Standard()
	}
	cs.mu.Unlock()
	args := ""
	if filename != "" {
		args = strings.TrimSuffix(filename, ".json") + "-args.json"
	}
	cs.Args = OpenArgs(args)
	go cs.ParseAll()
	return cs
}

// syncLocked reads the file of the commands if another process has
// changed it, under the mutex, returning whether it did. The parsed
// commands of the file are merged with those parsed here.
func (cs *Commands) syncLocked() bool {
	if !cs.file.changed() {
		return false
	}
	var fc struct {
		Version int
		Entries []*Entry
		Parsed  map[string]*Cmd
	}
	if errors.Log(cs.file.read(&fc)) != nil {
		return false
	}
	if fc.Version < 2 {
		return true
	}
	cs.Version = max(cs.Version, fc.Version)
	cs.Entries = fc.Entries
	if fc.Version != version {
		return true
	}
	for root, fcm := range fc.Parsed {
		if cm := cs.Parsed[root]; cm != nil {
			for _, sc := range cm.SubCmds {
				if !sc.IsParsed() {
					continue
				}
				if i := slices.IndexFunc(fcm.SubCmds, func(f *SubCmd) bool { return f.Name == sc.Name }); i < 0 {
					fcm.SubCmds = append(fcm.SubCmds, sc)
				} else if !fcm.SubCmds[i].IsParsed() {
					fcm.SubCmds[i] = sc
				}
			}
		}
		cs.Parsed[root] = fcm
	}
	return true
}

// reload reads the file of the commands if another process has changed
// it, updating the command bars and the editor if it did.
func (cs *Commands) reload() {
	cs.mu.Lock()
	changed := cs.syncLocked()
	cs.mu.Unlock()
	if changed {
		cs.updateViews(nil)
	}
}

// update changes the commands with the given function, and saves them,
// with the file locked, after reading the changes of other processes.
func (cs *Commands) update(change func()) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return cs.file.update(func() { cs.syncLocked() }, func() {
		change()
		cs.Version = version
	}, cs)
}

// Save saves the commands to their file, with the entries as they are
// here, as edited, and the parsed commands merged with those of other
// processes.
func (cs *Commands) Save() error {
	cs.mu.Lock()
	ens := cs.Entries
	cs.mu.Unlock()
	return cs.update(func() { cs.Entries = ens })
}

// cached returns the cached parse of the given command, such as "git",
// if any, as a copy, which is not changed by parsing.
func (cs *Commands) cached(cmd string) *Cmd {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.syncLocked()
	cm := cs.Parsed[cmd]
	if cm == nil {
		return nil
	}
	c := *cm
	c.SubCmds = slices.Clone(cm.SubCmds)
	return &c
}

// cachedSub returns the cached parse of the given subcommand, such as
// "git commit", or "ls" for the [MainName] one, if it has been parsed.
func (cs *Commands) cachedSub(cmd string) *SubCmd {
	root, name := splitCmd(cmd)
	cm := cs.cached(root)
	if cm == nil {
		return nil
	}
	if sc := cm.SubCmd(cmp.Or(name, MainName)); sc != nil && sc.IsParsed() {
		return sc
	}
	return nil
}

// Cmd returns the parsed command for the given command, such as "git",
// parsing its help and saving it if it has not been parsed yet, or if
// reparse is true, keeping the subcommands that have been parsed.
func (cs *Commands) Cmd(cmd string, reparse bool) (*Cmd, error) {
	if cm := cs.cached(cmd); cm != nil && !reparse {
		return cm, nil
	}
	cm, err := ParseCmd(cmd)
	if err != nil {
		return nil, err
	}
	err = cs.update(func() {
		if old := cs.Parsed[cmd]; old != nil {
			for i, sc := range cm.SubCmds {
				if osc := old.SubCmd(sc.Name); osc != nil && osc.IsParsed() && !sc.IsParsed() {
					cm.SubCmds[i] = osc
				}
			}
		}
		cs.Parsed[cmd] = cm
	})
	return cs.cached(cmd), err
}

// SubCmd returns the parsed subcommand for the given command line, such
// as "git commit", or "ls" for the [MainName] subcommand of a command
// without subcommands, parsing its help and saving it if it has not been
// parsed yet, or if reparse is true. Its documentation defaults to that
// given by its command (or parent subcommand). A subcommand that is not
// one of those of the command, such as "git stash list", is added to them.
func (cs *Commands) SubCmd(cmd string, reparse bool) (*SubCmd, error) {
	cmd = strings.Join(strings.Fields(cmd), " ")
	if sc := cs.cachedSub(cmd); sc != nil && !reparse {
		return sc, nil
	}
	root, name := splitCmd(cmd)
	cm, err := cs.Cmd(root, false)
	if err != nil {
		return nil, err
	}
	if name == "" {
		if cm.HasSubCmds() {
			return nil, fmt.Errorf("commands: %s has subcommands", root)
		}
		name = MainName
	}
	if name == MainName && reparse {
		if _, err := cs.Cmd(root, true); err != nil {
			return nil, err
		}
		return cs.cachedSub(cmd), nil
	}
	sc, err := Parse(cmd)
	if err != nil {
		return nil, err
	}
	sc.Name = name
	err = cs.update(func() {
		pc := cs.Parsed[root]
		if pc == nil {
			pc = cm
			cs.Parsed[root] = pc
		}
		doc := ""
		for _, other := range pc.SubCmds { // the doc of the command, or of the parent subcommand
			if other.Name == name {
				doc = cmp.Or(doc, other.Doc)
			}
			for _, sub := range other.Cmds {
				if sub.Cmd == cmd {
					doc = cmp.Or(doc, sub.Doc)
				}
			}
		}
		sc.Doc = cmp.Or(sc.Doc, doc)
		if i := slices.IndexFunc(pc.SubCmds, func(o *SubCmd) bool { return o.Name == name }); i >= 0 {
			pc.SubCmds[i] = sc
		} else {
			pc.SubCmds = append(pc.SubCmds, sc)
		}
	})
	return sc, err
}

// ParseAll parses the help of all of the commands that have not been parsed
// yet, and of their subcommands (but not those of the subcommands), saving
// them, and then updates the command bars. It is called in the background
// by [Open], so that the help is cached when the commands are used.
func (cs *Commands) ParseAll() {
	var cmds []string
	var add func(ens []*Entry)
	add = func(ens []*Entry) {
		for _, en := range ens {
			add(en.Children)
			if en.Command != "" {
				cmds = append(cmds, en.Command)
			}
		}
	}
	add(cs.Entries)
	for _, cmd := range cmds {
		root, name := splitCmd(cmd)
		cm, err := cs.Cmd(root, false)
		if errors.Log(err) != nil {
			continue
		}
		subs := []string{name}
		if name == "" {
			subs = nil
			for _, sc := range cm.SubCmds {
				subs = append(subs, sc.Name)
			}
		}
		for _, sub := range subs {
			full := root
			if sub != MainName {
				full += " " + sub
			}
			if cs.cachedSub(full) == nil {
				errors.Log1(cs.SubCmd(full, false))
			}
		}
	}
	cs.updateViews(nil)
}

// Standard returns the standard commands, which the user starts with: git,
// go, and core, with the menus of their subcommands, and a category of
// commands for files.
func Standard() []*Entry {
	category := func(name, root string, subs ...string) *Entry {
		en := &Entry{Name: name, Closed: true}
		for _, sub := range subs {
			cmd := sub
			if root != "" {
				cmd = root + " " + sub
			}
			en.Children = append(en.Children, &Entry{Command: cmd})
		}
		return en
	}
	return []*Entry{
		{Command: "git"},
		{Command: "go"},
		{Command: "core"},
		category("Files", "", "ls", "find", "grep", "du", "tar", "rsync", "chmod"),
	}
}
