// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"mime"

	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"slices"

	"github.com/emersion/go-message/charset"

	"cogentcore.org/cogent/mail/xoauth2"
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/core"
	"github.com/cogentcore/go-imap/v2"
	"github.com/cogentcore/go-imap/v2/imapclient"
	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
)

// Server is the configuration of the mail servers of an account.
type Server struct {
	// IMAP is the address of the IMAP server, with the port.
	IMAP string

	// SMTP is the address of the SMTP server, with the port.
	SMTP string

	// Insecure connects without TLS, as for a local test server.
	Insecure bool

	// Password, if set, is used to log in with the email address of the
	// account as the username, instead of OAuth.
	Password string
}

// imapAddress is the address of the Gmail IMAP server.
var imapAddress = "imap.gmail.com:993"

// server returns the server configuration of the given account:
// that in [App.servers] if any, or else Gmail.
func (a *App) server(email string) Server {
	if s, ok := a.servers[email]; ok {
		return s
	}
	return Server{IMAP: imapAddress, SMTP: smtpAddress}
}

// imapOptions are the options of the IMAP clients, which decode the
// headers of the messages (as in their envelopes) in all of the character
// sets that go-message supports, such as windows-1252, as importing its
// charset package also does for the messages themselves.
var imapOptions = &imapclient.Options{
	WordDecoder: &mime.WordDecoder{CharsetReader: charset.Reader},
}

// dialIMAP connects and logs in to the IMAP server of the given account.
func (a *App) dialIMAP(email string) (*imapclient.Client, error) {
	return a.dialIMAPWith(email, imapOptions)
}

// dialIMAPWith connects and logs in to the IMAP server of the given
// account with the given options.
func (a *App) dialIMAPWith(email string, options *imapclient.Options) (*imapclient.Client, error) {
	srv := a.server(email)
	var c *imapclient.Client
	var err error
	if srv.Insecure {
		c, err = imapclient.DialInsecure(srv.IMAP, options)
	} else {
		c, err = imapclient.DialTLS(srv.IMAP, options)
	}
	if err != nil {
		return nil, err
	}
	if srv.Password != "" {
		err = c.Login(email, srv.Password).Wait()
	} else {
		var auth sasl.Client
		auth, err = a.saslClient(email)
		if err == nil {
			err = c.Authenticate(auth)
			slog.Debug("mail: signing in to the IMAP server", "account", email, "err", err)
			if xe := (*xoauth2.Xoauth2Error)(nil); errors.As(err, &xe) {
				err = fmt.Errorf("the server did not accept the sign-in of %s (%w): use Sign in again in the menu of the account, and allow all of the access to Gmail", email, err)
			}
		}
	}
	if err == nil {
		// the capabilities after signing in, which [imapclient.Client.Caps]
		// can return as none if they take a while, as for a busy account
		_, err = c.Capability().Wait()
		if err != nil {
			err = fmt.Errorf("getting the capabilities of the server: %w", err)
		}
	}
	if err != nil {
		c.Close()
		return nil, err
	}
	return c, nil
}

// smtpAuth returns the SASL client for sending messages from the given account.
func (a *App) smtpAuth(email string) (sasl.Client, error) {
	if srv := a.server(email); srv.Password != "" {
		return sasl.NewPlainClient("", email, srv.Password), nil
	}
	return a.saslClient(email)
}

// send sends the given message with the SMTP server of the given account.
func (a *App) send(email string, to []string, r io.Reader) error {
	srv := a.server(email)
	auth, err := a.smtpAuth(email)
	if err != nil {
		return err
	}
	if srv.Insecure {
		return sendMailInsecure(srv.SMTP, auth, email, to, r)
	}
	return sendMail(srv.SMTP, auth, email, to, r)
}

// sendMailInsecure sends a message with SMTP without TLS,
// which [smtp.SendMail] requires.
func sendMailInsecure(addr string, auth sasl.Client, from string, to []string, r io.Reader) error {
	c, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer c.Close()
	if auth != nil {
		if err := c.Auth(auth); err != nil {
			return err
		}
	}
	if err := c.SendMail(from, to, r); err != nil {
		return err
	}
	return c.Quit()
}

// dataDir returns the directory of the data of the app: the messages
// and their cache.
func (a *App) dataDir() string {
	if a.dataDirectory != "" {
		return a.dataDirectory
	}
	return core.TheApp.AppDataDir()
}

// messagesDir returns the directory of the cached messages of the given account.
func (a *App) messagesDir(email string) string {
	return filepath.Join(a.dataDir(), "mail", accountName(email))
}

// trashNames are the names of trash mailboxes of servers without
// the special-use attribute for it.
var trashNames = []string{"[Gmail]/Trash", "Trash", "Deleted Messages", "Deleted Items"}

// findTrash returns the trash mailbox in the given mailboxes: the one with
// the special-use attribute if any, or else one with a standard name.
func findTrash(mailboxes []*imap.ListData) string {
	for _, mb := range mailboxes {
		if slices.Contains(mb.Attrs, imap.MailboxAttrTrash) {
			return mb.Mailbox
		}
	}
	for _, name := range trashNames {
		for _, mb := range mailboxes {
			if mb.Mailbox == name {
				return name
			}
		}
	}
	return ""
}
