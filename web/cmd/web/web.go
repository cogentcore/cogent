// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"cogentcore.org/cogent/web"
	"cogentcore.org/core/core"
)

func main() {
	b := core.NewBody("Cogent Web")
	pg := web.NewPage(b)
	b.AddTopBar(func(bar *core.Frame) {
		tb := core.NewToolbar(bar)
		tb.Maker(pg.MakeToolbar)
		pg.OnOpen = func(url string) { tb.Update() }
	})
	pg.OpenURL("https://google.com", nil)
	b.RunMainWindow()
}
