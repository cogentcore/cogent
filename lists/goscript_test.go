// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !goscript

package main

import (
	"testing"
	"time"

	"cogentcore.org/core/base/goscript"
)

// TestGoScript runs the tests of the merges in JavaScript, as for the web,
// where they use the encoding/json of goscript.
func TestGoScript(t *testing.T) {
	goscript.Test(t, goscript.Options{Run: "TestMerge|TestItemsRoundTrip|TestStructType", Timeout: 10 * time.Minute, Browser: true, BrowserChannel: "chrome"})
}
