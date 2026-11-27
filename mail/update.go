// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"time"
)

// syncUpdateInterval is the minimum time between the updates of the app
// for the changes from the sync (see [App.syncUpdate]). If it is zero, as
// in tests, the app is updated right away.
var syncUpdateInterval = time.Second

// syncUpdate updates the app for changes from the sync, such as new
// messages, at most once every [syncUpdateInterval], so that the sync
// does not make the app slow: updating the lists of messages and the
// counts of unread messages takes a while with many messages, under the
// lock of the window. It must be called without the lock of the window.
func (a *App) syncUpdate() {
	if syncUpdateInterval == 0 {
		a.AsyncLock()
		a.updateMail()
		a.AsyncUnlock()
		return
	}
	a.syncUpdateMu.Lock()
	defer a.syncUpdateMu.Unlock()
	if a.syncUpdatePending {
		return // the pending one includes these changes
	}
	a.syncUpdatePending = true
	time.AfterFunc(syncUpdateInterval, func() {
		a.syncUpdateMu.Lock()
		a.syncUpdatePending = false
		a.syncUpdateMu.Unlock()
		a.AsyncLock()
		a.updateMail()
		a.AsyncUnlock()
	})
}

// updateReading updates the panel of the message being read and the
// toolbar, as when the message being read changes, without the rest of
// the app, which takes a while with many messages.
func (a *App) updateReading() {
	if a.readPanel == nil {
		a.Update()
		return
	}
	a.readPanel.Update()
	if a.toolbar != nil {
		a.toolbar.Update()
	}
}

// updateMail updates the mailboxes, the list of messages, the status, and
// the toolbar, as after the messages or their flags change, without the
// panel of the message being read, unless it is gone, as laying out a large
// message again takes a while.
func (a *App) updateMail() {
	if a.listPanel == nil || a.sidebar == nil {
		a.Update()
		return
	}
	a.sidebar.Update()
	a.listPanel.Update()
	a.updateMessageWindows()
	if a.toolbar != nil {
		a.toolbar.Update()
	}
	if cm := a.readMessage; cm != nil && a.cache[a.currentEmail][cm.key] != cm {
		a.readMessage = nil // gone, as deleted on the server
		a.readPanel.Update()
	}
}
