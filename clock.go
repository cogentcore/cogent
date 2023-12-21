// Copyright (c) 2023, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"time"

	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
	"goki.dev/gi/v2/giv"
	"goki.dev/girl/styles"
)

func main() { gimain.Run(app) }

func app() {
	b := gi.NewAppBody("goki-clock")

	ts := gi.NewTabs(b).SetDeleteTabButtons(false)
	clock(ts)
	timers(ts)
	stopwatches(ts)
	alarms(ts)

	b.NewWindow().Run().Wait()
}

func clock(ts *gi.Tabs) {
	cl := ts.NewTab("Clock").Style(func(s *styles.Style) {
		s.Justify.Content = styles.Center
		s.Align.Content = styles.Center
		s.Align.Items = styles.Center
	})
	gi.NewLabel(cl).SetType(gi.LabelHeadlineMedium).
		SetText(time.Now().Format("Monday, January 2"))
	gi.NewLabel(cl).SetType(gi.LabelDisplayLarge).
		SetText(time.Now().Format("3:04 PM"))
}

type timer struct {
	left time.Time
}

func timers(ts *gi.Tabs) {
	tr := ts.NewTab("Timers")
	trd := 15 * time.Minute
	giv.NewValue(tr, &trd)
	gi.NewButton(tr).SetText("Start")
}

func stopwatches(ts *gi.Tabs) {
	sw := ts.NewTab("Stopwatches")
	gi.NewButton(sw).SetText("Start")
}

func alarms(ts *gi.Tabs) {
	al := ts.NewTab("Alarms")
	gi.NewButton(al).SetText("Create")
}
