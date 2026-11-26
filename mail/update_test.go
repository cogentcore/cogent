// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"cogentcore.org/core/core"
	"github.com/cogentcore/go-imap/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSyncUpdate tests that the updates of the app for the changes from
// the sync are combined, at most one per interval.
func TestSyncUpdate(t *testing.T) {
	old := syncUpdateInterval
	syncUpdateInterval = 100 * time.Millisecond
	t.Cleanup(func() { syncUpdateInterval = old })
	b, a := newTestApp(t, nil)
	var n atomic.Int32
	core.RenderImage(b, core.Locked(b, func() {
		a.listPanel.Updater(func() { n.Add(1) }) // the list, not the message being read
	}), func() {
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

// TestMarkReadRightAway tests that a message is shown as read as soon as
// it is selected, before the server is changed, and that the update for
// it does not lay out the message being read again.
func TestMarkReadRightAway(t *testing.T) {
	b, a := newTestApp(t, newIMAP(t))
	var reads atomic.Int32
	var hello *CacheMessage
	core.RenderImage(b, func() {
		require.NoError(t, a.cacheMessages())
	}, core.Locked(b, func() {
		a.readPanel.Updater(func() { reads.Add(1) })
		a.showMailbox(testEmail, "Work")
		hello = messageBySubject(a, "Hello")
		require.False(t, hello.isRead())
		var row *messageRow
		for _, mr := range a.listRows {
			if mr.message == hello {
				row = mr
			}
		}
		require.NotNil(t, row)
		require.True(t, row.Message.Unread)
		a.readMessage, a.readSingle = hello, true
		n := reads.Load()
		a.MarkAsRead()
		assert.True(t, slices.Contains(hello.Flags, imap.FlagSeen), "right away, with the lock still held")
		assert.False(t, row.Message.Unread, "not bold in the list")
		assert.Equal(t, n, reads.Load(), "the message being read not updated")
	}), func() {
		require.Eventually(t, func() bool {
			return slices.Contains(serverFlags(t, a, "Work", hello.Labels[len(hello.Labels)-1].UID), imap.FlagSeen)
		}, 5*time.Second, 10*time.Millisecond, "on the server")
	})
}
