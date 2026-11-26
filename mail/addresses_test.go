// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"testing"

	"cogentcore.org/cogent/mail/contacts"
	"cogentcore.org/core/core"
	"cogentcore.org/core/text/parse/complete"
	gomail "github.com/emersion/go-message/mail"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAddressCompletion tests completing the addresses of messages from
// the contacts and from the mail: those mail was sent to first, and the
// name of the address set by the completion.
func TestAddressCompletion(t *testing.T) {
	b, a := newTestApp(t, newIMAP(t))
	core.RenderImage(b, func() {
		require.NoError(t, a.cacheMessages())
	}, core.Locked(b, func() {
		s := a.contactsStore()
		require.NotNil(t, s)
		c := s.New()
		c.Name = "Alicia Keys"
		c.Emails = []contacts.Labeled{{Value: "alicia@music.test"}}
		require.NoError(t, s.Save(c))

		texts := func(md complete.Matches) []string {
			var res []string
			for _, c := range md.Matches {
				res = append(res, c.Text)
			}
			return res
		}
		md := a.addressMatches("ali")
		assert.Equal(t, "ali", md.Seed)
		assert.Equal(t, []string{"Alicia Keys <alicia@music.test>", "Alice <alice@example.com>"}, texts(md),
			"the contact, then the one sent to")
		assert.Equal(t, []string{"Bob <bob@example.com>"}, texts(a.addressMatches("bo")), "sent to")
		assert.Equal(t, []string{"Carol <carol@example.com>"}, texts(a.addressMatches("carol")), "a sender")
		assert.Empty(t, a.addressMatches("me@").Matches, "not the account")

		at := core.NewValue(&gomail.Address{}, "").(*AddressTextField)
		ed := completeAddress(at, "ali", 3, md.Matches[1])
		assert.Equal(t, "alice@example.com", ed.NewText, "the address in the field")
		assert.Equal(t, "Alice", at.Address.Name, "with the name")
		at.SetText("alice@example.com")
		require.NoError(t, at.Validator())
		assert.Equal(t, "Alice", at.Address.Name, "kept for the same address")
		at.SetText("zed@example.com")
		require.NoError(t, at.Validator())
		assert.Empty(t, at.Address.Name, "not for another one")
		assert.Equal(t, "zed@example.com", at.Address.Address)
	}))
}
