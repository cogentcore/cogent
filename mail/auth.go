// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"encoding/base32"
	"path/filepath"
	"slices"

	"cogentcore.org/cogent/mail/xoauth2"
	"cogentcore.org/core/base/auth"
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/core"
	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// auth authorizes access to the user's mail and sets [App.AuthClient].
// If the user does not already have a saved auth token, it calls [SignIn].
func (a *App) auth() error {
	email, err := a.signIn()
	if err != nil {
		return err
	}

	a.authClient[email] = xoauth2.NewXoauth2Client(email, a.authToken[email].AccessToken)
	return nil
}

// signIn displays a dialog for the user to sign in with the platform of their choice.
// It returns the user's email address.
func (a *App) signIn() (string, error) {
	d := core.NewBody("Sign in")
	email := make(chan string)
	fun := func(token *oauth2.Token, userInfo *oidc.UserInfo) {
		if !slices.Contains(Settings.Accounts, userInfo.Email) {
			Settings.Accounts = append(Settings.Accounts, userInfo.Email)
			errors.Log(core.SaveSettings(Settings))
		}
		a.currentEmail = userInfo.Email
		a.authToken[userInfo.Email] = token
		d.Close()
		email <- userInfo.Email
	}
	auth.Buttons(d, &auth.ButtonsConfig{
		SuccessFunc: fun,
		TokenFile: func(provider, email string) string {
			return filepath.Join(core.TheApp.AppDataDir(), "auth", filenameBase32(email), provider+"-token.json")
		},
		Accounts: Settings.Accounts,
		Scopes: map[string][]string{
			"google": {"https://mail.google.com/"},
		},
	})
	d.RunDialog(a)
	return <-email, nil
}

// filenameBase32 converts the given string to a filename-safe base32 version.
func filenameBase32(s string) string {
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString([]byte(s))
}
