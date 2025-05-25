// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Command chess implements a chess app.
package main

import (
	"cogentcore.org/cogent/chess"
	"cogentcore.org/core/core"
)

func main() {
	b := core.NewBody("Cogent Chess")
	chess.NewChess(b)
	b.RunMainWindow()
}
