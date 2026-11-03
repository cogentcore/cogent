// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Command lists is an app for lists, such as to-do and shopping lists,
// in groups, with the items of each list in categories, and the fields
// of the items given by the type of the list.
package main

//go:generate core generate

import (
	_ "embed"
	"os"
	"strings"

	"cogentcore.org/core/base/errors"
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
	dir := core.TheApp.AppDataDir()
	if len(os.Args) > 1 && !strings.HasPrefix(os.Args[1], "-") {
		dir = os.Args[1]
	}
	ls := NewLists(b)
	errors.Log(ls.Open(dir))
	b.AddTopBar(func(bar *core.Frame) {
		core.NewToolbar(bar).Maker(ls.MakeToolbar)
	})
	b.RunMainWindow()
}
