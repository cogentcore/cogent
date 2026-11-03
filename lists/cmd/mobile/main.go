// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Command mobile is Cogent Lists on mobile platforms (see package lists),
// with the layout for small screens, in which the tree of lists, the items
// of a list, and an item are each in a full window, in turn. It also runs
// on desktop platforms, for trying that layout.
package main

import (
	_ "embed"

	"cogentcore.org/cogent/lists"
	"cogentcore.org/core/core"
)

// appIcon is the icon of the app, which core build also sets, but go
// run does not.
//
//go:embed icon.svg
var appIcon string

func main() {
	core.AppIcon = appIcon
	b := core.NewBody("Cogent Lists")
	lists.NewApp(b, "", true)
	b.RunMainWindow()
}
