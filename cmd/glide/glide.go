// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"net/url"

	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
	"goki.dev/glide/glide"
	"goki.dev/goosi/events"
	"goki.dev/grr"
)

func main() { gimain.Run(app) }

func app() {
	sc := gi.NewScene("glide").SetTitle("Glide")
	pg := glide.NewPage(sc, "page")
	grr.Log0(pg.OpenURL("https://github.com/goki/gi"))

	gi.DefaultTopAppBar = func(tb *gi.TopAppBar) {
		gi.DefaultTopAppBarStd(tb)
		ch := tb.ChildByName("nav-bar").(*gi.Chooser)
		ch.AllowNew = true
		ch.OnChange(func(e events.Event) {
			u, err := url.Parse(ch.CurLabel)
			if err == nil {
				grr.Log0(pg.OpenURL(u.String()))
			} else {
				grr.Log0(pg.OpenURL("https://google.com/search?q=" + ch.CurLabel))
			}
			e.SetHandled()
		})
	}

	gi.NewWindow(sc).Run().Wait()
}
