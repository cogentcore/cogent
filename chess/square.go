// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package chess

import "cogentcore.org/core/core"

// Square represents one square on the chess board.
type Square struct {
	core.Frame

	// chess is the chess app.
	chess *Chess
}
