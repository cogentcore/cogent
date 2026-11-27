// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"slices"
	"testing"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"github.com/cogentcore/go-imap/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUntagged tests the Untagged mailbox of a Gmail account: the messages
// of All Mail without labels, as those archived, and those whose only label
// is deleted, which are otherwise not shown, and filing one out of it.
func TestUntagged(t *testing.T) {
	b, a := newTestingApp(t)
	you := Settings.Accounts[0].Email
	message := func(subject string) []byte {
		return []byte("From: zed@example.com\r\nTo: " + you + "\r\nSubject: " + subject + "\r\nContent-Type: text/plain\r\n\r\nHi.\r\n")
	}
	subjects := func() []string {
		var res []string
		for _, cm := range a.listCache {
			res = append(res, cm.Subject)
			for _, r := range cm.replies {
				res = append(res, r.Subject)
			}
		}
		return res
	}
	var all string
	core.RenderImage(b, func() {
		syncAll(t, a)
		a.AsyncLock()
		all = a.specialUse[you][imap.MailboxAttrAll]
		a.AsyncUnlock()
		require.NotEmpty(t, all)
		require.NoError(t, a.testServer.CreateMailbox(you, "Old"))
		require.NoError(t, a.testServer.Append(you, []string{all}, message("Archived"), []imap.Flag{imap.FlagSeen}, time.Now()))
		require.NoError(t, a.testServer.Append(you, []string{"Old"}, message("Only in Old"), nil, time.Now()))
		syncAll(t, a)
	}, core.Locked(b, func() {
		assert.Contains(t, a.mailboxLabels(you), untaggedLabel)
		assert.True(t, a.isSpecial(you, untaggedLabel), "not deletable")
		assert.False(t, a.isUserMailbox(you, untaggedLabel))
		a.showMailbox(you, untaggedLabel)
		assert.Contains(t, subjects(), "Archived")
		assert.NotContains(t, subjects(), "Only in Old", "in a label")
		for _, cm := range a.listCache {
			assert.False(t, slices.ContainsFunc(cm.Labels, func(l Label) bool { return l.Name == "INBOX" }), "not in the inbox: %s", cm.Subject)
		}
		a.DeleteMailbox(you, "Old")
	}), func() {
		require.Eventually(t, func() bool {
			a.AsyncLock()
			defer a.AsyncUnlock()
			return !slices.Contains(a.labels[you], "Old")
		}, 5*time.Second, 10*time.Millisecond, "deleted")
		syncAll(t, a)
	}, core.Locked(b, func() {
		a.Update()
		assert.Contains(t, subjects(), "Only in Old", "shown, not lost")
		assert.Equal(t, 1, a.unread[you][untaggedLabel], "its unread count")
		assert.GreaterOrEqual(t, a.totalMessages[you][untaggedLabel], 2)
		// filed out of it, into a label
		var cm *CacheMessage
		for _, m := range a.listCache {
			if m.Subject == "Only in Old" {
				cm = m
			}
		}
		require.NotNil(t, cm)
		a.fileTo(you, []*CacheMessage{cm}, untaggedLabel, "Family")
		assert.NotContains(t, subjects(), "Only in Old", "out of it right away")
	}))
}

// TestUntaggedNotSent tests that a message sent, in Sent, is not in
// Untagged, as it is shown in Sent.
func TestUntaggedNotSent(t *testing.T) {
	b, a := newTestingApp(t)
	you := Settings.Accounts[0].Email
	core.RenderImage(b, func() {
		syncAll(t, a)
		a.AsyncLock()
		sent := a.specialUse[you][imap.MailboxAttrSent]
		a.AsyncUnlock()
		require.NotEmpty(t, sent)
		msg := "From: " + you + "\r\nTo: zed@example.com\r\nSubject: Sent by me\r\nContent-Type: text/plain\r\n\r\nHi.\r\n"
		require.NoError(t, a.testServer.Append(you, []string{sent}, []byte(msg), []imap.Flag{imap.FlagSeen}, time.Now()))
		syncAll(t, a)
	}, core.Locked(b, func() {
		a.showMailbox(you, untaggedLabel)
		for _, cm := range a.listCache {
			assert.NotEqual(t, "Sent by me", cm.Subject, "in Sent")
		}
	}))
}

// TestUnreadOnly tests the filter of the list of messages that shows only
// the unread ones: the message being read stays shown once read, until
// another one is, and all are shown again without it.
func TestUnreadOnly(t *testing.T) {
	b, a := newTestingApp(t)
	you := Settings.Accounts[0].Email
	views := Settings.ListViews
	t.Cleanup(func() { Settings.ListViews = views })
	unread := func(cm *CacheMessage) bool { return !slices.Contains(cm.Flags, imap.FlagSeen) }
	var all int
	var first *CacheMessage
	core.RenderImage(b, func() { syncAll(t, a) }, core.Locked(b, func() {
		Settings.ListViews.SetFlag(false, ViewConversations)
		a.showMailbox(you, "INBOX")
		a.Update()
		all = len(a.listCache)
		a.UnreadOnly(true)
		require.NotEmpty(t, a.listCache, "unread messages")
		require.Less(t, len(a.listCache), all, "only those")
		for _, cm := range a.listCache {
			if cm != a.readMessage {
				assert.True(t, unread(cm), cm.Subject)
			}
		}
		// reading one: it stays until another one is read
		for i := 0; a.rowAt(i) != nil; i++ {
			if unread(a.rowAt(i).message) {
				first = a.rowAt(i).message
				a.list.SelectIndexEvent(i, events.SelectOne)
				break
			}
		}
		require.NotNil(t, first)
		a.Update()
		assert.Contains(t, a.listCache, first, "being read")
	}), func() {
		require.Eventually(t, func() bool {
			a.AsyncLock()
			defer a.AsyncUnlock()
			return !unread(first)
		}, 5*time.Second, 10*time.Millisecond, "read")
	}, core.Locked(b, func() {
		for i := 0; a.rowAt(i) != nil; i++ {
			if m := a.rowAt(i).message; m != first && unread(m) {
				a.list.SelectIndexEvent(i, events.SelectOne)
				break
			}
		}
		a.Update()
		assert.NotContains(t, a.listCache, first, "gone once another is read")
		a.UnreadOnly(false)
		assert.Len(t, a.listCache, all, "all again")
	}), core.Locked(b, func() {
		// the button, before the refresh button
		var filter *core.Button
		for w := range a.listPanel.AllWidgets() {
			if bt, ok := w.(*core.Button); ok && bt.Icon == icons.FilterList {
				filter = bt
			}
		}
		require.NotNil(t, filter)
		filter.Send(events.Click)
		assert.True(t, a.unreadOnly)
		filter.Send(events.Click)
		assert.False(t, a.unreadOnly)
	}))
}
