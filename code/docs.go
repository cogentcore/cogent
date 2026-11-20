// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"embed"

	"cogentcore.org/core/base/fsx"
)

//go:embed docs
var docsFS embed.FS

// Docs is the documentation of Cogent Code, whose home page is home.md,
// which the Help of the app menu opens (set by the cogentcode command
// with content.SetAppHelp).
var Docs = fsx.Sub(docsFS, "docs")
