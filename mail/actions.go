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
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-message/mail"
)

// readMessages returns the messages being read: [App.readMessage],
// with the other messages of its conversation unless [App.readSingle].
func (a *App) readMessages() []*CacheMessage {
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
// automatically saves the cache after the action is completed.
func (a *App) action(f func(c *imapclient.Client, cm *CacheMessage) error) {
	email := a.currentEmail
	msgs := a.readMessages()
	// Use a goroutine to prevent GUI freezing and a double mutex deadlock
	// with a combination of the renderContext mutex and the imapMu.
	go func() {
		mu := a.imapMu[email]
		mu.Lock()
		c := a.imapClient[email]
		for _, cm := range msgs {
			err := f(c, cm)
			if err != nil {
				core.ErrorSnackbar(a, err, "Error performing action")
				break
			}
		}
		err := a.saveCacheFile(a.cache[email], email)
		core.ErrorSnackbar(a, err, "Error saving cache")
		mu.Unlock()
		a.AsyncLock()
		a.Update()
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
// to those that the given function returns for it.
func (a *App) setLabels(labelsFor func(cm *CacheMessage) []string) {
	email := a.currentEmail
	targets := map[*CacheMessage][]string{}
	for _, cm := range a.readMessages() {
		targets[cm] = labelsFor(cm)
	}
	a.action(func(c *imapclient.Client, cm *CacheMessage) error {
		newLabels := targets[cm]
		if len(cm.Labels) == 0 {
			return nil
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
			}, nil).Wait()
			if err != nil {
				return err
			}
			err = c.UIDExpunge(label.UIDSet()).Wait()
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
	})
}

// Delete moves the current message to the trash.
func (a *App) Delete() { //types:add
	trash := a.trash[a.currentEmail]
	if trash == "" {
		core.ErrorSnackbar(a, fmt.Errorf("no trash mailbox for %s", a.currentEmail))
		return
	}
	a.setLabels(func(cm *CacheMessage) []string { return []string{trash} })
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
	a.composeMessage.Subject = a.readMessage.Subject
	prefix := "Re: "
	if forward {
		prefix = "Fwd: "
	}
	if !strings.HasPrefix(a.composeMessage.Subject, prefix) {
		a.composeMessage.Subject = prefix + a.composeMessage.Subject
	}
	a.composeMessage.inReplyTo = a.readMessage.MessageID
	a.composeMessage.references = append(a.readMessage.parsed.references, a.readMessage.MessageID)
	from := IMAPToMailAddresses(a.readMessage.From)[0].String()
	date := a.readMessage.Date.Format("Mon, Jan 2, 2006 at 3:04 PM")
	if forward {
		a.composeMessage.body = "\n\n> Begin forwarded message:\n>"
		a.composeMessage.body += "\n> From: " + from
		// Need 2 spaces to create a newline in markdown.
		a.composeMessage.body += "  \n> Subject: " + a.readMessage.Subject
		a.composeMessage.body += "  \n> Date: " + date
		to := make([]string, len(a.readMessage.To))
		for i, addr := range IMAPToMailAddresses(a.readMessage.To) {
			to[i] = addr.String()
		}
		a.composeMessage.body += "  \n> To: " + strings.Join(to, ", ")
	} else {
		a.composeMessage.body = "\n\n> On " + date + ", " + from + " wrote:"
	}
	a.composeMessage.body += "\n>\n> "
	a.composeMessage.body += strings.ReplaceAll(a.readMessage.parsed.plain, "\n", "\n> ")
	a.compose(title)
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
	if !slices.ContainsFunc(a.readMessages(), func(cm *CacheMessage) bool {
		return slices.Contains(cm.Flags, imap.FlagSeen) != seen
	}) {
		return // Already set correctly.
	}
	email := a.currentEmail
	a.action(func(c *imapclient.Client, cm *CacheMessage) error {
		if slices.Contains(cm.Flags, imap.FlagSeen) == seen {
			return nil // Already set correctly.
		}
		op := imap.StoreFlagsDel
		if seen {
			op = imap.StoreFlagsAdd
		}
		for _, label := range cm.Labels {
			if err := a.selectMailbox(c, email, label.Name); err != nil {
				return err
			}
			err := c.Store(label.UIDSet(), &imap.StoreFlags{
				Op:     op,
				Silent: true,
				Flags:  []imap.Flag{imap.FlagSeen},
			}, nil).Wait()
			if err != nil {
				return err
			}
		}
		// Also directly update the cache, with the lock of the window,
		// as it shows the flags.
		a.AsyncLock()
		defer a.AsyncUnlock()
		if seen {
			cm.Flags = append(cm.Flags, imap.FlagSeen)
		} else {
			cm.Flags = slices.DeleteFunc(cm.Flags, func(flag imap.Flag) bool {
				return flag == imap.FlagSeen
			})
		}
		return nil
	})
}
