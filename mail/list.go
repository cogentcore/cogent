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
	return !v.HasFlag(ViewConversations)
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

// updateConversations sets the replies of the messages of the current
// account, and returns the starts of the conversations, and the number of
// messages with the label being shown. On Gmail, the conversations are its
// threads (by X-GM-THRID), whose first message is the start, and only those
// with a message with the label being shown are made, as there can be very
// many messages; elsewhere, a conversation starts with the message that the
// others reply to (by [CacheMessage.InReplyTo]).
func (a *App) updateConversations() (starts []*CacheMessage, shown int) {
	mp := a.cache[a.currentEmail]
	want := map[uint64]bool{} // the threads with the label, on Gmail
	for _, cm := range mp {
		cm.replies = nil
		if a.hasLabel(cm) {
			shown++
			want[cm.GmailThreadID] = true
		}
	}
	if a.gmail[a.currentEmail] {
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

// hasLabel returns whether the given message has the label being shown,
// with the labels it is getting, if any (see [App.pendingLabels]).
func (a *App) hasLabel(cm *CacheMessage) bool {
	if a.isStarredLabel(a.showLabel) { // by the flag, as soon as it is starred
		return cm.isStarred()
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
				"account", a.currentEmail, "messages", len(a.cache[a.currentEmail]), "rows", len(a.listRows))
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
	email := a.currentEmail
	showLabels, showAttachments, showStars := a.labelsShown(), a.attachmentsShown(), a.starsShown()

	rowOf := func(cm *CacheMessage) *messageRow {
		mr := oldMessages[cm]
		if mr == nil {
			mr = &messageRow{message: cm}
		}
		mr.elsewhere = false
		mr.From = senderName(cm, email)
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
		for _, cm := range a.cache[email] {
			if a.searchResults[cm.key] {
				a.listCache = append(a.listCache, cm)
				rows = append(rows, rowOf(cm))
				if !slices.Contains(cm.Flags, imap.FlagSeen) {
					a.unreadMessages++
				}
			}
		}
		a.shownMessages = len(a.listCache)
		slices.SortFunc(rows, func(a, b *messageRow) int { return cmp.Compare(b.Date, a.Date) })
		a.listRows = rows
		return
	}
	for _, start := range starts {
		if flat {
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
			mr := rowOf(cm)
			mr.elsewhere = !a.hasLabel(cm)
			mr.Message.Dim = mr.elsewhere
			cr.Children = append(cr.Children, mr)
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
	slices.SortFunc(rows, func(a, b *messageRow) int {
		return cmp.Compare(b.Date, a.Date)
	})
	a.listRows = rows
}

// initList initializes the tree table of the list of messages.
func (a *App) initList(w *core.TreeTable) {
	w.SetReadOnly(true)
	w.ReadOnlyKeyNav = true
	w.ShowField = a.listField // before the items
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
		a.updateReading()
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
				if a.searchResults != nil {
					w.SetText("Search: " + a.currentEmail)
					return
				}
				w.SetText(friendlyLabelName(a.showLabel))
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
				total := a.totalMessages[a.currentEmail][a.showLabel]
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
			})
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

// SearchMessages shows the messages of the current account that match the given
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
	si := a.searches[a.currentEmail]
	if si == nil {
		return
	}
	keys, err := si.search(search, searchLimit)
	if err != nil {
		core.ErrorSnackbar(a, err, "Error searching")
		return
	}
	a.searchResults = map[string]bool{}
	for _, k := range keys {
		a.searchResults[k] = true
	}
	a.Update()
}

// FocusSearch moves the keyboard focus to the search field.
func (a *App) FocusSearch() { //types:add
	if a.searchField != nil {
		a.searchField.SetFocus()
	}
}
