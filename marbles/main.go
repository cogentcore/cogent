// Copyright (c) 2020, Kai O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Command marbles is Cogent Marbles, which drops marbles on the graphs of
// equations, which they bounce off of.
package main

//go:generate core generate

import (
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/core"
)

func main() {
	TheSettings.Defaults()

	b := core.NewBody("Cogent Marbles")
	b.AddTopBar(func(bar *core.Frame) {
		core.NewToolbar(bar).Maker(TheGraph.MakeToolbar)
	})

	TheGraph.Init(b)
	TheGraph.OpenAutoSave()

	b.RunMainWindow()
}

// TODO(kai/marbles): better error handling
func HandleError(err error) bool {
	return errors.Log(err) != nil
}
