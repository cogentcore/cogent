// Copyright (c) 2018, The gide / GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"os"
	"path/filepath"
	"strings"

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
	oswin.TheApp.SetAbout(`<code>Gide</code> is a graphical-interface (gi) integrated-development-environment (ide) written in the <b>GoGi</b> graphical interface system, within the <b>GoKi</b> tree framework.  See <a href="https://github.com/goki/gide">Gide on GitHub</a> and <a href="https://github.com/goki/gide/wiki">Gide wiki</a> for documentation.<br>
Gide is based on "projects" which are just directories containing files<br>
* Use <code>File/Open Path...</code> to open an existing directory.<br>
* Or <code>File/New Project...</code> to create a new directory for a new project<br>`)

	// oswin.TheApp.SetQuitCleanFunc(func() {
	// 	fmt.Printf("Doing final Quit cleanup here..\n")
	// })

	gide.InitPrefs()

	var path string
	var proj string

	// process command args
	if len(os.Args) > 1 {
		flag.StringVar(&path, "path", "", "path to open -- can be to a directory or a filename within the directory")
		flag.StringVar(&proj, "proj", "", "project file to open -- typically has .gide extension")
		// todo: other args?
		flag.Parse()
		if path == "" && proj == "" {
			if flag.NArg() > 0 {
				ext := strings.ToLower(filepath.Ext(flag.Arg(0)))
				if ext == ".gide" {
					proj = flag.Arg(0)
				} else {
					path = flag.Arg(0)
				}
			}
		}
	}

	recv := gi.Node2DBase{}
	recv.InitName(&recv, "gide_dummy")

	inQuitPrompt := false
	oswin.TheApp.SetQuitReqFunc(func() {
		if !inQuitPrompt {
			inQuitPrompt = true
			if gide.QuitReq() {
				oswin.TheApp.Quit()
			} else {
				inQuitPrompt = false
			}
		}
	})

	if proj != "" {
		proj, _ = filepath.Abs(proj)
		gide.OpenGideProj(proj)
	} else {
		if path != "" {
			path, _ = filepath.Abs(path)
		}
		gide.NewGideProjPath(path)
	}
	// above NewGideProj calls will have added to WinWait..
	gi.WinWait.Wait()
}
