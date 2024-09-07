// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package mail implements a GUI email client.
package mail

//go:generate core generate

import (
	"cmp"
	"slices"

	"cogentcore.org/core/core"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
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
	a.Frame.Init()
	a.AuthToken = map[string]*oauth2.Token{}
	a.AuthClient = map[string]sasl.Client{}
	a.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
	})

	tree.AddChildAt(a, "splits", func(w *core.Splits) {
		w.SetSplits(0.1, 0.2, 0.7)
		tree.AddChildAt(w, "mbox", func(w *core.Tree) {
			w.SetText("Mailboxes")
		})
		tree.AddChildAt(w, "list", func(w *core.List) {
			w.SetReadOnly(true)
			w.Updater(func() {
				sl := a.Cache[a.CurrentEmail][a.CurrentMailbox]
				slices.SortFunc(sl, func(a, b *CacheData) int {
					return cmp.Compare(b.Date.UnixNano(), a.Date.UnixNano())
				})
				w.SetSlice(&sl)
			})
		})
		tree.AddChildAt(w, "mail", func(w *core.Frame) {
			w.Styler(func(s *styles.Style) {
				s.Direction = styles.Column
			})
			tree.AddChildAt(w, "msv", func(w *core.Form) {
				w.SetReadOnly(true)
			})
			tree.AddChildAt(w, "mb", func(w *core.Frame) {
				w.Styler(func(s *styles.Style) {
					s.Direction = styles.Column
				})
			})
		})
	})
	a.Updater(func() {
		// a.UpdateReadMessage(ml, msv, mb)
	})
}

func (a *App) MakeToolbar(p *tree.Plan) {
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(a.Compose).SetIcon(icons.Send)
	})
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
