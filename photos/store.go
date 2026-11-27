// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package photos

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	bolt "go.etcd.io/bbolt"
)

// The buckets of the [store].
var (
	// photosBucket has the [Photo]s, as JSON, by their names.
	photosBucket = []byte("photos")

	// metaBucket has information about the library, such as whether the
	// folders of GoPix were imported.
	metaBucket = []byte("meta")
)

// store is the store of the information about the photos of a library:
// a bbolt database with the [Photo]s, written as they change.
type store struct {
	db *bolt.DB
}

// openStore opens the store in the given file, creating it if needed.
func openStore(filename string) (*store, error) {
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		return nil, err
	}
	db, err := bolt.Open(filename, 0o644, &bolt.Options{Timeout: 5 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("opening the store of photos: %w", err)
	}
	err = db.Update(func(tx *bolt.Tx) error {
		for _, b := range [][]byte{photosBucket, metaBucket} {
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
	return &store{db: db}, nil
}

// Close closes the store.
func (st *store) Close() error {
	return st.db.Close()
}

// load returns all of the photos in the store, by their names.
func (st *store) load() (map[string]*Photo, error) {
	res := map[string]*Photo{}
	err := st.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket(photosBucket).ForEach(func(k, v []byte) error {
			ph := &Photo{}
			if err := json.Unmarshal(v, ph); err != nil {
				return fmt.Errorf("photo %q: %w", k, err)
			}
			ph.Name = string(k)
			if ph.DateTaken.Location() != time.UTC { // as from GoPix
				ph.DateTaken = noZone(ph.DateTaken)
			}
			res[ph.Name] = ph
			return nil
		})
	})
	return res, err
}

// put stores the given photos, by their names, in one transaction.
func (st *store) put(phs ...*Photo) error {
	if len(phs) == 0 {
		return nil
	}
	return st.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(photosBucket)
		for _, ph := range phs {
			data, err := json.Marshal(ph)
			if err != nil {
				return err
			}
			if err := b.Put([]byte(ph.Name), data); err != nil {
				return err
			}
		}
		return nil
	})
}

// remove removes the photos with the given names.
func (st *store) remove(names ...string) error {
	if len(names) == 0 {
		return nil
	}
	return st.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(photosBucket)
		for _, n := range names {
			if err := b.Delete([]byte(n)); err != nil {
				return err
			}
		}
		return nil
	})
}

// rename stores the given photo under its new name, removing it under its
// old name, in one transaction.
func (st *store) rename(old string, ph *Photo) error {
	return st.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(photosBucket)
		data, err := json.Marshal(ph)
		if err != nil {
			return err
		}
		if err := b.Delete([]byte(old)); err != nil {
			return err
		}
		return b.Put([]byte(ph.Name), data)
	})
}

// renameAll stores the given photos under their new names, removing them
// under their old names, by which they are given, in one transaction.
func (st *store) renameAll(phs map[string]*Photo) error {
	return st.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(photosBucket)
		for old, ph := range phs {
			data, err := json.Marshal(ph)
			if err != nil {
				return err
			}
			if err := b.Delete([]byte(old)); err != nil {
				return err
			}
			if err := b.Put([]byte(ph.Name), data); err != nil {
				return err
			}
		}
		return nil
	})
}

// meta returns the value of the given key of the information about the
// library, or "" if it has none.
func (st *store) meta(key string) string {
	var res string
	st.db.View(func(tx *bolt.Tx) error {
		res = string(tx.Bucket(metaBucket).Get([]byte(key)))
		return nil
	})
	return res
}

// setMeta sets the value of the given key of the information about the
// library.
func (st *store) setMeta(key, value string) error {
	return st.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(metaBucket).Put([]byte(key), []byte(value))
	})
}
