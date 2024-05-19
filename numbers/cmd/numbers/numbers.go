// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"cogentcore.org/cogent/numbers/databrowser"
	"cogentcore.org/cogent/numbers/numshell"
	"cogentcore.org/core/core"
	"github.com/traefik/yaegi/interp"
)

func main() {
	// opts := cli.DefaultOptions("cosh", "An interactive tool for running and compiling Cogent Shell (cosh).")
	// cli.Run(opts, &Config{}, Run, Build)

	b := core.NewBody("Cogent Numbers")
	core.NewText(b).SetText("Welcome to the Numbers App")
	// b.AddAppBar(br.ConfigAppBar)

	in := numshell.NewInterpreter(interp.Options{})
	in.Interp.Use(databrowser.Symbols)
	go in.Interactive()

	b.RunMainWindow()
}
