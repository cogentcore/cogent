// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lang

import (
	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/parse"
	"cogentcore.org/core/text/parse/complete"
	"cogentcore.org/core/text/parse/parser"
	"cogentcore.org/core/text/textpos"
)

// ParseBackend is a [Backend] that uses the built-in [parse] system,
// which is based on an interactive parser that runs in-process.
// It returns zero values for any functionality that the parse system
// does not currently provide (diagnostics, hover, references, symbols,
// and formatting), which allows editors to use either backend through
// the same [Backend] interface.
//
// Note that the parse system must have language support registered
// through [parse.LanguageSupport] for a given language to be supported,
// and the [lines.Lines] buffer must have parsing enabled
// (see [highlighting.UseParse]).
//
// [parse]: https://pkg.go.dev/cogentcore.org/core/text/parse
// [highlighting.UseParse]: https://pkg.go.dev/cogentcore.org/core/text/highlighting#UseParse
type ParseBackend struct{}

// compile-time check that we implement the interface:
var _ Backend = (*ParseBackend)(nil)

// Name returns the name of this backend: "parse"
func (pb *ParseBackend) Name() string { return "parse" }

// langFor returns the [parse.LanguageProperties] for the given buffer,
// and the [parse.FileStates] state, nil,false if not supported.
func (pb *ParseBackend) langFor(ln *lines.Lines) (*parse.LanguageProperties, *parse.FileStates, bool) {
	if ln == nil {
		return nil, nil, false
	}
	lp, sfs := ln.ParseState()
	if lp == nil || lp.Lang == nil || sfs == nil {
		return nil, nil, false
	}
	return lp, sfs, true
}

// Supported returns whether the parse system supports the language
// of the given buffer.
func (pb *ParseBackend) Supported(ln *lines.Lines) bool {
	_, _, has := pb.langFor(ln)
	return has
}

// Stop does nothing for the parse backend.
func (pb *ParseBackend) Stop() {}

// Diagnostics returns nil: the parse backend does not provide diagnostics.
func (pb *ParseBackend) Diagnostics() <-chan FileDiags { return nil }

// Open does nothing for the parse backend,
// which syncs automatically with the [lines.Lines] markup machinery.
func (pb *ParseBackend) Open(ln *lines.Lines) {}

// Change does nothing for the parse backend,
// which syncs automatically with the [lines.Lines] markup machinery.
func (pb *ParseBackend) Change(ln *lines.Lines) {}

// Save does nothing for the parse backend.
func (pb *ParseBackend) Save(ln *lines.Lines) {}

// Close does nothing for the parse backend.
func (pb *ParseBackend) Close(ln *lines.Lines) {}

// Completions returns completions from the parse language for
// the given text and position.
func (pb *ParseBackend) Completions(ln *lines.Lines, text string, posLine, posChar int) complete.Matches {
	lp, sfs, has := pb.langFor(ln)
	if !has {
		return complete.Matches{}
	}
	parser.GUIActive = true
	return lp.Lang.CompleteLine(sfs, text, textpos.Pos{Line: posLine, Char: posChar})
}

// CompleteEdit returns the parse language edit for incorporating
// the given completion.
func (pb *ParseBackend) CompleteEdit(ln *lines.Lines, text string, cursorPos int, comp complete.Completion, seed string) complete.Edit {
	lp, sfs, has := pb.langFor(ln)
	if !has {
		return complete.Edit{NewText: comp.Text}
	}
	return lp.Lang.CompleteEdit(sfs, text, cursorPos, comp, seed)
}

// Hover returns "": the parse backend does not provide hover docs.
func (pb *ParseBackend) Hover(ln *lines.Lines, pos textpos.Pos) string { return "" }

// Definition returns the definition location of the symbol at the given
// position, using the parse language Lookup, which only returns a
// location if the symbol is defined in another file.
func (pb *ParseBackend) Definition(ln *lines.Lines, pos textpos.Pos) []Location {
	lp, sfs, has := pb.langFor(ln)
	if !has {
		return nil
	}
	parser.GUIActive = true
	ld := lp.Lang.Lookup(sfs, WordAt(ln, pos), pos)
	if ld.Filename == "" {
		return nil
	}
	reg := textpos.NewRegion(ld.StLine, 0, ld.EdLine, 0)
	return []Location{{Filename: ld.Filename, Region: reg}}
}

// References returns nil: the parse backend does not provide references.
func (pb *ParseBackend) References(ln *lines.Lines, pos textpos.Pos) []Location { return nil }

// Symbols returns nil: the parse backend provides symbols through the
// [parse.FileState] directly, which the editor can access in the usual way.
func (pb *ParseBackend) Symbols(ln *lines.Lines) []Symbol { return nil }

// WorkspaceSymbols returns nil: the parse backend does not provide
// workspace-wide symbol search through this interface.
func (pb *ParseBackend) WorkspaceSymbols(ln *lines.Lines, query string) []Symbol { return nil }

// Format returns false: the parse backend does not provide formatting,
// which is handled by external commands such as gofmt / goimports.
func (pb *ParseBackend) Format(ln *lines.Lines) bool { return false }
