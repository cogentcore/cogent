// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lists

import (
	"embed"

	"cogentcore.org/core/base/fsx"
)

//go:embed docs
var docsFS embed.FS

// Docs is the documentation of Cogent Lists, whose home page is home.md,
// which the Help of the app menu opens (see [NewApp]).
var Docs = fsx.Sub(docsFS, "docs")
