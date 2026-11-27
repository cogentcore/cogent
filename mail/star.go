// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"slices"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"github.com/cogentcore/go-imap/v2"
	"github.com/cogentcore/go-imap/v2/imapclient"
)

// Starring a message sets its standard IMAP \Flagged flag, which all of
// the mail programs show, as the star of Gmail and the flag of Apple Mail.
// The starred messages are shown in the Starred mailbox of the account
// (see [App.starredLabel]), by the flag, so that they are there as soon
// as they are starred.

// isStarred returns whether the given message is starred.
func (cm *CacheMessage) isStarred() bool {
	return slices.Contains(cm.Flags, imap.FlagFlagged)
}

// starredLabel returns the label of the mailbox of the starred messages
// of the given account: its mailbox of them, as [Gmail]/Starred, or else
// "Starred", a local one.
func (a *App) starredLabel(email string) string {
	if mb := a.specialUse[email][imap.MailboxAttrFlagged]; mb != "" {
		return mb
	}
	return "Starred"
}

// isStarredLabel returns whether the given label is that of the starred
// messages of the current account.
func (a *App) isStarredLabel(label string) bool {
	return label == a.starredLabel(a.currentEmail)
}

// Star stars the messages being read, or unstars them if they are all
// starred.
func (a *App) Star() { //types:add
	msgs := slices.DeleteFunc(a.readMessages(), (*CacheMessage).isLocal)
	if len(msgs) == 0 {
		return
	}
	star := slices.ContainsFunc(msgs, func(cm *CacheMessage) bool { return !cm.isStarred() })
	a.byAccount(msgs, func(email string, msgs []*CacheMessage, then ...func()) {
		a.setStarred(email, msgs, star)
	})
}

// setStarred stars or unstars the given messages of the given account
// (see [App.setFlagOn]).
func (a *App) setStarred(email string, msgs []*CacheMessage, star bool) {
	a.setFlagOn(email, msgs, imap.FlagFlagged, star)
}

// setFlagOn sets whether the given messages of the given account have the
// given flag, as \Seen or \Flagged: right away in the app, so that it shows
// as soon as they are clicked on, and then on the server, where a failure
// is shown, and the next sync gets their flags again.
func (a *App) setFlagOn(email string, msgs []*CacheMessage, flag imap.Flag, on bool) {
	var change []*CacheMessage
	for _, cm := range msgs {
		if !cm.isLocal() && slices.Contains(cm.Flags, flag) != on {
			change = append(change, cm)
			setFlag(cm, flag, on)
		}
	}
	if len(change) == 0 {
		return
	}
	a.updateMail()
	op := imap.StoreFlagsDel
	if on {
		op = imap.StoreFlagsAdd
	}
	a.actionOn(email, change, func(c *imapclient.Client, cm *CacheMessage) error {
		for _, label := range cm.Labels {
			if label.UID == 0 { // a Gmail label, whose message has the flags in its mailbox
				continue
			}
			if err := a.selectMailbox(c, email, label.Name); err != nil {
				return err
			}
			err := c.Store(label.UIDSet(), &imap.StoreFlags{Op: op, Silent: true,
				Flags: []imap.Flag{flag}}, nil).Close()
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// setFlag sets whether the given message has the given flag.
func setFlag(cm *CacheMessage, flag imap.Flag, on bool) {
	if slices.Contains(cm.Flags, flag) == on {
		return
	}
	if on {
		cm.Flags = append(slices.Clone(cm.Flags), flag)
	} else {
		cm.Flags = slices.DeleteFunc(slices.Clone(cm.Flags), func(f imap.Flag) bool { return f == flag })
	}
}

// starMark is whether a row of the list of messages is starred, shown as
// a star (see [StarIcon]).
type starMark bool

// StarIcon shows a [starMark]: a filled star if it is starred, or nothing.
type StarIcon struct {
	core.Icon
	Starred starMark
}

func (si *StarIcon) WidgetValue() any { return &si.Starred }

func (si *StarIcon) Init() {
	si.Icon.Init()
	si.Styler(func(s *styles.Style) {
		s.Color = colors.Scheme.Warn.Base
	})
	si.Updater(func() {
		if si.Starred {
			si.SetIcon(icons.StarFill)
		} else {
			si.SetIcon(icons.Blank)
		}
	})
}
