// Copyright (c) 2018, The gide / Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"os"
	"path/filepath"
	"strings"

	"github.com/goki/gide/v2/gide"
	"github.com/goki/gide/v2/gidev"
	"goki.dev/gi"
	"goki.dev/goosi"
)

func main() {
	pdir := gide.AppDataDir()
	lfnm := filepath.Join(pdir, "gide.log")

	gide.TheConsole.Init(lfnm)

	var path string
	var proj string

	ofs := goosi.TheApp.OpenFiles()
	if len(ofs) > 0 {
		path = ofs[0]
	} else if len(os.Args) > 1 {
		flag.StringVar(&path, "path", "", "path to open -- can be to a directory or a filename within the directory ")
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

	recv := gi.WidgetBase{}
	recv.InitName(&recv, "gide_dummy")

	inQuitPrompt := false
	goosi.TheApp.SetQuitReqFunc(func() {
		if !inQuitPrompt {
			inQuitPrompt = true
			if gidev.QuitReq() {
				goosi.TheApp.Quit()
			} else {
				inQuitPrompt = false
			}
		}
	})

	if proj != "" {
		proj, _ = filepath.Abs(proj)
		gidev.OpenGideProj(proj)
	} else {
		if path != "" {
			path, _ = filepath.Abs(path)
		}
		gidev.NewGideProjPath(path)
	}
	// above NewGideProj calls will have added to WinWait..
	gi.Wait()
}
