// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Command photos is Cogent Photos, an app for a library of photos, which
// opens the library in the directory given as its argument, if any,
// instead of that in the settings (~/Pictures/CogentPhotos by default).
package main

import (
	_ "embed"
	"os"
	"strings"

	"cogentcore.org/cogent/photos"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
)

//go:embed icon.svg
var appIcon string

func main() {
	core.AppIcon = appIcon
	b := core.NewBody("Cogent Photos")
	a := photos.NewApp(b)
	if len(os.Args) > 1 && !strings.HasPrefix(os.Args[1], "-") {
		a.Dir = os.Args[1] // the directory of the library
	}
	b.AddTopBar(func(bar *core.Frame) {
		core.NewToolbar(bar).Maker(a.MakeToolbar)
	})
	b.OnShow(func(e events.Event) {
		a.Start()
	})
	b.RunMainWindow()
}
