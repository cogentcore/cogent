// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/strcase"
	"cogentcore.org/core/core"
)

// Entry is a command, or a category of commands, in the [Commands], which
// are edited in a [core.TreeTable] (see [Commands.Edit]). The top-level
// entries are the buttons of the command bar (see [Commands.Bar]): those
// of categories have menus of their entries.
type Entry struct {

	// Name is the name of the command or category, which is shown for it.
	// The default for a command is its last word: "Commit" for "git commit".
	Name string

	// Command is the command, such as "git commit", which is empty for a
	// category.
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
	if f := strings.Fields(en.Command); len(f) > 0 {
		return strcase.ToSentence(f[len(f)-1])
	}
	return "Commands"
}

// Commands are the commands of the command bar, organized into categories
// by the user, with the cache of the parsed help of commands, which are
// saved in a JSON file (see [Open]).
type Commands struct {

	// Entries are the top-level entries, which are the buttons of the
	// command bar.
	Entries []*Entry

	// Parsed are the parsed commands, by command, such as "git commit",
	// which are parsed when they are first used (see [Commands.Cmd]).
	Parsed map[string]*Cmd

	// filename is the file the commands are saved in.
	filename string

	// mu protects Parsed and the saving of the file.
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
// they change. The help of the commands that have not been parsed yet is
// parsed in the background (see [Commands.ParseAll]).
func Open(filename string) *Commands {
	cs := &Commands{filename: filename}
	b, err := os.ReadFile(filename)
	if err == nil {
		errors.Log(jsonUnmarshal(b, cs))
	} else {
		cs.Entries = Standard()
	}
	if cs.Parsed == nil {
		cs.Parsed = map[string]*Cmd{}
	}
	go cs.ParseAll()
	return cs
}

// Save saves the commands to their file.
func (cs *Commands) Save() error {
	if cs.filename == "" {
		return nil
	}
	cs.mu.Lock()
	defer cs.mu.Unlock()
	b, err := json.MarshalIndent(cs, "", "\t")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(cs.filename), 0o755); err != nil {
		return err
	}
	return os.WriteFile(cs.filename, b, 0o644)
}

// cached returns the cached parse of the given command, if any.
func (cs *Commands) cached(cmd string) *Cmd {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return cs.Parsed[cmd]
}

// Cmd returns the parsed command for the given command, such as
// "git commit", parsing its help and saving it if it has not been parsed
// yet, or if reparse is true. The documentation of a subcommand defaults
// to that given by its parent command.
func (cs *Commands) Cmd(cmd string, reparse bool) (*Cmd, error) {
	cmd = strings.Join(strings.Fields(cmd), " ")
	if cm := cs.cached(cmd); cm != nil && !reparse {
		return cm, nil
	}
	cm, err := Parse(cmd)
	if err != nil {
		return nil, err
	}
	cs.mu.Lock()
	if cm.Doc == "" {
		if i := strings.LastIndex(cmd, " "); i > 0 {
			if parent := cs.Parsed[cmd[:i]]; parent != nil {
				for _, sub := range parent.Cmds {
					if sub.Cmd == cmd {
						cm.Doc = sub.Doc
					}
				}
			}
		}
	}
	cs.Parsed[cmd] = cm
	cs.mu.Unlock()
	return cm, cs.Save()
}

// ParseAll parses the help of all of the commands that have not been parsed
// yet, and their parent commands (for their subcommands), saving them. It
// is called in the background by [Open], so that the help is cached when
// the commands are used.
func (cs *Commands) ParseAll() {
	var cmds []string
	var add func(ens []*Entry)
	add = func(ens []*Entry) {
		for _, en := range ens {
			add(en.Children)
			if en.Command == "" {
				continue
			}
			f := strings.Fields(en.Command)
			for i := range f { // parents first
				cmds = append(cmds, strings.Join(f[:i+1], " "))
			}
		}
	}
	add(cs.Entries)
	for _, cmd := range cmds {
		if cs.cached(cmd) == nil {
			errors.Log1(cs.Cmd(cmd, false))
		}
	}
}

// Standard returns the standard commands, which the user starts with.
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
		category("Git", "git", "status", "add", "commit", "diff", "log", "show", "branch",
			"switch", "checkout", "merge", "rebase", "cherry-pick", "stash", "fetch",
			"pull", "push", "clone", "tag", "reset", "restore", "rm", "mv", "remote",
			"worktree", "blame", "grep", "clean"),
		category("Go", "go", "build", "run", "test", "install", "get", "mod", "vet",
			"doc", "generate", "clean", "env", "work"),
		category("Core", "core", "build", "run", "pack", "install", "generate", "setup"),
		category("Files", "", "ls", "find", "grep", "du", "tar", "rsync", "chmod"),
	}
}

// jsonUnmarshal unmarshals the given JSON into the given value.
var jsonUnmarshal = json.Unmarshal
