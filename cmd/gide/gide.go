// Copyright (c) 2018, The gide / GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
	"goki.dev/gide/v2/gide"
	"goki.dev/gide/v2/gidev"
	"goki.dev/goosi"
	"goki.dev/grr"
)

func main() { gimain.Run(app) }

func app() {
	// gi.FocusTrace = true
	// gi.LayoutTrace = true
	// gi.KeyEventTrace = true

	// goosi.TheApp.SetQuitCleanFunc(func() {
	// 	fmt.Printf("Doing final Quit cleanup here..\n")
	// })

	goosi.TheApp.SetName("gide") // needs to happen before prefs

	gide.InitPrefs()

	pdir := gi.AppDataDir()
	lfnm := filepath.Join(pdir, "gide.log")
	crnm := filepath.Join(pdir, "crash.log")

	_ = lfnm
	gide.TheConsole.Init(lfnm)

	defer func() {
		if r := recover(); r != nil {

			stack := string(debug.Stack())

			print := func(w io.Writer) {
				fmt.Fprintln(w, "panic:", r)
				fmt.Fprintln(w, "")
				fmt.Fprintln(w, "----- START OF STACK TRACE: -----")
				fmt.Fprintln(w, stack)
				fmt.Fprintln(w, "----- END OF STACK TRACE -----")
			}

			print(os.Stdout)
			print(gide.TheConsole.LogWrite)

			cf, err := os.Create(crnm)
			if grr.Log(err) == nil {
				print(cf)
				cf.Close()
			}
		}
	}()

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
