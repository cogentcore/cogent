// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

func main() {
	/*
			system.TheApp.SetName("pie")
			system.TheApp.SetAbout(`<code>Pie</code> is the interactive parser (pi) editor written in the <b>Cogent Core</b> graphical interface system, within the <b>Goki</b> tree framework.  See <a href="https://cogentcore.org/core/parse">Code on GitHub</a> and <a href="https://cogentcore.org/core/parse/wiki">Code wiki</a> for documentation.<br>
		<br>
		Version: ` + pi.VersionInfo())

			// system.TheApp.SetQuitCleanFunc(func() {
			// 	fmt.Printf("Doing final Quit cleanup here..\n")
			// })

			code.InitSettings()
			piv.InitSettings()

			code.TheConsole.Init("") // must do this after changing stdout

			// var path string
			// var proj string
			//
			// // process command args
			// if len(os.Args) > 1 {
			// 	flag.StringVar(&path, "path", "", "path to open -- can be to a directory or a filename within the directory")
			// 	flag.StringVar(&proj, "proj", "", "project file to open -- typically has .code extension")
			// 	// todo: other args?
			// 	flag.Parse()
			// 	if path == "" && proj == "" {
			// 		if flag.NArg() > 0 {
			// 			ext := strings.ToLower(filepath.Ext(flag.Arg(0)))
			// 			if ext == ".code" {
			// 				proj = flag.Arg(0)
			// 			} else {
			// 				path = flag.Arg(0)
			// 			}
			// 		}
			// 	}
			// }

			recv := core.WidgetBase{}
			recv.InitName(&recv, "pie_dummy")

			inQuitPrompt := false
			system.TheApp.SetQuitReqFunc(func() {
				if !inQuitPrompt {
					inQuitPrompt = true
					if piv.QuitReq() {
						system.TheApp.Quit()
					} else {
						inQuitPrompt = false
					}
				}
			})

			// if proj != "" {
			// 	proj, _ = filepath.Abs(proj)
			// 	code.OpenCodeProj(proj)
			// } else {
			// 	if path != "" {
			// 		path, _ = filepath.Abs(path)
			// 	}
			// 	code.NewCodeProjPath(path)
			// }

			piv.NewPiView()

			// above NewCodeProj calls will have added to WinWait..
			core.WinWait.Wait()
	*/
}
