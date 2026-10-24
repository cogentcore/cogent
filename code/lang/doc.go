// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package lang defines backend-neutral data types for the language
// services used by the Cogent Code editor: diagnostics, locations,
// and symbols, with positions in editor-native units (0-based lines,
// rune offsets within lines).
//
// These types are produced by the [cogentcore.org/cogent/code/lsp]
// sub-package, which drives gopls and other external language servers
// over the [LSP] protocol, and consumed by the editor without any
// dependency on the protocol-specific representations.
//
// [LSP]: https://microsoft.github.io/language-server-protocol/specification
package lang
