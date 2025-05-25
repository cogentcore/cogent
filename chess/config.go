// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package chess

// Config is the configuration for a new chess game.
type Config struct {
	Mode Modes
}

// Modes are the types of chess games.
type Modes int32 //enums:enum -transform sentence

const (
	// ModePassAndPlay is a chess game played locally between two players.
	ModePassAndPlay Modes = iota

	// ModeComputer is a chess game played against the computer (bot/AI).
	ModeComputer

	// ModeOnline is a chess game played online through the Lichess API.
	ModeOnline
)
