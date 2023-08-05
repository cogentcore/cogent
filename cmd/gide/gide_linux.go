// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build (linux && !android) || dragonfly || openbsd
// +build linux,!android dragonfly openbsd

package main

import "github.com/goki/gide/gide"

func init() {
	gide.DefaultKeyMap = gide.KeyMapName("LinuxStd")
	gide.SetActiveKeyMapName(gide.DefaultKeyMap)
}
