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
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/textcore"
	"github.com/mitchellh/go-homedir"
)

func init() {
	core.AddValueType[mail.Address, AddressTextField]()
	core.AddValueType[Attachment, AttachmentButton]()
	core.AddValueType[SignatureText, SignatureButton]()
}

// AddressTextField represents a [mail.Address] with a [core.TextField].
type AddressTextField struct {
	core.TextField
	Address mail.Address
}

func (at *AddressTextField) WidgetValue() any { return &at.Address }

func (at *AddressTextField) Init() {
	at.TextField.Init()
	at.Updater(func() {
		if at.IsReadOnly() && at.Address.Name != "" && at.Address.Name != at.Address.Address {
			at.SetText(fmt.Sprintf("%s (%s)", at.Address.Name, at.Address.Address))
			return
		}
		at.SetText(at.Address.Address)
	})
	at.SetValidator(func() error {
		text := at.Text()
		if !strings.Contains(text, "@") && !strings.Contains(text, ".") {
			return fmt.Errorf("invalid email address")
		}
		at.Address.Address = text
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
	sb.Updater(func() {
		first, _, _ := strings.Cut(strings.TrimSpace(string(*sb.Signature)), "\n")
		if first == "" {
			first = "(empty)"
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
