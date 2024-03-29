// Copyright 2023 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"cogentcore.org/core/events"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/giv"
	"cogentcore.org/core/keyfun"
)

func main() {
	b := gi.NewBody("Cogent Files")

	fv := giv.NewFileView(b)
	fv.Scene.OnKeyChord(func(e events.Event) {
		if keyfun.Of(e.KeyChord()) == keyfun.Accept {
			gi.TheApp.OpenURL(fv.SelectedFile())
		}
	})

	b.RunMainWindow()
}
