// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package contacts

import (
	"cmp"
	"slices"
	"strings"
)

// Candidate is an email address offered for completing an address being
// typed, from a contact or from the mail (see [Match]).
type Candidate struct {

	// Name is the name of the person or organization.
	Name string

	// Address is the email address.
	Address string

	// Score is how likely it is to be wanted from the mail, as how often
	// and recently mail was sent to it.
	Score float64

	// Contact is the UID of the contact of the address, if any, which
	// is authoritative: those of the contacts come first, in their order
	// in the contact (see [Match]).
	Contact string

	// Order is the index of the address in its contact.
	Order int
}

// Candidates returns the candidates of the email addresses of the contacts.
func (s *Store) Candidates() []Candidate {
	var res []Candidate
	for _, c := range s.Contacts() {
		for i, e := range c.Emails {
			res = append(res, Candidate{Name: c.Name, Address: e.Value, Contact: c.uid, Order: i})
		}
	}
	return res
}

// Merge returns the given candidates with one for each address, ignoring
// case, with the sum of their scores: that of a contact, with its name, or
// else the first with a name.
func Merge(cands ...[]Candidate) []Candidate {
	byAddr := map[string]*Candidate{}
	var order []string
	for _, cs := range cands {
		for _, c := range cs {
			k := strings.ToLower(strings.TrimSpace(c.Address))
			if k == "" {
				continue
			}
			m := byAddr[k]
			if m == nil {
				cc := c
				byAddr[k] = &cc
				order = append(order, k)
				continue
			}
			switch {
			case c.Contact != "" && m.Contact == "": // the contact is authoritative
				m.Name, m.Address, m.Contact, m.Order = c.Name, c.Address, c.Contact, c.Order
			case m.Name == "" && m.Contact == "":
				m.Name = c.Name
			}
			m.Score += c.Score
		}
	}
	res := make([]Candidate, 0, len(order))
	for _, k := range order {
		res = append(res, *byAddr[k])
	}
	return res
}

// Match returns up to n of the given candidates that match the given text
// being typed: each of its words starts a word of the name or the address,
// ignoring case. Those of the contacts come first, the contact most used in
// the mail first (by the scores of all of its addresses), with its
// addresses in its order, so that its first one is the default; then the
// others, the highest scores first.
func Match(cands []Candidate, text string, n int) []Candidate {
	words := strings.Fields(strings.ToLower(text))
	if len(words) == 0 {
		return nil
	}
	use := map[string]float64{} // of each contact
	for _, c := range cands {
		if c.Contact != "" {
			use[c.Contact] += c.Score
		}
	}
	var res []Candidate
	for _, c := range cands {
		if matchWords(words, c.Name+" "+c.Address) {
			res = append(res, c)
		}
	}
	slices.SortStableFunc(res, func(a, b Candidate) int {
		ac, bc := a.Contact != "", b.Contact != ""
		switch {
		case ac && !bc:
			return -1
		case bc && !ac:
			return 1
		case ac && bc:
			return cmp.Or(cmp.Compare(use[b.Contact], use[a.Contact]),
				cmp.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)),
				cmp.Compare(a.Contact, b.Contact), cmp.Compare(a.Order, b.Order))
		}
		return cmp.Or(cmp.Compare(b.Score, a.Score), cmp.Compare(strings.ToLower(a.Address), strings.ToLower(b.Address)))
	})
	if len(res) > n {
		res = res[:n]
	}
	return res
}
