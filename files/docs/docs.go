// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package docs is the documentation of Cogent Files, whose home page is
// home.md, which the Help of the app menu opens, and which is also on the
// website of the Cogent apps (cogentcore.org/cogent/site). It is a package
// of its own, as Cogent Files is a command (package main).
package docs

import "embed"

// FS is the documentation of Cogent Files.
//
//go:embed home.md icon.svg
var FS embed.FS
