// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"embed"

	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
	"goki.dev/glide/gidom"
	"goki.dev/grr"
)

//go:embed example.html
var exampleHTML embed.FS

func main() { gimain.Run(app) }

func app() {
	sc := gi.NewScene("gidom")
	b := grr.Log(exampleHTML.ReadFile("example.html"))
	grr.Log0(gidom.ReadHTML(gidom.BaseContext(), sc, bytes.NewBuffer(b)))
	gi.NewWindow(sc).Run().Wait()
}
