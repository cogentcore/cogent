// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/math32"
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

// TestReadScrollTop tests that another message is shown from its top,
// not at the scroll position of the one before, and that updating the
// same message keeps its position.
func TestReadScrollTop(t *testing.T) {
	b, a := newTestApp(t, nil)
	require.NoError(t, os.MkdirAll(a.messagesDir(testEmail), 0o700))
	long := func(name string) *CacheMessage {
		msg := "From: a@example.com\r\nSubject: " + name + "\r\nContent-Type: text/plain\r\n\r\n" +
			strings.Repeat("A line of the message.\n\n", 200)
		require.NoError(t, os.WriteFile(filepath.Join(a.messagesDir(testEmail), name), []byte(msg), 0o600))
		return &CacheMessage{key: name, Filename: name, Envelope: imap.Envelope{Subject: name, Date: time.Now()},
			Labels: []Label{{Name: "INBOX", UID: 1}}, Flags: []imap.Flag{imap.FlagSeen}}
	}
	one, two := long("one"), long("two")
	a.cache[testEmail] = map[string]*CacheMessage{"one": one, "two": two}
	atStart := func() bool { return a.readPanel.IsDimAtContentStart(math32.Y) }
	core.RenderImage(b, core.Locked(b, func() {
		a.readMessage, a.readSingle = one, true
		a.updateReading()
	}), core.Locked(b, func() {
		require.True(t, a.readPanel.HasScroll[math32.Y], "long enough to scroll")
		a.readPanel.ScrollDimToContentEnd(math32.Y)
	}), core.Locked(b, func() {
		require.False(t, atStart(), "scrolled down")
		a.updateReading()
	}), core.Locked(b, func() {
		assert.False(t, atStart(), "the same message: kept")
		a.readMessage = two
		a.updateReading()
	}), core.Locked(b, func() {
		assert.True(t, atStart(), "another message: from its top")
	}))
}
