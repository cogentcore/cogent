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
	"cogentcore.org/core/keymap"
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

	// cache contains the cached message data, keyed by account and then MessageID.
	cache map[string]map[string]*CacheData

	// listCache is a sorted view of [App.cache] for the current email account
	// and labels, used for displaying a [core.List] of messages. It should not
	// be used for any other purpose.
	listCache []*CacheData

	// readMessage is the current message we are reading
	readMessage *CacheData

	// readMessageReferences is the References header of the current readMessage.
	readMessageReferences []string

	// readMessagePlain is the plain text body of the current readMessage.
	readMessagePlain string

	// currentEmail is the current email account.
	currentEmail string

	// labels are all of the possible labels that messages have.
	labels map[string]bool

	// showLabel is the current label to show messages for.
	showLabel string
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
	a.labels = map[string]bool{}
	a.showLabel = "INBOX"
	a.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
	})

	tree.AddChild(a, func(w *core.Splits) {
		w.SetSplits(0.1, 0.2, 0.7)
		tree.AddChild(w, func(w *core.Tree) {
			w.SetText("Labels")
			w.Maker(func(p *tree.Plan) {
				for _, email := range Settings.Accounts {
					tree.AddAt(p, email, func(w *core.Tree) {
						w.Maker(func(p *tree.Plan) {
							labels := maps.Keys(a.labels)
							slices.Sort(labels)
							for _, label := range labels {
								tree.AddAt(p, label, func(w *core.Tree) {
									w.OnSelect(func(e events.Event) {
										a.showLabel = label
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
			w.SetSlice(&a.listCache)
			w.SetReadOnly(true)
			w.Updater(func() {
				a.listCache = nil
				mp := a.cache[a.currentEmail]
				for _, cd := range mp {
					for _, label := range cd.Labels {
						a.labels[label.Name] = true
						if label.Name == a.showLabel {
							a.listCache = append(a.listCache, cd)
							break
						}
					}
				}
				slices.SortFunc(a.listCache, func(a, b *CacheData) int {
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
		w.SetFunc(a.Compose).SetIcon(icons.Send).SetKey(keymap.New)
	})

	if a.readMessage != nil {
		tree.Add(p, func(w *core.Separator) {})
		tree.Add(p, func(w *core.FuncButton) {
			w.SetFunc(a.Move).SetIcon(icons.DriveFileMove).SetKey(keymap.Save)
		})
		tree.Add(p, func(w *core.FuncButton) {
			w.SetFunc(a.Reply).SetIcon(icons.Reply).SetKey(keymap.Replace)
		})
		tree.Add(p, func(w *core.FuncButton) {
			w.SetFunc(a.ReplyAll).SetIcon(icons.ReplyAll)
		})
		tree.Add(p, func(w *core.FuncButton) {
			w.SetFunc(a.Forward).SetIcon(icons.Forward)
		})
		tree.Add(p, func(w *core.FuncButton) {
			w.SetFunc(a.MarkAsUnread).SetIcon(icons.MarkAsUnread)
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
