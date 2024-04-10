// Copyright 2023 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/keyfun"
	"cogentcore.org/core/views"
)

func main() {
	b := core.NewBody("Cogent Files")

	fv := views.NewFileView(b)
	fv.Scene.OnKeyChord(func(e events.Event) {
		if keyfun.Of(e.KeyChord()) == keyfun.Accept {
			core.TheApp.OpenURL(fv.SelectedFile())
		}
	})

	b.RunMainWindow()
}
