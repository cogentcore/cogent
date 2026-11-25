// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"cmp"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"
	"time"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/tree"
	"github.com/emersion/go-imap/v2"
)

// messageRow is a row of the list of messages, in a [core.TreeTable]: a
// message, or a conversation with its messages as children.
type messageRow struct {
	// Message is the summary of the message or conversation in two
	// lines, the only column in the compact list.
	Message messageSummary

	// From is the sender of the message, or the senders of the
	// conversation, with the number of messages.
	From string `edit:"-"`

	// Subject is the subject of the message or conversation.
	Subject string `edit:"-"`

	// Date is the date of the message, or of the latest message of the
	// conversation, in a format that sorts in order.
	Date string `edit:"-"`

	// Labels are the other labels of the message or conversation
	// (see [otherLabels]), an optional column.
	Labels string `edit:"-"`

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

// messageSummary is the summary of a message or conversation in two
// lines, for the compact list (see [MessageSummaryView]).
type messageSummary struct {
	From, Date, Subject, Labels string

	// Unread is whether the message or conversation is unread,
	// which is shown in bold.
	Unread bool
}

// compactDate returns the given date in the short format of the compact
// list: the time today, the day this week, or else the date.
func compactDate(t time.Time) string {
	t = t.Local()
	now := time.Now()
	switch {
	case t.Year() == now.Year() && t.YearDay() == now.YearDay():
		return t.Format("3:04 PM")
	case now.Sub(t) < 6*24*time.Hour:
		return t.Format("Monday")
	case t.Year() == now.Year():
		return t.Format("Jan 2")
	}
	return t.Format("1/2/06")
}

// friendlyLabels returns the friendly names of the given labels, as text.
func friendlyLabels(labels []string) string {
	var res []string
	for _, l := range labels {
		res = append(res, friendlyLabelName(l))
	}
	return strings.Join(res, ", ")
}

// setSummary sets the Message summary of the row from its other fields.
func (mr *messageRow) setSummary(date time.Time) {
	mr.Message = messageSummary{From: mr.From, Date: compactDate(date), Subject: mr.Subject, Labels: mr.Labels,
		Unread: !mr.isRead()}
}

// compactList returns whether the list of messages is compact, as set in
// the settings, and always on mobile, with its narrow screen.
func compactList() bool {
	return Settings.CompactList || core.TheApp.SystemPlatform().IsMobile()
}

// listField returns whether to show the given field of the rows in the
// list of messages: only the Message summary in the compact list, and the
// Labels only if [SettingsData.LabelsColumn].
func listField(field reflect.StructField) bool {
	if compactList() {
		return field.Name == "Message"
	}
	switch field.Name {
	case "Message":
		return false
	case "Labels":
		return Settings.LabelsColumn
	}
	return true
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
		mr.Labels = friendlyLabels(otherLabels(cm))
		mr.setSummary(cm.Date)
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
		// the other labels of its messages, but for those of the sent
		// messages, which are in the conversations of other mailboxes
		var labels []string
		for _, cm := range msgs {
			for _, l := range otherLabels(cm) {
				if !slices.Contains(labels, l) && labelRank(l) != 2 {
					labels = append(labels, l)
				}
			}
		}
		cr.Labels = friendlyLabels(sortedLabels(labels))
		cr.setSummary(start.latestDate())
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
	w.ReadOnlyKeyNav = true
	w.ShowField = listField // before the items
	w.FitWidth = compactList()
	w.ContextMenus = nil // only that of the messages
	w.AddContextMenu(a.messageMenu)
	w.SetItems(&a.listRows)
	compact := compactList()
	if !compact {
		w.ColumnWidths = Settings.ColumnWidths
		sort := Settings.Sort
		if sort == "" {
			sort = "Date:down"
		}
		w.SetSortFieldName(sort)
	}
	// the column widths and sort are changed by the user
	w.OnChange(func(e events.Event) {
		if compact { // newest first, fitting the width
			return
		}
		Settings.ColumnWidths = w.ColumnWidths
		Settings.Sort = w.SortFieldName()
		a.saveSettings()
	})
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
		a.selectedRows = a.rowsSelected()
		mr := a.rowAt(w.SelectedIndex)
		if mr == nil {
			return
		}
		// The actions on messages, such as filing, apply to all of the
		// selected rows, and those on one message, such as replying,
		// to the last one selected.
		a.readMessage = mr.message
		// a message row is shown on its own, a conversation with all of its messages
		a.readSingle = !mr.conversation
		if len(a.selectedRows) <= 1 { // only when reading it
			a.MarkAsRead()
		}
		a.Update()
	})
	a.list = w
}

// rowsSelected returns the selected rows of the list of messages, in order.
func (a *App) rowsSelected() []*messageRow {
	if a.list == nil {
		return nil
	}
	idxs := slices.Sorted(maps.Keys(a.list.SelectedIndexes))
	var res []*messageRow
	for _, i := range idxs {
		if mr := a.rowAt(i); mr != nil {
			res = append(res, mr)
		}
	}
	return res
}

// rowMessages returns the messages of the given row: its message, with
// the others of its conversation for a conversation row.
func rowMessages(mr *messageRow) []*CacheMessage {
	if mr.conversation {
		return append([]*CacheMessage{mr.message}, mr.message.replies...)
	}
	return []*CacheMessage{mr.message}
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

// selectNext returns a function that selects the row of the list of
// messages that is at the index of the one selected now, which is the next
// one after it once the selected one is gone, as after filing or deleting
// it, reading it, or nothing if there are no rows.
func (a *App) selectNext() func() {
	idx := -1
	if a.list != nil {
		idx = a.list.SelectedIndex
		if len(a.list.SelectedIndexes) > 1 { // the first of them
			idx = slices.Min(slices.Collect(maps.Keys(a.list.SelectedIndexes)))
		}
	}
	return func() {
		if a.list == nil || idx < 0 {
			return
		}
		n := len(*a.list.Slice.(*[]*messageRow))
		if n == 0 {
			a.readMessage = nil
			a.selectedRows = nil
			a.Update()
			return
		}
		a.list.SelectIndexEvent(min(idx, n-1), events.SelectOne)
	}
}

// makeListPanel makes the panel of the list of messages: the name of the
// mailbox, the numbers of messages, the switches of the view of the list,
// and the list, which is made anew when the view changes.
func (a *App) makeListPanel(p *tree.Plan) {
	tree.Add(p, func(w *core.Text) {
		w.SetType(core.TextTitleMedium)
		w.Updater(func() {
			w.SetText(friendlyLabelName(a.showLabel))
		})
	})
	tree.Add(p, func(w *core.Text) {
		w.Updater(func() {
			w.Text = ""
			total := a.totalMessages[a.currentEmail][a.showLabel]
			if a.shownMessages < total { // still syncing
				w.Text += fmt.Sprintf("%d of ", a.shownMessages)
			}
			w.Text += fmt.Sprintf("%d messages", total)
			if !Settings.FlatList {
				w.Text += fmt.Sprintf(" in %d conversations", len(a.listCache))
			}
			if a.unreadMessages > 0 {
				w.Text += fmt.Sprintf(", %d unread", a.unreadMessages)
			}
		})
	})
	tree.Add(p, func(w *core.Separator) {})
	tree.Add(p, func(w *core.Frame) {
		w.Styler(func(s *styles.Style) {
			s.Wrap = true
		})
		w.Maker(func(p *tree.Plan) {
			add := func(name, tooltip string, get func() bool, set func(on bool)) {
				tree.AddAt(p, name, func(w *core.Switch) {
					w.SetText(name).SetTooltip(tooltip)
					w.Updater(func() {
						w.SetChecked(get())
					})
					w.OnChange(func(e events.Event) {
						set(w.IsChecked())
						a.saveSettings()
						a.Update()
					})
				})
			}
			add("Conversations", "Group the messages into conversations",
				func() bool { return !Settings.FlatList }, func(on bool) { Settings.FlatList = !on })
			if !core.TheApp.SystemPlatform().IsMobile() {
				add("Compact", "Show each message in two lines, in one column",
					func() bool { return Settings.CompactList }, func(on bool) { Settings.CompactList = on })
			}
			add("Labels", "Show the other labels of the messages, in a column or line",
				func() bool { return Settings.LabelsColumn }, func(on bool) { Settings.LabelsColumn = on })
		})
	})
	// a new list for the fields shown, which are set before the items
	tree.AddAt(p, fmt.Sprintf("list-%v-%v", compactList(), Settings.LabelsColumn), a.initList)
}
