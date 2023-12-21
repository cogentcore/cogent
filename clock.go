// Copyright (c) 2023, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"time"

	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
)

func main() { gimain.Run(app) }

func app() {
	b := gi.NewAppBody("goki-clock")
	gi.NewLabel(b).SetText(time.Now().Format("Monday, January 2"))
	gi.NewLabel(b).SetText(time.Now().Format("3:04 PM"))
	b.NewWindow().Run().Wait()
}
