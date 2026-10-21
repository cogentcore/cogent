// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// cogent-lsp is a command-line helper around the Cogent LSP client
// (cogentcore.org/cogent/code/lsp), exposing language server features
// (gopls and other registered servers) to shells, scripts, CI, and AI
// agents, without needing to open the editor.
//
// Positions are 1-based (human/editor convention). Workspace roots are
// found via standard project marker files (go.mod, package.json, etc)
// relative to each file. Position queries retry briefly after the
// server is ready, to absorb null responses from ongoing package
// loading. The rename and actions -a commands write files; use rename
// -n to preview. Exit status is 0 on success (clean check, or results
// found) and 1 for problems found, no results, or errors.
package main

import (
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"cogentcore.org/core/base/logx"
	"cogentcore.org/core/cli"
	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/textpos"

	"cogentcore.org/cogent/code/lang"
	"cogentcore.org/cogent/code/lsp"
)

//go:generate core generate -add-funcs=true

type Config struct {

	// files or FILE:LINE:COL targets, given after the command name
	Args []string `posarg:"all"`

	// rewrite files in place (format command)
	Write bool `flag:"write,w"`

	// preview changes without writing any files (rename command)
	DryRun bool `flag:"dry-run,n"`

	// apply the code action whose title matches (exact, then
	// case-insensitive substring) (actions command)
	Apply string `flag:"apply,a"`

	// overall wait for language server startup and responses,
	// as a Go duration string like 30s or 2m
	Timeout string `default:"30s"`

	// maximum number of symbols to report for workspace search
	Max int `default:"100"`
}

// queryRetry is the extra window position queries retry for, absorbing
// null responses from a server that is still loading packages, after
// which an empty result is taken as genuinely not found.
const queryRetry = 5 * time.Second

// retryQuery calls fn until it returns true, or the retry window elapses.
func retryQuery(fn func() bool) bool {
	deadline := time.Now().Add(queryRetry)
	for {
		if fn() {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// Check reports diagnostics for the given files, in
// path:line:col-line:col: severity: message format.
func Check(c *Config) error {
	setup()
	if len(c.Args) == 0 {
		return fmt.Errorf("check needs FILE arguments")
	}
	deadline, err := c.deadline()
	if err != nil {
		return err
	}
	b := newBackend()
	defer b.Stop()

	orig := make(map[string]string)
	pend := make(map[string]bool)
	for _, f := range c.Args {
		ln, abs, err := openFile(f)
		if err != nil {
			return err
		}
		if !b.Supported(ln) {
			return fmt.Errorf("no language server available for %s", f)
		}
		b.Open(ln)
		orig[abs] = f
		pend[abs] = true
	}
	nerr := 0
	for len(pend) > 0 {
		rem := time.Until(deadline)
		if rem <= 0 {
			break
		}
		select {
		case fd, ok := <-b.Diagnostics():
			if !ok {
				return fmt.Errorf("diagnostics channel closed unexpectedly")
			}
			if !pend[fd.Filename] {
				continue
			}
			delete(pend, fd.Filename)
			for _, d := range fd.Diags {
				name := orig[fd.Filename]
				if name == "" {
					name = fd.Filename
				}
				fmt.Printf("%s: %s: %s\n", posString(name, d.Region), sevWord(d.Severity), d.Msg)
				if d.Severity == lang.DiagErr {
					nerr++
				}
			}
		case <-time.After(rem):
			for f := range pend {
				fmt.Fprintf(os.Stderr, "timed out waiting for diagnostics of %s\n", orig[f])
			}
			return fmt.Errorf("timed out waiting for %d file(s)", len(pend))
		}
	}
	if nerr > 0 {
		return fmt.Errorf("%d error(s) found", nerr)
	}
	return nil
}

// Sig shows the signature help for the callable at FILE:LINE:COL,
// marking the active parameter with brackets.
func Sig(c *Config) error {
	return withPos(c, func(b *lsp.Backend, ln *lines.Lines, pos textpos.Pos) error {
		var sh *lsp.SignatureHelp
		act := -1
		if !retryQuery(func() bool {
			var err error
			sh, act, err = b.SignatureHelp(ln, pos, false)
			return sh != nil && err == nil
		}) {
			return fmt.Errorf("no callable at %s", c.Args[0])
		}
		for i := range sh.Signatures {
			mark, unmark := "", ""
			if i == act {
				mark, unmark = "[", "]"
			}
			fmt.Println(lsp.SignatureText(sh, i, sh.ActiveParam(), mark, unmark))
		}
		return nil
	})
}

// Hints dumps the inlay hints the server provides for FILE, one per
// line as path:line:col: label. Note: these are for programmatic use;
// the editor does not render inlay hints inline.
func Hints(c *Config) error {
	return withFile(c, "hints", func(b *lsp.Backend, ln *lines.Lines) error {
		hints, err := b.InlayHints(ln, textpos.Region{End: textpos.Pos{Line: ln.NumLines()}})
		if err != nil {
			return err
		}
		for _, h := range hints {
			if txt := h.LabelText(); txt != "" {
				fmt.Printf("%s:%d:%d: %s\n", ln.Filename(), h.Position.Line+1, h.Position.Character+1, txt)
			}
		}
		if len(hints) == 0 {
			fmt.Fprintln(os.Stderr, "no inlay hints (many servers gate these behind settings)")
		}
		return nil
	})
}

// Tokens dumps the semantic tokens the server provides for FILE,
// one per line as path:line:col-endcol: type (+modifiers).
func Tokens(c *Config) error {
	return withFile(c, "tokens", func(b *lsp.Backend, ln *lines.Lines) error {
		var toks []lsp.SemanticToken
		if !retryQuery(func() bool {
			var err error
			toks, err = b.SemanticTokensFull(ln)
			return len(toks) > 0 && err == nil
		}) {
			return fmt.Errorf("no semantic tokens for %s", c.Args[0])
		}
		for _, tk := range toks {
			mods := ""
			if len(tk.Modifiers) > 0 {
				mods = " (+" + strings.Join(tk.Modifiers, ",") + ")"
			}
			fmt.Printf("%s:%d:%d-%d: %s%s\n", ln.Filename(), tk.Start.Line+1, tk.Start.Char+1, tk.End.Char, tk.Type, mods)
		}
		return nil
	})
}

// Hover shows the hover documentation for the symbol at FILE:LINE:COL.
func Hover(c *Config) error {
	return withPos(c, func(b *lsp.Backend, ln *lines.Lines, pos textpos.Pos) error {
		var docs string
		if !retryQuery(func() bool { docs = b.Hover(ln, pos); return docs != "" }) {
			return fmt.Errorf("no documentation at %s", c.Args[0])
		}
		fmt.Println(docs)
		return nil
	})
}

// Definition shows the definition location(s) of the symbol at FILE:LINE:COL.
func Definition(c *Config) error {
	return withPos(c, func(b *lsp.Backend, ln *lines.Lines, pos textpos.Pos) error {
		var locs []lang.Location
		if !retryQuery(func() bool { locs = b.Definition(ln, pos); return len(locs) > 0 }) {
			return fmt.Errorf("no definition found at %s", c.Args[0])
		}
		for _, l := range locs {
			fmt.Println(locString(l))
		}
		return nil
	})
}

// Refs shows the reference location(s) of the symbol at FILE:LINE:COL.
func Refs(c *Config) error {
	return withPos(c, func(b *lsp.Backend, ln *lines.Lines, pos textpos.Pos) error {
		var locs []lang.Location
		if !retryQuery(func() bool { locs = b.References(ln, pos); return len(locs) > 0 }) {
			return fmt.Errorf("no references found at %s", c.Args[0])
		}
		for _, l := range locs {
			fmt.Println(locString(l))
		}
		return nil
	})
}

// Symbols lists the hierarchical document symbols for FILE.
func Symbols(c *Config) error {
	return withFile(c, "symbols", func(b *lsp.Backend, ln *lines.Lines) error {
		sl := b.Symbols(ln)
		if len(sl) == 0 {
			return fmt.Errorf("no symbols in %s", c.Args[0])
		}
		printSymbols(sl, 0)
		return nil
	})
}

// Workspace searches workspace-wide for symbols matching QUERY,
// using FILE to determine the language and project.
func Workspace(c *Config) error {
	setup()
	if len(c.Args) != 2 {
		return fmt.Errorf("workspace needs QUERY and FILE arguments")
	}
	deadline, err := c.deadline()
	if err != nil {
		return err
	}
	b := newBackend()
	defer b.Stop()
	ln, ok := openReady(b, c.Args[1], deadline)
	if !ok {
		return fmt.Errorf("language server not available for %s", c.Args[1])
	}
	sl := b.WorkspaceSymbols(ln, c.Args[0])
	if len(sl) == 0 {
		return fmt.Errorf("no workspace symbols matching %q", c.Args[0])
	}
	if c.Max > 0 && len(sl) > c.Max {
		fmt.Fprintf(os.Stderr, "showing first %d of %d matches (use -max for more)\n", c.Max, len(sl))
		sl = sl[:c.Max]
	}
	printSymbols(sl, 0)
	return nil
}

// Format formats the given files, writing to stdout
// or rewriting them in place with the -w flag.
func Format(c *Config) error {
	setup()
	if len(c.Args) == 0 {
		return fmt.Errorf("format needs FILE arguments")
	}
	deadline, err := c.deadline()
	if err != nil {
		return err
	}
	b := newBackend()
	defer b.Stop()
	for _, f := range c.Args {
		ln, ok := openReady(b, f, deadline)
		if !ok {
			return fmt.Errorf("language server not available for %s", f)
		}
		if !b.Format(ln) {
			fmt.Fprintf(os.Stderr, "no formatting edits for %s (unsupported, or already formatted)\n", f)
			continue
		}
		if c.Write {
			if err := os.WriteFile(f, []byte(ln.String()), 0644); err != nil {
				return err
			}
		} else {
			fmt.Print(ln.String())
		}
	}
	return nil
}

// Rename renames the symbol at FILE:LINE:COL to NEWNAME across the
// workspace, rewriting the affected files. Use -n to preview the
// per-file edit counts without writing anything.
func Rename(c *Config) error {
	setup()
	if len(c.Args) != 2 {
		return fmt.Errorf("rename needs FILE:LINE:COL and NEWNAME arguments")
	}
	deadline, err := c.deadline()
	if err != nil {
		return err
	}
	path, pos, err := parseLoc(c.Args[0])
	if err != nil {
		return err
	}
	newName := c.Args[1]
	b := newBackend()
	defer b.Stop()
	ln, ok := openReady(b, path, deadline)
	if !ok {
		return fmt.Errorf("language server not available for %s", path)
	}
	if c.DryRun {
		we, err := b.RenameEdits(ln, pos, newName)
		if err != nil {
			return err
		}
		if we == nil {
			return fmt.Errorf("nothing to rename at %s", c.Args[0])
		}
		for _, f := range editCounts(we) {
			fmt.Printf("%s: %d edit(s)\n", f.Filename, len(f.Edits))
		}
		return nil
	}
	out, err := b.Rename(ln, pos, newName)
	if err != nil {
		return err
	}
	return writeEdited(out, path, ln)
}

// Actions lists the quick fixes and refactorings the server offers at
// FILE:LINE:COL: organize imports, missing returns, error-check
// wrapping, typo fixes, extract function/variable, fill struct, etc.
// Diagnostics published for the file are sent as context, which is
// what turns the quick fixes on. With -a TITLE, applies the matching
// action and rewrites the affected files.
func Actions(c *Config) error {
	setup()
	if len(c.Args) != 1 {
		return fmt.Errorf("actions needs a FILE:LINE:COL argument")
	}
	deadline, err := c.deadline()
	if err != nil {
		return err
	}
	path, pos, err := parseLoc(c.Args[0])
	if err != nil {
		return err
	}
	b := newBackend()
	defer b.Stop()
	ln, abs, err := openFile(path)
	if err != nil {
		return err
	}
	if !b.Supported(ln) {
		return fmt.Errorf("no language server available for %s", path)
	}
	b.Open(ln)
	// collect the diagnostics published for this file as context:
	var diags []lang.Diagnostic
	done := false
	for !done && time.Now().Before(deadline) {
		rem := time.Until(deadline)
		select {
		case fd, ok := <-b.Diagnostics():
			if !ok {
				return fmt.Errorf("diagnostics channel closed unexpectedly")
			}
			if cliNormalize(fd.Filename) != cliNormalize(abs) {
				continue
			}
			diags = fd.Diags
			done = true
		case <-time.After(rem):
			done = true
		}
	}
	if !done {
		fmt.Fprintf(os.Stderr, "no diagnostics published for %s (continuing without context)\n", path)
	}
	// region: the word at the position, expanded to cover any
	// diagnostics overlapping it, which is what quick fixes attach to:
	reg := textpos.Region{Start: pos, End: pos}
	if ln.IsValidLine(pos.Line) {
		if w := ln.WordAt(pos); w.Start.Line == pos.Line && w.End.Char > w.Start.Char {
			reg = w
		}
	}
	var cdiags []lang.Diagnostic
	for _, d := range diags {
		if d.Region.End.Line >= reg.Start.Line && d.Region.Start.Line <= reg.End.Line {
			cdiags = append(cdiags, d)
			if d.Region.Start.Line < reg.Start.Line {
				reg.Start = d.Region.Start
			}
			if d.Region.End.Line > reg.End.Line ||
				(d.Region.End.Line == reg.End.Line && d.Region.End.Char > reg.End.Char) {
				reg.End = d.Region.End
			}
		}
	}
	if len(cdiags) == 0 {
		cdiags = diags
	}
	acts, err := b.CodeActions(ln, reg, cdiags)
	if err != nil {
		return err
	}
	if len(acts) == 0 {
		return fmt.Errorf("no code actions at %s", c.Args[0])
	}
	if c.Apply == "" {
		for _, ca := range acts {
			fmt.Printf("%s [%s]", ca.Title, ca.Kind)
			if ca.IsPreferred {
				fmt.Print(" (preferred)")
			}
			if ca.Disabled != nil {
				fmt.Printf(" -- disabled: %s", ca.Disabled.Reason)
			}
			fmt.Println()
		}
		return nil
	}
	sel := pickAction(acts, c.Apply)
	if sel == nil {
		return fmt.Errorf("no action matching %q at %s (list without -a to see options)", c.Apply, c.Args[0])
	}
	if sel.Disabled != nil {
		return fmt.Errorf("action %q is disabled: %s", sel.Title, sel.Disabled.Reason)
	}
	out, err := b.RunCodeAction(ln, sel)
	pendingShows.Wait()
	if err != nil {
		return err
	}
	return writeEdited(out, path, ln)
}

// writeEdited reports the edits applied per file, and writes out the
// given target file, which is open in this session, so the backend
// edited it in memory rather than writing it.
func writeEdited(out []lang.FileEdits, path string, ln *lines.Lines) error {
	target := cliNormalize(path)
	for _, f := range out {
		fmt.Printf("%s: %d edit(s)\n", f.Filename, len(f.Edits))
		if cliNormalize(f.Filename) == target {
			if err := os.WriteFile(path, []byte(ln.String()), 0644); err != nil {
				return err
			}
		}
	}
	return nil
}

// pickAction finds the action whose title matches the given pattern:
// exact match first, then case-insensitive substring.
func pickAction(acts []lsp.CodeAction, pat string) *lsp.CodeAction {
	for i := range acts {
		if acts[i].Title == pat {
			return &acts[i]
		}
	}
	lpat := strings.ToLower(pat)
	for i := range acts {
		if strings.Contains(strings.ToLower(acts[i].Title), lpat) {
			return &acts[i]
		}
	}
	return nil
}

// editCounts converts a workspace edit into per-file counts, for a
// dry-run report, without applying anything.
func editCounts(we *lsp.WorkspaceEdit) []lang.FileEdits {
	counts := map[string]int{}
	for u, es := range we.Changes {
		counts[lsp.DocumentURI(u).Path()] += len(es)
	}
	for _, de := range we.DocumentChanges {
		counts[de.TextDocument.URI.Path()] += len(de.Edits)
	}
	var out []lang.FileEdits
	for fname, n := range counts {
		out = append(out, lang.FileEdits{Filename: fname, Edits: make([]lang.TextEdit, n)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Filename < out[j].Filename })
	return out
}

// cliNormalize canonicalizes a path for comparing whether two names
// refer to the same file, resolving symlinks like /var -> /private/var.
func cliNormalize(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	if ev, err := filepath.EvalSymlinks(abs); err == nil {
		return ev
	}
	return abs
}

// withPos handles the common setup for position-based commands:
// parse FILE:LINE:COL, open the file, and wait for the server.
func withPos(c *Config, fn func(b *lsp.Backend, ln *lines.Lines, pos textpos.Pos) error) error {
	setup()
	if len(c.Args) != 1 {
		return fmt.Errorf("needs a FILE:LINE:COL argument")
	}
	deadline, err := c.deadline()
	if err != nil {
		return err
	}
	path, pos, err := parseLoc(c.Args[0])
	if err != nil {
		return err
	}
	b := newBackend()
	defer b.Stop()
	ln, ok := openReady(b, path, deadline)
	if !ok {
		return fmt.Errorf("language server not available for %s", path)
	}
	return fn(b, ln, pos)
}

// pendingShows waits for documents being shown, before the server stops.
var pendingShows sync.WaitGroup

// newBackend returns a new backend that prints the documents that
// servers ask to show, such as the pages for gopls Browse actions,
// as text for pages served by the server itself, and as a URL or
// location otherwise.
func newBackend() *lsp.Backend {
	b := lsp.NewBackend(nil)
	b.ShowDocument = func(uri string, loc *lang.Location) bool {
		switch {
		case loc != nil:
			fmt.Println("show:", locString(*loc))
		case lsp.IsLocalURL(uri):
			// must not block the server connection, which the server
			// needs to serve the page:
			pendingShows.Add(1)
			go func() {
				defer pendingShows.Done()
				title, text, err := lsp.DocumentText(uri)
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					return
				}
				fmt.Printf("%s\n\n%s", title, text)
			}()
		default:
			fmt.Println("open:", uri)
		}
		return true
	}
	return b
}

// withFile handles the common setup for single-file commands:
// open the FILE argument and wait for the server.
func withFile(c *Config, cmd string, fn func(b *lsp.Backend, ln *lines.Lines) error) error {
	setup()
	if len(c.Args) != 1 {
		return fmt.Errorf("%s needs a FILE argument", cmd)
	}
	deadline, err := c.deadline()
	if err != nil {
		return err
	}
	b := newBackend()
	defer b.Stop()
	ln, ok := openReady(b, c.Args[0], deadline)
	if !ok {
		return fmt.Errorf("language server not available for %s", c.Args[0])
	}
	return fn(b, ln)
}

// setup silences the client's internal logging unless -v was given.
func setup() {
	if logx.UserLevel > slog.LevelInfo {
		log.SetOutput(io.Discard)
	} else {
		lsp.Verbose = true
	}
}

func (c *Config) deadline() (time.Time, error) {
	d, err := time.ParseDuration(c.Timeout)
	if err != nil {
		return time.Time{}, fmt.Errorf("bad -timeout %q: %w", c.Timeout, err)
	}
	return time.Now().Add(d), nil
}

// openFile loads a file into a new Lines buffer with its filename set,
// so that the backend can detect its language.
func openFile(path string) (*lines.Lines, string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, "", err
	}
	txt, err := os.ReadFile(abs)
	if err != nil {
		return nil, "", err
	}
	ln := lines.NewLines()
	ln.SetFilename(abs)
	ln.SetText(txt)
	return ln, abs, nil
}

// openReady opens the file with the backend and waits until the
// language server has completed startup, up to the deadline.
func openReady(b *lsp.Backend, path string, deadline time.Time) (*lines.Lines, bool) {
	ln, _, err := openFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return nil, false
	}
	if !b.Supported(ln) {
		fmt.Fprintf(os.Stderr, "no language server available for %s\n", path)
		return nil, false
	}
	b.Open(ln)
	for time.Now().Before(deadline) {
		if b.Ready(ln) {
			return ln, true
		}
		time.Sleep(10 * time.Millisecond)
	}
	fmt.Fprintf(os.Stderr, "timed out waiting for language server for %s\n", path)
	return nil, false
}

// parseLoc parses FILE:LINE:COL with 1-based line and column.
func parseLoc(s string) (path string, pos textpos.Pos, err error) {
	i := strings.LastIndex(s, ":")
	j := -1
	if i > 0 {
		j = strings.LastIndex(s[:i], ":")
	}
	if j < 0 || i < 0 {
		return "", pos, fmt.Errorf("argument %q must be FILE:LINE:COL", s)
	}
	path = s[:j]
	li, err := strconv.Atoi(s[j+1 : i])
	if err != nil || li < 1 {
		return path, pos, fmt.Errorf("bad line in %q", s)
	}
	ch, err := strconv.Atoi(s[i+1:])
	if err != nil || ch < 1 {
		return path, pos, fmt.Errorf("bad column in %q", s)
	}
	pos = textpos.Pos{Line: li - 1, Char: ch - 1}
	return path, pos, nil
}

func sevWord(s lang.DiagSeverity) string {
	switch s {
	case lang.DiagErr:
		return "error"
	case lang.DiagWarn:
		return "warning"
	case lang.DiagInfo:
		return "info"
	}
	return "hint"
}

// posString formats a region as path:line:col-line:col, 1-based,
// in the style of gopls check and compiler errors.
func posString(path string, r textpos.Region) string {
	return fmt.Sprintf("%s:%d:%d-%d:%d", path, r.Start.Line+1, r.Start.Char+1, r.End.Line+1, r.End.Char+1)
}

func locString(l lang.Location) string {
	if l.Region.Start == l.Region.End {
		return fmt.Sprintf("%s:%d:%d", l.Filename, l.Region.Start.Line+1, l.Region.Start.Char+1)
	}
	return posString(l.Filename, l.Region)
}

func printSymbols(sl []lang.Symbol, indent int) {
	for _, sy := range sl {
		fmt.Printf("%s%s %s", strings.Repeat("  ", indent), sy.Kind, sy.Name)
		if sy.Detail != "" {
			fmt.Printf("  [%s]", sy.Detail)
		}
		fmt.Printf("  %s\n", locString(lang.Location{Filename: sy.Filename, Region: sy.Region}))
		printSymbols(sy.Children, indent+1)
	}
}

func main() { //types:skip
	opts := cli.DefaultOptions("cogent-lsp", "Command-line access to language server features (gopls and other servers) via the Cogent LSP client, for shells, scripts, CI, and AI agents.")
	opts.PrintSuccess = false
	cli.Run(opts, &Config{}, Check, Hover, Definition, Refs, Symbols, Workspace, Format, Rename, Actions, Sig, Hints, Tokens)
}
