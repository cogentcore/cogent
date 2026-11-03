// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lists

import (
	"fmt"
	"reflect"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
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
		tree.AddAt(p, "table", func(w *core.TreeTable) {
			w.ShowField = ls.mobileField // before the items, which initTable sets
			ls.initTable(w)
			w.SetRowButton(icons.Edit, "Edit the item, with all of its fields", ls.editItem)
		})
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

// mobileField returns whether to show the given field of the items in the
// mobile layout: the first field, which has the tree of the items, and the
// first check field of the type of the current list, such as Done.
func (ls *Lists) mobileField(field reflect.StructField) bool {
	lt := ls.TypeOf(ls.Current)
	if lt == nil || len(field.Index) != 1 {
		return true
	}
	i := field.Index[0]
	if i == 0 || i >= len(lt.Fields) { // the fields of the tree are hidden anyway
		return true
	}
	return i == lt.firstCheck()
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

// editItem opens the window of the form of the given item of the current
// list, a pointer to its struct, in the mobile layout, with all of its
// fields, whose edits are saved as they are made.
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
	ls.runStage(b)
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
