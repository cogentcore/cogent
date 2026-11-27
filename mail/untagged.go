// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"github.com/cogentcore/go-imap/v2"
)

// untaggedLabel is the label of the Untagged mailbox of a Gmail account,
// which is not on the server: the messages of All Mail without any other
// label (see [App.isUntagged]), as those archived in Gmail, and those whose
// only label was a label that was deleted, which would otherwise not be
// shown, as All Mail is not.
const untaggedLabel = "Untagged"

// isUntagged returns whether the given message of the given account is in
// its Untagged mailbox (see [untaggedLabel]): a message of a Gmail account
// in All Mail, without any other label than Important, which is not shown
// as a mailbox, as the messages archived: those that no other mailbox shows.
// Those in the Inbox, a label, the Sent, the Drafts, the Trash, or the
// Spam, or starred (in Starred), are not.
func (a *App) isUntagged(email string, cm *CacheMessage) bool {
	if !a.gmail[email] || cm.isLocal() {
		return false
	}
	su := a.specialUse[email]
	all := su[imap.MailboxAttrAll]
	if all == "" {
		return false
	}
	if cm.isStarred() {
		return false // in Starred
	}
	important := su[imap.MailboxAttrImportant]
	inAll := false
	for _, l := range cm.Labels {
		switch {
		case l.Name == all:
			inAll = true
		case l.Name != "" && l.Name == important:
		default:
			return false
		}
	}
	return inAll
}

// untaggedMessages returns the messages of the Untagged mailbox of the
// given account (see [App.isUntagged]).
func (a *App) untaggedMessages(email string) []*CacheMessage {
	var res []*CacheMessage
	for _, cm := range a.cache[email] {
		if a.isUntagged(email, cm) {
			res = append(res, cm)
		}
	}
	return res
}

// markUntaggedRead marks all of the messages of the Untagged mailbox of the
// given account as read, which is not a mailbox on the server, for its
// Mark all as read.
func (a *App) markUntaggedRead(email string) {
	a.setFlagOn(email, a.untaggedMessages(email), imap.FlagSeen, true)
}
