// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lang

import (
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

// TextEdit is a replacement of the text in a region with NewText.
// An empty NewText means a pure deletion of the region.
type TextEdit struct {

	// region of text to replace, using 0-based lines and rune positions
	Region textpos.Region

	// new text to insert (empty = delete the region)
	NewText string
}

// FileEdits are text edits to apply to a single file,
// as part of a workspace-wide edit such as a symbol rename.
type FileEdits struct {

	// full path of the file
	Filename string

	// edits to apply, in any order
	Edits []TextEdit
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
