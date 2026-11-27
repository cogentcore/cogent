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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAllInboxes tests showing the inboxes of all of the accounts
// together: the messages of each, reading one setting its account as the
// current one, deleting messages of two accounts at once into the trash of
// each, searching all of the accounts, and the node of the inboxes.
func TestAllInboxes(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	old := *Settings
	t.Cleanup(func() { *Settings = old })
	t.Cleanup(core.TrackWindows())
	b := core.NewBody()
	a := NewApp(b)
	require.NoError(t, a.StartTesting(TestingOptions{Seed: 1}))
	t.Cleanup(a.testServer.Close)
	t.Cleanup(a.closeStores)
	you, alice := Settings.Accounts[0].Email, Settings.Accounts[1].Email
	syncAll := func() {
		for {
			a.syncMore = false
			require.NoError(t, a.cacheMessages())
			if !a.syncMore {
				return
			}
		}
	}
	inInbox := func(cm *CacheMessage) bool {
		return slices.ContainsFunc(cm.Labels, func(l Label) bool { return l.Name == "INBOX" })
	}
	var deleted []*CacheMessage
	core.RenderImage(b, syncAll, core.Locked(b, func() {
		a.showMailbox(you, "INBOX")
		mine := len(a.listCache)
		a.ShowAllInboxes()
		assert.True(t, a.allInboxes)
		assert.Equal(t, allInboxesShown, Settings.ShownAccount, "shown when the app starts again")
		accounts := map[string]int{}
		for _, cm := range a.listCache {
			accounts[cm.account]++
		}
		assert.Equal(t, mine, accounts[you], "all of the inbox of each account")
		require.Greater(t, accounts[alice], 0)
		for i, r := range a.listRows {
			if i > 0 {
				assert.GreaterOrEqual(t, a.listRows[i-1].Date, r.Date, "the newest first")
			}
		}

		// the Inboxes node is the one selected
		var inboxes *MailboxTree
		for _, n := range a.mailboxes.Children {
			if mt, ok := n.(*MailboxTree); ok && mt.inboxes {
				inboxes = mt
			}
		}
		require.NotNil(t, inboxes)
		a.mailboxes.Update()
		sel := a.mailboxes.GetSelectedNodes()
		require.Len(t, sel, 1)
		assert.Equal(t, inboxes.This, sel[0])

		// reading a message makes its account the current one, and a
		// message of each account is deleted into its own trash
		var rows []int
		for _, email := range []string{alice, you} {
			for i, r := range a.listRows {
				if r.message.account == email {
					a.list.SelectIndexEvent(i, events.SelectOne)
					assert.Equal(t, email, a.currentEmail, "the account of the message read")
					rows = append(rows, i)
					deleted = append(deleted, r.message)
					break
				}
			}
		}
		require.Len(t, rows, 2)
		a.list.SelectIndexEvent(rows[0], events.SelectOne)
		a.list.SelectIndexEvent(rows[1], events.ExtendOne)
		require.Len(t, a.selectedRows, 2)
		a.Delete()
	}), func() {
		require.Eventually(t, func() bool {
			a.AsyncLock()
			defer a.AsyncUnlock()
			return !inInbox(deleted[0]) && !inInbox(deleted[1])
		}, 5*time.Second, 10*time.Millisecond, "out of the inboxes")
		syncAll()
		a.AsyncLock()
		for i, email := range []string{alice, you} {
			trash := a.trash[email]
			found := false
			for _, cm := range a.cache[email] {
				if cm.Subject == deleted[i].Subject && slices.ContainsFunc(cm.Labels, func(l Label) bool { return l.Name == trash }) {
					found = true
				}
			}
			assert.True(t, found, "in the trash of %s: %q", email, deleted[i].Subject)
		}
		a.AsyncUnlock()
	}, core.Locked(b, func() {
		a.SearchMessages("alpha")
		accounts := map[string]bool{}
		for _, cm := range a.listCache {
			accounts[cm.account] = true
		}
		assert.True(t, accounts[you] && accounts[alice], "found in all of the accounts: %v", accounts)
		a.showMailbox(alice, "INBOX")
		assert.False(t, a.allInboxes, "an inbox of one account")
		assert.Nil(t, a.searchResults)
		for _, cm := range a.listCache {
			assert.Equal(t, alice, cm.account)
		}
	}))
}
