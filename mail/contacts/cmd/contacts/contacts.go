// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Command contacts is Cogent Contacts, for looking up and editing the
// contacts that Cogent Mail completes the addresses of messages from.
package main

import (
	"cogentcore.org/cogent/mail/contacts"
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/core"
)

func main() {
	b := core.NewBody("Cogent Contacts")
	br := contacts.NewBrowser(b)
	s, err := contacts.Open(contacts.DefaultDir())
	if errors.Log(err) == nil {
		br.SetStore(s)
	}
	b.AddTopBar(func(bar *core.Frame) {
		core.NewToolbar(bar).Maker(br.MakeToolbar)
	})
	b.RunMainWindow()
}
