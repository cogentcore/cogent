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

	"cogentcore.org/core/base/iox/jsonx"
	"cogentcore.org/core/core"
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

// CacheData contains the data stored for a cached message in the cached messages file.
// It contains basic information about the message so that it can be displayed in the
// mail list in the GUI.
type CacheData struct {
	imap.Envelope
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

// ToMessage converts the [CacheData] to a [ReadMessage].
func (cd *CacheData) ToMessage() *ReadMessage {
	if cd == nil {
		return nil
	}
	return &ReadMessage{
		From:    IMAPToMailAddresses(cd.From),
		To:      IMAPToMailAddresses(cd.To),
		Subject: cd.Subject,
		Date:    cd.Date.Local(),
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
		a.cache = map[string]map[string]*CacheData{}
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
		a.cache[email] = map[string]*CacheData{}
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
		err := a.CacheMessagesForMailbox(c, email, mailbox.Mailbox)
		if err != nil {
			return fmt.Errorf("caching messages for mailbox %q: %w", mailbox.Mailbox, err)
		}
	}
	return nil
}

// CacheMessagesForMailbox caches all of the messages from the server
// that have not already been cached for the given email account and mailbox.
// It caches them in the app's data directory.
func (a *App) CacheMessagesForMailbox(c *imapclient.Client, email string, mailbox string) error {
	dir := filepath.Join(core.TheApp.AppDataDir(), "mail", FilenameBase32(email))
	err := os.MkdirAll(string(dir), 0700)
	if err != nil {
		return err
	}

	cacheFile := a.cacheFilename(email)
	err = os.MkdirAll(filepath.Dir(cacheFile), 0700)
	if err != nil {
		return err
	}

	cached := map[string]*CacheData{}
	err = jsonx.Open(&cached, cacheFile)
	if err != nil && !errors.Is(err, fs.ErrNotExist) && !errors.Is(err, io.EOF) {
		return fmt.Errorf("opening cache list: %w", err)
	}
	a.cache[email] = cached

	err = a.selectMailbox(c, email, mailbox)
	if err != nil {
		return err
	}

	// We want messages in this mailbox with UIDs we haven't already cached.
	criteria := &imap.SearchCriteria{}
	if len(cached) > 0 {
		uidset := imap.UIDSet{}
		for _, cd := range cached {
			for _, label := range cd.Labels {
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
func (a *App) CacheUIDs(uids []imap.UID, c *imapclient.Client, email string, mailbox string, dir string, cached map[string]*CacheData, cacheFile string) error {
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
				cd := cached[mdata.Envelope.MessageID]
				if !slices.ContainsFunc(cd.Labels, func(label Label) bool {
					return label.Name == mailbox
				}) {
					cd.Labels = append(cd.Labels, Label{mailbox, mdata.UID})
				}
			} else {
				// Otherwise, we add it as a new entry to the cache
				// and save the content to a file.
				cached[mdata.Envelope.MessageID] = &CacheData{
					Envelope: *mdata.Envelope,
					Flags:    mdata.Flags,
					Labels:   []Label{{mailbox, mdata.UID}},
				}

				f, err := os.Create(filepath.Join(dir, messageFilename(mdata.Envelope)))
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
			// we get interrupted or have an error.
			err = jsonx.Save(&cached, cacheFile)
			if err != nil {
				a.imapMu[email].Unlock()
				return fmt.Errorf("saving cache list: %w", err)
			}

			a.cache[email] = cached
			a.AsyncLock()
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

// messageFilename returns the filename for storing the message with the given envelope.
func messageFilename(env *imap.Envelope) string {
	return FilenameBase32(env.MessageID) + ".eml"
}
