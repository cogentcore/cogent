// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"os"
	"path/filepath"
	"strings"

	"cogentcore.org/cogent/code"
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/core"
)

func main() {
	pdir := core.TheApp.AppDataDir()
	lfnm := filepath.Join(pdir, "cogentcode.log")
	_ = lfnm

	// we must load the settings before initializing the console
	errors.Log(core.LoadAllSettings())

	// note: comment this out when printing out debug messages involving components of code itself!
	InitConsole(lfnm)

	var path string
	var proj string

	ofs := core.TheApp.OpenFiles()
	if len(ofs) > 0 {
		path = ofs[0]
	} else if len(os.Args) > 1 {
		flag.StringVar(&path, "path", "", "path to open -- can be to a directory or a filename within the directory ")
		flag.StringVar(&proj, "proj", "", "project file to open -- typically has .code extension")
		// todo: other args?
		flag.Parse() // note: this is causing delve to crash all the sudden!
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
		code.OpenCodeProject(proj)
	} else {
		if path != "" {
			path, _ = filepath.Abs(path)
		}
		code.NewCodeProjectPath(path)
	}
	core.Wait()
}
