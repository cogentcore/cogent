// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package chess

import (
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
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
		} else {
			s.Background = colors.Uniform(colors.FromRGB(235, 209, 166))
		}
	})
	tree.AddChildInit(sq, "icon", func(w *core.Icon) {
		w.Styler(func(s *styles.Style) {
			s.Font.Size.Dp(48)
		})
	})

	sq.Updater(func() {
		piece := sq.chess.game.CurrentPosition().Board().Piece(sq.square)
		sq.SetIcon(iconForPiece(piece))
	})
}

// isDark returns whether this is a dark-colored square.
func (sq *Square) isDark() bool {
	sqSum := int(sq.square.File()) + int(sq.square.Rank())
	return sqSum%2 == 0
}
