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
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
	"github.com/emersion/go-message/mail"
)

// displayMessage represents the metadata necessary to display a message
// for the user to read. It does not contain the actual message contents.
type displayMessage struct {
	From        []*mail.Address `display:"inline"`
	To          []*mail.Address `display:"inline"`
	Subject     string
	Date        time.Time
	Attachments []*Attachment `display:"inline"`
}

func (dm *displayMessage) ShouldDisplay(field string) bool {
	switch field {
	case "Attachments":
		return len(dm.Attachments) > 0
	}
	return true
}

// readMessageParsed contains data parsed from the current message we are reading.
type readMessageParsed struct {

	// references is the References header.
	references []string

	// plain is the plain text body.
	plain string

	// attachments are the attachments.
	attachments []*Attachment
}

// Attachment represents an email attachment when reading a message.
type Attachment struct {
	Filename string
	Data     []byte
}

// DisplayMessageFrame is a frame that displays the metadata and contents of a message.
type DisplayMessageFrame struct {
	core.Frame
	Message *CacheMessage
}

func (dmf *DisplayMessageFrame) WidgetValue() any { return &dmf.Message }

func (dmf *DisplayMessageFrame) Init() {
	dmf.Frame.Init()
	dmf.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 0)
		s.Direction = styles.Column
	})
	tree.AddChild(dmf, func(w *core.Form) {
		w.SetReadOnly(true)
		w.Updater(func() {
			w.SetStruct(dmf.Message.ToDisplay(&theApp.readMessageParsed))
		})
	})
	tree.AddChild(dmf, func(w *core.Frame) {
		w.Styler(func(s *styles.Style) {
			s.Direction = styles.Column
			s.Grow.Set(1, 0)
		})
		w.Updater(func() {
			core.ErrorSnackbar(w, dmf.displayMessageContents(w), "Error reading message")
		})
	})
}

// displayMessageContents updates the given frame to display the contents of
// the current message, if it does not already.
func (dmf *DisplayMessageFrame) displayMessageContents(w *core.Frame) error {
	if dmf.Message == w.Property("readMessage") {
		return nil
	}
	w.SetProperty("readMessage", dmf.Message)
	w.DeleteChildren()

	bemail := filenameBase32(theApp.currentEmail)

	f, err := os.Open(filepath.Join(core.TheApp.AppDataDir(), "mail", bemail, dmf.Message.Filename))
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
	theApp.readMessageParsed.references = refs

	theApp.readMessageParsed.attachments = nil
	gotHTML := false
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
				theApp.readMessageParsed.plain = string(b)
			case "text/html":
				err := htmlcore.ReadHTML(htmlcore.NewContext(), w, p.Body)
				if err != nil {
					return err
				}
				gotHTML = true
			}
		case *mail.AttachmentHeader:
			fname, err := h.Filename()
			if err != nil {
				return err
			}
			at := &Attachment{Filename: fname}
			at.Data, err = io.ReadAll(p.Body)
			if err != nil {
				return err
			}
			theApp.readMessageParsed.attachments = append(theApp.readMessageParsed.attachments, at)
		}
	}

	// we only handle the plain version if there is no HTML version
	if !gotHTML {
		err := htmlcore.ReadMDString(htmlcore.NewContext(), w, theApp.readMessageParsed.plain)
		if err != nil {
			return err
		}
	}
	return nil
}
