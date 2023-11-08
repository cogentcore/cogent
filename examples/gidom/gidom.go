// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
	"goki.dev/glide/gidom"
	"goki.dev/grr"
)

func main() { gimain.Run(app) }

func app() {
	sc := gi.NewScene("gidom")

	s := `
<h1>Gidom</h1>
<p>This is a demonstration of the various features of gidom</p>
<button>Hello, world!</button>
`
	grr.Log0(gidom.ReadHTMLString(sc, s))

	gi.NewWindow(sc).Run().Wait()
}
