// Copyright (c) 2018, The gide / GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"goki.dev/gi/v2/gimain"
)

func main() { gimain.Run(app) }

func app() {
	/*
			goosi.TheApp.SetName("pie")
			goosi.TheApp.SetAbout(`<code>Pie</code> is the interactive parser (pi) editor written in the <b>GoGi</b> graphical interface system, within the <b>GoKi</b> tree framework.  See <a href="https://goki.dev/pi">Gide on GitHub</a> and <a href="https://goki.dev/pi/wiki">Gide wiki</a> for documentation.<br>
		<br>
		Version: ` + pi.VersionInfo())

			// goosi.TheApp.SetQuitCleanFunc(func() {
			// 	fmt.Printf("Doing final Quit cleanup here..\n")
			// })

			gide.InitPrefs()
			piv.InitPrefs()

			gide.TheConsole.Init("") // must do this after changing stdout

			// var path string
			// var proj string
			//
			// // process command args
			// if len(os.Args) > 1 {
			// 	flag.StringVar(&path, "path", "", "path to open -- can be to a directory or a filename within the directory")
			// 	flag.StringVar(&proj, "proj", "", "project file to open -- typically has .gide extension")
			// 	// todo: other args?
			// 	flag.Parse()
			// 	if path == "" && proj == "" {
			// 		if flag.NArg() > 0 {
			// 			ext := strings.ToLower(filepath.Ext(flag.Arg(0)))
			// 			if ext == ".gide" {
			// 				proj = flag.Arg(0)
			// 			} else {
			// 				path = flag.Arg(0)
			// 			}
			// 		}
			// 	}
			// }

			recv := gi.WidgetBase{}
			recv.InitName(&recv, "pie_dummy")

			inQuitPrompt := false
			goosi.TheApp.SetQuitReqFunc(func() {
				if !inQuitPrompt {
					inQuitPrompt = true
					if piv.QuitReq() {
						goosi.TheApp.Quit()
					} else {
						inQuitPrompt = false
					}
				}
			})

			// if proj != "" {
			// 	proj, _ = filepath.Abs(proj)
			// 	gide.OpenGideProj(proj)
			// } else {
			// 	if path != "" {
			// 		path, _ = filepath.Abs(path)
			// 	}
			// 	gide.NewGideProjPath(path)
			// }

			piv.NewPiView()

			// above NewGideProj calls will have added to WinWait..
			gi.WinWait.Wait()
	*/
}
