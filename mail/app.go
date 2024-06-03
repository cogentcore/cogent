// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package mail implements a GUI email client.
package mail

//go:generate core generate

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/views"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-sasl"
	"golang.org/x/oauth2"
)

// App is an email client app.
type App struct {
	core.Frame

	// AuthToken contains the [oauth2.Token] for each account.
	AuthToken map[string]*oauth2.Token `set:"-"`

	// AuthClient contains the [sasl.Client] authentication for sending messages for each account.
	AuthClient map[string]sasl.Client `set:"-"`

	// IMAPCLient contains the imap clients for each account.
	IMAPClient map[string]*imapclient.Client `set:"-"`

	// ComposeMessage is the current message we are editing
	ComposeMessage *SendMessage `set:"-"`

	// Cache contains the cache data, keyed by account and then mailbox.
	Cache map[string]map[string][]*CacheData `set:"-"`

	// ReadMessage is the current message we are reading
	ReadMessage *CacheData `set:"-"`

	// The current email account
	CurrentEmail string `set:"-"`

	// The current mailbox
	CurrentMailbox string `set:"-"`
}

// needed for interface import
var _ tree.Node = (*App)(nil)

func (a *App) Init() {
	a.AuthToken = map[string]*oauth2.Token{}
	a.AuthClient = map[string]sasl.Client{}
	a.Style(func(s *styles.Style) {
		s.Grow.Set(1, 1)
	})

	core.AddChild(a, func(w *core.Splits) {
		core.AddChild(w, func(w *views.TreeView) {
			w.SetText("Mailboxes")
		})
	})
	a.Maker(func(p *core.Plan) {
		core.NewFrame(sp, "list").Style(func(s *styles.Style) {
			s.Direction = styles.Column
		})

		ml := core.NewFrame(sp, "mail")
		ml.Style(func(s *styles.Style) {
			s.Direction = styles.Column
		})
		views.NewStructView(ml, "msv").SetReadOnly(true)
		core.NewFrame(ml, "mb").Style(func(s *styles.Style) {
			s.Direction = styles.Column
		})

		// a.UpdateReadMessage(ml, msv, mb)

		sp.SetSplits(0.1, 0.2, 0.7)
	})
}

func (a *App) MakeToolbar(tb *core.Toolbar) { // TODO(config)
	views.NewFuncButton(tb, a.Compose).SetIcon(icons.Send)
}

func (a *App) GetMail() error {
	go func() {
		err := a.Auth()
		if err != nil {
			core.ErrorDialog(a, err, "Error authorizing")
			return
		}
		err = a.CacheMessages()
		if err != nil {
			core.ErrorDialog(a, err, "Error caching messages")
		}
	}()
	return nil
}
