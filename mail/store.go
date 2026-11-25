// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/cogentcore/go-imap/v2"
	bolt "go.etcd.io/bbolt"
)

// The buckets of a [messageStore].
var (
	// messagesBucket has the [CacheMessage]s, as JSON, by their keys
	// in the cache (see [cacheKey]).
	messagesBucket = []byte("messages")

	// mailboxesBucket has the [mailboxState]s, as JSON, by the names
	// of the mailboxes.
	mailboxesBucket = []byte("mailboxes")
)

// messageStore is the local store of the messages of an account, other
// than their contents, which are in files: a bbolt database with the
// [CacheMessage]s, written as they change, and the [mailboxState]s of
// the syncing of the mailboxes.
type messageStore struct {
	db *bolt.DB
}

// mailboxState is the state of the syncing of a mailbox of an account
// with the server, for getting only what changed since the last sync.
type mailboxState struct {

	// UIDValidity is the UIDVALIDITY of the mailbox: if it changes, the
	// UIDs of its messages are new, and those in the store are invalid.
	UIDValidity uint32

	// UIDNext is the UIDNEXT of the mailbox at the last sync: the
	// messages with lower UIDs have been synced.
	UIDNext imap.UID

	// HighestModSeq is the HIGHESTMODSEQ of the mailbox at the last sync,
	// for the changes after it, with CONDSTORE (0 if not supported).
	HighestModSeq uint64

	// Synced is when the mailbox was last fully synced.
	Synced time.Time
}

// openStore opens the store in the given file, creating it if needed.
func openStore(filename string) (*messageStore, error) {
	if err := os.MkdirAll(filepath.Dir(filename), 0o700); err != nil {
		return nil, err
	}
	db, err := bolt.Open(filename, 0o600, &bolt.Options{Timeout: 5 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("opening the store of messages: %w", err)
	}
	err = db.Update(func(tx *bolt.Tx) error {
		for _, b := range [][]byte{messagesBucket, mailboxesBucket} {
			if _, err := tx.CreateBucketIfNotExists(b); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		db.Close()
		return nil, err
	}
	return &messageStore{db: db}, nil
}

// Close closes the store.
func (ms *messageStore) Close() error {
	return ms.db.Close()
}

// load returns all of the messages in the store, by their keys.
func (ms *messageStore) load() (map[string]*CacheMessage, error) {
	res := map[string]*CacheMessage{}
	err := ms.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket(messagesBucket).ForEach(func(k, v []byte) error {
			cm := &CacheMessage{}
			if err := json.Unmarshal(v, cm); err != nil {
				return fmt.Errorf("message %q: %w", k, err)
			}
			res[string(k)] = cm
			return nil
		})
	})
	return res, err
}

// put stores the given messages, by their keys, in one transaction.
func (ms *messageStore) put(msgs map[string]*CacheMessage) error {
	if len(msgs) == 0 {
		return nil
	}
	return ms.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(messagesBucket)
		for k, cm := range msgs {
			data, err := json.Marshal(cm)
			if err != nil {
				return err
			}
			if err := b.Put([]byte(k), data); err != nil {
				return err
			}
		}
		return nil
	})
}

// remove removes the messages with the given keys from the store.
func (ms *messageStore) remove(keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return ms.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(messagesBucket)
		for _, k := range keys {
			if err := b.Delete([]byte(k)); err != nil {
				return err
			}
		}
		return nil
	})
}

// mailbox returns the state of the given mailbox, which is zero if it
// has not been synced.
func (ms *messageStore) mailbox(name string) (mailboxState, error) {
	var st mailboxState
	err := ms.db.View(func(tx *bolt.Tx) error {
		v := tx.Bucket(mailboxesBucket).Get([]byte(name))
		if v == nil {
			return nil
		}
		return json.Unmarshal(v, &st)
	})
	return st, err
}

// setMailbox sets the state of the given mailbox.
func (ms *messageStore) setMailbox(name string, st mailboxState) error {
	data, err := json.Marshal(st)
	if err != nil {
		return err
	}
	return ms.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(mailboxesBucket).Put([]byte(name), data)
	})
}

// removeMailbox removes the state of the given mailbox.
func (ms *messageStore) removeMailbox(name string) error {
	return ms.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(mailboxesBucket).Delete([]byte(name))
	})
}
