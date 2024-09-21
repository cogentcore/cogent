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
	"strings"
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
func (cm *CacheMessage) ToDisplay(rmp *readMessageParsed) *displayMessage {
	if cm == nil {
		return nil
	}
	return &displayMessage{
		From:        IMAPToMailAddresses(cm.From),
		To:          IMAPToMailAddresses(cm.To),
		Subject:     cm.Subject,
		Date:        cm.Date.Local(),
		Attachments: rmp.attachments,
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

// CacheMessages caches all of the messages from the server that
// have not already been cached. It caches them in the app's data directory.
func (a *App) CacheMessages() error {
	if a.cache == nil {
		a.cache = map[string]map[string]*CacheMessage{}
	}
	if a.imapClient == nil {
		a.imapClient = map[string]*imapclient.Client{}
	}
	if a.imapMu == nil {
		a.imapMu = map[string]*sync.Mutex{}
	}
	for _, account := range Settings.Accounts {
		err := a.CacheMessagesForAccount(account)
		if err != nil {
			return fmt.Errorf("caching messages for account %q: %w", account, err)
		}
	}
	return nil
}

// CacheMessages caches all of the messages from the server that
// have not already been cached for the given email account. It
// caches them in the app's data directory.
func (a *App) CacheMessagesForAccount(email string) error {
	if a.cache[email] == nil {
		a.cache[email] = map[string]*CacheMessage{}
	}

	c, err := imapclient.DialTLS("imap.gmail.com:993", nil)
	if err != nil {
		return fmt.Errorf("TLS dialing: %w", err)
	}
	defer c.Logout()

	a.imapClient[email] = c
	a.imapMu[email] = &sync.Mutex{}

	err = c.Authenticate(a.authClient[email])
	if err != nil {
		return fmt.Errorf("authenticating: %w", err)
	}

	dir := filepath.Join(core.TheApp.AppDataDir(), "mail", FilenameBase32(email))
	err = os.MkdirAll(string(dir), 0700)
	if err != nil {
		return err
	}

	cacheFile := a.cacheFilename(email)
	err = os.MkdirAll(filepath.Dir(cacheFile), 0700)
	if err != nil {
		return err
	}

	cached := map[string]*CacheMessage{}
	err = jsonx.Open(&cached, cacheFile)
	if err != nil && !errors.Is(err, fs.ErrNotExist) && !errors.Is(err, io.EOF) {
		return fmt.Errorf("opening cache list: %w", err)
	}
	a.cache[email] = cached

	mailboxes, err := c.List("", "*", nil).Collect()
	if err != nil {
		return fmt.Errorf("getting mailboxes: %w", err)
	}

	for _, mailbox := range mailboxes {
		a.labels[email] = append(a.labels[email], mailbox.Mailbox)
	}

	for _, mailbox := range mailboxes {
		if strings.HasPrefix(mailbox.Mailbox, "[Gmail]") {
			continue // TODO: skipping for now until we figure out a good way to handle
		}
		err := a.CacheMessagesForMailbox(c, email, mailbox.Mailbox, dir, cached, cacheFile)
		if err != nil {
			return fmt.Errorf("caching messages for mailbox %q: %w", mailbox.Mailbox, err)
		}
	}
	return nil
}

// CacheMessagesForMailbox caches all of the messages from the server
// that have not already been cached for the given email account and mailbox.
// It caches them in the app's data directory.
func (a *App) CacheMessagesForMailbox(c *imapclient.Client, email string, mailbox string, dir string, cached map[string]*CacheMessage, cacheFile string) error {
	err := a.selectMailbox(c, email, mailbox)
	if err != nil {
		return err
	}

	// We want messages in this mailbox with UIDs we haven't already cached.
	criteria := &imap.SearchCriteria{}
	if len(cached) > 0 {
		uidset := imap.UIDSet{}
		for _, cm := range cached {
			for _, label := range cm.Labels {
				if label.Name == mailbox {
					uidset.AddNum(label.UID)
				}
			}
		}

		nc := imap.SearchCriteria{}
		nc.UID = []imap.UIDSet{uidset}
		criteria.Not = append(criteria.Not, nc)
	}

	// these are the UIDs of the new messages
	uidsData, err := c.UIDSearch(criteria, nil).Wait()
	if err != nil {
		return fmt.Errorf("searching for uids: %w", err)
	}

	uids := uidsData.AllUIDs()
	if len(uids) == 0 {
		a.AsyncLock()
		a.Update()
		a.AsyncUnlock()
		return nil
	}
	return a.CacheUIDs(uids, c, email, mailbox, dir, cached, cacheFile)
}

// CacheUIDs caches the messages with the given UIDs in the context of the
// other given values, using an iterative batched approach that fetches the
// five next most recent messages at a time, allowing for concurrent mail
// modifiation operations and correct ordering.
func (a *App) CacheUIDs(uids []imap.UID, c *imapclient.Client, email string, mailbox string, dir string, cached map[string]*CacheMessage, cacheFile string) error {
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
			// we get interrupted or have an error. We save it through a temporary
			// file to avoid truncating it without writing it if we quit during the process.
			// We also start the AsyncLock here so that we cannot quit from the GUI while
			// saving the file.
			a.AsyncLock()
			err = jsonx.Save(&cached, cacheFile+".tmp")
			if err != nil {
				a.imapMu[email].Unlock()
				return fmt.Errorf("saving cache list: %w", err)
			}
			err = os.Rename(cacheFile+".tmp", cacheFile)
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
