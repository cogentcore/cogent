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
)

func main() {
	b := core.NewBody("Cogent Files")

	fp := core.NewFilePicker(b)
	fp.Scene.OnKeyChord(func(e events.Event) {
		if keymap.Of(e.KeyChord()) == keymap.Accept {
			core.TheApp.OpenURL("file://" + fp.SelectedFile())
		}
	})
	b.AddAppBar(fp.MakeToolbar)

	b.RunMainWindow()
}
