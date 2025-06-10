// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package clock

import (
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
)

// Timer is a timer that can be paused, resumed, reset, and removed.
type Timer struct {
	core.Frame

	// Duration is the total duration of the timer.
	Duration time.Duration

	// Remaining is the remaining time on the timer.
	Remaining time.Duration `set:"-"`
}

func (tm *Timer) Init() {
	tm.Frame.Init()
	tm.Styler(func(s *styles.Style) {
		s.CenterAll()
		s.Direction = styles.Column
	})
}
