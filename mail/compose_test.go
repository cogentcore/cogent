// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"strings"
	"testing"
	"time"

	"cogentcore.org/cogent/mail/contacts"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
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

// TestSignatureDash tests that a line of the signature that would be a
// list of one item in markdown, as "- Randy", is plain text in the HTML of
// the message, while the lists of the message itself are still lists.
func TestSignatureDash(t *testing.T) {
	old := Settings.Signatures
	t.Cleanup(func() { Settings.Signatures = old })
	Settings.Signatures = []Signature{{Name: "Short", Text: "- Randy\n"}, {Name: "Long", Text: "Randy\n- a\n- b\n"}}
	html := func(sig SignatureName, body string) string {
		d := &draft{Signature: sig, Body: body, Date: time.Now(),
			From: []*gomail.Address{{Address: testEmail}}, To: []*gomail.Address{{Address: "a@example.com"}}}
		msg, err := buildMessage(d, false)
		require.NoError(t, err)
		return string(msg)
	}
	sig := signatureNamed("Short")
	msg := html("Short", "Items:\n\n- one\n- two\n\n"+sig)
	assert.Contains(t, msg, "<li>one</li>", "the lists of the message")
	assert.NotContains(t, msg, "<li>Randy</li>")
	assert.Contains(t, msg, "<p>- Randy</p>", "the signature as written")
	assert.Contains(t, msg, "\r\n- Randy", "the plain text as written")
	msg = html("Long", "Hi\n\n"+signatureNamed("Long"))
	assert.Contains(t, msg, "<li>a</li>", "a list of more than one item in a signature")
	assert.Equal(t, "Randy  \n\\- x  \nHi", escapeLoneItems("Randy  \n- x  \nHi"))
	assert.Equal(t, "  \\* x", escapeLoneItems("  * x"))
	assert.Equal(t, "-- \nMe", escapeLoneItems("-- \nMe"), "the delimiter is not an item")
}

// TestComposeQuoteButtons tests the buttons for quoting the text of a
// message being written, which are at the end of the line of the sender,
// after its signature, and quote the line of the cursor.
func TestComposeQuoteButtons(t *testing.T) {
	b, a := newTestApp(t, nil)
	Settings.Accounts = gmailAccounts(testEmail)
	var cb *core.Body
	core.RenderImage(b, core.Locked(b, func() {
		a.composeMessage = &SendMessage{To: []*gomail.Address{{}}, body: "Hello"}
		cb = a.compose("Compose")
	}), core.Locked(b, func() {
		core.Locked(cb, func() {
			var sender core.Widget
			var ed *textcore.Editor
			for w := range cb.AllWidgets() {
				if strings.HasPrefix(w.AsTree().Name, "value-Sender-") {
					sender = w
				}
				if e, ok := w.(*textcore.Editor); ok {
					ed = e
				}
			}
			require.NotNil(t, sender, "the line of the sender")
			require.NotNil(t, ed)
			var names []string
			for _, c := range sender.AsTree().Children {
				names = append(names, c.AsTree().Name)
			}
			require.GreaterOrEqual(t, len(names), 2)
			assert.Equal(t, []string{"increase-quote", "decrease-quote"}, names[len(names)-2:], "at the end, after the signature: %v", names)
			assert.Contains(t, strings.Join(names, " "), "value-Signature", "on the same line")
			sg := sender.AsWidget().Geom
			assert.LessOrEqual(t, sg.Pos.Total.X+sg.Size.Actual.Total.X, cb.Scene.Geom.Size.Actual.Total.X, "fits in the window")
			inc := sender.AsTree().ChildByName("increase-quote").(*core.Button)
			dec := sender.AsTree().ChildByName("decrease-quote").(*core.Button)

			inc.Send(events.Click)
			assert.True(t, strings.HasPrefix(ed.Lines.String(), "> Hello"), ed.Lines.String())
			dec.Send(events.Click)
			assert.True(t, strings.HasPrefix(ed.Lines.String(), "Hello"), ed.Lines.String())
			cb.Scene.Close()
		})()
	}))
}
