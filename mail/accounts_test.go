// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/base/iox/tomlx"
	"cogentcore.org/core/core"
	"github.com/cogentcore/go-imap/v2"
	"github.com/cogentcore/go-imap/v2/imapclient"
	"github.com/cogentcore/go-imap/v2/imapserver"
	"github.com/cogentcore/go-imap/v2/imapserver/imapmemserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zalando/go-keyring"
)

// newPasswordIMAP starts an in-memory IMAP server like that of iCloud: its
// mailboxes have names, not special-use attributes, and it signs in with a
// password, for the given username. It returns its address.
func newPasswordIMAP(t *testing.T, username, password string) string {
	t.Helper()
	mem := imapmemserver.New()
	user := imapmemserver.NewUser(username, password)
	for _, mb := range []string{"INBOX", "Sent Messages", "Deleted Messages", "Junk", "Drafts"} {
		require.NoError(t, user.Create(mb, nil))
	}
	mem.AddUser(user)
	srv := imapserver.New(&imapserver.Options{
		NewSession: func(c *imapserver.Conn) (imapserver.Session, *imapserver.GreetingData, error) {
			return mem.NewSession(), nil, nil
		},
		Caps:         imap.CapSet{imap.CapIMAP4rev1: {}, imap.CapIMAP4rev2: {}},
		InsecureAuth: true,
	})
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	go srv.Serve(ln)
	t.Cleanup(func() { srv.Close() })
	return ln.Addr().String()
}

// gmailAccounts returns Gmail accounts of the given email addresses.
func gmailAccounts(emails ...string) []Account {
	var res []Account
	for _, e := range emails {
		res = append(res, Account{Type: Gmail, Email: e})
	}
	return res
}

// newAccountsApp returns a new body with an [App] without accounts, with
// temporary settings and data.
func newAccountsApp(t *testing.T) (*core.Body, *App) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	oldFile, oldAccounts := Settings.File, Settings.Accounts
	t.Cleanup(func() { Settings.File, Settings.Accounts = oldFile, oldAccounts })
	Settings.File = filepath.Join(t.TempDir(), "settings.toml")
	Settings.Accounts = nil
	Settings.ShownAccount, Settings.ShownMailbox = "", ""
	t.Cleanup(core.TrackWindows())
	b := core.NewBody()
	a := NewApp(b)
	a.dataDirectory = t.TempDir()
	t.Cleanup(a.closeStores)
	return b, a
}

// TestPasswordAccount tests adding an account that signs in with a
// password, as of iCloud: it signs in with its password, which is in the
// keychain, to servers of its own, whose mailboxes are known by their
// names, and a message sent from it is saved in its Sent mailbox, once.
func TestPasswordAccount(t *testing.T) {
	const email, password = "me@example.org", "app-specific"
	addr := newPasswordIMAP(t, email, password)
	b, a := newAccountsApp(t)
	na := &newAccount{Account: Account{Type: IMAP, Email: email, IMAP: addr, SMTP: "127.0.0.1:1", Insecure: true}, Password: "wrong"}
	require.Error(t, a.signInNew(context.Background(), na), "a wrong password")
	na.Password = password
	require.NoError(t, a.signInNew(context.Background(), na))
	assert.Empty(t, na.Username, "the email address")

	core.RenderImage(b, core.Locked(b, func() {
		require.NoError(t, a.addPasswordAccount(na))
	}), func() {
		assert.Equal(t, []string{email}, Settings.Emails())
		assert.True(t, usesPassword(email))
		assert.False(t, a.usesOAuth(email))
		p, err := keyring.Get(passwordService, email)
		require.NoError(t, err)
		assert.Equal(t, password, p, "in the keychain")
		assert.Equal(t, Account{Type: IMAP, Email: email, IMAP: addr, SMTP: "127.0.0.1:1", Insecure: true}, *Settings.account(email), "without the password")

		a.passwords = map[string]string{} // read from the keychain again
		require.NoError(t, a.cacheMessages())
		su := a.specialUse[email]
		assert.Equal(t, "Sent Messages", su[imap.MailboxAttrSent], "by its name")
		assert.Equal(t, "Junk", su[imap.MailboxAttrJunk])
		assert.Equal(t, "Deleted Messages", a.trash[email])
		assert.Equal(t, "Sent", friendlyLabelName("Sent Messages"))
		assert.Equal(t, "Trash", friendlyLabelName("Deleted Messages"))

		sent := []byte("From: me@example.org\r\nTo: you@example.com\r\nSubject: Hi\r\nMessage-ID: <1@example.org>\r\n\r\nHello\r\n")
		a.saveSent(email, sent, "1@example.org")
		a.saveSent(email, sent, "1@example.org") // not again
		c, err := imapclient.DialInsecure(addr, nil)
		require.NoError(t, err)
		defer c.Close()
		require.NoError(t, c.Login(email, password).Wait())
		sd, err := c.Select("Sent Messages", nil).Wait()
		require.NoError(t, err)
		assert.Equal(t, uint32(1), sd.NumMessages, "saved in Sent, once")
	}, core.Locked(b, func() { // removed, with its password
		a.RemoveAccount(email)
	}), func() {
		assert.False(t, usesPassword(email))
		assert.Empty(t, Settings.Accounts)
		_, err := keyring.Get(passwordService, email)
		assert.ErrorIs(t, err, keyring.ErrNotFound)
	})
}

// TestPasswordAccountUsername tests that an account whose server signs in
// with the name part of its email address (as iCloud can) is added with
// that username.
func TestPasswordAccountUsername(t *testing.T) {
	addr := newPasswordIMAP(t, "me", "secret")
	_, a := newAccountsApp(t)
	na := &newAccount{Account: Account{Type: IMAP, Email: "me@example.org", IMAP: addr, SMTP: "127.0.0.1:1", Insecure: true}, Password: "secret"}
	require.NoError(t, a.checkPassword(na))
	assert.Equal(t, "me", na.Username)
}

// TestAccountTypes tests the types of the accounts: the servers of iCloud,
// the fields shown for each type, and the accounts of earlier versions,
// which were all Gmail.
func TestAccountTypes(t *testing.T) {
	for _, email := range []string{"me@icloud.com", "Me@ME.com", "old@mac.com"} {
		assert.Equal(t, ICloud, typeOf(email), email)
	}
	assert.Equal(t, Gmail, typeOf("me@gmail.com"))
	ic := &Account{Type: ICloud, Email: "me@icloud.com"}
	assert.Equal(t, "imap.mail.me.com:993", ic.server().IMAP)
	assert.Equal(t, "smtp.mail.me.com:587", ic.server().SMTP)
	assert.False(t, ic.ShouldDisplay("IMAP"), "the servers of iCloud")
	assert.True(t, (&Account{Type: IMAP}).ShouldDisplay("SMTP"))
	assert.False(t, (&newAccount{}).ShouldDisplay("Password"), "Gmail: in the browser")
	assert.True(t, (&newAccount{Account: *ic}).ShouldDisplay("Password"))

	sd := &SettingsData{}
	require.NoError(t, tomlx.ReadBytes(sd, []byte("Accounts = [\"a@gmail.com\", \"b@gmail.com\"]\n")))
	assert.Equal(t, gmailAccounts("a@gmail.com", "b@gmail.com"), sd.Accounts)

	sd = &SettingsData{} // as saved since, with their types
	require.NoError(t, tomlx.ReadBytes(sd, []byte("[[Accounts]]\n  Type = 'Microsoft'\n  Email = 'me@ru.nl'\n\n[[Accounts]]\n  Type = 'Gmail'\n  Email = 'me@gmail.com'\n")))
	assert.Equal(t, []Account{{Type: Microsoft, Email: "me@ru.nl"}, gmailAccounts("me@gmail.com")[0]}, sd.Accounts)
	saved, err := tomlx.WriteBytes(sd)
	require.NoError(t, err)
	assert.Contains(t, string(saved), "[[Accounts]]", "saved as tables, under the same name")
}

// TestAccountDialog tests the dialog of adding an account: an address of
// iCloud makes it an account of iCloud, which asks for its password.
func TestAccountDialog(t *testing.T) {
	b, a := newAccountsApp(t)
	na := &newAccount{Account: Account{Type: Gmail}}
	img := core.RenderImage(b, core.Locked(b, func() {
		a.accountDialog(na)
	}), core.Locked(b, func() {
		d := core.TopScene(b)
		require.NotEqual(t, b.Scene, d)
		var form *core.Form
		for w := range d.AllWidgets() {
			if f, ok := w.(*core.Form); ok {
				form = f
			}
		}
		require.NotNil(t, form)
		na.Email = "me@icloud.com"
		form.SendChange()
	}), core.Locked(b, func() {
		assert.Equal(t, ICloud, na.Type)
		d := core.TopScene(b)
		labels := []string{}
		for w := range d.AllWidgets() {
			if tx, ok := w.(*core.Text); ok {
				labels = append(labels, tx.Text)
			}
		}
		assert.Contains(t, labels, "Password")
		assert.Contains(t, labels, "iCloud", "its type, as named")
		assert.NotContains(t, labels, "SMTP", "the servers of iCloud")
	}))
	if dir := os.Getenv("ACCOUNT_DIALOG_PNG"); dir != "" { // to see it
		require.NoError(t, imagex.Save(img, filepath.Join(dir, "account-dialog.png")))
	}
}

// TestMicrosoftAccount tests accounts of Microsoft: they sign in in the
// browser (no password), to the servers of Microsoft 365, with the scopes
// of its IMAP and SMTP, and the addresses of Outlook.com are of them.
func TestMicrosoftAccount(t *testing.T) {
	_, a := newAccountsApp(t)
	assert.Equal(t, Microsoft, typeOf("me@outlook.com"))
	assert.Equal(t, Gmail, typeOf("me@ru.nl"), "an organization: chosen")
	Settings.addAccount(Account{Type: Microsoft, Email: "me@ru.nl"})
	assert.False(t, usesPassword("me@ru.nl"))
	assert.True(t, a.usesOAuth("me@ru.nl"))
	srv := a.server("me@ru.nl")
	assert.Equal(t, "outlook.office365.com:993", srv.IMAP)
	assert.Equal(t, "smtp.office365.com:587", srv.SMTP)
	assert.Empty(t, srv.Password)
	assert.Equal(t, microsoftScopes, a.authConfig(context.Background(), "me@ru.nl").Scopes)
	assert.False(t, (&newAccount{Account: Account{Type: Microsoft}}).ShouldDisplay("Password"))
	assert.False(t, (&Account{Type: Microsoft}).ShouldDisplay("IMAP"))
	assert.Equal(t, "Microsoft 365", Microsoft.Label())

	t.Setenv("MICROSOFT_OAUTH2_CLIENT_ID", "")
	_, _, err := tokenSource(Microsoft, a.authConfig(context.Background(), "me@ru.nl"))
	assert.ErrorContains(t, err, "MICROSOFT_OAUTH2_CLIENT_ID")
}
