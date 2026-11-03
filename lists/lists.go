// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lists

import (
	"fmt"
	"image"
	"reflect"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/types"
	"cogentcore.org/core/undo"
)

// List is a list of items, or a group of lists, in the tree of lists.
type List struct {
	tree.NodeBase

	// ID is the unique identifier of the list or group, which stays the
	// same when it is renamed or moved, for syncing (see [Lists.Sync]),
	// and names the file of a list (see [listPath]).
	ID string `display:"-"`

	// Group is whether it is a group of lists, instead of a list.
	Group bool

	// TypeName is the name of the type of the list, which is set to that
	// of the current list for a new list (see [Lists.newLists]).
	TypeName string

	// Items are the saved items of the list.
	Items []*Item `display:"-"`

	// ColumnWidths are the widths of the columns of the list that are set,
	// by the names of the struct fields (see [core.Table.ColumnWidths]).
	ColumnWidths map[string]float32 `display:"-"`

	// Sort is the column the items are sorted by, if any, with its
	// direction (see [core.Table.SortFieldName]).
	Sort string `display:"-"`

	// Filtered is whether the items with the filter values of the fields
	// of the type of the list are hidden (see [Field.Filter]).
	Filtered bool `display:"-"`

	// HideGroupValues is whether the items with items in them, such as
	// categories, show only their first field
	// (see [core.TreeTable.HideGroupValues]).
	HideGroupValues bool `display:"-"`

	// undos is the undo stack of the edits of the list (see [List.undoStack]).
	undos *undo.Stack

	// undoState is the state of the list for undo after the last edit,
	// which is saved as the state before the next one.
	undoState []string

	// checks are the states of the check fields of the items that set date
	// fields, from the last edit, for finding those that are checked.
	checks map[checkKey]bool

	// values is the pointer to the slice of pointers to the items as
	// structs of the type of the list, for the [core.TreeTable], or
	// invalid if they are not made yet.
	values reflect.Value
}

// IsGroup returns whether the list is a group of lists.
func (ls *List) IsGroup() bool { return ls.Group }

// Lists is the app: a tree of lists, in groups, and the items of the
// current list, in categories.
type Lists struct {
	core.Frame

	// Dir is the directory of the files of the lists and the list types.
	Dir string `set:"-"`

	// Root is the root group of the tree of lists.
	Root *List `set:"-"`

	// Types are the list types, the library of the types of the lists.
	Types []*ListType `set:"-"`

	// Current is the current list, whose items are shown, or nil.
	Current *List `set:"-"`

	// Mobile is whether the lists have the layout for small screens, as
	// of phones, in which the tree of lists, the items of the current list,
	// and an item that is edited are each in a full window, in turn, with
	// back buttons (see [Lists.openItems]), instead of side by side.
	Mobile bool

	// tree is the tree of lists.
	tree *core.Tree

	// table is the tree table of the items of the current list.
	table *core.TreeTable

	// tableList is the list whose column widths and sort the table has.
	tableList *List

	// items is the body of the items of the current list,
	// in the mobile layout, while it is open (see [Lists.openItems]).
	items *core.Body

	// form is the body of the form of the item that is edited,
	// in the mobile layout, while it is open (see [Lists.editItem]).
	form *core.Body

	// syncs is the state of the syncing of the lists (see [Lists.Sync]).
	syncs syncer

	// keySeq1 is the first key of a key sequence being entered,
	// such as Control+X (see [Lists.listsKeys]).
	keySeq1 key.Chord
}

// saveChanges saves after a change, reporting an error.
func (ls *Lists) saveChanges() {
	if err := ls.Save(); err != nil {
		core.ErrorSnackbar(ls, err, "Error saving lists")
	}
	ls.edited()
}

// TypeOf returns the type of the given list, or nil for a group.
func (ls *Lists) TypeOf(l *List) *ListType {
	if l.IsGroup() {
		return nil
	}
	for _, lt := range ls.Types {
		if lt.Name == l.TypeName {
			return lt
		}
	}
	return nil
}

// keepUsedTypes adds list types for the lists whose types are not in the
// library, with the fields of their items, so that no values are lost.
func (ls *Lists) keepUsedTypes() {
	ls.Root.WalkDown(func(n tree.Node) bool {
		l := n.(*List)
		if l.IsGroup() || ls.TypeOf(l) != nil {
			return tree.Continue
		}
		lt := &ListType{Name: l.TypeName}
		seen := map[string]bool{}
		var addFields func(items []*Item)
		addFields = func(items []*Item) {
			for _, it := range items {
				for name := range it.Values {
					if !seen[name] {
						seen[name] = true
						lt.Fields = append(lt.Fields, &Field{Name: name})
					}
				}
				addFields(it.Items)
			}
		}
		addFields(l.Items)
		if len(lt.Fields) == 0 {
			lt.Fields = []*Field{{Name: "Item"}}
		}
		ls.Types = append(ls.Types, lt)
		return tree.Continue
	})
}

// values returns the pointer to the slice of pointers to the items of the
// given list as structs of its type, making them from its saved items if
// needed, for the [core.TreeTable].
func (ls *Lists) values(l *List) any {
	lt := ls.TypeOf(l)
	if l.values.IsValid() && l.values.Type().Elem().Elem().Elem() == lt.StructType() {
		return l.values.Interface()
	}
	initUndo(l)
	l.values = lt.newItems(l.Items)
	l.checks = lt.checkStates(l.values)
	return l.values.Interface()
}

// itemsChanged saves the items of the current list after they change.
func (ls *Lists) itemsChanged() {
	if l := ls.Current; l != nil && l.values.IsValid() {
		lt := ls.TypeOf(l)
		// with the dates set, and the items filtered by their new values
		if (lt.setDates(l.values, l.checks) || l.Filtered) && ls.table != nil {
			ls.table.Update()
		}
		l.checks = lt.checkStates(l.values)
		l.Items = lt.savedItems(l.values)
		saveUndo(l, "edit")
	}
	ls.saveChanges()
}

// SetCurrent sets the current list, whose items are shown.
func (ls *Lists) SetCurrent(l *List) {
	if l == ls.Current {
		return
	}
	ls.Current = l
	if tt := ls.table; tt != nil { // which keeps its view for items of the same type
		tt.SelectedIndex = -1
		tt.ResetSelectedIndexes()
		tt.StartIndex = 0
	}
	ls.Update()
}

// SetType sets the type of the current list, keeping the values
// of the fields with the same names.
func (ls *Lists) SetType(lt *ListType) {
	l := ls.Current
	if l == nil || l.IsGroup() || lt == nil || lt.Name == l.TypeName {
		return
	}
	initUndo(l)
	l.TypeName = lt.Name
	l.values = reflect.Value{}
	saveUndo(l, "type change to "+lt.Name)
	ls.saveChanges()
	ls.Update()
	ls.updateItems()
}

func (ls *Lists) Init() {
	ls.Frame.Init()
	ls.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
	})
	ls.OnFirst(events.KeyChord, ls.listsKeys)
	ls.OnFirst(events.MouseDown, func(e events.Event) {
		ls.syncIfIdle()
	})
	// all of the keys of undo and redo, which the widgets within
	// do not handle, such as a text field after its edits are accepted
	ls.OnFinal(events.KeyChord, func(e events.Event) {
		switch keymap.Of(e.KeyChord()) {
		case keymap.Undo:
			ls.Undo()
			e.SetHandled()
		case keymap.Redo:
			ls.Redo()
			e.SetHandled()
		}
	})
	ls.OnShow(func(e events.Event) {
		ls.applySplits()
	})
	// the layout is made when the plan is, after Mobile is set
	ls.Maker(func(p *tree.Plan) {
		if ls.Mobile {
			tree.AddAt(p, "tree-frame", ls.initTreeFrame)
			return
		}
		tree.AddAt(p, "splits", func(w *core.Splits) {
			w.SetSplits(0.2, 0.8)
			w.OnChange(func(e events.Event) {
				ls.saveSplits()
			})
			tree.AddChild(w, ls.initTreeFrame)
			tree.AddChild(w, func(w *core.Frame) {
				w.Styler(func(s *styles.Style) {
					s.Direction = styles.Column
					s.Grow.Set(1, 1)
				})
				w.Maker(ls.makeCurrent)
			})
		})
	})
}

// initTreeFrame initializes the frame of the tree of lists.
func (ls *Lists) initTreeFrame(w *core.Frame) {
	w.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Overflow.Set(styles.OverflowAuto)
		if ls.Mobile {
			s.Grow.Set(1, 1)
		}
	})
	tree.AddChild(w, func(w *core.Tree) {
		ls.tree = w
		w.NewNodeType = types.For[List]() // groups are made with NewList
		w.SetTreeInit(ls.initTree)
		w.OnSelect(func(e events.Event) {
			if sel := w.GetSelectedNodes(); len(sel) == 1 {
				if l, ok := sel[0].AsCoreTree().SyncNode.(*List); ok {
					ls.SetCurrent(l)
				}
			}
		})
		w.OnChange(func(e events.Event) {
			ls.treeChanged()
		})
		w.Updater(func() {
			if ls.Root != nil && w.SyncNode != ls.Root {
				w.SyncTree(ls.Root)
			}
		})
	})
}

// makeCurrent makes the widgets for the current list.
func (ls *Lists) makeCurrent(p *tree.Plan) {
	l := ls.Current
	if l == nil || l.IsGroup() {
		ls.table = nil
		tree.Add(p, func(w *core.Text) {
			w.Updater(func() {
				if l != nil {
					w.SetText(l.Name + " is a group of lists. Select a list, or make a new one.")
				} else {
					w.SetText("Select a list, or make a new one.")
				}
			})
		})
		return
	}
	ls.makeListHeader(p)
	tree.AddAt(p, "table", func(w *core.TreeTable) {
		ls.initTable(w)
	})
}

// makeListHeader adds the header of the current list to the given plan,
// with its title and the filter switch, which are in the top bar of the
// window of the items in the mobile layout (see [Lists.openItems]). The
// other settings of the list are in its settings (see [Lists.listSettings]).
func (ls *Lists) makeListHeader(p *tree.Plan) {
	if ls.Mobile {
		return
	}
	tree.AddAt(p, "header", func(w *core.Frame) {
		w.Styler(func(s *styles.Style) {
			s.Align.Items = styles.Center
			s.Gap.X.Em(1)
		})
		tree.AddChild(w, func(w *core.Text) {
			w.SetType(core.TextHeadlineSmall)
			w.Styler(func(s *styles.Style) {
				s.SetTextWrap(false)
			})
			w.Updater(func() {
				w.SetText(ls.Current.Name)
			})
		})
		w.Maker(ls.makeFilterSwitch)
	})
}

// makeFilterSwitch adds the switch for filtering the items of the current
// list to the given plan, for a list type with filter values.
func (ls *Lists) makeFilterSwitch(p *tree.Plan) {
	if lt := ls.TypeOf(ls.Current); lt == nil || !lt.hasFilter() {
		return
	}
	tree.AddAt(p, "filter", func(w *core.Switch) {
		w.SetText("Filter").SetTooltip("Whether to hide the items with the filter values of the fields of the type of the list, such as those that are done")
		w.OnChange(func(e events.Event) {
			ls.Current.Filtered = w.IsChecked()
			if ls.table != nil {
				ls.table.Update()
			}
			ls.saveChanges()
		})
		w.Updater(func() {
			w.SetChecked(ls.Current.Filtered)
		})
	})
}

// listSettings edits the settings of the current list in a dialog, with
// the given context, such as the button for it, returning its body, or nil
// for no list.
func (ls *Lists) listSettings(ctx core.Widget) *core.Body {
	l := ls.Current
	if l == nil || l.IsGroup() {
		return nil
	}
	d := core.NewBody(l.Name + " settings")
	ch := core.NewChooser(d)
	ch.SetTooltip("The type of the list, which has the fields of its items")
	for _, lt := range ls.Types {
		ch.Items = append(ch.Items, core.ChooserItem{Value: lt})
	}
	ch.SetCurrentValue(ls.TypeOf(l))
	ch.OnChange(func(e events.Event) {
		ls.SetType(ch.CurrentItem.Value.(*ListType))
	})
	sw := core.NewSwitch(d).SetText("Group values").SetChecked(!l.HideGroupValues)
	sw.SetTooltip("Whether to show the values of the other fields for the items that have items in them, such as categories")
	sw.OnChange(func(e events.Event) {
		l.HideGroupValues = !sw.IsChecked()
		if ls.table != nil {
			ls.table.Update()
		}
		ls.saveChanges()
	})
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddOK(bar).SetText("Done")
	})
	d.RunDialog(ctx)
	return d
}

// initTable initializes the tree table of the items of the current list.
func (ls *Lists) initTable(w *core.TreeTable) {
	ls.table = w
	ls.tableList = nil   // a new table
	w.SetDragMoves(true) // reordering the items, unlike the lists
	// the mobile layout fits the width of the screen, without the column
	// widths, which are those of this device, and not synced
	w.FitWidth = ls.Mobile
	w.OnChange(func(e events.Event) {
		if !ls.Mobile {
			ls.Current.ColumnWidths = w.ColumnWidths
		}
		ls.Current.Sort = w.SortFieldName()
		ls.itemsChanged()
	})
	w.Updater(func() {
		w.SetItems(ls.values(ls.Current))
		w.HideGroupValues = ls.Current.HideGroupValues
		w.ShowItem = nil
		if lt := ls.TypeOf(ls.Current); ls.Current.Filtered && lt.hasFilter() {
			w.ShowItem = lt.showItem
		}
		if ls.tableList != ls.Current { // the table has those of the list after
			ls.tableList = ls.Current
			if !ls.Mobile {
				w.ColumnWidths = ls.Current.ColumnWidths
			}
			w.SetSortFieldName(ls.Current.Sort)
		}
	})
}

// treeChanged updates after the tree of lists changes.
func (ls *Lists) treeChanged() {
	if l := ls.Current; l != nil && l != ls.Root && l.Parent == nil { // deleted
		ls.Current = nil
		ls.Update()
	}
	added := ls.newLists()
	if uniqueNames(ls.Root) || added {
		ls.tree.Resync()
	}
	if added {
		ls.focusTable() // after the new list is selected, and current
	}
	ls.saveChanges()
}

// focusTable puts the focus in the table of the items of the current list,
// after the next render, on its selected item, or else its first item, or
// on the table if it has no items, so that the keys for adding items go to it.
func (ls *Lists) focusTable() {
	if ls.Mobile {
		ls.openItems() // which has the table
	}
	ls.Defer(func() {
		tt := ls.table
		if tt == nil {
			return
		}
		switch {
		case tt.SelectedIndex >= 0 && tt.SelectedIndex < tt.SliceSize:
			tt.IndexGrabFocus(tt.SelectedIndex)
		case tt.SliceSize > 0:
			tt.SelectIndexEvent(0, events.SelectOne) // which focuses it
		default:
			tt.SetFocus()
		}
	})
	ls.NeedsRender() // for the deferred focus
}

// setUniqueName sets the name of the given list to the given name, with
// a number added if another list in its group has it, as the names of the
// lists in a group must be unique, for the tree of lists.
func setUniqueName(l *List, name string) {
	if l.Parent == nil {
		l.SetName(name)
		return
	}
	taken := func(nm string) bool {
		for _, c := range l.Parent.AsTree().Children {
			if c.AsTree() != l.AsTree() && c.AsTree().Name == nm {
				return true
			}
		}
		return false
	}
	nm := name
	for i := 2; taken(nm); i++ {
		nm = fmt.Sprintf("%s %d", name, i)
	}
	l.SetName(nm)
}

// uniqueNames makes the names of the lists in each group of the given
// list unique (see [setUniqueName]), returning whether any changed.
func uniqueNames(l *List) bool {
	changed := false
	for i, c := range l.Children {
		cl := c.(*List)
		for _, o := range l.Children[:i] {
			if o.AsTree().Name == cl.Name {
				setUniqueName(cl, cl.Name)
				changed = true
				break
			}
		}
		changed = uniqueNames(cl) || changed
	}
	return changed
}

// newLists sets the type and name of the new lists, added by inserting
// them in the tree, which have no type, to those of the current list,
// or the first list type, returning whether there are any.
func (ls *Lists) newLists() bool {
	typ := ls.Types[0].Name
	if cur := ls.Current; cur != nil && !cur.IsGroup() {
		typ = cur.TypeName
	}
	found := false
	ls.Root.WalkDown(func(n tree.Node) bool {
		if l := n.(*List); !l.IsGroup() && l.TypeName == "" {
			l.TypeName = typ
			setUniqueName(l, "New list")
			found = true
		}
		return tree.Continue
	})
	return found
}

// initTree initializes a node of the tree of lists.
func (ls *Lists) initTree(tr *core.Tree) {
	tr.ContextMenus = nil // replaces the general ones
	tr.AddContextMenu(func(m *core.Scene, pos image.Point) {
		ls.treeContextMenu(m, tr)
	})
	// these run before those of the tree, for which they open a node
	tr.On(events.KeyChord, func(e events.Event) {
		l := listOf(tr)
		if l == nil || l.IsGroup() {
			return
		}
		switch keymap.Of(e.KeyChord()) {
		case keymap.Enter, keymap.MoveRight:
			e.SetHandled()
			ls.SetCurrent(l)
			ls.focusTable()
		}
	})
	// in the mobile layout, a list opens in its window when it is tapped,
	// also when it is selected already, as when coming back to the tree;
	// the click is on the parts of the node, which handle it
	tr.Parts.OnClick(func(e events.Event) {
		if l := listOf(tr); ls.Mobile && l != nil && !l.IsGroup() {
			ls.SetCurrent(l)
			ls.openItems()
		}
	})
	tr.Updater(func() {
		if l, ok := tr.SyncNode.(*List); ok {
			if l.IsGroup() {
				tr.SetIcon(icons.Folder)
			} else {
				tr.SetIcon(icons.List)
			}
		}
	})
}

// listOf returns the list of the given node of the tree.
func listOf(tr *core.Tree) *List {
	l, _ := tr.SyncNode.(*List)
	return l
}

func (ls *Lists) treeContextMenu(m *core.Scene, tr *core.Tree) {
	l := listOf(tr)
	if l == nil {
		return
	}
	core.NewButton(m).SetText("New list").SetIcon(icons.Add).OnClick(func(e events.Event) {
		ls.NewList(l, false)
	})
	core.NewButton(m).SetText("New group").SetIcon(icons.CreateNewFolder).OnClick(func(e events.Event) {
		ls.NewList(l, true)
	})
	core.NewButton(m).SetText("Rename").SetIcon(icons.Edit).OnClick(func(e events.Event) {
		ls.Rename(l)
	})
	if l == ls.Root {
		return
	}
	core.NewButton(m).SetText("Duplicate").SetIcon(icons.ContentCopy).OnClick(func(e events.Event) {
		tr.Duplicate()
	})
	core.NewButton(m).SetText("Delete").SetIcon(icons.Delete).OnClick(func(e events.Event) {
		ls.confirmDelete(tr, l)
	})
	core.NewSeparator(m)
	core.NewButton(m).SetText("Copy").SetIcon(icons.Copy).OnClick(func(e events.Event) {
		tr.Copy()
	})
	core.NewButton(m).SetText("Cut").SetIcon(icons.Cut).OnClick(func(e events.Event) {
		tr.Cut()
	})
	core.NewButton(m).SetText("Paste").SetIcon(icons.Paste).OnClick(func(e events.Event) {
		tr.Paste()
	})
}

// confirmDelete deletes the given list, after confirmation.
func (ls *Lists) confirmDelete(tr *core.Tree, l *List) {
	d := core.NewBody("Delete " + l.Name)
	core.NewText(d).SetType(core.TextSupporting).SetText("Delete " + l.Name + ", with everything in it?")
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).SetText("Delete").OnClick(func(e events.Event) {
			l.Delete()
			ls.tree.Resync()
			ls.treeChanged()
		})
	})
	d.RunDialog(tr)
}

// NewList prompts for a new list, or group if group is true, which goes
// at the end of the given group, or after the given list.
func (ls *Lists) NewList(at *List, group bool) {
	nl := tree.New[List]()
	name := "New list"
	if group {
		name = "New group"
		nl.Group = true
	} else {
		nl.TypeName = ls.Types[0].Name
		if cur := ls.Current; cur != nil && !cur.IsGroup() {
			nl.TypeName = cur.TypeName
		}
	}
	d := core.NewBody(name)
	core.Bind(&name, core.NewTextField(d))
	if !group {
		ch := core.NewChooser(d)
		for _, lt := range ls.Types {
			ch.Items = append(ch.Items, core.ChooserItem{Value: lt})
		}
		ch.SetCurrentValue(ls.TypeOf(nl))
		ch.OnChange(func(e events.Event) {
			nl.TypeName = ch.CurrentItem.Value.(*ListType).Name
		})
	}
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).SetText("Make").OnClick(func(e events.Event) {
			if at == nil {
				at = ls.Root
			}
			if at.IsGroup() {
				at.AddChild(nl)
			} else {
				at.Parent.AsTree().InsertChild(nl, at.IndexInParent()+1)
			}
			setUniqueName(nl, name)
			ls.tree.Resync()
			if tr := ls.tree.FindSyncNode(nl); tr != nil {
				tr.SelectEvent(events.SelectOne)
			}
			ls.SetCurrent(nl)
			ls.saveChanges()
			ls.focusTable()
		})
	})
	d.RunDialog(ls)
}

// Rename prompts for a new name for the given list.
func (ls *Lists) Rename(l *List) {
	name := l.Name
	d := core.NewBody("Rename " + l.Name)
	core.Bind(&name, core.NewTextField(d))
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).SetText("Rename").OnClick(func(e events.Event) {
			setUniqueName(l, name)
			ls.tree.Resync()
			ls.Update()
			ls.saveChanges()
		})
	})
	d.RunDialog(ls)
}

// MakeToolbar makes the toolbar of the app, which has the tools for the
// lists, the items of the current list, the list types, and the syncing,
// or in the mobile layout only those that are not for the items, which
// are in the toolbar of the window of the items (see [Lists.openItems]).
func (ls *Lists) MakeToolbar(p *tree.Plan) {
	ls.makeListTools(p)
	tree.Add(p, func(w *core.Separator) {})
	if !ls.Mobile {
		ls.makeItemTools(p) // which end with Edit type, next to List types
	}
	ls.makeTypeTools(p)
	ls.makeSyncButton(p)
}

// makeListTools adds the tools for the lists to the given plan.
func (ls *Lists) makeListTools(p *tree.Plan) {
	tree.Add(p, func(w *core.Button) {
		w.SetText("New list").SetIcon(icons.Add).
			SetTooltip("Make a new list, in the current group or after the current list")
		w.OnClick(func(e events.Event) {
			ls.NewList(ls.Current, false)
		})
	})
	tree.Add(p, func(w *core.Button) {
		w.SetText("New group").SetIcon(icons.CreateNewFolder).
			SetTooltip("Make a new group of lists, in the current group or after the current list")
		w.OnClick(func(e events.Event) {
			ls.NewList(ls.Current, true)
		})
	})
}

// makeItemTools adds the tools for the items of the current list
// to the given plan.
func (ls *Lists) makeItemTools(p *tree.Plan) {
	table := func(f func(tt *core.TreeTable, idx int)) func(e events.Event) {
		return func(e events.Event) {
			if tt := ls.table; tt != nil {
				f(tt, tt.SelectedIndex)
			}
		}
	}
	tree.Add(p, func(w *core.Button) {
		w.SetText("Add item").SetIcon(icons.Add).SetKey(keymap.InsertAfter).
			SetTooltip("Add an item after the selected one, or at the end")
		w.OnClick(table(func(tt *core.TreeTable, idx int) {
			if idx < 0 {
				tt.NewAt(-1)
			} else {
				tt.InsertAfter(idx)
			}
		}))
	})
	tree.Add(p, func(w *core.Button) {
		w.SetText("Add sub-item").SetIcon(icons.SubdirectoryArrowRight).SetKey(keymap.InsertChild).
			SetTooltip("Add an item in the selected one, such as an item in a category")
		w.OnClick(table(func(tt *core.TreeTable, idx int) {
			tt.AddChild(idx)
		}))
	})
	tree.Add(p, func(w *core.Button) {
		w.SetIcon(icons.FormatIndentIncrease).SetShortcut(key.Chord("Command+]").PlatformChord()).
			SetTooltip("Indent: move the selected item into the item before it")
		w.OnClick(table(func(tt *core.TreeTable, idx int) {
			tt.Indent(idx)
		}))
	})
	tree.Add(p, func(w *core.Button) {
		w.SetIcon(icons.FormatIndentDecrease).SetShortcut(key.Chord("Command+[").PlatformChord()).
			SetTooltip("Outdent: move the selected item out of the item it is in")
		w.OnClick(table(func(tt *core.TreeTable, idx int) {
			tt.Outdent(idx)
		}))
	})
	tree.Add(p, func(w *core.Separator) {})
	tree.Add(p, func(w *core.Button) {
		w.SetIcon(icons.Undo).SetKey(keymap.Undo).
			SetTooltip("Undo the last edit of the current list, including a change of its type")
		w.OnClick(func(e events.Event) {
			ls.Undo()
		})
	})
	tree.Add(p, func(w *core.Button) {
		w.SetIcon(icons.Redo).SetKey(keymap.Redo).
			SetTooltip("Redo the last undone edit of the current list")
		w.OnClick(func(e events.Event) {
			ls.Redo()
		})
	})
	tree.Add(p, func(w *core.Separator) {})
	tree.Add(p, func(w *core.Button) {
		w.SetText("List settings").SetIcon(icons.Settings).
			SetTooltip("Edit the settings of the current list: its type, and whether to show the values of the groups of items")
		w.OnClick(func(e events.Event) {
			ls.listSettings(w)
		})
	})
	tree.Add(p, func(w *core.Button) {
		w.SetText("Edit type").SetIcon(icons.Edit).
			SetTooltip("Edit the fields of the type of the current list")
		w.OnClick(func(e events.Event) {
			if lt := ls.TypeOf(ls.Current); lt != nil {
				ls.EditType(lt)
			}
		})
	})
}

// makeTypeTools adds the tools for the list types to the given plan.
func (ls *Lists) makeTypeTools(p *tree.Plan) {
	tree.Add(p, func(w *core.Button) {
		w.SetText("List types").SetIcon(icons.LibraryBooks).
			SetTooltip("Edit the library of list types")
		w.OnClick(func(e events.Event) {
			ls.EditTypes()
		})
	})
}

// makeSyncButton adds the button of the status of the syncing, if the
// lists sync, to the given plan.
func (ls *Lists) makeSyncButton(p *tree.Plan) {
	if syncing() {
		tree.Add(p, func(w *core.Separator) {})
		tree.Add(p, func(w *core.Button) {
			ls.syncs.button = w
			w.SetType(core.ButtonAction)
			w.Updater(func() {
				icon, status := ls.syncStatus()
				w.SetIcon(icon).SetTooltip(status)
			})
			w.OnClick(func(e events.Event) {
				ls.Sync()
			})
		})
	}
}
