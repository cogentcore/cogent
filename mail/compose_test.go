// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"strings"
	"testing"

	"cogentcore.org/cogent/mail/contacts"
	"cogentcore.org/core/core"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/text/textcore"
	gomail "github.com/emersion/go-message/mail"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestComposeTabAndSender tests the compose window: Tab goes from To to the
// Subject, skipping Cc, Bcc, Attachments, and From, which is a chooser of
// the accounts, last.
func TestComposeTabAndSender(t *testing.T) {
	b, a := newTestApp(t, nil)
	Settings.Accounts = gmailAccounts(testEmail, "other@example.com")
	var cb *core.Body
	core.RenderImage(b, core.Locked(b, func() {
		a.composeMessage = &SendMessage{To: []*gomail.Address{{}}}
		cb = a.compose("Compose")
	}), core.Locked(b, func() {
		core.Locked(cb, func() {
			var fields []string
			var to, subject core.Widget
			var from *core.Chooser
			for w := range cb.AllWidgets() {
				name := w.AsTree().Name
				if strings.HasPrefix(name, "value-") {
					fields = append(fields, strings.Split(name, "-")[1])
				}
				switch {
				case strings.HasPrefix(name, "value-To-"):
					for cw := range w.AsWidget().AllWidgets() {
						if _, ok := cw.(*AddressTextField); ok && to == nil {
							to = cw
						}
					}
				case strings.HasPrefix(name, "value-Subject-"):
					subject = w
				case strings.HasPrefix(name, "value-From-"):
					from, _ = w.(*core.Chooser)
				}
			}
			assert.Equal(t, []string{"From", "Signature"}, fields[len(fields)-2:], "last, side by side")
			require.NotNil(t, from, "a chooser")
			assert.Len(t, from.Items, 2, "of the accounts")
			require.NotNil(t, to)
			require.NotNil(t, subject)
			to.AsWidget().SetFocus()
			cb.Events().FocusNextFrom(to)
			assert.True(t, subject.AsWidget().StateIs(states.Focused), "from To to the Subject")
			cb.Scene.Close()
		})()
	}))
}

// TestSwitchSender tests changing the sender of a message being written,
// which changes its signature and its automatic Bcc.
func TestSwitchSender(t *testing.T) {
	old := *Settings
	t.Cleanup(func() { *Settings = old })
	Settings.Signatures = []Signature{{Name: "Work", Text: "Dr. Me\n"}, {Name: "Home", Text: "- Me\n"}}
	Settings.DefaultSignatures = map[string]SignatureName{"work@x.test": "Work", "home@x.test": "Home"}
	Settings.AutoBCC = true
	b := core.NewBody()
	ed := textcore.NewEditor(b)
	sm := &SendMessage{Sender: MessageSender{From: "work@x.test", Signature: "Work"}, Bcc: []*gomail.Address{{Address: "work@x.test"}}}
	sm.body = "Hello\n\n" + signatureText("work@x.test") + "\n> quoted\n"
	ed.Lines.SetString(sm.body)
	switchSender(sm, ed, "work@x.test", "home@x.test")
	assert.Contains(t, sm.body, "- Me")
	assert.NotContains(t, sm.body, "Dr. Me", "the old signature replaced")
	assert.Contains(t, sm.body, "> quoted")
	assert.Equal(t, sm.body, ed.Lines.String(), "in the editor")
	assert.Equal(t, "home@x.test", sm.Bcc[0].Address, "the Bcc to the sender")
	assert.Equal(t, SignatureName("Home"), sm.Sender.Signature, "the default one of the account")

	// choosing another signature
	switchSignature(sm, ed, "Home", "Work")
	assert.Contains(t, sm.body, "Dr. Me")
	assert.NotContains(t, sm.body, "- Me")
	switchSignature(sm, ed, "Work", "")
	assert.NotContains(t, sm.body, "Dr. Me", "none")
	assert.Contains(t, sm.body, "> quoted")
}

// TestSenderName tests the names of the senders of messages: in the
// chooser of the account, and in the From of the messages sent: that of the
// contact with its address, or else the name in the messages sent from it.
func TestSenderName(t *testing.T) {
	b, a := newTestApp(t, newIMAP(t))
	core.RenderImage(b, func() {
		require.NoError(t, a.cacheMessages())
	}, core.Locked(b, func() {
		assert.Equal(t, "Me", a.senderName(testEmail), "in the messages sent from it")
		ch := SenderAccount("").Widget().(*core.Chooser)
		require.Len(t, ch.Items, 1)
		assert.Equal(t, "Me <me@example.com>", ch.Items[0].Text, "in the chooser")

		sm := &SendMessage{Sender: MessageSender{From: SenderAccount(testEmail)}, To: []*gomail.Address{{Address: "zed@example.com"}}, Subject: "Hi"}
		require.NoError(t, a.saveDraft(sm))
		msg, err := buildMessage(sm.draft, false)
		require.NoError(t, err)
		assert.Regexp(t, `From: "?Me"? <me@example.com>`, string(msg), "in the message")

		// a contact with the address: authoritative
		cs := a.contactsStore()
		require.NotNil(t, cs)
		c := cs.New()
		c.Name = "Dr. Me"
		c.Emails = []contacts.Labeled{{Value: "ME@example.com"}}
		require.NoError(t, cs.Save(c))
		assert.Equal(t, "Dr. Me", a.senderName(testEmail), "the name of the contact")
		assert.False(t, a.isMe("me@work.test"))

		// My card: the name from any address, all of whose addresses are mine
		mine := cs.New()
		mine.Name = "Randy"
		mine.Emails = []contacts.Labeled{{Value: "me@work.test"}}
		require.NoError(t, cs.Save(mine))
		cs.SetMyCard(mine)
		t.Cleanup(func() { cs.SetMyCard(nil) })
		assert.Equal(t, "Randy", a.senderName(testEmail), "the name of my card")
		assert.True(t, a.isMe("Me@Work.test"), "an address of my card")
		assert.True(t, a.isMe(testEmail), "an account")
		assert.False(t, a.isMe("zed@example.com"))

		// not a recipient of a reply to all
		sm = &SendMessage{Sender: MessageSender{From: SenderAccount(testEmail)},
			replyAll: &replyAll{to: []*gomail.Address{{Address: "me@work.test"}, {Address: "zed@example.com"}}}}
		sm.addReplyAll()
		require.Len(t, sm.To, 1)
		assert.Equal(t, "zed@example.com", sm.To[0].Address)
	}))
}

// TestMessagesOnly tests that the Drafts, Sent, and Trash show each of
// their messages on its own, not their conversations.
func TestMessagesOnly(t *testing.T) {
	b, a := newTestApp(t, newIMAP(t))
	core.RenderImage(b, func() {
		require.NoError(t, a.cacheMessages())
	}, core.Locked(b, func() {
		assert.False(t, a.flatList(), "conversations by default")
		for _, l := range []string{"[Gmail]/Drafts", "[Gmail]/Sent Mail", "[Gmail]/Trash", "Drafts", outboxLabel} {
			assert.True(t, messagesOnly(l), l)
		}
		assert.False(t, messagesOnly("INBOX"))
		hello := messageBySubject(a, "Hello")
		hello.Labels = append(hello.Labels, Label{Name: "[Gmail]/Trash"}) // of a conversation
		a.showMailbox(testEmail, "[Gmail]/Trash")
		assert.True(t, a.flatList())
		require.NotEmpty(t, a.listRows)
		for _, mr := range a.listRows {
			assert.False(t, mr.conversation, "each message on its own")
			assert.True(t, a.hasLabel(mr.message), "only those in the mailbox")
		}
	}))
}
