// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"os"

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
	// the page to open is the URL or local file in the first argument
	url := "https://google.com"
	if len(os.Args) > 1 {
		url = web.LocationURL(os.Args[1])
	}
	pg.OpenURL(url, nil)
	b.RunMainWindow()
}
