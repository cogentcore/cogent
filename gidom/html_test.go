// Copyright (c) 2023, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gidom

import (
	"testing"

	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
)

func TestRenderHTML(t *testing.T) {
	gimain.Run(func() {
		sc := gi.NewScene("test-render-html")
		s := `
		<h1>Gidom</h1>
		<p>This is a demonstration of the various features of gidom</p>
		<button>Hello, world!</button>
		`
		err := ReadHTMLString(BaseContext(), sc, s)
		if err != nil {
			t.Error(err)
		}
		sc.AssertPixelsOnShow(t, "test-render-html.png")
	})
}
