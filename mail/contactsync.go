// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"cogentcore.org/cogent/mail/contacts"
	"golang.org/x/oauth2"
)

// The contacts of the Gmail accounts for which [SettingsData.SyncContacts]
// is set are synced with Google Contacts, with CardDAV (see
// [contacts.Store.Sync]), in the syncs of the mail, every
// [contactsSyncInterval]. That needs the access to the contacts (see
// [contacts.GoogleScope]), which is asked for when signing in to them.

// contactsSyncInterval is the time between the syncs of the contacts.
var contactsSyncInterval = 5 * time.Minute

// syncsContacts returns whether the contacts of the given account are
// synced.
func (a *App) syncsContacts(email string) bool {
	return Settings.SyncContacts[email] && (a.usesOAuth(email) || a.contactsServer != "")
}

// newContactsAccount returns the account that the contacts made here are
// added to: [SettingsData.ContactsAccount] if its contacts are synced, or
// else the first of those whose contacts are synced, or "".
func newContactsAccount() string {
	if ca := string(Settings.ContactsAccount); ca != "" && Settings.SyncContacts[ca] {
		return ca
	}
	for _, email := range Settings.Accounts {
		if Settings.SyncContacts[email] {
			return email
		}
	}
	return ""
}

// SetSyncContacts sets whether the contacts of the given Gmail account are
// synced with Google Contacts. Turning it on asks for the access to them,
// signing in to the account again.
func (a *App) SetSyncContacts(email string, on bool) {
	if Settings.SyncContacts == nil {
		Settings.SyncContacts = map[string]bool{}
	}
	if on {
		Settings.SyncContacts[email] = true
	} else {
		delete(Settings.SyncContacts, email)
	}
	a.saveSettings()
	if s := a.contactsStore(); s != nil {
		s.NewAccount = newContactsAccount()
	}
	if !on {
		return
	}
	a.contactsSynced[email] = time.Time{} // in the next sync
	a.authMu.Lock()
	delete(a.tokens, email) // without the access to the contacts
	a.authMu.Unlock()
	a.signInDialogFor([]string{email}, "Sign in again in your browser to allow Cogent Mail to access your contacts, "+
		"checking the box for them on Google's page.")
}

// syncContacts syncs the contacts of the given account, if they are synced
// and it is time, as in each sync of its mail.
func (a *App) syncContacts(email string) error {
	a.AsyncLock()
	last := a.contactsSynced[email]
	a.AsyncUnlock()
	if !a.syncsContacts(email) || time.Since(last) < contactsSyncInterval {
		return nil
	}
	a.authMu.Lock()
	ts := a.tokens[email]
	a.authMu.Unlock()
	s := a.contactsStore()
	if (ts == nil && a.contactsServer == "") || s == nil {
		return nil // until signed in
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	server, ab := contacts.GoogleAddressBook(email)
	var client *http.Client
	if a.contactsServer != "" { // in tests
		server, ab, client = a.contactsServer, "", http.DefaultClient
	} else {
		client = oauth2.NewClient(ctx, ts)
	}
	acct, err := contacts.NewAccount(email, client, server, ab)
	if err != nil {
		return err
	}
	a.setStatus(email+": syncing contacts", -1)
	res, err := s.Sync(ctx, acct)
	if err != nil {
		return fmt.Errorf("syncing the contacts of %s: %w", email, err)
	}
	if res != (contacts.SyncResult{}) {
		slog.Info("mail: synced contacts", "account", email, "result", res.String())
	}
	a.AsyncLock()
	defer a.AsyncUnlock()
	a.contactsSynced[email] = time.Now()
	a.mailCandidates = nil // with the new names
	return nil
}

// authScopes returns the OAuth scopes of the given account: those of
// Gmail, and of its contacts if they are synced.
func authScopes(email string) []string {
	scopes := slices.Clone(gmailScopes)
	if Settings.SyncContacts[email] {
		scopes = append(scopes, contacts.GoogleScope)
	}
	return scopes
}
