// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"fmt"
	"slices"
	"strings"

	"cogentcore.org/core/base/iox/jsonx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-message/mail"
)

// action executes the given function in a goroutine with proper locking.
// This should be used for any user action that interacts with a message in IMAP.
// It also automatically saves the cache after the action is completed.
func (a *App) action(f func(c *imapclient.Client)) {
	// Use a goroutine to prevent GUI freezing and a double mutex deadlock
	// with a combination of the renderContext mutex and the imapMu.
	go func() {
		mu := a.imapMu[a.currentEmail]
		mu.Lock()
		f(a.imapClient[a.currentEmail])
		err := jsonx.Save(a.cache[a.currentEmail], a.cacheFilename(a.currentEmail))
		core.ErrorSnackbar(a, err, "Error saving cache")
		mu.Unlock()
		a.AsyncLock()
		a.Update()
		a.AsyncUnlock()
	}()
}

// actionLabels executes the given function for each label of the current message,
// selecting the mailbox for each one first.
func (a *App) actionLabels(f func(c *imapclient.Client, label Label)) {
	a.action(func(c *imapclient.Client) {
		for _, label := range a.readMessage.Labels {
			err := a.selectMailbox(c, a.currentEmail, label.Name)
			if err != nil {
				core.ErrorSnackbar(a, err)
				return
			}
			f(c, label)
		}
	})
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
			a.actionLabels(func(c *imapclient.Client, label Label) {
				if slices.Contains(newLabels, label.Name) {
					return
				}
				fmt.Println("remove", label.Name)
			})
			for _, newLabel := range newLabels {
				if slices.ContainsFunc(a.readMessage.Labels, func(label Label) bool {
					return label.Name == newLabel
				}) {
					continue
				}
				fmt.Println("add", newLabel)
			}
		})
	})
	d.RunDialog(a)
	// TODO: Move needs to be redesigned with the new many-to-many labeling paradigm.
	// a.actionLabels(func(c *imapclient.Client, label Label) {
	// 	uidset := imap.UIDSet{}
	// 	uidset.AddNum(label.UID)
	// 	mc := c.Move(uidset, mailbox)
	// 	_, err := mc.Wait()
	// 	core.ErrorSnackbar(a, err, "Error moving message")
	// })
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
	a.composeMessage.references = append(a.readMessageReferences, a.readMessage.MessageID)
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
	a.composeMessage.body += strings.ReplaceAll(a.readMessagePlain, "\n", "\n> ")
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

// markSeen sets the [imap.FlagSeen] flag of the current message.
func (a *App) markSeen(seen bool) {
	if slices.Contains(a.readMessage.Flags, imap.FlagSeen) == seen {
		// Already set correctly.
		return
	}
	a.actionLabels(func(c *imapclient.Client, label Label) {
		uidset := imap.UIDSet{}
		uidset.AddNum(label.UID)
		op := imap.StoreFlagsDel
		if seen {
			op = imap.StoreFlagsAdd
		}
		cmd := c.Store(uidset, &imap.StoreFlags{
			Op:    op,
			Flags: []imap.Flag{imap.FlagSeen},
		}, nil)
		err := cmd.Wait()
		if err != nil {
			core.ErrorSnackbar(a, err, "Error marking message as read")
			return
		}
		// Also directly update the cache:
		flags := &a.cache[a.currentEmail][a.readMessage.MessageID].Flags
		if seen && !slices.Contains(*flags, imap.FlagSeen) {
			*flags = append(*flags, imap.FlagSeen)
		} else if !seen {
			*flags = slices.DeleteFunc(*flags, func(flag imap.Flag) bool {
				return flag == imap.FlagSeen
			})
		}
	})
}
