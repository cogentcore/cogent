// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/giv"
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
	cl := ts.NewTab("Clock").Style(func(s *styles.Style) {
		s.Justify.Content = styles.Center
		s.Align.Content = styles.Center
		s.Align.Items = styles.Center
	})
	core.NewLabel(cl).SetType(core.LabelHeadlineMedium).
		SetText(time.Now().Format("Monday, January 2"))
	core.NewLabel(cl).SetType(core.LabelDisplayLarge).
		SetText(time.Now().Format("3:04 PM"))
}

type timer struct {
	left time.Time
}

func timers(ts *core.Tabs) {
	tr := ts.NewTab("Timers")
	trd := 15 * time.Minute
	giv.NewValue(tr, &trd)
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
