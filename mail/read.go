// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"cmp"
	"io"
	"os"
	"path/filepath"
	"slices"
	"time"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/htmlcore"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"github.com/emersion/go-message/mail"
)

// ReadMessage represents the data necessary to display a message
// for the user to read.
type ReadMessage struct {
	From    []*mail.Address `view:"inline"`
	To      []*mail.Address `view:"inline"`
	Subject string
	Date    time.Time
}

// UpdateMessageList updates the message list from [App.Cache].
func (a *App) UpdateMessageList() {
	cached := a.Cache[a.CurrentEmail][a.CurrentMailbox]

	a.AsyncLock()
	defer a.AsyncUnlock()

	list := a.FindPath("splits/list").(*core.Frame)

	if list.NumChildren() > 100 {
		return
	}

	list.DeleteChildren()

	slices.SortFunc(cached, func(a, b *CacheData) int {
		return cmp.Compare(b.Date.UnixNano(), a.Date.UnixNano())
	})

	for i, cd := range cached {
		cd := cd

		if i > 100 {
			break
		}

		fr := core.NewFrame(list).Styler(func(s *styles.Style) {
			s.Direction = styles.Column
		})

		fr.Styler(func(s *styles.Style) {
			s.SetAbilities(true, abilities.Activatable, abilities.Hoverable)
			s.Cursor = cursors.Pointer
		})
		fr.OnClick(func(e events.Event) {
			a.ReadMessage = cd
			errors.Log(a.UpdateReadMessage())
		})
		fr.AddContextMenu(func(m *core.Scene) {
			a.ReadMessage = cd
			core.NewFuncButton(m, a.MoveMessage).SetIcon(icons.Move).SetText("Move")
		})

		ftxt := ""
		for _, f := range cd.From {
			if f.Name != "" {
				ftxt += f.Name + " "
			} else {
				ftxt += f.Addr() + " "
			}
		}

		core.NewText(fr).SetType(core.TextTitleMedium).SetText(ftxt).
			Styler(func(s *styles.Style) {
				s.SetNonSelectable()
			}).
			SetName("from")
		core.NewText(fr).SetType(core.TextBodyMedium).SetText(cd.Subject).
			Styler(func(s *styles.Style) {
				s.SetNonSelectable()
			}).
			SetName("subject")
	}

	list.Update()
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
