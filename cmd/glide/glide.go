// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
	"goki.dev/glide/glide"
)

func main() { gimain.Run(app) }

func app() {
	sc := gi.NewScene("glide").SetTitle("Glide")
	glide.NewPage(sc, "page").OpenURL("https://google.com")
	gi.NewWindow(sc).Run().Wait()
}
