// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gidom

import (
	"fmt"

	"goki.dev/gi/v2/gi"
)

// ApplyStyle applies styling information to the given parent widget,
// using the given context. This should only be called in [ReadHTMLNode]
// after the widget has already been populated by the node tree.
func ApplyStyle(ctx Context, par gi.Widget) {
	fmt.Println(ctx.GetStyle())
}
