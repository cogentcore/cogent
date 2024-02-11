// Copyright (c) 2021, The Vector Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"cogentcore.org/cogent/vector"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/oswin"
)

func main() {
	gi.SetAppName("vector")
	gi.SetAppAbout(`Vector is a Go-rendered interactive drawing program for SVG vector dawings.  See <a href="https://goki.dev/vector">Vector on GitHub</a><br>
<br>
Version: ` + vector.Prefs.VersionInfo())

	vector.InitPrefs()

	/*
			pdir := oswin.TheApp.AppDataDir()
			pnm := filepath.Join(pdir, "vector.log")

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
		vector.NewDrawing(vector.Prefs.Size)
	} else {
		fdir, _ := filepath.Split(fnms[0])
		os.Chdir(fdir)
		fmt.Printf("fdir: %s\n", fdir)
		for _, fnm := range fnms {
			fmt.Println(fnm)
			vector.NewVectorWindow(fnm)
		}
	}
	gi.WinWait.Wait()
}
