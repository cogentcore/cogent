// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build windows
// +build windows

package main

func init() {
	DefaultKeyMap = KeyMapName("WindowsStandard")
	SetActiveKeyMapName(DefaultKeyMap)
}
