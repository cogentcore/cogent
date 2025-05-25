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
}

func (ch *Chess) Init() {
	ch.Frame.Init()
	ch.game = chess.NewGame()

	ch.Frame.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
		s.Direction = styles.Column
	})

	ch.Frame.Maker(func(p *tree.Plan) {
		for i := range 8 {
			for j := range 8 {
				tree.AddAt(p, strconv.Itoa(i)+strconv.Itoa(j), func(w *Square) {
					w.chess = ch
				})
			}
		}
	})
}
