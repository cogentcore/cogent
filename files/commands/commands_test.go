// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/core"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// gitCommitHelp is part of the output of git commit -h.
const gitCommitHelp = `usage: git commit [-a | --interactive | --patch] [-s] [-v] [-u[<mode>]] [--amend]
                  [--dry-run] [(-c | -C | --squash) <commit> | --fixup [(amend|reword):]<commit>]

    -q, --[no-]quiet      suppress summary after successful commit
    -v, --[no-]verbose    show diff in commit message template

Commit message options
    -F, --[no-]file <file>
                          read message from file
    --[no-]author <author>
                          override author for commit
    -m, --[no-]message <message>
                          commit message
    -u, --[no-]untracked-files[=<mode>]
                          show untracked files, optional modes: all, normal, no. (Default: all)
    --[no-]cleanup=<mode> how to strip spaces and #comments from message
`

// coreHelp is part of the output of core -h, from the cli package.
const coreHelp = `Command line tools for developing apps and libraries using the Cogent Core framework.

Usage:
    core [flags]
    core <subcommand> [flags]

Subcommands:
    build
        Build builds an executable for the package
        at the config path for the config platforms
    install

Flags:
  -v, -verbose bool
        whether to run the command in verbose mode
  -name string
        Name is the user-friendly name of the project.
`

func TestSetFromHelp(t *testing.T) {
	cm := &Cmd{Cmd: "git commit"}
	cm.SetFromHelp(gitCommitHelp)
	assert.Equal(t, "git commit [-a | --interactive | --patch] [-s] [-v] [-u[<mode>]] [--amend]\n[--dry-run] [(-c | -C | --squash) <commit> | --fixup [(amend|reword):]<commit>]", cm.Usage)
	require.Len(t, cm.Flags, 7)
	want := []Flag{
		{Name: "--quiet", Names: []string{"-q", "--quiet"}, Negatable: true, Doc: "suppress summary after successful commit"},
		{Name: "--verbose", Names: []string{"-v", "--verbose"}, Negatable: true, Doc: "show diff in commit message template"},
		{Name: "--file", Names: []string{"-F", "--file"}, Value: "<file>", Negatable: true, Doc: "read message from file"},
		{Name: "--author", Names: []string{"--author"}, Value: "<author>", Negatable: true, Doc: "override author for commit"},
		{Name: "--message", Names: []string{"-m", "--message"}, Value: "<message>", Negatable: true, Doc: "commit message"},
		{Name: "--untracked-files", Names: []string{"-u", "--untracked-files"}, Value: "<mode>", Optional: true, Negatable: true, Doc: "show untracked files, optional modes: all, normal, no. (Default: all)"},
		{Name: "--cleanup", Names: []string{"--cleanup"}, Value: "<mode>", Equals: true, Negatable: true, Doc: "how to strip spaces and #comments from message"},
	}
	for i, fl := range cm.Flags {
		assert.Equal(t, want[i], *fl)
	}
	assert.Empty(t, cm.Cmds)

	cm = &Cmd{Cmd: "core"}
	cm.SetFromHelp(coreHelp)
	assert.Equal(t, "core [flags]\ncore <subcommand> [flags]", cm.Usage)
	require.Len(t, cm.Cmds, 2)
	assert.Equal(t, "core build", cm.Cmds[0].Cmd)
	assert.Equal(t, "Build builds an executable for the package at the config path for the config platforms", cm.Cmds[0].Doc)
	assert.Equal(t, "core install", cm.Cmds[1].Cmd)
	require.Len(t, cm.Flags, 2)
	assert.True(t, cm.Flags[0].IsBool(), "a bool value hint is a flag without a value")
	assert.Equal(t, "-verbose", cm.Flags[0].Name)
	assert.Equal(t, "string", cm.Flags[1].Value)
}

func TestStripOverstrike(t *testing.T) {
	assert.Equal(t, "NAME bold", stripOverstrike("N\bNA\bAM\bME\bE b\bbo\bol\bld\bd"))
}

func TestCommandLine(t *testing.T) {
	cm := &Cmd{Cmd: "git commit"}
	cm.SetFromHelp(gitCommitHelp)
	values := cm.FormStruct()
	v := reflect.ValueOf(values).Elem()
	assert.Equal(t, "git commit", cm.CommandLine(values))
	v.FieldByName("Arguments").SetString("a.go b.go")
	v.FieldByName("Verbose").SetBool(true)
	v.FieldByName("Message").SetString("fix the bug's cause")
	v.FieldByName("UntrackedFiles").SetString("no")
	v.FieldByName("Cleanup").SetString("strip")
	assert.Equal(t, `git commit --verbose --message 'fix the bug'\''s cause' --untracked-files=no --cleanup=strip a.go b.go`, cm.CommandLine(values))

	fl := &Flag{Name: "-u", Names: []string{"-u"}, Value: "<mode>", Optional: true}
	assert.Equal(t, "-uall", fl.Arg("all"), "an optional value is attached")
	field, _ := reflect.TypeOf(values).Elem().FieldByName("Message")
	assert.Equal(t, "-m, --message &lt;message&gt;", field.Tag.Get("label"), "shown as markup")
	assert.Equal(t, "commit message", field.Tag.Get("doc"))
}

// TestParse tests parsing the help of real commands, which are skipped if
// they are not installed.
func TestParse(t *testing.T) {
	for _, cmd := range []string{"git", "git commit", "go build", "ls"} {
		if _, err := exec.LookPath(cmd[:2]); err != nil {
			continue
		}
		cm, err := Parse(cmd)
		require.NoError(t, err, cmd)
		assert.NotEmpty(t, cm.Flags, cmd)
		assert.NotEmpty(t, cm.Doc, cmd)
	}
	if _, err := exec.LookPath("git"); err == nil {
		cm, err := Parse("git")
		require.NoError(t, err)
		subs := []string{}
		for _, sub := range cm.Cmds {
			subs = append(subs, sub.Cmd)
		}
		assert.Contains(t, subs, "git commit")
	}
}

// TestCommands tests opening, saving, and caching the commands.
func TestCommands(t *testing.T) {
	file := filepath.Join(t.TempDir(), "commands.json")
	cs := &Commands{filename: file, Parsed: map[string]*Cmd{}}
	cs.Entries = Standard()
	cm := &Cmd{Cmd: "git commit"}
	cm.SetFromHelp(gitCommitHelp)
	cs.Parsed["git commit"] = cm
	require.NoError(t, cs.Save())
	_, err := os.Stat(file)
	require.NoError(t, err)

	cs2 := &Commands{filename: file}
	b, _ := os.ReadFile(file)
	require.NoError(t, jsonUnmarshal(b, cs2))
	assert.Equal(t, "Git", cs2.Entries[0].Label())
	assert.Equal(t, "Commit", cs2.Entries[0].Children[2].Label())
	assert.Equal(t, "git commit", cs2.Entries[0].Children[2].Command)
	got, err := cs2.Cmd("git  commit", false) // cached, not parsed
	require.NoError(t, err)
	assert.Len(t, got.Flags, 7)
}

// TestDialog tests the dialog of the form of a command, and running its
// command line.
func TestDialog(t *testing.T) {
	imagex.UpdateTestImagesFor(t)
	cs := &Commands{Parsed: map[string]*Cmd{}}
	cm := &Cmd{Cmd: "git commit", Doc: "Record changes to the repository"}
	cm.SetFromHelp(gitCommitHelp)
	ran := ""
	d := cs.NewDialog(nil, cm, func(cmdline string) { ran = cmdline })
	d.AssertRender(t, "dialog", func() {
		fm := tree.ChildByType[*core.Form](d)
		require.NotNil(t, fm)
		v := reflect.ValueOf(fm.Struct).Elem()
		v.FieldByName("Message").SetString("the message")
		fm.Update()
		fm.SendChange()
	})
	assert.Equal(t, "", ran, "not run until Run is clicked")
}
