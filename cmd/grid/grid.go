// Copyright (c) 2021, The Grid Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"os"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/gimain"
	"github.com/goki/grid/grid"
)

func main() {
	gimain.Main(func() {
		mainrun()
	})
}

func mainrun() {
	gi.SetAppName("grid")
	gi.SetAppAbout(`Grid is a Go-rendered interactive drawing program for SVG vector dawings.  See <a href="https://github.com/goki/grid">Grid on GitHub</a>`)

	fnm := ""
	if len(os.Args) > 1 {
		fnm = os.Args[1]
	}

	grid.NewGridWindow(fnm)
	gi.WinWait.Wait()
}
