// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"github.com/cogentcore/go-imap/v2"
	gomail "github.com/emersion/go-message/mail"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// addresses returns the email addresses of the given addresses.
func addresses(as []*gomail.Address) []string {
	var res []string
	for _, ad := range as {
		res = append(res, ad.Address)
	}
	return res
}

func TestParseMailto(t *testing.T) {
	m, err := parseMailto("mailto:bob@x.test,%22Carol%20C%22%20%3Ccarol@x.test%3E?subject=Hi%20there&cc=dan@x.test&bcc=eve@x.test&to=fay@x.test&body=a+b%0Ac")
	require.NoError(t, err)
	assert.Equal(t, []string{"bob@x.test", "carol@x.test", "fay@x.test"}, addresses(m.to))
	assert.Equal(t, "Carol C", m.to[1].Name)
	assert.Equal(t, []string{"dan@x.test"}, addresses(m.cc))
	assert.Equal(t, []string{"eve@x.test"}, addresses(m.bcc))
	assert.Equal(t, "Hi there", m.subject)
	assert.Equal(t, "a+b\nc", m.body, "a + is itself")

	m, err = parseMailto("MAILTO:?subject=x")
	require.NoError(t, err)
	assert.Empty(t, m.to)
	assert.True(t, isMailto("MailTo:a@b.c"))
	assert.False(t, isMailto("https://x.test"))
	_, err = parseMailto("mailto:not an address")
	assert.Error(t, err)
}

// TestReplyMailto tests that a mailto: link in a message opens a reply to
// the message to its address, which can be turned into a reply to all.
func TestReplyMailto(t *testing.T) {
	b, a := newTestApp(t, newIMAP(t))
	var cb *core.Body
	var sm *SendMessage
	core.RenderImage(b, func() {
		require.NoError(t, a.cacheMessages())
	}, core.Locked(b, func() {
		a.readMessage, a.readSingle = messageBySubject(a, "Hello"), true
		a.Update() // read, and so parsed
	}), core.Locked(b, func() {
		hello := a.readMessage
		cb = a.replyMailto("mailto:zed@example.com?body=About%20this", hello)
		require.NotNil(t, cb)
		sm = a.composeMessage
		assert.Equal(t, []string{"zed@example.com"}, addresses(sm.To))
		assert.Empty(t, sm.Cc)
		assert.Equal(t, "Re: Hello", sm.Subject)
		assert.Equal(t, hello.MessageID, sm.inReplyTo, "a reply to the message")
		assert.Contains(t, sm.body, "About this")
		assert.Contains(t, sm.body, "> Hi there", "quoted")
		require.NotNil(t, sm.replyAll)
		core.Locked(cb, func() {
			all := core.ButtonByText(cb.Scene, "Reply all") // in the bottom bar
			require.NotNil(t, all)
			all.Send(events.Click)
			assert.Equal(t, []string{"zed@example.com", "alice@example.com"}, addresses(sm.To),
				"the sender added, not me")
			assert.Nil(t, sm.replyAll)
			assert.True(t, all.IsDisabled(), "used")
			cb.Scene.Close()
		})()
	}))

	// a bad link is an error, not a window
	assert.Nil(t, a.replyMailto("mailto:not an address", messageBySubject(a, "Hello")))
}

// TestAddReplyAll tests the recipients added by a reply to all.
func TestAddReplyAll(t *testing.T) {
	sm := &SendMessage{Sender: MessageSender{From: "me@x.test"}, To: []*gomail.Address{{Address: "zed@x.test"}},
		replyAll: &replyAll{
			to: []*gomail.Address{{Name: "Al", Address: "al@x.test"}, {Address: "ME@x.test"}, {Address: "zed@x.test"}},
			cc: []*gomail.Address{{Address: "bo@x.test"}, {Address: "al@x.test"}},
		}}
	sm.addReplyAll()
	assert.Equal(t, []string{"zed@x.test", "al@x.test"}, addresses(sm.To))
	assert.Equal(t, "Al", sm.To[1].Name)
	assert.Equal(t, []string{"bo@x.test"}, addresses(sm.Cc))
}

// TestHeaderAddresses tests that the header of a message shows all of
// its recipients, each a mailto: link, and text with < as itself.
func TestHeaderAddresses(t *testing.T) {
	b, a := newTestApp(t, nil)
	msg := "From: me@example.com\r\nSubject: Many\r\nContent-Type: text/plain\r\n\r\nHello all.\r\n"
	require.NoError(t, os.MkdirAll(a.messagesDir(testEmail), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(a.messagesDir(testEmail), "many"), []byte(msg), 0o600))
	var to []imap.Address
	for i := range 5 {
		to = append(to, imap.Address{Name: fmt.Sprintf("Person %d", i), Mailbox: fmt.Sprintf("p%d", i), Host: "example.com"})
	}
	cm := &CacheMessage{Filename: "many", Envelope: imap.Envelope{Subject: "a <b> & c", Date: time.Now(), To: to,
		Cc: []imap.Address{{Mailbox: "cc", Host: "example.com"}}}}
	dmf := NewDisplayMessageFrame(b)
	dmf.SetMessage(cm)
	field := func(name string) *core.Text {
		for w := range dmf.AllWidgets() {
			if tx, ok := w.(*core.Text); ok && tx.Name == name {
				return tx
			}
		}
		return nil
	}
	core.RenderImage(b, core.Locked(b, func() {
		tx := field("To")
		require.NotNil(t, tx)
		require.Len(t, tx.Links, 5, "all of them")
		for i, l := range tx.Links {
			assert.Equal(t, fmt.Sprintf("mailto:p%d@example.com", i), l.URL)
			assert.Equal(t, fmt.Sprintf("Person %d <p%d@example.com>", i, i), l.Label)
		}
		cc := field("Cc")
		require.NotNil(t, cc)
		require.Len(t, cc.Links, 1)
		assert.Equal(t, "cc@example.com", cc.Links[0].Label)
		subject := field("Subject")
		require.NotNil(t, subject)
		assert.Empty(t, subject.Links)
		assert.Equal(t, "a <b> & c", string(subject.PaintText().Source.Join()), "not markup")
	}))
}

// TestLinkAddress tests the addresses of mailto: links pasted into an
// address field, and of other text.
func TestLinkAddress(t *testing.T) {
	assert.Equal(t, "bob@x.test", linkAddress("mailto:bob@x.test"))
	assert.Equal(t, "bob@x.test", linkAddress(" MailTo:bob@x.test?subject=Hi%20there "))
	assert.Equal(t, "carol@x.test", linkAddress("mailto:%22Carol%22%20%3Ccarol@x.test%3E,dan@x.test"), "the first one")
	assert.Equal(t, "bob@x.test", linkAddress("bob@x.test\n"))
	assert.Equal(t, "not an address", linkAddress("mailto:not an address"), "the rest, if not an address")
}

// TestPasteAddress tests pasting a mailto: link into an address field.
func TestPasteAddress(t *testing.T) {
	b := core.NewBody()
	at := core.NewValue(&gomail.Address{}, "").(*AddressTextField)
	b.AddChild(at)
	core.RenderImage(b, core.Locked(b, func() {
		at.Clipboard().Write(mimedata.NewText("mailto:zed@example.com?subject=Hello"))
		at.HandleEvent(events.NewKey(events.KeyChord, 0, key.CodeV, key.Meta))
		assert.Equal(t, "zed@example.com", at.Text())
	}))
}

// TestMessageHTML tests that the style of the body of the HTML of a message
// is not used, as in other mail programs.
func TestMessageHTML(t *testing.T) {
	assert.Equal(t, `<html><body dir="ltr" bgcolor="#ffffff"><p style="color:red">Hi</p></body></html>`,
		messageHTML(`<html><body style="max-width:420px;margin:0 auto;" dir="ltr" bgcolor="#ffffff"><p style="color:red">Hi</p></body></html>`))
	assert.Equal(t, `<BODY>x`, messageHTML(`<BODY STYLE='margin:0'>x`))
	assert.Equal(t, `<p>no body</p>`, messageHTML(`<p>no body</p>`))
}
