// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
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
