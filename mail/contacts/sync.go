// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package contacts

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/emersion/go-webdav"
	"github.com/emersion/go-webdav/carddav"
)

// The contacts are synced with the address books of accounts on CardDAV
// servers, as Google Contacts and iCloud, both ways (see [Store.Sync]):
// the contacts that are new or changed on the server are added or changed
// here, those changed here are changed there, and those deleted on either
// side are deleted on the other. A contact changed on both sides since the
// last sync gets the changes of the server. The contacts made here (with
// [Store.New]) are added to the account of [Store.NewAccount], if any; the
// others, as those imported from vCard files, stay here.

// GoogleScope is the OAuth scope of the access to Google Contacts with
// CardDAV.
const GoogleScope = "https://www.googleapis.com/auth/carddav"

// GoogleAddressBook returns the CardDAV server and the address book of the
// contacts of the given Google account.
func GoogleAddressBook(email string) (server, addressBook string) {
	return "https://www.googleapis.com", "/carddav/v1/principals/" + email + "/lists/default/"
}

// Account is the address book of an account on a CardDAV server that the
// contacts are synced with.
type Account struct {

	// Name is the name of the account, as its email address, which names
	// the state of its syncing.
	Name string

	// Client is the CardDAV client of the server, whose HTTP client signs
	// in to the account (as with OAuth).
	Client *carddav.Client

	// AddressBook is the path of the address book, or "" to use the
	// first one of the account, found on the server.
	AddressBook string

	// errors keeps the messages of the error responses of the server,
	// and is its HTTP client.
	errors *errorsClient

	// server is the URL of the server.
	server string
}

// NewAccount returns a new account with the given name, synced with the
// given address book (or the first one, for "") on the CardDAV server at
// the given URL, with the given HTTP client.
func NewAccount(name string, client webdav.HTTPClient, server, addressBook string) (*Account, error) {
	ec := &errorsClient{client: client}
	c, err := carddav.NewClient(ec, server)
	if err != nil {
		return nil, err
	}
	return &Account{Name: name, Client: c, AddressBook: addressBook, errors: ec, server: server}, nil
}

// errorsClient is an HTTP client that keeps the message of the last error
// response of the server, which the CardDAV client does not have in its
// errors, only the status, as Google says what is wrong in it (as that its
// API is not enabled).
type errorsClient struct {
	client webdav.HTTPClient

	mu      sync.Mutex
	message string
}

func (ec *errorsClient) Do(req *http.Request) (*http.Response, error) {
	resp, err := ec.client.Do(req)
	if err != nil || resp.StatusCode < 400 {
		return resp, err
	}
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewReader(data))
	msg := strings.TrimSpace(string(data))
	var gerr struct{ Error struct{ Message string } } // as Google sends
	if json.Unmarshal(data, &gerr) == nil && gerr.Error.Message != "" {
		msg = gerr.Error.Message
	}
	if len(msg) > 500 || strings.HasPrefix(msg, "<") { // not a message
		msg = ""
	}
	ec.mu.Lock()
	ec.message = msg
	ec.mu.Unlock()
	return resp, err
}

// explain returns the given error with the message of the last error
// response of the server, if any.
func (ec *errorsClient) explain(err error) error {
	if err == nil || ec == nil {
		return err
	}
	ec.mu.Lock()
	msg := ec.message
	ec.message = ""
	ec.mu.Unlock()
	if msg == "" {
		return err
	}
	return fmt.Errorf("%w: %s", err, msg)
}

// syncState is the state of the syncing of the contacts with an account,
// saved between syncs.
type syncState struct {

	// Cards are the cards on the server that the contacts are synced
	// with, by their paths.
	Cards map[string]*syncedCard

	// New are the UIDs of the contacts made here to add to the server.
	New map[string]bool
}

// syncedCard is a card on the server that a contact is synced with.
type syncedCard struct {

	// UID is the UID of the contact.
	UID string

	// ETag is the ETag of the card on the server at the last sync.
	ETag string

	// Hash is the hash of the contact at the last sync (see [Contact.hash]),
	// for whether it changed here since.
	Hash string
}

// SyncResult is the result of a sync of the contacts with an account.
type SyncResult struct {

	// Added, Changed, and Deleted are the numbers of contacts added,
	// changed, and deleted here.
	Added, Changed, Deleted int

	// Uploaded and Removed are the numbers of contacts added or changed,
	// and deleted, on the server.
	Uploaded, Removed int
}

// Changes returns whether the contacts here changed.
func (r SyncResult) Changes() bool {
	return r.Added+r.Changed+r.Deleted > 0
}

func (r SyncResult) String() string {
	return fmt.Sprintf("%d added, %d changed, %d deleted here; %d uploaded, %d removed there",
		r.Added, r.Changed, r.Deleted, r.Uploaded, r.Removed)
}

// hash returns a hash of the edited fields of the contact, for whether
// it changed since the last sync.
func (c *Contact) hash() string {
	data, _ := json.Marshal(c)
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// stateFile returns the file of the state of the syncing with the account
// with the given name.
func (s *Store) stateFile(account string) string {
	name := strings.Map(func(r rune) rune {
		if r < ' ' || strings.ContainsRune(`<>:"/\|?*`, r) {
			return '_'
		}
		return r
	}, account)
	return filepath.Join(filepath.Dir(s.Dir), "sync", name+".json")
}

// loadState returns the state of the syncing with the given account.
func (s *Store) loadState(account string) (*syncState, error) {
	st := &syncState{}
	data, err := os.ReadFile(s.stateFile(account))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	if err == nil {
		if err := json.Unmarshal(data, st); err != nil {
			return nil, err
		}
	}
	if st.Cards == nil {
		st.Cards = map[string]*syncedCard{}
	}
	if st.New == nil {
		st.New = map[string]bool{}
	}
	return st, nil
}

// saveState saves the state of the syncing with the given account.
func (s *Store) saveState(account string, st *syncState) error {
	fn := s.stateFile(account)
	if err := os.MkdirAll(filepath.Dir(fn), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(st, "", "\t")
	if err != nil {
		return err
	}
	return os.WriteFile(fn, data, 0o600)
}

// addNew records that the given new contact is to be added to the account
// of [Store.NewAccount], if any, in its next sync.
func (s *Store) addNew(c *Contact) error {
	account := s.NewAccount
	if account == "" {
		return nil
	}
	s.syncMu.Lock()
	defer s.syncMu.Unlock()
	st, err := s.loadState(account)
	if err != nil {
		return err
	}
	st.New[c.uid] = true
	return s.saveState(account, st)
}

// byUID returns the contact with the given UID, or nil.
func (s *Store) byUID(uid string) *Contact {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range s.contacts {
		if c.uid == uid {
			return c
		}
	}
	return nil
}

// replace replaces the given old contact with the given new one, or adds
// it for nil, saving it, with the UID of the old one.
func (s *Store) replace(old, c *Contact) error {
	if old != nil {
		c.uid = old.uid
		s.mu.Lock()
		for i, x := range s.contacts {
			if x == old {
				s.contacts[i] = c
			}
		}
		s.mu.Unlock()
	}
	return s.Save(c)
}

// Sync syncs the contacts with the address book of the given account,
// both ways, as described in the doc of the package, and notifies the
// functions of [Store.OnChange] if the contacts here changed.
func (s *Store) Sync(ctx context.Context, acct *Account) (res SyncResult, err error) {
	defer func() { err = acct.errors.explain(err) }()
	s.syncMu.Lock()
	defer s.syncMu.Unlock()
	defer func() {
		if res.Changes() {
			s.changed()
		}
	}()
	ab := acct.AddressBook
	if ab == "" {
		var err error
		ab, err = findAddressBook(ctx, acct.Client)
		if err != nil {
			return res, err
		}
	}
	st, err := s.loadState(acct.Name)
	if err != nil {
		return res, err
	}
	remote, err := acct.listCards(ctx, ab, st)
	if err != nil {
		return res, fmt.Errorf("getting the contacts of %s: %w", acct.Name, err)
	}
	seen := map[string]bool{}
	var errs []error
	for _, ao := range remote {
		seen[ao.Path] = true
		sc := st.Cards[ao.Path]
		if sc == nil { // new on the server
			c := fromCard(ao.Card)
			if c.uid == "" {
				c.uid = strings.TrimSuffix(path.Base(ao.Path), ".vcf")
			}
			old := s.byUID(c.uid) // as one uploaded before the state was lost
			if err := s.replace(old, c); err != nil {
				return res, err
			}
			if old == nil {
				res.Added++
			} else {
				res.Changed++
			}
			st.Cards[ao.Path] = &syncedCard{UID: c.uid, ETag: ao.ETag, Hash: c.hash()}
			continue
		}
		local := s.byUID(sc.UID)
		switch {
		case local == nil && ao.ETag == sc.ETag: // deleted here
			err := acct.Client.RemoveAll(ctx, ao.Path, &webdav.RemoveAllOptions{IfMatch: etagMatch(sc.ETag)})
			if err != nil {
				errs = append(errs, fmt.Errorf("deleting a contact on the server: %w", err))
				continue
			}
			delete(st.Cards, ao.Path)
			res.Removed++
		case local == nil || ao.ETag != sc.ETag: // changed on the server, which wins
			if local != nil && local.hash() != sc.Hash {
				slog.Info("contacts: changed both here and on the server, which wins", "name", local.Name, "account", acct.Name)
			}
			c := fromCard(ao.Card)
			c.uid = sc.UID
			if err := s.replace(local, c); err != nil {
				return res, err
			}
			if local == nil {
				res.Added++
			} else {
				res.Changed++
			}
			sc.ETag, sc.Hash = ao.ETag, c.hash()
		case local.hash() != sc.Hash: // changed here
			if err := s.upload(ctx, acct, ao.Path, local, sc, etagMatch(sc.ETag), ""); err != nil {
				errs = append(errs, err)
				continue
			}
			res.Uploaded++
		}
	}
	for p, sc := range st.Cards { // deleted on the server
		if seen[p] {
			continue
		}
		delete(st.Cards, p)
		local := s.byUID(sc.UID)
		if local == nil {
			continue
		}
		if local.hash() != sc.Hash { // changed here: kept, and added again
			st.New[local.uid] = true
			continue
		}
		if err := s.Delete(local); err != nil {
			return res, err
		}
		res.Deleted++
	}
	for uid := range st.New { // made here
		local := s.byUID(uid)
		if local == nil {
			delete(st.New, uid)
			continue
		}
		p := path.Join(ab, uid+".vcf")
		sc := &syncedCard{UID: uid}
		if err := s.upload(ctx, acct, p, local, sc, "", "*"); err != nil {
			errs = append(errs, err)
			continue
		}
		st.Cards[p] = sc
		delete(st.New, uid)
		res.Uploaded++
	}
	if err := s.saveState(acct.Name, st); err != nil {
		return res, err
	}
	return res, errors.Join(errs...)
}

// upload puts the card of the given contact at the given path of the
// server of the given account, with the given conditions, setting the
// ETag and the hash of the synced card.
func (s *Store) upload(ctx context.Context, acct *Account, p string, c *Contact, sc *syncedCard, ifMatch, ifNoneMatch webdav.ConditionalMatch) error {
	card := c.toCard()
	ao, err := acct.Client.PutAddressObject(ctx, p, card, &carddav.PutAddressObjectOptions{IfMatch: ifMatch, IfNoneMatch: ifNoneMatch})
	if err != nil {
		return fmt.Errorf("saving the contact %q on the server: %w", c.Name, err)
	}
	etag := ao.ETag
	if etag == "" { // not returned by some servers
		if got, err := acct.Client.GetAddressObject(ctx, p); err == nil {
			etag = got.ETag
		}
	}
	sc.ETag, sc.Hash = etag, c.hash()
	return nil
}

// multiGetBatch is the number of cards gotten in each request.
var multiGetBatch = 100

// listCards returns the cards of the given address book, with their paths
// and ETags, and the cards themselves of those that are new or changed since
// the sync of the given state (by their ETags), gotten in batches: the
// others have nil cards. It lists them with PROPFIND (see [Account.etags])
// and gets them with addressbook-multiget, which all of the servers have,
// as Google does not have addressbook-query (its "invalid argument").
func (acct *Account) listCards(ctx context.Context, ab string, st *syncState) ([]carddav.AddressObject, error) {
	c := acct.Client
	etags, err := acct.etags(ctx, ab)
	if err != nil {
		return nil, err
	}
	var res []carddav.AddressObject
	var need []string
	for p, etag := range etags {
		res = append(res, carddav.AddressObject{Path: p, ETag: etag})
		if sc := st.Cards[p]; sc == nil || sc.ETag != etag {
			need = append(need, p)
		}
	}
	slices.SortFunc(res, func(a, b carddav.AddressObject) int { return strings.Compare(a.Path, b.Path) })
	slices.Sort(need)
	cards := map[string]carddav.AddressObject{}
	for len(need) > 0 {
		batch := need[:min(multiGetBatch, len(need))]
		need = need[len(batch):]
		// all of the card, without allprop, which Google refuses as an
		// invalid argument
		aos, err := c.MultiGetAddressBook(ctx, ab, &carddav.AddressBookMultiGet{Paths: batch})
		if err != nil {
			return nil, err
		}
		for _, ao := range aos {
			cards[ao.Path] = ao
		}
	}
	for i := range res {
		if ao, ok := cards[res[i].Path]; ok {
			res[i].Card = ao.Card
			if ao.ETag != "" {
				res[i].ETag = ao.ETag
			}
		}
	}
	return res, nil
}

// etags returns the ETags of the cards of the given address book, by their
// paths, with a PROPFIND of only them (and their types): the listing of the
// WebDAV client needs their content lengths, which Google does not give.
func (acct *Account) etags(ctx context.Context, ab string) (map[string]string, error) {
	base, err := url.Parse(acct.server)
	if err != nil {
		return nil, err
	}
	u := base.ResolveReference(&url.URL{Path: ab})
	body := `<?xml version="1.0" encoding="utf-8"?>` +
		`<d:propfind xmlns:d="DAV:"><d:prop><d:getetag/><d:resourcetype/></d:prop></d:propfind>`
	req, err := http.NewRequestWithContext(ctx, "PROPFIND", u.String(), strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Depth", "1")
	req.Header.Set("Content-Type", "application/xml; charset=utf-8")
	resp, err := acct.errors.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusMultiStatus {
		return nil, fmt.Errorf("listing the contacts: %s", resp.Status)
	}
	var ms struct {
		Responses []struct {
			Href      string `xml:"href"`
			Propstats []struct {
				Status string `xml:"status"`
				Prop   struct {
					ETag         string `xml:"getetag"`
					ResourceType struct {
						Collection *struct{} `xml:"collection"`
					} `xml:"resourcetype"`
				} `xml:"prop"`
			} `xml:"propstat"`
		} `xml:"response"`
	}
	if err := xml.NewDecoder(resp.Body).Decode(&ms); err != nil {
		return nil, fmt.Errorf("listing the contacts: %w", err)
	}
	res := map[string]string{}
	for _, r := range ms.Responses {
		hu, err := url.Parse(r.Href)
		if err != nil {
			continue
		}
		etag, dir := "", false
		for _, ps := range r.Propstats {
			if !strings.Contains(ps.Status, " 200") {
				continue
			}
			if ps.Prop.ETag != "" {
				etag = ps.Prop.ETag
			}
			if ps.Prop.ResourceType.Collection != nil {
				dir = true
			}
		}
		if dir || strings.TrimSuffix(hu.Path, "/") == strings.TrimSuffix(ab, "/") {
			continue // the address book itself
		}
		res[hu.Path] = unquoteETag(etag)
	}
	return res, nil
}

// unquoteETag returns the given ETag without its quotes (and W/ of a weak
// one), as the CardDAV client has them.
func unquoteETag(etag string) string {
	etag = strings.TrimPrefix(strings.TrimSpace(etag), "W/")
	if s, err := strconv.Unquote(etag); err == nil {
		return s
	}
	return etag
}

// etagMatch returns the condition of a request matching the given ETag,
// as the client has them without their quotes, or none for "".
func etagMatch(etag string) webdav.ConditionalMatch {
	if etag == "" {
		return ""
	}
	return webdav.ConditionalMatch(strconv.Quote(etag))
}

// findAddressBook returns the path of the first address book of the user
// of the given client.
func findAddressBook(ctx context.Context, c *carddav.Client) (string, error) {
	principal, err := c.FindCurrentUserPrincipal(ctx)
	if err != nil {
		return "", fmt.Errorf("finding the user: %w", err)
	}
	home, err := c.FindAddressBookHomeSet(ctx, principal)
	if err != nil {
		return "", fmt.Errorf("finding the address books: %w", err)
	}
	abs, err := c.FindAddressBooks(ctx, home)
	if err != nil {
		return "", fmt.Errorf("finding the address books: %w", err)
	}
	if len(abs) == 0 {
		return "", errors.New("no address books")
	}
	return abs[0].Path, nil
}
