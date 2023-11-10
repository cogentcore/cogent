// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"embed"

	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
	"goki.dev/glide/gidom"
	"goki.dev/grr"
)

//go:embed *.md
var exampleMD embed.FS

func main() { gimain.Run(app) }

func app() {
	sc := gi.NewScene("gidom-md")
	b := grr.Log(exampleMD.ReadFile("example.md"))
	grr.Log0(gidom.ReadMD(sc, b, ""))
	gi.NewWindow(sc).Run().Wait()
}
