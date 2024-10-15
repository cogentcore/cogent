// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"cogentcore.org/cogent/mail"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
)

func main() {
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
