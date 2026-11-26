// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package contacts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// appleExport is two contacts as exported by Contacts on the Mac.
const appleExport = "BEGIN:VCARD\r\nVERSION:3.0\r\nPRODID:-//Apple Inc.//macOS 15.0//EN\r\n" +
	"N:Munakata;Yuko;;;\r\nFN:Yuko Munakata\r\nORG:UC Davis;Psychology;\r\nTITLE:Professor\r\n" +
	"item1.EMAIL;type=INTERNET;type=pref:ymunakata@ucdavis.edu\r\nitem1.X-ABLabel:_$!<Work>!$_\r\n" +
	"EMAIL;type=INTERNET;type=HOME:yuko@example.com\r\nTEL;type=CELL;type=VOICE;type=pref:+1 530 555 0100\r\n" +
	"ADR;type=HOME;type=pref:;;1 Main St;Davis;CA;95616;USA\r\nBDAY:1968-02-03\r\n" +
	"PHOTO;ENCODING=b;TYPE=JPEG:AAAA\r\nUID:A1B2C3D4-0000-1111-2222-333344445555\r\nEND:VCARD\r\n" +
	"BEGIN:VCARD\r\nVERSION:3.0\r\nN:;;;;\r\nFN:Acme Support\r\nEMAIL;type=INTERNET:help@acme.test\r\n" +
	"UID:acme-1\r\nEND:VCARD\r\n"

func TestStore(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	require.NoError(t, err)
	n, err := s.Import(strings.NewReader(appleExport))
	require.NoError(t, err)
	assert.Equal(t, 2, n)
	cs := s.Contacts()
	require.Len(t, cs, 2)
	assert.Equal(t, "Acme Support", cs[0].Name, "sorted by name")
	yuko := cs[1]
	assert.Equal(t, "Yuko Munakata", yuko.Name)
	assert.Equal(t, "UC Davis", yuko.Organization)
	assert.Equal(t, "Professor", yuko.Title)
	assert.Equal(t, []Labeled{{"", "ymunakata@ucdavis.edu"}, {"home", "yuko@example.com"}}, yuko.Emails)
	assert.Equal(t, []Labeled{{"cell", "+1 530 555 0100"}}, yuko.Phones)
	require.Len(t, yuko.Addresses, 1)
	assert.Equal(t, PostalAddress{"home", "1 Main St", "Davis", "CA", "95616", "USA"}, yuko.Addresses[0])
	assert.Equal(t, "1 Main St", yuko.Addresses[0].Label(), "its first line")
	assert.Equal(t, "Davis, CA", PostalAddress{Street: " \n", City: "Davis", Region: "CA"}.Label())
	assert.Equal(t, "Apt 2", PostalAddress{Street: "Apt 2\n1 Main St"}.Label())
	assert.Equal(t, "1968-02-03", yuko.Birthday)

	// reloaded from the files, one per contact
	files, _ := filepath.Glob(filepath.Join(dir, "*.vcf"))
	assert.Len(t, files, 2)
	s2, err := Open(dir)
	require.NoError(t, err)
	require.Len(t, s2.Contacts(), 2)
	y2 := s2.Contacts()[1]
	assert.Equal(t, yuko.Emails, y2.Emails)
	assert.Equal(t, "A1B2C3D4-0000-1111-2222-333344445555", y2.UID())

	// edited: the other fields of the card are kept
	y2.Notes = "Met at CNS"
	y2.Phones = append(y2.Phones, Labeled{"work", "555-0101"})
	require.NoError(t, s2.Save(y2))
	data, err := os.ReadFile(s2.file(y2))
	require.NoError(t, err)
	assert.Contains(t, string(data), "PHOTO", "kept")
	assert.Contains(t, string(data), "Met at CNS")
	s3, _ := Open(dir)
	assert.Equal(t, "Met at CNS", s3.Contacts()[1].Notes)
	assert.Len(t, s3.Contacts()[1].Phones, 2)

	// imported again: replaced, not added
	_, err = s3.Import(strings.NewReader(appleExport))
	require.NoError(t, err)
	assert.Len(t, s3.Contacts(), 2)

	// a new one, and deleted
	c := s3.New()
	c.Name = "Zed Zee"
	c.Emails = []Labeled{{"", "zed@example.com"}}
	require.NoError(t, s3.Save(c))
	assert.Len(t, s3.Contacts(), 3)
	assert.Len(t, s3.Search("zee"), 1)
	assert.Len(t, s3.Search("example"), 2, "by the email addresses")
	assert.Len(t, s3.Search("davis"), 1, "by the organization")
	assert.Len(t, s3.Search(""), 3)
	require.NoError(t, s3.Delete(c))
	assert.Len(t, s3.Contacts(), 2)
	assert.NoFileExists(t, s3.file(c))
}

func TestMatch(t *testing.T) {
	s, err := Open(t.TempDir())
	require.NoError(t, err)
	_, err = s.Import(strings.NewReader(appleExport))
	require.NoError(t, err)
	// in the mail: the second address of the contact used the most, and
	// another name for it
	mail := []Candidate{{Name: "YUKO M", Address: "YUKO@example.com", Score: 50},
		{Name: "Yuko M", Address: "ymunakata@ucdavis.edu", Score: 5},
		{Name: "Yuri Gagarin", Address: "yuri@space.test", Score: 500}, {Address: "help@acme.test", Score: 1}}
	cands := Merge(s.Candidates(), mail)
	assert.Len(t, cands, 4, "one per address")
	m := Match(cands, "yu", 10)
	require.Len(t, m, 3)
	assert.Equal(t, "ymunakata@ucdavis.edu", m[0].Address, "the first address of the contact first, as the default")
	assert.Equal(t, "yuko@example.com", m[1].Address, "then its second, however much it is used")
	assert.Equal(t, "Yuko Munakata", m[1].Name, "with the name of the contact")
	assert.Equal(t, "yuri@space.test", m[2].Address, "then the others, however much they are used")
	assert.Equal(t, float64(55), m[0].Score+m[1].Score, "the scores of the mail kept")
	assert.Len(t, Match(cands, "yuko muna", 10), 2, "all of the words: both of her addresses")
	assert.Len(t, Match(cands, "yuko gag", 10), 0)
	assert.Len(t, Match(cands, "ucdav", 10), 1, "by the address")
	assert.Len(t, Match(cands, "yu", 1), 1, "at most n")
	assert.Empty(t, Match(cands, " ", 10))

	// another contact matching, used more in the mail, comes first
	c := s.New()
	c.Name = "Yusuf Ali"
	c.Emails = []Labeled{{Value: "yusuf@x.test"}}
	require.NoError(t, s.Save(c))
	cands = Merge(s.Candidates(), mail, []Candidate{{Address: "yusuf@x.test", Score: 100}})
	m = Match(cands, "yu", 10)
	require.Len(t, m, 4)
	assert.Equal(t, []string{"yusuf@x.test", "ymunakata@ucdavis.edu", "yuko@example.com", "yuri@space.test"},
		[]string{m[0].Address, m[1].Address, m[2].Address, m[3].Address})
}

// TestNoName tests the names of the contacts without one.
func TestNoName(t *testing.T) {
	s, err := Open(t.TempDir())
	require.NoError(t, err)
	_, err = s.Import(strings.NewReader("BEGIN:VCARD\r\nVERSION:3.0\r\nN:;;;;\r\nFN:\r\nORG:Acme;\r\nUID:1\r\nEND:VCARD\r\n" +
		"BEGIN:VCARD\r\nVERSION:3.0\r\nN:;;;;\r\nTEL:555-0100\r\nUID:2\r\nEND:VCARD\r\n"))
	require.NoError(t, err)
	assert.Equal(t, []string{"555-0100", "Acme"}, []string{s.Contacts()[0].Name, s.Contacts()[1].Name})
}
