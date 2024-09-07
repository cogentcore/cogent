// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"fmt"
	"net/mail"

	"cogentcore.org/core/core"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/tree"
)

func init() {
	core.AddValueType[CacheData, MessageListItem]()
	core.AddValueType[mail.Address, AddressTextField]()
}

// MessageListItem represents a [CacheData] with a [core.Frame] for the message list.
type MessageListItem struct {
	core.Frame
	Data *CacheData
}

func (mi *MessageListItem) WidgetValue() any { return &mi.Data }

func (mi *MessageListItem) Init() {
	mi.Frame.Init()
	mi.Styler(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Activatable, abilities.Hoverable)
		s.Cursor = cursors.Pointer
		s.Direction = styles.Column
		s.Grow.Set(1, 0)
	})
	mi.OnClick(func(e events.Event) {
		theApp.readMessage = mi.Data
		theApp.Update()
	})
	mi.AddContextMenu(func(m *core.Scene) {
		theApp.readMessage = mi.Data
		core.NewFuncButton(m).SetFunc(theApp.MoveMessage).SetIcon(icons.Move).SetText("Move")
	})

	tree.AddChild(mi, func(w *core.Text) {
		w.SetType(core.TextTitleMedium)
		w.Styler(func(s *styles.Style) {
			s.SetNonSelectable()
		})
		w.Updater(func() {
			text := ""
			for _, f := range mi.Data.From {
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
		})
		w.Updater(func() {
			w.SetText(mi.Data.Subject)
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
}
