// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"log/slog"
	"path/filepath"

	"cogentcore.org/cogent/numbers/databrowser"
	"cogentcore.org/core/core"
)

func main() {
	// opts := cli.DefaultOptions("cosh", "An interactive tool for running and compiling Cogent Shell (cosh).")
	// cli.Run(opts, &Config{}, Run, Build)

	b := core.NewBody("Cogent Data Browser")
	br := databrowser.NewBrowser(b)
	br.SetDataRoot("/Users/oreilly/gruntdat/wc/hpc2/oreilly/deep_move/jobs") // todo: args
	scdr, err := filepath.Abs("./testdata")
	if err != nil {
		slog.Error(err.Error())
	}
	br.SetScriptsDir(scdr)
	br.GetScripts()
	b.AddAppBar(br.ConfigAppBar)

	go br.Shell()

	b.RunMainWindow()
}
