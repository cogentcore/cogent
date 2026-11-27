// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"syscall"
	"time"

	"github.com/cogentcore/go-imap/v2"
	bolt "go.etcd.io/bbolt"
)

// The app works offline with the local copy of the mail: the messages
// of each account are shown from its store as soon as the app starts,
// with its mailboxes as they were at the last sync (see [accountInfo]),
// before signing in and connecting. A saved sign-in that cannot be
// checked, as when offline, is tried again in each sync, and the
// messages that are sent wait in the outbox (see [draft]) until they can
// be sent.

// isOffline returns whether the given error is of not being able to reach
// a server, as when the computer is offline, instead of an error of the
// server.
func isOffline(err error) bool {
	if err == nil {
		return false
	}
	var ne net.Error
	return errors.As(err, &ne) || errors.Is(err, net.ErrClosed) ||
		errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, syscall.ECONNREFUSED) || errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.ENETUNREACH) || errors.Is(err, syscall.EHOSTUNREACH) ||
		errors.Is(err, errNotResponding)
}

// errNotResponding is the error of a connection that is not responding.
var errNotResponding = errors.New("the connection is not responding")

// accountInfo is the information about an account from its server, kept
// in its store, for showing its mail before connecting, as when offline.
type accountInfo struct {

	// Mailboxes are the names of its mailboxes.
	Mailboxes []string

	// Trash is its trash mailbox.
	Trash string

	// SpecialUse are its special-use mailboxes, by their attributes.
	SpecialUse map[imap.MailboxAttr]string

	// Gmail and CondStore are whether its server has the Gmail
	// extension and CONDSTORE.
	Gmail, CondStore bool
}

// accountBucket has the [accountInfo] of the account of a [messageStore],
// as JSON, at accountKey.
var accountBucket, accountKey = []byte("account"), []byte("info")

// info returns the information about the account of the store, which is
// zero if it has not been saved.
func (ms *messageStore) info() (accountInfo, error) {
	var info accountInfo
	err := ms.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(accountBucket)
		if b == nil {
			return nil
		}
		v := b.Get(accountKey)
		if v == nil {
			return nil
		}
		return json.Unmarshal(v, &info)
	})
	return info, err
}

// setInfo saves the given information about the account of the store.
func (ms *messageStore) setInfo(info accountInfo) error {
	data, err := json.Marshal(info)
	if err != nil {
		return err
	}
	return ms.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(accountBucket)
		if err != nil {
			return err
		}
		return b.Put(accountKey, data)
	})
}

// saveAccountInfo saves the information about the given account from its
// server in its store. It must be called with the lock of the window.
func (a *App) saveAccountInfo(email string) error {
	st := a.stores[email]
	if st == nil {
		return nil
	}
	return st.setInfo(accountInfo{Mailboxes: a.labels[email], Trash: a.trash[email],
		SpecialUse: a.specialUse[email], Gmail: a.gmail[email], CondStore: a.condStore[email]})
}

// setAccountInfo sets the information about the given account from the
// given saved one, if it is not known from its server yet. It must be
// called with the lock of the window.
func (a *App) setAccountInfo(email string, info accountInfo) {
	if a.labels[email] != nil || info.Mailboxes == nil {
		return
	}
	a.labels[email] = info.Mailboxes
	a.trash[email] = info.Trash
	a.specialUse[email] = info.SpecialUse
	a.gmail[email] = info.Gmail
	a.condStore[email] = info.CondStore
}

// openAllLocal opens the local copy of the mail of each of the accounts
// that is not open, to show it before connecting, logging any errors.
func (a *App) openAllLocal() {
	for _, email := range Settings.Emails() {
		if a.stores[email] != nil {
			continue
		}
		start := time.Now()
		err := a.openLocal(email)
		slog.Debug("mail: opened the local mail", "account", email, "messages", len(a.cache[email]),
			"mailboxes", len(a.labels[email]), "took", time.Since(start), "err", err)
		if err != nil {
			a.setStatus("Error opening the mail of "+email+": "+err.Error(), -1)
		}
	}
	a.AsyncLock()
	a.Update()
	a.AsyncUnlock()
	slog.Debug("mail: updated after opening the local mail")
}
