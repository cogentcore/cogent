// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Command clock provides a clock app that provides customizable
// timers, alarms, stopwatches, and world clocks.
package main

import (
	"cogentcore.org/cogent/clock"
	"cogentcore.org/core/core"
)

func main() {
	b := core.NewBody("Cogent Clock")
	clock.NewClock(b)
	b.RunMainWindow()
}
