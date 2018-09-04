// Copyright (c) 2018, The gide / GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"os"
	"path/filepath"

	"github.com/goki/gi"
	"github.com/goki/gi/gimain"
	"github.com/goki/gi/oswin"
	"github.com/goki/gide"
)

func main() {
	gimain.Main(func() {
		mainrun()
	})
}

func mainrun() {
	oswin.TheApp.SetName("gide")
	oswin.TheApp.SetAbout(`<code>gide</code> is a graphical-interface integrated-development-environment written in the <b>GoGi</b> graphical interface system, within the <b>GoKi</b> tree framework.  See <a href="https://github.com/goki/gide">gide on GitHub</a>`)

	// oswin.TheApp.SetQuitCleanFunc(func() {
	// 	fmt.Printf("Doing final Quit cleanup here..\n")
	// })

	var path string

	// process command args
	if len(os.Args) > 1 {
		flag.StringVar(&path, "path", "./", "path to open -- can be to a directory or a filename within the directory")
		// todo: other args?
		flag.Parse()
		if path == "" {
			if flag.NArg() > 0 {
				path = flag.Arg(0)
			}
		}
	}

	path, _ = filepath.Abs(path)
	gide.NewGideProj(path)
	// above NewGideProj calls will have added to WinWait..
	gi.WinWait.Wait()
}
