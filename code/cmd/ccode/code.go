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
	"cogentcore.org/core/gi"
	"cogentcore.org/core/goosi"
)

func main() {
	pdir := code.AppDataDir()
	lfnm := filepath.Join(pdir, "code.log")

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
		flag.Parse()
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

	recv := gi.WidgetBase{}
	recv.InitName(&recv, "code_dummy")

	inQuitPrompt := false
	goosi.TheApp.SetQuitReqFunc(func() {
		if !inQuitPrompt {
			inQuitPrompt = true
			if codev.QuitReq() {
				goosi.TheApp.Quit()
			} else {
				inQuitPrompt = false
			}
		}
	})

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
