// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"fmt"
	"net/mail"
	"os"
	"path/filepath"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"github.com/mitchellh/go-homedir"
)

func init() {
	core.AddValueType[mail.Address, AddressTextField]()
	core.AddValueType[Attachment, AttachmentButton]()
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
	ab.SetIcon(icons.Download).SetType(core.ButtonTonal)
	ab.Updater(func() {
		ab.SetText(ab.Attachment.Filename)
	})
	ab.OnClick(func(e events.Event) {
		fb := core.NewSoloFuncButton(ab).SetFunc(func(filename core.Filename) error {
			return os.WriteFile(string(filename), ab.Attachment.Data, 0666)
		})
		fb.Args[0].Value = core.Filename(filepath.Join(errors.Log1(homedir.Dir()), "Downloads", ab.Attachment.Filename))
		fb.CallFunc()
	})
}
