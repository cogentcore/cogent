// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package clock

import (
	"testing"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// timers returns the timers of the given clock.
func timers(cl *Clock) []*Timer {
	var res []*Timer
	for w := range cl.AllWidgets() {
		if tm, ok := w.(*Timer); ok {
			res = append(res, tm)
		}
	}
	return res
}

// TestClock tests the tabs of the clock, and starting, pausing, resuming,
// and resetting a timer, which pauses itself when it is done.
func TestClock(t *testing.T) {
	b := core.NewBody()
	cl := NewClock(b)
	var tm *Timer
	core.RenderImage(b, core.Locked(b, func() {
		names := []string{}
		for i := range cl.NumTabs() {
			_, tb := cl.TabAtIndex(i)
			names = append(names, tb.Text)
		}
		assert.Equal(t, []string{"Clock", "Timers", "Stopwatches", "Alarms"}, names)
		cl.SelectTabByName("Timers")
	}), core.Locked(b, func() {
		bt := core.ButtonByText(cl, "Start")
		require.NotNil(t, bt)
		bt.Send(events.Click)
		tms := timers(cl)
		require.Len(t, tms, 1)
		tm = tms[0]
		assert.Equal(t, 15*time.Minute, tm.Duration)
		assert.False(t, tm.Paused)

		tm.pauseButton.Send(events.Click)
		assert.True(t, tm.Paused)
		assert.Equal(t, "Resume", tm.pauseButton.Tooltip)
		tm.pauseButton.Send(events.Click)
		assert.False(t, tm.Paused)
		assert.Greater(t, tm.durationPaused, time.Duration(0))

		tm.SetDuration(time.Millisecond)
		tm.Update()
	}), func() {
		require.Eventually(t, func() bool {
			b.AsyncLock()
			defer b.AsyncUnlock()
			return tm.done
		}, 5*time.Second, 10*time.Millisecond, "the timer is done")
	}, core.Locked(b, func() {
		assert.True(t, tm.Paused, "it pauses when done")
		assert.True(t, tm.pauseButton.IsDisabled())
		var reset *core.Button
		for w := range tm.AllWidgets() {
			if bt, ok := w.(*core.Button); ok && bt.Tooltip == "Reset" {
				reset = bt
			}
		}
		require.NotNil(t, reset)
		tm.SetDuration(time.Minute)
		reset.Send(events.Click)
		assert.False(t, tm.done)
		assert.Equal(t, time.Duration(0), tm.durationPaused)
	}))
}
