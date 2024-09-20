// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"os"
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
	From        []*mail.Address `display:"inline"`
	To          []*mail.Address `display:"inline"`
	Subject     string
	Attachments []core.Filename `display:"inline"`

	body       string
	inReplyTo  string
	references []string
}

// Compose opens a dialog to send a new message.
func (a *App) Compose() { //types:add
	a.composeMessage = &SendMessage{}
	a.composeMessage.To = []*mail.Address{{}}
	a.compose("Compose")
}

// compose is the implementation of the email comoposition dialog,
// which is called by other higher-level functions.
func (a *App) compose(title string) {
	a.composeMessage.From = []*mail.Address{{Address: Settings.Accounts[0]}}
	b := core.NewBody(title)
	core.NewForm(b).SetStruct(a.composeMessage)
	ed := texteditor.NewEditor(b)
	core.Bind(&a.composeMessage.body, ed)
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
			a.Send()
		})
	})
	b.RunWindowDialog(a)
}

// Send sends the current message
func (a *App) Send() error { //types:add
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
	h.SetMsgIDList("In-Reply-To", []string{a.composeMessage.inReplyTo})
	h.SetMsgIDList("References", a.composeMessage.references)

	mw, err := mail.CreateWriter(&b, h)
	if err != nil {
		return err
	}

	iw, err := mw.CreateInline()
	if err != nil {
		return err
	}

	var ph mail.InlineHeader
	ph.Set("Content-Type", "text/plain")
	pw, err := iw.CreatePart(ph)
	if err != nil {
		return err
	}
	pw.Write([]byte(a.composeMessage.body))
	err = pw.Close()
	if err != nil {
		return err
	}

	var hh mail.InlineHeader
	hh.Set("Content-Type", "text/html")
	hw, err := iw.CreatePart(hh)
	if err != nil {
		return err
	}
	err = goldmark.Convert([]byte(a.composeMessage.body), hw)
	if err != nil {
		return err
	}
	err = hw.Close()
	if err != nil {
		return err
	}
	err = iw.Close()
	if err != nil {
		return err
	}

	for _, at := range a.composeMessage.Attachments {
		ah := mail.AttachmentHeader{}
		ah.SetFilename(string(at))
		aw, err := mw.CreateAttachment(ah)
		if err != nil {
			return err
		}
		f, err := os.Open(string(at))
		if err != nil {
			return err
		}
		_, err = io.Copy(aw, f)
		if err != nil {
			return err
		}
		err = aw.Close()
		if err != nil {
			return err
		}
	}
	err = mw.Close()
	if err != nil {
		return err
	}

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
