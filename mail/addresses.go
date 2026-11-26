// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"cogentcore.org/cogent/mail/contacts"
	"cogentcore.org/core/core"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/text/parse/complete"
	"github.com/cogentcore/go-imap/v2"
	"github.com/emersion/go-message/mail"
)

// errContacts is the error of the contacts that cannot be opened, whose
// details are logged.
var errContacts = errors.New("the contacts could not be opened")

// The addresses of messages being written are completed from the contacts
// (see [contacts.Store]) and from the addresses in the mail: those that
// mail was sent to from the accounts, which count the most, and those that
// sent mail to them, the more recent the more (see [App.addressCandidates]).

// candidatesMaxAge is how long the candidates from the mail are kept
// before they are found again, as new mail comes.
var candidatesMaxAge = 2 * time.Minute

// contactsStore returns the store of the contacts, opening it if needed:
// the standard one shared with Cogent Contacts, or one of its own in
// testing mode. It returns nil if it cannot be opened.
func (a *App) contactsStore() *contacts.Store {
	if a.contacts != nil {
		return a.contacts
	}
	dir := contacts.DefaultDir()
	if a.testServer != nil || a.dataDirectory != "" { // not the real contacts
		dir = filepath.Join(a.dataDir(), "contacts")
	}
	s, err := contacts.Open(dir)
	if err != nil {
		slog.Error("mail: opening the contacts", "err", err)
		return nil
	}
	s.NewAccount = newContactsAccount()
	a.contacts = s
	return s
}

// addressCandidates returns the candidates for completing the addresses
// of messages: those of the contacts, and those from the mail, which are
// found again after [candidatesMaxAge]. It must be called with the lock
// of the window.
func (a *App) addressCandidates() []contacts.Candidate {
	if a.mailCandidates == nil || time.Since(a.candidatesTime) > candidatesMaxAge {
		a.mailCandidates = a.candidatesFromMail()
		a.candidatesTime = time.Now()
	}
	var fromContacts []contacts.Candidate
	if s := a.contactsStore(); s != nil {
		fromContacts = s.Candidates()
	}
	return contacts.Merge(fromContacts, a.mailCandidates)
}

// candidatesFromMail returns the candidates for completing addresses from
// the mail of all of the accounts: the recipients of the messages sent
// from them, with 3 for each, and the senders of the others, with 1, each
// counting less the older it is, as 1/(1 + its age in years).
func (a *App) candidatesFromMail() []contacts.Candidate {
	now := time.Now()
	mine := func(addr string) bool {
		return slices.ContainsFunc(Settings.Accounts, func(acct string) bool { return strings.EqualFold(acct, addr) })
	}
	var res []contacts.Candidate
	add := func(addrs []imap.Address, weight float64, date time.Time) {
		age := now.Sub(date).Hours() / (24 * 365)
		w := weight / (1 + max(age, 0))
		for _, ad := range addrs {
			if addr := ad.Addr(); addr != "" && !mine(addr) && strings.Contains(addr, "@") {
				res = append(res, contacts.Candidate{Name: ad.Name, Address: addr, Score: w})
			}
		}
	}
	for _, mp := range a.cache {
		for _, cm := range mp {
			if len(cm.From) == 0 || cm.isLocal() {
				continue
			}
			if mine(cm.From[0].Addr()) {
				add(cm.To, 3, cm.Date)
				add(cm.Cc, 3, cm.Date)
				add(cm.Bcc, 3, cm.Date)
			} else {
				add(cm.From, 1, cm.Date)
			}
		}
	}
	return contacts.Merge(res)
}

// addressMatches returns the completions of the given text being typed in
// an address field (see [AddressTextField]).
func (a *App) addressMatches(text string) complete.Matches {
	md := complete.Matches{Seed: text}
	for _, c := range contacts.Match(a.addressCandidates(), text, 20) {
		label := addressText(&mail.Address{Name: c.Name, Address: c.Address})
		icon := icons.Mail
		if c.Score >= contacts.ContactScore {
			icon = icons.Person
		}
		md.Matches = append(md.Matches, complete.Completion{Text: label, Label: label, Icon: icon})
	}
	return md
}

// Contacts opens a window to look up and edit the contacts, from which
// the addresses of the messages being written are completed.
func (a *App) Contacts() { //types:add
	s := a.contactsStore()
	if s == nil {
		core.ErrorSnackbar(a, errContacts)
		return
	}
	if core.RecycleMainWindow(s) {
		return
	}
	b := core.NewBody("Contacts").SetData(s)
	br := contacts.NewBrowser(b).SetStore(s)
	b.AddTopBar(func(bar *core.Frame) {
		core.NewToolbar(bar).Maker(br.MakeToolbar)
	})
	b.RunWindow()
}

// completeAddress returns the edit of an address field for the given
// completion (see [App.addressMatches]), whose text is the address with its
// name (see [addressText]), which replaces all of the given text of the
// field, as the text being completed.
func completeAddress(text string, c complete.Completion) complete.Edit {
	return complete.Edit{NewText: c.Text, ForwardDelete: utf8.RuneCountInString(text)}
}

// addressText returns the given address as it is edited in an address
// field: Name <address>, with the name in quotes if it has characters
// that need them (as a comma), or just the address without a name.
func addressText(ad *mail.Address) string {
	name, addr := strings.TrimSpace(ad.Name), strings.TrimSpace(ad.Address)
	if name == "" || strings.EqualFold(name, addr) {
		return addr
	}
	if strings.ContainsAny(name, `,;:<>@()[]\".`) {
		name = strconv.Quote(name)
	}
	return name + " <" + addr + ">"
}

// parseAddressText returns the address of the given text of an address
// field: Name <address>, or just the address.
func parseAddressText(text string) (*mail.Address, error) {
	text = strings.TrimSpace(text)
	if strings.Contains(text, "<") {
		ad, err := mail.ParseAddress(text)
		if err != nil {
			return nil, fmt.Errorf("invalid email address: %w", err)
		}
		return ad, nil
	}
	if !strings.Contains(text, "@") && !strings.Contains(text, ".") {
		return nil, errors.New("invalid email address")
	}
	return &mail.Address{Address: text}, nil
}
