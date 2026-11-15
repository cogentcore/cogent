// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lists

import (
	"encoding/json"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// menuButton calls the context menu of the tree of the given list, and
// clicks its button with the given text.
func menuButton(t *testing.T, ls *Lists, l *List, text string) {
	t.Helper()
	tr := ls.tree.FindSyncNode(l)
	require.NotNil(t, tr, l.Name)
	m := core.NewBody()
	ls.treeContextMenu(m.Scene, tr)
	bt := core.ButtonByText(m.Scene, text)
	require.NotNil(t, bt, text)
	bt.Send(events.Click)
}

// dialogOK sets the text of the text field of the dialog on top, if
// given, and clicks its button with the given text.
func dialogOK(t *testing.T, b *core.Body, text, button string) {
	t.Helper()
	d := core.TopScene(b)
	require.NotNil(t, d)
	if text != "" {
		var tf *core.TextField
		for w := range d.AllWidgets() {
			if x, ok := w.(*core.TextField); ok {
				tf = x
				break
			}
		}
		require.NotNil(t, tf)
		tf.SetText(text)
		tf.SendChange()
	}
	bt := core.ButtonByText(d, button)
	require.NotNil(t, bt, button)
	bt.Send(events.Click)
}

// TestTreeMenu tests the actions of the context menu of the tree of the
// lists: new lists and groups, renaming, duplicating, deleting, and
// copying, cutting, and pasting.
func TestTreeMenu(t *testing.T) {
	b := core.NewBody()
	ls := testLists(t, b)
	t.Cleanup(core.TrackWindows())
	core.RenderImage(b, core.Locked(b, func() {
		gr := findList(ls, "Groceries")
		ls.SetCurrent(gr)
		menuButton(t, ls, gr, "New list")
	}), core.Locked(b, func() {
		dialogOK(t, b, "Snacks", "Make")
		sn := ls.Root.FindPath("Home/Snacks")
		require.NotNil(t, sn, "after Groceries")
		assert.Equal(t, findList(ls, "Groceries").TypeName, sn.(*List).TypeName, "the type of the current list")
		assert.Equal(t, sn, ls.Current)
		menuButton(t, ls, findList(ls, "Snacks"), "New group")
	}), core.Locked(b, func() {
		dialogOK(t, b, "Projects", "Make")
		pr := ls.Root.FindPath("Home/Projects")
		require.NotNil(t, pr)
		assert.True(t, pr.(*List).IsGroup())
		menuButton(t, ls, pr.(*List), "New list")
	}), core.Locked(b, func() {
		dialogOK(t, b, "", "Make")
		assert.NotNil(t, ls.Root.FindPath("Home/Projects/New list"), "in the group")
		menuButton(t, ls, findList(ls, "Snacks"), "Rename")
	}), core.Locked(b, func() {
		dialogOK(t, b, "Chores", "Rename") // a name that is used
		assert.Nil(t, ls.Root.FindPath("Home/Snacks"))
		assert.NotEqual(t, "Chores", findList(ls, "Groceries").Name)
		var renamed *List
		for _, k := range ls.Root.FindPath("Home").AsTree().Children {
			if l := k.(*List); l.Name != "Groceries" && l.Name != "Chores" && !l.IsGroup() {
				renamed = l
			}
		}
		require.NotNil(t, renamed, "a unique name")
		assert.Contains(t, renamed.Name, "Chores")
		menuButton(t, ls, renamed, "Delete")
	}), core.Locked(b, func() {
		n := len(ls.Root.FindPath("Home").AsTree().Children)
		dialogOK(t, b, "", "Delete")
		assert.Len(t, ls.Root.FindPath("Home").AsTree().Children, n-1)
		menuButton(t, ls, findList(ls, "Groceries"), "Duplicate")
		menuButton(t, ls, findList(ls, "Groceries"), "Copy")
		menuButton(t, ls, findList(ls, "Chores"), "Paste")
		menuButton(t, ls, findList(ls, "Chores"), "Cut")
	}), core.Locked(b, func() {
		menuButton(t, ls, ls.Root, "Rename") // the root has no delete
		bt := core.ButtonByText(core.TopScene(b), "Cancel")
		require.NotNil(t, bt)
		bt.Send(events.Click)
	}))
}

// TestKeepUsedTypes tests that a list whose type is not in the library
// gets a type with the fields of its items, so that no values are lost.
func TestKeepUsedTypes(t *testing.T) {
	b := core.NewBody()
	ls := testLists(t, b)
	n := len(ls.Types)
	gone := &List{TypeName: "Gone", Items: []*Item{
		{Values: map[string]json.RawMessage{"A": []byte(`"a"`)}, Items: []*Item{{Values: map[string]json.RawMessage{"B": []byte("1"), "A": []byte(`"x"`)}}}},
	}}
	gone.SetName("Gone list")
	ls.Root.AddChild(gone)
	empty := &List{TypeName: "Empty"}
	empty.SetName("Empty list")
	ls.Root.AddChild(empty)
	ls.keepUsedTypes()
	require.Len(t, ls.Types, n+2)
	lt := ls.TypeOf(gone)
	require.NotNil(t, lt)
	var names []string
	for _, f := range lt.Fields {
		names = append(names, f.Name)
	}
	assert.ElementsMatch(t, []string{"A", "B"}, names)
	assert.Equal(t, "Item", ls.TypeOf(empty).Fields[0].Name)
	ls.keepUsedTypes() // again: no more
	assert.Len(t, ls.Types, n+2)
}

// TestTypeDialogs tests opening the dialogs for editing the list types,
// and syncing without a repository.
func TestTypeDialogs(t *testing.T) {
	b := core.NewBody()
	ls := testLists(t, b)
	t.Cleanup(core.TrackWindows())
	core.RenderImage(b, core.Locked(b, func() {
		ls.EditType(ls.Types[0])
		ls.EditTypes()
		ls.StartSync() // no repository: nothing to sync
		assert.True(t, ls.syncs.auto)
		theLists = ls
		Settings.Apply()
		theLists = nil
	}))
}
