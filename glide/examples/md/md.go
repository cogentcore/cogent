// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"embed"

	"cogentcore.org/cogent/glide/gidom"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/grr"
)

//go:embed *.md
var exampleMD embed.FS

func main() {
	b := gi.NewBody("gidom-md")
	h := grr.Log1(exampleMD.ReadFile("example.md"))
	grr.Log(gidom.ReadMD(gidom.BaseContext(), b, h))
	b.NewWindow().Run().Wait()
}
