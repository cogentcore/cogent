// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"reflect"
	"slices"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/tree"
)

// typesSnapshot has the names of the list types and their fields before
// they are edited, for finding those that are renamed.
type typesSnapshot struct {
	types  map[string]*ListType
	fields map[*Field]string
}

// snapshotTypes returns a snapshot of the list types before editing.
func (ls *Lists) snapshotTypes() *typesSnapshot {
	snap := &typesSnapshot{types: map[string]*ListType{}, fields: map[*Field]string{}}
	for _, lt := range ls.Types {
		snap.types[lt.Name] = lt
		for _, f := range lt.Fields {
			snap.fields[f] = f.Name
		}
	}
	return snap
}

// typesEdited updates the lists after the list types are edited: the
// lists of renamed types get the new names, and their values of renamed
// fields stay with them. Types that are still used are not deleted.
func (ls *Lists) typesEdited(snap *typesSnapshot) {
	var kept []string
	ls.Root.WalkDown(func(n tree.Node) bool {
		l := n.(*List)
		lt := snap.types[l.TypeName]
		if l.IsGroup() || lt == nil {
			return tree.Continue
		}
		if !slices.Contains(ls.Types, lt) {
			// a type that is moved by dragging is a copy, with the same name
			if i := slices.IndexFunc(ls.Types, func(o *ListType) bool { return o.Name == lt.Name }); i >= 0 {
				lt = ls.Types[i]
			} else {
				ls.Types = append(ls.Types, lt)
				kept = append(kept, lt.Name)
			}
		}
		l.TypeName = lt.Name
		for _, f := range lt.Fields {
			if old, ok := snap.fields[f]; ok && old != f.Name {
				renameField(l.Items, old, f.Name)
				// the widths are by the names of the struct fields
				oldKey, newKey := goName(old, map[string]bool{}), goName(f.Name, map[string]bool{})
				if w, has := l.ColumnWidths[oldKey]; has {
					delete(l.ColumnWidths, oldKey)
					l.ColumnWidths[newKey] = w
				}
			}
		}
		if len(lt.Fields) == 0 {
			lt.Fields = []*Field{{Name: "Item"}}
		}
		l.values = reflect.Value{}
		return tree.Continue
	})
	if len(kept) > 0 {
		core.MessageSnackbar(ls, "Kept the list types that lists have: "+kept[0])
	}
	ls.saveChanges()
	ls.Update()
}

// fieldsDoc explains the fields of a list type.
const fieldsDoc = "The fields of the items are the columns of the lists. The first field is the tree of the items, where the categories, such as the aisles of a store, have the items in them."

// EditType edits the given list type, in a dialog.
func (ls *Lists) EditType(lt *ListType) {
	snap := ls.snapshotTypes()
	d := core.NewBody("Edit list type")
	core.Bind(&lt.Name, core.NewTextField(d)).SetTooltip("The name of the list type")
	core.NewText(d).SetType(core.TextSupporting).SetText(fieldsDoc)
	// the indexes are for dragging the fields to reorder them
	core.NewTable(d).SetSlice(&lt.Fields).SetShowIndexes(true).SetDragMoves(true)
	d.OnClose(func(e events.Event) {
		ls.typesEdited(snap)
	})
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddOK(bar).SetText("Done")
	})
	d.RunWindowDialog(ls)
}

// EditTypes edits the library of list types, in a dialog.
func (ls *Lists) EditTypes() {
	snap := ls.snapshotTypes()
	d := core.NewBody("List types")
	core.NewText(d).SetType(core.TextSupporting).SetText("The types of the lists, with the fields of their items. " + fieldsDoc)
	// the indexes are for dragging the types to reorder them,
	// the first of which is for new lists
	core.NewTable(d).SetSlice(&ls.Types).SetShowIndexes(true).SetDragMoves(true)
	d.OnClose(func(e events.Event) {
		ls.typesEdited(snap)
	})
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddOK(bar).SetText("Done")
	})
	d.RunWindowDialog(ls)
}
