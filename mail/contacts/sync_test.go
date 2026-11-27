// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package contacts

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"cogentcore.org/cogent/mail/contacts/carddavtest"
	"github.com/emersion/go-vcard"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// names returns the names of the contacts of the given store.
func names(s *Store) []string {
	var res []string
	for _, c := range s.Contacts() {
		res = append(res, c.Name)
	}
	return res
}

// TestSync tests syncing the contacts with an address book on a CardDAV
// server, both ways.
func TestSync(t *testing.T) {
	srv := carddavtest.New()
	t.Cleanup(srv.Close)
	srv.Put("a", "Alice Adams", "alice@x.test")
	srv.Put("b", "Bob Brown", "bob@x.test")
	srv.Put("c", "Carol Cole", "carol@x.test")

	s, err := Open(t.TempDir())
	require.NoError(t, err)
	acct, err := NewAccount("me@x.test", http.DefaultClient, srv.URL, "") // the address book is found
	require.NoError(t, err)
	changes := 0
	s.OnChange(func() { changes++ })
	ctx := context.Background()
	sync := func() SyncResult {
		t.Helper()
		res, err := s.Sync(ctx, acct)
		require.NoError(t, err)
		return res
	}

	res := sync()
	assert.Equal(t, SyncResult{Added: 3}, res, "from the server")
	assert.Equal(t, []string{"Alice Adams", "Bob Brown", "Carol Cole"}, names(s))
	assert.Equal(t, 1, changes, "notified")
	assert.Equal(t, SyncResult{}, sync(), "nothing new")
	assert.Equal(t, 1, changes, "not notified without changes")

	// changed here
	alice := s.byUID("a")
	alice.Notes = "Neighbor"
	require.NoError(t, s.Save(alice))
	assert.Equal(t, SyncResult{Uploaded: 1}, sync())
	assert.Equal(t, "Neighbor", srv.Card("a").Value(vcard.FieldNote), "on the server")
	assert.Equal(t, SyncResult{}, sync(), "nothing new after uploading")

	// changed on the server, shown in a browser of the contacts
	br := &Browser{}
	br.SetStore(s)
	br.Select(s.byUID("b"))
	srv.Put("b", "Bob Browning", "bob@x.test")
	assert.Equal(t, SyncResult{Changed: 1}, sync())
	assert.Equal(t, "Bob Browning", s.byUID("b").Name)
	br.updateShown() // as in an update of the browser
	require.NotNil(t, br.Current())
	assert.Equal(t, "Bob Browning", br.Current().Name, "the new one shown")

	// changed on both: the server wins
	carol := s.byUID("c")
	carol.Notes = "here"
	require.NoError(t, s.Save(carol))
	srv.Put("c", "Carol Coleman", "carol@x.test")
	assert.Equal(t, SyncResult{Changed: 1}, sync())
	assert.Equal(t, "Carol Coleman", s.byUID("c").Name)
	assert.Empty(t, s.byUID("c").Notes)

	// deleted here
	require.NoError(t, s.Delete(s.byUID("a")))
	assert.Equal(t, SyncResult{Removed: 1}, sync())
	assert.Nil(t, srv.Card("a"), "deleted on the server")

	// deleted on the server
	srv.Delete("b")
	assert.Equal(t, SyncResult{Deleted: 1}, sync())
	assert.Nil(t, s.byUID("b"))

	// made here: added to the account of new contacts, but not imported ones
	s.NewAccount = acct.Name
	c := s.New()
	c.Name = "Dan Dee"
	c.Emails = []Labeled{{Value: "dan@x.test"}}
	require.NoError(t, s.Save(c))
	_, err = s.Import(strings.NewReader("BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Imported Ivy\r\nN:Ivy;Imported;;;\r\nUID:ivy\r\nEND:VCARD\r\n"))
	require.NoError(t, err)
	assert.Equal(t, SyncResult{Uploaded: 1}, sync())
	card := srv.Card(c.UID())
	require.NotNil(t, card, "on the server")
	assert.Equal(t, "Dan Dee", card.Value(vcard.FieldFormattedName))
	assert.Nil(t, srv.Card("ivy"), "imported: only here")
	assert.Equal(t, SyncResult{}, sync())

	// the state is kept: a new store of the same contacts has nothing new
	s2, err := Open(s.Dir)
	require.NoError(t, err)
	res, err = s2.Sync(ctx, acct)
	require.NoError(t, err)
	assert.Equal(t, SyncResult{}, res)
}

// TestSyncError tests that the error of a sync has the message of the
// error response of the server, as Google says what is wrong in it.
func TestSyncError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error": {"code": 403, "message": "Google Contacts CardDAV API has not been used in project 1 before or it is disabled.", "status": "PERMISSION_DENIED"}}`))
	}))
	t.Cleanup(srv.Close)
	s, err := Open(t.TempDir())
	require.NoError(t, err)
	acct, err := NewAccount("me@x.test", http.DefaultClient, srv.URL, "/book/")
	require.NoError(t, err)
	_, err = s.Sync(context.Background(), acct)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "403")
	assert.Contains(t, err.Error(), "CardDAV API has not been used in project 1 before or it is disabled", "the message of the server")
}
