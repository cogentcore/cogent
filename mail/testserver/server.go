// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package testserver provides a local, simulated mail server for testing
// Cogent Mail without a real email account: an in-memory IMAP server and an
// SMTP server that delivers messages between its accounts. In Gmail mode
// (the default), it imitates the Gmail mailboxes: [Gmail]/All Mail has a
// copy of every message, sent messages go to [Gmail]/Sent Mail, and so on.
// Messages to addresses that are not accounts of the server bounce back
// to the sender.
package testserver

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
	"github.com/emersion/go-imap/v2/imapserver/imapmemserver"
	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
)

// Password is the password of all of the accounts of the server.
const Password = "password"

// Options are the options for [New].
type Options struct {
	// Accounts are the email addresses of the accounts of the server.
	// The default is [DefaultAccounts].
	Accounts []string

	// Plain uses the mailboxes of a plain IMAP server (Sent, Trash, Junk,
	// Drafts) instead of the Gmail ones, and does not keep All Mail.
	Plain bool

	// IMAPAddr and SMTPAddr are the addresses to listen on.
	// The default is a free port on 127.0.0.1.
	IMAPAddr, SMTPAddr string
}

// DefaultAccounts are the default accounts of the server.
var DefaultAccounts = []string{"you@mail.test", "alice@mail.test", "bob@mail.test"}

// Server is a running simulated mail server.
type Server struct {
	Options

	// IMAPAddr and SMTPAddr are the addresses the servers listen on.
	IMAPAddr, SMTPAddr string

	// OnDeliver is called after a message is delivered to any account,
	// with the addresses of the accounts that received it.
	OnDeliver func(accounts []string)

	mem   *imapmemserver.Server
	users map[string]*imapmemserver.User
	imap  *imapserver.Server
	smtp  *smtp.Server

	mu       sync.Mutex
	msgCount int
}

// New starts a new server with the given options. It has the standard
// mailboxes, but no messages: see [Server.Seed].
func New(opts Options) (*Server, error) {
	if len(opts.Accounts) == 0 {
		opts.Accounts = DefaultAccounts
	}
	for i, a := range opts.Accounts {
		opts.Accounts[i] = strings.ToLower(a)
	}
	s := &Server{Options: opts, mem: imapmemserver.New(), users: map[string]*imapmemserver.User{}}
	for _, a := range opts.Accounts {
		u := imapmemserver.NewUser(a, Password)
		for _, mb := range s.StandardMailboxes() {
			if err := u.Create(mb, nil); err != nil {
				return nil, err
			}
		}
		s.mem.AddUser(u)
		s.users[a] = u
	}

	s.imap = imapserver.New(&imapserver.Options{
		NewSession: func(c *imapserver.Conn) (imapserver.Session, *imapserver.GreetingData, error) {
			return s.mem.NewSession(), nil, nil
		},
		Caps:         imap.CapSet{imap.CapIMAP4rev1: {}, imap.CapIMAP4rev2: {}},
		InsecureAuth: true,
	})
	iln, err := listen(opts.IMAPAddr)
	if err != nil {
		return nil, err
	}
	s.IMAPAddr = iln.Addr().String()
	go s.imap.Serve(iln)

	s.smtp = smtp.NewServer(smtp.BackendFunc(func(c *smtp.Conn) (smtp.Session, error) {
		return &smtpSession{s: s}, nil
	}))
	s.smtp.Domain = "mail.test"
	s.smtp.AllowInsecureAuth = true
	sln, err := listen(opts.SMTPAddr)
	if err != nil {
		s.imap.Close()
		return nil, err
	}
	s.SMTPAddr = sln.Addr().String()
	go s.smtp.Serve(sln)
	return s, nil
}

func listen(addr string) (net.Listener, error) {
	if addr == "" {
		addr = "127.0.0.1:0"
	}
	return net.Listen("tcp", addr)
}

// Close stops the server.
func (s *Server) Close() {
	s.imap.Close()
	s.smtp.Close()
}

// StandardMailboxes returns the mailboxes that every account has.
func (s *Server) StandardMailboxes() []string {
	if s.Plain {
		return []string{"INBOX", "Drafts", "Sent", "Junk", "Trash"}
	}
	return []string{"INBOX", "[Gmail]", "[Gmail]/All Mail", "[Gmail]/Drafts",
		"[Gmail]/Important", "[Gmail]/Sent Mail", "[Gmail]/Spam",
		"[Gmail]/Starred", "[Gmail]/Trash"}
}

// SentMailbox returns the name of the mailbox of sent messages.
func (s *Server) SentMailbox() string {
	if s.Plain {
		return "Sent"
	}
	return "[Gmail]/Sent Mail"
}

// HasAccount returns whether the given address is an account of the server.
func (s *Server) HasAccount(addr string) bool {
	return s.users[strings.ToLower(addr)] != nil
}

// CreateMailbox creates the given mailbox (with any parents) for the
// given account, if it does not already exist.
func (s *Server) CreateMailbox(account, mailbox string) error {
	u := s.users[account]
	if u == nil {
		return fmt.Errorf("testserver: no account %q", account)
	}
	parts := strings.Split(mailbox, "/")
	for i := range parts {
		name := strings.Join(parts[:i+1], "/")
		if _, err := u.Status(name, &imap.StatusOptions{}); err == nil {
			continue
		}
		if err := u.Create(name, nil); err != nil {
			return err
		}
	}
	return nil
}

// Append adds the given message to the given mailboxes of the given account,
// creating them as needed. In Gmail mode, it is also added to All Mail.
// The time is the internal date; zero is now.
func (s *Server) Append(account string, mailboxes []string, msg []byte, flags []imap.Flag, t time.Time) error {
	u := s.users[account]
	if u == nil {
		return fmt.Errorf("testserver: no account %q", account)
	}
	if !s.Plain {
		mailboxes = append(mailboxes, "[Gmail]/All Mail")
	}
	for _, mb := range mailboxes {
		if err := s.CreateMailbox(account, mb); err != nil {
			return err
		}
		_, err := u.Append(mb, bytes.NewReader(msg), &imap.AppendOptions{Flags: flags, Time: t})
		if err != nil {
			return fmt.Errorf("testserver: appending to %q of %q: %w", mb, account, err)
		}
	}
	return nil
}

// Deliver delivers the given message from the given sender to the given
// recipients, as the SMTP server does: it goes to the INBOX of each
// recipient that is an account, and a copy goes to the sent mailbox of the
// sender if it is an account. Other recipients get a bounce message back.
func (s *Server) Deliver(from string, to []string, msg []byte) error {
	from = strings.ToLower(from)
	msg = s.addMessageID(msg)
	var got, unknown []string
	toSelf := false
	for _, r := range to {
		r = strings.ToLower(r)
		if r == from {
			toSelf = true
		}
		if s.HasAccount(r) {
			got = append(got, r)
		} else {
			unknown = append(unknown, r)
		}
	}
	if s.HasAccount(from) {
		err := s.Append(from, []string{s.SentMailbox()}, msg, []imap.Flag{imap.FlagSeen}, time.Time{})
		if err != nil {
			return err
		}
		if !toSelf {
			got = append(got, from)
		}
	}
	for _, r := range got {
		if r == from && !toSelf {
			continue
		}
		if err := s.Append(r, []string{"INBOX"}, msg, nil, time.Time{}); err != nil {
			return err
		}
	}
	if len(unknown) > 0 && s.HasAccount(from) {
		if err := s.Append(from, []string{"INBOX"}, s.bounce(from, unknown, msg), nil, time.Time{}); err != nil {
			return err
		}
	}
	slog.Info("testserver: delivered", "from", from, "to", to, "bounced", unknown)
	if s.OnDeliver != nil && len(got) > 0 {
		s.OnDeliver(got)
	}
	return nil
}

// bounce returns a bounce message for the given unknown recipients.
func (s *Server) bounce(to string, unknown []string, msg []byte) []byte {
	var b bytes.Buffer
	fmt.Fprintf(&b, "From: Mail Delivery Subsystem <mailer-daemon@mail.test>\r\n")
	fmt.Fprintf(&b, "To: %s\r\n", to)
	fmt.Fprintf(&b, "Subject: Delivery Status Notification (Failure)\r\n")
	fmt.Fprintf(&b, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	fmt.Fprintf(&b, "Message-ID: %s\r\n", s.newMessageID())
	fmt.Fprintf(&b, "Content-Type: text/plain; charset=utf-8\r\n\r\n")
	fmt.Fprintf(&b, "This is the simulated test mail server: your message was not delivered to\r\n%s\r\n", strings.Join(unknown, "\r\n"))
	fmt.Fprintf(&b, "because they are not accounts of the test server.\r\n\r\n")
	fmt.Fprintf(&b, "----- Original message headers -----\r\n\r\n")
	if hdr, _, ok := bytes.Cut(msg, []byte("\r\n\r\n")); ok {
		b.Write(hdr)
	}
	return b.Bytes()
}

// addMessageID returns the given message with a new Message-ID header
// if it does not have one, as Gmail adds.
func (s *Server) addMessageID(msg []byte) []byte {
	hdr, _, _ := bytes.Cut(msg, []byte("\r\n\r\n"))
	for line := range strings.SplitSeq(string(hdr), "\r\n") {
		if strings.HasPrefix(strings.ToLower(line), "message-id:") {
			return msg
		}
	}
	return append([]byte("Message-ID: "+s.newMessageID()+"\r\n"), msg...)
}

// newMessageID returns a new unique Message-ID with angle brackets.
func (s *Server) newMessageID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.msgCount++
	return fmt.Sprintf("<%d.%d@mail.test>", time.Now().UnixNano(), s.msgCount)
}

// smtpSession is a session of the SMTP server.
type smtpSession struct {
	s    *Server
	from string
	to   []string
}

func (ss *smtpSession) AuthMechanisms() []string { return []string{sasl.Plain} }

func (ss *smtpSession) Auth(mech string) (sasl.Server, error) {
	return sasl.NewPlainServer(func(identity, username, password string) error {
		if !ss.s.HasAccount(username) || password != Password {
			return fmt.Errorf("testserver: invalid username or password")
		}
		return nil
	}), nil
}

func (ss *smtpSession) Mail(from string, opts *smtp.MailOptions) error {
	ss.from = from
	return nil
}

func (ss *smtpSession) Rcpt(to string, opts *smtp.RcptOptions) error {
	ss.to = append(ss.to, to)
	return nil
}

func (ss *smtpSession) Data(r io.Reader) error {
	msg, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	return ss.s.Deliver(ss.from, ss.to, msg)
}

func (ss *smtpSession) Reset() {
	ss.from = ""
	ss.to = nil
}

func (ss *smtpSession) Logout() error { return nil }
