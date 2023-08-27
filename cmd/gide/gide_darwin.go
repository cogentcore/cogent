// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build darwin
// +build darwin

package main

import "goki.dev/gide/gide"

func init() {
	gide.DefaultKeyMap = gide.KeyMapName("MacStd")
	gide.SetActiveKeyMapName(gide.DefaultKeyMap)
}
