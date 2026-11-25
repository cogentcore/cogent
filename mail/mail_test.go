// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"fmt"
	"image"
	"io"
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
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-imap/v2/imapserver"
	"github.com/emersion/go-imap/v2/imapserver/imapmemserver"
	gomail "github.com/emersion/go-message/mail"
	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	Settings.Accounts = []string{testEmail}
	t.Cleanup(core.TrackWindows())
	b := core.NewBody()
	a := NewApp(b)
	b.AddTopBar(func(bar *core.Frame) {
		core.NewToolbar(bar).Maker(a.MakeToolbar)
	})
	a.currentEmail = testEmail
	a.imapClient[testEmail] = c
	a.imapMu[testEmail] = &sync.Mutex{}
	require.NoError(t, os.MkdirAll(filepath.Join(core.TheApp.AppDataDir(), "mail", filenameBase32(testEmail)), 0o700))
	require.NoError(t, os.MkdirAll(filepath.Dir(a.cacheFilename(testEmail)), 0o700))
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
		// the cache file is saved
		data, err := os.ReadFile(a.cacheFilename(testEmail))
		require.NoError(t, err)
		assert.Contains(t, string(data), "Report")
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
		assert.Len(t, a.listCache, 1)
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
	assert.Equal(t, "MFRGG", filenameBase32("abc"))
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

		a.composeMessage.From = nil
		assert.Error(t, a.Send(), "no sender")
		a.composeMessage.From = []*gomail.Address{{Address: testEmail}}
		a.composeMessage.Attachments = []core.Filename{"none.txt"}
		assert.Error(t, a.Send(), "no attachment")
	}))

	// a server that is not there is an error, not a panic
	smtpAddress = "127.0.0.1:1"
	a.composeMessage.Attachments = nil
	assert.Error(t, a.Send())
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
	you, alice := Settings.Accounts[0], Settings.Accounts[1]
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
		assert.Equal(t, you, a.composeMessage.From[0].Address)
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
	t.Cleanup(func() { Settings.FlatList = false })
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

		Settings.FlatList = true
		a.Update()
		require.Len(t, a.listRows, 2)
		assert.False(t, a.listRows[0].conversation)
		assert.Equal(t, "Re: Hello", a.listRows[0].Subject)
		Settings.FlatList = false
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
	you, bob := Settings.Accounts[0], Settings.Accounts[2]
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
	you := Settings.Accounts[0]
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
		assert.Equal(t, []string{"INBOX", "[Gmail]/Drafts", "[Gmail]/Sent Mail", "[Gmail]/Spam", "[Gmail]/Trash",
			"Admin", "Family", "Lists", "Newsletters", "Projects"}, names)
		inbox = node(acct, "INBOX")
		assert.True(t, inbox.StateIs(states.Selected), "the mailbox shown")
		assert.Equal(t, fmt.Sprintf("Inbox (%d)", a.unread[you]["INBOX"]), inbox.Text)
		assert.NotZero(t, a.unread[you]["INBOX"])
		assert.Contains(t, acct.Text, "you@mail.test (")
		assert.Equal(t, icons.Inbox, inbox.Icon)
		projects := node(acct, "Projects")
		assert.Equal(t, icons.Folder, projects.Icon, "with subfolders")
		assert.Equal(t, icons.Label, node(projects, "Projects/Alpha").Icon, "a leaf")
		inbox.SetFocus()
	}), func() {
		b.Scene.RenderWindow().SystemWindow.Events().KeyChord(0, key.CodeDownArrow, 0)
	}, core.Locked(b, func() {
		assert.Equal(t, "[Gmail]/Drafts", a.showLabel, "moved with the keyboard")
		assert.False(t, inbox.StateIs(states.Selected))
		a.showMailbox(Settings.Accounts[1], "INBOX")
	}), core.Locked(b, func() {
		assert.True(t, node(node(a.mailboxes, Settings.Accounts[1]), "INBOX").StateIs(states.Selected))
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
		assert.Equal(t, "\n\n--  \nMe  \nThe Lab\n", a.composeMessage.body)
		a.readMessage = messageBySubject(a, "Report")
		a.readSingle = true
		a.Update()
	}), core.Locked(b, func() {
		a.Reply()
		assert.True(t, strings.HasPrefix(a.composeMessage.body, "\n\n--  \nMe  \nThe Lab\n\n\n> On "), a.composeMessage.body)
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
	Settings.Accounts = []string{"a@x.test", "b@x.test"}
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
		assert.Equal(t, []string{"Reply=Command+R", "Reply all=Command+Shift+R", "Forward=Command+Shift+F",
			"File=Control+O", "Tag=Control+T", "Delete=Backspace", "Mark as read=", "Mark as unread=Command+Shift+U"}, shortcuts)
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
		a.composeMessage.To[0].Address = "dan@example.com"
		a.composeMessage.Cc = []*gomail.Address{{Address: "erin@example.com"}, {}}
		a.composeMessage.Bcc = []*gomail.Address{{Address: "frank@example.com"}, {Address: "DAN@example.com"}}
		a.composeMessage.Subject = "Copies"
		require.NoError(t, a.Send())
		be.mu.Lock()
		defer be.mu.Unlock()
		assert.Equal(t, []string{"dan@example.com", "erin@example.com", "frank@example.com", testEmail}, be.to)
		require.Len(t, be.data, 1)
		assert.Contains(t, be.data[0], "Cc: <erin@example.com>")
		assert.NotContains(t, be.data[0], "frank")
		assert.NotContains(t, be.data[0], "Bcc")
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
	you, alice := Settings.Accounts[0], Settings.Accounts[1]
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
		assert.Equal(t, []string{alice, you, Settings.Accounts[2]}, a.filingAccounts())
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
	var hello, report *CacheMessage
	core.RenderImage(b, func() {
		require.NoError(t, a.cacheMessages())
		hello = messageBySubject(a, "Hello")
		report = messageBySubject(a, "Report")
	}, core.Locked(b, func() {
		a.showMailbox(testEmail, "Work")
		require.Len(t, a.listRows, 2)
		a.list.SelectIndexEvent(0, events.SelectOne)
		a.list.SelectIndexEvent(1, events.ExtendOne)
		require.Len(t, a.selectedRows, 2)
		msgs := a.readMessages()
		assert.Len(t, msgs, 3, "the report, and the conversation")
		assert.Contains(t, msgs, report)
		assert.Contains(t, msgs, hello.replies[0])
		assert.False(t, slices.Contains(hello.Flags, imap.FlagSeen), "not marked as read")
		a.fileTo("Work", "[Gmail]/Trash")
	}), func() {
		require.Eventually(t, func() bool {
			a.imapMu[testEmail].Lock()
			defer a.imapMu[testEmail].Unlock()
			for _, cm := range []*CacheMessage{hello, hello.replies[0], report} {
				if slices.ContainsFunc(cm.Labels, func(l Label) bool { return l.Name == "Work" }) {
					return false
				}
			}
			return true
		}, 5*time.Second, 10*time.Millisecond, "all filed out of Work")
	}, core.Locked(b, func() {
		assert.Empty(t, a.listRows)
		assert.Empty(t, a.selectedRows)
		assert.Contains(t, labelNames(report), "[Gmail]/Trash")
		assert.Contains(t, labelNames(hello), "INBOX", "its other labels stay")
	}))
}
