// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"cogentcore.org/cogent/glide"
	"cogentcore.org/core/gi"
)

func main() {
	b := gi.NewBody("glide").SetTitle("Glide")
	pg := glide.NewPage(b, "page")
	pg.OpenURL("https://google.com")
	b.AddAppBar(pg.AppBar)
	b.RunMainWindow()
}
