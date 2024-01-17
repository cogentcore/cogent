// Copyright (c) 2023, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"embed"

	"goki.dev/gi/v2/gi"
	"goki.dev/glide/gidom"
	"goki.dev/grr"
)

//go:embed *.md
var exampleMD embed.FS

func main() {
	b := gi.NewBody("gidom-md")
	h := grr.Log1(exampleMD.ReadFile("example.md"))
	grr.Log(gidom.ReadMD(gidom.BaseContext(), b, h))
	b.NewWindow().Run().Wait()
}
