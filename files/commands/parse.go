// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package commands

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode"
)

// HelpTimeout is the time a help command can take before it is stopped.
var HelpTimeout = 10 * time.Second

// ParseCmd returns the [Cmd] for the given command, such as git, parsed
// from the output of its help commands (see [Parse]), with its subcommands
// to be parsed when they are used, or else its one [MainName] subcommand,
// which is the command itself, parsed.
func ParseCmd(cmd string) (*Cmd, error) {
	sc, err := Parse(cmd)
	if err != nil {
		return nil, err
	}
	cm := &Cmd{Cmd: sc.Cmd, Doc: sc.Doc}
	if len(sc.Cmds) == 0 {
		sc.Name = MainName
		cm.SubCmds = []*SubCmd{sc}
		return cm, nil
	}
	for _, sub := range sc.Cmds {
		_, sub.Name = splitCmd(sub.Cmd)
		cm.SubCmds = append(cm.SubCmds, sub)
	}
	return cm, nil
}

// Parse returns the [SubCmd] for the given command, such as "git commit",
// parsed from the output of its help commands: those of -h, --help, help,
// and man, as the commands have them. The flags are those of the output
// with the most flags, and the subcommands those of the output with the most
// subcommands, other than a man page, whose indented lists are not commands.
func Parse(cmd string) (*SubCmd, error) {
	words := strings.Fields(cmd)
	if len(words) == 0 {
		return nil, errors.New("commands: no command to parse")
	}
	dir, err := os.MkdirTemp("", "commands-help")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	var best, cmds *SubCmd
	doc, manDoc, usage := "", "", ""
	for _, hc := range helpCommands(words) {
		out := runHelp(dir, hc)
		if strings.TrimSpace(out) == "" {
			continue
		}
		cm := &SubCmd{Cmd: strings.Join(words, " "), Source: strings.Join(hc, " ")}
		cm.SetFromHelp(out)
		// the indented lists of man pages (which help commands may show) are
		// not subcommands; a man page of the same name may be about something
		// else (as for core)
		man := hc[0] == "man" || isManPage(out)
		if man {
			cm.Cmds = nil
		}
		if best == nil || flagScore(cm) > flagScore(best) {
			best = cm
		}
		if cmds == nil || len(cm.Cmds) > len(cmds.Cmds) {
			cmds = cm
		}
		if hc[0] == "man" {
			manDoc = cm.Doc
		} else {
			doc = cmp.Or(doc, cm.Doc)
		}
		usage = cmp.Or(usage, cm.Usage)
	}
	if best == nil {
		return nil, fmt.Errorf("commands: no help found for %q", cmd)
	}
	best.Cmds = cmds.Cmds
	if strings.HasPrefix(best.Source, "man ") { // else it may be about something else
		doc = cmp.Or(doc, manDoc)
	}
	best.Doc = cmp.Or(best.Doc, doc)
	best.Usage = cmp.Or(best.Usage, usage)
	return best, nil
}

// flagScore returns the score of the flags of the given parse, for choosing
// the help output with the most flags, and then the most documented ones.
func flagScore(cm *SubCmd) int {
	n := 2 * len(cm.Flags)
	for _, fl := range cm.Flags {
		if fl.Doc != "" {
			n++
		}
	}
	return n
}

// helpCommands returns the commands to get the help of the given command.
func helpCommands(words []string) [][]string {
	with := func(args ...string) []string {
		return append(append([]string{}, words...), args...)
	}
	hcs := [][]string{with("-h"), with("--help")}
	if len(words) > 1 {
		hcs = append(hcs, append([]string{words[0], "help"}, words[1:]...))
		hcs = append(hcs, []string{"man", strings.Join(words, "-")})
	} else {
		// (git help -a has all of the commands of git, not just the common ones)
		hcs = append(hcs, with("help"), with("help", "-a"), []string{"man", words[0]})
	}
	return hcs
}

// runHelp runs the given help command in the given (empty) directory,
// with no input, returning its output, with that of errors, which is empty
// if the command was not found. The help commands of some programs exit
// with an error.
func runHelp(dir string, args []string) string {
	ctx, cancel := context.WithTimeout(context.Background(), HelpTimeout)
	defer cancel()
	if _, err := exec.LookPath(args[0]); err != nil {
		// apps started from the desktop may not have the PATH of the shell
		sh := os.Getenv("SHELL")
		if sh == "" {
			return ""
		}
		quoted := make([]string, len(args))
		for i, a := range args {
			quoted[i] = shellQuote(a)
		}
		args = []string{sh, "-l", "-c", strings.Join(quoted, " ")}
	}
	c := exec.CommandContext(ctx, args[0], args[1:]...)
	c.Dir = dir
	c.Env = append(os.Environ(), "PAGER=cat", "MANPAGER=cat", "GIT_PAGER=cat", "MANWIDTH=100", "TERM=dumb")
	var out bytes.Buffer
	c.Stdout, c.Stderr = &out, &out
	err := c.Run()
	var exitErr *exec.ExitError
	if err != nil && !errors.As(err, &exitErr) {
		return ""
	}
	return stripOverstrike(out.String())
}

// isManPage returns whether the given help output is a man page, with the
// NAME and SYNOPSIS headings.
func isManPage(out string) bool {
	return strings.Contains(out, "\nNAME\n") && strings.Contains(out, "\nSYNOPSIS\n")
}

// overstrike matches the overstrike formatting of man pages: a character
// followed by a backspace.
var overstrike = regexp.MustCompile(".\b")

// stripOverstrike removes the overstrike formatting of man pages.
func stripOverstrike(s string) string {
	return overstrike.ReplaceAllString(s, "")
}

// block is a name, such as that of a flag or a subcommand, with its
// documentation, in help output, under a section heading, such as
// "The commands are:".
type block struct {
	name, doc, section string
	indent             int
}

// SetFromHelp sets the documentation, usage, flags, and subcommands of the
// command from the given help output.
func (cm *SubCmd) SetFromHelp(help string) {
	lines := strings.Split(strings.ReplaceAll(help, "\r\n", "\n"), "\n")
	usage, doc := usageAndDoc(lines)
	cm.Usage = cmp.Or(usage, cm.Usage)
	cm.Doc = cmp.Or(doc, cm.Doc)
	flags := map[string]bool{}
	cmds := map[string]bool{}
	for _, b := range helpBlocks(lines) {
		if strings.HasPrefix(b.name, "-") {
			fl := parseFlag(b.name)
			if fl == nil || flags[fl.Name] {
				continue
			}
			if fl.Value == "bool" { // as in the help of Go flag packages
				fl.Value = ""
			}
			flags[fl.Name] = true
			fl.Doc = b.doc
			cm.Flags = append(cm.Flags, fl)
			continue
		}
		if !isSubcommand(b.name) || cmds[b.name] || b.name == "help" || !commandSection(b.section) {
			continue
		}
		cmds[b.name] = true
		cm.Cmds = append(cm.Cmds, &SubCmd{Cmd: cm.Cmd + " " + b.name, Doc: b.doc, Group: groupName(b.section)})
	}
	if len(cm.Flags) == 0 { // as for go mod tidy
		cm.Flags = usageFlags(cm.Usage, lines)
	}
	// a section with one subcommand is an example in the text, as in the
	// help of go test
	n := map[string]int{}
	for _, sub := range cm.Cmds {
		n[sub.Group]++
	}
	cm.Cmds = slices.DeleteFunc(cm.Cmds, func(sub *SubCmd) bool { return n[sub.Group] < 2 })
	if len(cm.Cmds) > 0 {
		return
	}
	// subcommands that are only in the usage lines, as for git stash list,
	// with their usage as their documentation
	for _, line := range strings.Split(cm.Usage, "\n") {
		rest, ok := strings.CutPrefix(line, cm.Cmd+" ")
		f := strings.Fields(rest)
		if !ok || len(f) == 0 || !isSubcommand(f[0]) || cmds[f[0]] || f[0] == "help" {
			continue
		}
		cmds[f[0]] = true
		cm.Cmds = append(cm.Cmds, &SubCmd{Cmd: cm.Cmd + " " + f[0], Doc: line})
	}
}

// usageFlag matches a bracketed flag in a usage line, such as "[-v]",
// "[-go=version]", "[-C dir]", or "[-a | -b]", not containing brackets.
var usageFlag = regexp.MustCompile(`\[(-[^\[\]]*)\]`)

// usageFlags returns the flags in the given usage lines, for help output
// that only has them there, as for go mod tidy, with their documentation
// from the given lines of the help output: the sentence about each, such
// as "The -v flag causes tidy to print ...".
func usageFlags(usage string, lines []string) []*Flag {
	text := strings.Join(strings.Fields(strings.Join(lines, " ")), " ")
	var flags []*Flag
	seen := map[string]bool{}
	for _, m := range usageFlag.FindAllStringSubmatch(usage, -1) {
		for _, alt := range strings.Split(m[1], "|") {
			fl := parseFlag(strings.TrimSpace(alt))
			if fl == nil || seen[fl.Name] || len(strings.Fields(alt)) > 2 {
				continue
			}
			seen[fl.Name] = true
			fl.Doc = flagSentence(text, fl.Name)
			flags = append(flags, fl)
		}
	}
	return flags
}

// flagSentence returns the sentence of the given text that is about the
// given flag, starting with "The -v flag", or "" if there is none.
func flagSentence(text, name string) string {
	i := strings.Index(text, "The "+name+" flag")
	if i < 0 {
		return ""
	}
	s := text[i:]
	if j := strings.Index(s, ". "); j >= 0 {
		s = s[:j+1]
	}
	return s
}

// commandSection returns whether the given section of help output can
// have subcommands: not help topics, as of go, or the low-level commands,
// interfaces, and external commands of git help -a.
func commandSection(section string) bool {
	s := strings.ToLower(section)
	for _, not := range []string{"topic", "low-level", "interface", "external", "guide"} {
		if strings.Contains(s, not) {
			return false
		}
	}
	return true
}

// groupName returns the name of the group of the subcommands in the given
// section of help output: its heading, without a colon, or "" for the
// usual headings, such as "The commands are:".
func groupName(section string) string {
	s := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(section), ":"))
	switch strings.ToLower(s) {
	case "the commands are", "commands", "subcommands", "available commands":
		return ""
	}
	return s
}

// isSubcommand returns whether the given name of a block is that of a
// subcommand: a lowercase word.
func isSubcommand(name string) bool {
	if name == "" || !unicode.IsLower(rune(name[0])) {
		return false
	}
	return !strings.ContainsFunc(name, func(r rune) bool {
		return !(unicode.IsLower(r) || unicode.IsDigit(r) || r == '-' || r == '_')
	})
}

// indentOf returns the indentation of the given line, with tabs to 8.
func indentOf(line string) int {
	n := 0
	for _, r := range line {
		switch r {
		case ' ':
			n++
		case '\t':
			n += 8 - n%8
		default:
			return n
		}
	}
	return n
}

// nameGap matches the gap between a name and its documentation on one
// line: two or more spaces, or a tab.
var nameGap = regexp.MustCompile(`\s{2,}|\t`)

// helpBlocks returns the blocks of names with their documentation in the
// given lines of help output: indented lines starting with a name, followed
// by its documentation, after a gap on the same line, or on more indented
// lines after it.
func helpBlocks(lines []string) []block {
	var blocks []block
	var cur *block
	section := ""
	finish := func() {
		if cur != nil && cur.name != "" {
			blocks = append(blocks, *cur)
		}
		cur = nil
	}
	for _, line := range lines {
		text := strings.TrimSpace(line)
		n := indentOf(line)
		if n == 0 && text != "" {
			section = text
		}
		switch {
		case text == "" || n == 0:
			finish()
		case cur != nil && n > cur.indent:
			if cur.doc != "" {
				cur.doc += " "
			}
			cur.doc += text
		case strings.HasPrefix(text, "-") || isSubcommand(text) || nameGap.MatchString(text):
			finish()
			cur = &block{name: text, indent: n, section: section}
			if loc := nameGap.FindStringIndex(text); loc != nil {
				cur.name, cur.doc = text[:loc[0]], strings.TrimSpace(text[loc[1]:])
			} else if strings.HasPrefix(text, "-") {
				cur.name, cur.doc = splitFlagDoc(text)
			}
		default:
			finish()
		}
	}
	finish()
	return blocks
}

// splitFlagDoc splits the given line of help output that starts with a
// flag, with no gap of two spaces before its documentation, into the
// flag, with its value hint, and its documentation, which are separated by
// one space when the flag fills the column of flags. There is no
// documentation unless three or more words follow the flag and any value
// hint, so that "-C dir" is a flag with a value.
func splitFlagDoc(text string) (name, doc string) {
	words := strings.Fields(text)
	i := 0
	for i < len(words) && strings.HasPrefix(words[i], "-") {
		i++
	}
	if i < len(words) && strings.ContainsAny(words[i][:1], "<[") {
		i++ // a value hint
	}
	if len(words)-i < 3 {
		return text, ""
	}
	return strings.Join(words[:i], " "), strings.Join(words[i:], " ")
}

// flagName matches a valid name of a flag.
var flagName = regexp.MustCompile(`^--?[A-Za-z0-9][A-Za-z0-9_.-]*$`)

// parseFlag returns the flag of the given name of a block of help output,
// such as "-m, --[no-]message <message>", or nil if it has no flag names.
func parseFlag(name string) *Flag {
	fl := &Flag{}
	for _, tok := range strings.Fields(strings.ReplaceAll(name, ",", " ")) {
		if !strings.HasPrefix(tok, "-") {
			if fl.Value == "" {
				fl.Value = tok
			}
			continue
		}
		if strings.Contains(tok, "[no-]") {
			fl.Negatable = true
			tok = strings.Replace(tok, "[no-]", "", 1)
		}
		if i := strings.IndexAny(tok, "=["); i > 0 {
			val := tok[i:]
			tok = tok[:i]
			if strings.HasPrefix(val, "[") {
				fl.Optional = true
				val = strings.TrimSuffix(strings.TrimPrefix(val, "["), "]")
			}
			if strings.HasPrefix(val, "=") {
				fl.Equals = !fl.Optional
				val = val[1:]
			}
			if val != "" {
				fl.Value = val
			}
		}
		if !flagName.MatchString(tok) {
			continue
		}
		fl.Names = append(fl.Names, tok)
		if len(tok) > len(fl.Name) {
			fl.Name = tok
		}
	}
	if len(fl.Names) == 0 {
		return nil
	}
	return fl
}

// usageAndDoc returns the usage synopsis and the documentation of the
// command in the given lines of help output: the usage lines (or the
// SYNOPSIS of a man page) and the NAME of a man page, or else the first
// unindented paragraph after the usage.
func usageAndDoc(lines []string) (usage, doc string) {
	var us, ds, para []string
	section := ""
	inUsage, afterUsage := false, false
	for _, line := range lines {
		text := strings.TrimSpace(line)
		n := indentOf(line)
		lower := strings.ToLower(text)
		switch {
		case n == 0 && text != "" && text == strings.ToUpper(text) && !strings.ContainsAny(text, "-:"):
			section = text // a heading of a man page
			inUsage = false
			continue
		case strings.HasPrefix(lower, "usage:"):
			inUsage = true
			us = append(us, strings.TrimSpace(text[len("usage:"):]))
			continue
		}
		switch {
		case inUsage && text != "" && !strings.HasSuffix(lower, ":"):
			us = append(us, strings.TrimSpace(strings.TrimPrefix(text, "or:")))
			continue
		case inUsage:
			inUsage, afterUsage = false, true
		case section == "SYNOPSIS" && text != "":
			us = append(us, text)
		case section == "NAME" && text != "" && len(ds) == 0:
			if _, after, ok := strings.Cut(text, " - "); ok {
				ds = append(ds, after)
			}
		case afterUsage && n == 0 && text != "":
			para = append(para, text)
		case afterUsage && len(para) > 0:
			afterUsage = false
		}
	}
	// the first paragraph after the usage is the documentation if it is a
	// sentence, not a heading
	if p := strings.Join(para, " "); len(ds) == 0 && strings.Contains(p, ".") {
		if i := strings.Index(p, ". "); i > 0 { // the first sentence
			p = p[:i+1]
		}
		ds = append(ds, p)
	}
	us = slices.DeleteFunc(us, func(s string) bool { return s == "" })
	return strings.Join(us, "\n"), strings.Join(ds, " ")
}
