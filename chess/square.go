// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package chess

import (
	"fmt"

	"cogentcore.org/core/core"
	"github.com/corentings/chess/v2"
)

// Square represents one square on the chess board.
type Square struct {
	core.Frame

	// chess is the chess app.
	chess *Chess

	// square is the position of the square.
	square chess.Square
}

func (sq *Square) Init() {
	sq.Frame.Init()

	sq.Frame.Updater(func() {
		piece := sq.chess.game.CurrentPosition().Board().Piece(sq.square)
		fmt.Println(piece)
	})
}
