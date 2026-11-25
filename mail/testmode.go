// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"cogentcore.org/cogent/mail/testserver"
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/core"
)

// TestingOptions are the options for [App.StartTesting].
type TestingOptions struct {
	// Keep keeps the mail of the test server between runs, in the testing
	// directory of the data of the app, instead of starting with new
	// sample mail each time.
	Keep bool

	// Plain uses a plain IMAP server instead of one like Gmail;
	// see [testserver.Options.Plain].
	Plain bool

	// Seed is the random seed of the sample mail.
	Seed uint64
}

// StartTesting starts a local simulated mail server ([testserver]) with
// sample mail for several accounts, and sets up the app to use it instead
// of real accounts, with its own settings and data in a separate directory,
// so that nothing of the real accounts is used or changed. It must be called
// after the settings are loaded (by [core.NewBody]) and before [App.GetMail].
// The server is closed when the app quits.
func (a *App) StartTesting(opts TestingOptions) error {
	srv, err := testserver.New(testserver.Options{Plain: opts.Plain})
	if err != nil {
		return err
	}
	var dir string
	if opts.Keep {
		dir = filepath.Join(core.TheApp.AppDataDir(), "testing")
	} else {
		dir, err = os.MkdirTemp("", "cogent-mail-testing-")
		if err != nil {
			srv.Close()
			return err
		}
	}
	serverDir := filepath.Join(dir, "server")
	loaded, err := srv.Load(serverDir)
	if err != nil {
		srv.Close()
		return err
	}
	if !loaded {
		if err := srv.Seed(opts.Seed); err != nil {
			srv.Close()
			return err
		}
	}

	// The cached data of the app always starts fresh, since the UIDs
	// of a reloaded server are new.
	a.dataDirectory = filepath.Join(dir, "app")
	if err := os.RemoveAll(a.dataDirectory); err != nil {
		srv.Close()
		return err
	}
	Settings.File = filepath.Join(dir, "settings.toml")
	Settings.Accounts = srv.Accounts
	for _, account := range srv.Accounts {
		a.servers[account] = Server{IMAP: srv.IMAPAddr, SMTP: srv.SMTPAddr, Insecure: true, Password: testserver.Password}
	}
	a.currentEmail = srv.Accounts[0]
	a.syncInterval = 15 * time.Second
	a.testServer = srv
	srv.OnDeliver = func(accounts []string) { a.SyncNow() }

	core.TheApp.AddQuitCleanFunc(func() {
		if opts.Keep {
			errors.Log(srv.Save(serverDir))
		}
		srv.Close()
		if !opts.Keep {
			errors.Log(os.RemoveAll(dir))
		}
	})
	slog.Info("mail: testing", "dir", dir, "imap", srv.IMAPAddr, "smtp", srv.SMTPAddr, "accounts", srv.Accounts, "loaded", loaded)
	return nil
}

// ReceiveTestMessage makes the test server deliver a new random message to
// the INBOX of the current account, as if it were just received.
// It is only available in testing mode.
func (a *App) ReceiveTestMessage() error { //types:add
	if a.testServer == nil {
		return fmt.Errorf("not in testing mode")
	}
	return a.testServer.RandomMessage(a.currentEmail)
}
