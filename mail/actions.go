// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"slices"
	"strings"

	"cogentcore.org/core/core"
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-message/mail"
)

// action executes the given function in a goroutine with proper locking.
// This should be used for any user action that interacts with a message in IMAP.
func (a *App) action(f func()) {
	// Use a goroutine to prevent GUI freezing and a double mutex deadlock
	// with a combination of the renderContext mutex and the imapMu.
	go func() {
		mu := a.imapMu[a.currentEmail]
		mu.Lock()
		f()
		mu.Unlock()
		a.AsyncLock()
		a.Update()
		a.AsyncUnlock()
	}()
}

// Move moves the current message to the given mailbox.
func (a *App) Move(mailbox string) { //types:add
	a.action(func() {
		c := a.imapClient[a.currentEmail]
		uidset := imap.UIDSet{}
		// TODO: we are not guaranteed to be in the right mailbox at this point.
		// The same is true for other similar actions.
		uidset.AddNum(a.readMessage.UID)
		mc := c.Move(uidset, mailbox)
		_, err := mc.Wait()
		core.ErrorSnackbar(a, err, "Error moving message")
	})
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
	a.action(func() {
		c := a.imapClient[a.currentEmail]
		uidset := imap.UIDSet{}
		uidset.AddNum(a.readMessage.UID)
		op := imap.StoreFlagsDel
		if seen {
			op = imap.StoreFlagsAdd
		}
		cmd := c.Store(uidset, &imap.StoreFlags{
			Op:    op,
			Flags: []imap.Flag{imap.FlagSeen},
		}, nil)
		err := cmd.Wait()
		core.ErrorSnackbar(a, err, "Error marking message as read")
	})
}
