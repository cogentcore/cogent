// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"fmt"
	"slices"
	"strings"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"github.com/cogentcore/go-imap/v2"
	"github.com/cogentcore/go-imap/v2/imapclient"
	"github.com/emersion/go-message/mail"
)

// readMessages returns the messages that the actions on messages apply
// to: those of all of the selected rows of the list if there are more than
// one (see [rowMessages]), or else [App.readMessage], with the other
// messages of its conversation unless [App.readSingle].
func (a *App) readMessages() []*CacheMessage {
	if len(a.selectedRows) > 1 { // all of the messages of the selected rows
		var res []*CacheMessage
		for _, mr := range a.selectedRows {
			for _, cm := range rowMessages(mr) {
				if !slices.Contains(res, cm) {
					res = append(res, cm)
				}
			}
		}
		return res
	}
	if a.readMessage == nil {
		return nil
	}
	if a.readSingle {
		return []*CacheMessage{a.readMessage}
	}
	return append([]*CacheMessage{a.readMessage}, a.readMessage.replies...)
}

// action executes the given function for each of the messages being read
// ([App.readMessages]) in a goroutine with proper locking. This should be
// used for any user action that interacts with messages in IMAP. It also
// automatically saves the cache after the action is completed. Any then
// function is called after the app is updated, with the lock of the window.
func (a *App) action(f func(c *imapclient.Client, cm *CacheMessage) error, then ...func()) {
	a.actionOn(a.currentEmail, a.readMessages(), f, then...)
}

// actionOn is [App.action] for the given messages of the given account.
func (a *App) actionOn(email string, msgs []*CacheMessage, f func(c *imapclient.Client, cm *CacheMessage) error, then ...func()) {
	// Use a goroutine to prevent GUI freezing and a double mutex deadlock
	// with a combination of the renderContext mutex and the imapMu.
	done := func() { // with the lock of the window
		for _, f := range then {
			f()
		}
	}
	msgs = slices.DeleteFunc(slices.Clone(msgs), (*CacheMessage).isLocal) // not on the server
	if len(msgs) == 0 {
		done()
		return
	}
	notConnected := fmt.Errorf("%s is not connected: try again once it is", email)
	mu := a.imapMu[email]
	if mu == nil || a.imapClient[email] == nil {
		core.ErrorSnackbar(a, notConnected, "Offline")
		done()
		return
	}
	go func() {
		mu.Lock()
		c := a.imapClient[email]
		if c == nil { // dropped while waiting
			mu.Unlock()
			a.AsyncLock()
			core.ErrorSnackbar(a, notConnected, "Offline")
			done()
			a.AsyncUnlock()
			return
		}
		for _, cm := range msgs {
			err := f(c, cm)
			if err != nil {
				core.ErrorSnackbar(a, err, "Error performing action")
				break
			}
		}
		err := a.saveMessages(email, msgs...)
		core.ErrorSnackbar(a, err, "Error saving messages")
		mu.Unlock()
		a.AsyncLock()
		a.updateMail()
		for _, f := range then {
			f()
		}
		a.AsyncUnlock()
	}()
}

// tableLabel is used for displaying labels in a table
// for user selection.
type tableLabel struct {
	name  string // the true underlying name
	On    bool   `display:"checkbox"`
	Label string `edit:"-"` // the friendly label name
}

// Label opens a dialog for changing the labels (mailboxes) of the current message.
func (a *App) Label() { //types:add
	d := core.NewBody("Label")
	labels := make([]tableLabel, len(a.readMessage.Labels))
	for i, label := range a.readMessage.Labels {
		labels[i] = tableLabel{name: label.Name, On: label.Name != "INBOX", Label: friendlyLabelName(label.Name)}
	}
	var tb *core.Table
	ch := core.NewChooser(d).SetEditable(true).SetAllowNew(true)
	for _, label := range a.labels[a.currentEmail] {
		ch.Items = append(ch.Items, core.ChooserItem{Value: label, Text: friendlyLabelName(label)})
	}
	ch.OnChange(func(e events.Event) {
		labels = append(labels, tableLabel{name: ch.CurrentItem.Value.(string), On: true, Label: ch.CurrentItem.Text})
		tb.Update()
	})
	ch.OnFinal(events.Change, func(e events.Event) {
		if ch.CurrentItem.Text == "" {
			return
		}
		ch.CurrentItem = core.ChooserItem{}
		ch.SetCurrentValue("")
	})
	tb = core.NewTable(d).SetSlice(&labels)
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).SetText("Save").OnClick(func(e events.Event) {
			newLabels := []string{}
			for _, label := range labels {
				if label.On {
					newLabels = append(newLabels, label.name)
				}
			}
			if len(newLabels) == 0 {
				core.ErrorSnackbar(a, fmt.Errorf("specify at least one label"))
				return
			}
			a.label(newLabels)
		})
	})
	d.RunDialog(a)
}

// label changes the labels of the current message to the given labels.
// newLabels are the labels we want to end up with, in contrast to the
// old labels we started with, which are a.readMessage.Labels. For a
// conversation, the labels added and removed are added to and removed
// from each of its messages, which keep their other labels (as Sent).
func (a *App) label(newLabels []string) {
	var old []string
	for _, l := range a.readMessage.Labels {
		old = append(old, l.Name)
	}
	a.setLabels(func(cm *CacheMessage) []string {
		var res []string
		for _, l := range cm.Labels {
			if slices.Contains(newLabels, l.Name) || !slices.Contains(old, l.Name) {
				res = append(res, l.Name)
			}
		}
		for _, nl := range newLabels {
			if !slices.Contains(res, nl) {
				res = append(res, nl)
			}
		}
		return res
	})
}

// setLabels changes the labels of each of the messages being read
// to those that the given function returns for it. Any then function
// is called after, as for [App.action].
func (a *App) setLabels(labelsFor func(cm *CacheMessage) []string, then ...func()) {
	a.setLabelsOn(a.currentEmail, a.readMessages(), labelsFor, then...)
}

// setLabelsOn is [App.setLabels] for the given messages of the given account.
// The messages are shown with their new labels right away (see
// [App.pendingLabels]), as out of the mailbox being shown when they are
// filed out of it, and the then functions are called then, while the
// server changes them, which takes a while; if that fails, they are
// shown with their labels again.
func (a *App) setLabelsOn(email string, msgs []*CacheMessage, labelsFor func(cm *CacheMessage) []string, then ...func()) {
	targets := map[*CacheMessage][]string{}
	for _, cm := range msgs {
		targets[cm] = labelsFor(cm)
	}
	a.setPending(targets)
	a.updateMail()
	for _, f := range then {
		f()
	}
	a.actionOn(email, msgs, func(c *imapclient.Client, cm *CacheMessage) error {
		newLabels := targets[cm]
		if len(cm.Labels) == 0 {
			return nil
		}
		if a.gmail[email] {
			return a.setGmailLabels(c, email, cm, newLabels)
		}
		// resultantLabels are the labels we apply to cm.Labels after
		// the process is over.
		resultantLabels := slices.Clone(cm.Labels)
		// We copy the message from its first label to all of the new labels.
		src := cm.Labels[0]
		for _, newLabel := range newLabels {
			if slices.ContainsFunc(cm.Labels, func(label Label) bool {
				return label.Name == newLabel
			}) {
				continue // Already have this label.
			}
			if err := a.selectMailbox(c, email, src.Name); err != nil {
				return err
			}
			cd, err := c.Copy(src.UIDSet(), newLabel).Wait()
			if err != nil {
				return err
			}
			// Add this new label to the cache.
			resultantLabels = append(resultantLabels, Label{newLabel, cd.DestUIDs[0].Start})
		}
		// We remove the message from each old label.
		for _, label := range cm.Labels {
			if slices.Contains(newLabels, label.Name) {
				continue // Still have this label.
			}
			if err := a.selectMailbox(c, email, label.Name); err != nil {
				return err
			}
			err := c.Store(label.UIDSet(), &imap.StoreFlags{
				Op:     imap.StoreFlagsAdd,
				Silent: true,
				Flags:  []imap.Flag{imap.FlagDeleted},
			}, nil).Close()
			if err != nil {
				return err
			}
			err = c.UIDExpunge(label.UIDSet()).Close()
			if err != nil {
				return err
			}
			resultantLabels = slices.DeleteFunc(resultantLabels, func(l Label) bool {
				return l == label
			})
		}
		// Save resultantLabels to the cache, with the lock of the window,
		// as it shows the labels.
		a.AsyncLock()
		cm.Labels = resultantLabels
		a.AsyncUnlock()
		return nil
	}, func() { a.clearPending(targets) })
}

// setPending sets the labels that the given messages are shown with,
// while the server changes them (see [App.pendingLabels]).
func (a *App) setPending(targets map[*CacheMessage][]string) {
	if a.pendingLabels == nil {
		a.pendingLabels = map[*CacheMessage][]string{}
	}
	for cm, labels := range targets {
		a.pendingLabels[cm] = labels
	}
}

// clearPending stops showing the given messages with their pending labels,
// once the server has changed them (or failed to), updating the app.
// It must be called with the lock of the window.
func (a *App) clearPending(targets map[*CacheMessage][]string) {
	for cm := range targets {
		delete(a.pendingLabels, cm)
	}
	a.updateMail()
}

// Delete moves the current message to the trash.
func (a *App) Delete() { //types:add
	if slices.ContainsFunc(a.readMessages(), (*CacheMessage).isLocal) {
		a.DiscardDraft() // the local ones, not on the server
		if !slices.ContainsFunc(a.readMessages(), func(cm *CacheMessage) bool { return !cm.isLocal() }) {
			return
		}
	}
	trash := a.trash[a.currentEmail]
	if trash == "" {
		core.ErrorSnackbar(a, fmt.Errorf("no trash mailbox for %s", a.currentEmail))
		return
	}
	a.setLabels(func(cm *CacheMessage) []string { return []string{trash} }, a.selectNext())
}

// Reply opens a dialog to reply to the current message.
func (a *App) Reply() { //types:add
	a.composeMessage = &SendMessage{}
	a.composeMessage.To = IMAPToMailAddresses(a.readMessage.From)
	// If we sent the original message, reply to the original receiver instead of ourself.
	if a.composeMessage.To[0].Address == a.currentEmail {
		a.composeMessage.To = IMAPToMailAddresses(a.readMessage.To)
	}
	a.reply("Reply", false)
}

// ReplyAll opens a dialog to reply to all people involved in the current message.
func (a *App) ReplyAll() { //types:add
	a.composeMessage = &SendMessage{}
	a.composeMessage.To = append(IMAPToMailAddresses(a.readMessage.From), IMAPToMailAddresses(a.readMessage.To)...)
	a.composeMessage.Cc = slices.DeleteFunc(IMAPToMailAddresses(a.readMessage.Cc), func(ma *mail.Address) bool {
		return ma.Address == a.currentEmail
	})
	a.reply("Reply all", false)
}

// Forward opens a dialog to forward the current message to others.
func (a *App) Forward() { //types:add
	a.composeMessage = &SendMessage{}
	a.composeMessage.To = []*mail.Address{{}}
	a.reply("Forward", true)
}

// reply is the implementation of the email reply dialog,
// used by other higher-level functions. forward is whether
// this is actually a forward instead of a reply.
func (a *App) reply(title string, forward bool) {
	a.replyTo(a.readMessage, title, forward)
	a.compose(title)
}

// replyTo sets [App.composeMessage], whose recipients are set, to a reply
// to the given message, or a forward of it, with its subject, references,
// and quoted text.
func (a *App) replyTo(cm *CacheMessage, title string, forward bool) {
	// If we have more than one receiver, then we should not be one of them.
	if len(a.composeMessage.To) > 1 {
		a.composeMessage.To = slices.DeleteFunc(a.composeMessage.To, func(ma *mail.Address) bool {
			return ma.Address == a.currentEmail
		})
		// If all of the receivers were us, then we should reply to ourself.
		if len(a.composeMessage.To) == 0 {
			a.composeMessage.To = []*mail.Address{{Address: a.currentEmail}}
		}
	}
	a.composeMessage.Subject = cm.Subject
	prefix := "Re: "
	if forward {
		prefix = "Fwd: "
	}
	if !strings.HasPrefix(a.composeMessage.Subject, prefix) {
		a.composeMessage.Subject = prefix + a.composeMessage.Subject
	}
	a.composeMessage.inReplyTo = cm.MessageID
	a.composeMessage.references = append(slices.Clone(cm.parsed.references), cm.MessageID)
	from := ""
	if len(cm.From) > 0 {
		from = IMAPToMailAddresses(cm.From)[0].String()
	}
	date := cm.Date.Format("Mon, Jan 2, 2006 at 3:04 PM")
	if forward {
		a.composeMessage.body = "\n\n> Begin forwarded message:\n>"
		a.composeMessage.body += "\n> From: " + from
		// Need 2 spaces to create a newline in markdown.
		a.composeMessage.body += "  \n> Subject: " + cm.Subject
		a.composeMessage.body += "  \n> Date: " + date
		to := make([]string, len(cm.To))
		for i, addr := range IMAPToMailAddresses(cm.To) {
			to[i] = addr.String()
		}
		a.composeMessage.body += "  \n> To: " + strings.Join(to, ", ")
	} else {
		a.composeMessage.body = "\n\n> On " + date + ", " + from + " wrote:"
	}
	a.composeMessage.body += "\n>\n> "
	a.composeMessage.body += strings.ReplaceAll(cm.parsed.plain, "\n", "\n> ")
}

// MarkAsRead marks the current message as read.
func (a *App) MarkAsRead() { //types:add
	a.markSeen(true)
}

// MarkAsUnread marks the current message as unread.
func (a *App) MarkAsUnread() { //types:add
	a.markSeen(false)
}

// markSeen sets the [imap.FlagSeen] flag of the messages being read.
func (a *App) markSeen(seen bool) {
	a.setFlagOn(a.currentEmail, a.readMessages(), imap.FlagSeen, seen)
}

// viewLabel returns the label of the given message that has its UID, on
// Gmail its mailbox: All Mail, the Trash, or the Spam, and false if none
// does.
func viewLabel(cm *CacheMessage) (Label, bool) {
	for _, l := range cm.Labels {
		if l.UID > 0 {
			return l, true
		}
	}
	return Label{}, false
}

// setGmailLabels changes the labels of the given message of the given
// account on Gmail to the given ones, with the client of the account:
// moving it to the Trash or the Spam if they are in them, and else adding
// and removing its Gmail labels (X-GM-LABELS) as needed, in its mailbox.
func (a *App) setGmailLabels(c *imapclient.Client, email string, cm *CacheMessage, labels []string) error {
	view, ok := viewLabel(cm)
	if !ok {
		return fmt.Errorf("the message %q is not synced yet", cm.Subject)
	}
	if err := a.selectMailbox(c, email, view.Name); err != nil {
		return err
	}
	for _, attr := range []imap.MailboxAttr{imap.MailboxAttrTrash, imap.MailboxAttrJunk} {
		mb := a.specialUse[email][attr]
		if mb == "" || mb == view.Name || !slices.Contains(labels, mb) {
			continue
		}
		md, err := c.Move(view.UIDSet(), mb).Wait()
		if err != nil {
			return err
		}
		uid := imap.UID(0)
		if md != nil {
			if dst, ok := md.DestUIDs.(imap.UIDSet); ok {
				if uids, ok := dst.Nums(); ok && len(uids) == 1 {
					uid = uids[0]
				}
			}
		}
		a.AsyncLock()
		cm.Labels = []Label{{mb, uid}}
		a.AsyncUnlock()
		return nil
	}
	if view.Name != a.specialUse[email][imap.MailboxAttrAll] && !slices.Contains(labels, view.Name) {
		// out of the Trash or the Spam: moved to the first of the labels,
		// and it is in All Mail again, with a new UID after the next sync
		dest := a.specialUse[email][imap.MailboxAttrAll]
		if len(labels) > 0 {
			dest = labels[0]
		}
		if _, err := c.Move(view.UIDSet(), dest).Wait(); err != nil {
			return err
		}
		res := []Label{}
		for _, l := range labels {
			res = append(res, Label{Name: l})
		}
		a.AsyncLock()
		cm.Labels = res
		a.AsyncUnlock()
		a.SyncNow()
		return nil
	}
	var add, remove []string
	for _, l := range labels {
		if l != view.Name && !slices.ContainsFunc(cm.Labels, func(x Label) bool { return x.Name == l }) {
			add = append(add, a.gmailLabel(email, l))
		}
	}
	for _, l := range cm.Labels {
		if l.UID == 0 && !slices.Contains(labels, l.Name) {
			remove = append(remove, a.gmailLabel(email, l.Name))
		}
	}
	for _, change := range []struct {
		op     imap.StoreFlagsOp
		labels []string
	}{{imap.StoreFlagsAdd, add}, {imap.StoreFlagsDel, remove}} {
		if len(change.labels) == 0 {
			continue
		}
		err := c.StoreGmailLabels(view.UIDSet(), &imap.StoreGmailLabels{Op: change.op, Silent: true, Labels: change.labels}).Close()
		if err != nil {
			return err
		}
	}
	res := []Label{view}
	for _, l := range labels {
		if l != view.Name {
			res = append(res, Label{Name: l})
		}
	}
	a.AsyncLock()
	cm.Labels = res
	a.AsyncUnlock()
	return nil
}
