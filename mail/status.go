// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"fmt"
	"time"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
)

// syncStatus is the status of the syncing of the mail with the servers,
// shown below the mailboxes.
type syncStatus struct {

	// text says what is being done, or is "" between syncs.
	text string

	// progress is how much of it is done, from 0 to 1, or -1 if unknown.
	progress float32

	// err is the error of the last sync, if it failed.
	err error

	// synced is when the last sync finished.
	synced time.Time
}

// setStatus sets the status of the syncing to the given text, saying what
// is being done, and progress from 0 to 1 (or -1 if unknown).
func (a *App) setStatus(text string, progress float32) {
	a.AsyncLock()
	defer a.AsyncUnlock()
	a.status.text, a.status.progress = text, progress
	if a.statusBar != nil {
		a.statusBar.Update()
	}
}

// syncDone sets the status for the end of a sync with the given error.
func (a *App) syncDone(err error) {
	a.AsyncLock()
	defer a.AsyncUnlock()
	a.status.text, a.status.progress = "", -1
	a.status.err = err
	if err == nil {
		a.status.synced = time.Now()
	}
	if a.statusBar != nil {
		a.statusBar.Update()
	}
}

// progressStatus sets the status to the given action of the given account
// on n of the total, with its progress, every step of n and at the end.
func (a *App) progressStatus(email, action string, n, total, step int) {
	if total <= 0 || (n%step != 0 && n != total) {
		return
	}
	a.setStatus(fmt.Sprintf("%s: %s, %s of %s", email, action, thousands(n), thousands(total)), float32(n)/float32(total))
}

// thousands returns the given number with commas between the thousands.
func thousands(n int) string {
	s := fmt.Sprint(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

// initStatusBar initializes the bar below the mailboxes that shows the
// status of the syncing: what is being done, with a meter of its progress,
// or the error of the last sync, which can be clicked for all of it, or
// when it was last synced.
func (a *App) initStatusBar(w *core.Frame) {
	a.statusBar = w
	w.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 0)
		s.Gap.Set(units.Dp(2))
		s.Padding.Set(units.Dp(4), units.Dp(8))
	})
	w.Maker(func(p *tree.Plan) {
		st := a.status
		if st.err != nil && st.text == "" {
			tree.Add(p, func(w *core.Button) {
				w.SetType(core.ButtonText).SetIcon(icons.Error)
				w.Styler(func(s *styles.Style) {
					s.Color = colors.Scheme.Error.Base
				})
				w.Updater(func() {
					w.SetText("Not synced: " + firstLine(a.status.err.Error()))
					w.SetTooltip(a.status.err.Error())
				})
				w.OnClick(func(e events.Event) {
					if err := a.status.err; err != nil {
						core.ErrorDialog(a, err, "Error syncing")
					}
				})
			})
			return
		}
		tree.Add(p, func(w *core.Text) {
			w.SetType(core.TextSupporting)
			w.Updater(func() {
				switch {
				case a.status.text != "":
					w.SetText(a.status.text)
				case !a.status.synced.IsZero():
					w.SetText("Updated " + a.status.synced.Format(time.Kitchen))
				default:
					w.SetText("")
				}
			})
		})
		if st.text != "" && st.progress >= 0 {
			tree.Add(p, func(w *core.Meter) {
				w.Styler(func(s *styles.Style) {
					s.Min.Set(units.Em(8), units.Dp(4))
					s.Grow.Set(1, 0)
				})
				w.Updater(func() {
					w.SetValue(a.status.progress)
				})
			})
		}
	})
}

// firstLine returns the first line of the given text, cut short if long.
func firstLine(s string) string {
	for i, r := range s {
		if r == '\n' || i > 80 {
			return s[:i] + "..."
		}
	}
	return s
}
