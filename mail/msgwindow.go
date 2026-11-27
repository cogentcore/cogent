// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"cmp"
	"slices"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/tree"
)

// openMessageWindow opens the messages of the given row of the list of
// messages in a window of their own, as when it is double-clicked, so that
// several can be seen at once: its message, or all of those of its
// conversation, the newest first. The window scrolls with the keys, and is
// updated with the app, as when the message is downloaded.
func (a *App) openMessageWindow(mr *messageRow) {
	msgs := rowMessages(mr)
	slices.SortFunc(msgs, func(x, y *CacheMessage) int {
		return cmp.Compare(y.Date.UnixNano(), x.Date.UnixNano())
	})
	title := mr.message.Subject
	if title == "" {
		title = "(no subject)"
	}
	// "View Message:" windows share their remembered size and position,
	// by the part of the title before the colon
	b := core.NewBody("View Message: " + title)
	b.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.SetAbilities(true, abilities.Focusable) // to scroll with the keys
	})
	for _, cm := range msgs {
		tree.AddChild(b, func(w *DisplayMessageFrame) {
			w.Updater(func() {
				w.SetMessage(cm)
			})
		})
	}
	a.messageWindows = append(a.messageWindows, b)
	b.OnClose(func(e events.Event) {
		inApp(b, func() {
			a.messageWindows = slices.DeleteFunc(a.messageWindows, func(w *core.Body) bool { return w == b })
		})
	})
	b.OnShow(func(e events.Event) {
		b.SetFocusQuiet()
	})
	b.RunWindowDialog(a)
}

// updateMessageWindows updates the windows of messages (see
// [App.openMessageWindow]) for the changes of their messages, as when
// they are downloaded. It is called with the lock of the main window, and
// updates them in the background, with their own locks, never holding
// both, which could deadlock with [inApp].
func (a *App) updateMessageWindows() {
	for _, b := range a.messageWindows {
		go func() {
			if b.AsyncLockIfShown() {
				b.Update()
				b.AsyncUnlock()
			}
		}()
	}
}

// inApp calls the given function with the lock of the main window, which
// the state of the app is used with: right away for a widget in the main
// window, which has it, or else in the background, as for a widget of a
// window of messages (see [App.openMessageWindow]).
func inApp(w core.Widget, f func()) {
	if w.AsWidget().Scene == theApp.Scene {
		f()
		return
	}
	go func() {
		theApp.AsyncLock()
		defer theApp.AsyncUnlock()
		f()
	}()
}
