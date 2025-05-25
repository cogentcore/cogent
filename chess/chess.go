// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package chess implements a chess app.
package chess

//go:generate core generate

import (
	"cogentcore.org/core/core"
	"github.com/corentings/chess/v2"
)

// Chess is the main widget of the chess app.
type Chess struct {
	core.Frame

	game *chess.Game
}
