// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package clock

import (
	"time"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
)

// Timer is a timer that can be paused, resumed, reset, and removed.
type Timer struct {
	core.Frame

	// Duration is the total duration of the timer.
	Duration time.Duration

	// Start is when the timer was started.
	Start time.Time

	// Paused is whether the timer is currently paused.
	Paused bool
}

func (tm *Timer) Init() {
	tm.Frame.Init()
	tm.Styler(func(s *styles.Style) {
		s.CenterAll()
		s.Direction = styles.Column
		s.Background = colors.Scheme.SurfaceContainer
		s.Border.Radius = styles.BorderRadiusLarge
		s.Min.Set(units.Em(15))
		s.Padding.Set(units.Em(2))
	})

	tree.AddChild(tm, func(w *core.Text) {
		w.SetType(core.TextHeadlineLarge)
		w.Updater(func() {
			remaining := tm.Duration - time.Since(tm.Start)
			remaining = remaining.Round(time.Second)
			w.SetText(remaining.String())
		})
		w.Animate(func(a *core.Animation) {
			if tm.Paused {
				return
			}
			w.Update() // TODO: optimize?
		})
	})
	tree.AddChild(tm, func(w *core.Frame) {
		tree.AddChild(w, func(w *core.Button) {
			w.SetType(core.ButtonTonal)
			w.Updater(func() {
				if tm.Paused {
					w.SetIcon(icons.PlayArrowFill).SetTooltip("Resume")
				} else {
					w.SetIcon(icons.PauseFill).SetTooltip("Pause")
				}
			})
			w.Styler(func(s *styles.Style) {
				if tm.Paused {
					s.Color = colors.Scheme.Success.Base
					s.Background = colors.Scheme.Success.Container
				} else {
					s.Color = colors.Scheme.Warn.Base
					s.Background = colors.Scheme.Warn.Container
				}
			})
			w.OnClick(func(e events.Event) {
				tm.Paused = !tm.Paused
				w.Update()
			})
		})
	})
}

func (cl *Clock) timerTab() {
	tab, _ := cl.NewTab("Timers")
	trd := 15 * time.Minute
	core.Bind(&trd, core.NewDurationInput(tab))
	start := core.NewButton(tab).SetText("Start")
	start.OnClick(func(e events.Event) {
		NewTimer(tab).SetDuration(trd).SetStart(time.Now())
		tab.Update()
	})
}
