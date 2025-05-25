// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package chess

import (
	"fmt"
	"math/rand/v2"
	"strconv"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
	"github.com/corentings/chess/v2"
)

// play is the page for a chess game.
func (ch *Chess) play(pg *core.Pages) {
	core.NewText(pg).SetType(core.TextHeadlineLarge).SetText(ch.config.Mode.String())

	grid := core.NewFrame(pg)
	grid.Styler(func(s *styles.Style) {
		s.Display = styles.Grid
		s.Columns = 8
		s.Gap.Zero()
	})

	grid.Maker(func(p *tree.Plan) {
		for rank := range 8 {
			for file := range 8 {
				tree.AddAt(p, strconv.Itoa(rank)+strconv.Itoa(file), func(w *Square) {
					w.chess = ch
					w.Updater(func() {
						if ch.game.CurrentPosition().Turn() == chess.White || ch.config.Mode == ModeBot {
							w.square = chess.NewSquare(chess.File(file), chess.Rank(7-rank))
						} else {
							w.square = chess.NewSquare(chess.File(7-file), chess.Rank(rank))
						}
					})
				})
			}
		}
	})

	grid.Updater(func() {
		status := ch.game.CurrentPosition().Status()
		if status == chess.NoMethod {
			return
		}

		result := fmt.Sprintf("%v %v", status, ch.game.Outcome())
		d := core.NewBody(result)
		d.AddBottomBar(func(bar *core.Frame) {
			d.AddCancel(bar).SetText("View board")
			d.AddOK(bar).SetText("Home").OnClick(func(e events.Event) {
				pg.Open("home")
			})
		})
		d.RunDialog(ch)
	})
}

// makeMove makes the given move.
func (ch *Chess) makeMove(move *chess.Move) {
	ch.game.Move(move, nil)

	if ch.config.Mode == ModeBot {
		moves := ch.game.ValidMoves()
		if n := len(moves); n > 0 {
			ch.game.Move(&moves[rand.IntN(n)], nil)
		}
	}

	ch.currentSquare = chess.NoSquare
	ch.moves = nil
	ch.Update()
}
