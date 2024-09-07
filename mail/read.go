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

// UpdateReadMessage updates the view of the message currently being read.
func (a *App) UpdateReadMessage() error {
	msv := a.FindPath("splits/mail/msv").(*core.Form)
	msv.SetStruct(a.ReadMessage.ToMessage())

	mb := a.FindPath("splits/mail/mb").(*core.Frame)
	mb.DeleteChildren()

	bemail := FilenameBase32(a.CurrentEmail)

	f, err := os.Open(filepath.Join(core.TheApp.AppDataDir(), "mail", bemail, a.ReadMessage.Filename))
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
				err := htmlcore.ReadHTML(htmlcore.NewContext(), mb, p.Body)
				if err != nil {
					return err
				}
				gotHTML = true
			}
		}
	}

	// we only handle the plain version if there is no HTML version
	if !gotHTML && plain != nil {
		err := htmlcore.ReadMD(htmlcore.NewContext(), mb, errors.Log1(io.ReadAll(plain.Body)))
		if err != nil {
			return err
		}
	}

	mb.Update()
	return nil
}
