// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Command lists is Cogent Lists on desktop platforms (see package lists),
// which opens the lists in the directory given as its argument, if any.
package main

import (
	_ "embed"
	"os"
	"strings"

	"cogentcore.org/cogent/lists"
	"cogentcore.org/core/core"
)

// appIcon is the icon of the app, which core build also sets, but go
// install does not, as for an app installed with AppGoSymlink.
//
//go:embed icon.svg
var appIcon string

func main() {
	core.AppIcon = appIcon
	b := core.NewBody("Cogent Lists")
	dir := ""
	if len(os.Args) > 1 && !strings.HasPrefix(os.Args[1], "-") {
		dir = os.Args[1]
	}
	lists.NewApp(b, dir)
	b.RunMainWindow()
}
