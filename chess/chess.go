// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package chess implements a chess app.
package chess

//go:generate core generate

import (
	"strconv"

	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
	"github.com/corentings/chess/v2"
)

// Chess is the main widget of the chess app.
type Chess struct {
	core.Frame

	game *chess.Game

	// moves is the list of potential moves for the current piece.
	moves []chess.Move
}

func (ch *Chess) Init() {
	ch.Frame.Init()
	ch.game = chess.NewGame()

	ch.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
		s.Display = styles.Grid
		s.Columns = 8
		s.Gap.Zero()
	})

	ch.Maker(func(p *tree.Plan) {
		for orank := range 8 {
			for file := range 8 {
				tree.AddAt(p, strconv.Itoa(orank)+strconv.Itoa(file), func(w *Square) {
					w.chess = ch
					w.square = chess.NewSquare(chess.File(file), chess.Rank(7-orank))
				})
			}
		}
	})
}
