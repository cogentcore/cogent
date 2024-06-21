// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Command clock provides a clock app that provides customizable
// timers, alarms, stopwatches, and world clocks.
package main

import (
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
)

func main() {
	b := core.NewBody("Cogent Clock")

	ts := core.NewTabs(b)
	clock(ts)
	timers(ts)
	stopwatches(ts)
	alarms(ts)

	b.RunMainWindow()
}

func clock(ts *core.Tabs) {
	cl := ts.NewTab("Clock")
	cl.Styler(func(s *styles.Style) {
		s.CenterAll()
	})
	core.NewText(cl).SetType(core.TextHeadlineMedium).
		SetText(time.Now().Format("Monday, January 2"))
	core.NewText(cl).SetType(core.TextDisplayLarge).
		SetText(time.Now().Format("3:04 PM"))
}

type timer struct {
	left time.Time
}

func timers(ts *core.Tabs) {
	tr := ts.NewTab("Timers")
	trd := 15 * time.Minute
	trv := core.NewValue(&trd, "")
	tr.AddChild(trv)
	core.NewButton(tr).SetText("Start")
}

func stopwatches(ts *core.Tabs) {
	sw := ts.NewTab("Stopwatches")
	core.NewButton(sw).SetText("Start")
}

func alarms(ts *core.Tabs) {
	al := ts.NewTab("Alarms")
	core.NewButton(al).SetText("Create")
}
