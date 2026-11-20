// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"testing"

	"cogentcore.org/cogent/files/docs"
	"cogentcore.org/core/content"
	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
)

// TestDocs tests that the documentation of the app renders, with its
// home page first.
func TestDocs(t *testing.T) {
	b := core.NewBody()
	b.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(1000), units.Dp(800)) })
	content.NewContent(b).SetSource(docs.FS)
	core.AssertRender(t, b, "docs")
}
