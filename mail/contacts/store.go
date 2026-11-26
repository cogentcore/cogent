// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package contacts

import (
	"bytes"
	"cmp"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"cogentcore.org/core/core"
	"github.com/emersion/go-vcard"
)

// Store is the contacts in a directory, each in its own vCard file,
// named by its UID, which other programs can read and write.
type Store struct {

	// Dir is the directory of the vCard files.
	Dir string

	// NewAccount is the name of the account that the contacts made here
	// (with [Store.New]) are added to, in its next sync (see [Store.Sync]),
	// or "" to keep them here.
	NewAccount string

	// mu is the mutex of contacts and onChange.
	mu sync.Mutex

	// syncMu makes one sync happen at a time, and is the mutex of the
	// states of the syncing.
	syncMu sync.Mutex

	// contacts are all of the contacts, sorted by name.
	contacts []*Contact

	// onChange are the functions called when the contacts change in a sync.
	onChange []func()
}

// OnChange adds a function that is called when the contacts change in a
// sync (see [Store.Sync]), in the goroutine of the sync, as for updating
// a browser of them.
func (s *Store) OnChange(f func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onChange = append(s.onChange, f)
}

// changed calls the functions of [Store.OnChange].
func (s *Store) changed() {
	s.mu.Lock()
	fs := slices.Clone(s.onChange)
	s.mu.Unlock()
	for _, f := range fs {
		f()
	}
}

// DefaultDir returns the standard directory of the contacts, shared by the
// apps that use them, as Cogent Mail and Contacts.
func DefaultDir() string {
	return filepath.Join(core.TheApp.DataDir(), "Cogent Contacts", "cards")
}

// Open opens the store of the contacts in the given directory, making it
// if needed, loading all of them.
func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	s := &Store{Dir: dir}
	files, err := filepath.Glob(filepath.Join(dir, "*.vcf"))
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		cards, err := readCards(f)
		if err != nil {
			slog.Error("contacts: reading a contact", "file", f, "err", err)
			continue
		}
		for _, card := range cards {
			c := fromCard(card)
			if c.uid == "" {
				c.uid = strings.TrimSuffix(filepath.Base(f), ".vcf")
			}
			s.contacts = append(s.contacts, c)
		}
	}
	s.sort()
	storesMu.Lock()
	stores = append(stores, s) // whose labels are offered (see [Labels])
	storesMu.Unlock()
	return s, nil
}

// readCards reads the vCards in the given file.
func readCards(file string) ([]vcard.Card, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return decodeCards(f)
}

// decodeCards decodes all of the vCards from the given reader.
func decodeCards(r io.Reader) ([]vcard.Card, error) {
	dec := vcard.NewDecoder(r)
	var cards []vcard.Card
	for {
		card, err := dec.Decode()
		if errors.Is(err, io.EOF) {
			return cards, nil
		}
		if err != nil {
			return cards, err
		}
		cards = append(cards, card)
	}
}

// sort sorts the contacts by name. It must be called with the lock.
func (s *Store) sort() {
	slices.SortStableFunc(s.contacts, func(a, b *Contact) int {
		return cmp.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})
}

// Contacts returns all of the contacts, sorted by name.
func (s *Store) Contacts() []*Contact {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.contacts)
}

// Search returns the contacts that match the given search: those in
// which each of its words starts a word of the name, organization, or an
// email address or phone number, ignoring case; all of them for "".
func (s *Store) Search(search string) []*Contact {
	words := strings.Fields(strings.ToLower(search))
	var res []*Contact
	for _, c := range s.Contacts() {
		text := []string{c.Name, c.Organization}
		for _, e := range c.Emails {
			text = append(text, e.Value)
		}
		for _, p := range c.Phones {
			text = append(text, p.Value)
		}
		if matchWords(words, strings.Join(text, " ")) {
			res = append(res, c)
		}
	}
	return res
}

// matchWords returns whether each of the given lowercase words starts a
// word of the given text, ignoring case, where the words of the text are
// separated by spaces and punctuation, as in email addresses.
func matchWords(words []string, text string) bool {
	tw := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return r == ' ' || r == '@' || r == '.' || r == '<' || r == '>' || r == ',' || r == '-' || r == '_' || r == '(' || r == ')'
	})
	for _, w := range words {
		if !slices.ContainsFunc(tw, func(t string) bool { return strings.HasPrefix(t, w) }) {
			return false
		}
	}
	return true
}

// New returns a new contact, which is not in the store until it is saved,
// when it is to be added to the account of [Store.NewAccount], if any.
func (s *Store) New() *Contact {
	return &Contact{uid: newUID(), isNew: true}
}

// newUID returns a new unique ID for a contact.
func newUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	h := hex.EncodeToString(b)
	return fmt.Sprintf("%s-%s-%s-%s-%s", h[:8], h[8:12], h[12:16], h[16:20], h[20:])
}

// file returns the file of the given contact.
func (s *Store) file(c *Contact) string {
	name := strings.Map(func(r rune) rune {
		if r < ' ' || strings.ContainsRune(`<>:"/\|?*`, r) {
			return '_'
		}
		return r
	}, c.uid)
	return filepath.Join(s.Dir, name+".vcf")
}

// Save saves the given contact in its file, adding it to the store if it
// is new.
func (s *Store) Save(c *Contact) error {
	if c.uid == "" {
		c.uid = newUID()
	}
	var b bytes.Buffer
	if err := vcard.NewEncoder(&b).Encode(c.toCard()); err != nil {
		return err
	}
	if err := os.WriteFile(s.file(c), b.Bytes(), 0o600); err != nil {
		return err
	}
	s.mu.Lock()
	if !slices.Contains(s.contacts, c) {
		s.contacts = append(s.contacts, c)
	}
	s.sort()
	s.mu.Unlock()
	if c.isNew {
		c.isNew = false
		return s.addNew(c)
	}
	return nil
}

// Delete deletes the given contact, with its file.
func (s *Store) Delete(c *Contact) error {
	if err := os.Remove(s.file(c)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.contacts = slices.DeleteFunc(s.contacts, func(x *Contact) bool { return x == c })
	return nil
}

// Import adds the contacts in the given vCards, as a .vcf file exported
// from another program (as Contacts on the Mac), replacing those with the
// same UIDs, and returns the number of them.
func (s *Store) Import(r io.Reader) (int, error) {
	cards, err := decodeCards(r)
	if err != nil && len(cards) == 0 {
		return 0, err
	}
	n := 0
	for _, card := range cards {
		c := fromCard(card)
		if c.uid == "" {
			c.uid = newUID()
		}
		s.mu.Lock()
		if i := slices.IndexFunc(s.contacts, func(x *Contact) bool { return x.uid == c.uid }); i >= 0 {
			s.contacts[i] = c // replaced
		}
		s.mu.Unlock()
		if err := s.Save(c); err != nil {
			return n, err
		}
		n++
	}
	return n, err
}

// ImportFile imports the contacts in the given .vcf file (see [Store.Import]).
func (s *Store) ImportFile(file string) (int, error) {
	f, err := os.Open(file)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return s.Import(f)
}
