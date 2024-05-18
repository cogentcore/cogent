// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"path/filepath"

	"cogentcore.org/cogent/numbers/databrowser"
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/core"
)

func main() {
	// opts := cli.DefaultOptions("cosh", "An interactive tool for running and compiling Cogent Shell (cosh).")
	// cli.Run(opts, &Config{}, Run, Build)

	b := core.NewBody("Cogent Numbers")
	br := databrowser.NewBrowser(b)
	ddr := errors.Log1(filepath.Abs("testdata"))
	br.SetDataRoot(ddr) // todo: args
	scdr := errors.Log1(filepath.Abs("testdata/proj1/dbscripts"))
	fmt.Println("script dir:", scdr)
	br.SetScriptsDir(scdr)
	br.GetScripts()
	b.AddAppBar(br.ConfigAppBar)

	go br.Shell()

	b.RunMainWindow()
}
