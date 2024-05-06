// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build (linux && !android) || dragonfly || openbsd
// +build linux,!android dragonfly openbsd

package main

func init() {
	DefaultKeyMap = KeyMapName("LinuxStandard")
	SetActiveKeyMapName(DefaultKeyMap)
}
