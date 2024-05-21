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

func (a *App) OnInit() {
	a.AuthToken = map[string]*oauth2.Token{}
	a.AuthClient = map[string]sasl.Client{}
	a.SetStyles()
}

func (a *App) SetStyles() {
	a.Style(func(s *styles.Style) {
		s.Grow.Set(1, 1)
	})
}

func (a *App) ConfigToolbar(tb *core.Toolbar) {
	views.NewFuncButton(tb, a.Compose).SetIcon(icons.Send)
}

func (a *App) Config(c *core.Plan) {
	if a.HasChildren() {
		return
	}

	sp := core.NewSplits(a, "splits")

	views.NewTreeView(sp, "mbox").SetText("Mailboxes")

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
}

// // UpdateReadMessage updates the view of the message currently being read.
// func (a *App) UpdateReadMessage(ml *core.Frame, msv *views.StructView, mb *core.Frame) {
// 	if a.ReadMessage == nil {
// 		return
// 	}

// 	msv.SetStruct(a.ReadMessage)

// 	update := mb.UpdateStart()
// 	if mb.HasChildren() {
// 		mb.DeleteChildren(true)
// 	}

// 	mr := grr.Log(mail.CreateReader(a.ReadMessage.BodyReader))
// 	for {
// 		p, err := mr.NextPart()
// 		if err == io.EOF {
// 			break
// 		} else if err != nil {
// 			grr.Log0(err)
// 		}

// 		switch h := p.Header.(type) {
// 		case *mail.InlineHeader:
// 			ct, _ := grr.Log2(h.ContentType())
// 			switch ct {
// 			case "text/plain":
// 				grr.Log0(gidom.ReadMD(gidom.BaseContext(), mb, grr.Log(io.ReadAll(p.Body))))
// 			case "text/html":
// 				grr.Log0(gidom.ReadHTML(gidom.BaseContext(), mb, p.Body))
// 			}
// 		}
// 	}
// 	mb.Update()
// 	mb.UpdateEndLayout(update)
// }

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
