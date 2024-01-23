// Copyright 2023 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"cogentcore.org/core/events"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/giv"
)

func main() {
	b := gi.NewAppBody("Cogent Files")

	fv := giv.NewFileView(b)
	fv.OnDoubleClick(func(e events.Event) {
		gi.OpenURL(fv.SelectedFile())
	})

	b.StartMainWindow()
}
