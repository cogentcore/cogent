// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package mail implements a GUI email client.
package mail

//go:generate core generate

import (
	"cmp"
	"slices"
	"sync"

	"golang.org/x/exp/maps"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
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

	// authToken contains the [oauth2.Token] for each account.
	authToken map[string]*oauth2.Token

	// authClient contains the [sasl.Client] authentication for sending messages for each account.
	authClient map[string]sasl.Client

	// imapClient contains the imap clients for each account.
	imapClient map[string]*imapclient.Client

	// imapMu contains the imap client mutexes for each account.
	imapMu map[string]*sync.Mutex

	// composeMessage is the current message we are editing
	composeMessage *SendMessage

	// cache contains the cache data, keyed by account and then mailbox.
	cache map[string]map[string][]*CacheData

	// currentCache is [App.cache] for the current email account and mailbox.
	currentCache []*CacheData

	// readMessage is the current message we are reading
	readMessage *CacheData

	// The current email account
	currentEmail string

	// The current mailbox
	currentMailbox string
}

// needed for interface import
var _ tree.Node = (*App)(nil)

// theApp is the current app instance.
// TODO: ideally we could remove this.
var theApp *App

func (a *App) Init() {
	theApp = a
	a.Frame.Init()
	a.authToken = map[string]*oauth2.Token{}
	a.authClient = map[string]sasl.Client{}
	a.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
	})

	tree.AddChild(a, func(w *core.Splits) {
		w.SetSplits(0.1, 0.2, 0.7)
		tree.AddChild(w, func(w *core.Tree) {
			w.SetText("Mailboxes")
			w.Maker(func(p *tree.Plan) {
				for _, email := range Settings.Accounts {
					tree.AddAt(p, email, func(w *core.Tree) {
						w.Maker(func(p *tree.Plan) {
							mailboxes := maps.Keys(a.cache[email])
							slices.Sort(mailboxes)
							for _, mailbox := range mailboxes {
								tree.AddAt(p, mailbox, func(w *core.Tree) {
									w.OnSelect(func(e events.Event) {
										a.currentMailbox = mailbox
										a.Update()
									})
								})
							}
						})
					})
				}
			})
		})
		tree.AddChild(w, func(w *core.List) {
			w.SetSlice(&a.currentCache)
			w.SetReadOnly(true)
			w.Updater(func() {
				a.currentCache = a.cache[a.currentEmail][a.currentMailbox]
				slices.SortFunc(a.currentCache, func(a, b *CacheData) int {
					return cmp.Compare(b.Date.UnixNano(), a.Date.UnixNano())
				})
			})
		})
		tree.AddChild(w, func(w *core.Frame) {
			w.Styler(func(s *styles.Style) {
				s.Direction = styles.Column
			})
			tree.AddChild(w, func(w *core.Form) {
				w.SetReadOnly(true)
				w.Updater(func() {
					w.SetStruct(a.readMessage.ToMessage())
				})
			})
			tree.AddChild(w, func(w *core.Frame) {
				w.Styler(func(s *styles.Style) {
					s.Direction = styles.Column
					s.Grow.Set(1, 0)
				})
				w.Updater(func() {
					core.ErrorSnackbar(w, a.updateReadMessage(w), "Error reading message")
				})
			})
		})
	})
}

func (a *App) MakeToolbar(p *tree.Plan) {
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(a.Compose).SetIcon(icons.Send)
	})

	if a.readMessage != nil {
		tree.Add(p, func(w *core.Separator) {})
		tree.Add(p, func(w *core.FuncButton) {
			w.SetFunc(a.MoveMessage).SetText("Move").SetIcon(icons.DriveFileMove)
		})
	}
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
