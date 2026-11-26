// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"context"
	"encoding/base32"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"

	"cogentcore.org/cogent/mail/xoauth2"
	"cogentcore.org/core/base/auth"
	"cogentcore.org/core/base/auth/cicons"
	cerrors "cogentcore.org/core/base/errors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"github.com/emersion/go-sasl"
	"golang.org/x/oauth2"
)

// gmailScopes are the OAuth scopes of the access to Gmail: all of the
// mail, with IMAP and SMTP.
var gmailScopes = []string{"https://mail.google.com/"}

// tokenFile returns the file of the saved OAuth token of the given account.
func (a *App) tokenFile(email string) string {
	if email == "" {
		return ""
	}
	return filepath.Join(a.dataDir(), "auth", accountName(email), "google-token.json")
}

// authConfig returns the configuration of signing in to the given
// account, or to a new one for "".
func (a *App) authConfig(ctx context.Context, email string) *auth.AuthConfig {
	ac := &auth.AuthConfig{
		Ctx:       ctx,
		TokenFile: a.tokenFile,
		Scopes:    gmailScopes,
	}
	if email == "" {
		ac.SelectAccount = true
	} else {
		ac.Accounts = []string{email}
		ac.LoginHint = email
	}
	return ac
}

// usesOAuth returns whether the given account signs in with OAuth,
// as Gmail does, instead of a password.
func (a *App) usesOAuth(email string) bool {
	return a.server(email).Password == ""
}

// signedIn returns whether the given account can connect to its server:
// it is connected, or it has a password, or a token from signing in.
func (a *App) signedIn(email string) bool {
	if !a.usesOAuth(email) {
		return true
	}
	if a.imapClient[email] != nil {
		return true
	}
	a.authMu.Lock()
	defer a.authMu.Unlock()
	return a.tokens[email] != nil
}

// setTokens sets the source of the OAuth tokens of the given account.
func (a *App) setTokens(email string, ts oauth2.TokenSource) {
	a.authMu.Lock()
	defer a.authMu.Unlock()
	a.tokens[email] = ts
}

// saslClient returns the SASL client for signing in to the servers of the
// given account with OAuth, with an access token that is valid now, as
// they expire after an hour.
func (a *App) saslClient(email string) (sasl.Client, error) {
	a.authMu.Lock()
	ts := a.tokens[email]
	a.authMu.Unlock()
	if ts == nil {
		return nil, fmt.Errorf("not signed in to %s", email)
	}
	token, err := ts.Token()
	if err != nil {
		slog.Debug("mail: getting the access token failed", "account", email, "err", err)
		return nil, fmt.Errorf("signing in to %s: %w", email, err)
	}
	slog.Debug("mail: access token for signing in to the server", "account", email,
		"expiry", token.Expiry, "grantedScopes", token.Extra("scope"))
	return xoauth2.NewXoauth2Client(email, token.AccessToken), nil
}

// signInSaved signs in to each of the accounts that use OAuth with its
// saved token, returning those that need the user to sign in again,
// without a saved token that works. Those whose saved token could not be
// checked, as when offline, are not among them: they are tried again in
// each sync (see [App.authErrs]).
func (a *App) signInSaved() []string {
	var need []string
	for _, email := range slices.Clone(Settings.Accounts) {
		if !a.usesOAuth(email) || a.signedIn(email) {
			continue
		}
		if n, _ := a.signInSavedTo(email); n {
			need = append(need, email)
		}
	}
	return need
}

// signInSavedTo signs in to the given account with its saved token,
// returning whether the user needs to sign in again, without a saved
// token that works, and any error, which is in [App.authErrs] if it is
// of not being able to check the token, as when offline.
func (a *App) signInSavedTo(email string) (need bool, err error) {
	ac := a.authConfig(context.Background(), email)
	ac.NoPrompt = true
	ts, userInfo, err := auth.GoogleTokenSource(ac)
	if err == nil && userInfo.Email != email {
		err = fmt.Errorf("the saved token of %s is for %s", email, userInfo.Email)
	}
	offline := isOffline(err)
	slog.Debug("mail: signing in with the saved token", "account", email, "err", err, "offline", offline)
	a.authMu.Lock()
	if offline {
		a.authErrs[email] = err
	} else {
		delete(a.authErrs, email)
	}
	a.authMu.Unlock()
	switch {
	case offline:
		return false, err
	case errors.Is(err, auth.ErrNoToken):
		return true, err
	case err != nil:
		cerrors.Log(err)
		return true, err
	}
	a.setTokens(email, ts)
	return false, nil
}

// authErr returns the error of not being able to sign in to the given
// account with its saved token, as when offline, if any.
func (a *App) authErr(email string) error {
	a.authMu.Lock()
	defer a.authMu.Unlock()
	return a.authErrs[email]
}

// signedInTo records that the user signed in to the given account with
// the given tokens, adding it to the accounts if it is new, and syncs it.
// It must be called with the lock of the window.
func (a *App) signedInTo(email string, ts oauth2.TokenSource) {
	slog.Debug("mail: signed in to the account", "account", email, "new", !slices.Contains(Settings.Accounts, email))
	a.setTokens(email, ts)
	if !slices.Contains(Settings.Accounts, email) {
		Settings.Accounts = append(Settings.Accounts, email)
		a.saveSettings()
	}
	if a.currentEmail == "" {
		a.currentEmail = email
	}
	a.Update()
	a.SyncNow()
}

// AddAccount adds another Gmail account, which the user signs in to in
// the browser, choosing the account there.
func (a *App) AddAccount() { //types:add
	a.signInDialog(nil)
}

// SignInAgain signs in to the given account again in the browser, as
// when its saved sign-in no longer works.
func (a *App) SignInAgain(email string) {
	a.signInDialog([]string{email})
}

// allowAccess tells the user to allow the access to Gmail when signing in,
// as Google's page asks for it with a box that is not checked.
const allowAccess = "On Google's page, check the box that allows Cogent Mail to read, compose, send, and delete your email, which it needs to show and send your mail."

// signInDialog opens a dialog for signing in in the browser to each of
// the given accounts, or to a new one, chosen there, for none.
func (a *App) signInDialog(accounts []string) {
	title := "Add account"
	if len(accounts) > 0 {
		title = "Sign in"
	}
	d := core.NewBody(title)
	ctx, cancel := context.WithCancel(context.Background())
	d.OnClose(func(e events.Event) { cancel() })
	if len(accounts) == 0 {
		core.NewText(d).SetText("Sign in to the Gmail account to add in your browser.")
		accounts = []string{""}
	} else {
		core.NewText(d).SetText("Sign in again in your browser, as the saved sign-in no longer works.")
	}
	core.NewText(d).SetType(core.TextSupporting).SetText(allowAccess)
	status := core.NewText(d)
	left := len(accounts)
	for _, email := range accounts {
		bt := core.NewButton(d).SetType(core.ButtonOutlined).SetIcon(cicons.SignInWithGoogle)
		text := "Sign in with Google"
		if email != "" {
			text = "Sign in to " + email
		}
		bt.SetText(text)
		bt.OnClick(func(e events.Event) {
			slog.Debug("mail: signing in", "account", email)
			bt.SetEnabled(false).Update()
			status.SetText("Waiting for you to sign in in the browser...").Update()
			go func() {
				ts, userInfo, err := auth.GoogleTokenSource(a.authConfig(ctx, email))
				if err != nil {
					slog.Debug("mail: signing in failed", "account", email, "err", err)
				} else {
					slog.Debug("mail: signed in", "account", email, "as", userInfo.Email)
				}
				if errors.Is(err, context.Canceled) {
					return // the dialog was closed
				}
				d.AsyncLock()
				defer d.AsyncUnlock()
				if err != nil {
					bt.SetEnabled(true).Update()
					if _, ok := errors.AsType[*auth.ScopesError](err); ok {
						status.SetText("Not signed in: the access to Gmail was not allowed. Sign in again, and check the box to allow it.").Update()
						return
					}
					status.SetText("").Update()
					core.ErrorDialog(d, err, "Error signing in")
					return
				}
				a.signedInTo(userInfo.Email, ts)
				left--
				if left == 0 {
					d.Close()
					return
				}
				status.SetText("Signed in to " + userInfo.Email).Update()
				bt.SetText("Signed in").SetIcon(icons.Check).Update()
			}()
		})
	}
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
	})
	d.RunDialog(a)
}

// confirmRemoveAccount asks whether to remove the given account,
// and removes it if so.
func (a *App) confirmRemoveAccount(email string) {
	d := core.NewBody("Remove account")
	core.NewText(d).SetText("Remove " + email + " from Cogent Mail? Its mail stays on the server; the copy here is deleted.")
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).SetText("Remove").OnClick(func(e events.Event) {
			a.RemoveAccount(email)
		})
	})
	d.RunDialog(a)
}

// RemoveAccount removes the given account from the app: its sign-in, and
// its local copy of the mail, which stays on the server.
func (a *App) RemoveAccount(email string) {
	Settings.Accounts = slices.DeleteFunc(Settings.Accounts, func(s string) bool { return s == email })
	if Settings.FilingAccount == AccountName(email) {
		Settings.FilingAccount = ""
	}
	a.saveSettings()
	if a.currentEmail == email {
		a.currentEmail = ""
		if len(Settings.Accounts) > 0 {
			a.currentEmail = Settings.Accounts[0]
		}
		a.showLabel = "INBOX"
		a.readMessage = nil
	}
	c, mu := a.imapClient[email], a.imapMu[email]
	st, si := a.stores[email], a.searches[email]
	delete(a.imapClient, email)
	delete(a.imapMu, email)
	delete(a.stores, email)
	delete(a.searches, email)
	delete(a.cache, email)
	delete(a.totalMessages, email)
	a.stopIdle(email)
	a.authMu.Lock()
	delete(a.tokens, email)
	delete(a.authErrs, email)
	a.authMu.Unlock()
	a.draftsMu.Lock()
	for id, d := range a.drafts {
		if d.Account == email {
			delete(a.drafts, id)
		}
	}
	a.draftsMu.Unlock()
	a.Update()
	go func() { // after any sync of it
		if mu != nil {
			mu.Lock()
			defer mu.Unlock()
		}
		if c != nil {
			c.Logout().Wait()
			c.Close()
		}
		if st != nil {
			st.Close()
		}
		if si != nil {
			si.Close()
		}
		cerrors.Log(auth.DeleteToken(a.tokenFile(email)))
		for _, f := range []string{a.messagesDir(email), a.storeFile(email), a.searchDir(email),
			filepath.Dir(a.tokenFile(email))} {
			cerrors.Log(os.RemoveAll(f))
		}
	}()
}

// filenameBase32 converts the given string to a filename-safe base32 version,
// as the files of the accounts were named before (see [App.migrateFiles]).
func filenameBase32(s string) string {
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString([]byte(s))
}
