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
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/textcore"
	"github.com/emersion/go-message/mail"
	"github.com/emersion/go-smtp"
	"github.com/yuin/goldmark"
)

// SendMessage represents the data necessary for the user to send a message.
// Its fields are shown in its compose window in this order, with the
// sender last, as it is usually the default one; Tab goes from To to the
// Subject and then the message, skipping the others, which are clicked.
type SendMessage struct {
	To          []*mail.Address `display:"inline"`
	Cc          []*mail.Address `display:"inline" tab:"-"`
	Bcc         []*mail.Address `display:"inline" tab:"-"`
	Subject     string
	Attachments []core.Filename `display:"inline" tab:"-"`
	Sender      MessageSender   `display:"inline" tab:"-"`

	body       string
	inReplyTo  string
	references []string

	// draft is its draft, once it is saved, as it is written.
	draft *draft

	// replyAll, if set, are the recipients to add for a reply to all,
	// with the Reply all button of its compose window, as for a reply
	// to the address of a mailto: link (see [App.replyMailto]).
	replyAll *replyAll

	// orig is the message that this is a copy of, with its contents at
	// one time, for saving it (see [SendMessage.snapshot]), whose draft
	// is the draft of this one.
	orig *SendMessage

	// seq is the number of the copy, and savedSeq is that of the latest
	// copy saved, under the lock of the app.
	seq, savedSeq int
}

// holder returns the message that holds the draft of this one:
// its original if it is a copy.
func (sm *SendMessage) holder() *SendMessage {
	if sm.orig != nil {
		return sm.orig
	}
	return sm
}

// snapshot returns a copy of the message with its contents now, with the
// given number, for saving it with the lock of the app, while it can
// change in its compose window, which has its own lock.
func (sm *SendMessage) snapshot(seq int) *SendMessage {
	cp := func(as []*mail.Address) []*mail.Address {
		res := make([]*mail.Address, 0, len(as))
		for _, ad := range as {
			if ad != nil {
				c := *ad
				res = append(res, &c)
			}
		}
		return res
	}
	return &SendMessage{Sender: sm.Sender, To: cp(sm.To), Cc: cp(sm.Cc), Bcc: cp(sm.Bcc),
		Subject: sm.Subject, Attachments: slices.Clone(sm.Attachments), body: sm.body,
		inReplyTo: sm.inReplyTo, references: slices.Clone(sm.references), orig: sm, seq: seq}
}

// MessageSender is the sender of a message being written: the account that
// it is from, and its signature, shown side by side.
type MessageSender struct {

	// From is the account that the message is sent from.
	From SenderAccount

	// Signature is the signature of the message, which is that of the
	// account by default, and which is changed in the message when it is
	// changed (see [switchSignature]).
	Signature SignatureName
}

// SenderAccount is the account that a message is sent from, chosen from
// the accounts.
type SenderAccount string

func (sa SenderAccount) Widget() core.Value {
	ch := core.NewChooser()
	for _, account := range Settings.Accounts {
		text := account
		if theApp != nil { // with its name, as Name <address>
			text = addressText(&mail.Address{Name: theApp.senderName(account), Address: account})
		}
		ch.Items = append(ch.Items, core.ChooserItem{Value: SenderAccount(account), Text: text})
	}
	return ch
}

// switchSender changes the signature of the given message being written in
// the given editor to the default one of the given new account, and its
// automatic Bcc from the given old account to it, as when its account is
// changed.
func switchSender(sm *SendMessage, ed *textcore.Editor, old, new string) {
	sig := Settings.DefaultSignatures[new]
	switchSignature(sm, ed, sm.Sender.Signature, sig)
	sm.Sender.Signature = sig
	if Settings.AutoBCC {
		for _, ad := range sm.Bcc {
			if ad != nil && strings.EqualFold(ad.Address, old) {
				ad.Address = new
			}
		}
	}
}

// switchSignature changes the given old signature of the given message being
// written in the given editor to the given new one, replacing its text, or
// adding the new one above any quoted text, as in a new message, if the old
// one is none.
func switchSignature(sm *SendMessage, ed *textcore.Editor, old, new SignatureName) {
	body := ed.Lines.String()
	os, ns := signatureNamed(old), signatureNamed(new)
	if os == ns {
		return
	}
	switch {
	case os != "" && strings.Contains(body, os):
		body = strings.Replace(body, os, ns, 1)
	case os == "" && ns != "":
		body = "\n\n" + ns + body
	}
	ed.Lines.SetString(body)
	sm.body = body
}

// Compose opens a dialog to send a new message.
func (a *App) Compose() { //types:add
	a.composeMessage = &SendMessage{}
	a.composeMessage.To = []*mail.Address{{}}
	a.compose("Compose")
}

// compose is the implementation of the email composition window for
// [App.composeMessage], which is called by other higher-level functions,
// returning its body. The message is saved as a draft as it is written,
// once it is changed, and when the window is closed, unless it is sent or
// discarded; closing a new message asks whether to save it as a draft or
// discard it.
func (a *App) compose(title string) *core.Body {
	sm := a.composeMessage
	if sm.draft == nil { // a new message
		from := a.currentEmail
		if from == "" && len(Settings.Accounts) > 0 {
			from = Settings.Accounts[0]
		}
		sm.Sender.From = SenderAccount(from)
		sm.Sender.Signature = Settings.DefaultSignatures[from]
		if Settings.AutoBCC { // shown, so that it can be removed for a message
			sm.Bcc = append(sm.Bcc, &mail.Address{Address: from})
		}
		if sig := signatureNamed(sm.Sender.Signature); sig != "" {
			// above any quoted message, as usual
			sm.body = "\n\n" + sig + sm.body
		}
	}
	// All of the compose windows have the same size, which is remembered
	// by the part of the title before the colon.
	b := core.NewBody("Message: " + title)
	b.Styler(func(s *styles.Style) {
		s.Min.Set(units.Ch(80), units.Ch(80))
	})
	form := core.NewForm(b).SetStruct(sm)
	ed := textcore.NewEditor(b)
	core.Bind(&sm.body, ed)
	ed.Lines.SetLanguage(fileinfo.Markdown)
	ed.Lines.Settings.LineNumbers = false
	ed.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
	})

	// Saving the draft once it is changed. The compose window has its own
	// lock, so the message is copied with it, and saved with the lock of
	// the app, in order (see [SendMessage.snapshot]).
	initial := sm.contents()
	changed := sm.draft != nil
	done := false // sent or discarded
	seq := 0
	withApp := func(f func(c *SendMessage)) {
		sm.body = ed.Lines.String()
		seq++
		c := sm.snapshot(seq)
		go func() {
			a.AsyncLock()
			defer a.AsyncUnlock()
			if c.seq < sm.savedSeq {
				return // a later one is done
			}
			sm.savedSeq = c.seq
			f(c)
		}()
	}
	save := func() {
		sm.body = ed.Lines.String()
		if done || (!changed && sm.contents() == initial) {
			return
		}
		changed = true
		withApp(func(c *SendMessage) {
			core.ErrorSnackbar(a, a.saveDraft(c), "Error saving draft")
		})
	}
	var timer *time.Timer
	stopTimer := func() {
		if timer != nil {
			timer.Stop()
		}
	}
	change := func(e events.Event) { // after a pause in writing
		stopTimer()
		timer = time.AfterFunc(draftSaveDelay, func() {
			if b.AsyncLockIfShown() {
				defer b.AsyncUnlock()
				save()
			}
		})
	}
	// the + buttons for more recipients are clicked, not tabbed to, so that
	// Tab goes from To to the Subject
	form.FinalUpdater(func() {
		for w := range form.AllWidgets() {
			if w.AsTree().Name == "add-button" && w.AsWidget().Property("skip-tab") == nil {
				w.AsWidget().SetProperty("skip-tab", true)
				w.AsWidget().FirstStyler(func(s *styles.Style) {
					s.SetAbilities(true, abilities.SkipTab)
				})
			}
		}
	})
	sender, sig := string(sm.Sender.From), sm.Sender.Signature
	form.OnChange(func(e events.Event) {
		switch {
		case string(sm.Sender.From) != sender: // its signature and Bcc
			switchSender(sm, ed, sender, string(sm.Sender.From))
		case sm.Sender.Signature != sig:
			switchSignature(sm, ed, sig, sm.Sender.Signature)
		default:
			return
		}
		sender, sig = string(sm.Sender.From), sm.Sender.Signature
		form.Update()
	})
	form.OnChange(change)
	form.OnInput(change)
	ed.OnInput(change)
	ed.OnChange(change)
	b.OnClose(func(e events.Event) {
		stopTimer()
		save()
	})
	discard := func() {
		stopTimer()
		done = true
		withApp(func(c *SendMessage) {
			if d := sm.draft; d != nil {
				a.removeDraft(d)
				a.Update()
			}
		})
	}
	// A new message that is closed without being sent asks whether to save
	// it as a draft (it is saved as it is written, in case of a crash, and
	// removed if not), so that drafts do not accumulate from messages that
	// are not sent; a draft being edited is saved.
	isNew := sm.draft == nil
	decided := false // to save it as a draft, with its button
	b.AddCloseDialog(func(d *core.Body) bool {
		sm.body = ed.Lines.String()
		if !isNew || done || decided || (!changed && sm.contents() == initial) {
			return false
		}
		d.SetTitle("Save draft?")
		core.NewText(d).SetType(core.TextSupporting).SetText("Save the message in Drafts, to finish it later, or discard it?")
		d.AddBottomBar(func(bar *core.Frame) {
			core.NewButton(bar).SetType(core.ButtonText).SetText("Discard").SetIcon(icons.Delete).OnClick(func(e events.Event) {
				discard()
				b.Close()
			})
			d.AddOK(bar).SetText("Save draft").OnClick(func(e events.Event) {
				decided = true
				b.Close()
			})
		})
		return true
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
		if sm.replyAll != nil { // a reply to a link, which can be to all
			all := core.NewButton(bar).SetType(core.ButtonText).SetText("Reply all").SetIcon(icons.ReplyAll)
			all.SetTooltip("Reply to all of the message, adding its sender and recipients")
			all.OnClick(func(e events.Event) {
				sm.addReplyAll()
				form.Update()
				change(e)
				all.SetEnabled(false).Update()
			})
		}
		core.NewButton(bar).SetType(core.ButtonText).SetText("Discard").SetIcon(icons.Delete).
			OnClick(func(e events.Event) {
				discard()
				b.Close()
			})
		b.AddCancel(bar).SetText("Save draft").OnClick(func(e events.Event) {
			decided = true
		})
		b.AddOK(bar).SetText("Send").OnClick(func(e events.Event) {
			stopTimer()
			done = true
			withApp(func(c *SendMessage) {
				d, err := a.queue(c)
				if err != nil {
					core.ErrorSnackbar(a, err, "Error sending message (it is in Drafts)")
					return
				}
				go a.sendOutboxShowing(d.Account)
			})
		})
	})
	b.RunWindowDialog(a)
	return b
}

// Send sends [App.composeMessage] now, putting it in the outbox first,
// where it stays if it cannot be sent, as when offline, to be sent in a
// later sync. It must be called with the lock of the window.
func (a *App) Send() error { //types:add
	d, err := a.queue(a.composeMessage)
	if err != nil {
		return err
	}
	if !a.startSending(d) {
		return nil
	}
	err = a.sendDraft(d)
	if err == nil {
		a.removeDraft(d)
		a.SyncNow() // to show it in the sent mailbox
	}
	a.Update()
	if se, ok := errors.AsType[*smtp.SMTPError](err); ok {
		slog.Error("error sending message: SMTP error:", "code", se.Code, "enhancedCode", se.EnhancedCode, "message", se.Message)
	}
	return err
}

// contents returns the contents of the message, as text, for checking
// whether it has changed.
func (sm *SendMessage) contents() string {
	var b strings.Builder
	fmt.Fprintln(&b, sm.Sender.From, sm.Sender.Signature)
	for _, as := range [][]*mail.Address{sm.To, sm.Cc, sm.Bcc} {
		for _, ad := range as {
			if ad != nil {
				fmt.Fprintf(&b, "%s <%s>,", ad.Name, ad.Address)
			}
		}
		b.WriteString("\n")
	}
	fmt.Fprintln(&b, sm.Subject, sm.Attachments)
	b.WriteString(strings.TrimRight(sm.body, "\n")) // as the editor adds one
	return b.String()
}

// buildMessage returns the message of the given draft, in MIME, with
// its text as plain text and HTML (from markdown), and its attachments.
// Attachments that cannot be read are an error if strict, and else left
// out, as for a draft that is still being written.
func buildMessage(d *draft, strict bool) ([]byte, error) {
	var b bytes.Buffer
	var h mail.Header
	h.SetDate(d.Date)
	h.SetAddressList("From", d.From)
	h.SetAddressList("To", nonEmpty(d.To))
	if cc := nonEmpty(d.Cc); len(cc) > 0 {
		h.SetAddressList("Cc", cc)
	}
	// the Bcc recipients are not in the header
	h.SetSubject(d.Subject)
	h.SetMessageID(d.MessageID)
	if d.InReplyTo != "" {
		h.SetMsgIDList("In-Reply-To", []string{d.InReplyTo})
	}
	if len(d.References) > 0 {
		h.SetMsgIDList("References", d.References)
	}

	mw, err := mail.CreateWriter(&b, h)
	if err != nil {
		return nil, err
	}
	iw, err := mw.CreateInline()
	if err != nil {
		return nil, err
	}
	var ph mail.InlineHeader
	ph.SetContentType("text/plain", nil)
	pw, err := iw.CreatePart(ph)
	if err != nil {
		return nil, err
	}
	pw.Write([]byte(d.Body))
	if err := pw.Close(); err != nil {
		return nil, err
	}
	var hh mail.InlineHeader
	hh.SetContentType("text/html", nil)
	hw, err := iw.CreatePart(hh)
	if err != nil {
		return nil, err
	}
	if err := goldmark.Convert([]byte(d.Body), hw); err != nil {
		return nil, err
	}
	if err := hw.Close(); err != nil {
		return nil, err
	}
	if err := iw.Close(); err != nil {
		return nil, err
	}

	for _, at := range d.Attachments {
		fname := string(at)
		if fname == "" {
			continue
		}
		ah := mail.AttachmentHeader{}
		ah.SetFilename(filepath.Base(fname))
		fi, err := fileinfo.NewFileInfo(fname)
		if err != nil {
			if !strict {
				continue
			}
			return nil, err
		}
		ah.SetContentType(fi.Mime, nil)
		aw, err := mw.CreateAttachment(ah)
		if err != nil {
			return nil, err
		}
		f, err := os.Open(fname)
		if err != nil {
			return nil, err
		}
		_, err = io.Copy(aw, f)
		f.Close()
		if err != nil {
			return nil, err
		}
		if err := aw.Close(); err != nil {
			return nil, err
		}
	}
	if err := mw.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
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
// from the given account, for SMTP: To, Cc, and Bcc, each once.
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
