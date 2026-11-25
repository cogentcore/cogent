// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"crypto/sha256"
	"fmt"
	"net/mail"
	"os"
	"sync/atomic"
	"time"

	"github.com/cogentcore/go-imap/v2"
	"github.com/cogentcore/go-imap/v2/imapclient"
)

// CacheMessage contains the data stored for a cached message in the cached messages file.
// It contains basic information about the message so that it can be displayed in the
// mail list in the GUI.
type CacheMessage struct {
	imap.Envelope

	// Filename is the unique filename of the cached message contents.
	Filename string

	// Flags are the IMAP flags associated with the message.
	Flags []imap.Flag

	// GmailMsgID and GmailThreadID are the IDs of the message and its
	// conversation (thread) on Gmail, for an account on Gmail.
	GmailMsgID, GmailThreadID uint64 `json:",omitempty"`

	// Labels are the labels associated with the message.
	// Labels are many-to-many, similar to gmail. All labels
	// also correspond to IMAP mailboxes.
	Labels []Label

	// replies are other messages that are replies to this message.
	// They are not stored in the cache file or computed ahead of time;
	// rather, they are used for conversation combination in the list GUI.
	replies []*CacheMessage

	// key is the key of the message in the cache and the store.
	key string

	// parsed contains data parsed from this message. This is populated live
	// and not stored in the cache file.
	parsed readMessageParsed
}

// Label represents a Label associated with a message.
// It contains the name of the Label and the UID of the message
// in the IMAP mailbox corresponding to the Label.
type Label struct {
	Name string
	UID  imap.UID
}

// UIDSet returns an [imap.UIDSet] that contains just the UID
// of the message in the IMAP mailbox corresponding to the [Label].
func (lb *Label) UIDSet() imap.UIDSet {
	uidset := imap.UIDSet{}
	uidset.AddNum(lb.UID)
	return uidset
}

// IMAPToMailAddresses converts the given [imap.Address]es to [mail.Address]es.
func IMAPToMailAddresses(as []imap.Address) []*mail.Address {
	res := make([]*mail.Address, len(as))
	for i, a := range as {
		res[i] = &mail.Address{
			Name:    a.Name,
			Address: a.Addr(),
		}
	}
	return res
}

// cacheMessages syncs the messages of all of the accounts with their servers.
func (a *App) cacheMessages() error {
	for _, account := range Settings.Accounts {
		err := a.syncAccount(account)
		if err != nil {
			return fmt.Errorf("syncing messages for account %q: %w", account, err)
		}
	}
	return nil
}

// listMailboxes gets the mailboxes of the given account from the server,
// setting its labels, its trash mailbox, and its special-use mailboxes.
// It must be called with the IMAP mutex of the account held, and not the
// lock of the window.
func (a *App) listMailboxes(c *imapclient.Client, email string) ([]*imap.ListData, error) {
	mailboxes, err := c.List("", "*", nil).Collect()
	if err != nil {
		return nil, fmt.Errorf("getting mailboxes: %w", err)
	}
	a.AsyncLock()
	a.labels[email] = []string{}
	for _, mailbox := range mailboxes {
		a.labels[email] = append(a.labels[email], mailbox.Mailbox)
	}
	a.trash[email] = findTrash(mailboxes)
	su := map[imap.MailboxAttr]string{}
	for _, mb := range mailboxes {
		for _, attr := range mb.Attrs {
			su[attr] = mb.Mailbox
		}
	}
	a.specialUse[email] = su
	a.AsyncUnlock()
	return mailboxes, nil
}

// cacheKey returns the key of the message with the given envelope in the
// cache: its Message-ID, or, for a message without one, a hash of its
// envelope, which is the same for its copies in other mailboxes.
func cacheKey(env *imap.Envelope) string {
	if env.MessageID != "" {
		return env.MessageID
	}
	h := sha256.New()
	fmt.Fprintf(h, "%d\n%s\n%v\n%v\n%v", env.Date.UnixNano(), env.Subject, env.From, env.To, env.Cc)
	return fmt.Sprintf("no-message-id-%x", h.Sum(nil)[:16])
}

var messageFilenameCounter uint64

// messageFilename returns a unique filename for storing a message
// based on the current time, the current process ID, and an atomic counter,
// which ensures uniqueness.
func messageFilename() string {
	return fmt.Sprintf("%d%d%d", time.Now().UnixMilli(), os.Getpid(), atomic.AddUint64(&messageFilenameCounter, 1))
}
