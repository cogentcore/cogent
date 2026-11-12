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
	"cogentcore.org/core/tree"
)

// appIcon is the icon of the app, which core build also sets, but go
// install does not, as for an app installed with AppGoSymlink.
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
	InitBody(b, root)
	b.RunMainWindow()
}

// InitBody initializes the given body of a window as a Files window, with
// the given root directory, and returns its [Files]. It is used for the
// main window of the app, and for the new windows that tabs are dragged
// into (see [core.Tabs.DragToWindow]).
func InitBody(b *core.Body, root string) *Files {
	core.NewWindowFunc = newWindow
	fl := NewFiles(b).SetRoot(root)
	b.AddTopBar(func(bar *core.Frame) {
		core.NewToolbar(bar).Maker(fl.MakeToolbar)
	})
	return fl
}

// newWindow returns the body of a new Files window, for
// [core.NewWindowFunc], with the root of the Files window that it
// is made from, or else the home directory.
func newWindow(from *core.Scene) *core.Body {
	root := ""
	if from != nil && from.Body != nil {
		if ff := tree.ChildByType[*Files](from.Body); ff != nil {
			root = ff.Root
		}
	}
	if root == "" {
		root, _ = os.UserHomeDir()
	}
	b := core.NewBody("Cogent Files")
	InitBody(b, root)
	return b
}
