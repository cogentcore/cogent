// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"testing"

	"cogentcore.org/cogent/mail/contacts"
	"cogentcore.org/cogent/mail/contacts/carddavtest"
	"cogentcore.org/core/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSyncContacts tests syncing the contacts of an account in the syncs
// of its mail, which completion then offers, and adding the contacts made
// here to it.
func TestSyncContacts(t *testing.T) {
	srv := carddavtest.New()
	t.Cleanup(srv.Close)
	srv.Put("zoe", "Zoe Zimmer", "zoe@contacts.test")
	b, a := newTestApp(t, newIMAP(t))
	a.contactsServer = srv.URL
	old := Settings.SyncContacts
	t.Cleanup(func() { Settings.SyncContacts = old })
	Settings.SyncContacts = map[string]bool{testEmail: true}
	assert.Contains(t, authScopes(testEmail), contacts.GoogleScope, "the access to the contacts")
	assert.NotContains(t, authScopes("other@example.com"), contacts.GoogleScope)
	core.RenderImage(b, func() {
		require.NoError(t, a.cacheMessages())
	}, core.Locked(b, func() {
		s := a.contactsStore()
		require.NotNil(t, s)
		assert.Equal(t, testEmail, s.NewAccount, "made here: to the account")
		oldCA := Settings.ContactsAccount
		t.Cleanup(func() { Settings.ContactsAccount = oldCA })
		Settings.ContactsAccount = "other@example.com"
		assert.Equal(t, testEmail, newContactsAccount(), "not an account whose contacts are not synced")
		Settings.SyncContacts["other@example.com"] = true
		Settings.Apply()
		assert.Equal(t, "other@example.com", s.NewAccount, "the one chosen in the settings")
		delete(Settings.SyncContacts, "other@example.com")
		Settings.ContactsAccount = ""
		Settings.Apply()
		assert.Equal(t, testEmail, s.NewAccount, "by default, the first")
		require.Len(t, s.Contacts(), 1, "from the server")
		md := a.addressMatches("zo")
		require.Len(t, md.Matches, 1)
		assert.Equal(t, "Zoe Zimmer <zoe@contacts.test>", md.Matches[0].Text)

		c := s.New()
		c.Name = "Yan Young"
		c.Emails = []contacts.Labeled{{Value: "yan@contacts.test"}}
		require.NoError(t, s.Save(c))
		a.contactsSynced[testEmail] = a.contactsSynced[testEmail].AddDate(0, 0, -1) // time for the next one (with the lock)
	}), func() {
		require.NoError(t, a.cacheMessages())
		assert.Equal(t, []string{"Yan Young", "Zoe Zimmer"}, srv.Names(), "added to the account")
		require.NoError(t, a.cacheMessages())
		assert.Len(t, srv.Names(), 2, "not again, nor before the interval")
	})
}
