// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/text/rich"
	"github.com/emersion/go-imap/v2"
)

// messageRow is a row of the list of messages, in a [core.TreeTable]: a
// message, or a conversation with its messages as children.
type messageRow struct {
	// From is the sender of the message, or the senders of the
	// conversation, with the number of messages.
	From string `edit:"-"`

	// Subject is the subject of the message or conversation.
	Subject string `edit:"-"`

	// Date is the date of the message, or of the latest message of the
	// conversation, in a format that sorts in order.
	Date string `edit:"-"`

	// Children are the messages of a conversation, in a conversation row.
	Children []*messageRow `display:"-"`

	// Closed is whether a conversation row is closed.
	Closed bool `display:"-" treetable:"closed"`

	// message is the message of a message row, or the start of the
	// conversation of a conversation row.
	message *CacheMessage

	// conversation is whether this is a conversation row.
	conversation bool
}

// listDateFormat is the format of the dates of the list of messages,
// which sorts in order.
const listDateFormat = "2006-01-02 15:04"

// isRead returns whether the message or conversation of the row is read.
func (mr *messageRow) isRead() bool {
	if mr.conversation {
		return mr.message.isRead()
	}
	return slices.Contains(mr.message.Flags, imap.FlagSeen)
}

// senderName returns the name of the sender of the given message,
// or "You" for the given account.
func senderName(cm *CacheMessage, email string) string {
	if len(cm.From) == 0 {
		return ""
	}
	f := cm.From[0]
	if f.Addr() == email {
		return "You"
	}
	if f.Name != "" {
		return f.Name
	}
	return f.Addr()
}

// updateConversations sets the replies of the messages of the current
// account, and returns the starts of the conversations.
func (a *App) updateConversations() []*CacheMessage {
	mp := a.cache[a.currentEmail]
	for _, cm := range mp {
		cm.replies = nil
	}
	var starts []*CacheMessage
	for _, cm := range mp {
		start := a.conversationStart(mp, cm)
		if start == cm {
			starts = append(starts, cm)
		} else {
			start.replies = append(start.replies, cm)
		}
	}
	return starts
}

// hasLabel returns whether the given message has the label being shown.
func (a *App) hasLabel(cm *CacheMessage) bool {
	return slices.ContainsFunc(cm.Labels, func(l Label) bool { return l.Name == a.showLabel })
}

// updateList updates the rows of the list of messages for the current
// account and label: conversations of messages, unless
// [SettingsData.FlatList], with any message of the conversation having
// the label, or the messages with the label. It reuses the rows of the
// same messages, so that the selection and closed conversations stay.
func (a *App) updateList() {
	a.listCache = nil
	a.unreadMessages = 0
	a.shownMessages = 0
	starts := a.updateConversations()
	oldMessages, oldConversations := a.messageRows, a.conversationRows
	a.messageRows = map[*CacheMessage]*messageRow{}
	if !Settings.FlatList { // else they are kept for when it is not
		a.conversationRows = map[*CacheMessage]*messageRow{}
	}
	email := a.currentEmail

	rowOf := func(cm *CacheMessage) *messageRow {
		mr := oldMessages[cm]
		if mr == nil {
			mr = &messageRow{message: cm}
		}
		mr.From = senderName(cm, email)
		mr.Subject = cm.Subject
		mr.Date = cm.Date.Local().Format(listDateFormat)
		a.messageRows[cm] = mr
		return mr
	}

	rows := a.listRows[:0]
	for _, start := range starts {
		if Settings.FlatList {
			for _, cm := range append([]*CacheMessage{start}, start.replies...) {
				if a.hasLabel(cm) {
					a.listCache = append(a.listCache, cm)
					rows = append(rows, rowOf(cm))
				}
			}
			continue
		}
		if !a.hasLabel(start) && !slices.ContainsFunc(start.replies, a.hasLabel) {
			continue
		}
		a.listCache = append(a.listCache, start)
		if len(start.replies) == 0 {
			rows = append(rows, rowOf(start))
			continue
		}
		cr := oldConversations[start]
		if cr == nil {
			cr = &messageRow{message: start, conversation: true, Closed: true}
		}
		a.conversationRows[start] = cr
		msgs := append([]*CacheMessage{start}, start.replies...)
		slices.SortFunc(msgs, func(a, b *CacheMessage) int {
			return cmp.Compare(b.Date.UnixNano(), a.Date.UnixNano())
		})
		cr.Children = cr.Children[:0]
		for _, cm := range msgs {
			cr.Children = append(cr.Children, rowOf(cm))
		}
		// the senders in the order they joined the conversation
		var senders []string
		for _, cm := range slices.Backward(msgs) {
			if s := senderName(cm, email); !slices.Contains(senders, s) {
				senders = append(senders, s)
			}
		}
		cr.From = fmt.Sprintf("%s (%d)", strings.Join(senders, ", "), len(msgs))
		cr.Subject = start.Subject
		cr.Date = start.latestDate().Local().Format(listDateFormat)
		rows = append(rows, cr)
	}
	for _, cm := range a.cache[email] {
		if a.hasLabel(cm) {
			a.shownMessages++
		}
	}
	for _, cm := range a.listCache {
		if Settings.FlatList && !slices.Contains(cm.Flags, imap.FlagSeen) || !Settings.FlatList && !cm.isRead() {
			a.unreadMessages++
		}
	}
	slices.SortFunc(rows, func(a, b *messageRow) int {
		return cmp.Compare(b.Date, a.Date)
	})
	a.listRows = rows
}

// initList initializes the tree table of the list of messages.
func (a *App) initList(w *core.TreeTable) {
	w.SetReadOnly(true)
	w.SetItems(&a.listRows)
	w.SetSortFieldName("Date:down")
	w.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
	})
	w.TableStyler = func(w core.Widget, s *styles.Style, row, col int) {
		mr := a.rowAt(row)
		if mr == nil {
			return
		}
		if !mr.isRead() {
			s.Font.Weight = rich.Bold
			if col == 0 {
				s.Color = colors.Scheme.Primary.Base
			}
		}
	}
	w.OnSelect(func(e events.Event) {
		mr := a.rowAt(w.SelectedIndex)
		if mr == nil {
			return
		}
		a.readMessage = mr.message
		// a message row is shown on its own, a conversation with all of its messages
		a.readSingle = !mr.conversation
		a.MarkAsRead()
		a.Update()
	})
	a.list = w
}

// rowAt returns the visible row of the list of messages with the given
// index, or nil if there is none.
func (a *App) rowAt(idx int) *messageRow {
	if a.list == nil || a.list.Slice == nil {
		return nil
	}
	rows := *a.list.Slice.(*[]*messageRow)
	if idx < 0 || idx >= len(rows) {
		return nil
	}
	return rows[idx]
}
