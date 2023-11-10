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
	sc := gi.NewScene("glide").SetTitle("Glide")
	pg := glide.NewPage(sc, "page")
	grr.Log0(pg.OpenURL("https://google.com"))
	gi.DefaultTopAppBar = pg.TopAppBar
	gi.NewWindow(sc).Run().Wait()
}
