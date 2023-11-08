// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build offscreen

package gidom

import (
	"testing"

	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
	"goki.dev/goosi"
)

func TestRenderHTML(t *testing.T) {
	gimain.Run(func() {
		sc := gi.NewScene("test-render-html")

		s := `
		<h1>Gidom</h1>
		<p>This is a demonstration of the various features of gidom</p>
		<button>Hello, world!</button>
		`
		err := ReadHTMLString(sc, s)
		if err != nil {
			t.Error(err)
		}

		gi.NewWindow(sc).Run()

		goosi.CaptureAs("test-render-html.png")
	})
}
