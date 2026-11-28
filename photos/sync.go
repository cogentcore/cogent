// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package photos

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"time"

	"cogentcore.org/cogent/google"
	"cogentcore.org/core/base/auth"
	"cogentcore.org/core/core"
	"golang.org/x/oauth2"
)

// googleTokenFile returns the name of the saved OAuth token of the given
// Google account, in the system keychain (see [auth.SaveToken]).
func googleTokenFile(email string) string {
	if email == "" {
		return ""
	}
	return filepath.Join(core.TheApp.AppDataDir(), "auth", email, "google-token.json")
}

// googleClient signs in to the Google account of the settings, or to one
// chosen in the browser if there is none, returning a client of the
// Picker API of Google Photos.
func googleClient(ctx context.Context) (*pickerClient, error) {
	google.UseBuiltinCredentials()
	acct := Settings.GoogleAccount
	ac := &auth.AuthConfig{Ctx: ctx, TokenFile: googleTokenFile, Scopes: []string{pickerScope}}
	if acct == "" {
		ac.SelectAccount = true
	} else {
		ac.Accounts = []string{acct}
		ac.LoginHint = acct
	}
	ts, user, err := auth.GoogleTokenSource(ac)
	if err != nil {
		return nil, err
	}
	if user != nil && user.Email != "" && user.Email != acct {
		Settings.GoogleAccount = user.Email
	}
	return &pickerClient{http: oauth2.NewClient(ctx, ts)}, nil
}

// SyncGoogle adds photos from Google Photos, as those from a phone
// that syncs with it: they are picked in Google Photos in the browser,
// newest first, and then downloaded, except for those already here.
// Google does not let apps read all of the photos, nor give them the
// locations of photos (since March 2025).
func (a *App) SyncGoogle() { //types:add
	if a.lib == nil || a.syncCancel != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.syncCancel = cancel
	a.updateToolbar()
	go func() {
		a.busy.Lock()
		defer a.busy.Unlock()
		res, err := a.syncGoogle(ctx)
		cancel()
		a.AsyncLock()
		defer a.AsyncUnlock()
		a.syncCancel = nil
		a.statusText = ""
		if errors.Is(err, context.Canceled) {
			err = nil
		}
		core.ErrorSnackbar(a, err, "Error syncing with Google Photos")
		if res != nil {
			core.MessageSnackbar(a, res.String())
			if len(res.Added) > 0 && a.shown != allKey {
				a.shown = allKey // where they are, at the top
				a.main.SetPhotos(nil)
			}
		}
		saveSettings(a)
		a.refresh()
	}()
}

// syncGoogle does [App.SyncGoogle], in its own goroutine.
func (a *App) syncGoogle(ctx context.Context) (*GoogleResult, error) {
	a.setStatus("Signing in to Google Photos…")
	pc, err := googleClient(ctx)
	if err != nil {
		return nil, err
	}
	ps, err := pc.createSession(ctx)
	if err != nil {
		return nil, err
	}
	defer pc.deleteSession(context.Background(), ps.ID)
	pick := "Pick the photos in Google Photos in your browser, then press Done."
	if last := Settings.GoogleSynced; !last.IsZero() {
		pick = "Pick the photos since the last sync, " + last.Format("Jan 2, 2006") + ", in Google Photos in your browser, then press Done."
	}
	a.setStatus(pick)
	u := ps.PickerURI
	if !strings.HasSuffix(u, "/autoclose") {
		u = strings.TrimSuffix(u, "/") + "/autoclose" // closes the tab when done
	}
	core.TheApp.OpenURL(u)
	picked, err := pc.waitPicked(ctx, ps)
	if err != nil || !picked {
		return nil, err
	}
	a.setStatus("Getting the photos picked…")
	items, err := pc.listItems(ctx, ps.ID)
	if err != nil {
		return nil, err
	}
	res, err := a.lib.addGoogle(ctx, pc, items, a.progress("Downloading"))
	if err == nil || len(res.Added) > 0 {
		Settings.GoogleSynced = time.Now()
	}
	return res, err
}

// CancelSync stops syncing with Google Photos.
func (a *App) CancelSync() { //types:add
	if a.syncCancel != nil {
		a.syncCancel()
	}
}
