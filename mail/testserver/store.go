// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package testserver

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

// storedMailbox is a mailbox in the saved state of the server.
type storedMailbox struct {
	Name     string
	Messages []storedMessage
}

type storedMessage struct {
	Flags []imap.Flag
	Time  time.Time
	Data  []byte
}

// stateFile returns the file of the saved state of the given account in the given directory.
func stateFile(dir, account string) string {
	return filepath.Join(dir, account+".json")
}

// Save saves all of the mailboxes and messages of all of the accounts
// to the given directory, one JSON file per account, for [Server.Load].
func (s *Server) Save(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	for _, account := range s.Accounts {
		mbs, err := s.read(account)
		if err != nil {
			return fmt.Errorf("testserver: reading %q: %w", account, err)
		}
		b, err := json.Marshal(mbs)
		if err != nil {
			return err
		}
		fn := stateFile(dir, account)
		if err := os.WriteFile(fn+".tmp", b, 0o600); err != nil {
			return err
		}
		if err := os.Rename(fn+".tmp", fn); err != nil {
			return err
		}
	}
	return nil
}

// read returns all of the mailboxes and messages of the given account.
func (s *Server) read(account string) ([]storedMailbox, error) {
	c, err := imapclient.DialInsecure(s.IMAPAddr, nil)
	if err != nil {
		return nil, err
	}
	defer c.Close()
	if err := c.Login(account, Password).Wait(); err != nil {
		return nil, err
	}
	list, err := c.List("", "*", nil).Collect()
	if err != nil {
		return nil, err
	}
	var mbs []storedMailbox
	for _, l := range list {
		mb := storedMailbox{Name: l.Mailbox}
		sd, err := c.Select(l.Mailbox, &imap.SelectOptions{ReadOnly: true}).Wait()
		if err != nil {
			return nil, err
		}
		if sd.NumMessages > 0 {
			msgs, err := c.Fetch(imap.SeqSet{imap.SeqRange{Start: 1, Stop: 0}}, &imap.FetchOptions{
				Flags: true, InternalDate: true, UID: true,
				BodySection: []*imap.FetchItemBodySection{{Peek: true}},
			}).Collect()
			if err != nil {
				return nil, err
			}
			for _, m := range msgs {
				var data []byte
				for _, d := range m.BodySection {
					data = d
				}
				mb.Messages = append(mb.Messages, storedMessage{Flags: m.Flags, Time: m.InternalDate, Data: data})
			}
		}
		mbs = append(mbs, mb)
	}
	return mbs, c.Logout().Wait()
}

// Load loads the mailboxes and messages saved by [Server.Save] from the
// given directory, adding them to those of the server. It returns false
// if there is no saved state for any account.
func (s *Server) Load(dir string) (bool, error) {
	any := false
	for _, account := range s.Accounts {
		b, err := os.ReadFile(stateFile(dir, account))
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return any, err
		}
		any = true
		var mbs []storedMailbox
		if err := json.Unmarshal(b, &mbs); err != nil {
			return any, fmt.Errorf("testserver: loading %q: %w", account, err)
		}
		u := s.users[account]
		for _, mb := range mbs {
			if err := s.CreateMailbox(account, mb.Name); err != nil {
				return any, err
			}
			for _, m := range mb.Messages {
				_, err := u.Append(mb.Name, bytes.NewReader(m.Data), &imap.AppendOptions{Flags: m.Flags, Time: m.Time})
				if err != nil {
					return any, err
				}
			}
		}
	}
	return any, nil
}
