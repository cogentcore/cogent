// Copyright 2023 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Command files provides a cross-platform file explorer
// with powerful navigation and searching support.
package main

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/views"
)

func main() {
	b := core.NewBody("Cogent Files")

	fv := views.NewFilePicker(b)
	fv.Scene.OnKeyChord(func(e events.Event) {
		if keymap.Of(e.KeyChord()) == keymap.Accept {
			core.TheApp.OpenURL(fv.SelectedFile())
		}
	})

	b.RunMainWindow()
}
