// Copyright (c) 2021, The Grid Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/gimain"
	"github.com/goki/gi/oswin"
	"github.com/goki/grid/grid"
)

func main() {
	// vgpu.Debug = true
	gimain.Main(func() {
		mainrun()
	})
}

func mainrun() {
	gi.SetAppName("grid")
	gi.SetAppAbout(`Grid is a Go-rendered interactive drawing program for SVG vector dawings.  See <a href="https://github.com/goki/grid">Grid on GitHub</a><br>
<br>
Version: ` + grid.Prefs.VersionInfo())

	grid.InitPrefs()

	/*
			pdir := oswin.TheApp.AppPrefsDir()
			pnm := filepath.Join(pdir, "grid.log")

			lf, err := os.Create(pnm)
			if err == nil {
				os.Stdout = lf
				os.Stderr = lf
				log.SetOutput(lf)
			}

		defer func() {
			if r := recover(); r != nil {
				fmt.Println("stacktrace from panic: \n" + string(debug.Stack()))
				lf.Close()
				os.Exit(1)
			}
			lf.Close()
		}()
	*/

	ofs := oswin.TheApp.OpenFiles()

	var fnms []string
	if len(ofs) > 0 {
		fnms = ofs
	} else if len(os.Args) > 1 {
		fnms = os.Args[1:]
	}

	if len(fnms) == 0 {
		os.Chdir(gi.Prefs.User.HomeDir)
		grid.NewDrawing(grid.Prefs.Size)
	} else {
		fdir, _ := filepath.Split(fnms[0])
		os.Chdir(fdir)
		fmt.Printf("fdir: %s\n", fdir)
		for _, fnm := range fnms {
			fmt.Println(fnm)
			grid.NewGridWindow(fnm)
		}
	}
	gi.WinWait.Wait()
}
