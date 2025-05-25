// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package chess

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
	"github.com/corentings/chess/v2"
)

// Square represents one square on the chess board.
type Square struct {
	core.Button

	// chess is the chess app.
	chess *Chess

	// square is the position of the square.
	square chess.Square
}

func (sq *Square) Init() {
	sq.Button.Init()
	sq.Styler(func(s *styles.Style) {
		s.Border.Width.Zero()
		s.Border.Radius.Zero()
	})

	sq.Updater(func() {
		piece := sq.chess.game.CurrentPosition().Board().Piece(sq.square)
		sq.SetIcon(iconForPiece(piece))
	})
}
