// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"cogentcore.org/core/base/auth"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"github.com/cogentcore/go-imap/v2"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"
)

// AccountTypes are the kinds of email accounts, which sign in differently.
type AccountTypes int32 //enums:enum

const (
	// Gmail is a Google account, which is signed in to in the browser.
	Gmail AccountTypes = iota

	// Microsoft is a Microsoft 365 (Exchange Online) account of an
	// organization, or a personal Outlook.com one, which is signed in to
	// in the browser (as with the single sign-on of the organization).
	// Its organization must allow IMAP, and may need to approve the app.
	Microsoft

	// ICloud is an iCloud account (as of an @icloud.com or @me.com
	// address), which signs in with an app-specific password, made at
	// account.apple.com under Sign-In and Security.
	ICloud

	// IMAP is another account, which signs in with a password to its own
	// IMAP and SMTP servers.
	IMAP
)

// accountTypeNames are the names of the [AccountTypes], as shown.
var accountTypeNames = map[AccountTypes]string{Gmail: "Gmail", Microsoft: "Microsoft 365", ICloud: "iCloud", IMAP: "IMAP"}

// Label returns the name of the type as shown, as iCloud.
func (t AccountTypes) Label() string { return accountTypeNames[t] }

// Account is an email account, of the [SettingsData.Accounts]. The
// password of one that signs in with a password is in the keychain of the
// system, not here.
type Account struct { //types:add -setters=false

	// Type is the kind of the account, which determines how it signs in.
	Type AccountTypes

	// Email is the email address of the account.
	Email string `width:"30"`

	// IMAP is the address of the IMAP server, with the port, as
	// imap.example.com:993.
	IMAP string

	// SMTP is the address of the SMTP server, with the port, as
	// smtp.example.com:587.
	SMTP string

	// Username is the username for signing in to the servers, if it is
	// not the email address.
	Username string

	// Insecure connects without TLS, as for a local test server.
	Insecure bool `display:"-"`
}

// ShouldDisplay shows the servers only for an account of type [IMAP].
func (ac *Account) ShouldDisplay(field string) bool {
	switch field {
	case "IMAP", "SMTP", "Username":
		return ac.Type == IMAP
	}
	return true
}

// iCloudServer is the server of iCloud accounts.
var iCloudServer = Account{IMAP: "imap.mail.me.com:993", SMTP: "smtp.mail.me.com:587"}

// microsoftServer is the server of Microsoft accounts, which signs in
// with OAuth.
var microsoftServer = Account{IMAP: "outlook.office365.com:993", SMTP: "smtp.office365.com:587"}

// microsoftScopes are the OAuth scopes of the access to the mail of
// Microsoft accounts, with IMAP and SMTP.
var microsoftScopes = []string{"https://outlook.office.com/IMAP.AccessAsUser.All", "https://outlook.office.com/SMTP.Send"}

// iCloudDomains are the domains of the addresses of iCloud, and
// microsoftDomains those of personal Microsoft accounts.
var (
	iCloudDomains    = []string{"icloud.com", "me.com", "mac.com"}
	microsoftDomains = []string{"outlook.com", "hotmail.com", "live.com", "msn.com"}
)

// typeOf returns the type of account of the given email address: [ICloud]
// or [Microsoft] for those of iCloud or Outlook.com, or else [Gmail]
// (which an organization can have, as Microsoft 365 too).
func typeOf(email string) AccountTypes {
	_, domain, _ := strings.Cut(strings.ToLower(strings.TrimSpace(email)), "@")
	switch {
	case slices.Contains(iCloudDomains, domain):
		return ICloud
	case slices.Contains(microsoftDomains, domain):
		return Microsoft
	}
	return Gmail
}

// accountType returns the type of the given account, of the settings:
// [Gmail] if it is not one of them.
func accountType(email string) AccountTypes {
	if ac := Settings.account(email); ac != nil {
		return ac.Type
	}
	return Gmail
}

// tokenSource signs in to an account of the given type that signs in
// in the browser (Gmail or Microsoft) with the given configuration.
func tokenSource(t AccountTypes, c *auth.AuthConfig) (oauth2.TokenSource, *oidc.UserInfo, error) {
	if t == Microsoft {
		if os.Getenv("MICROSOFT_OAUTH2_CLIENT_ID") == "" && c.ClientID == "" {
			return nil, nil, errors.New("signing in to Microsoft accounts needs the client ID of the app registered with Microsoft, in the environment variable MICROSOFT_OAUTH2_CLIENT_ID")
		}
		return auth.MicrosoftTokenSource(c)
	}
	return auth.GoogleTokenSource(c)
}

// server returns the server configuration of the account, without its
// password: those of iCloud for [ICloud].
func (ac *Account) server() Server {
	s := Server{IMAP: ac.IMAP, SMTP: ac.SMTP, Username: ac.Username, Insecure: ac.Insecure}
	switch ac.Type {
	case ICloud:
		s.IMAP, s.SMTP = iCloudServer.IMAP, iCloudServer.SMTP
	case Microsoft:
		s.IMAP, s.SMTP = microsoftServer.IMAP, microsoftServer.SMTP
	}
	return s
}

// Emails returns the email addresses of the accounts.
func (sd *SettingsData) Emails() []string {
	res := make([]string, 0, len(sd.Accounts))
	for _, ac := range sd.Accounts {
		if e := strings.TrimSpace(ac.Email); e != "" {
			res = append(res, e)
		}
	}
	return res
}

// hasAccount returns whether there is an account with the given email
// address.
func (sd *SettingsData) hasAccount(email string) bool {
	return sd.account(email) != nil
}

// account returns the account with the given email address, or nil.
func (sd *SettingsData) account(email string) *Account {
	for i := range sd.Accounts {
		if strings.TrimSpace(sd.Accounts[i].Email) == email {
			return &sd.Accounts[i]
		}
	}
	return nil
}

// addAccount adds the given account, replacing that of its email address,
// if any.
func (sd *SettingsData) addAccount(ac Account) {
	if old := sd.account(ac.Email); old != nil {
		*old = ac
		return
	}
	sd.Accounts = append(sd.Accounts, ac)
}

// removeAccount removes the account with the given email address.
func (sd *SettingsData) removeAccount(email string) {
	sd.Accounts = slices.DeleteFunc(sd.Accounts, func(ac Account) bool { return strings.TrimSpace(ac.Email) == email })
}

// UnmarshalText makes a Gmail account of the given email address, as the
// accounts of earlier versions are in the settings, all of them Gmail. An
// account saved since is a table, which is read field by field.
func (ac *Account) UnmarshalText(text []byte) error {
	*ac = Account{Type: Gmail, Email: strings.TrimSpace(string(text))}
	return nil
}

// usesPassword returns whether the given account signs in with a password,
// instead of in the browser, as Gmail does.
func usesPassword(email string) bool {
	ac := Settings.account(email)
	return ac != nil && (ac.Type == ICloud || ac.Type == IMAP)
}

// signsInWithBrowser returns whether accounts of the type sign in in the
// browser (with OAuth), not with a password.
func (t AccountTypes) signsInWithBrowser() bool {
	return t == Gmail || t == Microsoft
}

// passwordService is the service of the passwords of the accounts in the
// keychain of the system.
const passwordService = "Cogent Mail"

// password returns the password of the given account that signs in with
// one, from the keychain, cached, or "" if there is none.
func (a *App) password(email string) string {
	a.authMu.Lock()
	defer a.authMu.Unlock()
	if p, ok := a.passwords[email]; ok {
		return p
	}
	p, err := keyring.Get(passwordService, email)
	if err != nil && !errors.Is(err, keyring.ErrNotFound) {
		slog.Error("mail: getting the password of an account from the keychain", "account", email, "err", err)
	}
	a.passwords[email] = p
	return p
}

// setPassword saves the password of the given account in the keychain.
func (a *App) setPassword(email, password string) error {
	if err := keyring.Set(passwordService, email, password); err != nil {
		return fmt.Errorf("saving the password in the keychain: %w", err)
	}
	a.authMu.Lock()
	a.passwords[email] = password
	a.authMu.Unlock()
	return nil
}

// Password is a password, edited in a text field that does not show it.
type Password string

func (p Password) Widget() core.Value {
	return core.NewTextField().SetTypePassword()
}

// newAccount is an account being added, in the dialog of [App.AddAccount].
type newAccount struct {
	Account

	// Password is the password, which is saved in the keychain of the
	// system. For iCloud, it is an app-specific password, made at
	// account.apple.com under Sign-In and Security, not the password of
	// the Apple Account.
	Password Password
}

// ShouldDisplay shows the password only for an account that signs in with
// one, not Gmail.
func (na *newAccount) ShouldDisplay(field string) bool {
	if field == "Password" {
		return !na.Type.signsInWithBrowser()
	}
	return na.Account.ShouldDisplay(field)
}

// checkPassword checks that the given account can sign in to its IMAP
// server with its password, as its email address, or else the name part
// of it, as iCloud can need, setting its Username to the one that works.
func (a *App) checkPassword(na *newAccount) error {
	if na.Password == "" {
		return errors.New("enter the password")
	}
	srv := na.server()
	if srv.IMAP == "" || srv.SMTP == "" {
		return errors.New("enter the IMAP and SMTP servers, as host:port")
	}
	names := []string{na.Email}
	if srv.Username != "" {
		names = []string{srv.Username}
	} else if name, _, _ := strings.Cut(na.Email, "@"); name != "" {
		names = append(names, name)
	}
	var err error
	for _, name := range names {
		srv.Password, srv.Username = string(na.Password), name
		c, derr := a.dialIMAPServer(na.Email, srv)
		slog.Debug("mail: checking the sign-in of an account", "account", na.Email, "username", name, "err", derr)
		if derr == nil {
			c.Close()
			na.Username = ""
			if name != na.Email {
				na.Username = name
			}
			return nil
		}
		if err == nil {
			err = derr
		}
	}
	return fmt.Errorf("signing in to %s: %w", srv.IMAP, err)
}

// addPasswordAccount adds the given account, which signs in with its
// password, which is checked, saving the password in the keychain, and
// syncs it. It must be called with the lock of the window.
func (a *App) addPasswordAccount(na *newAccount) error {
	if err := a.setPassword(na.Email, string(na.Password)); err != nil {
		return err
	}
	Settings.addAccount(na.Account)
	if a.knownAccounts != nil {
		a.knownAccounts[na.Email] = true // not offered to sign in again
	}
	if a.currentEmail == "" {
		a.currentEmail = na.Email
	}
	a.saveSettings()
	a.Update()
	a.SyncNow()
	return nil
}

// AddAccount adds an email account: of Gmail, which is signed in to in the
// browser, or of iCloud or another server, which signs in with a password,
// in a dialog for entering everything first.
func (a *App) AddAccount() { //types:add
	a.accountDialog(&newAccount{Account: Account{Type: Gmail}})
}

// changePassword opens the dialog for entering the password of the given
// account that signs in with one again, as when it changed.
func (a *App) changePassword(email string) {
	ac := Settings.account(email)
	if ac == nil {
		return
	}
	a.accountDialog(&newAccount{Account: *ac})
}

// accountDialog opens the dialog for adding the given account, or for
// entering its password again if it is one of the accounts, after which
// it signs in, in the browser for Gmail.
func (a *App) accountDialog(na *newAccount) {
	again := Settings.hasAccount(na.Email)
	title := "Add account"
	if again {
		title = "Sign in to " + na.Email
		a.signInShown[na.Email] = true
	}
	d := core.NewBody(title)
	ctx, cancel := context.WithCancel(context.Background())
	d.OnClose(func(e events.Event) {
		cancel()
		if again {
			delete(a.signInShown, na.Email)
		}
	})
	email := na.Email
	var note *core.Text
	form := core.NewForm(d).SetStruct(na)
	form.OnChange(func(e events.Event) {
		if again {
			na.Email = email // the account whose password is entered again
		}
		// the type of an address of iCloud, once it is typed
		if na.Type == Gmail && typeOf(na.Email) != Gmail {
			na.Type = typeOf(na.Email)
		}
		form.Update()
		note.Update()
	})
	note = core.NewText(d).SetType(core.TextSupporting)
	note.Updater(func() {
		switch na.Type {
		case Gmail:
			note.SetText(allowAccess)
		case Microsoft:
			note.SetText("Sign in to your Microsoft 365 or Outlook.com account in your browser, as with the single sign-on of your organization, which must allow IMAP, and may need to approve Cogent Mail.")
		case ICloud:
			note.SetText("Use an app-specific password, made at account.apple.com under Sign-In and Security, not the password of your Apple Account.")
		default:
			note.SetText("")
		}
	})
	status := core.NewText(d)
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		ok := d.AddOK(bar).SetText("Sign in").SetIcon(icons.Login)
		ok.OnClick(func(e events.Event) {
			e.SetHandled() // closed once signed in
			na.Email = strings.TrimSpace(na.Email)
			if !na.Type.signsInWithBrowser() && !strings.Contains(na.Email, "@") {
				core.MessageSnackbar(d, "Enter the email address")
				return
			}
			ok.SetEnabled(false).Update()
			status.SetText("Signing in...").Update()
			if na.Type.signsInWithBrowser() {
				status.SetText("Waiting for you to sign in in the browser...").Update()
			}
			go func() {
				err := a.signInNew(ctx, na)
				if errors.Is(err, context.Canceled) {
					return // the dialog was closed
				}
				d.AsyncLock() // (the lock of the window of the app)
				defer d.AsyncUnlock()
				ok.SetEnabled(true).Update()
				status.SetText("").Update()
				if err != nil {
					if _, ok := errors.AsType[*auth.ScopesError](err); ok {
						status.SetText("Not signed in: the access to Gmail was not allowed. Sign in again, and check the box to allow it.").Update()
						return
					}
					core.ErrorDialog(d, err, "Error signing in")
					return
				}
				if !na.Type.signsInWithBrowser() {
					if err := a.addPasswordAccount(na); err != nil {
						core.ErrorDialog(d, err, "Error adding the account")
						return
					}
				}
				d.Close()
			}()
		})
	})
	d.RunDialog(a)
}

// signInNew signs in to the given account being added: in the browser for
// Gmail, adding it once signed in (as the account chosen there), or else
// by checking its password.
func (a *App) signInNew(ctx context.Context, na *newAccount) error {
	if !na.Type.signsInWithBrowser() {
		return a.checkPassword(na)
	}
	slog.Debug("mail: signing in", "account", na.Email, "type", na.Type)
	ts, userInfo, err := tokenSource(na.Type, a.authConfigFor(ctx, na.Email, na.Type))
	if err != nil {
		slog.Debug("mail: signing in failed", "account", na.Email, "err", err)
		return err
	}
	slog.Debug("mail: signed in", "account", na.Email, "as", userInfo.Email)
	a.AsyncLock()
	a.signedInTo(userInfo.Email, na.Type, ts)
	a.AsyncUnlock()
	return nil
}

// saveSent saves a copy of the given message, just sent from the given
// account with its SMTP server, in its Sent mailbox, if it is not Gmail
// (which saves the messages sent with its SMTP server), and the message is
// not there already (as some other servers save them too).
func (a *App) saveSent(email string, msg []byte, messageID string) {
	a.AsyncLock()
	gmail, sent := a.gmail[email], a.specialUse[email][imap.MailboxAttrSent]
	c, mu := a.imapClient[email], a.imapMu[email]
	a.AsyncUnlock()
	if gmail || sent == "" || mu == nil {
		return
	}
	if c == nil { // not connected now: with a connection of its own
		var err error
		c, err = a.dialIMAP(email)
		if err != nil {
			slog.Error("mail: saving a sent message in the Sent mailbox", "account", email, "err", err)
			return
		}
		defer c.Close()
		mu = &sync.Mutex{}
	}
	err := func() error {
		mu.Lock()
		defer mu.Unlock()
		if messageID != "" {
			if c == a.imapClient[email] {
				if err := a.selectMailbox(c, email, sent); err != nil {
					return err
				}
			} else if _, err := c.Select(sent, nil).Wait(); err != nil {
				return err
			}
			id := "<" + strings.Trim(messageID, "<>") + ">"
			sd, err := c.UIDSearch(&imap.SearchCriteria{Header: []imap.SearchCriteriaHeaderField{{Key: "Message-ID", Value: id}}}, nil).Wait()
			if err != nil {
				return err
			}
			if len(sd.AllUIDs()) > 0 {
				slog.Debug("mail: the sent message is in the Sent mailbox already", "account", email, "mailbox", sent)
				return nil
			}
		}
		cmd := c.Append(sent, int64(len(msg)), &imap.AppendOptions{Flags: []imap.Flag{imap.FlagSeen}, Time: time.Now()})
		if _, err := cmd.Write(msg); err != nil {
			cmd.Close()
			return err
		}
		if err := cmd.Close(); err != nil {
			return err
		}
		_, err := cmd.Wait()
		return err
	}()
	if err != nil {
		slog.Error("mail: saving a sent message in the Sent mailbox", "account", email, "mailbox", sent, "err", err)
		return
	}
	a.SyncNow() // to show it in the Sent mailbox
}
