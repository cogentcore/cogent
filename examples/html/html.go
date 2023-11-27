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
	gi.SetAppName("gidom-html")
	b := gi.NewBody()
	h := grr.Log1(exampleHTML.ReadFile("example.html"))
	grr.Log(gidom.ReadHTML(gidom.BaseContext(), b, bytes.NewBuffer(h)))
	b.NewWindow().Run().Wait()
}
