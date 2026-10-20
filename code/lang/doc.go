// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package lang defines a backend-neutral interface for the language
// services used by the Cogent Code editor: diagnostics, completions,
// hover docs, definitions, references, symbols, and formatting.
//
// The interface can be implemented either by an external language
// server accessed over the [LSP] protocol (see the
// [cogentcore.org/cogent/code/lsp] sub-package, which drives gopls and
// other language servers), or by the built-in [parse] system (see the
// ParseBackend in this package). The parse implementation returns zero
// values for everything it does not currently implement, so editors can
// use either backend through the same API.
//
// [LSP]: https://microsoft.github.io/language-server-protocol/specification
// [parse]: https://pkg.go.dev/cogentcore.org/core/text/parse
package lang
