// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"fmt"
	"image"
	"net/mail"
	"os"
	"path/filepath"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/parse/complete"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/textcore"
	"cogentcore.org/core/tree"
	"github.com/mitchellh/go-homedir"
)

func init() {
	core.AddValueType[mail.Address, AddressTextField]()
	core.AddValueType[Attachment, AttachmentButton]()
	core.AddValueType[SignatureText, SignatureButton]()
	core.AddValueType[messageSummary, MessageSummaryView]()
	core.AddValueType[starMark, StarIcon]()
}

// AddressTextField represents a [mail.Address] with a [core.TextField].
// When it is edited, it shows the address with its name, if any, as
// Name <address>, as which it can be typed, or just the address.
type AddressTextField struct {
	core.TextField
	Address mail.Address
}

func (at *AddressTextField) WidgetValue() any { return &at.Address }

func (at *AddressTextField) Init() {
	at.TextField.Init()
	at.SetPasteFilter(linkAddress) // as a mailto: link copied from a page
	if !at.IsReadOnly() {
		// completing the names and addresses of the contacts and the mail
		at.SetCompleter(at, func(data any, text string, posLine, posChar int) complete.Matches {
			if theApp == nil {
				return complete.Matches{}
			}
			return theApp.addressMatches(text)
		}, func(data any, text string, cursorPos int, c complete.Completion, seed string) complete.Edit {
			return completeAddress(text, c)
		})
	}
	at.Updater(func() {
		if at.IsReadOnly() && at.Address.Name != "" && at.Address.Name != at.Address.Address {
			at.SetText(fmt.Sprintf("%s (%s)", at.Address.Name, at.Address.Address))
			return
		}
		at.SetText(addressText(&at.Address))
	})
	at.SetValidator(func() error {
		ad, err := parseAddressText(at.Text())
		if err != nil {
			return err
		}
		at.Address = *ad
		return nil
	})
}

// AttachmentButton represents an [Attachment] with a [core.Button].
type AttachmentButton struct {
	core.Button
	Attachment *Attachment
}

func (ab *AttachmentButton) WidgetValue() any { return &ab.Attachment }

func (ab *AttachmentButton) Init() {
	ab.Button.Init()
	ab.SetIcon(icons.Attachment).SetType(core.ButtonTonal)
	ab.SetTooltip("Click to open; right-click to save")
	ab.Updater(func() {
		ab.SetText(ab.Attachment.Filename)
	})
	ab.OnClick(func(e events.Event) {
		core.ErrorSnackbar(ab, ab.Attachment.Open(), "Error opening attachment")
	})
	ab.AddContextMenu(func(m *core.Scene, pos image.Point) {
		core.NewButton(m).SetText("Open").SetIcon(icons.OpenInNew).OnClick(func(e events.Event) {
			core.ErrorSnackbar(ab, ab.Attachment.Open(), "Error opening attachment")
		})
		core.NewButton(m).SetText("Save as...").SetIcon(icons.Download).OnClick(func(e events.Event) {
			ab.saveAs()
		})
	})
}

// saveAs prompts for the file to save the attachment to.
func (ab *AttachmentButton) saveAs() {
	fb := core.NewSoloFuncButton(ab).SetFunc(func(filename core.Filename) error {
		return os.WriteFile(string(filename), ab.Attachment.Data, 0666)
	})
	fb.Args[0].Value = core.Filename(filepath.Join(errors.Log1(homedir.Dir()), "Downloads", ab.Attachment.Filename))
	fb.CallFunc()
}

// attachmentsDir is the temporary directory of the attachments that have
// been opened, which is removed when the app quits.
var attachmentsDir string

// openURL opens the given URL in its default app.
// It is a variable for the tests, which do not open apps.
var openURL = func(url string) { core.TheApp.OpenURL(url) }

// Open opens the attachment in the default app for its type, as a file
// in a new temporary directory, so that each has its own name.
func (at *Attachment) Open() error {
	if attachmentsDir == "" {
		dir, err := os.MkdirTemp("", "cogent-mail-attachments-")
		if err != nil {
			return err
		}
		attachmentsDir = dir
		core.TheApp.AddQuitCleanFunc(func() { os.RemoveAll(dir) })
	}
	dir, err := os.MkdirTemp(attachmentsDir, "")
	if err != nil {
		return err
	}
	name := filepath.Base(at.Filename)
	if name == "." || name == "/" || name == "" {
		name = "attachment"
	}
	fname := filepath.Join(dir, name)
	if err := os.WriteFile(fname, at.Data, 0o600); err != nil {
		return err
	}
	at.openedFile = fname
	openURL(fsx.FileURL(fname))
	return nil
}

// SignatureButton represents a [SignatureText] with a [core.Button] that
// shows its first line, and opens a dialog to edit all of it.
type SignatureButton struct {
	core.Button
	Signature *SignatureText
}

func (sb *SignatureButton) WidgetValue() any { return &sb.Signature }

func (sb *SignatureButton) Init() {
	sb.Button.Init()
	sb.SetType(core.ButtonTonal).SetIcon(icons.Edit)
	sb.Styler(func(s *styles.Style) {
		s.Max.X.Ch(60) // not as wide as a long first line
	})
	sb.Updater(func() {
		first, _, _ := strings.Cut(strings.TrimSpace(string(*sb.Signature)), "\n")
		if first == "" {
			first = "(empty)"
		}
		if r := []rune(first); len(r) > 50 {
			first = string(r[:50]) + "…"
		}
		sb.SetText(first)
	})
	sb.OnClick(func(e events.Event) {
		d := core.NewBody("Signature")
		ed := textcore.NewEditor(d)
		ed.Lines.SetLanguage(fileinfo.Markdown)
		ed.Lines.Settings.LineNumbers = false
		ed.Lines.SetString(string(*sb.Signature))
		ed.Styler(func(s *styles.Style) {
			s.Min.Set(units.Ch(60), units.Em(10))
			s.Grow.Set(1, 1)
		})
		d.AddBottomBar(func(bar *core.Frame) {
			d.AddCancel(bar)
			d.AddOK(bar).OnClick(func(e events.Event) {
				*sb.Signature = SignatureText(ed.Lines.String())
				sb.Update()
				sb.SendChange()
			})
		})
		d.RunDialog(sb)
	})
}

// MessageSummaryView shows the summary of a message or conversation in
// two lines, for the compact list of messages: the sender and the date,
// and the subject, with the other labels if they are shown ([ViewLabels]).
type MessageSummaryView struct {
	core.Frame
	Summary *messageSummary
}

func (sv *MessageSummaryView) WidgetValue() any { return &sv.Summary }

func (sv *MessageSummaryView) Init() {
	sv.Frame.Init()
	sv.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 0)
		s.Gap.Zero()
		if sv.Summary.Dim {
			s.Opacity = 0.6
		}
	})
	tree.AddChild(sv, func(w *core.Frame) {
		w.Styler(func(s *styles.Style) {
			s.Grow.Set(1, 0)
		})
		tree.AddChild(w, func(w *core.Icon) {
			w.Styler(func(s *styles.Style) {
				s.Color = colors.Scheme.Warn.Base
			})
			w.Updater(func() {
				if sv.Summary.Starred {
					w.SetIcon(icons.StarFill)
				} else {
					w.SetIcon(icons.None)
				}
			})
		})
		tree.AddChild(w, func(w *core.Text) {
			w.Styler(func(s *styles.Style) {
				s.SetTextWrap(false)
				s.Grow.Set(1, 0)
				if sv.Summary.Unread {
					s.Font.Weight = rich.Bold
				}
			})
			w.Updater(func() { w.SetText(sv.Summary.From) })
		})
		tree.AddChild(w, func(w *core.Text) {
			w.SetType(core.TextBodySmall)
			w.Styler(func(s *styles.Style) {
				s.SetTextWrap(false)
			})
			w.Updater(func() { w.SetText(sv.Summary.Date) })
		})
	})
	tree.AddChild(sv, func(w *core.Text) {
		w.SetType(core.TextBodySmall)
		w.Styler(func(s *styles.Style) {
			s.SetTextWrap(false)
			s.Color = colors.Scheme.OnSurfaceVariant
			if sv.Summary.Unread {
				s.Font.Weight = rich.Bold
			}
		})
		w.Updater(func() {
			text := sv.Summary.Subject
			if sv.Summary.Labels != "" {
				text += "  ·  " + sv.Summary.Labels
			}
			if sv.Summary.Attachments != "" {
				text += "  ·  " + sv.Summary.Attachments
			}
			w.SetText(text)
		})
	})
}
