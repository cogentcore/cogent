// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package icons

import (
	"embed"

	"cogentcore.org/core/icons"
)

//go:embed *.svg
var Icons embed.FS

func init() {
	icons.AddFS(Icons)
}
