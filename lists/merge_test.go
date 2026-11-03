// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lists

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// item returns a new item with the given ID and values of the
// fields Item and Done, as JSON, and items in it.
func item(id, name, done string, items ...*Item) *Item {
	it := &Item{ID: id, Values: map[string]json.RawMessage{"Item": json.RawMessage(`"` + name + `"`)}, Items: items}
	if done != "" {
		it.Values["Done"] = json.RawMessage(done)
	}
	return it
}

// listJSON returns the file of a list with the given items.
func listJSON(items ...*Item) []byte {
	return marshal(&listFile{Name: "L", Type: "To do", Items: items})
}

// itemsOf returns the items of the given file of a list, as text: the
// IDs, names and done values, with the items in them in parentheses.
func itemsOf(t *testing.T, b []byte) string {
	var lf listFile
	assert.NoError(t, json.Unmarshal(b, &lf))
	var str func(items []*Item) string
	str = func(items []*Item) string {
		var s []string
		for _, it := range items {
			v := it.ID + ":" + strings.Trim(string(it.Values["Item"]), `"`)
			if d := it.Values["Done"]; d != nil {
				v += "=" + string(d)
			}
			if len(it.Items) > 0 {
				v += "(" + str(it.Items) + ")"
			}
			s = append(s, v)
		}
		return strings.Join(s, " ")
	}
	return str(lf.Items)
}

// mergeListFiles returns the items of the merge of the given files of a
// list, as text (see itemsOf).
func mergeListFiles(t *testing.T, base, ours, theirs []byte, oursWins bool) string {
	return itemsOf(t, marshal(mergeList(base, ours, theirs, "L", oursWins)))
}

func TestMergeItems(t *testing.T) {
	base := listJSON(item("a", "Apples", "false"), item("b", "Bread", "false"), item("c", "Cheese", ""))
	tests := []struct {
		name, want   string
		ours, theirs []byte
		oursWins     bool
	}{
		{"different fields of an item",
			"a:Green apples=true b:Bread=false c:Cheese",
			listJSON(item("a", "Green apples", "false"), item("b", "Bread", "false"), item("c", "Cheese", "")),
			listJSON(item("a", "Apples", "true"), item("b", "Bread", "false"), item("c", "Cheese", "")), false},
		{"the same field, theirs",
			"a:Red apples=false b:Bread=false c:Cheese",
			listJSON(item("a", "Green apples", "false"), item("b", "Bread", "false"), item("c", "Cheese", "")),
			listJSON(item("a", "Red apples", "false"), item("b", "Bread", "false"), item("c", "Cheese", "")), false},
		{"the same field, ours",
			"a:Green apples=false b:Bread=false c:Cheese",
			listJSON(item("a", "Green apples", "false"), item("b", "Bread", "false"), item("c", "Cheese", "")),
			listJSON(item("a", "Red apples", "false"), item("b", "Bread", "false"), item("c", "Cheese", "")), true},
		{"added on both sides, after the items before them",
			"a:Apples=false o:Oats b:Bread=false c:Cheese t:Tea",
			listJSON(item("a", "Apples", "false"), item("o", "Oats", ""), item("b", "Bread", "false"), item("c", "Cheese", "")),
			listJSON(item("a", "Apples", "false"), item("b", "Bread", "false"), item("c", "Cheese", ""), item("t", "Tea", "")), false},
		{"deleted on one side, and changed on the other",
			"a:Apples=true c:Cheese",
			listJSON(item("a", "Apples", "true"), item("b", "Bread", "false"), item("c", "Cheese", "")),
			listJSON(item("a", "Apples", "false"), item("c", "Cheese", "")), false},
		{"deleted on one side, and changed on the other, which is kept",
			"a:Apples=false b:Bread=true c:Cheese",
			listJSON(item("a", "Apples", "false"), item("b", "Bread", "true"), item("c", "Cheese", "")),
			listJSON(item("a", "Apples", "false"), item("c", "Cheese", "")), false},
		{"reordered on one side, and changed on the other",
			"c:Cheese a:Apples=true b:Bread=false",
			listJSON(item("a", "Apples", "true"), item("b", "Bread", "false"), item("c", "Cheese", "")),
			listJSON(item("c", "Cheese", ""), item("a", "Apples", "false"), item("b", "Bread", "false")), false},
		{"added in an item, which is deleted on the other side, at the top",
			"a:Apples=false c:Cheese w:Wheat",
			listJSON(item("a", "Apples", "false"), item("b", "Bread", "false", item("w", "Wheat", "")), item("c", "Cheese", "")),
			listJSON(item("a", "Apples", "false"), item("c", "Cheese", "")), false},
		{"moved into each other, which is undone for one",
			"b:Bread=false(a:Apples=false) c:Cheese",
			listJSON(item("b", "Bread", "false", item("a", "Apples", "false")), item("c", "Cheese", "")),
			listJSON(item("a", "Apples", "false", item("b", "Bread", "false")), item("c", "Cheese", "")), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, mergeListFiles(t, base, tt.ours, tt.theirs, tt.oursWins))
			// the same from the other side, in which only the order of
			// the items added on both sides differs
			if !strings.HasPrefix(tt.name, "added on both") {
				assert.Equal(t, tt.want, mergeListFiles(t, base, tt.theirs, tt.ours, !tt.oursWins), "swapped")
			}
		})
	}
}

func TestMergeListSettings(t *testing.T) {
	file := func(sort string, widths map[string]float32, filtered bool) []byte {
		return marshal(&listFile{Name: "L", Type: "To do", Sort: sort, ColumnWidths: widths, Filtered: filtered, Items: []*Item{item("a", "A", "")}})
	}
	lf := mergeList(file("", nil, false), file("Item", nil, false), file("", map[string]float32{"Item": 90}, true), "New name", false)
	assert.Equal(t, "Item", lf.Sort)
	assert.Equal(t, map[string]float32{"Item": 90}, lf.ColumnWidths)
	assert.True(t, lf.Filtered)
	assert.Equal(t, "New name", lf.Name)
	assert.Equal(t, "To do", lf.Type)
	assert.Len(t, lf.Items, 1)
}

// listsFiles returns the files of the lists with the given tree of lists,
// with a file for each list with the given items, by its ID.
func listsFiles(lists []*treeNode, items map[string][]*Item, types ...*ListType) map[string][]byte {
	files := map[string][]byte{treeFile: marshal(&treeNode{Name: "Lists", Group: true, Lists: lists}), typesFile: marshal(types)}
	var add func(tns []*treeNode)
	add = func(tns []*treeNode) {
		for _, tn := range tns {
			if !tn.Group {
				files[listPath(tn.ID)] = marshal(&listFile{Name: tn.Name, Type: "To do", Items: items[tn.ID]})
			}
			add(tn.Lists)
		}
	}
	add(lists)
	return files
}

// treeOf returns the tree of lists of the given files, as text.
func treeOf(t *testing.T, files map[string][]byte) string {
	var root treeNode
	assert.NoError(t, json.Unmarshal(files[treeFile], &root))
	var str func(tns []*treeNode) string
	str = func(tns []*treeNode) string {
		var s []string
		for _, tn := range tns {
			v := tn.Name
			if tn.Group {
				v += "(" + str(tn.Lists) + ")"
			}
			s = append(s, v)
		}
		return strings.Join(s, " ")
	}
	return str(root.Lists)
}

func TestMergeFiles(t *testing.T) {
	home := func(lists ...*treeNode) *treeNode { return &treeNode{ID: "h", Name: "Home", Group: true, Lists: lists} }
	gr := &treeNode{ID: "g", Name: "Groceries"}
	ch := &treeNode{ID: "c", Name: "Chores"}
	todo := &ListType{Name: "To do", Fields: []*Field{{Name: "Item"}}}
	shop := &ListType{Name: "Shopping", Fields: []*Field{{Name: "Item"}}}
	books := &ListType{Name: "Books", Fields: []*Field{{Name: "Title"}}}
	items := map[string][]*Item{"g": {item("a", "Apples", "")}, "c": {item("l", "Laundry", "")}}
	base := listsFiles([]*treeNode{home(gr, ch)}, items, todo)

	// a list is renamed and moved on one side, and deleted on the other,
	// which also edits the other list; with a type added on each side
	ours := listsFiles([]*treeNode{home(ch), {ID: "g", Name: "Food"}}, items, todo, shop)
	edited := map[string][]*Item{"c": {item("l", "Laundry", "true")}}
	theirs := listsFiles([]*treeNode{home(ch)}, edited, todo, books)
	merged := mergeFiles(base, ours, theirs, false)
	assert.Equal(t, "Home(Chores) Food", treeOf(t, merged), "the changed list is kept")
	assert.Equal(t, "l:Laundry=true", itemsOf(t, merged[listPath("c")]))
	assert.Equal(t, "a:Apples", itemsOf(t, merged[listPath("g")]))
	var types []*ListType
	assert.NoError(t, json.Unmarshal(merged[typesFile], &types))
	assert.Len(t, types, 3)

	// a list that is deleted, and not changed on the other side,
	// is deleted, with its file
	theirs = listsFiles([]*treeNode{home(gr)}, items, todo)
	merged = mergeFiles(base, base, theirs, false)
	assert.Equal(t, "Home(Groceries)", treeOf(t, merged))
	assert.NotContains(t, merged, listPath("c"))

	// no changes on one side are the files of the other, exactly
	assert.Equal(t, theirs, mergeFiles(base, base, theirs, true))
	assert.Equal(t, ours, mergeFiles(base, ours, base, false))
	assert.Equal(t, ours, mergeFiles(ours, ours, ours, false))
}

// TestMergeTypes tests merging the list types by their fields and the
// settings of the fields, also without a base, as on the first sync of
// a device whose types do not have the newer settings.
func TestMergeTypes(t *testing.T) {
	old := []*ListType{{Name: "To do", Fields: []*Field{{Name: "Task"}, {Name: "Done", Type: Check}, {Name: "Done on", Type: Date}}}}
	newer := []*ListType{{Name: "To do", Fields: []*Field{{Name: "Task"}, {Name: "Done", Type: Check, Filter: "true"}, {Name: "Done on", Type: Date, SetBy: "Done"}}}}
	typesOf := func(b []byte) []*ListType {
		var types []*ListType
		assert.NoError(t, json.Unmarshal(b, &types))
		return types
	}
	merged := mergeFiles(nil, map[string][]byte{typesFile: marshal(old)}, map[string][]byte{typesFile: marshal(newer)}, true)
	assert.Equal(t, newer, typesOf(merged[typesFile]), "the settings that are set win")

	// different settings of a field, and a field added, on each side
	ours := []*ListType{{Name: "To do", Fields: []*Field{{Name: "Task", Tags: `width:"30"`}, {Name: "Done", Type: Check, Filter: "true"}, {Name: "Done on", Type: Date, SetBy: "Done"}, {Name: "Notes"}}}}
	theirs := []*ListType{{Name: "To do", Fields: []*Field{{Name: "Task", Default: "New"}, {Name: "Done", Type: Check, Filter: "false"}, {Name: "Done on", Type: Date, SetBy: "Done"}}}, {Name: "Books", Fields: []*Field{{Name: "Title"}}}}
	merged = mergeFiles(map[string][]byte{typesFile: marshal(newer)}, map[string][]byte{typesFile: marshal(ours)}, map[string][]byte{typesFile: marshal(theirs)}, false)
	want := []*ListType{{Name: "To do", Fields: []*Field{{Name: "Task", Default: "New", Tags: `width:"30"`}, {Name: "Done", Type: Check, Filter: "false"}, {Name: "Done on", Type: Date, SetBy: "Done"}, {Name: "Notes"}}}, {Name: "Books", Fields: []*Field{{Name: "Title"}}}}
	assert.Equal(t, want, typesOf(merged[typesFile]))
}
