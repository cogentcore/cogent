// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package clock provides a clock app that provides customizable
// timers, alarms, stopwatches, and world clocks.
package clock

//go:generate core generate

import (
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
)

type Clock struct {
	core.Tabs
}

func (cl *Clock) Init() {
	cl.Tabs.Init()
}

func (cl *Clock) OnAdd() {
	cl.Tabs.OnAdd()
	cl.clockTab()
	cl.timerTab()
	cl.stopwatchTab()
	cl.alarmTab()
}

func (cl *Clock) clockTab() {
	tab, _ := cl.NewTab("Clock")
	tab.Styler(func(s *styles.Style) {
		s.CenterAll()
	})
	date := core.NewText(tab).SetType(core.TextHeadlineMedium)
	date.Updater(func() {
		date.SetText(time.Now().Format("Monday, January 2"))
	})
	tm := core.NewText(tab).SetType(core.TextDisplayLarge)
	tm.Updater(func() {
		tm.SetText(time.Now().Format("3:04:05 PM"))
	})
	tm.Animate(func(a *core.Animation) {
		tm.UpdateRender()
	})
}

func (cl *Clock) stopwatchTab() {
	tab, _ := cl.NewTab("Stopwatches")
	core.NewButton(tab).SetText("Start")
}

func (cl *Clock) alarmTab() {
	tab, _ := cl.NewTab("Alarms")
	core.NewButton(tab).SetText("Create")
}
