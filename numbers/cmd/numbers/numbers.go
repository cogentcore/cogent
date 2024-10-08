// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Command numbers is an interactive cli for
// Numbers data management, analysis and math system.
package main

import (
	"cogentcore.org/core/cli"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/goal/interpreter"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/tensor/databrowser"
	"cogentcore.org/core/tensor/datafs"
	"cogentcore.org/core/tree"
)

func main() { //types:skip
	opts := cli.DefaultOptions("numbers", "Cogent Numbers, an interactive tool for data management, analysis and math.")
	cfg := &interpreter.Config{}
	cfg.InteractiveFunc = Interactive
	cli.Run(opts, cfg, interpreter.Run, interpreter.Build)
}

// Interactive runs an interactive shell that allows the user to input numbers.
func Interactive(c *interpreter.Config, in *interpreter.Interpreter) error {
	in.HistFile = "~/.numbers-history"
	br := databrowser.NewBrowserWindow(datafs.CurRoot, "Cogent Numbers")
	b := br.Parent.(*core.Body)
	b.AddTopBar(func(bar *core.Frame) {
		tb := core.NewToolbar(bar)
		// tb.Maker(tbv.MakeToolbar)
		tb.Maker(func(p *tree.Plan) {
			tree.Add(p, func(w *core.Button) {
				w.SetText("README").SetIcon(icons.FileMarkdown).
					SetTooltip("open README help file").OnClick(func(e events.Event) {
					core.TheApp.OpenURL("https://github.com/cogentcore/core/blob/main/tensor/examples/planets/README.md")
				})
			})
		})
	})
	b.OnShow(func(e events.Event) {
		go func() {
			if c.Expr != "" {
				in.Eval(c.Expr)
			}
			in.Interactive()
		}()
	})
	core.Wait()
	return nil
}

