// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"io"
	"os"
	"path/filepath"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/htmlcore"
	"github.com/emersion/go-message/mail"
)

// ReadMessage represents the data necessary to display a message
// for the user to read.
type ReadMessage struct {
	From    []*mail.Address `display:"inline"`
	To      []*mail.Address `display:"inline"`
	Subject string
	Date    time.Time
}

// updateReadMessage updates the given frame to display the contents of
// the current message, if it does not already.
func (a *App) updateReadMessage(w *core.Frame) error {
	if a.readMessage == w.Property("readMessage") {
		return nil
	}
	w.SetProperty("readMessage", a.readMessage)
	w.DeleteChildren()
	if a.readMessage == nil {
		return nil
	}

	bemail := FilenameBase32(a.currentEmail)

	f, err := os.Open(filepath.Join(core.TheApp.AppDataDir(), "mail", bemail, messageFilename(&a.readMessage.Envelope)))
	if err != nil {
		return err
	}
	defer f.Close()

	mr, err := mail.CreateReader(f)
	if err != nil {
		return err
	}

	refs, err := mr.Header.MsgIDList("References")
	if err != nil {
		return err
	}
	a.readMessageReferences = refs

	var gotHTML bool

	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		switch h := p.Header.(type) {
		case *mail.InlineHeader:
			ct, _, err := h.ContentType()
			if err != nil {
				return err
			}

			switch ct {
			case "text/plain":
				b, err := io.ReadAll(p.Body)
				if err != nil {
					return err
				}
				a.readMessagePlain = string(b)
			case "text/html":
				err := htmlcore.ReadHTML(htmlcore.NewContext(), w, p.Body)
				if err != nil {
					return err
				}
				gotHTML = true
			}
		}
	}

	// we only handle the plain version if there is no HTML version
	if !gotHTML {
		err := htmlcore.ReadMDString(htmlcore.NewContext(), w, a.readMessagePlain)
		if err != nil {
			return err
		}
	}
	return nil
}
