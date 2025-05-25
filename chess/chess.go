// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package chess implements a chess app.
package chess

//go:generate core generate

import (
	"fmt"
	"strconv"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
	"github.com/corentings/chess/v2"
)

// Chess is the main widget of the chess app.
type Chess struct {
	core.Frame

	game *chess.Game

	// currentSquare is the current square from which a piece may be moved.
	currentSquare chess.Square

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
		for rank := range 8 {
			for file := range 8 {
				tree.AddAt(p, strconv.Itoa(rank)+strconv.Itoa(file), func(w *Square) {
					w.chess = ch
					w.Updater(func() {
						if ch.game.CurrentPosition().Turn() == chess.White {
							w.square = chess.NewSquare(chess.File(file), chess.Rank(7-rank))
						} else {
							w.square = chess.NewSquare(chess.File(7-file), chess.Rank(rank))
						}
					})
				})
			}
		}
	})

	ch.Updater(func() {
		status := ch.game.CurrentPosition().Status()
		if status == chess.NoMethod {
			return
		}

		result := fmt.Sprintf("%v %v", status, ch.game.Outcome())
		d := core.NewBody(result)
		d.AddBottomBar(func(bar *core.Frame) {
			d.AddCancel(bar).SetText("View board")
			d.AddOK(bar).SetText("New game").OnClick(func(e events.Event) {
				ch.game = chess.NewGame()
				ch.Update()
			})
		})
		d.RunDialog(ch)
	})
}
