// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"bytes"
	"fmt"
	"log/slog"
	"time"

	"cogentcore.org/core/base/fileinfo"
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

	body      string
	inReplyTo string
}

// Compose opens a dialog to send a new message.
func (a *App) Compose() { //types:add
	a.composeMessage = &SendMessage{}
	a.composeMessage.From = []*mail.Address{{Address: Settings.Accounts[0]}}
	a.composeMessage.To = []*mail.Address{{}}
	a.compose("Compose")
}

// compose is the implementation of the email comoposition dialog,
// which is called by other higher-level functions.
func (a *App) compose(title string) {
	b := core.NewBody(title)
	core.NewForm(b).SetStruct(a.composeMessage)
	ed := texteditor.NewEditor(b)
	ed.Buffer.SetLanguage(fileinfo.Markdown)
	ed.Buffer.Options.LineNumbers = false
	ed.Styler(func(s *styles.Style) {
		s.SetMono(false)
		s.Grow.Set(1, 1)
	})
	b.AddBottomBar(func(bar *core.Frame) {
		b.AddCancel(bar)
		b.AddOK(bar).SetText("Send").OnClick(func(e events.Event) {
			a.composeMessage.body = ed.Buffer.String()
			a.SendMessage()
		})
	})
	b.RunWindowDialog(a)
}

// SendMessage sends the current message
func (a *App) SendMessage() error { //types:add
	if len(a.composeMessage.From) != 1 {
		return fmt.Errorf("expected 1 sender, but got %d", len(a.composeMessage.From))
	}
	email := a.composeMessage.From[0].Address

	var b bytes.Buffer

	var h mail.Header
	h.SetDate(time.Now())
	h.SetAddressList("From", a.composeMessage.From)
	h.SetAddressList("To", a.composeMessage.To)
	h.SetSubject(a.composeMessage.Subject)
	h.SetText("In-Reply-To", "<"+a.composeMessage.inReplyTo+">")

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
	pw.Write([]byte(a.composeMessage.body))
	pw.Close()

	var hh mail.InlineHeader
	hh.Set("Content-Type", "text/html")
	hw, err := tw.CreatePart(hh)
	if err != nil {
		return err
	}
	err = goldmark.Convert([]byte(a.composeMessage.body), hw)
	if err != nil {
		return err
	}
	hw.Close()

	to := make([]string, len(a.composeMessage.To))
	for i, t := range a.composeMessage.To {
		to[i] = t.Address
	}

	err = smtp.SendMail(
		"smtp.gmail.com:587",
		a.authClient[email],
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
