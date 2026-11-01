// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js

package code

import (
	"bytes"
	"os"

	"cogentcore.org/lab/goal/goalfmt"
)

// formatGoal formats the given saved Goal file and fixes its imports,
// as goimports does for Go files, with goalfmt, writing it if it changed,
// and returns whether it did.
func formatGoal(fname string) (bool, error) {
	src, err := os.ReadFile(fname)
	if err != nil {
		return false, err
	}
	res, err := goalfmt.Source(fname, src, nil)
	if err != nil || bytes.Equal(src, res) {
		return false, err
	}
	return true, os.WriteFile(fname, res, 0666)
}
