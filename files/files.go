// Copyright 2023 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Command files provides a cross-platform file explorer
// with powerful navigation and searching support.
package main

//go:generate core generate

import (
	"os"
	"strings"

	"cogentcore.org/core/core"
)

func main() {
	TheConsole.Init()
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
