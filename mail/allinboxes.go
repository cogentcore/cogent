// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"slices"
)

// allInboxesShown is the [SettingsData.ShownAccount] when the inboxes of
// all of the accounts are shown (see [App.allInboxes]).
const allInboxesShown = "*"

// ShowAllInboxes shows the messages in the inboxes of all of the accounts
// together, the newest first.
func (a *App) ShowAllInboxes() {
	changed := !a.allInboxes
	if changed {
		a.selectedRows = nil
		a.searchResults = nil // back to the mailbox
		if a.searchField != nil {
			a.searchField.SetText("")
		}
	}
	a.allInboxes = true
	a.showLabel = "INBOX"
	if a.currentEmail == "" && len(Settings.Accounts) > 0 {
		a.currentEmail = Settings.Accounts[0].Email
	}
	if Settings.ShownAccount != allInboxesShown || Settings.ShownMailbox != "INBOX" {
		Settings.ShownAccount, Settings.ShownMailbox = allInboxesShown, "INBOX"
		a.saveSettings() // shown when the app starts again
	}
	a.Update()
	if changed { // its first message, as selected
		a.selectFirst()
	}
}

// listAccounts returns the accounts whose messages the list of messages
// shows: all of them for [App.allInboxes], or else the current one.
func (a *App) listAccounts() []string {
	if a.allInboxes {
		return Settings.Emails()
	}
	if a.currentEmail == "" {
		return nil
	}
	return []string{a.currentEmail}
}

// accountOf returns the account of the given message, the current one if
// it is not known (see [CacheMessage.account]).
func (a *App) accountOf(cm *CacheMessage) string {
	if cm.account != "" {
		return cm.account
	}
	return a.currentEmail
}

// byAccount calls the given function with the given messages of each
// account, in the order of the messages, as for an action on the
// selected messages of all of the inboxes (see [App.allInboxes]). The then
// functions are passed only to the last call, so that they are called once.
func (a *App) byAccount(msgs []*CacheMessage, f func(email string, msgs []*CacheMessage, then ...func()), then ...func()) {
	var emails []string
	groups := map[string][]*CacheMessage{}
	for _, cm := range msgs {
		email := a.accountOf(cm)
		if !slices.Contains(emails, email) {
			emails = append(emails, email)
		}
		groups[email] = append(groups[email], cm)
	}
	if len(emails) == 0 {
		f(a.currentEmail, nil, then...)
		return
	}
	for i, email := range emails {
		if i == len(emails)-1 {
			f(email, groups[email], then...)
		} else {
			f(email, groups[email])
		}
	}
}
