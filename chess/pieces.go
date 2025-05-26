// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package chess

import (
	"embed"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/icons"
	"github.com/corentings/chess/v2"
)

//go:embed pieces
var pieces embed.FS

func iconForPiece(p chess.Piece) icons.Icon {
	if p == chess.NoPiece {
		return icons.Blank
	}
	pname := p.Color().String() + p.Type().String()
	b := errors.Log1(pieces.ReadFile("pieces/" + pname + ".svg"))
	return icons.Icon(b)
}
