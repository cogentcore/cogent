// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lists

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/system"
)

// Reminders are notifications of the operating system for the dates of
// the items in the date fields with the [Remind] flag, as for due dates,
// at the time of day of the list type (see [ListType.Reminder]), or of
// the [ReminderSettings], for dates without times. They are scheduled
// with [system.Notifier.Sync] whenever the lists change, so that the
// operating system shows them even when the app is not running (except
// on Linux and the web). A click on one opens its list.

// ReminderSettings are the settings for reminders.
type ReminderSettings struct {

	// On is whether reminders are shown, for the date fields of the list
	// types with the Remind flag, such as Due. Turning it on asks for
	// permission to show notifications, if it is not given yet.
	On bool `default:"true"`

	// Time is the time of day of the reminders, such as 9:00 or 5:30pm,
	// for the dates without times, for the list types without their own
	// reminder time.
	Time string `default:"9:00"`
}

// reminderLayouts are the layouts of the times of day of reminders.
var reminderLayouts = []string{"15:04", "3:04pm", "3:04 pm", "3pm", "3 pm", "15"}

// parseTimeOfDay returns the time of day of the given text, such as 9:00
// or 5:30pm, as a duration since midnight.
func parseTimeOfDay(s string) (time.Duration, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	for _, layout := range reminderLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return time.Duration(t.Hour())*time.Hour + time.Duration(t.Minute())*time.Minute, nil
		}
	}
	return 0, fmt.Errorf("invalid time of day %q: it must be like 9:00 or 5:30pm", s)
}

// reminderTime returns the time of day of the reminders of the list type.
func (lt *ListType) reminderTime() time.Duration {
	for _, s := range []string{lt.Reminder, Settings.Reminders.Time} {
		if strings.TrimSpace(s) == "" {
			continue
		}
		if d, err := parseTimeOfDay(s); err == nil {
			return d
		}
	}
	return 9 * time.Hour
}

// reminderSep separates the parts of the IDs of reminders:
// the ID of the list, of the item, and the name of the field.
const reminderSep = "/"

// reminders returns the reminders of the lists, if they are on, for the
// items whose dates are after now (see [Reminders]).
func (ls *Lists) reminders(now time.Time) []*system.Notification {
	if !Settings.Reminders.On || ls.Root == nil {
		return nil
	}
	var ns []*system.Notification
	for n := range ls.Root.All() {
		l := n.(*List)
		lt := ls.TypeOf(l)
		if lt == nil || !slices.ContainsFunc(lt.Fields, func(f *Field) bool { return f.Flags.HasFlag(Remind) }) {
			continue
		}
		var walk func(items []*Item)
		walk = func(items []*Item) {
			for _, it := range items {
				ns = append(ns, lt.itemReminders(l, it, now)...)
				walk(it.Items)
			}
		}
		walk(l.Items)
	}
	return ns
}

// itemReminders returns the reminders of the given item of the given list,
// of this type, for its dates after now, if it is not done: if it has none
// of the filter values of the fields (see [ListType.showItem]).
func (lt *ListType) itemReminders(l *List, it *Item, now time.Time) []*system.Notification {
	for _, f := range lt.Fields {
		filter := strings.TrimSpace(f.Filter)
		if filter == "" {
			continue
		}
		var v any
		if json.Unmarshal(it.Values[f.Name], &v) == nil && strings.EqualFold(reflectx.ToString(v), filter) {
			return nil
		}
	}
	var ns []*system.Notification
	for _, f := range lt.Fields {
		if !f.Flags.HasFlag(Remind) || (f.Type != Date && f.Type != DateTime) {
			continue
		}
		var t time.Time
		if json.Unmarshal(it.Values[f.Name], &t) != nil || t.IsZero() {
			continue
		}
		t = t.Local()
		if f.Type == Date {
			y, m, d := t.Date()
			t = time.Date(y, m, d, 0, 0, 0, 0, time.Local).Add(lt.reminderTime())
		}
		if !t.After(now) {
			continue
		}
		ns = append(ns, &system.Notification{
			ID:    l.ID + reminderSep + it.ID + reminderSep + f.Name,
			Title: lt.itemTitle(it),
			Body:  l.Name + ": " + f.Name + " " + t.Format("Mon Jan 2"),
			Time:  t,
		})
	}
	return ns
}

// itemTitle returns the value of the first field of the given item,
// for the title of its reminders.
func (lt *ListType) itemTitle(it *Item) string {
	if len(lt.Fields) == 0 {
		return ""
	}
	var v any
	json.Unmarshal(it.Values[lt.Fields[0].Name], &v)
	if s := reflectx.ToString(v); s != "" {
		return s
	}
	return "Reminder"
}

// reminderState is the state of the reminders of the lists.
type reminderState struct {

	// synced are the reminders that were last scheduled, which are not
	// scheduled again if they are the same.
	synced []system.Notification

	// asked is whether permission was asked for.
	asked bool

	// clicks is whether the click function is set.
	clicks bool

	// on is whether reminders were on in the settings, for asking for
	// permission when they are turned on.
	on bool
}

// syncReminders schedules the reminders of the lists, if they changed
// (see [Reminders]), asking for permission the first time there are any.
func (ls *Lists) syncReminders() {
	nf := system.TheApp.Notifier()
	rs := &ls.remind
	if !rs.clicks {
		rs.clicks = true
		rs.on = Settings.Reminders.On
		nf.OnClick(ls.reminderClicked)
	}
	ns := system.ScheduleOrder(ls.reminders(time.Now()), time.Now())
	same := len(ns) == len(rs.synced)
	for i := 0; same && i < len(ns); i++ {
		same = *ns[i] == rs.synced[i]
	}
	if same {
		return
	}
	rs.synced = rs.synced[:0]
	for _, n := range ns {
		rs.synced = append(rs.synced, *n)
	}
	if len(ns) > 0 && !rs.asked {
		rs.asked = true
		nf.RequestPermission(func(allowed bool) {
			if !allowed {
				slog.Info("Lists: notifications are not allowed for reminders")
			}
		})
	}
	if err := nf.Sync(ns); err != nil && !errors.Is(err, errors.ErrUnsupported) {
		slog.Error("Lists: error scheduling reminders", "err", err)
	}
}

// reminderClicked opens the list of the reminder with the given ID,
// which was clicked.
func (ls *Lists) reminderClicked(id string) {
	listID, _, _ := strings.Cut(id, reminderSep)
	ls.AsyncLock()
	defer ls.AsyncUnlock()
	for n := range ls.Root.All() {
		if l := n.(*List); l.ID == listID && !l.IsGroup() {
			ls.SetCurrent(l)
			if ls.tree != nil {
				if tn := ls.tree.FindSyncNode(l); tn != nil {
					tn.Select()
				}
			}
			ls.openItems()
			return
		}
	}
}

// requestReminders asks for permission to show the reminders, after they
// are turned on in the settings, telling the user if it is not given.
func (ls *Lists) requestReminders() {
	ls.remind.asked = true
	system.TheApp.Notifier().RequestPermission(func(allowed bool) {
		if allowed {
			return
		}
		ls.AsyncLock()
		defer ls.AsyncUnlock()
		core.MessageSnackbar(ls, "Notifications are not allowed for reminders: they can be allowed in the settings of the system")
	})
}
