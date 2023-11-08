// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gidom

import (
	"bytes"
	"fmt"

	"github.com/yuin/goldmark"
	"goki.dev/gi/v2/gi"
)

// ReadMD reads MD (markdown) from the given bytes and adds
// corresponding GoGi widgets to the given [gi.Widget].
func ReadMD(par gi.Widget, b []byte) error {
	var buf bytes.Buffer
	err := goldmark.Convert(b, &buf)
	if err != nil {
		return fmt.Errorf("error parsing MD (markdown): %w", err)
	}
	return ReadHTML(par, &buf)
}

// ReadMDString reads MD (markdown) from the given string and adds
// corresponding GoGi widgets to the given [gi.Widget].
func ReadMDString(par gi.Widget, s string) error {
	return ReadMD(par, []byte(s))
}
