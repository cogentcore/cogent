// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"bytes"
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/textcore"
	"github.com/emersion/go-message/mail"
	"github.com/emersion/go-smtp"
	"github.com/yuin/goldmark"
)

// SendMessage represents the data necessary for the user to send a message.
type SendMessage struct {
	From        []*mail.Address `display:"inline"`
	To          []*mail.Address `display:"inline"`
	Cc          []*mail.Address `display:"inline"`
	Bcc         []*mail.Address `display:"inline"`
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
	from := a.currentEmail
	if from == "" && len(Settings.Accounts) > 0 {
		from = Settings.Accounts[0]
	}
	a.composeMessage.From = []*mail.Address{{Address: from}}
	if sig := signatureText(from); sig != "" {
		// above any quoted message, as usual
		a.composeMessage.body = "\n\n" + sig + a.composeMessage.body
	}
	// All of the compose windows have the same size, which is remembered
	// by the part of the title before the colon.
	b := core.NewBody("Message: " + title)
	b.Styler(func(s *styles.Style) {
		s.Min.Set(units.Ch(80), units.Ch(80))
	})
	core.NewForm(b).SetStruct(a.composeMessage)
	ed := textcore.NewEditor(b)
	core.Bind(&a.composeMessage.body, ed)
	ed.Lines.SetLanguage(fileinfo.Markdown)
	ed.Lines.Settings.LineNumbers = false
	ed.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
	})
	b.AddTopBar(func(bar *core.Frame) {
		tb := core.NewToolbar(bar)
		core.NewButton(tb).SetText("Increase quote").SetIcon(icons.FormatIndentIncrease).
			SetShortcut("Command+'").OnClick(func(e events.Event) {
			ChangeQuoteLevel(ed, 1)
		})
		core.NewButton(tb).SetText("Decrease quote").SetIcon(icons.FormatIndentDecrease).
			SetShortcut("Command+Alt+'").OnClick(func(e events.Event) {
			ChangeQuoteLevel(ed, -1)
		})
	})
	b.AddBottomBar(func(bar *core.Frame) {
		b.AddCancel(bar)
		b.AddOK(bar).SetText("Send").OnClick(func(e events.Event) {
			a.composeMessage.body = ed.Lines.String()
			core.ErrorSnackbar(a, a.Send(), "Error sending message")
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
	h.SetAddressList("To", nonEmpty(a.composeMessage.To))
	if cc := nonEmpty(a.composeMessage.Cc); len(cc) > 0 {
		h.SetAddressList("Cc", cc)
	}
	// the Bcc recipients are not in the header
	h.SetSubject(a.composeMessage.Subject)
	h.SetMessageID(newMessageID(email))
	if a.composeMessage.inReplyTo != "" {
		h.SetMsgIDList("In-Reply-To", []string{a.composeMessage.inReplyTo})
	}
	if len(a.composeMessage.references) > 0 {
		h.SetMsgIDList("References", a.composeMessage.references)
	}

	mw, err := mail.CreateWriter(&b, h)
	if err != nil {
		return err
	}

	iw, err := mw.CreateInline()
	if err != nil {
		return err
	}

	var ph mail.InlineHeader
	ph.SetContentType("text/plain", nil)
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
	hh.SetContentType("text/html", nil)
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
		fname := string(at)
		ah := mail.AttachmentHeader{}
		ah.SetFilename(filepath.Base(fname))
		fi, err := fileinfo.NewFileInfo(fname)
		if err != nil {
			return err
		}
		ah.SetContentType(fi.Mime, nil)
		aw, err := mw.CreateAttachment(ah)
		if err != nil {
			return err
		}
		f, err := os.Open(fname)
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

	to := recipients(email, a.composeMessage)
	if len(to) == 0 {
		return fmt.Errorf("no recipients")
	}
	err = a.send(email, to, &b)
	if err == nil {
		a.SyncNow() // to show it in the sent mailbox
	}
	if se, ok := errors.AsType[*smtp.SMTPError](err); ok {
		slog.Error("error sending message: SMTP error:", "code", se.Code, "enhancedCode", se.EnhancedCode, "message", se.Message)
	}
	return err
}

// nonEmpty returns the given addresses without the empty ones,
// as of new fields that are not filled in.
func nonEmpty(as []*mail.Address) []*mail.Address {
	var res []*mail.Address
	for _, a := range as {
		if a != nil && strings.TrimSpace(a.Address) != "" {
			res = append(res, a)
		}
	}
	return res
}

// recipients returns the addresses of the recipients of the given message
// from the given account, for SMTP: To, Cc, and Bcc, with the sender if
// [SettingsData.AutoBCC], each once.
func recipients(from string, sm *SendMessage) []string {
	var res []string
	add := func(addr string) {
		if !slices.ContainsFunc(res, func(r string) bool { return strings.EqualFold(r, addr) }) {
			res = append(res, addr)
		}
	}
	for _, as := range [][]*mail.Address{sm.To, sm.Cc, sm.Bcc} {
		for _, a := range nonEmpty(as) {
			add(strings.TrimSpace(a.Address))
		}
	}
	if Settings.AutoBCC && len(res) > 0 {
		add(from)
	}
	return res
}

// newMessageID returns a new unique Message-ID (without angle brackets)
// for a message from the given address, with its domain, so that the ID
// does not include the name of this computer.
func newMessageID(from string) string {
	domain := "localhost"
	if _, d, ok := strings.Cut(from, "@"); ok && d != "" {
		domain = d
	}
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%d.%s@%s", time.Now().UnixNano(), base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b), domain)
}

// smtpAddress is the address of the SMTP server for sending messages.
var smtpAddress = "smtp.gmail.com:587"

// sendMail sends a message with SMTP, which tests replace.
var sendMail = smtp.SendMail
