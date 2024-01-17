// Copyright (c) 2023, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gidom

import (
	"bytes"
	"fmt"

	"cogentcore.org/core/gi/v2/gi"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

// ReadMD reads MD (markdown) from the given bytes and adds corresponding
// GoGi widgets to the given [gi.Widget], using the given context.
func ReadMD(ctx Context, par gi.Widget, b []byte) error {
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
		),
	)
	var buf bytes.Buffer
	err := md.Convert(b, &buf)
	if err != nil {
		return fmt.Errorf("error parsing MD (markdown): %w", err)
	}
	return ReadHTML(ctx, par, &buf)
}

// ReadMDString reads MD (markdown) from the given string and adds
// corresponding GoGi widgets to the given [gi.Widget], using the given context.
func ReadMDString(ctx Context, par gi.Widget, s string) error {
	return ReadMD(ctx, par, []byte(s))
}
