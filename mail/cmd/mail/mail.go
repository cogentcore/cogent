// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"cogentcore.org/cogent/mail"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	_ "embed"
)

// appIcon is the icon of the app, which core build also sets, but go
// install does not, as for an app installed with AppGoSymlink.
//
//go:embed icon.svg
var appIcon string

func main() {
	core.AppIcon = appIcon
	b := core.NewBody("Cogent Mail")
	a := mail.NewApp(b)
	b.AddTopBar(func(bar *core.Frame) {
		core.NewToolbar(bar).Maker(a.MakeToolbar)
	})
	b.OnShow(func(e events.Event) {
		a.GetMail()
	})
	b.RunMainWindow()
}
