// Copyright 2023 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Command files provides a cross-platform file explorer
// with powerful navigation and searching support.
package main

//go:generate core generate

import (
	_ "embed"
	"os"
	"strings"

	"cogentcore.org/core/core"
)

// appIcon is the icon of the app, in the Dock on macOS.
//
//go:embed icon.svg
var appIcon string

func main() {
	TheConsole.Init()
	core.AppIcon = appIcon
	root, err := os.UserHomeDir()
	if err != nil {
		root = "."
	}
	if len(os.Args) > 1 && !strings.HasPrefix(os.Args[1], "-") {
		root = os.Args[1]
	}
	b := core.NewBody("Cogent Files")
	fl := NewFiles(b).SetRoot(root)
	b.AddTopBar(func(bar *core.Frame) {
		core.NewToolbar(bar).Maker(fl.MakeToolbar)
	})
	b.RunMainWindow()
}
