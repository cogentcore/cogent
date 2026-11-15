// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Command settings is Cogent Settings, which edits the settings of the
// Cogent Core apps.
package main

import (
	"cogentcore.org/core/core"
)

func main() {
	b := core.NewBody("Cogent Settings")
	core.SettingsEditor(b)
	b.RunMainWindow()
}
