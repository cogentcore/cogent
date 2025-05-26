// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package chess

import (
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
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
	sq.SetType(core.ButtonAction)
	sq.Styler(func(s *styles.Style) {
		s.Border.Width.Zero()
		s.MaxBorder.Width.Zero()
		s.Border.Radius.Zero()

		s.Padding.Set(units.Dp(16))

		if sq.isDark() {
			s.Background = colors.Uniform(colors.FromRGB(165, 117, 81))
			s.StateColor = colors.Uniform(colors.White)
		} else {
			s.Background = colors.Uniform(colors.FromRGB(235, 209, 166))
			s.StateColor = colors.Uniform(colors.Black)
		}

		if sq.moveTarget() != nil {
			s.StateColor = colors.Uniform(colors.Yellow)
			s.StateLayer += 0.2
		}
	})
	tree.AddChildInit(sq, "icon", func(w *core.Icon) {
		w.Styler(func(s *styles.Style) {
			s.Font.Size.Dp(64)
		})
	})

	sq.Updater(func() {
		piece := sq.chess.game.CurrentPosition().Board().Piece(sq.square)
		sq.SetIcon(iconForPiece(piece))
	})

	sq.OnClick(func(e events.Event) {
		if move := sq.moveTarget(); move != nil {
			sq.chess.makeMove(move)
			return
		}

		if sq.chess.currentSquare == sq.square {
			sq.chess.currentSquare = chess.NoSquare
			sq.chess.moves = nil
		} else {
			sq.chess.currentSquare = sq.square
			sq.chess.moves = sq.moves()
		}
		sq.chess.Restyle()
	})
}

// isDark returns whether this is a dark-colored square.
func (sq *Square) isDark() bool {
	sqSum := int(sq.square.File()) + int(sq.square.Rank())
	return sqSum%2 == 0
}

// moves returns the moves available from this square.
func (sq *Square) moves() []chess.Move {
	res := []chess.Move{}
	moves := sq.chess.game.ValidMoves()
	for _, move := range moves {
		if move.S1() == sq.square {
			res = append(res, move)
		}
	}
	return res
}

// moveTarget returns the potential move this square is a target of, if there is one.
// Otherwise, it returns nil.
func (sq *Square) moveTarget() *chess.Move {
	for _, move := range sq.chess.moves {
		if move.S2() == sq.square {
			return &move
		}
	}
	return nil
}
