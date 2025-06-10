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
	cl, _ := ts.NewTab("Clock")
	cl.Styler(func(s *styles.Style) {
		s.CenterAll()
	})
	date := core.NewText(cl).SetType(core.TextHeadlineMedium)
	date.Updater(func() {
		date.SetText(time.Now().Format("Monday, January 2"))
	})
	tm := core.NewText(cl).SetType(core.TextDisplayLarge)
	tm.Updater(func() {
		tm.SetText(time.Now().Format("3:04:05 PM"))
	})
	tm.Animate(func(a *core.Animation) {
		tm.UpdateRender()
	})
}

type timer struct {
	left time.Time
}

func timers(ts *core.Tabs) {
	tr, _ := ts.NewTab("Timers")
	trd := 15 * time.Minute
	trv := core.NewValue(&trd, "")
	tr.AddChild(trv)
	core.NewButton(tr).SetText("Start")
}

func stopwatches(ts *core.Tabs) {
	sw, _ := ts.NewTab("Stopwatches")
	core.NewButton(sw).SetText("Start")
}

func alarms(ts *core.Tabs) {
	al, _ := ts.NewTab("Alarms")
	core.NewButton(al).SetText("Create")
}
