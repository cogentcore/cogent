// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"cogentcore.org/core/core"
)

func main() {
	b := core.NewBody("Cogent Settings")
	core.SettingsEditor(b)
	b.RunMainWindow()
}
