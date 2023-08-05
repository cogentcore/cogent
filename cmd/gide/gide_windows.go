// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build windows
// +build windows

package main

import "github.com/goki/gide/gide"

func init() {
	gide.DefaultKeyMap = gide.KeyMapName("WindowsStd")
	gide.SetActiveKeyMapName(gide.DefaultKeyMap)
}
