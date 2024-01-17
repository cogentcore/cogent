// Copyright (c) 2023, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"embed"

	"cogentcore.org/cogent/glide/gidom"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/grr"
)

//go:embed example.html
var exampleHTML embed.FS

func main() {
	b := gi.NewAppBody("gidom-html")
	h := grr.Log1(exampleHTML.ReadFile("example.html"))
	grr.Log(gidom.ReadHTML(gidom.BaseContext(), b, bytes.NewBuffer(h)))
	b.NewWindow().Run().Wait()
}
