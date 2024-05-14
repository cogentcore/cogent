// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"cogentcore.org/cogent/numbers/databrowser"
	"cogentcore.org/core/core"
)

func main() {
	b := core.NewBody("Cogent Data Browser")
	br := databrowser.NewBrowser(b)
	br.SetDataRoot("/Users/oreilly/gruntdat/wc/hpc2/oreilly/deep_move").SetScriptsDir("testdata")
	br.GetScripts()
	b.AddAppBar(br.ConfigAppBar)

	b.RunMainWindow()
}
