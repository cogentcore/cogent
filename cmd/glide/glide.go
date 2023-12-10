// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
	"goki.dev/glide/glide"
	"goki.dev/grr"
)

func main() { gimain.Run(app) }

func app() {
	b := gi.NewAppBody("glide").SetTitle("Glide")
	pg := glide.NewPage(b, "page")
	grr.Log(pg.OpenURL("https://google.com"))
	b.AddAppBar(pg.AppBar)
	b.NewWindow().Run().Wait()
}
