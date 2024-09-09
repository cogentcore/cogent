// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"cogentcore.org/core/core"
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-message/mail"
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
	a.composeMessage.From = []*mail.Address{{Address: Settings.Accounts[0]}}
	a.composeMessage.To = []*mail.Address{{}}
	a.compose("Reply")
}
