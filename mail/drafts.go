// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/tree"
	"github.com/cogentcore/go-imap/v2"
	"github.com/emersion/go-message/mail"
)

// draft is a message that is being written, which is saved as it is
// written, or that is in the outbox, waiting to be sent, as when offline.
// They are kept locally, in the local directory of the messages of their
// account (see [App.localDir]): their data in a JSON file, and the message
// in a file next to it, for reading it like the other messages. They are
// shown as local messages (see [CacheMessage.isLocal]) in the Drafts
// mailbox of their account, and in its Outbox ([outboxLabel]).
type draft struct {

	// ID is the unique ID of the draft, the base of the names of its files.
	ID string

	// Account is the account it is from.
	Account string

	From, To, Cc, Bcc []*mail.Address
	Subject           string
	Attachments       []core.Filename
	Body              string
	InReplyTo         string

	// Signature is the name of the signature of the message.
	Signature SignatureName `json:",omitempty"`

	References []string

	// MessageID is the Message-ID of the message (without angle brackets).
	MessageID string

	// Date is when it was last saved, or sent for one in the outbox.
	Date time.Time

	// Queued is whether it is in the outbox, waiting to be sent.
	Queued bool

	// Error is the error of the server refusing to send it, if it did,
	// in which case it is not sent again until the user sends it again.
	Error string `json:",omitempty"`

	// sending is whether it is being sent, under [App.draftsMu].
	sending bool
}

// outboxLabel is the label of the local mailbox of the messages
// waiting to be sent: the outbox.
const outboxLabel = "Outbox"

// localPrefix is the start of the keys of the local messages in the cache,
// the drafts and those in the outbox, followed by their IDs.
const localPrefix = "local-"

// isLocal returns whether the message is local (a draft or in the outbox),
// not on the server.
func (cm *CacheMessage) isLocal() bool {
	return strings.HasPrefix(cm.key, localPrefix)
}

// localDir returns the directory of the drafts of the given account.
func (a *App) localDir(email string) string {
	return filepath.Join(a.messagesDir(email), "local")
}

// draftsLabel returns the label of the drafts of the given account:
// its Drafts mailbox, or "Drafts" if it is not known.
func (a *App) draftsLabel(email string) string {
	if mb := a.specialUse[email][imap.MailboxAttrDrafts]; mb != "" {
		return mb
	}
	return "Drafts"
}

// mailboxLabels returns the labels of the mailboxes of the given account,
// with the local ones of its drafts and outbox, if they are not on the
// server, as the outbox. It is called for each mailbox of the tree of
// mailboxes, so it only looks at the drafts, not all of the messages.
func (a *App) mailboxLabels(email string) []string {
	labels := a.labels[email]
	if st := a.starredLabel(email); len(labels) > 0 && !slices.Contains(labels, st) {
		labels = append(slices.Clone(labels), st) // a local one, by the flag
	}
	a.draftsMu.Lock()
	defer a.draftsMu.Unlock()
	for _, d := range a.drafts {
		if d.Account != email {
			continue
		}
		label := a.draftsLabel(email)
		if d.Queued {
			label = outboxLabel
		}
		if !slices.Contains(labels, label) {
			labels = append(slices.Clone(labels), label)
		}
	}
	return labels
}

// newDraftID returns a new unique ID for a draft.
func newDraftID() string {
	b := make([]byte, 10)
	rand.Read(b)
	return strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b))
}

// draftOf returns the draft of the given local message, or nil.
func (a *App) draftOf(cm *CacheMessage) *draft {
	if cm == nil || !cm.isLocal() {
		return nil
	}
	a.draftsMu.Lock()
	defer a.draftsMu.Unlock()
	return a.drafts[strings.TrimPrefix(cm.key, localPrefix)]
}

// loadDrafts loads the drafts of the given account from its local
// directory, adding them to the cache. It must be called with the lock
// of the window.
func (a *App) loadDrafts(email string) {
	files, _ := filepath.Glob(filepath.Join(a.localDir(email), "*.json"))
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			slog.Error("mail: loading draft", "file", f, "err", err)
			continue
		}
		d := &draft{}
		if err := json.Unmarshal(data, d); err != nil {
			slog.Error("mail: loading draft", "file", f, "err", err)
			continue
		}
		d.Account = email
		a.draftsMu.Lock()
		a.drafts[d.ID] = d
		a.draftsMu.Unlock()
		a.setLocal(d)
	}
}

// setLocal sets the local message of the given draft in the cache of its
// account. It must be called with the lock of the window.
func (a *App) setLocal(d *draft) {
	mp := a.cache[d.Account]
	if mp == nil {
		mp = map[string]*CacheMessage{}
		a.cache[d.Account] = mp
	}
	label := a.draftsLabel(d.Account)
	if d.Queued {
		label = outboxLabel
	}
	cm := &CacheMessage{
		Envelope: imap.Envelope{Date: d.Date, Subject: d.Subject, MessageID: d.MessageID,
			From: imapAddresses(d.From), To: imapAddresses(nonEmpty(d.To)),
			Cc: imapAddresses(nonEmpty(d.Cc)), Bcc: imapAddresses(nonEmpty(d.Bcc))},
		Filename: filepath.Join("local", d.ID+".eml"),
		Flags:    []imap.Flag{imap.FlagSeen},
		Labels:   []Label{{Name: label}},
		key:      localPrefix + d.ID,
	}
	for _, at := range d.Attachments {
		if at != "" {
			cm.Attachments++
		}
	}
	cm.AttachmentsCounted = true
	if d.InReplyTo != "" {
		cm.InReplyTo = []string{d.InReplyTo}
		for _, o := range mp { // in its conversation
			if o.MessageID == d.InReplyTo && !o.isLocal() {
				cm.GmailThreadID = o.GmailThreadID
				break
			}
		}
	}
	if old := mp[cm.key]; old != nil && a.readMessage == old {
		a.readMessage = cm
	}
	mp[cm.key] = cm
}

// imapAddresses converts the given addresses to IMAP addresses.
func imapAddresses(as []*mail.Address) []imap.Address {
	var res []imap.Address
	for _, ad := range as {
		if ad == nil {
			continue
		}
		mb, host, _ := strings.Cut(strings.TrimSpace(ad.Address), "@")
		res = append(res, imap.Address{Name: ad.Name, Mailbox: mb, Host: host})
	}
	return res
}

// writeDraft writes the files of the given draft: its data and its message.
func (a *App) writeDraft(d *draft) error {
	dir := a.localDir(d.Account)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	msg, err := buildMessage(d, d.Queued) // the attachments must be there to send it
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(d, "", "\t")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, d.ID+".eml"), msg, 0o600); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, d.ID+".json"), data, 0o600)
}

// removeDraft removes the given draft, with its files, as after it is sent
// or discarded (see [App.deleteDraft]) or moved to another account.
// It must be called with the lock of the window.
func (a *App) removeDraft(d *draft) {
	a.draftsMu.Lock()
	delete(a.drafts, d.ID)
	a.draftsMu.Unlock()
	dir := a.localDir(d.Account)
	os.Remove(filepath.Join(dir, d.ID+".json"))
	os.Remove(filepath.Join(dir, d.ID+".eml"))
	key := localPrefix + d.ID
	if cm := a.cache[d.Account][key]; cm != nil {
		delete(a.cache[d.Account], key)
		if a.readMessage == cm {
			a.readMessage = nil
		}
		a.selectedRows = slices.DeleteFunc(a.selectedRows, func(mr *messageRow) bool { return mr.message == cm })
	}
}

// setDraft sets the given draft from the given message being written.
func (sm *SendMessage) setDraft(d *draft) {
	d.From = []*mail.Address{{Address: string(sm.Sender.From)}}
	d.Signature = sm.Sender.Signature
	d.To, d.Cc, d.Bcc = sm.To, sm.Cc, sm.Bcc
	d.Subject, d.Attachments, d.Body = sm.Subject, sm.Attachments, sm.body
	d.InReplyTo, d.References = sm.inReplyTo, sm.references
}

// sendMessage returns the message being written of the draft.
func (d *draft) sendMessage() *SendMessage {
	var from SenderAccount
	if len(d.From) > 0 && d.From[0] != nil {
		from = SenderAccount(d.From[0].Address)
	}
	return &SendMessage{Sender: MessageSender{From: from, Signature: d.Signature}, To: d.To, Cc: d.Cc, Bcc: d.Bcc, Subject: d.Subject,
		Attachments: d.Attachments, body: d.Body, inReplyTo: d.InReplyTo, references: d.References, draft: d}
}

// saveDraft saves the given message being written as a draft, creating
// it if it is new, as the draft of its holder (see [SendMessage.holder]).
// It must be called with the lock of the window of the app.
func (a *App) saveDraft(sm *SendMessage) error {
	h := sm.holder()
	a.draftsMu.Lock()
	if d := h.draft; d != nil && d.Account != "" && a.drafts[d.ID] != d {
		h.draft = nil // sent or discarded, so this is a new one
	}
	a.draftsMu.Unlock()
	if h.draft == nil {
		h.draft = &draft{ID: newDraftID()}
	}
	d := h.draft
	a.draftsMu.Lock()
	if d.sending || d.Queued {
		a.draftsMu.Unlock()
		return nil // being sent, as it was
	}
	a.draftsMu.Unlock()
	from := a.currentEmail
	if sm.Sender.From != "" {
		from = string(sm.Sender.From)
	}
	if d.Account != "" && d.Account != from { // moved to another account
		a.removeDraft(d)
	}
	d.Account = from
	sm.setDraft(d)
	if len(d.From) == 1 && d.From[0] != nil { // with the name of the sender
		d.From[0].Name = a.senderName(d.From[0].Address)
	}
	if d.MessageID == "" {
		d.MessageID = newMessageID(from)
	}
	d.Date = time.Now()
	if err := a.writeDraft(d); err != nil {
		return err
	}
	a.draftsMu.Lock()
	a.drafts[d.ID] = d
	a.draftsMu.Unlock()
	a.setLocal(d)
	a.Update()
	return nil
}

// queue puts the given message being written in the outbox, to be sent
// (see [App.sendOutbox]), after checking that it can be sent, and returns
// its draft. If it cannot be sent, it is saved as a draft, and the error
// is returned. It must be called with the lock of the window.
func (a *App) queue(sm *SendMessage) (*draft, error) {
	if sm.Sender.From == "" {
		return nil, errors.New("no sender")
	}
	if err := a.saveDraft(sm); err != nil {
		return nil, err
	}
	d := sm.holder().draft
	if len(recipients(d.Account, sm)) == 0 {
		return nil, fmt.Errorf("no recipients")
	}
	d.Queued, d.Error, d.Date = true, "", time.Now()
	if err := a.writeDraft(d); err != nil { // checks the attachments
		d.Queued = false
		a.writeDraft(d)
		return nil, err
	}
	a.setLocal(d)
	a.Update()
	return d, nil
}

// startSending returns whether the given draft is now being sent, which it
// is if it is in the outbox and not already being sent.
func (a *App) startSending(d *draft) bool {
	a.draftsMu.Lock()
	defer a.draftsMu.Unlock()
	if !d.Queued || d.sending || a.drafts[d.ID] != d {
		return false
	}
	d.sending = true
	return true
}

// sendDraft sends the message of the given draft in the outbox, which is
// being sent (see [App.startSending]), and records any error of the server
// refusing it, after which it is no longer being sent. It does not remove it.
func (a *App) sendDraft(d *draft) error {
	defer func() {
		a.draftsMu.Lock()
		d.sending = false
		a.draftsMu.Unlock()
	}()
	msg, err := os.ReadFile(filepath.Join(a.localDir(d.Account), d.ID+".eml"))
	if err != nil {
		return err
	}
	to := recipients(d.Account, d.sendMessage())
	err = a.send(d.Account, to, strings.NewReader(string(msg)))
	if err == nil {
		go a.saveSent(d.Account, msg, d.MessageID)
	}
	if err != nil && !isOffline(err) {
		a.draftsMu.Lock()
		d.Error = err.Error()
		a.draftsMu.Unlock()
		if werr := a.writeDraft(d); werr != nil {
			slog.Error("mail: saving the error of sending", "err", werr)
		}
	}
	return err
}

// outbox returns the drafts of the given account in the outbox that are
// to be sent: those that are not being sent and that the server has not
// refused, the oldest first.
func (a *App) outbox(email string) []*draft {
	a.draftsMu.Lock()
	defer a.draftsMu.Unlock()
	var res []*draft
	for _, d := range a.drafts {
		if d.Account == email && d.Queued && !d.sending && d.Error == "" {
			res = append(res, d)
		}
	}
	slices.SortFunc(res, func(x, y *draft) int { return x.Date.Compare(y.Date) })
	return res
}

// outboxCount returns the number of messages in the outbox of all of the
// accounts, and how many of them the server refused.
func (a *App) outboxCount() (n, refused int) {
	a.draftsMu.Lock()
	defer a.draftsMu.Unlock()
	for _, d := range a.drafts {
		if d.Queued {
			n++
			if d.Error != "" {
				refused++
			}
		}
	}
	return
}

// sendOutbox sends the messages in the outbox of the given account,
// removing those that are sent. It stops at the first error of not
// being able to reach the server, as when offline, leaving them for the
// next sync. It must be called without the lock of the window.
func (a *App) sendOutbox(email string) error {
	var errs []error
	sent := false
	for _, d := range a.outbox(email) {
		if !a.startSending(d) {
			continue
		}
		err := a.sendDraft(d)
		a.AsyncLock()
		if err == nil {
			a.deleteDraft(d)
			sent = true
		}
		a.Update()
		a.AsyncUnlock()
		if err != nil {
			errs = append(errs, fmt.Errorf("%q: %w", d.Subject, err))
			if isOffline(err) {
				break
			}
		}
	}
	if sent {
		a.SyncNow() // to show them in the sent mailbox
	}
	return joinErrors(errs)
}

// joinErrors joins the given errors, returning nil for none.
func joinErrors(errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	if len(errs) == 1 {
		return errs[0]
	}
	return fmt.Errorf("%w (and %d more)", errs[0], len(errs)-1)
}

// isDraft returns whether the given message is a draft, being written,
// not one in the outbox, waiting to be sent, which double-clicking edits
// (see [App.EditDraft]) instead of showing in a window of its own.
func (a *App) isDraft(cm *CacheMessage) bool {
	d := a.draftOf(cm)
	if d == nil {
		return false
	}
	a.draftsMu.Lock()
	defer a.draftsMu.Unlock()
	return !d.Queued && !d.sending
}

// EditDraft opens the draft being read for writing it more. A message in
// the outbox goes back to the drafts until it is sent again.
func (a *App) EditDraft() { //types:add
	d := a.draftOf(a.readMessage)
	if d == nil {
		return
	}
	a.draftsMu.Lock()
	if d.sending {
		a.draftsMu.Unlock()
		core.MessageSnackbar(a, "The message is being sent")
		return
	}
	if d.Queued {
		d.Queued, d.Error = false, ""
	}
	a.draftsMu.Unlock()
	core.ErrorSnackbar(a, a.writeDraft(d), "Error saving draft")
	a.setLocal(d)
	a.Update()
	a.composeMessage = d.sendMessage()
	a.compose("Draft")
}

// DiscardDraft deletes the drafts and messages in the outbox being read.
func (a *App) DiscardDraft() { //types:add
	for _, cm := range a.readMessages() {
		d := a.draftOf(cm)
		if d == nil {
			continue
		}
		a.draftsMu.Lock()
		sending := d.sending
		a.draftsMu.Unlock()
		if sending {
			core.MessageSnackbar(a, "The message is being sent")
			continue
		}
		a.deleteDraft(d)
	}
	a.Update()
}

// SendAgain sends the messages in the outbox being read that the server
// refused, as after fixing the problem, and any others waiting.
func (a *App) SendAgain() { //types:add
	email := a.currentEmail
	for _, cm := range a.readMessages() {
		if d := a.draftOf(cm); d != nil && d.Queued {
			a.draftsMu.Lock()
			d.Error = ""
			a.draftsMu.Unlock()
		}
	}
	go a.sendOutboxShowing(email)
}

// sendOutboxShowing sends the outbox of the given account (see
// [App.sendOutbox]), showing any error other than being offline.
func (a *App) sendOutboxShowing(email string) {
	err := a.sendOutbox(email)
	if isOffline(err) {
		core.MessageSnackbar(a, "Offline: the message will be sent once connected")
		return
	}
	core.ErrorSnackbar(a, err, "Error sending message")
}

// draftButtons adds the buttons of the actions on the local messages
// being read: the drafts and those in the outbox.
func (a *App) draftButtons(p *tree.Plan) {
	d := a.draftOf(a.readMessage)
	tree.AddAt(p, "EditDraft", func(w *core.FuncButton) {
		w.SetFunc(a.EditDraft).SetText("Edit").SetIcon(icons.Edit)
	})
	if d != nil && d.Queued {
		tree.AddAt(p, "SendAgain", func(w *core.FuncButton) {
			w.SetFunc(a.SendAgain).SetText("Send now").SetIcon(icons.Send)
		})
	}
	tree.AddAt(p, "DiscardDraft", func(w *core.FuncButton) {
		w.SetFunc(a.DiscardDraft).SetText("Discard").SetIcon(icons.Delete).SetShortcut(shortcutOf("Delete"))
	})
}

// draftStatus returns the text of the status of the given local message,
// shown when reading it, or "".
func (a *App) draftStatus(cm *CacheMessage) string {
	d := a.draftOf(cm)
	if d == nil {
		return ""
	}
	a.draftsMu.Lock()
	defer a.draftsMu.Unlock()
	switch {
	case d.Error != "":
		return "Not sent: " + d.Error
	case d.sending:
		return "Sending..."
	case d.Queued:
		return "Waiting to be sent, once connected"
	}
	return "Draft, saved " + d.Date.Local().Format("Jan 2 3:04 PM")
}

// draftSaveDelay is the pause in writing a message after which it is
// saved as a draft.
var draftSaveDelay = 2 * time.Second
