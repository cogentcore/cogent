// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"testing"
	"time"

	"cogentcore.org/cogent/mail/contacts"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events/key"
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

		// as core.TextField applies a completion: from the start of the
		// seed (the text before the cursor), it deletes ForwardDelete runes
		// and inserts the new text
		apply := func(text string, cursor int, seed string, ed complete.Edit) string {
			rn := []rune(text)
			st := cursor - len([]rune(seed))
			return string(rn[:st]) + ed.NewText + string(rn[st+ed.ForwardDelete:])
		}
		alice := md.Matches[1]
		assert.Equal(t, "Alice <alice@example.com>", apply("ali", 3, "ali", completeAddress("ali", alice)),
			"all of the typed text replaced")
		assert.Equal(t, "Alice <alice@example.com>", apply("  ali", 5, "  ali", completeAddress("  ali", alice)))

		at := core.NewValue(&gomail.Address{}, "").(*AddressTextField)
		at.SetText("Alice <alice@example.com>")
		require.NoError(t, at.Validator())
		assert.Equal(t, gomail.Address{Name: "Alice", Address: "alice@example.com"}, at.Address, "the name and the address")
		at.SetText("zed@example.com")
		require.NoError(t, at.Validator())
		assert.Equal(t, gomail.Address{Address: "zed@example.com"}, at.Address, "just the address")
		at.SetText("Bad <")
		assert.Error(t, at.Validator())
		assert.Equal(t, `"Keys, Alicia" <a@x.test>`, addressText(&gomail.Address{Name: "Keys, Alicia", Address: "a@x.test"}), "quoted")
		ad, err := parseAddressText(`"Keys, Alicia" <a@x.test>`)
		require.NoError(t, err)
		assert.Equal(t, "Keys, Alicia", ad.Name, "back")
		assert.Equal(t, "a@x.test", addressText(&gomail.Address{Name: "a@x.test", Address: "a@x.test"}), "no name like the address")
	}))
}

// TestAddressCompletionTyping tests completing an address as it is typed
// in an address field, accepting the first completion with Return, which
// replaces the typed text with the address and its name.
func TestAddressCompletionTyping(t *testing.T) {
	b, a := newTestApp(t, newIMAP(t))
	var at *AddressTextField
	core.RenderImage(b, func() {
		require.NoError(t, a.cacheMessages())
	}, core.Locked(b, func() {
		at = core.NewValue(&gomail.Address{}, "").(*AddressTextField)
		b.AddChild(at)
		b.Update()
	}), core.Locked(b, func() {
		at.SetFocus()
	}), func() {
		src := b.Scene.RenderWindow().SystemWindow.Events()
		for _, r := range "bo" {
			src.KeyChord(r, key.CodeUnknown, 0)
		}
	}, func() {
		require.Eventually(t, func() bool {
			ok := false
			core.Locked(b, func() { ok = core.TopScene(b) != b.Scene })() // the popup of the completions
			return ok
		}, 5*time.Second, 20*time.Millisecond, "the completions")
		b.Scene.RenderWindow().SystemWindow.Events().KeyChord(0, key.CodeReturnEnter, 0)
	}, core.Locked(b, func() {
		assert.Equal(t, "Bob <bob@example.com>", at.Text(), "the typed text replaced")
		assert.Equal(t, gomail.Address{Name: "Bob", Address: "bob@example.com"}, at.Address)
	}))
}
