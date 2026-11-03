// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	// In tests, the types of package main have its import path instead of
	// main, under which typegen registers them, so they are added under it.
	pkg := reflect.TypeFor[Lists]().PkgPath()
	for name, typ := range types.Types {
		if strings.HasPrefix(name, "main.") {
			types.Types[pkg+strings.TrimPrefix(name, "main")] = typ
		}
	}
}

// testLists returns the app with a copy of the sample lists,
// in the given body.
func testLists(t *testing.T, b *core.Body) *Lists {
	dir := t.TempDir()
	data, err := os.ReadFile(filepath.Join("testdata", "sample", listsFile))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, listsFile), data, 0o644))
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
	assert.Equal(t, 7, st.NumField())
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
