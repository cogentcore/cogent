// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"strings"

	"cogentcore.org/core/core"
	"github.com/emersion/go-imap/v2"
)

// Move moves the current message to the given mailbox.
func (a *App) Move(mailbox string) { //types:add
	// Use a goroutine to prevent GUI freezing and a double mutex deadlock
	// with a combination of the renderContext mutex and the imapMu.
	go func() {
		mu := a.imapMu[a.currentEmail]
		mu.Lock()
		defer mu.Unlock()
		c := a.imapClient[a.currentEmail]
		uidset := imap.UIDSet{}
		uidset.AddNum(a.readMessage.UID)
		mc := c.Move(uidset, mailbox)
		_, err := mc.Wait()
		core.ErrorSnackbar(a, err, "Error moving message")
	}()
}

// Reply opens a dialog to reply to the current message.
func (a *App) Reply() { //types:add
	a.composeMessage = &SendMessage{}
	a.composeMessage.To = IMAPToMailAddresses(a.readMessage.From)
	a.reply("Reply")
}

// reply is the implementation of the email reply dialog,
// used by other higher-level functions.
func (a *App) reply(title string) {
	a.composeMessage.Subject = a.readMessage.Subject
	if !strings.HasPrefix(a.composeMessage.Subject, "Re: ") {
		a.composeMessage.Subject = "Re: " + a.composeMessage.Subject
	}
	a.composeMessage.inReplyTo = a.readMessage.MessageID
	a.composeMessage.references = []string{a.readMessage.MessageID} // TODO: append to any existing references in the readMessage
	a.compose(title)
}
