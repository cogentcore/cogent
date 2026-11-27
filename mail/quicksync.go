// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"errors"
	"log/slog"
	"time"

	"github.com/cogentcore/go-imap/v2"
)

// inboxSyncInterval is the time between the quick syncs of the inboxes
// (see [App.quickSyncLoop]), which are also synced as soon as their
// servers report new mail (see [App.idle]).
var inboxSyncInterval = 15 * time.Second

// errNotConnected is the error of syncing a mailbox of an account that is
// not connected to its server, which the regular sync connects to.
var errNotConnected = errors.New("not connected to the server")

// SyncInboxNow syncs the inboxes now instead of at the next quick sync.
func (a *App) SyncInboxNow() {
	select {
	case a.syncInbox <- struct{}{}:
	default: // already pending
	}
}

// quickSyncLoop syncs just the inboxes of the accounts that are connected,
// every [inboxSyncInterval] and when asked to (see [App.SyncInboxNow]), so
// that new mail is shown soon, along with the regular syncs of everything,
// which can take a while, as when downloading many messages.
func (a *App) quickSyncLoop() {
	for {
		select {
		case <-a.syncInbox:
		case <-time.After(inboxSyncInterval):
		}
		for _, email := range Settings.Emails() {
			if err := a.quickSync(email, "INBOX"); err != nil && !errors.Is(err, errNotConnected) {
				slog.Info("mail: quick sync of the inbox failed", "account", email, "err", err)
				a.SyncNow() // which connects again
			}
		}
	}
}

// quickSync syncs the mailbox of the given account that has the messages
// with the given label, without showing it in the status, if the account
// is connected (see [errNotConnected]).
func (a *App) quickSync(email, label string) error {
	mailbox := a.serverMailbox(email, label)
	if mailbox == "" {
		return errNotConnected
	}
	if err := a.syncMailbox(email, mailbox, true); err != nil {
		return err
	}
	a.countMessages(email)
	a.syncUpdate()
	return nil
}

// serverMailbox returns the mailbox on the server of the given account
// that has the messages with the given label: on Gmail, All Mail, the
// Trash, or the Spam, and on other servers, that of the label. It returns
// "" if the account is not connected or the label is not on the server,
// as the outbox.
func (a *App) serverMailbox(email, label string) string {
	a.AsyncLock()
	defer a.AsyncUnlock()
	if a.imapClient[email] == nil || a.stores[email] == nil {
		return ""
	}
	if a.gmail[email] {
		su := a.specialUse[email]
		if label == su[imap.MailboxAttrTrash] || label == su[imap.MailboxAttrJunk] {
			return label
		}
		return su[imap.MailboxAttrAll]
	}
	for _, l := range a.labels[email] {
		if l == label {
			return label
		}
	}
	return ""
}

// Refresh syncs the mailbox being shown with the server now, as when
// waiting for a message, and then everything else; for all of the
// inboxes, those of all of the accounts.
func (a *App) Refresh() { //types:add
	emails, label := a.listAccounts(), a.showLabel
	go func() {
		for _, email := range emails {
			err := a.quickSync(email, label)
			if err != nil && !errors.Is(err, errNotConnected) {
				slog.Info("mail: refreshing failed", "account", email, "mailbox", label, "err", err)
			}
		}
		a.SyncNow()
	}()
}
