// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package lsp implements a Language Server Protocol (LSP) client for
// the Cogent Code editor, driving external language server programs
// such as gopls (the Go language server) as subprocesses that speak
// JSON-RPC over stdio.
//
// The protocol request flow, initialize handshake, client capabilities,
// and server downcall handling are modeled on the sample client provided
// with gopls: golang.org/x/tools/gopls/internal/cmd (BSD license). The
// gopls internal packages cannot be imported directly from external
// modules (even when vendored), so this package implements the needed
// subset of the protocol directly:
// https://microsoft.github.io/language-server-protocol/specification
//
// The [Backend] type implements the [lang.Backend] interface from
// cogentcore.org/cogent/code/lang, providing diagnostics, completions,
// hover docs, definitions, references, symbols, and formatting for any
// language with an available server in the [Registry]
// (gopls for Go, and others by default).
package lsp
