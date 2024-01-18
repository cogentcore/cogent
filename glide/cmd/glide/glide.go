// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"cogentcore.org/cogent/glide"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/grr"
)

func main() {
	b := gi.NewAppBody("glide").SetTitle("Glide")
	pg := glide.NewPage(b, "page")
	grr.Log(pg.OpenURL("https://google.com"))
	b.AddAppBar(pg.AppBar)
	b.NewWindow().Run().Wait()
}
