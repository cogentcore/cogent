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

	// Score is how likely it is to be wanted, as how often and recently
	// mail was sent to it; the contacts have [ContactScore] more.
	Score float64
}

// ContactScore is the score that the candidates of the contacts have
// more than those only from the mail, so that they come first.
const ContactScore = 1000

// Candidates returns the candidates of the email addresses of the contacts.
func (s *Store) Candidates() []Candidate {
	var res []Candidate
	for _, c := range s.Contacts() {
		for _, e := range c.Emails {
			res = append(res, Candidate{Name: c.Name, Address: e.Value, Score: ContactScore})
		}
	}
	return res
}

// Merge returns the given candidates with one for each address, ignoring
// case: the one with a name, preferring those of the contacts, with the
// sum of their scores.
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
			if m.Name == "" || (c.Score >= ContactScore && m.Score < ContactScore && c.Name != "") {
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
// ignoring case; the highest scores first.
func Match(cands []Candidate, text string, n int) []Candidate {
	words := strings.Fields(strings.ToLower(text))
	if len(words) == 0 {
		return nil
	}
	var res []Candidate
	for _, c := range cands {
		if matchWords(words, c.Name+" "+c.Address) {
			res = append(res, c)
		}
	}
	slices.SortStableFunc(res, func(a, b Candidate) int { return cmp.Compare(b.Score, a.Score) })
	if len(res) > n {
		res = res[:n]
	}
	return res
}
