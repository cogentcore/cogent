// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vector

import (
	"embed"
	"io/fs"

	"cogentcore.org/core/grr"
	"cogentcore.org/core/icons"
)

// Icons contains all of the Vector icons.
//
//go:embed icons/*.svg
var Icons embed.FS

func init() {
	icons.AddFS(grr.Log1(fs.Sub(Icons, "icons")))
}
