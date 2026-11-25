// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package testserver

import (
	"slices"
	"strings"
	"testing"

	"github.com/cogentcore/go-imap/v2"
	"github.com/cogentcore/go-imap/v2/imapclient"
	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newServer(t *testing.T, opts Options) *Server {
	t.Helper()
	s, err := New(opts)
	require.NoError(t, err)
	t.Cleanup(s.Close)
	return s
}

// login returns an IMAP client logged in to the given account.
func login(t *testing.T, s *Server, account string) *imapclient.Client {
	t.Helper()
	c, err := imapclient.DialInsecure(s.IMAPAddr, nil)
	require.NoError(t, err)
	t.Cleanup(func() { c.Close() })
	require.NoError(t, c.Login(account, Password).Wait())
	return c
}

// count returns the number of messages in the given mailbox.
func count(t *testing.T, c *imapclient.Client, mailbox string) int {
	t.Helper()
	sd, err := c.Status(mailbox, &imap.StatusOptions{NumMessages: true}).Wait()
	require.NoError(t, err)
	return int(*sd.NumMessages)
}

// send sends a message with SMTP as the given account.
func send(t *testing.T, s *Server, from string, to []string, subject string) error {
	t.Helper()
	c, err := smtp.Dial(s.SMTPAddr)
	require.NoError(t, err)
	defer c.Close()
	if err := c.Auth(sasl.NewPlainClient("", from, Password)); err != nil {
		return err
	}
	msg := "From: " + from + "\r\nTo: " + strings.Join(to, ", ") + "\r\nSubject: " + subject +
		"\r\nMessage-ID: <" + subject + "@mail.test>\r\n\r\nHello\r\n"
	return c.SendMail(from, to, strings.NewReader(msg))
}

func TestDeliver(t *testing.T) {
	s := newServer(t, Options{})
	var delivered []string
	s.OnDeliver = func(accounts []string) { delivered = append(delivered, accounts...) }
	you, alice := DefaultAccounts[0], DefaultAccounts[1]

	require.NoError(t, send(t, s, you, []string{alice, "stranger@example.org"}, "hi"))
	cy, ca := login(t, s, you), login(t, s, alice)
	assert.Equal(t, 1, count(t, ca, "INBOX"))
	assert.Equal(t, 1, count(t, ca, "[Gmail]/All Mail"))
	assert.Equal(t, 1, count(t, cy, "[Gmail]/Sent Mail"))
	assert.Equal(t, 1, count(t, cy, "INBOX"), "the bounce")
	assert.Equal(t, 2, count(t, cy, "[Gmail]/All Mail"))
	assert.Equal(t, []string{alice, you}, delivered)

	require.NoError(t, send(t, s, you, []string{you}, "self"))
	assert.Equal(t, 2, count(t, cy, "INBOX"), "to self is in the INBOX too")

	assert.Error(t, func() error {
		c, err := smtp.Dial(s.SMTPAddr)
		require.NoError(t, err)
		defer c.Close()
		return c.Auth(sasl.NewPlainClient("", you, "wrong"))
	}(), "the wrong password")
}

func TestPlain(t *testing.T) {
	s := newServer(t, Options{Plain: true, Accounts: []string{"Me@Plain.test"}})
	require.NoError(t, send(t, s, "me@plain.test", []string{"me@plain.test"}, "hi"))
	c := login(t, s, "me@plain.test")
	assert.Equal(t, 1, count(t, c, "Sent"))
	assert.Equal(t, 1, count(t, c, "INBOX"))
	list, err := c.List("", "*", nil).Collect()
	require.NoError(t, err)
	for _, l := range list {
		assert.False(t, strings.HasPrefix(l.Mailbox, "[Gmail]"))
	}
}

func TestSeedSaveLoad(t *testing.T) {
	s := newServer(t, Options{})
	require.NoError(t, s.Seed(1))
	you := DefaultAccounts[0]
	c := login(t, s, you)
	list, err := c.List("", "*", nil).Collect()
	require.NoError(t, err)
	var names []string
	for _, l := range list {
		names = append(names, l.Mailbox)
	}
	for _, mb := range []string{"Projects", "Projects/Alpha", "Admin/Receipts", "Lists/go-nuts"} {
		assert.True(t, slices.Contains(names, mb), mb)
	}
	all := count(t, c, "[Gmail]/All Mail")
	assert.Greater(t, all, 150)
	assert.Greater(t, count(t, c, "INBOX"), 5)
	assert.Greater(t, count(t, c, "[Gmail]/Sent Mail"), 10)
	assert.Greater(t, count(t, login(t, s, DefaultAccounts[1]), "INBOX"), 0, "alice has your messages")

	dir := t.TempDir()
	require.NoError(t, s.Save(dir))
	s2 := newServer(t, Options{})
	ok, err := s2.Load(dir)
	require.NoError(t, err)
	assert.True(t, ok)
	c2 := login(t, s2, you)
	assert.Equal(t, all, count(t, c2, "[Gmail]/All Mail"))
	assert.Equal(t, count(t, c, "Projects/Alpha"), count(t, c2, "Projects/Alpha"))

	ok, err = newServer(t, Options{}).Load(t.TempDir())
	require.NoError(t, err)
	assert.False(t, ok, "nothing saved")

	require.NoError(t, s.RandomMessage(you))
	assert.Equal(t, all+1, count(t, c, "[Gmail]/All Mail"))
}
