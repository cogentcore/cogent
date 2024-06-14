// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"bytes"
	"fmt"
	"log/slog"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/texteditor"
	"github.com/emersion/go-message/mail"
	"github.com/emersion/go-smtp"
	"github.com/yuin/goldmark"
)

// SendMessage represents the data necessary for the user to send a message.
type SendMessage struct {
	From    []*mail.Address `display:"inline"`
	To      []*mail.Address `display:"inline"`
	Subject string
	Body    string `display:"-"`
}

// Compose pulls up a dialog to send a new message
func (a *App) Compose() { //types:add
	a.ComposeMessage = &SendMessage{}
	a.ComposeMessage.From = []*mail.Address{{Address: Settings.Accounts[0]}}
	a.ComposeMessage.To = []*mail.Address{{}}
	b := core.NewBody().AddTitle("Send message")
	core.NewForm(b).SetStruct(a.ComposeMessage)
	te := texteditor.NewSoloEditor(b)
	te.Buffer.SetLang("md")
	te.Buffer.Options.LineNumbers = false
	te.Styler(func(s *styles.Style) {
		s.SetMono(false)
	})
	b.AddBottomBar(func(pw core.Widget) {
		b.AddCancel(pw)
		b.AddOK(pw).SetText("Send").OnClick(func(e events.Event) {
			a.ComposeMessage.Body = te.Buffer.String()
			a.SendMessage()
		})
	})
	b.RunFullDialog(a)
}

// SendMessage sends the current message
func (a *App) SendMessage() error { //types:add
	if len(a.ComposeMessage.From) != 1 {
		return fmt.Errorf("expected 1 sender, but got %d", len(a.ComposeMessage.From))
	}
	email := a.ComposeMessage.From[0].Address

	var b bytes.Buffer

	var h mail.Header
	h.SetDate(time.Now())
	h.SetAddressList("From", a.ComposeMessage.From)
	h.SetAddressList("To", a.ComposeMessage.To)
	h.SetSubject(a.ComposeMessage.Subject)

	mw, err := mail.CreateWriter(&b, h)
	if err != nil {
		return err
	}

	tw, err := mw.CreateInline()
	if err != nil {
		return err
	}
	defer tw.Close()

	var ph mail.InlineHeader
	ph.Set("Content-Type", "text/plain")
	pw, err := tw.CreatePart(ph)
	if err != nil {
		return err
	}
	pw.Write([]byte(a.ComposeMessage.Body))
	pw.Close()

	var hh mail.InlineHeader
	hh.Set("Content-Type", "text/html")
	hw, err := tw.CreatePart(hh)
	if err != nil {
		return err
	}
	err = goldmark.Convert([]byte(a.ComposeMessage.Body), hw)
	if err != nil {
		return err
	}
	hw.Close()

	to := make([]string, len(a.ComposeMessage.To))
	for i, t := range a.ComposeMessage.To {
		to[i] = t.Address
	}

	err = smtp.SendMail(
		"smtp.gmail.com:587",
		a.AuthClient[email],
		email,
		to,
		&b,
	)
	if err != nil {
		se := err.(*smtp.SMTPError)
		slog.Error("error sending message: SMTP error:", "code", se.Code, "enhancedCode", se.EnhancedCode, "message", se.Message)
	}
	return err
}
