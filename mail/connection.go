// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"time"

	"github.com/cogentcore/go-imap/v2"
	"github.com/cogentcore/go-imap/v2/imapclient"
)

// The times of checking the connections to the servers.
var (
	// connectionTimeout is how long a check of a connection (a NOOP)
	// can take before the connection is taken to be dead, as after the
	// computer sleeps or the network changes.
	connectionTimeout = 30 * time.Second

	// idleCheckInterval is the time between checks of the connection
	// that is idling (see [App.idle]).
	idleCheckInterval = 10 * time.Minute

	// idleRetryMax is the maximum time before connecting again to idle
	// after the connection failed; it starts at idleRetryMin and doubles.
	idleRetryMin, idleRetryMax = 5 * time.Second, 5 * time.Minute
)

// checkConnection checks that the connection to the IMAP server of the
// given account still works, if there is one, closing it if not, so that
// the next sync connects again.
func (a *App) checkConnection(email string) {
	c, mu := a.imapClient[email], a.imapMu[email]
	if c == nil {
		return
	}
	select {
	case <-c.Closed():
		a.dropClient(email, c)
		return
	default:
	}
	done := make(chan error, 1)
	go func() {
		mu.Lock() // which a command that hangs holds, as for a dead connection
		defer mu.Unlock()
		done <- c.Noop().Wait()
	}()
	select {
	case err := <-done:
		if err != nil {
			slog.Info("mail: connection lost", "account", email, "err", err)
			a.dropClient(email, c)
		}
	case <-time.After(connectionTimeout):
		slog.Info("mail: connection not responding", "account", email)
		a.dropClient(email, c)
	}
}

// dropClient closes the given IMAP client of the given account, if it is
// still the current one, and forgets it, so that the next sync connects
// again. Closing it ends the commands waiting on it with errors.
func (a *App) dropClient(email string, c *imapclient.Client) {
	c.Close()
	a.AsyncLock()
	defer a.AsyncUnlock()
	if a.imapClient[email] == c {
		delete(a.imapClient, email)
		a.setSelected(email, "")
	}
}

// startIdle starts a connection that idles for the given account (see
// [App.idle]), if there is not one already.
func (a *App) startIdle(email string) {
	a.AsyncLock()
	defer a.AsyncUnlock()
	if a.idleStop[email] != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.idleStop[email] = cancel
	go a.idleLoop(ctx, email)
}

// stopIdle stops the connection that idles for the given account, if any.
// It must be called with the lock of the window.
func (a *App) stopIdle(email string) {
	if cancel := a.idleStop[email]; cancel != nil {
		cancel()
		delete(a.idleStop, email)
	}
}

// errNoIdle is the error of [App.idle] when the server does not support IDLE.
var errNoIdle = errors.New("the server does not support IDLE")

// idleLoop keeps a connection that idles for the given account (see
// [App.idle]), connecting again when it fails, until the context is done.
func (a *App) idleLoop(ctx context.Context, email string) {
	retry := idleRetryMin
	for ctx.Err() == nil {
		start := time.Now()
		err := a.idle(ctx, email)
		if ctx.Err() != nil {
			return
		}
		if errors.Is(err, errNoIdle) {
			slog.Info("mail: not idling, which the server does not support", "account", email)
			return // the regular syncs still get the new mail
		}
		slog.Info("mail: idling stopped, connecting again", "account", email, "err", err, "in", retry)
		if time.Since(start) > idleCheckInterval {
			retry = idleRetryMin // it worked for a while
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(retry):
		}
		retry = min(2*retry, idleRetryMax)
	}
}

// idle keeps a second connection to the IMAP server of the given account
// idling on its INBOX, syncing the inboxes (see [App.quickSyncLoop]) as
// soon as the server reports a change there, as new mail, instead of
// waiting for the next quick sync. It checks that the connection still
// works every [idleCheckInterval], as after the computer sleeps, returning
// an error if not, and returns nil when the context is done.
func (a *App) idle(ctx context.Context, email string) error {
	opts := *imapOptions
	opts.UnilateralDataHandler = &imapclient.UnilateralDataHandler{
		Mailbox: func(data *imapclient.UnilateralDataMailbox) { a.SyncInboxNow() },
		Expunge: func(seqNum uint32) { a.SyncInboxNow() },
		Fetch: func(msg *imapclient.FetchMessageData) {
			msg.Collect() // the data must be read
			a.SyncInboxNow()
		},
	}
	c, err := a.dialIMAPWith(email, &opts)
	if err != nil {
		return err
	}
	defer c.Close()
	if !c.Caps().Has(imap.CapIdle) {
		return errNoIdle
	}
	if _, err := c.Select("INBOX", &imap.SelectOptions{ReadOnly: true}).Wait(); err != nil {
		return err
	}
	for {
		cmd, err := c.Idle()
		if err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			cmd.Close()
			cmd.Wait()
			return nil
		case <-c.Closed():
			return net.ErrClosed
		case <-time.After(idleCheckInterval):
		}
		if err := cmd.Close(); err != nil {
			return err
		}
		if err := cmd.Wait(); err != nil {
			return err
		}
		done := make(chan error, 1)
		go func() { done <- c.Noop().Wait() }()
		select {
		case err := <-done:
			if err != nil {
				return err
			}
		case <-time.After(connectionTimeout):
			return errNotResponding
		}
	}
}
