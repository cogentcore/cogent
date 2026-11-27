// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Command mailtestserver runs the simulated test mail server of Cogent Mail
// ([testserver]) on its own, with sample mail, for use with any mail client.
// All of the accounts have the password "password"; neither server uses TLS.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"

	"cogentcore.org/cogent/mail/testserver"
)

func main() {
	imapAddr := flag.String("imap", "127.0.0.1:1143", "the address of the IMAP server")
	smtpAddr := flag.String("smtp", "127.0.0.1:1025", "the address of the SMTP server")
	accounts := flag.String("accounts", strings.Join(testserver.DefaultAccounts, ","), "the email addresses of the accounts, separated by commas")
	plain := flag.Bool("plain", false, "use a plain IMAP server instead of one like Gmail")
	dir := flag.String("dir", "", "the directory to load the mail from and save it to on exit (default: new sample mail, not saved)")
	seed := flag.Uint64("seed", 0, "the random seed of the sample mail")
	flag.Parse()

	srv, err := testserver.New(testserver.Options{Accounts: strings.Split(*accounts, ","), Plain: *plain, IMAPAddr: *imapAddr, SMTPAddr: *smtpAddr})
	if err != nil {
		log.Fatal(err)
	}
	loaded := false
	if *dir != "" {
		loaded, err = srv.Load(*dir)
		if err != nil {
			log.Fatal(err)
		}
	}
	if !loaded {
		if err := srv.Seed(*seed); err != nil {
			log.Fatal(err)
		}
	}
	fmt.Printf("IMAP: %s  SMTP: %s  (no TLS)\naccounts: %s  password: %s\n", srv.IMAPAddr, srv.SMTPAddr, strings.Join(srv.Accounts, ", "), testserver.Password)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop
	if *dir != "" {
		if err := srv.Save(*dir); err != nil {
			log.Fatal(err)
		}
		fmt.Println("saved to", *dir)
	}
	srv.Close()
}
