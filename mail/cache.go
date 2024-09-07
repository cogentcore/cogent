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
	"strconv"
	"strings"

	"cogentcore.org/core/base/iox/jsonx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

// CacheData contains the data stored for a cached message in the cached messages file.
// It contains basic information about the message so that it can be displayed in the
// mail list in the GUI.
type CacheData struct {
	imap.Envelope
	UID      imap.UID
	Filename string
}

// ToMessage converts the [CacheData] to a [ReadMessage].
func (cd *CacheData) ToMessage() *ReadMessage {
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
	if a.Cache == nil {
		a.Cache = map[string]map[string][]*CacheData{}
	}
	if a.IMAPClient == nil {
		a.IMAPClient = map[string]*imapclient.Client{}
	}
	mbox := a.FindPath("splits/mbox").(*core.Tree)
	mbox.AsyncLock()
	mbox.DeleteChildren()
	mbox.AsyncUnlock()
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
	if a.Cache[email] == nil {
		a.Cache[email] = map[string][]*CacheData{}
	}

	c, err := imapclient.DialTLS("imap.gmail.com:993", nil)
	if err != nil {
		return fmt.Errorf("TLS dialing: %w", err)
	}
	defer c.Logout()

	a.IMAPClient[email] = c

	err = c.Authenticate(a.AuthClient[email])
	if err != nil {
		return fmt.Errorf("authenticating: %w", err)
	}

	mailboxes, err := c.List("", "*", nil).Collect()
	if err != nil {
		return fmt.Errorf("getting mailboxes: %w", err)
	}

	for _, mailbox := range mailboxes {
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
	if a.CurrentMailbox == "" {
		a.CurrentMailbox = mailbox
	}

	bemail := FilenameBase32(email)

	a.AsyncLock()
	mbox := a.FindPath("splits/mbox").(*core.Tree)
	embox := mbox.ChildByName(bemail)
	if embox == nil {
		embox = core.NewTree(mbox).SetText(email) // TODO(config)
		embox.AsTree().SetName(bemail)
	}
	core.NewTree(embox).SetText(mailbox).OnClick(func(e events.Event) {
		a.CurrentMailbox = mailbox
		a.Update()
	})
	a.AsyncUnlock()

	dir := filepath.Join(core.TheApp.AppDataDir(), "mail", bemail)
	err := os.MkdirAll(string(dir), 0700)
	if err != nil {
		return err
	}

	cachedFile := filepath.Join(core.TheApp.AppDataDir(), "caching", bemail, "cached-messages.json")
	err = os.MkdirAll(filepath.Dir(cachedFile), 0700)
	if err != nil {
		return err
	}

	var cached []*CacheData
	err = jsonx.Open(&cached, cachedFile)
	if err != nil && !errors.Is(err, fs.ErrNotExist) && !errors.Is(err, io.EOF) {
		return fmt.Errorf("opening cache list: %w", err)
	}
	a.Cache[email][mailbox] = cached

	_, err = c.Select(mailbox, nil).Wait()
	if err != nil {
		return fmt.Errorf("opening mailbox: %w", err)
	}

	// we want messages with UIDs not in the list we already cached
	criteria := &imap.SearchCriteria{}
	if len(cached) > 0 {
		uidset := imap.UIDSet{}
		for _, c := range cached {
			uidset.AddNum(c.UID)
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
	return a.CacheUIDs(uids, c, email, mailbox, dir, cached, cachedFile)
}

// CacheUIDs caches the messages with the given UIDs in the context of the
// other given values, using an iterative batched approach that fetches the
// five next most recent messages at a time, allowing for concurrent mail
// modifiation operations and correct ordering.
func (a *App) CacheUIDs(uids []imap.UID, c *imapclient.Client, email string, mailbox string, dir string, cached []*CacheData, cachedFile string) error {
	for len(uids) > 0 {
		num := min(5, len(uids))
		cuids := uids[len(uids)-num:] // the current batch of UIDs
		uids = uids[:len(uids)-num]   // the remaining UIDs

		fuidset := imap.UIDSet{}
		fuidset.AddNum(cuids...)

		fetchOptions := &imap.FetchOptions{
			Envelope: true,
			UID:      true,
			BodySection: []*imap.FetchItemBodySection{
				{Specifier: imap.PartSpecifierHeader},
				{Specifier: imap.PartSpecifierText},
			},
		}

		mcmd := c.Fetch(fuidset, fetchOptions)

		for {
			msg := mcmd.Next()
			if msg == nil {
				break
			}

			mdata, err := msg.Collect()
			if err != nil {
				return err
			}

			filename := strconv.FormatUint(uint64(mdata.UID), 32)
			filename = strings.Repeat("0", 7-len(filename)) + filename
			f, err := os.Create(filepath.Join(dir, filename))
			if err != nil {
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
				return fmt.Errorf("writing message: %w", err)
			}

			err = f.Close()
			if err != nil {
				return fmt.Errorf("closing message: %w", err)
			}

			cd := &CacheData{
				Envelope: *mdata.Envelope,
				UID:      mdata.UID,
				Filename: filename,
			}

			// we need to save the list of cached messages every time in case
			// we get interrupted or have an error
			cached = append(cached, cd)
			err = jsonx.Save(&cached, cachedFile)
			if err != nil {
				return fmt.Errorf("saving cache list: %w", err)
			}

			a.Cache[email][mailbox] = cached
			a.AsyncLock()
			a.Update()
			a.AsyncUnlock()
		}

		err := mcmd.Close()
		if err != nil {
			return fmt.Errorf("fetching messages: %w", err)
		}
	}
	return nil
}
