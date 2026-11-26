// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lists

import (
	"encoding/json"
	"errors"
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
// at the reminder times of the list type (see [ListType.Reminders]), or of
// the [ReminderSettings]: each is a number of days before the date, at a
// time of day, such as 1 day before at 9:00pm, and on the day at 9:00am.
// A date and time also reminds at that time. An item that is overdue
// reminds at all of the times of day, every day, until it is done. They are scheduled with
// [system.Notifier.Sync] whenever the lists change, so that the operating
// system shows them even when the app is not running (except on Linux and
// the web). A click on one opens its list.

// ReminderSettings are the settings for reminders.
type ReminderSettings struct {

	// On is whether reminders are shown, for the date fields of the list
	// types with the Remind flag, such as Due. Turning it on asks for
	// permission to show notifications, if it is not given yet.
	On bool `default:"true"`

	// Times are the times of the reminders, each a number of days before
	// the date, at a time of day, for the list types without their own
	// reminder times. They are 1 day before at 9:00pm and on the day at
	// 9:00am by default.
	Times []ReminderTime
}

// ReminderTime is a time of reminders: a number of days before a date,
// at a time of day.
type ReminderTime struct {

	// DaysPrior is the number of days before the date, 0 for on the day.
	DaysPrior int `min:"0"`

	// Time is the time of day, of which only the hour and minute are used.
	Time time.Time `display:"time"`
}

// defaultReminderTimes returns the default times of reminders:
// 1 day before at 9:00pm, and on the day at 9:00am.
func defaultReminderTimes() []ReminderTime {
	return []ReminderTime{{DaysPrior: 1, Time: timeOfDay(21, 0)}, {DaysPrior: 0, Time: timeOfDay(9, 0)}}
}

// timeOfDay returns the given time of day, as a [ReminderTime.Time].
func timeOfDay(hour, minute int) time.Time {
	return time.Date(2000, 1, 1, hour, minute, 0, 0, time.Local)
}

// reminderTimes returns the times of the reminders of the list type:
// its own, or those of the settings.
func (lt *ListType) reminderTimes() []ReminderTime {
	if len(lt.Reminders) > 0 {
		return lt.Reminders
	}
	return Settings.Reminders.Times
}

// at returns the time of the reminder for the given day, in local time,
// and false if it has no time of day.
func (rt ReminderTime) at(day time.Time) (time.Time, bool) {
	if rt.Time.IsZero() {
		return time.Time{}, false
	}
	y, m, d := day.Date()
	return time.Date(y, m, d-rt.DaysPrior, rt.Time.Hour(), rt.Time.Minute(), 0, 0, time.Local), true
}

// overdueDays is the number of days after now for which the daily
// reminders of the overdue items are scheduled, as they are scheduled
// again whenever the app runs.
const overdueDays = 2

// reminderSep separates the parts of the IDs of reminders:
// the ID of the list, of the item, the name of the field, and the time.
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
// of this type, after now, if it is not done: if it has none of the filter
// values of the fields (see [ListType.showItem]). After its date, it is
// overdue, and has reminders at all of the times of day, every day.
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
		var date time.Time
		if json.Unmarshal(it.Values[f.Name], &date) != nil || date.IsZero() {
			continue
		}
		date = date.Local()
		due := date.Format("Mon Jan 2")
		var times []time.Time
		if f.Type == DateTime {
			due += date.Format(" 3:04pm")
			times = append(times, date)
		}
		// the item is overdue after the time, or the day, of the date
		overdue := date
		if f.Type == Date {
			y, m, d := date.Date()
			overdue = time.Date(y, m, d+1, 0, 0, 0, 0, time.Local)
		}
		rts := lt.reminderTimes()
		for _, rt := range rts {
			if t, ok := rt.at(date); ok && t.Before(overdue) {
				times = append(times, t)
			}
		}
		// when it is overdue, at all of the times of day, every day
		end := now.AddDate(0, 0, overdueDays)
		y, m, d := overdue.Date()
		for day := time.Date(y, m, d, 0, 0, 0, 0, time.Local); !day.After(end); day = day.AddDate(0, 0, 1) {
			for _, rt := range rts {
				if t, ok := (ReminderTime{Time: rt.Time}).at(day); ok && !t.Before(overdue) && !t.After(end) {
					times = append(times, t)
				}
			}
		}
		slices.SortFunc(times, func(a, b time.Time) int { return a.Compare(b) })
		times = slices.CompactFunc(times, time.Time.Equal)
		for _, t := range times {
			if !t.After(now) {
				continue
			}
			body := l.Name + ": " + f.Name + " " + due
			if !t.Before(overdue) {
				body = l.Name + ": overdue, " + f.Name + " " + due
			}
			ns = append(ns, &system.Notification{
				ID:    l.ID + reminderSep + it.ID + reminderSep + f.Name + reminderSep + t.Format("20060102T1504"),
				Title: lt.itemTitle(it),
				Body:  body,
				Time:  t,
			})
		}
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
