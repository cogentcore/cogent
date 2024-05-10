// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"bytes"
	"cmp"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"time"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/htmlview"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/texteditor"
	"cogentcore.org/core/views"
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-message/mail"
	"github.com/emersion/go-smtp"
	"github.com/yuin/goldmark"
)

// Message contains the relevant information for an email message.
type Message struct {
	From       []*mail.Address `view:"inline"`
	To         []*mail.Address `view:"inline"`
	Subject    string
	Body       string             `view:"-"` // only for sending
	BodyReader imap.LiteralReader `view:"-"` // only for receiving
}

// Compose pulls up a dialog to send a new message
func (a *App) Compose() { //types:add
	a.ComposeMessage = &Message{}
	a.ComposeMessage.From = []*mail.Address{{Address: Settings.Accounts[0]}}
	a.ComposeMessage.To = []*mail.Address{{}}
	b := core.NewBody().AddTitle("Send message")
	views.NewStructView(b).SetStruct(a.ComposeMessage)
	te := texteditor.NewSoloEditor(b)
	te.Buffer.SetLang("md")
	te.Buffer.Options.LineNumbers = false
	te.Style(func(s *styles.Style) {
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

// UpdateMessageList updates the message list from [App.Cache].
func (a *App) UpdateMessageList() {
	cached := a.Cache[a.CurrentEmail][a.CurrentMailbox]

	a.AsyncLock()
	defer a.AsyncUnlock()

	list := a.FindPath("splits/list").(*core.Frame)

	if list.NumChildren() > 100 {
		return
	}

	list.DeleteChildren()

	slices.SortFunc(cached, func(a, b *CacheData) int {
		return cmp.Compare(b.Date.UnixNano(), a.Date.UnixNano())
	})

	for i, cd := range cached {
		cd := cd

		if i > 100 {
			break
		}

		fr := core.NewFrame(list).Style(func(s *styles.Style) {
			s.Direction = styles.Column
		})

		fr.Style(func(s *styles.Style) {
			s.SetAbilities(true, abilities.Activatable, abilities.Hoverable)
			s.Cursor = cursors.Pointer
		})
		fr.OnClick(func(e events.Event) {
			a.ReadMessage = cd
			errors.Log(a.UpdateReadMessage())
		})
		fr.AddContextMenu(func(m *core.Scene) {
			a.ReadMessage = cd
			views.NewFuncButton(m, a.MoveMessage).SetIcon(icons.Move).SetText("Move")
		})

		ftxt := ""
		for _, f := range cd.From {
			ftxt += f.Name + " "
		}

		core.NewText(fr, "from").SetType(core.TextTitleMedium).SetText(ftxt).
			Style(func(s *styles.Style) {
				s.SetNonSelectable()
				s.FillMargin = false
			})
		core.NewText(fr, "subject").SetType(core.TextBodyMedium).SetText(cd.Subject).
			Style(func(s *styles.Style) {
				s.SetNonSelectable()
				s.FillMargin = false
			})
	}

	list.Update()
}

// UpdateReadMessage updates the view of the message currently being read.
func (a *App) UpdateReadMessage() error {
	msv := a.FindPath("splits/mail/msv").(*views.StructView)
	msv.SetStruct(a.ReadMessage.ToMessage())

	mb := a.FindPath("splits/mail/mb").(*core.Frame)
	mb.DeleteChildren()

	bemail := FilenameBase32(a.CurrentEmail)
	bmbox := FilenameBase32(a.CurrentMailbox)
	// there can be flags at the end of the filename, so we have to glob it
	glob := filepath.Join(core.TheApp.AppDataDir(), "mail", bemail, bmbox, "cur", a.ReadMessage.Filename+"*")
	matches, err := filepath.Glob(glob)
	if err != nil {
		return err
	}
	if len(matches) != 1 {
		return fmt.Errorf("expected 1 match for filepath glob but got %d: %s", len(matches), glob)
	}

	f, err := os.Open(matches[0])
	if err != nil {
		return err
	}
	defer f.Close()

	mr, err := mail.CreateReader(f)
	if err != nil {
		return err
	}

	var plain *mail.Part
	var gotHTML bool

	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		switch h := p.Header.(type) {
		case *mail.InlineHeader:
			ct, _, err := h.ContentType()
			if err != nil {
				return err
			}

			switch ct {
			case "text/plain":
				plain = p
			case "text/html":
				err := htmlview.ReadHTML(htmlview.NewContext(), mb, p.Body)
				if err != nil {
					return err
				}
				gotHTML = true
			}
		}
	}

	// we only handle the plain version if there is no HTML version
	if !gotHTML && plain != nil {
		err := htmlview.ReadMD(htmlview.NewContext(), mb, errors.Log1(io.ReadAll(plain.Body)))
		if err != nil {
			return err
		}
	}

	mb.Update()
	return nil
}

// MoveMessage moves the current message to the given mailbox.
func (a *App) MoveMessage(mailbox string) error { //types:add
	c := a.IMAPClient[a.CurrentEmail]
	uidset := imap.UIDSet{}
	uidset.AddNum(a.ReadMessage.UID)
	fmt.Println(uidset)
	mc := c.Move(uidset, mailbox)
	fmt.Println("mc", mc)
	md, err := mc.Wait()
	fmt.Println("md", md, err)
	return err
}

/*
// GetMessages fetches the messages from the server
func (a *App) GetMessages() error { //types:add
	c, err := imapclient.DialTLS("imap.gmail.com:993", nil)
	if err != nil {
		return err
	}
	defer c.Logout()

	err = c.Authenticate(a.AuthClient)
	if err != nil {
		return err
	}

	ibox, err := c.Select("INBOX", false)
	if err != nil {
		return err
	}

	// Get the last 40 messages
	from := uint32(1)
	to := ibox.Messages
	if ibox.Messages > 39 {
		// We're using unsigned integers here, only subtract if the result is > 0
		from = ibox.Messages - 39
	}
	seqset := new(imap.SeqSet)
	seqset.AddRange(from, to)

	var sect imap.BodySectionName

	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)
	go func() {
		done <- c.Fetch(seqset, []imap.FetchItem{imap.FetchEnvelope, sect.FetchItem()}, messages)
	}()

	a.Messages = make([]*Message, 0)
	for msg := range messages {

		from := make([]*mail.Address, len(msg.Envelope.From))
		for i, fr := range msg.Envelope.From {
			from[i] = &mail.Address{Name: fr.PersonalName, Address: fr.Address()}
		}
		to := make([]*mail.Address, len(msg.Envelope.To))
		for i, fr := range msg.Envelope.To {
			to[i] = &mail.Address{Name: fr.PersonalName, Address: fr.Address()}
		}

		m := &Message{
			From:       from,
			To:         to,
			Subject:    msg.Envelope.Subject,
			BodyReader: msg.GetBody(&sect),
		}
		a.Messages = append(a.Messages, m)
	}
	slices.Reverse(a.Messages)

	if err := <-done; err != nil {
		return err
	}
	return nil
}
*/
