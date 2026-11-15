// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package chess

import (
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"github.com/corentings/chess/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// squares returns the squares of the board of the given app, by their
// position.
func squares(ch *Chess) map[chess.Square]*Square {
	res := map[chess.Square]*Square{}
	for w := range ch.AllWidgets() {
		if sq, ok := w.(*Square); ok {
			res[sq.square] = sq
		}
	}
	return res
}

// assertBoard asserts that the board of the given app shows the pieces of
// the current position of its game.
func assertBoard(t *testing.T, ch *Chess) {
	t.Helper()
	sqs := squares(ch)
	require.Len(t, sqs, 64)
	board := ch.game.CurrentPosition().Board()
	for s, sq := range sqs {
		assert.Equal(t, iconForPiece(board.Piece(s)), sq.Icon, "%v", s)
	}
}

// newGame returns a new body with a chess app with the given mode, and a
// function that starts a new game.
func newGame(t *testing.T, mode Modes) (*core.Body, *Chess, func()) {
	t.Helper()
	t.Cleanup(core.TrackWindows())
	b := core.NewBody()
	ch := NewChess(b)
	ch.config.Mode = mode
	return b, ch, core.Locked(b, func() {
		bt := core.ButtonByText(ch, "New game")
		require.NotNil(t, bt)
		bt.Send(events.Click)
	})
}

// TestPassAndPlay tests moving pieces in a game between two players, in
// which the board turns to the player whose turn it is.
func TestPassAndPlay(t *testing.T) {
	b, ch, start := newGame(t, ModePassAndPlay)
	core.RenderImage(b, start, core.Locked(b, func() {
		assertBoard(t, ch)
		sqs := squares(ch)
		sqs[chess.E2].Send(events.Click)
		assert.Equal(t, chess.E2, ch.currentSquare)
		assert.Len(t, ch.moves, 2)
		assert.NotNil(t, sqs[chess.E4].moveTarget())
		sqs[chess.E2].Send(events.Click) // again: deselects
		assert.Equal(t, chess.NoSquare, ch.currentSquare)
		assert.Empty(t, ch.moves)
		sqs[chess.E2].Send(events.Click)
		sqs[chess.E4].Send(events.Click)
		assert.Equal(t, chess.Black, ch.game.CurrentPosition().Turn())
		assert.Empty(t, ch.moves)
	}), core.Locked(b, func() {
		assertBoard(t, ch)
		assert.Equal(t, icons.Blank, squares(ch)[chess.E2].Icon)
	}))
}

// TestBot tests that the bot moves after the player.
func TestBot(t *testing.T) {
	b, ch, start := newGame(t, ModeBot)
	core.RenderImage(b, start, core.Locked(b, func() {
		sqs := squares(ch)
		sqs[chess.D2].Send(events.Click)
		sqs[chess.D4].Send(events.Click)
		assert.Equal(t, chess.White, ch.game.CurrentPosition().Turn(), "after the move of the bot")
		assert.Len(t, ch.game.Moves(), 2)
	}), core.Locked(b, func() {
		assertBoard(t, ch)
	}))
}

// TestGameOver tests the dialog at the end of a game, which can go back
// to the home page.
func TestGameOver(t *testing.T) {
	b, ch, start := newGame(t, ModePassAndPlay)
	core.RenderImage(b, start, core.Locked(b, func() {
		for _, m := range []string{"f3", "e5", "g4"} {
			require.NoError(t, ch.game.PushNotationMove(m, chess.AlgebraicNotation{}, nil))
		}
		ch.Update()
		sqs := squares(ch)
		sqs[chess.D8].Send(events.Click)
		sqs[chess.H4].Send(events.Click) // checkmate
		assert.Equal(t, chess.BlackWon, ch.game.Outcome())
	}), core.Locked(b, func() {
		bt := core.ButtonByText(core.TopScene(b), "Home")
		require.NotNil(t, bt, "the dialog")
		bt.Send(events.Click)
	}), core.Locked(b, func() {
		assert.NotNil(t, core.ButtonByText(ch, "New game"), "the home page")
	}))
}

// TestModes tests the names of the modes.
func TestModes(t *testing.T) {
	assert.Equal(t, "Pass and play", ModePassAndPlay.String())
	assert.Equal(t, []Modes{ModePassAndPlay, ModeBot, ModeOnline}, ModesValues())
}
