// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/htmlcore"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/tree"
	"github.com/cogentcore/go-imap/v2"
	"github.com/emersion/go-message/mail"
)

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

	// openedFile is the temporary file it was last opened from, if any.
	openedFile string
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
	var header *core.Frame
	tree.AddChild(dmf, func(w *core.Frame) {
		header = w
		dmf.initHeader(w)
	})
	tree.AddChild(dmf, func(w *core.Frame) {
		w.Styler(func(s *styles.Style) {
			s.Direction = styles.Column
			s.Grow.Set(1, 0)
		})
		w.Updater(func() {
			if dmf.Message.Filename == "" { // not downloaded yet
				if w.Property("readMessage") != "downloading" {
					w.SetProperty("readMessage", "downloading")
					w.DeleteChildren()
					core.NewText(w).SetType(core.TextSupporting).SetText("Downloading the message...")
				}
				theApp.downloadNow(theApp.currentEmail, dmf.Message)
				return
			}
			if dmf.Message != w.Property("readMessage") {
				core.ErrorSnackbar(w, dmf.displayMessageContents(w), "Error reading message")
				header.Update() // with the attachments, which are now parsed
			}
		})
	})
}

// messageContext returns a new context for reading the HTML (or markdown)
// of a message, with a bar on the left of quoted text.
func messageContext() *htmlcore.Context {
	bar := colors.AsHex(colors.ToUniform(colors.Scheme.Primary.Base))
	text := colors.AsHex(colors.ToUniform(colors.Scheme.OnSurfaceVariant))
	return htmlcore.NewContext().SetStyle(fmt.Sprintf(
		"blockquote { margin: 0.5em 0 0.5em 0.25em; padding-left: 0.75em; border-left: 3px solid %s; color: %s; }", bar, text))
}

// headerDateFormat is the format of the date in the header of a message.
const headerDateFormat = "Mon, Jan 2, 2006 at 3:04 PM"

// addressList returns the given addresses as text, with their names.
func addressList(as []imap.Address) string {
	var res []string
	for _, a := range as {
		if a.Name != "" && a.Name != a.Addr() {
			res = append(res, fmt.Sprintf("%s <%s>", a.Name, a.Addr()))
		} else {
			res = append(res, a.Addr())
		}
	}
	return strings.Join(res, ", ")
}

// otherLabels returns the labels of the given message other than the
// mailbox being shown and those that are not shown, in the order of the
// mailboxes.
func otherLabels(cm *CacheMessage) []string {
	var res []string
	for _, l := range cm.Labels {
		if l.Name != theApp.showLabel && !skipLabels[l.Name] {
			res = append(res, l.Name)
		}
	}
	return sortedLabels(res)
}

// initHeader initializes the header of the message: the names of the
// fields in bold, with their values, and its other labels, which go to
// their mailboxes when clicked, and its attachments.
func (dmf *DisplayMessageFrame) initHeader(w *core.Frame) {
	w.Styler(func(s *styles.Style) {
		s.Display = styles.Grid
		s.Columns = 2
		s.Grow.Set(1, 0)
		s.Gap.Set(units.Em(1), units.Em(0.25))
	})
	w.Maker(func(p *tree.Plan) {
		cm := dmf.Message
		field := func(name string, value func(p *tree.Plan)) {
			tree.AddAt(p, name+"-label", func(w *core.Text) {
				w.SetText(name)
				w.Styler(func(s *styles.Style) {
					s.Font.Weight = rich.Bold
					s.Align.Self = styles.Start
				})
			})
			value(p)
		}
		text := func(name, value string) {
			if value == "" {
				return
			}
			field(name, func(p *tree.Plan) {
				tree.AddAt(p, name, func(w *core.Text) {
					w.Updater(func() { w.SetText(value) })
				})
			})
		}
		text("From", addressList(cm.From))
		text("To", addressList(cm.To))
		text("Cc", addressList(cm.Cc))
		text("Subject", cm.Subject)
		text("Date", cm.Date.Local().Format(headerDateFormat))
		if labels := otherLabels(cm); len(labels) > 0 {
			field("Labels", func(p *tree.Plan) {
				tree.AddAt(p, "Labels", func(w *core.Frame) {
					w.Styler(func(s *styles.Style) {
						s.Wrap = true
						s.Grow.Set(1, 0)
						s.Gap.Set(units.Em(0.25))
					})
					w.Maker(func(p *tree.Plan) {
						for _, label := range otherLabels(dmf.Message) {
							tree.AddAt(p, label, func(w *core.Button) {
								w.SetType(core.ButtonTonal).SetText(friendlyLabelName(label)).SetIcon(icons.Label)
								w.SetTooltip("Show the messages of " + friendlyLabelName(label))
								w.OnClick(func(e events.Event) {
									theApp.showMailbox(theApp.currentEmail, label)
								})
							})
						}
					})
				})
			})
		}
		if len(cm.parsed.attachments) > 0 {
			field("Attachments", func(p *tree.Plan) {
				tree.AddAt(p, "Attachments", func(w *core.Frame) {
					w.Styler(func(s *styles.Style) {
						s.Wrap = true
						s.Grow.Set(1, 0)
						s.Gap.Set(units.Em(0.25))
					})
					w.Maker(func(p *tree.Plan) {
						for i, at := range dmf.Message.parsed.attachments {
							tree.AddAt(p, strconv.Itoa(i), func(w *AttachmentButton) {
								w.Updater(func() { w.Attachment = at })
							})
						}
					})
				})
			})
		}
	})
}

// displayMessageContents updates the given frame to display the contents of
// the current message, if it does not already.
func (dmf *DisplayMessageFrame) displayMessageContents(w *core.Frame) error {
	w.SetProperty("readMessage", dmf.Message)
	w.DeleteChildren()

	f, err := os.Open(filepath.Join(theApp.messagesDir(theApp.currentEmail), dmf.Message.Filename))
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
	dmf.Message.parsed.references = refs

	dmf.Message.parsed.attachments = nil
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
				dmf.Message.parsed.plain = string(b)
			case "text/html":
				ctx := messageContext()
				err := htmlcore.ReadHTML(ctx, w, p.Body)
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
			dmf.Message.parsed.attachments = append(dmf.Message.parsed.attachments, at)
		}
	}

	// we only handle the plain version if there is no HTML version
	if !gotHTML {
		err := htmlcore.ReadMDString(messageContext(), w, dmf.Message.parsed.plain)
		if err != nil {
			return err
		}
	}
	return nil
}
