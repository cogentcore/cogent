// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package chess

import (
	"embed"

	"cogentcore.org/core/base/errors"
	"github.com/corentings/chess/v2"
)

//go:embed pieces
var pieces embed.FS

func svgForPiece(p chess.Piece) string {
	pname := p.Color().String() + p.Type().String()
	b := errors.Log1(pieces.ReadFile(pname))
	return string(b)
}
