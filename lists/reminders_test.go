// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lists

import (
	"encoding/json"
	"testing"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/system"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFieldFlags tests the flags of fields in files, as text, also in a
// merge of the list types.
func TestFieldFlags(t *testing.T) {
	var f Field
	require.NoError(t, json.Unmarshal([]byte(`{"Name": "Due", "Flags": "Detail|Remind"}`), &f))
	assert.True(t, f.Flags.HasFlag(Detail) && f.Flags.HasFlag(Remind))
	f.Flags = detail
	assert.Contains(t, string(marshal(&f)), `"Flags": "Detail"`)

	// the reminder times changed in ours, and the Detail flag in theirs
	base := marshal([]*ListType{{Name: todoType, Fields: []*Field{{Name: "Task"}, {Name: "Notes"}}}})
	ours := marshal([]*ListType{{Name: todoType, Reminders: []ReminderTime{{DaysPrior: 2, Time: "8:00"}}, Fields: []*Field{{Name: "Task"}, {Name: "Notes"}}}})
	theirs := marshal([]*ListType{{Name: todoType, Fields: []*Field{{Name: "Task"}, {Name: "Notes", Flags: detail}}}})
	merged := mergeFiles(map[string][]byte{typesFile: base}, map[string][]byte{typesFile: ours}, map[string][]byte{typesFile: theirs}, false)
	var types []*ListType
	require.NoError(t, json.Unmarshal(merged[typesFile], &types))
	require.Len(t, types, 1)
	assert.Equal(t, []ReminderTime{{DaysPrior: 2, Time: "8:00"}}, types[0].Reminders)
	require.Len(t, types[0].Fields, 2)
	assert.Equal(t, detail, types[0].Fields[1].Flags)
}

func TestParseTimeOfDay(t *testing.T) {
	for s, want := range map[string]time.Duration{
		"9:00": 9 * time.Hour, "17:30": 17*time.Hour + 30*time.Minute, "5:30pm": 17*time.Hour + 30*time.Minute,
		"5:30 PM": 17*time.Hour + 30*time.Minute, "8am": 8 * time.Hour, " 12 am ": 0, "7": 7 * time.Hour,
	} {
		d, err := parseTimeOfDay(s)
		if assert.NoError(t, err, s) {
			assert.Equal(t, want, d, s)
		}
	}
	_, err := parseTimeOfDay("noon")
	assert.Error(t, err)
}

// reminderItem returns a saved item with the given values.
func reminderItem(t *testing.T, id string, values map[string]any) *Item {
	it := &Item{ID: id, Values: map[string]json.RawMessage{}}
	for k, v := range values {
		b, err := json.Marshal(v)
		require.NoError(t, err)
		it.Values[k] = b
	}
	return it
}

func TestReminders(t *testing.T) {
	b := core.NewBody()
	ls := testLists(t, b)
	Settings.Reminders = ReminderSettings{On: true, Times: defaultReminderTimes()}
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.Local)
	day := func(d int) time.Time { return time.Date(2026, 11, d, 0, 0, 0, 0, time.Local) }

	l := findList(ls, "Chores")
	lt := ls.TypeOf(l)
	require.Equal(t, todoType, lt.Name)
	l.Items = []*Item{
		reminderItem(t, "a", map[string]any{"Task": "Today", "Due": day(20)}), // past 9:00am
		reminderItem(t, "b", map[string]any{"Task": "Tomorrow", "Due": day(21)}),
		reminderItem(t, "c", map[string]any{"Task": "Done", "Done": true, "Due": day(22)}),
		reminderItem(t, "d", map[string]any{"Task": "No date"}),
		{ID: "e", Values: map[string]json.RawMessage{}, Items: []*Item{
			reminderItem(t, "f", map[string]any{"Task": "In a category", "Due": day(23)}),
		}},
	}
	// 1 day before at 9:00pm, and on the day at 9:00am
	ns := ls.reminders(now)
	require.Len(t, ns, 4)
	assert.Equal(t, system.Notification{ID: l.ID + "/b/Due/20261120T2100", Title: "Tomorrow", Body: "Chores: Due Sat Nov 21", Time: day(20).Add(21 * time.Hour)}, *ns[0])
	assert.Equal(t, day(21).Add(9*time.Hour), ns[1].Time)
	assert.Equal(t, "In a category", ns[2].Title)
	assert.Equal(t, day(22).Add(21*time.Hour), ns[2].Time)
	assert.Equal(t, day(23).Add(9*time.Hour), ns[3].Time)

	lt.Reminders = []ReminderTime{{Time: "5:30pm"}} // the list type's own times
	ns = ls.reminders(now)
	require.Len(t, ns, 3, "today at 5:30pm")
	assert.Equal(t, day(20).Add(17*time.Hour+30*time.Minute), ns[0].Time)
	lt.Reminders = nil

	Settings.Reminders.On = false
	assert.Empty(t, ls.reminders(now))
	Settings.Reminders.On = true

	lt.Fields[2].Flags = 0 // Due without Remind
	assert.Empty(t, ls.reminders(now))
	lt.Fields[2].Flags = remind
}

// TestDateTimeReminders tests the reminders of a date and time: at that
// time, and at the reminder times before it.
func TestDateTimeReminders(t *testing.T) {
	Settings.Reminders = ReminderSettings{On: true, Times: defaultReminderTimes()}
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.Local)
	at := time.Date(2026, 11, 21, 15, 0, 0, 0, time.Local)
	lt := &ListType{Fields: []*Field{{Name: "Task"}, {Name: "At", Type: DateTime, Flags: remind}}}
	l := &List{ID: "l", Name: "Meetings"}
	it := reminderItem(t, "m", map[string]any{"Task": "Meet", "At": at})
	times := func() []time.Time {
		var ts []time.Time
		for _, n := range lt.itemReminders(l, it, now) {
			ts = append(ts, n.Time)
		}
		return ts
	}
	ns := lt.itemReminders(l, it, now)
	require.Len(t, ns, 3)
	assert.Equal(t, "Meetings: At Sat Nov 21 3:00pm", ns[0].Body)
	assert.Equal(t, []time.Time{at.Add(-18 * time.Hour), at.Add(-6 * time.Hour), at}, times())

	// times on the day after it are left out, and one at it is not repeated
	lt.Reminders = []ReminderTime{{Time: "5pm"}, {Time: "3pm"}}
	assert.Equal(t, []time.Time{at}, times())
}

// TestReminderClicked tests that a click on a reminder opens its list.
func TestReminderClicked(t *testing.T) {
	b := core.NewBody()
	ls := testLists(t, b)
	core.AssertRender(t, b, "reminder-clicked", func() {
		l := findList(ls, "Chores")
		ls.reminderClicked(l.ID + "/x/Due")
		b.AsyncLock()
		assert.Same(t, l, ls.Current)
		b.AsyncUnlock()
	})
}
