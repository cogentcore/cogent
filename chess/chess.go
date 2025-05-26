// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package chess implements a chess app.
package chess

//go:generate core generate

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"github.com/corentings/chess/v2"
)

// Chess is the main widget of the chess app.
type Chess struct {
	core.Pages

	// config is the configuration for a new chess game.
	config Config

	game *chess.Game

	// currentSquare is the current square from which a piece may be moved.
	currentSquare chess.Square

	// moves is the list of potential moves for the current piece.
	moves []chess.Move
}

func (ch *Chess) Init() {
	ch.Pages.Init()

	ch.AddPage("home", func(pg *core.Pages) {
		core.NewText(pg).SetType(core.TextHeadlineLarge).SetText("Cogent Chess")
		core.NewForm(pg).SetStruct(&ch.config)
		new := core.NewButton(pg).SetText("New game")
		new.OnClick(func(e events.Event) {
			ch.game = chess.NewGame()
			pg.Open("play")
		})
	})

	ch.AddPage("play", ch.play)
}
