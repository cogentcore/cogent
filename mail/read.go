// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"io"
	"os"
	"path/filepath"
	"time"

	"cogentcore.org/core/base/errors"
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

// updateReadMessage updates the given frame to display the contents of the current message.
func (a *App) updateReadMessage(w *core.Frame) error {
	w.DeleteChildren()

	if a.readMessage == nil {
		return nil
	}

	bemail := FilenameBase32(a.currentEmail)

	f, err := os.Open(filepath.Join(core.TheApp.AppDataDir(), "mail", bemail, a.readMessage.Filename))
	if err != nil {
		return err
	}
	defer f.Close()

	mr, err := mail.CreateReader(f)
	if err != nil {
		return err
	}

	var plain *mail.Part
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
				plain = p
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
	if !gotHTML && plain != nil {
		err := htmlcore.ReadMD(htmlcore.NewContext(), w, errors.Log1(io.ReadAll(plain.Body)))
		if err != nil {
			return err
		}
	}
	return nil
}
