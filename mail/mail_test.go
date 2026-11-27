// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"errors"
	"fmt"
	"image"
	"io"
	"math/rand/v2"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/tree"
	"github.com/cogentcore/go-imap/v2"
	"github.com/cogentcore/go-imap/v2/imapclient"
	"github.com/cogentcore/go-imap/v2/imapserver"
	"github.com/cogentcore/go-imap/v2/imapserver/imapmemserver"
	gomail "github.com/emersion/go-message/mail"
	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// testEmail is the email address of the account of the tests.
const testEmail = "me@example.com"

// testMessages are the messages of the tests, by mailbox: a message with
// HTML and an attachment, a reply to it, and a message in another mailbox.
var testMessages = []struct {
	mailbox, text string
	flags         []imap.Flag
}{
	{"INBOX", "From: Alice <alice@example.com>\r\n" +
		"To: me@example.com\r\n" +
		"Subject: Hello\r\n" +
		"Date: Mon, 2 Jan 2006 15:04:05 +0000\r\n" +
		"Message-ID: <1@example.com>\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: multipart/mixed; boundary=b\r\n" +
		"\r\n" +
		"--b\r\n" +
		"Content-Type: text/plain\r\n" +
		"\r\n" +
		"Hi there\r\n" +
		"how are you\r\n" +
		"--b\r\n" +
		"Content-Type: text/html\r\n" +
		"\r\n" +
		"<p>Hi <b>there</b></p>\r\n" +
		"--b\r\n" +
		"Content-Type: text/plain\r\n" +
		"Content-Disposition: attachment; filename=notes.txt\r\n" +
		"\r\n" +
		"some notes\r\n" +
		"--b--\r\n", nil},
	{"INBOX", "From: Me <me@example.com>\r\n" +
		"To: Alice <alice@example.com>, Bob <bob@example.com>\r\n" +
		"Subject: Re: Hello\r\n" +
		"Date: Tue, 3 Jan 2006 15:04:05 +0000\r\n" +
		"Message-ID: <2@example.com>\r\n" +
		"In-Reply-To: <1@example.com>\r\n" +
		"References: <1@example.com>\r\n" +
		"Content-Type: text/plain\r\n" +
		"\r\n" +
		"Fine *thanks*\r\n", []imap.Flag{imap.FlagSeen}},
	{"Work", "From: Carol <carol@example.com>\r\n" +
		"To: me@example.com\r\n" +
		"Subject: Report\r\n" +
		"Date: Wed, 4 Jan 2006 15:04:05 +0000\r\n" +
		"Message-ID: <3@example.com>\r\n" +
		"Content-Type: text/plain\r\n" +
		"\r\n" +
		"The report\r\n", nil},
}

// newIMAP starts an in-memory IMAP server with the mailboxes and messages
// of the tests, and returns a client logged in to it.
func newIMAP(t *testing.T) *imapclient.Client {
	t.Helper()
	mem := imapmemserver.New()
	user := imapmemserver.NewUser(testEmail, "password")
	for _, mb := range []string{"INBOX", "Work", "[Gmail]", "[Gmail]/Trash"} {
		require.NoError(t, user.Create(mb, nil))
	}
	mem.AddUser(user)
	srv := imapserver.New(&imapserver.Options{
		NewSession: func(c *imapserver.Conn) (imapserver.Session, *imapserver.GreetingData, error) {
			return mem.NewSession(), nil, nil
		},
		Caps:         imap.CapSet{imap.CapIMAP4rev1: {}, imap.CapIMAP4rev2: {}},
		InsecureAuth: true,
	})
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	go srv.Serve(ln)
	t.Cleanup(func() { srv.Close() })

	c, err := imapclient.DialInsecure(ln.Addr().String(), nil)
	require.NoError(t, err)
	t.Cleanup(func() { c.Close() })
	require.NoError(t, c.Login(testEmail, "password").Wait())
	for _, m := range testMessages {
		cmd := c.Append(m.mailbox, int64(len(m.text)), &imap.AppendOptions{Flags: m.flags})
		_, err := cmd.Write([]byte(m.text))
		require.NoError(t, err)
		require.NoError(t, cmd.Close())
		_, err = cmd.Wait()
		require.NoError(t, err)
	}
	// the first message is also in Work
	_, err = c.Select("INBOX", nil).Wait()
	require.NoError(t, err)
	_, err = c.Copy(imap.SeqSetNum(1), "Work").Wait()
	require.NoError(t, err)
	err = c.Unselect().Wait()
	require.NoError(t, err)
	return c
}

// newTestApp returns a new body with an [App] for the account of the
// tests, using the given IMAP client, with a temporary home directory for
// its data.
func newTestApp(t *testing.T, c *imapclient.Client) (*core.Body, *App) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	Settings.Accounts = gmailAccounts(testEmail)
	Settings.ShownAccount, Settings.ShownMailbox = "", "" // not that of another test
	t.Cleanup(core.TrackWindows())
	b := core.NewBody()
	a := NewApp(b)
	b.AddTopBar(func(bar *core.Frame) {
		core.NewToolbar(bar).Maker(a.MakeToolbar)
	})
	a.currentEmail = testEmail
	a.dataDirectory = t.TempDir()
	a.imapClient[testEmail] = c
	a.imapMu[testEmail] = &sync.Mutex{}
	a.tokens[testEmail] = oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"}) // signed in
	t.Cleanup(a.closeStores)
	return b, a
}

// messageBySubject returns the cached message with the given subject.
func messageBySubject(a *App, subject string) *CacheMessage {
	for _, cm := range a.cache[testEmail] {
		if cm.Subject == subject {
			return cm
		}
	}
	return nil
}

// labelNames returns the names of the labels of the given message, sorted.
func labelNames(cm *CacheMessage) []string {
	var names []string
	for _, l := range cm.Labels {
		names = append(names, l.Name)
	}
	slices.Sort(names)
	return names
}

// serverFlags returns the flags on the server of the message with the
// given UID in the given mailbox.
func serverFlags(t *testing.T, a *App, mailbox string, uid imap.UID) []imap.Flag {
	a.imapMu[testEmail].Lock()
	defer a.imapMu[testEmail].Unlock()
	c := a.imapClient[testEmail]
	require.NoError(t, a.selectMailbox(c, testEmail, mailbox))
	msgs, err := c.Fetch(imap.UIDSetNum(uid), &imap.FetchOptions{Flags: true}).Collect()
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	return msgs[0].Flags
}

// TestApp tests caching the messages of the mailboxes, showing them as
// conversations, reading them, and marking them as read and unread.
func TestApp(t *testing.T) {
	b, a := newTestApp(t, newIMAP(t))
	var hello *CacheMessage
	core.RenderImage(b, func() {
		require.NoError(t, a.cacheMessages())
		require.Len(t, a.cache[testEmail], 3)
		hello = messageBySubject(a, "Hello")
		require.NotNil(t, hello)
		assert.Equal(t, []string{"INBOX", "Work"}, labelNames(hello), "in both mailboxes, cached once")
		assert.Equal(t, "Alice", hello.From[0].Name)
		assert.Equal(t, []string{"INBOX", "Work", "[Gmail]", "[Gmail]/Trash"}, a.labels[testEmail])
		assert.Equal(t, 2, a.totalMessages[testEmail]["Work"])

		require.NoError(t, a.cacheMessages(), "again: only the flags")
		assert.Len(t, a.cache[testEmail], 3)
	}, core.Locked(b, func() {
		a.Update()
		require.Len(t, a.listCache, 1, "the reply is in the conversation")
		assert.Equal(t, hello, a.listCache[0])
		require.Len(t, hello.replies, 1)
		assert.Equal(t, "Re: Hello", hello.replies[0].Subject)
		assert.Equal(t, 1, a.unreadMessages)
		assert.False(t, hello.isRead())
		assert.Equal(t, hello.replies[0].Date, hello.latestDate())

		// the conversation is one closed row, with both messages in it
		require.Len(t, a.listRows, 1)
		row := a.listRows[0]
		assert.True(t, row.conversation)
		assert.True(t, row.Closed)
		assert.Equal(t, "Alice, You (2)", row.From)
		require.Len(t, row.Children, 2)
		assert.Equal(t, "Re: Hello", row.Children[0].Subject, "the newest first")
		assert.Equal(t, a.rowAt(0), row)
		assert.Nil(t, a.rowAt(1), "closed")

		// selecting the conversation reads it, and marks it as read
		a.list.SelectIndexEvent(0, events.SelectOne)
	}), func() {
		require.Eventually(t, func() bool {
			a.imapMu[testEmail].Lock()
			defer a.imapMu[testEmail].Unlock()
			return slices.Contains(hello.Flags, imap.FlagSeen)
		}, 5*time.Second, 10*time.Millisecond)
		assert.Contains(t, serverFlags(t, a, "INBOX", hello.Labels[0].UID), imap.FlagSeen)
	}, core.Locked(b, func() {
		assert.Equal(t, hello, a.readMessage)
		assert.Equal(t, "Hi there\r\nhow are you", hello.parsed.plain)
		require.Len(t, hello.parsed.attachments, 1)
		assert.Equal(t, "notes.txt", hello.parsed.attachments[0].Filename)
		assert.Equal(t, "some notes", strings.TrimSpace(string(hello.parsed.attachments[0].Data)))
		reply := hello.replies[0]
		assert.Equal(t, []string{"1@example.com"}, reply.parsed.references)
		assert.True(t, hello.isRead())
		a.MarkAsRead() // already
		a.MarkAsUnread()
	}), func() {
		require.Eventually(t, func() bool {
			a.imapMu[testEmail].Lock()
			defer a.imapMu[testEmail].Unlock()
			return !slices.Contains(hello.Flags, imap.FlagSeen)
		}, 5*time.Second, 10*time.Millisecond)
		assert.NotContains(t, serverFlags(t, a, "Work", hello.Labels[1].UID), imap.FlagSeen)
		// the messages are in the store
		stored, err := a.stores[testEmail].load()
		require.NoError(t, err)
		assert.Len(t, stored, 3)
		assert.Equal(t, hello.Flags, stored[hello.key].Flags)
	})
}

// TestLabels tests changing the labels of a message, deleting it, and
// the tree of labels.
func TestLabels(t *testing.T) {
	b, a := newTestApp(t, newIMAP(t))
	var hello *CacheMessage
	waitLabels := func(want ...string) {
		t.Helper()
		require.Eventually(t, func() bool {
			a.imapMu[testEmail].Lock()
			defer a.imapMu[testEmail].Unlock()
			return slices.Equal(want, labelNames(hello))
		}, 5*time.Second, 10*time.Millisecond, "%v", want)
	}
	core.RenderImage(b, func() {
		require.NoError(t, a.cacheMessages())
		hello = messageBySubject(a, "Hello")
	}, core.Locked(b, func() {
		a.readMessage = hello
		a.label([]string{"INBOX"})
	}), func() {
		waitLabels("INBOX")
	}, core.Locked(b, func() {
		a.Delete()
	}), func() {
		waitLabels("[Gmail]/Trash")
		require.NoError(t, a.cacheMessages())
		assert.Equal(t, []string{"[Gmail]/Trash"}, labelNames(messageBySubject(a, "Hello")), "the server agrees")
		assert.Len(t, a.cache[testEmail], 3)
	}, core.Locked(b, func() {
		a.showLabel = "[Gmail]/Trash"
		a.Update()
		assert.Len(t, a.listCache, 2, "the messages of the conversation, each on its own (see messagesOnly)")
		a.Label() // the dialog
	}), core.Locked(b, func() {
		d := core.TopScene(b)
		require.NotNil(t, d)
		bt := core.ButtonByText(d, "Save")
		require.NotNil(t, bt)
		bt.Send(events.Click)
	}), func() {
		waitLabels("[Gmail]/Trash")
	})
}

// TestFriendlyLabels tests the user-friendly names of the labels.
func TestFriendlyLabels(t *testing.T) {
	assert.Equal(t, "Inbox", friendlyLabelName("INBOX"))
	assert.Equal(t, "Sent", friendlyLabelName("[Gmail]/Sent Mail"))
	assert.Equal(t, "Work/Project", friendlyLabelName("Work/Project"))
}

// smtpBackend is an SMTP server backend that keeps the messages sent.
type smtpBackend struct {
	mu   sync.Mutex
	data []string
	to   []string
}

func (be *smtpBackend) NewSession(c *smtp.Conn) (smtp.Session, error) {
	return &smtpSession{be: be}, nil
}

type smtpSession struct{ be *smtpBackend }

func (s *smtpSession) Reset()        {}
func (s *smtpSession) Logout() error { return nil }
func (s *smtpSession) Mail(from string, opts *smtp.MailOptions) error {
	return nil
}
func (s *smtpSession) Rcpt(to string, opts *smtp.RcptOptions) error {
	s.be.mu.Lock()
	defer s.be.mu.Unlock()
	s.be.to = append(s.be.to, to)
	return nil
}
func (s *smtpSession) Data(r io.Reader) error {
	b, err := io.ReadAll(r)
	s.be.mu.Lock()
	defer s.be.mu.Unlock()
	s.be.data = append(s.be.data, string(b))
	return err
}

// newSMTP starts an SMTP server for sending messages in the tests.
func newSMTP(t *testing.T) *smtpBackend {
	t.Helper()
	be := &smtpBackend{}
	srv := smtp.NewServer(be)
	srv.AllowInsecureAuth = true
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	go srv.Serve(ln)
	t.Cleanup(func() { srv.Close() })
	oldAddress, oldSend := smtpAddress, sendMail
	smtpAddress = ln.Addr().String()
	// without STARTTLS, which smtp.SendMail requires
	sendMail = func(addr string, a sasl.Client, from string, to []string, r io.Reader) error {
		c, err := smtp.Dial(addr)
		if err != nil {
			return err
		}
		defer c.Close()
		return c.SendMail(from, to, r)
	}
	t.Cleanup(func() { smtpAddress, sendMail = oldAddress, oldSend })
	return be
}

// TestReplySend tests replying to and forwarding a message, and sending
// a message with an attachment.
func TestReplySend(t *testing.T) {
	be := newSMTP(t)
	b, a := newTestApp(t, newIMAP(t))
	att := filepath.Join(t.TempDir(), "notes.txt")
	require.NoError(t, os.WriteFile(att, []byte("the notes"), 0o644))
	var hello *CacheMessage
	core.RenderImage(b, func() {
		require.NoError(t, a.cacheMessages())
		hello = messageBySubject(a, "Hello")
	}, core.Locked(b, func() {
		a.readMessage = hello
		a.Update()
	}), core.Locked(b, func() {
		a.Reply()
		assert.Equal(t, "Re: Hello", a.composeMessage.Subject)
		require.Len(t, a.composeMessage.To, 1)
		assert.Equal(t, "alice@example.com", a.composeMessage.To[0].Address)
		assert.Contains(t, a.composeMessage.body, `"Alice" <alice@example.com> wrote:`)
		assert.Contains(t, a.composeMessage.body, "> Hi there")
		assert.Equal(t, "1@example.com", a.composeMessage.inReplyTo)

		a.Forward()
		assert.Equal(t, "Fwd: Hello", a.composeMessage.Subject)
		assert.Contains(t, a.composeMessage.body, "Begin forwarded message")

		a.readMessage = hello.replies[0] // from me
		a.Reply()
		require.Len(t, a.composeMessage.To, 2, "to the receivers of my message")
		a.ReplyAll()
		assert.Equal(t, "Re: Hello", a.composeMessage.Subject, "no second Re:")
		var to []string
		for _, ad := range a.composeMessage.To {
			to = append(to, ad.Address)
		}
		assert.Equal(t, []string{"alice@example.com", "bob@example.com"}, to, "not me")

		a.Compose()
		a.composeMessage.To[0].Address = "dan@example.com"
		a.composeMessage.Subject = "News"
		a.composeMessage.body = "Some *news*"
		a.composeMessage.Attachments = append(a.composeMessage.Attachments, core.Filename(att))
		require.NoError(t, a.Send())
		be.mu.Lock()
		defer be.mu.Unlock()
		require.Len(t, be.data, 1)
		assert.Equal(t, []string{"dan@example.com"}, be.to)
		msg := be.data[0]
		assert.Contains(t, msg, "Subject: News")
		assert.Contains(t, msg, "<em>news</em>", "the HTML version")
		assert.Contains(t, msg, `filename=notes.txt`)

		a.composeMessage.Sender.From = ""
		assert.Error(t, a.Send(), "no sender")
		a.composeMessage.Sender.From = testEmail
		a.composeMessage.Attachments = []core.Filename{"none.txt"}
		assert.Error(t, a.Send(), "no attachment")
	}))

	// a server that is not there is an error, not a panic, and the
	// message stays in the outbox, to be sent later
	smtpAddress = "127.0.0.1:1"
	a.composeMessage.Attachments = nil
	err := a.Send()
	assert.Error(t, err)
	assert.True(t, isOffline(err), err)
	n, refused := a.outboxCount()
	assert.Equal(t, 1, n)
	assert.Equal(t, 0, refused)
}

// TestForwardAttachments tests including the attachments of the message
// forwarded or replied to: by default when forwarding, and not replying,
// switching them on and off, and removing their files once sent.
func TestForwardAttachments(t *testing.T) {
	be := newSMTP(t)
	b, a := newTestApp(t, newIMAP(t))
	var hello *CacheMessage
	core.RenderImage(b, func() {
		require.NoError(t, a.cacheMessages())
		hello = messageBySubject(a, "Hello")
	}, core.Locked(b, func() {
		a.readMessage = hello
		a.Update()
	}), core.Locked(b, func() {
		hello.parsed.attachments = []*Attachment{{Filename: "report.pdf", Data: []byte("the report")}}

		a.Reply()
		assert.Empty(t, a.composeMessage.Attachments, "not in a reply by default")
		sm := a.composeMessage
		require.NoError(t, a.includeAttachments(sm, true))
		require.Len(t, sm.Attachments, 1)
		f := string(sm.Attachments[0])
		assert.Equal(t, "report.pdf", filepath.Base(f))
		data, err := os.ReadFile(f)
		require.NoError(t, err)
		assert.Equal(t, "the report", string(data))
		require.NoError(t, a.includeAttachments(sm, false))
		assert.Empty(t, sm.Attachments)
		assert.NoFileExists(t, f, "removed when switched off")

		a.Forward()
		require.Len(t, a.composeMessage.Attachments, 1, "in a forward by default")
		f = string(a.composeMessage.Attachments[0])
		assert.True(t, a.isForwarded(a.composeMessage.Attachments[0]))
		a.composeMessage.To[0].Address = "dan@example.com"
		require.NoError(t, a.Send())
		be.mu.Lock()
		defer be.mu.Unlock()
		require.Len(t, be.data, 1)
		assert.Contains(t, be.data[0], "filename=report.pdf")
		assert.NoFileExists(t, f, "removed once sent")
	}))
}

// TestTestingMode tests the testing mode: syncing the sample mail of all
// of the accounts of the test server, switching between accounts,
// sending a message from one to another, and receiving a test message.
func TestTestingMode(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	oldFile, oldAccounts := Settings.File, Settings.Accounts
	t.Cleanup(func() { Settings.File, Settings.Accounts = oldFile, oldAccounts })
	t.Cleanup(core.TrackWindows())
	b := core.NewBody()
	a := NewApp(b)
	require.NoError(t, a.StartTesting(TestingOptions{Seed: 1}))
	t.Cleanup(a.testServer.Close)
	you, alice := Settings.Accounts[0].Email, Settings.Accounts[1].Email
	assert.Equal(t, you, a.currentEmail)
	assert.NotEqual(t, oldFile, Settings.File, "not the real settings")
	assert.NotEqual(t, core.TheApp.AppDataDir(), a.dataDir(), "not the real data")

	syncAll := func() {
		t.Helper()
		for {
			a.syncMore = false
			require.NoError(t, a.cacheMessages())
			if !a.syncMore {
				return
			}
		}
	}
	core.RenderImage(b, func() {
		syncAll()
		assert.Greater(t, len(a.cache[you]), 150)
		assert.NotEmpty(t, a.cache[alice])
		assert.Equal(t, "[Gmail]/Trash", a.trash[you])
		assert.Contains(t, a.labels[you], "Projects/Alpha")
	}, core.Locked(b, func() {
		a.showMailbox(you, "INBOX")
		assert.NotEmpty(t, a.listCache)
		a.showMailbox(alice, "INBOX")
		assert.Equal(t, alice, a.currentEmail)
		n := len(a.listCache)
		assert.NotZero(t, n)

		a.showMailbox(you, "INBOX")
		a.Compose()
		assert.Equal(t, SenderAccount(you), a.composeMessage.Sender.From)
		a.composeMessage.To[0].Address = alice
		a.composeMessage.Subject = "Testing mode"
		a.composeMessage.body = "Hello *Alice*"
		require.NoError(t, a.Send())
		require.NoError(t, a.ReceiveTestMessage())
	}), func() {
		syncAll()
		assert.NotNil(t, findSubject(a, alice, "Testing mode", "INBOX"), "alice got it")
		assert.NotNil(t, findSubject(a, you, "Testing mode", "[Gmail]/Sent Mail"), "you sent it")
		unread := 0
		for _, cm := range a.cache[you] {
			if cm.Date.After(time.Now().Add(-time.Minute)) && !slices.Contains(cm.Flags, imap.FlagSeen) {
				unread++
			}
		}
		assert.Equal(t, 1, unread, "the received test message")
	})
}

// findSubject returns the cached message of the given account with the
// given subject in the given mailbox.
func findSubject(a *App, email, subject, mailbox string) *CacheMessage {
	for _, cm := range a.cache[email] {
		if cm.Subject == subject && slices.ContainsFunc(cm.Labels, func(l Label) bool { return l.Name == mailbox }) {
			return cm
		}
	}
	return nil
}

// TestList tests the list of messages: opening a conversation, reading one
// of its messages on its own, the flat list, and filing a conversation.
func TestList(t *testing.T) {
	b, a := newTestApp(t, newIMAP(t))
	t.Cleanup(func() { Settings.ListViews = 0; Settings.Defaults() })
	var hello *CacheMessage
	core.RenderImage(b, func() {
		require.NoError(t, a.cacheMessages())
		hello = messageBySubject(a, "Hello")
	}, core.Locked(b, func() {
		a.Update()
		a.list.Open(0)
		a.Update()
		child := a.rowAt(1)
		require.NotNil(t, child)
		assert.Equal(t, "Re: Hello", child.Subject)
		a.list.SelectIndexEvent(1, events.SelectOne)
		assert.True(t, a.readSingle)
		assert.Equal(t, hello.replies[0], a.readMessage)
		assert.Len(t, a.readMessages(), 1)

		Settings.ListViews = 0 // flat
		a.Update()
		require.Len(t, a.listRows, 2)
		assert.False(t, a.listRows[0].conversation)
		assert.Equal(t, "Re: Hello", a.listRows[0].Subject)
		Settings.Defaults() // conversations
		a.Update()
		require.Len(t, a.listRows, 1)
		assert.True(t, a.listRows[0].conversation)
		assert.False(t, a.listRows[0].Closed, "still open")

		// filing the conversation out of the INBOX files both messages
		a.list.SelectIndexEvent(0, events.SelectOne)
		assert.False(t, a.readSingle)
		a.label([]string{"Work"})
	}), func() {
		require.Eventually(t, func() bool {
			a.imapMu[testEmail].Lock()
			defer a.imapMu[testEmail].Unlock()
			return slices.Equal([]string{"Work"}, labelNames(hello)) &&
				slices.Equal([]string{"Work"}, labelNames(hello.replies[0]))
		}, 5*time.Second, 10*time.Millisecond)
		require.NoError(t, a.cacheMessages())
	}, core.Locked(b, func() {
		a.Update()
		assert.Empty(t, a.listRows, "the INBOX is empty")
		a.showMailbox(testEmail, "Work")
		require.Len(t, a.listRows, 2, "the conversation and the report")
	}))
}

// TestSendBetweenAccounts tests that messages sent between the accounts of
// the test server each arrive, with their own Message-ID with the domain
// of the sender, as messages without one were all cached as one.
func TestSendBetweenAccounts(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	oldFile, oldAccounts := Settings.File, Settings.Accounts
	t.Cleanup(func() { Settings.File, Settings.Accounts = oldFile, oldAccounts })
	t.Cleanup(core.TrackWindows())
	b := core.NewBody()
	a := NewApp(b)
	require.NoError(t, a.StartTesting(TestingOptions{Seed: 1}))
	t.Cleanup(a.testServer.Close)
	you, bob := Settings.Accounts[0].Email, Settings.Accounts[2].Email
	syncAll := func() {
		for {
			a.syncMore = false
			require.NoError(t, a.cacheMessages())
			if !a.syncMore {
				return
			}
		}
	}
	send := func(from, to, subject string) {
		a.showMailbox(from, "INBOX")
		a.Compose()
		a.composeMessage.To[0].Address = to
		a.composeMessage.Subject = subject
		require.NoError(t, a.Send())
	}
	core.RenderImage(b, syncAll, core.Locked(b, func() {
		send(you, bob, "one")
		send(bob, you, "two")
		send(bob, you, "three")
	}), func() {
		syncAll()
		for _, subject := range []string{"two", "three"} {
			cm := findSubject(a, you, subject, "INBOX")
			require.NotNil(t, cm, subject)
			assert.True(t, strings.HasSuffix(cm.MessageID, "@mail.test"), cm.MessageID)
			assert.NotNil(t, findSubject(a, bob, subject, "[Gmail]/Sent Mail"), subject)
		}
		assert.NotNil(t, findSubject(a, you, "one", "[Gmail]/Sent Mail"))
		assert.NotNil(t, findSubject(a, bob, "one", "INBOX"))
		assert.NotEqual(t, cacheKey(&imap.Envelope{Subject: "a"}), cacheKey(&imap.Envelope{Subject: "b"}), "no Message-ID")
	})
}

// TestMailboxes tests the tree of mailboxes: the order, icons, and unread
// counts of the mailboxes, the selection of the mailbox being shown, and
// moving to the next mailbox with the keyboard.
func TestMailboxes(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	oldFile, oldAccounts := Settings.File, Settings.Accounts
	t.Cleanup(func() { Settings.File, Settings.Accounts = oldFile, oldAccounts })
	t.Cleanup(core.TrackWindows())
	b := core.NewBody()
	a := NewApp(b)
	require.NoError(t, a.StartTesting(TestingOptions{Seed: 1}))
	t.Cleanup(a.testServer.Close)
	you := Settings.Accounts[0].Email
	node := func(parent tree.Node, name string) *core.Tree {
		t.Helper()
		n := parent.AsTree().ChildByName(name)
		require.NotNil(t, n, name)
		return n.(core.Treer).AsCoreTree()
	}
	var inbox *core.Tree
	core.RenderImage(b, func() {
		for {
			a.syncMore = false
			require.NoError(t, a.cacheMessages())
			if !a.syncMore {
				break
			}
		}
	}, core.Locked(b, func() {
		a.Update()
	}), core.Locked(b, func() {
		acct := node(a.mailboxes, you)
		var names []string
		for _, c := range acct.Children {
			names = append(names, c.AsTree().Name)
		}
		assert.Equal(t, []string{"[Gmail]/Starred", "[Gmail]/Drafts", "[Gmail]/Sent Mail", untaggedLabel, "[Gmail]/Spam", "[Gmail]/Trash",
			"Admin", "Family", "Lists", "Newsletters", "Projects"}, names, "the INBOX is in the inboxes")
		inboxes := node(a.mailboxes, "Inboxes")
		assert.Equal(t, "Inboxes", a.mailboxes.Children[0].AsTree().Name, "first")
		assert.False(t, inboxes.Closed)
		inbox = node(inboxes, you)
		assert.True(t, inbox.StateIs(states.Selected), "the mailbox shown")
		assert.Equal(t, fmt.Sprintf("%s\u2002(%d)", you, a.unread[you]["INBOX"]), inbox.Text)
		n := 0
		for _, email := range Settings.Emails() {
			n += a.unread[email]["INBOX"]
		}
		assert.Equal(t, fmt.Sprintf("Inboxes\u2002(%d)", n), inboxes.Text, "all of the unread")
		assert.NotZero(t, a.unread[you]["INBOX"])
		assert.Contains(t, acct.Text, "you@mail.test\u2002(", "a wider space before the count")
		assert.Equal(t, icons.Inbox, inbox.Icon)
		projects := node(acct, "Projects")
		assert.Equal(t, icons.Folder, projects.Icon, "with subfolders")
		assert.Equal(t, icons.Label, node(projects, "Projects/Alpha").Icon, "a leaf")
		inbox.SetFocus()
	}), func() {
		b.Scene.RenderWindow().SystemWindow.Events().KeyChord(0, key.CodeDownArrow, 0)
	}, core.Locked(b, func() {
		assert.Equal(t, Settings.Accounts[1].Email, a.currentEmail, "moved with the keyboard, to the next inbox")
		assert.Equal(t, "INBOX", a.showLabel)
		assert.False(t, inbox.StateIs(states.Selected))
		a.showMailbox(you, "[Gmail]/Starred")
	}), core.Locked(b, func() {
		assert.True(t, node(node(a.mailboxes, you), "[Gmail]/Starred").StateIs(states.Selected))
		assert.False(t, node(node(a.mailboxes, "Inboxes"), Settings.Accounts[1].Email).StateIs(states.Selected))
		assert.False(t, node(node(a.mailboxes, you), "[Gmail]/Drafts").StateIs(states.Selected))
	}))
}

// TestFileKeys tests filing a message with Control+O, which selects the
// next message, tagging one with Control+T, and moving among the panels
// with Control+X Control+O.
func TestFileKeys(t *testing.T) {
	b, a := newTestApp(t, newIMAP(t))
	var hello, report *CacheMessage
	keys := func(chords ...key.Chord) func() {
		return func() {
			src := b.Scene.RenderWindow().SystemWindow.Events()
			for _, c := range chords {
				r, code, mods, err := c.Decode()
				require.NoError(t, err)
				src.KeyChord(r, code, mods)
			}
		}
	}
	choose := func(label string) func() {
		return core.Locked(b, func() {
			d := core.TopScene(b)
			require.NotNil(t, d)
			var ch *core.Chooser
			for w := range d.AllWidgets() {
				if c, ok := w.(*core.Chooser); ok {
					ch = c
				}
			}
			require.NotNil(t, ch, "the dialog of mailboxes")
			ch.SetCurrentValue(mailboxID{testEmail, label})
			ch.SendChange()
		})
	}
	core.RenderImage(b, func() {
		require.NoError(t, a.cacheMessages())
		hello = messageBySubject(a, "Hello")
		report = messageBySubject(a, "Report")
	}, core.Locked(b, func() {
		a.showMailbox(testEmail, "Work")
		require.Len(t, a.listRows, 2)
		a.list.SelectIndexEvent(0, events.SelectOne)
		require.Equal(t, report, a.readMessage, "the newest")
		a.list.SetFocus()
	}), keys("Control+T"), choose("[Gmail]/Trash"), func() {
		require.Eventually(t, func() bool {
			a.imapMu[testEmail].Lock()
			defer a.imapMu[testEmail].Unlock()
			return slices.Equal([]string{"Work", "[Gmail]/Trash"}, labelNames(report))
		}, 5*time.Second, 10*time.Millisecond, "tagged, still in Work")
	}, keys("Control+O"), choose("INBOX"), func() {
		require.Eventually(t, func() bool {
			a.imapMu[testEmail].Lock()
			defer a.imapMu[testEmail].Unlock()
			return slices.Equal([]string{"INBOX", "[Gmail]/Trash"}, labelNames(report))
		}, 5*time.Second, 10*time.Millisecond, "filed out of Work")
	}, core.Locked(b, func() {
		require.Len(t, a.listRows, 1)
		assert.Equal(t, hello, a.readMessage, "the next one is selected")
		a.list.SetFocus()
	}), func() { keys(seqStart(t), "Control+P")() }, core.Locked(b, func() {
		assert.True(t, a.mailboxes.ContainsFocus(), "the previous panel")
		work := a.mailboxes.ChildByName(testEmail).AsTree().ChildByName("Work").(core.Treer).AsCoreTree()
		assert.Len(t, a.mailboxes.GetSelectedNodes(), 1)
		assert.True(t, work.ContainsFocus(), "the mailbox shown")
		assert.False(t, work.StateIs(states.Focused), "quietly, without the highlight")
	}), func() { keys(seqStart(t), "Control+O")() }, core.Locked(b, func() {
		assert.True(t, a.list.ContainsFocus(), "the next panel")
	}), keys("Control+Shift+Tab"), core.Locked(b, func() {
		assert.True(t, a.mailboxes.ContainsFocus(), "the previous panel, again")
	}), keys("Control+Tab"), core.Locked(b, func() {
		assert.True(t, a.list.ContainsFocus(), "the next panel, again")
	}))
}

// seqStart returns the first chord of the key sequences of the active key
// map, which differs by platform (Control+X on the Mac).
func seqStart(t *testing.T) key.Chord {
	for c, f := range *keymap.ActiveMap {
		if f == keymap.MultiA {
			return c
		}
	}
	t.Fatal("no key sequences in the key map")
	return ""
}

// TestAttachmentOpen tests opening attachments in their default apps,
// each from its own temporary file.
func TestAttachmentOpen(t *testing.T) {
	var opened []string
	old := openURL
	openURL = func(url string) { opened = append(opened, url) }
	t.Cleanup(func() { openURL = old })
	a1 := &Attachment{Filename: "notes.txt", Data: []byte("one")}
	a2 := &Attachment{Filename: "../notes.txt", Data: []byte("two")}
	require.NoError(t, a1.Open())
	require.NoError(t, a2.Open())
	require.Len(t, opened, 2)
	assert.NotEqual(t, a1.openedFile, a2.openedFile, "the same name, in different directories")
	assert.Equal(t, "notes.txt", filepath.Base(a2.openedFile))
	assert.True(t, strings.HasPrefix(a2.openedFile, attachmentsDir), "not outside of it")
	data, err := os.ReadFile(a2.openedFile)
	require.NoError(t, err)
	assert.Equal(t, "two", string(data))
	assert.True(t, strings.HasPrefix(opened[0], "file://"), opened[0])
}

// TestSignature tests adding the default signature of the account to new
// messages, above the quoted message of a reply.
func TestSignature(t *testing.T) {
	b, a := newTestApp(t, newIMAP(t))
	Settings.Signatures = []Signature{{Name: "Work", Text: "Me\nThe Lab\n"}, {Name: "Home", Text: "me"}}
	Settings.DefaultSignatures = map[string]SignatureName{testEmail: "Work"}
	t.Cleanup(func() { Settings.Signatures, Settings.DefaultSignatures = nil, nil })
	core.RenderImage(b, func() {
		require.NoError(t, a.cacheMessages())
	}, core.Locked(b, func() {
		a.Compose()
		assert.Equal(t, "\n\nMe  \nThe Lab\n", a.composeMessage.body)
		a.readMessage = messageBySubject(a, "Report")
		a.readSingle = true
		a.Update()
	}), core.Locked(b, func() {
		a.Reply()
		assert.True(t, strings.HasPrefix(a.composeMessage.body, "\n\nMe  \nThe Lab\n\n\n> On "), a.composeMessage.body)
		Settings.DefaultSignatures = nil
		a.Compose()
		assert.NotContains(t, a.composeMessage.body, "--", "no default")
	}))
}

// TestSignatureSettings tests that each account has an entry for its
// default signature, chosen from the names of the signatures.
func TestSignatureSettings(t *testing.T) {
	old := *Settings
	t.Cleanup(func() { *Settings = old })
	Settings.Accounts = gmailAccounts("a@x.test", "b@x.test")
	Settings.DefaultSignatures = map[string]SignatureName{"b@x.test": "Work"}
	Settings.Signatures = []Signature{{Name: "Work"}, {Name: "Home"}}
	Settings.Apply()
	assert.Equal(t, map[string]SignatureName{"a@x.test": "", "b@x.test": "Work"}, Settings.DefaultSignatures)
	ch, ok := SignatureName("").Widget().(*core.Chooser)
	require.True(t, ok)
	require.Len(t, ch.Items, 3)
	assert.Equal(t, SignatureName("Home"), ch.Items[2].Value)
}

// TestSavedLayout tests saving the column widths and sort of the list of
// messages, the splits, and which mailboxes are open, in the settings,
// and applying them to a new app.
func TestSavedLayout(t *testing.T) {
	old := *Settings
	t.Cleanup(func() { *Settings = old })
	Settings.File = filepath.Join(t.TempDir(), "settings.toml")
	b, a := newTestApp(t, newIMAP(t))
	core.RenderImage(b, func() {
		require.NoError(t, a.cacheMessages())
	}, core.Locked(b, func() {
		a.Update()
	}), core.Locked(b, func() {
		a.list.ColumnWidths = map[string]float32{"From": 123}
		a.list.SetSortFieldName("Subject:up")
		a.list.SendChange()
		a.splits.SetSplits(0.2, 0.3, 0.5)
		a.splits.SendChange()
		acct := a.mailboxes.ChildByName(testEmail).(*MailboxTree)
		assert.False(t, acct.Closed, "an account is open")
		acct.Close()
	}))
	assert.Equal(t, map[string]float32{"From": 123}, Settings.ColumnWidths)
	assert.Equal(t, "Subject:up", Settings.Sort)
	assert.InDeltaSlice(t, []float32{0.2, 0.3, 0.5}, Settings.Splits, 0.001)
	assert.Equal(t, map[string]bool{testEmail: false}, Settings.OpenMailboxes)
	saved, err := os.ReadFile(Settings.File)
	require.NoError(t, err)
	assert.Contains(t, string(saved), "Subject:up")

	b2, a2 := newTestApp(t, a.imapClient[testEmail])
	core.RenderImage(b2, func() {
		require.NoError(t, a2.cacheMessages())
	}, core.Locked(b2, func() {
		a2.Update()
	}), core.Locked(b2, func() {
		assert.Equal(t, "Subject:up", a2.list.SortFieldName())
		assert.Equal(t, float32(123), a2.list.ColumnWidths["From"])
		assert.InDeltaSlice(t, []float32{0.2, 0.3, 0.5}, a2.splits.Splits(), 0.001)
		assert.True(t, a2.mailboxes.ChildByName(testEmail).(*MailboxTree).Closed)
	}))
}

// TestShortcuts tests the keyboard shortcuts of the actions on messages,
// and the context menu of the list of messages, which shows them.
func TestShortcuts(t *testing.T) {
	b, a := newTestApp(t, newIMAP(t))
	core.RenderImage(b, func() {
		require.NoError(t, a.cacheMessages())
	}, core.Locked(b, func() {
		a.Update()
		a.list.SelectIndexEvent(0, events.SelectOne)
		a.list.SetFocusQuiet()
		assert.Nil(t, a.composeMessage)
	}), func() {
		r, code, mods, err := key.Chord("Command+Shift+R").PlatformChord().Decode()
		require.NoError(t, err)
		b.Scene.RenderWindow().SystemWindow.Events().KeyChord(r, code, mods)
	}, core.Locked(b, func() {
		require.NotNil(t, a.composeMessage, "reply all")
		assert.Equal(t, "Re: Hello", a.composeMessage.Subject)
		m := core.NewScene()
		a.messageMenu(m, image.Point{})
		var shortcuts []string
		for w := range m.AllWidgets() {
			if bt, ok := w.(*core.FuncButton); ok {
				shortcuts = append(shortcuts, bt.Text+"="+string(bt.Shortcut))
			}
		}
		assert.Equal(t, []string{"Reply=Command+R", "Reply all=Command+W", "Forward=Command+F",
			"File=Control+O", "Tag=Control+T", "Star=Command+Shift+L", "Delete=Delete",
			"Remove attachments=Command+X", "Mark as read=", "Mark as unread=Command+Shift+U"}, shortcuts)
	}))
}

// TestBcc tests sending with Cc and Bcc, and the automatic Bcc to the
// sender, which are not in the header.
func TestBcc(t *testing.T) {
	be := newSMTP(t)
	b, a := newTestApp(t, newIMAP(t))
	Settings.AutoBCC = true
	t.Cleanup(func() { Settings.AutoBCC = false })
	core.RenderImage(b, func() {
		require.NoError(t, a.cacheMessages())
	}, core.Locked(b, func() {
		a.Compose()
		require.Len(t, a.composeMessage.Bcc, 1, "the sender, shown")
		assert.Equal(t, testEmail, a.composeMessage.Bcc[0].Address)
		a.composeMessage.To[0].Address = "dan@example.com"
		a.composeMessage.Cc = []*gomail.Address{{Address: "erin@example.com"}, {}}
		a.composeMessage.Bcc = append(a.composeMessage.Bcc, &gomail.Address{Address: "frank@example.com"}, &gomail.Address{Address: "DAN@example.com"})
		a.composeMessage.Subject = "Copies"
		require.NoError(t, a.Send())
		be.mu.Lock()
		defer be.mu.Unlock()
		assert.Equal(t, []string{"dan@example.com", "erin@example.com", testEmail, "frank@example.com"}, be.to)
		require.Len(t, be.data, 1)
		assert.Contains(t, be.data[0], "Cc: <erin@example.com>")
		assert.NotContains(t, be.data[0], "frank")
		assert.NotContains(t, be.data[0], "Bcc")

		Settings.AutoBCC = false
		a.Compose()
		assert.Empty(t, nonEmpty(a.composeMessage.Bcc), "not without AutoBCC")
	}))
}

// TestFileToAccount tests filing a conversation into a folder of another
// account, the filing account, which copies its messages there and
// removes them from the mailbox of the account they are in.
func TestFileToAccount(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	old := *Settings
	t.Cleanup(func() { *Settings = old })
	t.Cleanup(core.TrackWindows())
	b := core.NewBody()
	a := NewApp(b)
	require.NoError(t, a.StartTesting(TestingOptions{Seed: 1}))
	t.Cleanup(a.testServer.Close)
	you, alice := Settings.Accounts[0].Email, Settings.Accounts[1].Email
	require.NoError(t, a.testServer.CreateMailbox(alice, "Filed/Mail"))
	Settings.FilingAccount = AccountName(alice)
	syncAll := func() {
		for {
			a.syncMore = false
			require.NoError(t, a.cacheMessages())
			if !a.syncMore {
				return
			}
		}
	}
	var conv *messageRow
	var subject string
	var n int
	core.RenderImage(b, syncAll, core.Locked(b, func() {
		a.showMailbox(you, "INBOX")
		for i, r := range a.listRows {
			if r.conversation {
				conv = r
				a.list.SelectIndexEvent(i, events.SelectOne)
				break
			}
		}
		require.NotNil(t, conv)
		subject = conv.message.Subject
		n = len(a.readMessages())
		assert.Equal(t, []string{alice, you, Settings.Accounts[2].Email}, a.filingAccounts())
		a.File()
	}), core.Locked(b, func() {
		d := core.TopScene(b)
		var ch *core.Chooser
		for w := range d.AllWidgets() {
			if c, ok := w.(*core.Chooser); ok {
				ch = c
			}
		}
		require.NotNil(t, ch)
		assert.Equal(t, "Inbox", ch.Items[0].Text, "the filing account first")
		assert.True(t, slices.ContainsFunc(ch.Items, func(it core.ChooserItem) bool {
			return it.Text == you+": Projects/Alpha"
		}), "then the others, with their accounts")
		ch.SetCurrentValue(mailboxID{alice, "Filed/Mail"})
		ch.SendChange()
	}), func() {
		require.Eventually(t, func() bool {
			a.AsyncLock()
			defer a.AsyncUnlock()
			for _, cm := range conv.message.replies {
				if slices.ContainsFunc(cm.Labels, func(l Label) bool { return l.Name == "INBOX" }) {
					return false
				}
			}
			return !slices.ContainsFunc(conv.message.Labels, func(l Label) bool { return l.Name == "INBOX" })
		}, 5*time.Second, 10*time.Millisecond, "out of the INBOX")
		syncAll()
		filed := 0
		for _, cm := range a.cache[alice] {
			if strings.TrimPrefix(cm.Subject, "Re: ") == subject &&
				slices.ContainsFunc(cm.Labels, func(l Label) bool { return l.Name == "Filed/Mail" }) {
				filed++
			}
		}
		assert.Equal(t, n, filed, "all of the conversation")
	})
}

// TestMultiSelect tests selecting more than one row of the list of
// messages, which are not marked as read, and filing all of them at once.
func TestMultiSelect(t *testing.T) {
	b, a := newTestApp(t, newIMAP(t))
	var hello, reply, report *CacheMessage
	core.RenderImage(b, func() {
		require.NoError(t, a.cacheMessages())
		hello = messageBySubject(a, "Hello")
		report = messageBySubject(a, "Report")
	}, core.Locked(b, func() {
		a.showMailbox(testEmail, "Work")
		reply = hello.replies[0] // the replies are set by updates of the list
		require.Len(t, a.listRows, 2)
		a.list.SelectIndexEvent(0, events.SelectOne)
		a.list.SelectIndexEvent(1, events.ExtendOne)
		require.Len(t, a.selectedRows, 2)
		msgs := a.readMessages()
		assert.Len(t, msgs, 3, "the report, and the conversation")
		assert.Contains(t, msgs, report)
		assert.Contains(t, msgs, hello.replies[0])
		assert.False(t, slices.Contains(hello.Flags, imap.FlagSeen), "not marked as read")
		a.fileTo(a.currentEmail, msgs, "Work", "[Gmail]/Trash", a.selectNext())
	}), func() {
		require.Eventually(t, func() bool {
			a.AsyncLock() // as the labels are set, and the list updated
			defer a.AsyncUnlock()
			for _, cm := range []*CacheMessage{hello, reply, report} {
				if slices.ContainsFunc(cm.Labels, func(l Label) bool { return l.Name == "Work" }) {
					return false
				}
			}
			return len(a.listRows) == 0 && len(a.pendingLabels) == 0
		}, 5*time.Second, 10*time.Millisecond, "all filed out of Work")
	}, core.Locked(b, func() {
		assert.Empty(t, a.listRows)
		assert.Empty(t, a.selectedRows)
		assert.Contains(t, labelNames(report), "[Gmail]/Trash")
		assert.Contains(t, labelNames(hello), "INBOX", "its other labels stay")
	}))
}

// TestViewMenu tests the view menu of the list of messages, whose
// switches set the view of the list in the settings.
func TestViewMenu(t *testing.T) {
	old := *Settings
	t.Cleanup(func() { *Settings = old })
	Settings.File = filepath.Join(t.TempDir(), "settings.toml")
	Settings.ListViews = 0
	Settings.Defaults()
	b, a := newTestApp(t, newIMAP(t))
	core.RenderImage(b, func() {
		require.NoError(t, a.cacheMessages())
	}, core.Locked(b, func() {
		a.Update()
		assert.False(t, a.flatList(), "conversations by default")
		m := core.NewScene()
		a.viewMenu(m, image.Point{})
		var sw *core.Switches
		for w := range m.AllWidgets() {
			if s, ok := w.(*core.Switches); ok {
				sw = s
			}
		}
		require.NotNil(t, sw)
		var names []string
		for _, it := range sw.Items {
			names = append(names, it.Text)
		}
		assert.Equal(t, []string{"Conversations", "Compact", "Labels", "Attachments", "Stars"}, names)
		sw.Update() // makes the switches
		var compact *core.Switch
		for w := range sw.AllWidgets() {
			if s, ok := w.(*core.Switch); ok && s.Text == "Compact" {
				compact = s
			}
		}
		require.NotNil(t, compact)
		compact.Send(events.Click)
	}), core.Locked(b, func() {
		assert.True(t, a.compactList())
		assert.False(t, a.flatList())
		assert.True(t, Settings.ListViews.HasFlag(ViewCompact))
		assert.True(t, a.list.FitWidth, "a new compact list")
		saved, err := os.ReadFile(Settings.File)
		require.NoError(t, err)
		assert.Contains(t, string(saved), "ListViews")
	}))
}

// TestFileOneOfConversation tests filing one message of a conversation
// into another account, which files only it, out of the INBOX, as only
// the row of a conversation files all of its messages, and that the
// messages of a conversation in other mailboxes are dimmed.
func TestFileOneOfConversation(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	old := *Settings
	t.Cleanup(func() { *Settings = old })
	t.Cleanup(core.TrackWindows())
	b := core.NewBody()
	a := NewApp(b)
	require.NoError(t, a.StartTesting(TestingOptions{Seed: 1}))
	t.Cleanup(a.testServer.Close)
	you, alice := Settings.Accounts[0].Email, Settings.Accounts[1].Email
	Settings.FilingAccount = AccountName(you)
	syncAll := func() {
		for {
			a.syncMore = false
			require.NoError(t, a.cacheMessages())
			if !a.syncMore {
				return
			}
		}
	}
	inInbox := func(cm *CacheMessage) bool {
		return slices.ContainsFunc(cm.Labels, func(l Label) bool { return l.Name == "INBOX" })
	}
	var conv, child *messageRow
	core.RenderImage(b, syncAll, core.Locked(b, func() {
		a.showMailbox(alice, "INBOX")
	}), core.Locked(b, func() {
		// a conversation with more than one message in the INBOX
		for i, r := range a.listRows {
			if r.conversation && len(slices.DeleteFunc(rowMessages(r), func(cm *CacheMessage) bool { return !inInbox(cm) })) > 1 {
				conv = r
				a.list.Open(i)
				break
			}
		}
		require.NotNil(t, conv)
	}), core.Locked(b, func() {
		for i := range 30 {
			r := a.rowAt(i)
			if r == nil {
				break
			}
			if slices.Contains(conv.Children, r) {
				assert.Equal(t, !inInbox(r.message), r.elsewhere, "dimmed if not in the INBOX")
				if child == nil && inInbox(r.message) {
					child = r
					a.list.SelectIndexEvent(i, events.SelectOne)
				}
			}
		}
		require.NotNil(t, child)
		require.Len(t, a.readMessages(), 1, "only the message of the row")
		a.fileToAccount(a.currentEmail, a.readMessages(), "INBOX", mailboxID{you, "Projects/Alpha"}, a.selectNext())
	}), func() {
		require.Eventually(t, func() bool {
			a.AsyncLock()
			defer a.AsyncUnlock()
			return !inInbox(child.message)
		}, 5*time.Second, 10*time.Millisecond, "out of the INBOX of alice")
		syncAll()
	}, core.Locked(b, func() {
		inboxLeft := 0
		for _, cm := range rowMessages(conv) {
			if inInbox(cm) {
				inboxLeft++
			}
		}
		assert.Positive(t, inboxLeft, "the others stay in the INBOX")
		n := 0
		for _, cm := range a.cache[you] {
			if slices.ContainsFunc(cm.Labels, func(l Label) bool { return l.Name == "Projects/Alpha" }) &&
				cm.MessageID == child.message.MessageID {
				n++
			}
		}
		assert.Equal(t, 1, n, "filed in you")
	}))
}

// TestGmailSync tests syncing with the Gmail-like test server: the changes
// of flags and labels made by another client, through CONDSTORE, and a
// message that is deleted, which is removed with its file and from the
// search index, which has the text of the messages that are downloaded.
func TestGmailSync(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	old := *Settings
	t.Cleanup(func() { *Settings = old })
	t.Cleanup(core.TrackWindows())
	b := core.NewBody()
	a := NewApp(b)
	require.NoError(t, a.StartTesting(TestingOptions{Seed: 1}))
	t.Cleanup(a.testServer.Close)
	t.Cleanup(a.closeStores)
	you := Settings.Accounts[0].Email
	syncAll := func() {
		for {
			a.syncMore = false
			require.NoError(t, a.syncAccount(you))
			if !a.syncMore {
				return
			}
		}
	}
	other, err := imapclient.DialInsecure(a.testServer.IMAPAddr, nil)
	require.NoError(t, err)
	t.Cleanup(func() { other.Close() })
	require.NoError(t, other.Login(you, "password").Wait())
	var cm *CacheMessage
	core.RenderImage(b, func() {
		syncAll()
		assert.True(t, a.gmail[you])
		assert.True(t, a.condStore[you])
		st, err := a.stores[you].mailbox("[Gmail]/All Mail")
		require.NoError(t, err)
		assert.NotZero(t, st.HighestModSeq)
		a.AsyncLock()
		for _, m := range a.cache[you] {
			if strings.HasPrefix(m.Subject, "Hello from") && !slices.Contains(m.Flags, imap.FlagSeen) {
				cm = m
			}
		}
		a.AsyncUnlock()
		require.NotNil(t, cm)
		assert.NotEmpty(t, cm.Filename, "downloaded")
		keys, err := a.searches[you].search(`subject:"`+cm.Subject+`"`, 10)
		require.NoError(t, err)
		assert.Contains(t, keys, cm.key, "indexed")

		// another client reads it and labels it, in All Mail
		view, ok := viewLabel(cm)
		require.True(t, ok)
		_, err = other.Select(view.Name, nil).Wait()
		require.NoError(t, err)
		require.NoError(t, other.Store(view.UIDSet(), &imap.StoreFlags{Op: imap.StoreFlagsAdd, Silent: true, Flags: []imap.Flag{imap.FlagSeen}}, nil).Close())
		require.NoError(t, other.StoreGmailLabels(view.UIDSet(), &imap.StoreGmailLabels{Op: imap.StoreFlagsAdd, Silent: true, Labels: []string{"Family"}}).Close())
		syncAll()
		a.AsyncLock()
		assert.Contains(t, cm.Flags, imap.FlagSeen)
		assert.True(t, slices.ContainsFunc(cm.Labels, func(l Label) bool { return l.Name == "Family" }))
		a.AsyncUnlock()

		// another client deletes it, out of the Trash
		md, err := other.Move(view.UIDSet(), "[Gmail]/Trash").Wait()
		require.NoError(t, err)
		_, err = other.Select("[Gmail]/Trash", nil).Wait()
		require.NoError(t, err)
		require.NoError(t, other.Store(md.DestUIDs, &imap.StoreFlags{Op: imap.StoreFlagsAdd, Silent: true, Flags: []imap.Flag{imap.FlagDeleted}}, nil).Close())
		require.NoError(t, other.Expunge().Close())
		file := filepath.Join(a.messagesDir(you), cm.Filename)
		_, err = os.Stat(file)
		require.NoError(t, err)
		syncAll()
		a.AsyncLock()
		assert.Nil(t, a.cache[you][cm.key], "gone")
		a.AsyncUnlock()
		_, err = os.Stat(file)
		assert.Error(t, err, "its file is removed")
		keys, err = a.searches[you].search(`subject:"`+cm.Subject+`"`, 10)
		require.NoError(t, err)
		assert.NotContains(t, keys, cm.key)
		stored, err := a.stores[you].load()
		require.NoError(t, err)
		assert.NotContains(t, stored, cm.key)
	})
}

// TestStoreReload tests that the messages are loaded from the store when
// the app starts again, before syncing.
func TestStoreReload(t *testing.T) {
	c := newIMAP(t)
	b, a := newTestApp(t, c)
	dir := a.dataDirectory
	core.RenderImage(b, func() {
		require.NoError(t, a.cacheMessages())
	})
	a.closeStores()
	b2, a2 := newTestApp(t, c)
	a2.dataDirectory = dir
	core.RenderImage(b2, func() {
		require.NoError(t, a2.openLocal(testEmail))
		assert.Len(t, a2.cache[testEmail], 3, "before syncing")
		hello := messageBySubject(a2, "Hello")
		require.NotNil(t, hello)
		assert.Equal(t, []string{"INBOX", "Work"}, labelNames(hello))
		assert.NotEmpty(t, hello.Filename)
	})
}

// TestSearchMessages tests searching the messages of an account from the
// search field of the list, and clearing the search.
func TestSearchMessages(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	old := *Settings
	t.Cleanup(func() { *Settings = old })
	t.Cleanup(core.TrackWindows())
	b := core.NewBody()
	a := NewApp(b)
	require.NoError(t, a.StartTesting(TestingOptions{Seed: 1}))
	t.Cleanup(a.testServer.Close)
	t.Cleanup(a.closeStores)
	you := Settings.Accounts[0].Email
	core.RenderImage(b, func() {
		for {
			a.syncMore = false
			require.NoError(t, a.syncAccount(you))
			if !a.syncMore {
				break
			}
		}
	}, core.Locked(b, func() {
		a.showMailbox(you, "INBOX")
		inbox := len(a.listRows)
		a.searchField.SetText("from:carol")
		a.searchField.SendChange()
		require.NotEmpty(t, a.listRows)
		for _, r := range a.listRows {
			assert.False(t, r.conversation, "each message on its own")
			assert.Contains(t, r.From, "Carol", r.Subject)
		}
		assert.Greater(t, len(a.listRows), inbox/2, "in all of the mailboxes")
		a.SearchMessages(`"go-nuts" -proposal`)
		require.NotEmpty(t, a.listRows)
		for _, r := range a.listRows {
			assert.NotContains(t, r.Subject, "proposal")
		}
		a.SearchMessages("nothingmatchesthis")
		assert.Empty(t, a.listRows)
		a.SearchMessages("")
		assert.Len(t, a.listRows, inbox, "the mailbox again")
		a.SearchMessages("from:carol")
		a.showMailbox(you, "Family")
		assert.Nil(t, a.searchResults, "another mailbox ends the search")
		assert.Equal(t, "", a.searchField.Text())
	}))
}

// TestSearchTyping tests typing in the search field while reading a
// message: the keys edit the text, and Backspace does not delete the
// message, while the shortcuts with Command still apply.
func TestSearchTyping(t *testing.T) {
	b, a := newTestApp(t, newIMAP(t))
	var hello *CacheMessage
	var labels []string
	keys := func(chords ...key.Chord) func() {
		return func() {
			src := b.Scene.RenderWindow().SystemWindow.Events()
			for _, c := range chords {
				r, code, mods, err := c.PlatformChord().Decode()
				require.NoError(t, err)
				src.KeyChord(r, code, mods)
			}
		}
	}
	core.RenderImage(b, func() {
		require.NoError(t, a.cacheMessages())
		hello = messageBySubject(a, "Hello")
		labels = labelNames(hello)
	}, core.Locked(b, func() {
		a.Update()
		a.list.SelectIndexEvent(0, events.SelectOne)
		require.NotNil(t, a.readMessage)
		a.FocusSearch()
	}), keys("h", "e", "l", "x", "Backspace", "l", "o"), core.Locked(b, func() {
		assert.Equal(t, "hello", a.searchField.Text())
		assert.Equal(t, labels, labelNames(hello), "not deleted")
		assert.Nil(t, a.composeMessage)
	}), keys("Command+R"), core.Locked(b, func() {
		assert.NotNil(t, a.composeMessage, "reply, from the search field")
	}))
}

// TestAccounts tests an account that is not signed in, which is not synced
// until it is, and removing an account with its local mail.
func TestAccounts(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	oldFile, oldAccounts := Settings.File, Settings.Accounts
	t.Cleanup(func() { Settings.File, Settings.Accounts = oldFile, oldAccounts })
	t.Cleanup(core.TrackWindows())
	b := core.NewBody()
	a := NewApp(b)
	require.NoError(t, a.StartTesting(TestingOptions{Seed: 1}))
	t.Cleanup(a.testServer.Close)
	t.Cleanup(a.closeStores)
	you, alice := Settings.Accounts[0].Email, Settings.Accounts[1].Email
	gmail := "new@gmail.com" // signs in with OAuth
	Settings.addAccount(Account{Type: Gmail, Email: gmail})
	core.RenderImage(b, func() {
		assert.True(t, a.signedIn(you), "with a password")
		assert.False(t, a.signedIn(gmail))
		assert.Equal(t, []string{gmail}, a.signInSaved(), "no saved token")
		a.syncMore = false
		require.NoError(t, a.cacheMessages(), "skips the account that is not signed in")
		assert.NotEmpty(t, a.cache[alice])
		assert.Nil(t, a.stores[gmail])
		err := a.send(gmail, []string{you}, strings.NewReader("Subject: x\r\n\r\nx\r\n"))
		assert.ErrorContains(t, err, "not signed in to new@gmail.com")
	}, core.Locked(b, func() {
		a.showMailbox(alice, "INBOX")
		a.RemoveAccount(alice)
		assert.Equal(t, []string{you, Settings.Accounts[1].Email, gmail}, Settings.Emails())
		assert.NotContains(t, Settings.Emails(), alice)
		assert.Equal(t, you, a.currentEmail, "the first account is shown")
		assert.Nil(t, a.cache[alice])
	}), func() {
		require.Eventually(t, func() bool {
			_, err := os.Stat(a.messagesDir(alice))
			return os.IsNotExist(err)
		}, 5*time.Second, 10*time.Millisecond, "the local mail is deleted")
		a.syncMore = false
		require.NoError(t, a.cacheMessages())
		assert.Nil(t, a.stores[alice], "no longer synced")
	})
}

// TestIndexMissing tests adding the downloaded messages that are not in
// the search index to it, a limited number at a time.
func TestIndexMissing(t *testing.T) {
	b, a := newTestApp(t, newIMAP(t))
	core.RenderImage(b, func() {
		require.NoError(t, a.cacheMessages())
		si := a.searches[testEmail]
		hello := messageBySubject(a, "Hello")
		require.NotEmpty(t, hello.Filename)
		require.True(t, si.has(hello.key))
		require.NoError(t, si.remove(hello.key))
		a.indexChecked = map[string]map[string]bool{}

		n := len(a.cache[testEmail])
		a.syncMore = false
		require.NoError(t, a.indexMissing(testEmail, 1))
		assert.True(t, a.syncMore, "more to check")
		a.syncMore = false
		require.NoError(t, a.indexMissing(testEmail, n))
		assert.False(t, a.syncMore)
		assert.True(t, si.has(hello.key), "indexed again")
		assert.Len(t, a.indexChecked[testEmail], n)
	})
}

// TestMixedTextParts tests a message with text, an attachment, and then a
// blank text part, as Apple Mail sends: the first text is shown.
func TestMixedTextParts(t *testing.T) {
	b, a := newTestApp(t, newIMAP(t))
	msg := "From: me@example.com\r\nSubject: Re: files\r\nMIME-Version: 1.0\r\n" +
		"Content-Type: multipart/mixed; boundary=\"B\"\r\n\r\n" +
		"--B\r\nContent-Type: text/plain; charset=us-ascii\r\n\r\nHere are the two files.\r\n" +
		"--B\r\nContent-Disposition: attachment; filename=\"a.docx\"\r\nContent-Type: application/octet-stream\r\n\r\nxyz\r\n" +
		"--B\r\nContent-Type: text/plain; charset=us-ascii\r\n\r\n\r\n\r\n" +
		"--B--\r\n"
	require.NoError(t, os.MkdirAll(a.messagesDir(testEmail), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(a.messagesDir(testEmail), "mixed"), []byte(msg), 0o600))
	cm := &CacheMessage{Filename: "mixed"}
	dmf := NewDisplayMessageFrame(b)
	dmf.Message = cm
	w := core.NewFrame(b)
	require.NoError(t, dmf.displayMessageContents(w))
	assert.Equal(t, "Here are the two files.", cm.parsed.plain)
	require.Len(t, cm.parsed.attachments, 1)
	require.NoError(t, dmf.displayMessageContents(w))
	assert.Equal(t, "Here are the two files.", cm.parsed.plain, "not twice when shown again")
}

// newTestingApp returns an app in testing mode, with a seeded test server.
func newTestingApp(t *testing.T) (*core.Body, *App) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	oldFile, oldAccounts := Settings.File, Settings.Accounts
	t.Cleanup(func() { Settings.File, Settings.Accounts = oldFile, oldAccounts })
	t.Cleanup(core.TrackWindows())
	Settings.ShownAccount, Settings.ShownMailbox = "", "" // not that of another test
	b := core.NewBody()
	a := NewApp(b)
	require.NoError(t, a.StartTesting(TestingOptions{Seed: 1}))
	// the settings of testing mode are kept between runs, in the data of
	// the app (shared by the tests, and the testing mode of the user): a
	// file of this test's own, and not the mailbox shown by another one
	Settings.File = filepath.Join(t.TempDir(), "settings.toml")
	Settings.ShownAccount, Settings.ShownMailbox = "", ""
	t.Cleanup(a.testServer.Close)
	t.Cleanup(a.closeStores)
	return b, a
}

// TestReconnect tests connecting again after the connection to the
// server is closed, as after the computer sleeps.
func TestReconnect(t *testing.T) {
	b, a := newTestingApp(t)
	you := Settings.Accounts[0].Email
	core.RenderImage(b, func() {
		a.syncMore = false
		require.NoError(t, a.cacheMessages())
		old := a.imapClient[you]
		require.NotNil(t, old)
		old.Close() // the connection is lost
		a.syncMore = false
		require.NoError(t, a.cacheMessages(), "connects again")
		assert.NotNil(t, a.imapClient[you])
		assert.NotSame(t, old, a.imapClient[you])

		// a failed sync drops the connection, for the next sync
		c := a.imapClient[you]
		a.setSelected(you, "")
		assert.Error(t, a.syncMailbox(you, "No such mailbox", false))
		a.dropClient(you, c)
		assert.Nil(t, a.imapClient[you])
		a.syncMore = false
		require.NoError(t, a.cacheMessages())
		assert.NotNil(t, a.imapClient[you])
	})
}

// TestIdle tests the connection that idles on the INBOX, which syncs
// when new mail comes.
func TestIdle(t *testing.T) {
	b, a := newTestingApp(t)
	you := Settings.Accounts[0].Email
	core.RenderImage(b, func() { // the lock of the window needs it shown
		a.startIdle(you)
		select {
		case <-a.syncNow: // not yet
		default:
		}
		require.Eventually(t, func() bool {
			require.NoError(t, a.testServer.RandomMessage(you)) // new mail
			select {
			case <-a.syncNow:
				return true
			case <-time.After(200 * time.Millisecond):
				return false
			}
		}, 10*time.Second, 10*time.Millisecond, "a sync once idling")
	}, core.Locked(b, func() {
		a.stopIdle(you)
		assert.Empty(t, a.idleStop)
	}))
}

// TestStatusBar tests the status of the syncing below the mailboxes.
func TestStatusBar(t *testing.T) {
	b, a := newTestApp(t, newIMAP(t))
	texts := func() []string {
		var res []string
		for w := range a.statusBar.AllWidgets() {
			switch w := w.(type) {
			case *core.Text:
				if _, ok := w.Parent.(*core.Button); !ok { // not the text of a button
					res = append(res, w.Text)
				}
			case *core.Button:
				res = append(res, w.Text)
			case *core.Meter:
				res = append(res, fmt.Sprintf("meter %.2f", w.Value))
			}
		}
		return res
	}
	var heights []float32
	height := func() { heights = append(heights, a.statusBar.Geom.Size.Actual.Total.Y) }
	core.RenderImage(b, func() {
		a.progressStatus(testEmail, "downloading messages", 250, 1000, 50)
	}, core.Locked(b, func() {
		height()
		assert.Equal(t, []string{testEmail + ": downloading messages, 250 of 1,000", "meter 0.25"}, texts())
	}), func() {
		a.syncDone(errors.New("dial tcp: no route to host"))
	}, core.Locked(b, func() {
		height()
		assert.Equal(t, []string{"Not synced: dial tcp: no route to host"}, texts())
	}), func() {
		a.syncDone(nil)
	}, core.Locked(b, func() {
		height()
		got := texts()
		require.Len(t, got, 1)
		assert.True(t, strings.HasPrefix(got[0], "Updated "), got[0])
	}))
	require.Len(t, heights, 3)
	assert.NotZero(t, heights[0])
	assert.Equal(t, heights[0], heights[1], "a fixed height")
	assert.Equal(t, heights[0], heights[2], "a fixed height")
}

// TestReadDownloading tests that a message that is read before it is
// downloaded shows that it is being downloaded, and then the message, once
// it is (which the updates of the sync, without the message being read,
// did not show).
func TestReadDownloading(t *testing.T) {
	b, a := newTestApp(t, newIMAP(t))
	var hello *CacheMessage
	// shows returns whether the panel of the message being read has the
	// given text
	shows := func(text string) bool {
		found := false
		a.readPanel.WalkDown(func(n tree.Node) bool {
			if tx, ok := n.(*core.Text); ok && strings.Contains(tx.Text, text) {
				found = true
			}
			return !found
		})
		return found
	}
	core.RenderImage(b, func() {
		require.NoError(t, a.cacheMessages())
		hello = messageBySubject(a, "Hello")
		require.NotNil(t, hello)
	}, core.Locked(b, func() {
		hello.Filename = "" // not downloaded yet
		a.readMessage = hello
		a.Update()
		assert.True(t, shows("Downloading the message"))
	}), func() {
		require.Eventually(t, func() bool {
			a.AsyncLock()
			defer a.AsyncUnlock()
			return hello.Filename != "" && !shows("Downloading the message")
		}, 5*time.Second, 10*time.Millisecond)
		a.AsyncLock()
		defer a.AsyncUnlock()
		shown := false
		a.readPanel.WalkDown(func(n tree.Node) bool {
			if sm, ok := n.AsTree().Property("readMessage").(shownMessage); ok && sm.message == hello {
				shown = true
			}
			return !shown
		})
		assert.True(t, shown, "the message")
	})
}

// TestMailKeys tests the key map of Cogent Mail ([SettingsData.Keys]):
// Command+W replies to all, which does not close the window without a
// message, Command+F forwards, and the keys can be changed.
func TestMailKeys(t *testing.T) {
	b, a := newTestApp(t, newIMAP(t))
	old := Settings.Keys
	t.Cleanup(func() { Settings.Keys = old })
	Settings.Keys = DefaultKeys()
	press := func(c key.Chord) func() {
		return func() {
			r, code, mods, err := c.PlatformChord().Decode()
			require.NoError(t, err)
			b.Scene.RenderWindow().SystemWindow.Events().KeyChord(r, code, mods)
		}
	}
	core.RenderImage(b, func() {
		require.NoError(t, a.cacheMessages())
	}, core.Locked(b, func() {
		a.Update()
		a.list.SetFocusQuiet()
	}), press("Command+W"), core.Locked(b, func() {
		assert.NotNil(t, b.Scene.RenderWindow(), "not closed")
		assert.Nil(t, a.composeMessage, "no message")
		a.list.SelectIndexEvent(0, events.SelectOne)
	}), press("Command+W"), core.Locked(b, func() {
		require.NotNil(t, a.composeMessage, "reply all")
		assert.Equal(t, "Re: Hello", a.composeMessage.Subject)
		sc, _, ok := shortcutFor(key.Chord("Command+F").PlatformChord())
		assert.True(t, ok)
		assert.Equal(t, KeyForward, sc.kf)
		Settings.Keys = KeyMap{"Command+E": KeyReplyAll}
		assert.Equal(t, key.Chord("Command+E"), shortcutOf("ReplyAll"))
		assert.Empty(t, shortcutOf("Forward"))
		sc, _, ok = shortcutFor(key.Chord("Command+E").PlatformChord())
		assert.True(t, ok)
		assert.Equal(t, KeyReplyAll, sc.kf)
		_, _, ok = shortcutFor(key.Chord("Command+F").PlatformChord())
		assert.False(t, ok, "not in the changed map")
	}))
}

// TestDefaultKeys tests the defaults of the keyboard shortcuts: all of
// them at first, those removed are not added again, those of new actions
// are, for chords that are free, and text keys, as Command+X for Cut, are
// not shortcuts while typing.
func TestDefaultKeys(t *testing.T) {
	sd := &SettingsData{}
	sd.Defaults()
	sd.addDefaultKeys()
	assert.Equal(t, DefaultKeys(), sd.Keys)
	delete(sd.Keys, "Command+X")
	sd.addDefaultKeys()
	assert.NotContains(t, sd.Keys, key.Chord("Command+X"), "removed")

	sd.KeysKnown = slices.DeleteFunc(sd.KeysKnown, func(f KeyFunctions) bool { return f == KeyRefresh || f == KeyStar })
	delete(sd.Keys, "Command+Shift+N")
	sd.Keys["Command+Shift+L"] = KeyReply // its chord is used
	sd.addDefaultKeys()
	assert.Equal(t, KeyRefresh, sd.Keys["Command+Shift+N"], "a new action")
	assert.Equal(t, KeyReply, sd.Keys["Command+Shift+L"], "not changed")

	x := key.Chord("Command+X")
	assert.True(t, isTextKey(x.PlatformChord(), x), "Cut")
	r := key.Chord("Command+R")
	assert.False(t, isTextKey(r.PlatformChord(), r))
	o := key.Chord("Control+O")
	assert.True(t, isTextKey(o.PlatformChord(), o), "without Command")
}

// TestBuiltinCredentials tests that the built-in OAuth client of Google is
// used when the environment does not set one, decoded, and not otherwise.
func TestBuiltinCredentials(t *testing.T) {
	t.Setenv("GOOGLE_OAUTH2_CLIENT_ID", "")
	t.Setenv("GOOGLE_OAUTH2_CLIENT_SECRET", "")
	useBuiltinCredentials()
	id := os.Getenv("GOOGLE_OAUTH2_CLIENT_ID")
	assert.True(t, strings.HasSuffix(id, ".apps.googleusercontent.com"), "a Google client ID")
	assert.True(t, strings.HasPrefix(os.Getenv("GOOGLE_OAUTH2_CLIENT_SECRET"), "GOCSPX"), "a Google client secret")
	t.Setenv("GOOGLE_OAUTH2_CLIENT_ID", "mine")
	useBuiltinCredentials()
	assert.Equal(t, "mine", os.Getenv("GOOGLE_OAUTH2_CLIENT_ID"), "one's own, if set")
}

// TestNewestFirst tests the order of the rows of the list of messages:
// the newest first by their whole times, not the minutes shown, and then
// by their keys, so that the order is the same each time.
func TestNewestFirst(t *testing.T) {
	now := time.Now().Truncate(time.Minute)
	row := func(key string, d time.Duration) *messageRow {
		cm := &CacheMessage{key: key, Envelope: imap.Envelope{Date: now.Add(d)}}
		return &messageRow{message: cm, Date: cm.Date.Format(listDateFormat)}
	}
	a, b, c := row("a", 10*time.Second), row("b", 30*time.Second), row("c", 10*time.Second)
	for range 20 {
		rows := []*messageRow{a, c, b}
		rand.Shuffle(len(rows), func(i, j int) { rows[i], rows[j] = rows[j], rows[i] })
		slices.SortFunc(rows, newestFirst)
		assert.Equal(t, []*messageRow{b, a, c}, rows, "the same minute, in the same order")
	}
}
