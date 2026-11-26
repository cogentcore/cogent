// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package contacts provides contacts stored as standard vCards (.vcf
// files), with a browser to look them up and edit them, as in the
// Contacts app of the Mac, and the candidates for completing the
// addresses of email messages (see [Match]).
package contacts

//go:generate core generate

import (
	"slices"
	"strings"
	"sync"

	"cogentcore.org/core/core"
	"github.com/emersion/go-vcard"
)

// Contact is a person or organization, as a vCard, with the fields that
// are edited, which are set from the card and back to it when it is
// saved; its other fields, as a photo, are kept as they are.
type Contact struct {

	// Name is the full name, as shown.
	Name string

	// Organization is the company or organization.
	Organization string

	// Title is the job title.
	Title string

	// Emails are the email addresses.
	Emails []Labeled `display:"inline"`

	// Phones are the phone numbers.
	Phones []Labeled `display:"inline"`

	// Addresses are the postal addresses.
	Addresses []PostalAddress `display:"inline"`

	// Birthday is the date of birth, as YYYY-MM-DD (or --MM-DD
	// without the year).
	Birthday string

	// Notes are notes about the contact.
	Notes string `display:"multiline"`

	// uid is the unique ID of the contact (the UID of its card).
	uid string

	// card is the vCard of the contact.
	card vcard.Card

	// isNew is whether the contact was made with [Store.New] and has not
	// been saved yet.
	isNew bool
}

// Labeled is a value with a label, as an email address for "work".
type Labeled struct {

	// Label is the kind of the value, as home or work.
	Label Label

	// Value is the value, as the email address.
	Value string
}

// PostalAddress is a postal address, with a label, as home or work.
type PostalAddress struct {
	Label      Label
	Street     string
	City       string
	Region     string
	PostalCode string
	Country    string
}

// UID returns the unique ID of the contact.
func (c *Contact) UID() string { return c.uid }

// Email returns the first email address of the contact, or "".
func (c *Contact) Email() string {
	if len(c.Emails) == 0 {
		return ""
	}
	return c.Emails[0].Value
}

// fromCard returns a new contact from the given vCard.
func fromCard(card vcard.Card) *Contact {
	c := &Contact{card: card, uid: card.Value(vcard.FieldUID)}
	c.Name = strings.TrimSpace(card.PreferredValue(vcard.FieldFormattedName))
	if n := card.Name(); n != nil && c.Name == "" {
		c.Name = strings.TrimSpace(strings.Join(nonEmpty(n.HonorificPrefix, n.GivenName, n.AdditionalName, n.FamilyName, n.HonorificSuffix), " "))
	}
	org, _, _ := strings.Cut(card.Value(vcard.FieldOrganization), ";") // the units after it
	c.Organization = org
	c.Title = card.Value(vcard.FieldTitle)
	c.Emails = labeledValues(card, vcard.FieldEmail)
	c.Phones = labeledValues(card, vcard.FieldTelephone)
	for _, a := range card.Addresses() {
		c.Addresses = append(c.Addresses, PostalAddress{Label: Label(label(a.Field)), Street: a.StreetAddress,
			City: a.Locality, Region: a.Region, PostalCode: a.PostalCode, Country: a.Country})
	}
	c.Birthday = card.Value(vcard.FieldBirthday)
	c.Notes = card.Value(vcard.FieldNote)
	if c.Name == "" { // as shown: the organization, email, or phone
		switch {
		case c.Organization != "":
			c.Name = c.Organization
		case len(c.Emails) > 0:
			c.Name = c.Emails[0].Value
		case len(c.Phones) > 0:
			c.Name = c.Phones[0].Value
		}
	}
	return c
}

// toCard sets the card of the contact from its fields, keeping the
// others, and returns it.
func (c *Contact) toCard() vcard.Card {
	card := c.card
	if card == nil {
		card = vcard.Card{}
		c.card = card
	}
	if card.Value(vcard.FieldVersion) == "" {
		card.SetValue(vcard.FieldVersion, "3.0")
	}
	card.SetValue(vcard.FieldUID, c.uid)
	card.SetValue(vcard.FieldFormattedName, c.Name)
	if card.Name() == nil { // required in vCard 3.0
		given, family := splitName(c.Name)
		card.SetName(&vcard.Name{GivenName: given, FamilyName: family})
	}
	setValue(card, vcard.FieldOrganization, c.Organization)
	setValue(card, vcard.FieldTitle, c.Title)
	setLabeled(card, vcard.FieldEmail, c.Emails)
	setLabeled(card, vcard.FieldTelephone, c.Phones)
	delete(card, vcard.FieldAddress)
	for _, a := range c.Addresses {
		if a == (PostalAddress{Label: a.Label}) {
			continue // empty
		}
		card.AddAddress(&vcard.Address{Field: labelField(string(a.Label)), StreetAddress: a.Street,
			Locality: a.City, Region: a.Region, PostalCode: a.PostalCode, Country: a.Country})
	}
	setValue(card, vcard.FieldBirthday, c.Birthday)
	setValue(card, vcard.FieldNote, c.Notes)
	return card
}

// standardTypes are the types of vCard fields that are not shown as their
// labels, as they are not about the use of the value.
var standardTypes = map[string]bool{"internet": true, "pref": true, "voice": true, "x400": true}

// label returns the label of the given vCard field: the first type of
// it that is about its use, as home or work.
func label(f *vcard.Field) string {
	if f == nil {
		return ""
	}
	for _, t := range f.Params.Types() {
		if !standardTypes[strings.ToLower(t)] {
			return strings.ToLower(t)
		}
	}
	return ""
}

// labelField returns a new vCard field with the given label as its type.
func labelField(lbl string) *vcard.Field {
	f := &vcard.Field{Params: vcard.Params{}}
	if lbl = strings.TrimSpace(lbl); lbl != "" {
		f.Params.Add(vcard.ParamType, lbl)
	}
	return f
}

// labeledValues returns the values of the given field of the given card,
// with their labels.
func labeledValues(card vcard.Card, k string) []Labeled {
	var res []Labeled
	for _, f := range card[k] {
		if v := strings.TrimSpace(f.Value); v != "" {
			res = append(res, Labeled{Label: Label(label(f)), Value: v})
		}
	}
	return res
}

// setLabeled sets the given field of the given card to the given values.
func setLabeled(card vcard.Card, k string, vals []Labeled) {
	delete(card, k)
	for _, v := range vals {
		if strings.TrimSpace(v.Value) == "" {
			continue
		}
		f := labelField(string(v.Label))
		f.Value = strings.TrimSpace(v.Value)
		card.Add(k, f)
	}
}

// setValue sets the given field of the given card to the given value,
// or removes it for "".
func setValue(card vcard.Card, k, v string) {
	if v == "" {
		delete(card, k)
		return
	}
	card.SetValue(k, v)
}

// splitName splits the given full name into the given and family names:
// the family name is the last word.
func splitName(name string) (given, family string) {
	words := strings.Fields(name)
	if len(words) < 2 {
		return name, ""
	}
	return strings.Join(words[:len(words)-1], " "), words[len(words)-1]
}

// nonEmpty returns the given strings that are not empty.
func nonEmpty(ss ...string) []string {
	var res []string
	for _, s := range ss {
		if s != "" {
			res = append(res, s)
		}
	}
	return res
}

// Label is the label of a value of a contact, as home or work, chosen in
// an editable chooser from the standard ones and those of the contacts
// (see [Labels]), or typed, which adds it to them.
type Label string

func (l Label) Widget() core.Value {
	ch := core.NewChooser().SetEditable(true).SetAllowNew(true)
	for _, lb := range Labels() {
		ch.Items = append(ch.Items, core.ChooserItem{Value: Label(lb), Text: lb})
	}
	return ch
}

// StandardLabels are the labels that are always offered for the values
// of contacts, as well as those of the contacts (see [Labels]).
var StandardLabels = []string{"home", "work", "mobile", "main", "other"}

// stores are the open stores of contacts, whose labels are offered.
var (
	stores   []*Store
	storesMu sync.Mutex
)

// Labels returns the labels offered for the values of contacts: the
// [StandardLabels], and those of the contacts of the open stores, sorted.
func Labels() []string {
	res := slices.Clone(StandardLabels)
	add := func(l Label) {
		if s := strings.TrimSpace(string(l)); s != "" && !slices.Contains(res, s) {
			res = append(res, s)
		}
	}
	storesMu.Lock()
	ss := slices.Clone(stores)
	storesMu.Unlock()
	for _, s := range ss {
		for _, c := range s.Contacts() {
			for _, v := range c.Emails {
				add(v.Label)
			}
			for _, v := range c.Phones {
				add(v.Label)
			}
			for _, a := range c.Addresses {
				add(a.Label)
			}
		}
	}
	slices.Sort(res)
	return res
}
