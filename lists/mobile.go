// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lists

import (
	"fmt"
	"reflect"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
)

// The mobile layout (see [Lists.Mobile]) has three levels, each in a full
// window, in turn: the tree of lists, which is the body of the app, the
// items of the current list, which opens when it is tapped, with only the
// tree of the items and the first check field, such as Done, and a button
// to edit each item, which opens its form, with all of its fields.

// runStage runs a full-window stage of the given body in the window of the
// lists, with a back button and the title of the body, as on mobile
// platforms, where they always are, so that the mobile layout is the same
// on desktop platforms, for trying it.
func (ls *Lists) runStage(b *core.Body) {
	b.NewWindow().SetNewWindow(false).SetFullWindow(true).SetBackButton(true).
		SetDisplayTitle(true).SetContext(ls).Run()
}

// openItems opens the window of the items of the current list, in the
// mobile layout, with the tools for the items in its top bar, if it is a
// list and the window is not open already.
func (ls *Lists) openItems() {
	l := ls.Current
	if !ls.Mobile || l == nil || l.IsGroup() || ls.items != nil {
		return
	}
	b := core.NewBody(l.Name)
	ls.items = b
	b.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
	})
	// the filter switch is in the title line, after the title
	b.AddTopBar(func(bar *core.Frame) {
		core.NewStretch(bar)
		core.NewFrame(bar).Maker(ls.makeFilterSwitch)
	})
	// the tools are at the bottom, where they have the width of the
	// screen, with the others in the overflow menu
	b.AddBottomBar(func(bar *core.Frame) {
		core.NewToolbar(bar).Maker(ls.makeItemTools)
	})
	b.Maker(func(p *tree.Plan) {
		if ls.Current == nil || ls.Current.IsGroup() {
			return
		}
		ls.makeListHeader(p)
		tree.AddAt(p, "table", ls.initTable)
	})
	b.Updater(func() {
		if ls.Current != nil {
			b.SetTitle(ls.Current.Name)
		}
	})
	b.OnClose(func(e events.Event) {
		ls.items = nil
		ls.table = nil
	})
	ls.runStage(b)
}

// tableField returns whether to show the given field of the items of the
// current list in the table (see [ListType.inTable]).
func (ls *Lists) tableField(field reflect.StructField) bool {
	lt := ls.TypeOf(ls.Current)
	if lt == nil || len(field.Index) != 1 || field.Index[0] >= len(lt.Fields) {
		return true // the fields of the tree are hidden anyway
	}
	return lt.inTable(field.Index[0], ls.Mobile)
}

// inTable returns whether the field with the given index is in the table
// of the items, in the mobile layout if mobile is true: the first field,
// which has the tree of the items, and the others that are not detail
// fields (see [Field.Detail]), and in the mobile layout, only the first
// check field, such as Done. The others are only in the form of an item
// (see [Lists.editItem]).
func (lt *ListType) inTable(i int, mobile bool) bool {
	switch {
	case i == 0:
		return true
	case lt.Fields[i].Detail:
		return false
	}
	return !mobile || i == lt.firstCheck()
}

// hasFormFields returns whether the list type has fields that are only in
// the form of an item, and not in the table, in the mobile layout if mobile
// is true, for which the items have edit buttons.
func (lt *ListType) hasFormFields(mobile bool) bool {
	for i := range lt.Fields {
		if !lt.inTable(i, mobile) {
			return true
		}
	}
	return false
}

// firstCheck returns the index of the first check field, or -1 if none.
func (lt *ListType) firstCheck() int {
	for i, f := range lt.Fields {
		if f.Type == Check {
			return i
		}
	}
	return -1
}

// editItem opens the form of the given item of the current list, a pointer
// to its struct, with all of its fields, whose edits are saved as they are
// made: in its window in the mobile layout, and in a dialog otherwise.
func (ls *Lists) editItem(item any) {
	if ls.form != nil {
		return
	}
	title := "Item"
	if v := reflect.ValueOf(item).Elem(); v.NumField() > 0 {
		if s := fmt.Sprint(v.Field(0).Interface()); s != "" {
			title = s
		}
	}
	b := core.NewBody(title)
	ls.form = b
	core.NewForm(b).SetStruct(item).OnChange(func(e events.Event) {
		ls.itemsChanged()
	})
	b.OnClose(func(e events.Event) {
		ls.form = nil
		if ls.table != nil { // with the edited values
			ls.table.Update()
		}
	})
	if ls.Mobile {
		ls.runStage(b)
		return
	}
	b.AddBottomBar(func(bar *core.Frame) {
		b.AddOK(bar).SetText("Done")
	})
	b.RunDialog(ls)
}

// updateItems updates the window of the items of the current list, and
// closes the form of an item, in the mobile layout, after the lists
// change other than in the window, as by undo or a sync: the form has an
// item that may no longer be in the list.
func (ls *Lists) updateItems() {
	if ls.form != nil {
		ls.form.Scene.Close()
	}
	if ls.items == nil {
		return
	}
	if ls.Current == nil || ls.Current.IsGroup() {
		ls.items.Scene.Close()
		return
	}
	ls.items.Scene.Update() // with the filter switch in the top bar
}
