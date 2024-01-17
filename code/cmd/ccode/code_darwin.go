// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build darwin
// +build darwin

package main

import "cogentcore.org/cogent/code/code"

func init() {
	code.DefaultKeyMap = code.KeyMapName("MacStd")
	code.SetActiveKeyMapName(code.DefaultKeyMap)
}
