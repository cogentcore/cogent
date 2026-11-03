// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lists

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testMobile returns the app with a copy of the sample lists, in the mobile
// layout, in the given body, which is the size of a phone.
func testMobile(t *testing.T, b *core.Body) *Lists {
	ls := testLists(t, b)
	ls.Mobile = true
	b.Styler(func(s *styles.Style) {
		s.Min.Set(units.Dp(360), units.Dp(640))
	})
	b.AddTopBar(func(bar *core.Frame) { // as NewApp does
		core.NewToolbar(bar).Maker(ls.MakeToolbar)
	})
	return ls
}

// click clicks the center of the given widget, with the given events.
func click(src *events.Source, w core.Widget) {
	bb := w.AsWidget().Geom.TotalBBox
	pt := bb.Min.Add(bb.Size().Div(2))
	src.MouseButton(events.MouseDown, events.Left, pt, 0)
	src.MouseButton(events.MouseUp, events.Left, pt, 0)
	time.Sleep(300 * time.Millisecond)
}

// TestMobile tests the levels of the mobile layout: tapping a list in the
// tree of lists opens its items, with the tree of the items and the first
// check field, and an edit button for each, which opens the form of the
// item, whose edits are saved, and the back buttons close them.
func TestMobile(t *testing.T) {
	b := core.NewBody()
	ls := testMobile(t, b)
	b.AssertRender(t, "mobile", func() {
		src := b.Scene.RenderWindow().SystemWindow.Events()
		b.AsyncLock()
		assert.Nil(t, ls.splits(), "no splits")
		chores := findList(ls, "Chores")
		tr := ls.tree.FindSyncNode(chores)
		require.NotNil(t, tr)
		b.AsyncUnlock()
		click(src, tr)

		b.AsyncLock()
		require.NotNil(t, ls.items, "the items are open")
		assert.Equal(t, chores, ls.Current)
		tt := ls.table
		require.NotNil(t, tt)
		names := []string{}
		for i := range reflect.TypeOf(tt.Items).Elem().Elem().Elem().NumField() {
			f := reflect.TypeOf(tt.Items).Elem().Elem().Elem().Field(i)
			if ls.mobileField(f) && f.Tag.Get("display") != "-" {
				names = append(names, f.Name)
			}
		}
		assert.Equal(t, []string{"Task", "Done"}, names)
		nw, _ := tt.RowWidgetNs()
		edit, ok := tt.ListGrid.Child(nw - 1).(*core.Button) // of the first row
		require.True(t, ok)
		b.AsyncUnlock()
		click(src, edit)

		b.AsyncLock()
		require.NotNil(t, ls.form, "the form of the item is open")
		form := ls.form.Child(0).(*core.Form)
		item := reflect.ValueOf(form.Struct).Elem()
		item.Field(0).SetString("Do the laundry")
		form.SendChange()
		assert.Equal(t, `"Do the laundry"`, string(chores.Items[0].Values["Task"]), "saved")
		ls.form.Scene.Close()
		b.AsyncUnlock()
		time.Sleep(300 * time.Millisecond)

		b.AsyncLock()
		assert.Nil(t, ls.form)
		require.NotNil(t, ls.items)
		ls.items.Scene.Close()
		b.AsyncUnlock()
		time.Sleep(300 * time.Millisecond)

		b.AsyncLock()
		defer b.AsyncUnlock()
		assert.Nil(t, ls.items)
		assert.Nil(t, ls.table)
		var saved listFile
		require.NoError(t, json.Unmarshal(ls.files()[listPath(chores.ID)], &saved))
		assert.Equal(t, `"Do the laundry"`, string(saved.Items[0].Values["Task"]))
	})
}

// openMobileItems opens the items of the given list in the mobile layout,
// by tapping it, returning the edit button of its first item.
func openMobileItems(t *testing.T, b *core.Body, ls *Lists, name string) *core.Button {
	src := b.Scene.RenderWindow().SystemWindow.Events()
	b.AsyncLock()
	tr := ls.tree.FindSyncNode(findList(ls, name))
	require.NotNil(t, tr)
	b.AsyncUnlock()
	click(src, tr)
	b.AsyncLock()
	defer b.AsyncUnlock()
	require.NotNil(t, ls.table)
	nw, _ := ls.table.RowWidgetNs()
	edit, ok := ls.table.ListGrid.Child(nw - 1).(*core.Button)
	require.True(t, ok)
	return edit
}

// TestMobileItems tests the window of the items of a list in the mobile
// layout, with the tree of the items and the first check field.
func TestMobileItems(t *testing.T) {
	b := core.NewBody()
	ls := testMobile(t, b)
	b.AssertRender(t, "mobile-items", func() {
		openMobileItems(t, b, ls, "Groceries")
		b.AsyncLock()
		defer b.AsyncUnlock()
		assert.Equal(t, "Groceries", ls.items.Title)
	})
}

// TestMobileForm tests the window of the form of an item in the mobile
// layout, with all of its fields.
func TestMobileForm(t *testing.T) {
	b := core.NewBody()
	ls := testMobile(t, b)
	b.AssertRender(t, "mobile-form", func() {
		edit := openMobileItems(t, b, ls, "Chores")
		click(b.Scene.RenderWindow().SystemWindow.Events(), edit)
		b.AsyncLock()
		defer b.AsyncUnlock()
		require.NotNil(t, ls.form)
		assert.Equal(t, "Laundry", ls.form.Title)
	})
}
