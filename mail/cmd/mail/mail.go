// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Command mail is Cogent Mail, an email client.
package main

import (
	"flag"
	"log"

	"cogentcore.org/cogent/mail"
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	_ "embed"
)

// appIcon is the icon of the app, which core build also sets, but go
// install does not, as for an app installed with AppGoSymlink.
//
//go:embed icon.svg
var appIcon string

func main() {
	testing := flag.Bool("testing", false, "use a local simulated mail server with sample mail for several accounts, instead of the real accounts")
	keep := flag.Bool("keep", false, "with -testing, keep the mail of the test server between runs")
	plain := flag.Bool("plain", false, "with -testing, use a plain IMAP server instead of one like Gmail")
	flag.Parse()

	core.AppIcon = appIcon
	b := core.NewBody("Cogent Mail")
	a := mail.NewApp(b)
	if *testing {
		// never fall back to the real accounts
		err := errors.Log(a.StartTesting(mail.TestingOptions{Keep: *keep, Plain: *plain}))
		if err != nil {
			log.Fatalln("starting testing mode:", err)
		}
	}
	b.AddTopBar(func(bar *core.Frame) {
		core.NewToolbar(bar).Maker(a.MakeToolbar)
	})
	b.OnShow(func(e events.Event) {
		a.GetMail()
	})
	b.RunMainWindow()
}
