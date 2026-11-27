// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"cmp"
	"fmt"
	"image"
	"log/slog"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/tree"
	"github.com/cogentcore/go-imap/v2"
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

	// Attachments is the number of attachments of the message, or of the
	// messages of the conversation, an optional column, empty for none.
	Attachments string `edit:"-"`

	// Starred is whether the message, or a message of the conversation,
	// is starred, an optional column, the last one, where it stands out.
	Starred starMark `edit:"-"`

	// Children are the messages of a conversation, in a conversation row.
	Children []*messageRow `display:"-"`

	// Closed is whether a conversation row is closed.
	Closed bool `display:"-" treetable:"closed"`

	// message is the message of a message row, or the start of the
	// conversation of a conversation row.
	message *CacheMessage

	// conversation is whether this is a conversation row.
	conversation bool

	// elsewhere is whether this is a message of a conversation that is
	// not in the mailbox being shown, which is dimmed.
	elsewhere bool
}

// messageSummary is the summary of a message or conversation in two
// lines, for the compact list (see [MessageSummaryView]).
type messageSummary struct {
	From, Date, Subject, Labels string

	// Attachments is the number of attachments, as text, if shown,
	// or "" for none.
	Attachments string

	// Starred is whether it is starred, if the stars are shown.
	Starred bool

	// Unread is whether the message or conversation is unread,
	// which is shown in bold.
	Unread bool

	// Dim is whether the message is not in the mailbox being shown,
	// in a conversation, which is dimmed.
	Dim bool
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
// The labels and attachments are only in it if they are shown.
func (mr *messageRow) setSummary(date time.Time, labels, attachments, stars bool) {
	mr.Message = messageSummary{From: mr.From, Date: compactDate(date), Subject: mr.Subject, Unread: !mr.isRead(),
		Starred: stars && bool(mr.Starred)}
	if labels {
		mr.Message.Labels = mr.Labels
	}
	if attachments && mr.Attachments != "" {
		mr.Message.Attachments = mr.Attachments + " attachment"
		if mr.Attachments != "1" {
			mr.Message.Attachments += "s"
		}
	}
}

// attachmentsText returns the given number of attachments as text,
// or "" for none.
func attachmentsText(n int) string {
	if n == 0 {
		return ""
	}
	return strconv.Itoa(n)
}

// listViews returns the options of the view of the list of messages.
func (a *App) listViews() ListViews {
	return Settings.ListViews
}

// flatList returns whether the list of messages shows each message on its
// own, instead of grouped into conversations.
func (a *App) flatList() bool {
	v := a.listViews()
	return !v.HasFlag(ViewConversations) || messagesOnly(a.showLabel)
}

// messagesOnly returns whether the mailbox with the given label shows each
// of its messages on its own, not their conversations, as those of the
// Drafts, Sent, and Trash, whose own messages are what is wanted there.
func messagesOnly(label string) bool {
	switch friendlyLabelName(label) {
	case "Drafts", "Sent", "Trash", outboxLabel:
		return true
	}
	return false
}

// compactList returns whether the list of messages is compact, as set in
// its view, and always on mobile, with its narrow screen.
func (a *App) compactList() bool {
	v := a.listViews()
	return v.HasFlag(ViewCompact) || core.TheApp.SystemPlatform().IsMobile()
}

// labelsShown returns whether the list of messages shows their other labels.
func (a *App) labelsShown() bool {
	v := a.listViews()
	return v.HasFlag(ViewLabels)
}

// starsShown returns whether the list of messages shows their stars.
func (a *App) starsShown() bool {
	v := a.listViews()
	return v.HasFlag(ViewStars)
}

// attachmentsShown returns whether the list of messages shows the
// numbers of their attachments.
func (a *App) attachmentsShown() bool {
	v := a.listViews()
	return v.HasFlag(ViewAttachments)
}

// listField returns whether to show the given field of the rows in the
// list of messages: only the Message summary in the compact list, and the
// Labels only if [ViewLabels].
func (a *App) listField(field reflect.StructField) bool {
	if a.compactList() {
		return field.Name == "Message"
	}
	switch field.Name {
	case "Message":
		return false
	case "Labels":
		return a.labelsShown()
	case "Attachments":
		return a.attachmentsShown()
	case "Starred":
		return a.starsShown()
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

// updateConversations sets the replies of the messages of the accounts
// shown (see [App.listAccounts]), and returns the starts of the
// conversations, and the number of messages with the label being shown
// (see [App.accountConversations]).
func (a *App) updateConversations() (starts []*CacheMessage, shown int) {
	for _, email := range a.listAccounts() {
		s, n := a.accountConversations(email)
		starts = append(starts, s...)
		shown += n
	}
	return starts, shown
}

// accountConversations sets the replies of the messages of the given
// account, and returns the starts of the conversations, and the number of
// messages with the label being shown. On Gmail, the conversations are its
// threads (by X-GM-THRID), whose first message is the start, and only those
// with a message with the label being shown are made, as there can be very
// many messages; elsewhere, a conversation starts with the message that the
// others reply to (by [CacheMessage.InReplyTo]).
func (a *App) accountConversations(email string) (starts []*CacheMessage, shown int) {
	mp := a.cache[email]
	want := map[uint64]bool{} // the threads with the label, on Gmail
	for _, cm := range mp {
		cm.account = email
		cm.replies = nil
		if a.hasLabel(cm) {
			shown++
			want[cm.GmailThreadID] = true
		}
	}
	if a.gmail[email] {
		threads := map[uint64][]*CacheMessage{}
		for _, cm := range mp {
			if !want[cm.GmailThreadID] {
				continue
			}
			if cm.GmailThreadID == 0 { // a local message on its own
				starts = append(starts, cm)
				continue
			}
			threads[cm.GmailThreadID] = append(threads[cm.GmailThreadID], cm)
		}
		for _, msgs := range threads {
			slices.SortFunc(msgs, func(x, y *CacheMessage) int { return cmp.Compare(x.Date.UnixNano(), y.Date.UnixNano()) })
			start := msgs[0]
			start.replies = msgs[1:]
			starts = append(starts, start)
		}
		return starts, shown
	}
	byID := map[string]*CacheMessage{}
	for _, cm := range mp {
		if cm.MessageID != "" {
			byID[cm.MessageID] = cm
		}
	}
	for _, cm := range mp {
		start := conversationStart(byID, cm)
		if start == cm {
			starts = append(starts, cm)
		} else {
			start.replies = append(start.replies, cm)
		}
	}
	return starts, shown
}

// labelsOf returns labels with the given names, without UIDs.
func labelsOf(names []string) []Label {
	res := make([]Label, len(names))
	for i, n := range names {
		res[i] = Label{Name: n}
	}
	return res
}

// hasLabel returns whether the given message has the label being shown,
// with the labels it is getting, if any (see [App.pendingLabels]).
func (a *App) hasLabel(cm *CacheMessage) bool {
	if a.isStarredLabel(a.showLabel) { // by the flag, as soon as it is starred
		return cm.isStarred()
	}
	if a.showLabel == untaggedLabel { // by the labels, also those it is getting
		if labels, ok := a.pendingLabels[cm]; ok {
			return a.isUntagged(a.accountOf(cm), &CacheMessage{Labels: labelsOf(labels), key: cm.key})
		}
		return a.isUntagged(a.accountOf(cm), cm)
	}
	if labels, ok := a.pendingLabels[cm]; ok {
		return slices.Contains(labels, a.showLabel)
	}
	return slices.ContainsFunc(cm.Labels, func(l Label) bool { return l.Name == a.showLabel })
}

// updateList updates the rows of the list of messages for the current
// account and label: conversations of messages, unless
// [App.flatList], with any message of the conversation having
// the label, or the messages with the label. It reuses the rows of the
// same messages, so that the selection and closed conversations stay.
func (a *App) updateList() {
	start := time.Now()
	defer func() {
		if d := time.Since(start); d > 50*time.Millisecond {
			slog.Debug("mail: updating the list of messages took a while", "took", d,
				"accounts", a.listAccounts(), "rows", len(a.listRows))
		}
	}()
	a.listCache = nil
	a.unreadMessages = 0
	a.shownMessages = 0
	starts, shown := a.updateConversations()
	oldMessages, oldConversations := a.messageRows, a.conversationRows
	a.messageRows = map[*CacheMessage]*messageRow{}
	flat := a.flatList()
	if !flat { // else they are kept for when it is not
		a.conversationRows = map[*CacheMessage]*messageRow{}
	}
	showLabels, showAttachments, showStars := a.labelsShown(), a.attachmentsShown(), a.starsShown()

	rowOf := func(cm *CacheMessage) *messageRow {
		mr := oldMessages[cm]
		if mr == nil {
			mr = &messageRow{message: cm}
		}
		mr.elsewhere = false
		mr.From = senderName(cm, cm.account)
		mr.Subject = cm.Subject
		mr.Date = cm.Date.Local().Format(listDateFormat)
		mr.Labels = friendlyLabels(otherLabels(cm))
		mr.Attachments = attachmentsText(cm.Attachments)
		mr.Starred = starMark(cm.isStarred())
		mr.setSummary(cm.Date, showLabels, showAttachments, showStars)
		a.messageRows[cm] = mr
		return mr
	}

	rows := a.listRows[:0]
	if a.searchResults != nil { // each message found, on its own
		for _, email := range a.listAccounts() {
			for _, cm := range a.cache[email] {
				cm.account = email
				if !a.searchResults[searchKey(email, cm.key)] || !a.unreadShown(cm, false) {
					continue
				}
				a.listCache = append(a.listCache, cm)
				rows = append(rows, rowOf(cm))
				if !slices.Contains(cm.Flags, imap.FlagSeen) {
					a.unreadMessages++
				}
			}
		}
		a.shownMessages = len(a.listCache)
		slices.SortFunc(rows, newestFirst)
		a.listRows = rows
		return
	}
	for _, start := range starts {
		if flat {
			for _, cm := range append([]*CacheMessage{start}, start.replies...) {
				if a.hasLabel(cm) && a.unreadShown(cm, false) {
					a.listCache = append(a.listCache, cm)
					rows = append(rows, rowOf(cm))
				}
			}
			continue
		}
		if !a.hasLabel(start) && !slices.ContainsFunc(start.replies, a.hasLabel) || !a.unreadShown(start, true) {
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
			mr := rowOf(cm)
			mr.elsewhere = !a.hasLabel(cm)
			mr.Message.Dim = mr.elsewhere
			cr.Children = append(cr.Children, mr)
		}
		// the senders in the order they joined the conversation
		var senders []string
		for _, cm := range slices.Backward(msgs) {
			if s := senderName(cm, cm.account); !slices.Contains(senders, s) {
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
				if !slices.Contains(labels, l) && friendlyLabelName(l) != "Sent" {
					labels = append(labels, l)
				}
			}
		}
		cr.Labels = friendlyLabels(sortedLabels(labels))
		n := 0
		for _, cm := range msgs {
			n += cm.Attachments
		}
		cr.Attachments = attachmentsText(n)
		cr.Starred = starMark(slices.ContainsFunc(msgs, (*CacheMessage).isStarred))
		cr.setSummary(start.latestDate(), showLabels, showAttachments, showStars)
		rows = append(rows, cr)
	}
	a.shownMessages = shown
	for _, cm := range a.listCache {
		if flat && !slices.Contains(cm.Flags, imap.FlagSeen) || !flat && !cm.isRead() {
			a.unreadMessages++
		}
	}
	slices.SortFunc(rows, newestFirst)
	a.listRows = rows
}

// unreadShown returns whether the list of messages shows the given message
// (or conversation, of which it is the start) for [App.unreadOnly]: all of
// them if it is off, and otherwise those that are unread (conversations
// with any unread message), and those being read or selected, which stay
// shown until another one is, rather than leaving the list when they are
// read, as when they are clicked.
func (a *App) unreadShown(cm *CacheMessage, conversation bool) bool {
	if !a.unreadOnly || cm == a.readMessage {
		return true
	}
	if slices.ContainsFunc(a.selectedRows, func(mr *messageRow) bool { return mr.message == cm }) {
		return true
	}
	if conversation {
		return !cm.isRead()
	}
	return !slices.Contains(cm.Flags, imap.FlagSeen)
}

// UnreadOnly sets whether the list of messages shows only the unread
// messages (see [App.unreadOnly]).
func (a *App) UnreadOnly(on bool) {
	a.unreadOnly = on
	a.Update()
}

// newestFirst compares the given rows of the list of messages for their
// order, the newest first: by their whole times (those of the latest
// messages of conversations), then their keys, so that rows of the same
// minute, as shown, stay in the same order each time the list is updated,
// instead of swapping (the sort of the list is stable), as the selection
// of the rows did, with an unstable sort by the minutes.
func newestFirst(x, y *messageRow) int {
	tx, ty := x.message.Date, y.message.Date
	if x.conversation {
		tx = x.message.latestDate()
	}
	if y.conversation {
		ty = y.message.latestDate()
	}
	return cmp.Or(ty.Compare(tx), cmp.Compare(x.message.key, y.message.key))
}

// initList initializes the tree table of the list of messages.
func (a *App) initList(w *core.TreeTable) {
	w.SetReadOnly(true)
	w.ReadOnlyKeyNav = true
	w.ReadOnlyMultiSelect = true                                                                  // several selected rows are shown as selected
	w.ShowField = a.listField                                                                     // before the items
	w.HeaderIcons = map[string]icons.Icon{"Attachments": icons.AttachFile, "Starred": icons.Star} // narrow columns
	w.FitWidth = a.compactList()
	w.ContextMenus = nil // only that of the messages
	w.AddContextMenu(a.messageMenu)
	w.SetItems(&a.listRows)
	compact := a.compactList()
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
		if mr.elsewhere { // in another mailbox: see its labels
			s.Color = colors.Scheme.OnSurfaceVariant
			s.Opacity = 0.6
		}
		if !mr.isRead() {
			s.Font.Weight = rich.Bold
			if col == 0 {
				s.Color = colors.Scheme.Primary.Base
			}
		}
	}
	// the clicks and selections, in the debug log, as for selecting several
	w.OnFirst(events.Click, func(e events.Event) {
		slog.Debug("mail: click on the list", "mods", e.Modifiers().String(), "mode", e.SelectMode().String(), "selected", len(w.SelectedIndexes))
	})
	w.OnSelect(func(e events.Event) {
		a.selectedRows = a.rowsSelected()
		slog.Debug("mail: selected in the list", "rows", len(a.selectedRows), "index", w.SelectedIndex)
		if len(a.selectedRows) > 1 { // the first selected is read, not marked read
			a.readRow(a.selectedRows[0], false)
			return
		}
		if mr := a.rowAt(w.SelectedIndex); mr != nil {
			a.readRow(mr, true)
		}
	})
	w.OnDoubleClick(func(e events.Event) {
		mr := a.rowAt(w.SelectedIndex)
		if mr == nil {
			return
		}
		if !mr.conversation && a.isDraft(mr.message) { // written more, not just read
			a.readMessage, a.readSingle = mr.message, true
			a.EditDraft()
			return
		}
		a.openMessageWindow(mr)
	})
	a.list = w
}

// readRow reads the message (or conversation) of the given row, the last
// one selected, which is marked as read if markRead and it is the only one
// selected. The actions on messages, such as filing, apply to all of the
// selected rows, and those on one message, such as replying, to the last
// one selected.
func (a *App) readRow(mr *messageRow, markRead bool) {
	a.readMessage = mr.message
	if a.allInboxes { // the actions on it are in its account
		a.currentEmail = a.accountOf(mr.message)
	}
	// a message row is shown on its own, a conversation with all of its messages
	a.readSingle = !mr.conversation
	if markRead && len(a.selectedRows) <= 1 { // only when reading it
		a.MarkAsRead()
	}
	a.updateReading()
}

// selectFirst selects the first row of the list of messages and reads it,
// as when another mailbox is shown, or nothing if the list is empty,
// without moving the keyboard focus (as from the mailboxes) into the list,
// and without marking it as read, which going through the mailboxes would
// do to all of their first messages: that is done when it is clicked.
func (a *App) selectFirst() {
	a.readMessage, a.selectedRows = nil, nil
	if a.list != nil {
		a.list.ResetSelectedIndexes()
		a.list.SelectedIndex = -1
		if mr := a.rowAt(0); mr != nil {
			a.list.SelectedIndex = 0
			a.list.SelectIndex(0)
			a.selectedRows = []*messageRow{mr}
			a.readRow(mr, false)
			return
		}
		a.list.NeedsRender()
	}
	a.updateReading()
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
	tree.Add(p, func(w *core.Frame) {
		w.Styler(func(s *styles.Style) {
			s.Align.Items = styles.Center
			s.Grow.Set(1, 0)
			s.Gap.X.Em(0.75)
			s.FixedSpacing = true // not narrowed by a smaller Spacing setting
		})
		tree.AddChild(w, func(w *core.Text) {
			w.SetType(core.TextTitleMedium)
			w.Updater(func() {
				switch {
				case a.searchResults != nil && a.allInboxes:
					w.SetText("Search: all accounts")
				case a.searchResults != nil:
					w.SetText("Search: " + a.currentEmail)
				case a.allInboxes:
					w.SetText("Inboxes")
				default:
					w.SetText(a.mailboxName(a.currentEmail, a.showLabel))
				}
			})
		})
		tree.AddChild(w, func(w *core.Text) { // the count, after the name
			w.SetType(core.TextBodySmall)
			w.Styler(func(s *styles.Style) {
				s.Grow.Set(1, 0)
				s.Color = colors.Scheme.OnSurfaceVariant
			})
			w.Updater(func() {
				w.Text = ""
				if a.searchResults != nil {
					w.Text = fmt.Sprintf("%d messages found", len(a.listCache))
					if a.unreadMessages > 0 {
						w.Text += fmt.Sprintf(", %d unread", a.unreadMessages)
					}
					return
				}
				total := 0
				for _, email := range a.listAccounts() {
					total += a.totalMessages[email][a.showLabel]
				}
				if a.shownMessages < total { // still syncing
					w.Text += fmt.Sprintf("%d of ", a.shownMessages)
				}
				w.Text += fmt.Sprintf("%d messages", total)
				if !a.flatList() {
					w.Text += fmt.Sprintf(" in %d conversations", len(a.listCache))
				}
				if a.unreadMessages > 0 {
					w.Text += fmt.Sprintf(", %d unread", a.unreadMessages)
				}
				if a.unreadOnly {
					w.Text += " (showing the unread)"
				}
			})
		})
		tree.AddChild(w, func(w *core.Button) {
			w.SetIcon(icons.FilterList)
			w.Updater(func() {
				if a.unreadOnly { // shown as on
					w.SetType(core.ButtonTonal).SetTooltip("Showing only the unread messages: click to show all of them")
				} else {
					w.SetType(core.ButtonAction).SetTooltip("Show only the unread messages")
				}
			})
			w.OnClick(func(e events.Event) { a.UnreadOnly(!a.unreadOnly) })
		})
		tree.AddChild(w, func(w *core.Button) {
			w.SetType(core.ButtonAction).SetIcon(icons.Refresh)
			w.SetTooltip("Check for new messages in this mailbox now")
			w.OnClick(func(e events.Event) { a.Refresh() })
		})
		tree.AddChild(w, func(w *core.Button) {
			w.SetType(core.ButtonAction).SetIcon(icons.Tune)
			w.SetTooltip("Settings for the messages list")
			w.SetMenu(a.viewMenu)
		})
	})
	tree.Add(p, func(w *core.TextField) {
		a.searchField = w
		w.SetPlaceholder("Search").SetLeadingIcon(icons.Search)
		w.SetTooltip(`Search the messages of the account: all of the words, "phrases", from: to: cc: subject: body: attachment: for those fields, and -word for not; Escape clears it`)
		w.Styler(func(s *styles.Style) {
			s.Grow.Set(1, 0)
			s.Max.X.Zero()
		})
		w.SetTrailingIcon(icons.Close, func(e events.Event) {
			w.SetText("")
			a.SearchMessages("")
		})
		w.OnChange(func(e events.Event) {
			a.SearchMessages(w.Text())
		})
		w.OnKeyChord(func(e events.Event) {
			if e.KeyChord() == "Escape" {
				e.SetHandled()
				w.SetText("")
				a.SearchMessages("")
			}
		})
	})
	// a new list for the fields shown, which are set before the items
	tree.AddAt(p, fmt.Sprintf("list-%v-%v-%v-%v", a.compactList(), a.labelsShown(), a.attachmentsShown(), a.starsShown()), a.initList)
}

// viewMenu is the menu of the view of the list of messages:
// the switches of the [SettingsData.ListViews].
func (a *App) viewMenu(m *core.Scene, pos image.Point) {
	sw := core.NewSwitches(m) // chips, for bit flags
	core.Bind(&Settings.ListViews, sw)
	sw.OnChange(func(e events.Event) {
		a.saveSettings()
		a.Update()
	})
}

// searchLimit is the maximum number of messages found by a search.
var searchLimit = 1000

// SearchMessages shows the messages of the current account (or of all of
// them, for all of the inboxes) that match the given
// search (see [searchIndex.query]), the newest first, or the messages of
// the mailbox being shown again for an empty one.
func (a *App) SearchMessages(search string) { //types:add
	search = strings.TrimSpace(search)
	a.searchText = search
	a.selectedRows = nil
	if search == "" {
		a.searchResults = nil
		a.Update()
		return
	}
	results := map[string]bool{}
	searched := false
	for _, email := range a.listAccounts() {
		si := a.searches[email]
		if si == nil {
			continue
		}
		searched = true
		keys, err := si.search(search, searchLimit)
		if err != nil {
			core.ErrorSnackbar(a, err, "Error searching")
			return
		}
		for _, k := range keys {
			results[searchKey(email, k)] = true
		}
	}
	if !searched {
		return
	}
	a.searchResults = results
	a.Update()
}

// searchKey returns the key of the message of the given account with the
// given key in [App.searchResults], which can have those of all of the
// accounts (see [App.allInboxes]).
func searchKey(email, key string) string {
	return email + "\x00" + key
}

// FocusSearch moves the keyboard focus to the search field.
func (a *App) FocusSearch() { //types:add
	if a.searchField != nil {
		a.searchField.SetFocus()
	}
}
