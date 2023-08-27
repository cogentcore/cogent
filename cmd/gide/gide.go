// Copyright (c) 2018, The gide / GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/gimain"
	"github.com/goki/gi/oswin"
	"goki.dev/gide/gide"
	"goki.dev/gide/gidev"
)

func main() {
	gimain.Main(func() {
		mainrun()
	})
}

func mainrun() {
	oswin.TheApp.SetName("gide")
	oswin.TheApp.SetAbout(`<code>Gide</code> is a graphical-interface (gi) integrated-development-environment (ide) written in the <b>GoGi</b> graphical interface system, within the <b>GoKi</b> tree framework.  See <a href="https://goki.dev/gide/gide">Gide on GitHub</a> and <a href="https://goki.dev/gide/wiki">Gide wiki</a> for documentation.<br>
Gide is based on "projects" which are just directories containing files<br>
* Use <code>File/Open Path...</code> to open an existing directory.<br>
* Or <code>File/New Project...</code> to create a new directory for a new project<br>
<br>
Version: ` + gide.Prefs.VersionInfo())

	// oswin.TheApp.SetQuitCleanFunc(func() {
	// 	fmt.Printf("Doing final Quit cleanup here..\n")
	// })

	gide.InitPrefs()

	pdir := oswin.TheApp.AppPrefsDir()
	lfnm := filepath.Join(pdir, "gide.log")
	crnm := filepath.Join(pdir, "crash.log")

	gide.TheConsole.Init(lfnm)

	defer func() {
		r := recover()
		stack := string(debug.Stack())
		if r != nil {
			fmt.Printf("stacktrace from panic:\n%v\n%s\n", r, stack)
			fmt.Fprintf(gide.TheConsole.LogWrite, "stacktrace from panic:\n%s\n%s\n", r, stack)
		}
		cf, _ := os.Create(crnm)
		fmt.Fprintf(cf, "stacktrace from panic:\n%v\n%s\n", r, stack)
		cf.Close()
		gide.TheConsole.Close()
		if r != nil {
			os.Exit(1)
		}
	}()

	var path string
	var proj string

	ofs := oswin.TheApp.OpenFiles()
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

	recv := gi.Node2DBase{}
	recv.InitName(&recv, "gide_dummy")

	inQuitPrompt := false
	oswin.TheApp.SetQuitReqFunc(func() {
		if !inQuitPrompt {
			inQuitPrompt = true
			if gidev.QuitReq() {
				oswin.TheApp.Quit()
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
	gi.WinWait.Wait()
}
