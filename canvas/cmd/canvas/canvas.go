// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	_ "embed"
	"os"

	"cogentcore.org/cogent/canvas"
	"cogentcore.org/core/core"
	"cogentcore.org/core/system"
)

// appIcon is the icon of the app, which core build also sets, but go
// install does not, as for an app installed with AppGoSymlink.
//
//go:embed icon.svg
var appIcon string

func main() {
	core.AppIcon = appIcon
	ofs := core.TheApp.OpenFiles() // note: on mac, this doesn't happen in time here
	// so you need to check in the OnShow for the first window and open then.

	var fnms []string
	if len(ofs) > 0 {
		fnms = ofs
	} else if !system.GenerateHTMLArg() && len(os.Args) > 1 {
		fnms = os.Args[1:]
	}

	if len(fnms) == 0 {
		canvas.NewWindow("")
	} else {
		for _, fnm := range fnms {
			canvas.NewWindow(fnm)
		}
	}
	core.Wait()
}
