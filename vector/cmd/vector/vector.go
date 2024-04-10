// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"os"

	"cogentcore.org/cogent/vector"
	"cogentcore.org/core/core"
)

func main() {
	vector.InitSettings()

	ofs := core.TheApp.OpenFiles()

	var fnms []string
	if len(ofs) > 0 {
		fnms = ofs
	} else if len(os.Args) > 1 {
		fnms = os.Args[1:]
	}

	if len(fnms) == 0 {
		vector.NewVectorWindow("")
	} else {
		for _, fnm := range fnms {
			vector.NewVectorWindow(fnm)
		}
	}
	core.Wait()
}
