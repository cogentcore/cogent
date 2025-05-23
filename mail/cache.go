// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/mail"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"cogentcore.org/core/base/iox/jsonx"
	"cogentcore.org/core/core"
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
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

	// Labels are the labels associated with the message.
	// Labels are many-to-many, similar to gmail. All labels
	// also correspond to IMAP mailboxes.
	Labels []Label

	// replies are other messages that are replies to this message.
	// They are not stored in the cache file or computed ahead of time;
	// rather, they are used for conversation combination in the list GUI.
	replies []*CacheMessage

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

// ToDisplay converts the [CacheMessage] to a [displayMessage]
// with the given additional [readMessageParsed] data.
func (cm *CacheMessage) ToDisplay() *displayMessage {
	if cm == nil {
		return nil
	}
	return &displayMessage{
		From:        IMAPToMailAddresses(cm.From),
		To:          IMAPToMailAddresses(cm.To),
		Subject:     cm.Subject,
		Date:        cm.Date.Local(),
		Attachments: cm.parsed.attachments,
	}
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

// cacheMessages caches all of the messages from the server that
// have not already been cached. It caches them in the app's data directory.
func (a *App) cacheMessages() error {
	for _, account := range Settings.Accounts {
		err := a.cacheMessagesForAccount(account)
		if err != nil {
			return fmt.Errorf("caching messages for account %q: %w", account, err)
		}
	}
	return nil
}

// CacheMessages caches all of the messages from the server that
// have not already been cached for the given email account. It
// caches them in the app's data directory.
func (a *App) cacheMessagesForAccount(email string) error {
	if a.cache[email] == nil {
		a.cache[email] = map[string]*CacheMessage{}
	}
	if a.totalMessages[email] == nil {
		a.totalMessages[email] = map[string]int{}
	}

	dir := filepath.Join(core.TheApp.AppDataDir(), "mail", filenameBase32(email))
	cached := a.cache[email]

	var err error
	c := a.imapClient[email]
	if c == nil {
		c, err = imapclient.DialTLS("imap.gmail.com:993", nil)
		if err != nil {
			return fmt.Errorf("TLS dialing: %w", err)
		}
		// defer c.Logout() // TODO: Logout in QuitClean or something similar

		a.imapClient[email] = c
		a.imapMu[email] = &sync.Mutex{}

		err = c.Authenticate(a.authClient[email])
		if err != nil {
			return fmt.Errorf("authenticating: %w", err)
		}

		err = os.MkdirAll(string(dir), 0700)
		if err != nil {
			return err
		}

		cacheFile := a.cacheFilename(email)
		err = os.MkdirAll(filepath.Dir(cacheFile), 0700)
		if err != nil {
			return err
		}

		cached = map[string]*CacheMessage{}
		err = jsonx.Open(&cached, cacheFile)
		if err != nil && !errors.Is(err, fs.ErrNotExist) && !errors.Is(err, io.EOF) {
			return fmt.Errorf("opening cache list: %w", err)
		}
		a.cache[email] = cached
	}

	a.imapMu[email].Lock()
	mailboxes, err := c.List("", "*", nil).Collect()
	a.imapMu[email].Unlock()
	if err != nil {
		return fmt.Errorf("getting mailboxes: %w", err)
	}

	a.AsyncLock()
	a.labels[email] = []string{}
	for _, mailbox := range mailboxes {
		a.labels[email] = append(a.labels[email], mailbox.Mailbox)
	}
	a.AsyncUnlock()

	for _, mailbox := range mailboxes {
		if skipLabels[mailbox.Mailbox] {
			continue
		}
		err := a.cacheMessagesForMailbox(c, email, mailbox.Mailbox, dir, cached)
		if err != nil {
			return fmt.Errorf("caching messages for mailbox %q: %w", mailbox.Mailbox, err)
		}
	}
	return nil
}

// cacheMessagesForMailbox caches all of the messages from the server
// that have not already been cached for the given email account and mailbox.
// It caches them in the app's data directory.
func (a *App) cacheMessagesForMailbox(c *imapclient.Client, email string, mailbox string, dir string, cached map[string]*CacheMessage) error {
	a.imapMu[email].Lock()
	err := a.selectMailbox(c, email, mailbox)
	if err != nil {
		a.imapMu[email].Unlock()
		return err
	}

	uidsData, err := c.UIDSearch(&imap.SearchCriteria{}, nil).Wait()
	a.imapMu[email].Unlock()
	if err != nil {
		return fmt.Errorf("searching for uids: %w", err)
	}

	// These are all of the UIDs, including those we have already cached.
	uids := uidsData.AllUIDs()
	if len(uids) == 0 {
		a.AsyncLock()
		a.Update()
		a.AsyncUnlock()
		return nil
	}

	err = a.cleanCache(cached, email, mailbox, uids)
	if err != nil {
		return err
	}

	alreadyHaveSlice := []imap.UID{}
	alreadyHaveMap := map[imap.UID]bool{}
	for _, cm := range cached {
		for _, label := range cm.Labels {
			if label.Name == mailbox {
				alreadyHaveSlice = append(alreadyHaveSlice, label.UID)
				alreadyHaveMap[label.UID] = true
			}
		}
	}

	// We sync the flags of all UIDs we have already cached.
	err = a.syncFlags(alreadyHaveSlice, c, email, mailbox, cached)
	if err != nil {
		return err
	}

	// We filter out the UIDs that are already cached.
	uids = slices.DeleteFunc(uids, func(uid imap.UID) bool {
		return alreadyHaveMap[uid]
	})
	// We only cache in baches of 100 UIDs per mailbox to allow us to
	// get to multiple mailboxes quickly. We want the last UIDs since
	// those are typically the most recent messages.
	if len(uids) > 100 {
		uids = uids[len(uids)-100:]
	}

	return a.cacheUIDs(uids, c, email, mailbox, dir, cached)
}

// cleanCache removes cached messages from the given mailbox if
// they are not part of the given list of UIDs.
func (a *App) cleanCache(cached map[string]*CacheMessage, email string, mailbox string, uids []imap.UID) error {
	for id, cm := range cached {
		cm.Labels = slices.DeleteFunc(cm.Labels, func(label Label) bool {
			return label.Name == mailbox && !slices.Contains(uids, label.UID)
		})
		// We can remove the message since it is removed from all mailboxes.
		if len(cm.Labels) == 0 {
			delete(cached, id)
		}
	}
	a.imapMu[email].Lock()
	err := a.saveCacheFile(cached, email)
	a.imapMu[email].Unlock()
	return err
}

// cacheUIDs caches the messages with the given UIDs in the context of the
// other given values, using an iterative batched approach that fetches the
// five next most recent messages at a time, allowing for concurrent mail
// modifiation operations and correct ordering.
func (a *App) cacheUIDs(uids []imap.UID, c *imapclient.Client, email string, mailbox string, dir string, cached map[string]*CacheMessage) error {
	for len(uids) > 0 {
		num := min(5, len(uids))
		cuids := uids[len(uids)-num:] // the current batch of UIDs
		uids = uids[:len(uids)-num]   // the remaining UIDs

		fuidset := imap.UIDSet{}
		fuidset.AddNum(cuids...)

		fetchOptions := &imap.FetchOptions{
			Envelope: true,
			Flags:    true,
			UID:      true,
			BodySection: []*imap.FetchItemBodySection{
				{Specifier: imap.PartSpecifierHeader, Peek: true},
				{Specifier: imap.PartSpecifierText, Peek: true},
			},
		}

		a.imapMu[email].Lock()
		// We must reselect the mailbox in case the user has changed it
		// by doing actions in another mailbox. This is a no-op if it is
		// already selected.
		err := a.selectMailbox(c, email, mailbox)
		if err != nil {
			a.imapMu[email].Unlock()
			return err
		}
		mcmd := c.Fetch(fuidset, fetchOptions)

		for {
			msg := mcmd.Next()
			if msg == nil {
				break
			}

			mdata, err := msg.Collect()
			if err != nil {
				a.imapMu[email].Unlock()
				return err
			}

			// If the message is already cached (likely in another mailbox),
			// we update its labels to include this mailbox if it doesn't already.
			if _, already := cached[mdata.Envelope.MessageID]; already {
				cm := cached[mdata.Envelope.MessageID]
				if !slices.ContainsFunc(cm.Labels, func(label Label) bool {
					return label.Name == mailbox
				}) {
					cm.Labels = append(cm.Labels, Label{mailbox, mdata.UID})
				}
			} else {
				// Otherwise, we add it as a new entry to the cache
				// and save the content to a file.
				filename := messageFilename()
				cached[mdata.Envelope.MessageID] = &CacheMessage{
					Envelope: *mdata.Envelope,
					Filename: filename,
					Flags:    mdata.Flags,
					Labels:   []Label{{mailbox, mdata.UID}},
				}

				f, err := os.Create(filepath.Join(dir, filename))
				if err != nil {
					a.imapMu[email].Unlock()
					return err
				}

				var header, text []byte

				for k, v := range mdata.BodySection {
					if k.Specifier == imap.PartSpecifierHeader {
						header = v
					} else if k.Specifier == imap.PartSpecifierText {
						text = v
					}
				}

				_, err = f.Write(append(header, text...))
				if err != nil {
					a.imapMu[email].Unlock()
					return fmt.Errorf("writing message: %w", err)
				}

				err = f.Close()
				if err != nil {
					a.imapMu[email].Unlock()
					return fmt.Errorf("closing message: %w", err)
				}
			}

			// We need to save the list of cached messages every time in case
			// we get interrupted or have an error. We also start the AsyncLock
			// here so that we cannot quit from the GUI while saving the file.
			a.AsyncLock()
			err = a.saveCacheFile(cached, email)
			if err != nil {
				a.imapMu[email].Unlock()
				return err
			}

			a.cache[email] = cached
			a.Update()
			a.AsyncUnlock()
		}

		err = mcmd.Close()
		a.imapMu[email].Unlock()
		if err != nil {
			return fmt.Errorf("fetching messages: %w", err)
		}
	}
	return nil
}

var messageFilenameCounter uint64

// messageFilename returns a unique filename for storing a message
// based on the current time, the current process ID, and an atomic counter,
// which ensures uniqueness.
func messageFilename() string {
	return fmt.Sprintf("%d%d%d", time.Now().UnixMilli(), os.Getpid(), atomic.AddUint64(&messageFilenameCounter, 1))
}

// syncFlags updates the IMAP flags of cached messages to match those on the server.
func (a *App) syncFlags(uids []imap.UID, c *imapclient.Client, email string, mailbox string, cached map[string]*CacheMessage) error {
	if len(uids) == 0 {
		return nil
	}

	uidToMessage := map[imap.UID]*CacheMessage{}
	for _, cm := range cached {
		for _, label := range cm.Labels {
			if label.Name == mailbox {
				uidToMessage[label.UID] = cm
			}
		}
	}

	uidset := imap.UIDSet{}
	uidset.AddNum(uids...)

	fetchOptions := &imap.FetchOptions{Flags: true, UID: true}
	a.imapMu[email].Lock()
	// We must reselect the mailbox in case the user has changed it
	// by doing actions in another mailbox. This is a no-op if it is
	// already selected.
	err := a.selectMailbox(c, email, mailbox)
	if err != nil {
		a.imapMu[email].Unlock()
		return err
	}
	cmd := c.Fetch(uidset, fetchOptions)
	for {
		msg := cmd.Next()
		if msg == nil {
			break
		}

		mdata, err := msg.Collect()
		if err != nil {
			a.imapMu[email].Unlock()
			return err
		}
		uidToMessage[mdata.UID].Flags = mdata.Flags
	}
	err = a.saveCacheFile(cached, email)
	a.imapMu[email].Unlock()
	return err
}
