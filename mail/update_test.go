// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"sync/atomic"
	"testing"
	"time"

	"cogentcore.org/core/core"
	"github.com/stretchr/testify/assert"
)

// TestSyncUpdate tests that the updates of the app for the changes from
// the sync are combined, at most one per interval.
func TestSyncUpdate(t *testing.T) {
	old := syncUpdateInterval
	syncUpdateInterval = 100 * time.Millisecond
	t.Cleanup(func() { syncUpdateInterval = old })
	b, a := newTestApp(t, nil)
	var n atomic.Int32
	a.Updater(func() { n.Add(1) })
	core.RenderImage(b, func() {
		start := n.Load()
		for range 20 {
			a.syncUpdate()
		}
		assert.Equal(t, start, n.Load(), "not right away")
		assert.Eventually(t, func() bool { return n.Load() == start+1 }, 2*time.Second, 10*time.Millisecond)
		time.Sleep(3 * syncUpdateInterval)
		assert.Equal(t, start+1, n.Load(), "once for all of them")
	})
}

// TestShownMailbox tests that the mailbox shown last is shown when the
// app starts again.
func TestShownMailbox(t *testing.T) {
	b, a := newTestApp(t, nil)
	core.RenderImage(b, core.Locked(b, func() {
		a.showMailbox(testEmail, "Work")
	}))
	assert.Equal(t, testEmail, Settings.ShownAccount)
	assert.Equal(t, "Work", Settings.ShownMailbox)
	a2 := NewApp(core.NewBody())
	theApp = a
	assert.Equal(t, testEmail, a2.currentEmail)
	assert.Equal(t, "Work", a2.showLabel)

	Settings.ShownAccount = "gone@example.com" // an account that was removed
	a3 := NewApp(core.NewBody())
	theApp = a
	assert.Equal(t, "INBOX", a3.showLabel)
	Settings.ShownAccount, Settings.ShownMailbox = "", ""
}
