// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"errors"
	"testing"

	"cogentcore.org/core/core"

	"github.com/cogentcore/go-imap/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestQuickSync tests the quick sync of a mailbox, as of the inbox and
// with Refresh: a message just received there is in the cache after it,
// and mailboxes not on the server, as the outbox, are not synced.
func TestQuickSync(t *testing.T) {
	c := newIMAP(t)
	b, a := newTestApp(t, c)
	core.RenderImage(b, func() {
		assert.ErrorIs(t, a.quickSync(testEmail, "INBOX"), errNotConnected, "before the first sync")
		require.NoError(t, a.cacheMessages())
		assert.Equal(t, "INBOX", a.serverMailbox(testEmail, "INBOX"))
		assert.Empty(t, a.serverMailbox(testEmail, outboxLabel))

		msg := "From: you@example.com\r\nTo: " + testEmail + "\r\nSubject: Your code\r\nMessage-ID: <code@example.com>\r\n\r\n123456\r\n"
		a.imapMu[testEmail].Lock()
		cmd := c.Append("INBOX", int64(len(msg)), nil)
		_, err := cmd.Write([]byte(msg))
		require.NoError(t, err)
		require.NoError(t, cmd.Close())
		_, err = cmd.Wait()
		a.imapMu[testEmail].Unlock()
		require.NoError(t, err)
		assert.Nil(t, messageBySubject(a, "Your code"))

		require.NoError(t, a.quickSync(testEmail, "INBOX"))
		cm := messageBySubject(a, "Your code")
		require.NotNil(t, cm)
		assert.Equal(t, "INBOX", cm.Labels[0].Name)
		assert.NotContains(t, a.status.text, "syncing", "quietly")
	})
}

// TestQuickSyncGmail tests the quick sync of the inbox of an account like
// those of Gmail, whose messages are synced from All Mail: a message just
// received is in the cache, in the inbox, after it.
func TestQuickSyncGmail(t *testing.T) {
	b, a := newTestingApp(t)
	core.RenderImage(b, func() {
		require.NoError(t, a.cacheMessages())
		email := a.currentEmail
		require.True(t, a.gmail[email])
		assert.Equal(t, a.specialUse[email][imap.MailboxAttrAll], a.serverMailbox(email, "INBOX"))
		assert.Equal(t, a.trash[email], a.serverMailbox(email, a.trash[email]))
		inbox := func() int {
			a.AsyncLock()
			defer a.AsyncUnlock()
			n := 0
			for _, cm := range a.cache[email] {
				for _, l := range cm.Labels {
					if l.Name == "INBOX" {
						n++
					}
				}
			}
			return n
		}
		n := inbox()
		require.NoError(t, a.ReceiveTestMessage())
		require.NoError(t, a.quickSync(email, "INBOX"))
		assert.Equal(t, n+1, inbox())
	})
}

// TestQuickSyncConcurrent tests quick syncs of the inbox along with the
// regular sync, as [App.quickSyncLoop] runs them (run with -race).
func TestQuickSyncConcurrent(t *testing.T) {
	b, a := newTestingApp(t)
	core.RenderImage(b, func() {
		require.NoError(t, a.cacheMessages())
		email := a.currentEmail
		done := make(chan struct{})
		go func() {
			defer close(done)
			for range 5 {
				assert.NoError(t, a.ReceiveTestMessage())
				assert.NoError(t, a.quickSync(email, "INBOX"))
			}
		}()
		// the inboxes of all of the accounts, while the regular syncs sync
		// the other ones, each with only the IMAP mutex of its account (as
		// the selected mailboxes of all of them were in one map without a
		// mutex of its own: a fatal concurrent map write)
		others := make(chan struct{})
		go func() {
			defer close(others)
			for range 10 {
				for _, e := range Settings.Emails() {
					if err := a.quickSync(e, "INBOX"); err != nil && !errors.Is(err, errNotConnected) {
						assert.NoError(t, err)
					}
				}
			}
		}()
		for range 3 {
			require.NoError(t, a.cacheMessages())
		}
		<-done
		<-others
		require.NoError(t, a.cacheMessages())
	})
}
