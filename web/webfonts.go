// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"cogentcore.org/core/htmlcore"
	"github.com/tdewolff/font"
)

func init() {
	// The web fonts of pages are mostly WOFF2 (with brotli compression),
	// which the text shapers do not read: they are decoded to TrueType or
	// OpenType (as are WOFF and EOT).
	htmlcore.FontDecoder = font.ToSFNT
}
