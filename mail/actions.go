// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"cogentcore.org/core/core"
	"github.com/emersion/go-imap/v2"
)

// MoveMessage moves the current message to the given mailbox.
func (a *App) MoveMessage(mailbox string) { //types:add
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
