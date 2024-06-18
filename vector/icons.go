// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vector

import (
	"embed"

	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/icons"
)

// Icons contains all of the Vector icons.
//
//go:embed icons/*.svg
var Icons embed.FS

func init() {
	icons.AddFS(fsx.Sub(Icons, "icons"))
}
