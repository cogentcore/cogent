// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lists

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles/states"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testLists returns the app with a copy of the sample lists,
// in the given body.
func testLists(t *testing.T, b *core.Body) *Lists {
	Settings.Splits = nil // the standard splits, not those of other tests or the user
	Settings.Sync = SyncSettings{}
	dir := t.TempDir()
	data, err := os.ReadFile(filepath.Join("testdata", "sample", oldListsFile))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, oldListsFile), data, 0o644))
	ls := NewLists(b)
	require.NoError(t, ls.Open(dir))
	return ls
}

// findList returns the list with the given name.
func findList(ls *Lists, name string) *List {
	return ls.Root.FindPath("Home/" + name).(*List)
}

func TestStructType(t *testing.T) {
	lt := &ListType{Name: "Test", Fields: []*Field{
		{Name: "Item"},
		{Name: "Due date", Type: Date},
		{Name: "count", Type: Integer, Default: "3", Tags: `min:"1"`},
		{Name: "Item"},
		{Name: "2nd", Type: Check},
	}}
	st := lt.StructType()
	assert.Equal(t, 8, st.NumField())
	names := []string{}
	for i := range 5 {
		names = append(names, st.Field(i).Name)
	}
	assert.Equal(t, []string{"Item", "DueDate", "Count", "Item2", "F2nd"}, names)
	assert.Equal(t, `display:"date"`, string(st.Field(1).Tag))
	assert.Equal(t, `label:"count" default:"3" min:"1"`, string(st.Field(2).Tag))
	assert.Equal(t, st, lt.StructType(), "the same type for the same fields")
}

func TestItemsRoundTrip(t *testing.T) {
	lt := standardTypes()[1] // shopping
	saved := []*Item{
		{Values: map[string]json.RawMessage{"Item": []byte(`"Produce"`)}, Closed: true, Items: []*Item{
			{Values: map[string]json.RawMessage{"Item": []byte(`"Apples"`), "Quantity": []byte(`6`)}},
		}},
	}
	items := lt.newItems(saved)
	apples := items.Elem().Index(0).Elem().Field(6).Index(0).Elem().Elem()
	assert.Equal(t, "Apples", apples.Field(0).String())
	assert.Equal(t, int64(6), apples.Field(2).Int())
	assert.True(t, items.Elem().Index(0).Elem().Field(5).Bool(), "closed")

	back := lt.savedItems(items)
	assert.Equal(t, `"Produce"`, string(back[0].Values["Item"]))
	assert.True(t, back[0].Closed)
	assert.Equal(t, `6`, string(back[0].Items[0].Values["Quantity"]))
	assert.Equal(t, `1`, string(back[0].Values["Quantity"]), "the default")
	assert.NotEmpty(t, back[0].ID, "a new ID")
	assert.NotEqual(t, back[0].ID, back[0].Items[0].ID)
	again := lt.savedItems(items)
	assert.Equal(t, back[0].ID, again[0].ID, "the same ID, which is in the struct")
	assert.Equal(t, back[0].Items[0].ID, again[0].Items[0].ID)

	// a copy of an item, with the same ID, gets a new one
	items.Elem().Set(reflect.Append(items.Elem(), items.Elem().Index(0)))
	copies := lt.savedItems(items)
	assert.Equal(t, back[0].ID, copies[0].ID)
	assert.NotEqual(t, copies[0].ID, copies[1].ID)
}

// TestItemsOtherTypes tests that the values of the items whose fields have
// other types than when they were saved, as after a change of the type of
// a field, are the zero values, as for a check field with a text value.
func TestItemsOtherTypes(t *testing.T) {
	lt := standardTypes()[0] // to do
	saved := []*Item{{Values: map[string]json.RawMessage{"Task": []byte(`"Laundry"`), "Done": []byte(`"yes"`), "Priority": []byte(`"high"`)}}}
	items := lt.newItems(saved)
	item := items.Elem().Index(0).Elem()
	assert.Equal(t, "Laundry", item.Field(0).String())
	assert.False(t, item.Field(1).Bool())
	assert.Equal(t, int64(3), item.Field(3).Int(), "the default")
	back := lt.savedItems(items)
	assert.Equal(t, `false`, string(back[0].Values["Done"]))
	lt.checkStates(items)
}

func TestOpenSave(t *testing.T) {
	ls := testLists(t, core.NewBody())
	assert.Len(t, ls.Types, 5, "the standard types")
	gr := findList(ls, "Groceries")
	assert.Equal(t, "Shopping", gr.TypeName)
	assert.Len(t, gr.Items, 3)
	gr.ColumnWidths = map[string]float32{"Item": 120}
	require.NoError(t, ls.Save())

	ls2 := NewLists(core.NewBody())
	require.NoError(t, ls2.Open(ls.Dir))
	assert.Equal(t, ls.Root.listData(), ls2.Root.listData())
	assert.Equal(t, ls.files(), ls2.files(), "with the same IDs")

	// the old file of all of the lists is converted to the files of each list
	assert.NoFileExists(t, filepath.Join(ls.Dir, oldListsFile))
	assert.FileExists(t, filepath.Join(ls.Dir, oldListsBackup))
	assert.FileExists(t, filepath.Join(ls.Dir, treeFile))
	assert.FileExists(t, filepath.Join(ls.Dir, filepath.FromSlash(listPath(gr.ID))))

	// the file of a deleted list is deleted
	gr.Delete()
	require.NoError(t, ls.Save())
	assert.NoFileExists(t, filepath.Join(ls.Dir, filepath.FromSlash(listPath(gr.ID))))
}

func TestTypesEdited(t *testing.T) {
	ls := testLists(t, core.NewBody())
	gr := findList(ls, "Groceries")
	gr.ColumnWidths = map[string]float32{"Quantity": 50}
	snap := ls.snapshotTypes()
	shopping := ls.TypeOf(gr)
	shopping.Name = "Groceries"
	shopping.Fields[2].Name = "Item count" // was Quantity
	ls.typesEdited(snap)
	assert.Equal(t, "Groceries", gr.TypeName)
	assert.Equal(t, `6`, string(gr.Items[0].Items[0].Values["Item count"]))
	assert.Equal(t, map[string]float32{"ItemCount": 50}, gr.ColumnWidths)

	snap = ls.snapshotTypes()
	ls.Types = ls.Types[2:] // deletes the to-do and shopping types
	ls.typesEdited(snap)
	assert.Equal(t, shopping, ls.TypeOf(gr), "the type is kept, as it is used")
}

func TestLists(t *testing.T) {
	b := core.NewBody()
	ls := testLists(t, b)
	ls.Current = findList(ls, "Groceries")
	ls.Current.ColumnWidths = map[string]float32{"Item": 160}
	b.AssertRender(t, "lists", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		assert.Equal(t, ls.Current.ColumnWidths, ls.table.ColumnWidths, "the widths of the list")
	})
}

// TestEditItems tests that edits of the items of the current list
// are saved, and that the current list changes when one is selected.
func TestEditItems(t *testing.T) {
	b := core.NewBody()
	ls := testLists(t, b)
	b.AssertRender(t, "edit-items", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		tr := ls.tree.FindSyncNode(findList(ls, "Chores"))
		require.NotNil(t, tr)
		tr.SelectEvent(events.SelectOne)
		assert.Equal(t, findList(ls, "Chores"), ls.Current)
		ls.Update()
		require.NotNil(t, ls.table)
		ls.table.AddChild(1) // in Fix faucet

		saved := NewLists(core.NewBody())
		require.NoError(t, saved.Open(ls.Dir))
		chores := findList(saved, "Chores")
		require.Len(t, chores.Items[1].Items, 1)
		assert.Equal(t, `3`, string(chores.Items[1].Items[0].Values["Priority"]), "the default")
	})
}

// TestTreeInsert tests that inserting in the tree of lists adds a new
// list of the type of the current list, without a dialog.
func TestTreeInsert(t *testing.T) {
	b := core.NewBody()
	ls := testLists(t, b)
	b.AssertRender(t, "tree-insert", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		gr := findList(ls, "Groceries")
		ls.SetCurrent(gr)
		ls.tree.FindSyncNode(gr).InsertAfter()
		nl, ok := ls.Root.FindPath("Home/New list").(*List)
		require.True(t, ok, "the new list")
		assert.Equal(t, 1, nl.IndexInParent(), "after Groceries")
		assert.False(t, nl.IsGroup())
		assert.Equal(t, "Shopping", nl.TypeName)
		home := ls.Root.FindPath("Home").(*List)
		ls.tree.FindSyncNode(home).InsertBefore()
		assert.False(t, ls.Root.Child(0).(*List).IsGroup(), "a list, not a group, before Home")
	})
}

// TestUndo tests undoing and redoing edits of a list, including a change
// of its type, which are saved.
func TestUndo(t *testing.T) {
	b := core.NewBody()
	ls := testLists(t, b)
	b.AssertRender(t, "undo", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		gr := findList(ls, "Groceries")
		ls.SetCurrent(gr)
		names := func() []string {
			var ns []string
			for _, it := range gr.Items {
				ns = append(ns, string(it.Values["Item"]))
			}
			return ns
		}
		orig := names()
		ls.table.InsertAfter(0) // after Produce, as its sibling
		assert.Len(t, gr.Items, 4)
		ls.SetType(ls.Types[0]) // to do
		assert.Equal(t, "To do", gr.TypeName)

		ls.Undo()
		assert.Equal(t, "Shopping", gr.TypeName, "undo the type change")
		assert.Len(t, gr.Items, 4)
		ls.Undo()
		assert.Equal(t, orig, names(), "undo the insert")
		ls.Undo() // nothing more

		ls.Redo()
		assert.Len(t, gr.Items, 4, "redo the insert")
		ls.Redo()
		assert.Equal(t, "To do", gr.TypeName, "redo the type change")
		ls.Redo() // nothing more
		ls.Undo()
		assert.Equal(t, "Shopping", gr.TypeName)

		saved := NewLists(core.NewBody())
		require.NoError(t, saved.Open(ls.Dir))
		assert.Equal(t, "Shopping", findList(saved, "Groceries").TypeName, "saved")
	})
}

// TestDeleteField tests deleting a field in the middle of the type of
// the current list, which is shown.
func TestDeleteField(t *testing.T) {
	b := core.NewBody()
	ls := testLists(t, b)
	b.AssertRender(t, "delete-field", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		chores := findList(ls, "Chores")
		ls.SetCurrent(chores)
		snap := ls.snapshotTypes()
		todo := ls.TypeOf(chores)
		todo.Fields = append(todo.Fields[:3], todo.Fields[4:]...) // Priority
		ls.typesEdited(snap)
		ls.Restyle()
		assert.Len(t, ls.table.ColumnWidths, 0)
		assert.Equal(t, `"washer"`, string(chores.Items[1].Values["Notes"]))
	})
}

// TestUndoFocus tests that undo keeps the focus where it is.
func TestUndoFocus(t *testing.T) {
	b := core.NewBody()
	ls := testLists(t, b)
	b.AssertRender(t, "undo-focus", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		ls.SetCurrent(findList(ls, "Groceries"))
		tt := ls.table
		nw, _ := tt.RowWidgetNs()
		cell := tt.ListGrid.Child(1 * nw).AsTree() // Apples
		tf := cell.Child(cell.NumChildren() - 1).(*core.TextField)
		tf.SetText("Pears").SendChange()
		tf.SetFocus()
		tt.SelectIndexEvent(1, events.SelectOne)
		ls.Undo()
		assert.Equal(t, "Apples", tf.Text(), "undone in the same text field")
		assert.True(t, tf.StateIs(states.Focused), "which has the focus")
		assert.Equal(t, 1, tt.SelectedIndex)
	})
}

// TestUniqueNames tests that the lists in a group have unique names,
// for inserting new lists twice, renaming, and loading.
func TestUniqueNames(t *testing.T) {
	b := core.NewBody()
	ls := testLists(t, b)
	b.AssertRender(t, "unique-names", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		gr := findList(ls, "Groceries")
		ls.SetCurrent(gr)
		ls.tree.FindSyncNode(gr).InsertBefore()
		ls.tree.FindSyncNode(gr).InsertBefore()
		home := ls.Root.FindPath("Home").(*List)
		var names []string
		for _, c := range home.Children {
			names = append(names, c.AsTree().Name)
		}
		assert.Equal(t, []string{"New list", "New list 2", "Groceries", "Chores"}, names)

		setUniqueName(findList(ls, "Chores"), "Groceries")
		assert.Equal(t, "Groceries 2", home.Child(3).AsTree().Name)

		dup := &listData{Name: "Lists", Lists: []*listData{{Name: "A"}, {Name: "A"}}}
		l := newList(dup)
		assert.Equal(t, "A 2", l.Child(1).AsTree().Name)
	})
}

// TestNewListFocus tests that the table of a new list made in the tree of
// lists gets the focus, so that the insert keys add items to it.
func TestNewListFocus(t *testing.T) {
	b := core.NewBody()
	ls := testLists(t, b)
	b.AssertRender(t, "new-list-focus", func() {
		b.AsyncLock()
		gr := findList(ls, "Groceries")
		ls.SetCurrent(gr)
		tr := ls.tree.FindSyncNode(gr)
		tr.SetFocus()
		tr.InsertAfter()
		b.AsyncUnlock()
		time.Sleep(300 * time.Millisecond)
		b.AsyncLock()
		nl := ls.Root.FindPath("Home/New list").(*List)
		assert.Equal(t, nl, ls.Current)
		assert.True(t, ls.table.ContainsFocus() || ls.table.StateIs(states.Focused), "the table of the new list")
		b.AsyncUnlock()
		src := b.Scene.RenderWindow().SystemWindow.Events()
		for ch, kf := range *keymap.ActiveMap {
			if kf == keymap.Insert {
				r, code, mods, err := ch.Decode()
				require.NoError(t, err)
				src.Key(events.KeyDown, r, code, mods)
				src.Key(events.KeyUp, r, code, mods)
				break
			}
		}
		time.Sleep(200 * time.Millisecond)
		b.AsyncLock()
		defer b.AsyncUnlock()
		assert.Len(t, nl.Items, 1, "an item added by the insert key")
	})
}

// TestSortSaved tests that sorting the items of a list by a column, by
// clicking its header, is saved, and restored when the list is shown.
func TestSortSaved(t *testing.T) {
	b := core.NewBody()
	ls := testLists(t, b)
	b.AssertRender(t, "sort-saved", func() {
		b.AsyncLock()
		gr := findList(ls, "Groceries")
		ls.SetCurrent(gr)
		b.AsyncUnlock()
		time.Sleep(200 * time.Millisecond)
		b.AsyncLock()
		header := ls.table.Child(0).AsTree() // the header of Got is after Item and its handle
		bb := header.Child(2).(core.Widget).AsWidget().Geom.TotalBBox
		b.AsyncUnlock()
		src := b.Scene.RenderWindow().SystemWindow.Events()
		pt := bb.Min.Add(bb.Size().Div(2))
		src.MouseButton(events.MouseDown, events.Left, pt, 0)
		src.MouseButton(events.MouseUp, events.Left, pt, 0)
		time.Sleep(200 * time.Millisecond)
		b.AsyncLock()
		defer b.AsyncUnlock()
		assert.Equal(t, "Got:up", gr.Sort)
		produce := gr.Items[0].Items
		assert.Equal(t, `true`, string(produce[1].Values["Got"]), "Spinach, got, after Apples")

		saved := NewLists(core.NewBody())
		require.NoError(t, saved.Open(ls.Dir))
		assert.Equal(t, "Got:up", findList(saved, "Groceries").Sort, "saved")

		ls.SetCurrent(findList(ls, "Chores"))
		assert.Equal(t, "", ls.table.SortFieldName(), "not for another list")
		ls.SetCurrent(gr)
		assert.Equal(t, "Got:up", ls.table.SortFieldName(), "restored")
	})
}

// TestHideGroupValues tests the switch for the values of the groups,
// such as categories, which is saved with the list.
func TestHideGroupValues(t *testing.T) {
	b := core.NewBody()
	ls := testLists(t, b)
	b.AssertRender(t, "hide-group-values", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		rel := ls.Root.FindPath("Work/Release").(*List)
		ls.SetCurrent(rel)
		var sw *core.Switch
		ls.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
			if s, ok := cw.(*core.Switch); ok && s.Text == "Group values" {
				sw = s
			}
			return sw == nil
		})
		require.NotNil(t, sw)
		assert.True(t, sw.IsChecked(), "shown by default")
		sw.SetChecked(false)
		sw.SendChange()
		assert.True(t, rel.HideGroupValues)
		assert.True(t, ls.table.HideGroupValues)

		saved := NewLists(core.NewBody())
		require.NoError(t, saved.Open(ls.Dir))
		assert.True(t, saved.Root.FindPath("Work/Release").(*List).HideGroupValues, "saved")
	})
}

// TestUndoKeys tests undoing an edit with each of the keys of undo, after
// the edit is accepted in its text field, which keeps the focus, so that
// typing goes on in it.
func TestUndoKeys(t *testing.T) {
	b := core.NewBody()
	ls := testLists(t, b)
	b.AddTopBar(func(bar *core.Frame) {
		core.NewToolbar(bar).Maker(ls.MakeToolbar)
	})
	b.AssertRender(t, "undo-keys", func() {
		b.AsyncLock()
		ls.SetCurrent(findList(ls, "Chores"))
		b.AsyncUnlock()
		time.Sleep(200 * time.Millisecond)
		b.AsyncLock()
		cell := ls.table.ListGrid.Child(0).AsTree()
		tf := cell.Child(cell.NumChildren() - 1).(*core.TextField)
		bb := tf.Geom.TotalBBox
		b.AsyncUnlock()
		src := b.Scene.RenderWindow().SystemWindow.Events()
		press := func(ch key.Chord) {
			r, code, mods, err := ch.Decode()
			require.NoError(t, err)
			src.Key(events.KeyDown, r, code, mods)
			src.Key(events.KeyUp, r, code, mods)
			time.Sleep(200 * time.Millisecond)
		}
		chordOf := func(kf keymap.Functions) key.Chord {
			for ch, f := range *keymap.ActiveMap {
				if f == kf {
					return ch
				}
			}
			return ""
		}
		for ch, kf := range *keymap.ActiveMap {
			if kf != keymap.Undo {
				continue
			}
			pt := bb.Min.Add(bb.Size().Div(2))
			src.MouseButton(events.MouseDown, events.Left, pt, 0)
			src.MouseButton(events.MouseUp, events.Left, pt, 0)
			time.Sleep(200 * time.Millisecond)
			press("x")
			press(chordOf(keymap.Accept))
			press(ch)
			b.AsyncLock()
			assert.Equal(t, "Laundry", tf.Text(), "undone with %s", ch)
			b.AsyncUnlock()
			press("y")
			b.AsyncLock()
			assert.Equal(t, "Laundryy", tf.Text(), "typing goes on in the text field, after %s", ch)
			tf.SetText("Laundry").SendChange()
			b.AsyncUnlock()
		}
	})
}

// TestTreeKeysFocusTable tests that the right arrow and enter keys on a
// list in the tree of lists put the focus in its table, but not on a group.
func TestTreeKeysFocusTable(t *testing.T) {
	b := core.NewBody()
	ls := testLists(t, b)
	b.AssertRender(t, "tree-keys-focus", func() {
		src := b.Scene.RenderWindow().SystemWindow.Events()
		for _, code := range []key.Codes{key.CodeRightArrow, key.CodeReturnEnter} {
			b.AsyncLock()
			chores := findList(ls, "Chores")
			ls.SetCurrent(chores)
			ls.tree.FindSyncNode(chores).SetFocus()
			b.AsyncUnlock()
			src.Key(events.KeyDown, 0, code, 0)
			src.Key(events.KeyUp, 0, code, 0)
			time.Sleep(300 * time.Millisecond)
			b.AsyncLock()
			assert.True(t, ls.table.ContainsFocus(), "the table has the focus, after %v", code)
			assert.False(t, ls.tree.ContainsFocus())
			b.AsyncUnlock()
		}
		b.AsyncLock()
		ls.tree.FindSyncNode(ls.Root.FindPath("Home")).SetFocus()
		b.AsyncUnlock()
		src.Key(events.KeyDown, 0, key.CodeRightArrow, 0)
		src.Key(events.KeyUp, 0, key.CodeRightArrow, 0)
		time.Sleep(300 * time.Millisecond)
		b.AsyncLock()
		defer b.AsyncUnlock()
		assert.True(t, ls.tree.ContainsFocus(), "not for a group")
	})
}

// TestSetBy tests that checking Done sets the date of Done on, if it
// is not set, only for the items that are checked then.
func TestSetBy(t *testing.T) {
	b := core.NewBody()
	ls := testLists(t, b)
	b.AssertRender(t, "set-by", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		chores := findList(ls, "Chores")
		ls.SetCurrent(chores)
		lt := ls.TypeOf(chores)
		done, doneOn := 1, len(lt.Fields)-1
		assert.Equal(t, "Done on", lt.Fields[doneOn].Name)
		item := func(i int) reflect.Value {
			return reflect.ValueOf(ls.table.Items).Elem().Index(i).Elem()
		}
		item(1).Field(done).SetBool(true) // Fix faucet, as by its switch
		ls.itemsChanged()

		y, m, d := time.Now().Date()
		today := time.Date(y, m, d, 0, 0, 0, 0, time.Local)
		assert.Equal(t, today, item(1).Field(doneOn).Interface(), "set when checked")
		assert.True(t, item(0).Field(doneOn).Interface().(time.Time).IsZero(), "not for Laundry, already done")

		earlier := today.AddDate(0, 0, -3)
		item(1).Field(doneOn).Set(reflect.ValueOf(earlier))
		item(1).Field(done).SetBool(false)
		ls.itemsChanged()
		item(1).Field(done).SetBool(true)
		ls.itemsChanged()
		assert.Equal(t, earlier, item(1).Field(doneOn).Interface(), "not when it is set")
	})
}

// TestFieldTooltips tests that the headers of the table of the fields of a
// list type have the docs of the fields as their tooltips, and that the
// table shows the indexes, for reordering the fields by dragging.
func TestFieldTooltips(t *testing.T) {
	b := core.NewBody()
	lt := standardTypes()[0]
	tb := core.NewTable(b).SetSlice(&lt.Fields).SetShowIndexes(true)
	b.AssertRender(t, "field-tooltips", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		var tips []string
		for _, n := range tb.Child(0).AsTree().Children { // the header
			if bt, ok := n.(*core.Button); ok {
				tips = append(tips, bt.Tooltip)
			}
		}
		require.Len(t, tips, 6)
		assert.Contains(t, tips[4], "Set by is, for a date field, the name of a check field")
	})
}

// TestTypeMoved tests that a list type that is moved in the library of
// list types, which makes a copy of it, is still the type of its lists.
func TestTypeMoved(t *testing.T) {
	ls := testLists(t, core.NewBody())
	snap := ls.snapshotTypes()
	shopping := *ls.TypeOf(findList(ls, "Groceries")) // a copy, as dragging makes
	ls.Types = append([]*ListType{&shopping}, slices.DeleteFunc(ls.Types, func(lt *ListType) bool { return lt.Name == "Shopping" })...)
	ls.typesEdited(snap)
	assert.Len(t, ls.Types, 5, "not kept again")
	assert.Same(t, &shopping, ls.TypeOf(findList(ls, "Groceries")))
}

// TestFilter tests filtering the items by the filter values of the fields
// of the type of a list, such as true for Done, with the Filter switch.
func TestFilter(t *testing.T) {
	b := core.NewBody()
	ls := testLists(t, b)
	b.AssertRender(t, "filter", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		chores := findList(ls, "Chores")
		ls.SetCurrent(chores)
		var sw *core.Switch
		ls.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
			if s, ok := cw.(*core.Switch); ok && s.Text == "Filter" {
				sw = s
			}
			return sw == nil
		})
		require.NotNil(t, sw, "a switch for a list type with filter values")
		assert.Equal(t, 2, ls.table.SliceSize)
		sw.SetChecked(true)
		sw.SendChange()
		assert.True(t, chores.Filtered)
		assert.Equal(t, 1, ls.table.SliceSize, "without Laundry, which is done")

		item := reflect.ValueOf(ls.table.Items).Elem().Index(1).Elem() // Fix faucet
		item.Field(1).SetBool(true)
		ls.itemsChanged()
		assert.Equal(t, 0, ls.table.SliceSize, "and now without Fix faucet")

		saved := NewLists(core.NewBody())
		require.NoError(t, saved.Open(ls.Dir))
		assert.True(t, findList(saved, "Chores").Filtered, "saved")
	})
}

// TestSettingsSplits tests that the proportions of the splits are saved
// in the settings, and applied.
func TestSettingsSplits(t *testing.T) {
	file := Settings.File
	Settings.File = filepath.Join(t.TempDir(), "settings.toml")
	defer func() { Settings.File = file }()
	b := core.NewBody()
	ls := testLists(t, b)
	b.AssertRender(t, "settings-splits", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		ls.splits().SetSplits(0.35, 0.65)
		ls.saveSplits()
		assert.InDeltaSlice(t, []float32{0.35, 0.65}, Settings.Splits, 0.001)
		_, err := os.Stat(Settings.File)
		assert.NoError(t, err, "saved")
		ls.splits().SetSplits(0.5, 0.5)
		ls.applySplits()
		assert.InDeltaSlice(t, []float32{0.35, 0.65}, ls.splits().Splits(), 0.001)
	})
}

// TestFocusPanels tests that the keys of the panels, such as Control+X o,
// move the focus between the tree of lists and the table of the current list.
func TestFocusPanels(t *testing.T) {
	b := core.NewBody()
	ls := testLists(t, b)
	b.AssertRender(t, "focus-panels", func() {
		src := b.Scene.RenderWindow().SystemWindow.Events()
		b.AsyncLock()
		chores := findList(ls, "Chores")
		ls.SetCurrent(chores)
		ls.tree.FindSyncNode(chores).SetFocus()
		b.AsyncUnlock()
		n := 0
		var seqs []string
		for seq, kf := range *keymap.ActiveMap {
			if kf == KeyNextPanel || kf == KeyPrevPanel {
				seqs = append(seqs, string(seq))
			}
		}
		slices.Sort(seqs) // the same order, for the same image
		for _, seq := range seqs {
			for _, ch := range strings.Fields(seq) {
				r, code, mods, err := key.Chord(ch).Decode()
				require.NoError(t, err)
				src.Key(events.KeyDown, r, code, mods)
				src.Key(events.KeyUp, r, code, mods)
				time.Sleep(100 * time.Millisecond)
			}
			time.Sleep(200 * time.Millisecond)
			n++
			b.AsyncLock()
			inTable := n%2 == 1
			assert.Equal(t, inTable, ls.table.ContainsFocus(), "the table has the focus after %v", seq)
			assert.Equal(t, !inTable, ls.tree.ContainsFocus(), "the tree has the focus after %v", seq)
			b.AsyncUnlock()
		}
		assert.Equal(t, 6, n)
	})
}
