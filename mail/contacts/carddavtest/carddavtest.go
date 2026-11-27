// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package carddavtest provides a CardDAV server with one address book in
// memory, for testing the syncing of contacts.
package carddavtest

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"

	"github.com/emersion/go-vcard"
	"github.com/emersion/go-webdav"
	"github.com/emersion/go-webdav/carddav"
)

// Server is a CardDAV server with one address book in memory, at
// [AddressBook], whose user is at /me/.
type Server struct {
	*httptest.Server

	mu    sync.Mutex
	cards map[string]carddav.AddressObject
	etag  int
}

// AddressBook is the path of the address book of the server.
const AddressBook = "/me/contacts/default/"

// New returns a new running server, which must be closed.
func New() *Server {
	s := &Server{cards: map[string]carddav.AddressObject{}}
	s.Server = httptest.NewServer(&carddav.Handler{Backend: (*backend)(s)})
	return s
}

// Put puts a card with the given UID, full name, and email address in the
// address book, as if it were changed on the server.
func (s *Server) Put(uid, name, email string) {
	card := vcard.Card{}
	card.SetValue(vcard.FieldVersion, "3.0")
	card.SetValue(vcard.FieldUID, uid)
	card.SetValue(vcard.FieldFormattedName, name)
	card.SetValue(vcard.FieldEmail, email)
	(*backend)(s).PutAddressObject(context.Background(), AddressBook+uid+".vcf", card, nil)
}

// Delete deletes the card with the given UID, as if on the server.
func (s *Server) Delete(uid string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.cards, AddressBook+uid+".vcf")
}

// Card returns the card with the given path in the address book, or nil.
func (s *Server) Card(uid string) vcard.Card {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ao, ok := s.cards[AddressBook+uid+".vcf"]; ok {
		return ao.Card
	}
	return nil
}

// Names returns the full names of the cards, sorted.
func (s *Server) Names() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var res []string
	for _, ao := range s.cards {
		res = append(res, ao.Card.Value(vcard.FieldFormattedName))
	}
	slices.Sort(res)
	return res
}

// backend is the CardDAV backend of a [Server].
type backend Server

func (be *backend) CurrentUserPrincipal(ctx context.Context) (string, error) { return "/me/", nil }
func (be *backend) AddressBookHomeSetPath(ctx context.Context) (string, error) {
	return "/me/contacts/", nil
}
func (be *backend) ListAddressBooks(ctx context.Context) ([]carddav.AddressBook, error) {
	return []carddav.AddressBook{{Path: AddressBook, Name: "Contacts"}}, nil
}
func (be *backend) GetAddressBook(ctx context.Context, p string) (*carddav.AddressBook, error) {
	return &carddav.AddressBook{Path: AddressBook, Name: "Contacts"}, nil
}
func (be *backend) CreateAddressBook(ctx context.Context, ab *carddav.AddressBook) error { return nil }
func (be *backend) DeleteAddressBook(ctx context.Context, p string) error                { return nil }

func (be *backend) GetAddressObject(ctx context.Context, p string, req *carddav.AddressDataRequest) (*carddav.AddressObject, error) {
	be.mu.Lock()
	defer be.mu.Unlock()
	ao, ok := be.cards[p]
	if !ok {
		return nil, webdav.NewHTTPError(http.StatusNotFound, fmt.Errorf("no %s", p))
	}
	return &ao, nil
}

func (be *backend) ListAddressObjects(ctx context.Context, p string, req *carddav.AddressDataRequest) ([]carddav.AddressObject, error) {
	be.mu.Lock()
	defer be.mu.Unlock()
	var res []carddav.AddressObject
	for _, ao := range be.cards {
		res = append(res, ao)
	}
	return res, nil
}

func (be *backend) QueryAddressObjects(ctx context.Context, p string, q *carddav.AddressBookQuery) ([]carddav.AddressObject, error) {
	return be.ListAddressObjects(ctx, p, &q.DataRequest)
}

func (be *backend) PutAddressObject(ctx context.Context, p string, card vcard.Card, opts *carddav.PutAddressObjectOptions) (*carddav.AddressObject, error) {
	be.mu.Lock()
	defer be.mu.Unlock()
	old, exists := be.cards[p]
	if opts != nil && opts.IfNoneMatch.IsWildcard() && exists {
		return nil, webdav.NewHTTPError(http.StatusPreconditionFailed, fmt.Errorf("%s exists", p))
	}
	if opts != nil && opts.IfMatch.IsSet() {
		if etag, _ := opts.IfMatch.ETag(); !exists || etag != old.ETag {
			return nil, webdav.NewHTTPError(http.StatusPreconditionFailed, fmt.Errorf("%s changed", p))
		}
	}
	be.etag++
	ao := carddav.AddressObject{Path: p, ETag: fmt.Sprint(be.etag), Card: card}
	be.cards[p] = ao
	return &ao, nil
}

func (be *backend) DeleteAddressObject(ctx context.Context, p string) error {
	be.mu.Lock()
	defer be.mu.Unlock()
	delete(be.cards, p)
	return nil
}
