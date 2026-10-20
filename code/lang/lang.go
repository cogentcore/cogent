// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lang

import (
	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/parse/complete"
	"cogentcore.org/core/text/textpos"
	"cogentcore.org/core/text/token"
)

// DiagSeverity indicates how severe a [Diagnostic] is.
type DiagSeverity int32 //enums:enum -trim-prefix Diag

const (
	// DiagErr is an error that prevents compilation
	DiagErr DiagSeverity = iota

	// DiagWarn is a warning that should be addressed
	DiagWarn

	// DiagInfo is an informational message
	DiagInfo

	// DiagHint is a hint for possible improvement
	DiagHint
)

// Diagnostic represents a diagnostic message (error, warning, etc)
// for a specific region of a file, as reported by the language backend.
type Diagnostic struct {

	// region of the diagnostic in the file, using 0-based lines and
	// rune positions within lines
	Region textpos.Region

	// human-readable diagnostic message
	Msg string

	// severity of the diagnostic
	Severity DiagSeverity

	// source of the diagnostic, e.g., a linter name (optional)
	Source string

	// machine-readable code identifying the diagnostic (optional)
	Code string
}

// FileDiags is a complete set of diagnostics for a given file.
// An empty slice means all prior diagnostics for the file are resolved.
type FileDiags struct {

	// full path of the file
	Filename string

	// current diagnostics for the file
	Diags []Diagnostic
}

// Location represents a location of a symbol or reference in a file.
type Location struct {

	// full path of the file
	Filename string

	// region of the location, using 0-based lines and rune positions
	Region textpos.Region
}

// Symbol represents a named language symbol: function, variable, type, etc.
// It corresponds to the LSP DocumentSymbol / SymbolInformation structures,
// and the [syms.Symbol] type used by the parse system.
type Symbol struct {

	// name of the symbol
	Name string

	// additional detail such as the signature of a function (optional)
	Detail string

	// kind of symbol, using the standard token.Tokens list,
	// which provides icons etc
	Kind token.Tokens

	// full path of the file in which the symbol is defined
	Filename string

	// region of the definition, using 0-based lines and rune positions
	Region textpos.Region

	// children of the symbol, e.g. methods and fields of a type,
	// used for hierarchical document symbols
	Children []Symbol
}

// Backend is the interface for a language service backend, providing
// editor language features for files. There can be multiple Backends
// available, and a given file may only be supported by some of them,
// according to the language of the file. See the [ParseBackend] in this
// package which wraps the cogent [parse] system, and the [lsp.Backend]
// in cogentcore.org/cogent/code/lsp which drives external language
// servers such as gopls over the [LSP] protocol.
//
// All positions are in the editor-native units: 0-based line numbers,
// and rune offsets within lines ([textpos.Pos]). Backends are responsible
// for any conversions needed (e.g., to UTF-16 offsets for LSP).
//
// [parse]: https://pkg.go.dev/cogentcore.org/core/text/parse
// [lsp.Backend]: https://pkg.go.dev/cogentcore.org/cogent/code/lsp
// [LSP]: https://microsoft.github.io/language-server-protocol/specification
type Backend interface {

	// Name returns the name of this backend, e.g., "parse" or "gopls"
	Name() string

	// Supported returns whether this backend supports the language
	// of the given buffer.
	Supported(ln *lines.Lines) bool

	// Stop shuts down the backend, releasing any resources
	// such as running language server processes.
	Stop()

	// Diagnostics returns a channel through which diagnostics for
	// open files are asynchronously delivered. It can return nil if
	// the backend does not provide diagnostics. The channel remains
	// open until [Stop] is called.
	Diagnostics() <-chan FileDiags

	// Open notifies the backend that the given file has been opened
	// in the editor, and its contents should be tracked.
	Open(ln *lines.Lines)

	// Change notifies the backend that the contents of an open
	// file have changed; it syncs the current contents of the buffer.
	Change(ln *lines.Lines)

	// Save notifies the backend that an open file has been saved
	// to its disk location.
	Save(ln *lines.Lines)

	// Close notifies the backend that the file is no longer open
	// in the editor.
	Close(ln *lines.Lines)

	// Completions returns completion matches for the given text at the
	// given position. text is the text of the line up to the cursor,
	// and posLine, posChar are the 0-based line and rune position of the
	// cursor. A short timeout should be used so that the editor is not
	// blocked waiting for a slow backend.
	Completions(ln *lines.Lines, text string, posLine, posChar int) complete.Matches

	// CompleteEdit returns the edit needed to incorporate the given
	// completion selection into the text, with the cursor at cursorPos
	// runes into text, and the given seed prefix already typed.
	CompleteEdit(ln *lines.Lines, text string, cursorPos int, comp complete.Completion, seed string) complete.Edit

	// Hover returns documentation for the symbol at the given position,
	// as markdown text. It uses a short timeout, and returns "" if
	// nothing is available in time.
	Hover(ln *lines.Lines, pos textpos.Pos) string

	// Definition returns the locations where the symbol at the given
	// position is defined.
	Definition(ln *lines.Lines, pos textpos.Pos) []Location

	// References returns all locations that reference the symbol at
	// the given position.
	References(ln *lines.Lines, pos textpos.Pos) []Location

	// Symbols returns the hierarchical symbols defined within the
	// given file, nil if not supported.
	Symbols(ln *lines.Lines) []Symbol

	// WorkspaceSymbols returns symbols matching the given query across
	// the entire workspace (project), nil if not supported.
	WorkspaceSymbols(ln *lines.Lines, query string) []Symbol

	// Format formats the file according to the language's standard
	// style, applying the edits directly to the buffer.
	// It returns true if any edits were applied.
	Format(ln *lines.Lines) bool
}
