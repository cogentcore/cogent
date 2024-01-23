// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"cogentcore.org/core/gi"
	"cogentcore.org/core/giv"
)

func main() {
	b := gi.NewAppBody("Cogent Settings")
	giv.SettingsView(b)
	b.StartMainWindow()
}
