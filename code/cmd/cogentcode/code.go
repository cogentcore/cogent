// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"os"
	"path/filepath"
	"strings"

	"cogentcore.org/cogent/code/code"
	"cogentcore.org/cogent/code/codev"
	_ "cogentcore.org/cogent/code/icons"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/goosi"
	"cogentcore.org/core/grr"
)

func main() {
	pdir := gi.TheApp.AppDataDir()
	lfnm := filepath.Join(pdir, "cogentcode.log")

	// we must load the settings before initializing the console
	grr.Log(gi.LoadAllSettings())

	code.TheConsole.Init(lfnm)

	var path string
	var proj string

	ofs := goosi.TheApp.OpenFiles()
	if len(ofs) > 0 {
		path = ofs[0]
	} else if len(os.Args) > 1 {
		flag.StringVar(&path, "path", "", "path to open -- can be to a directory or a filename within the directory ")
		flag.StringVar(&proj, "proj", "", "project file to open -- typically has .code extension")
		// todo: other args?
		// flag.Parse() // note: this is causing delve to crash all the sudden!
		if path == "" && proj == "" {
			if flag.NArg() > 0 {
				ext := strings.ToLower(filepath.Ext(flag.Arg(0)))
				if ext == ".code" {
					proj = flag.Arg(0)
				} else {
					path = flag.Arg(0)
				}
			}
		}
	}

	if proj != "" {
		proj, _ = filepath.Abs(proj)
		codev.OpenCodeProj(proj)
	} else {
		if path != "" {
			path, _ = filepath.Abs(path)
		}
		codev.NewCodeProjPath(path)
	}
	// above NewCodeProj calls will have added to WinWait..
	gi.Wait()
}
