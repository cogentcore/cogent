// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"fmt"
	"net/mail"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/tree"
	"github.com/emersion/go-imap/v2"
	"github.com/mitchellh/go-homedir"
)

func init() {
	core.AddValueType[CacheMessage, MessageListItem]()
	core.AddValueType[mail.Address, AddressTextField]()
	core.AddValueType[Attachment, AttachmentButton]()
}

// MessageListItem represents a [CacheMessage] with a [core.Frame] for the message list.
type MessageListItem struct {
	core.Frame
	Message *CacheMessage
}

func (mi *MessageListItem) WidgetValue() any { return &mi.Message }

func (mi *MessageListItem) Init() {
	mi.Frame.Init()
	mi.Styler(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Activatable, abilities.Hoverable)
		s.Cursor = cursors.Pointer
		s.Direction = styles.Column
		s.Grow.Set(1, 0)
	})
	mi.OnClick(func(e events.Event) {
		theApp.readMessage = mi.Message
		theApp.MarkAsRead()
		theApp.Update()
	})

	tree.AddChild(mi, func(w *core.Text) {
		w.SetType(core.TextTitleMedium)
		w.Styler(func(s *styles.Style) {
			s.SetNonSelectable()
			s.SetTextWrap(false)
		})
		w.Updater(func() {
			text := ""
			if !slices.Contains(mi.Message.Flags, imap.FlagSeen) {
				text = fmt.Sprintf(`<span color="%s">•</span> `, colors.AsHex(colors.ToUniform(colors.Scheme.Primary.Base)))
			}
			for _, f := range mi.Message.From {
				if f.Name != "" {
					text += f.Name + " "
				} else {
					text += f.Addr() + " "
				}
			}
			w.SetText(text)
		})
	})
	tree.AddChild(mi, func(w *core.Text) {
		w.SetType(core.TextBodyMedium)
		w.Styler(func(s *styles.Style) {
			s.SetNonSelectable()
			s.SetTextWrap(false)
		})
		w.Updater(func() {
			w.SetText(mi.Message.Subject)
		})
	})
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
