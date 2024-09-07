// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"fmt"

	"github.com/emersion/go-imap/v2"
)

// MoveMessage moves the current message to the given mailbox.
func (a *App) MoveMessage(mailbox string) error { //types:add
	c := a.imapClient[a.currentEmail]
	uidset := imap.UIDSet{}
	uidset.AddNum(a.readMessage.UID)
	fmt.Println(uidset)
	mc := c.Move(uidset, mailbox)
	fmt.Println("mc", mc)
	md, err := mc.Wait()
	fmt.Println("md", md, err)
	return err
}
