// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"image"
	"os"
	"path/filepath"
	"reflect"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/iox/jsonx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/types"
)

// List is a list of items, or a group of lists, in the tree of lists.
type List struct {
	tree.NodeBase

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

	// tree is the tree of lists.
	tree *core.Tree

	// table is the tree table of the items of the current list.
	table *core.TreeTable
}

const (
	listsFile = "lists.json"
	typesFile = "types.json"
)

// listData is the saved form of a list or group, with the lists in it.
type listData struct {
	Name         string
	Type         string             `json:",omitempty"`
	ColumnWidths map[string]float32 `json:",omitempty"`
	Items        []*Item            `json:",omitempty"`
	Lists        []*listData        `json:",omitempty"`
}

// Open opens the lists and list types in the given directory, starting
// with the standard list types and an empty list for the files that do
// not exist yet.
func (ls *Lists) Open(dir string) error {
	ls.Dir = dir
	ls.Types = nil
	tf := filepath.Join(dir, typesFile)
	if err := jsonx.Open(&ls.Types, tf); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if len(ls.Types) == 0 {
		ls.Types = standardTypes()
	}
	var root listData
	lf := filepath.Join(dir, listsFile)
	if err := jsonx.Open(&root, lf); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		root = listData{Name: "Lists", Lists: []*listData{{Name: "To do", Type: "To do"}}}
	}
	ls.Root = newList(&root)
	ls.Root.Group = true
	ls.Current = nil
	ls.keepUsedTypes()
	ls.Update()
	return nil
}

// newList returns a new list made from the given saved list.
func newList(ld *listData) *List {
	l := tree.New[List]()
	l.SetName(ld.Name)
	l.Group = ld.Type == ""
	l.TypeName = ld.Type
	l.ColumnWidths = ld.ColumnWidths
	l.Items = ld.Items
	for _, c := range ld.Lists {
		l.AddChild(newList(c))
	}
	return l
}

// listData returns the saved form of the given list.
func (ls *List) listData() *listData {
	ld := &listData{Name: ls.Name, Type: ls.TypeName, ColumnWidths: ls.ColumnWidths, Items: ls.Items}
	for _, c := range ls.Children {
		ld.Lists = append(ld.Lists, c.(*List).listData())
	}
	return ld
}

// Save saves the lists and the list types in [Lists.Dir].
func (ls *Lists) Save() error {
	if ls.Dir == "" || ls.Root == nil {
		return nil
	}
	if err := os.MkdirAll(ls.Dir, 0o755); err != nil {
		return err
	}
	if err := jsonx.SaveIndent(ls.Types, filepath.Join(ls.Dir, typesFile)); err != nil {
		return err
	}
	return jsonx.SaveIndent(ls.Root.listData(), filepath.Join(ls.Dir, listsFile))
}

// saveChanges saves after a change, reporting an error.
func (ls *Lists) saveChanges() {
	if err := ls.Save(); err != nil {
		core.ErrorSnackbar(ls, err, "Error saving lists")
	}
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
	l.values = lt.newItems(l.Items)
	return l.values.Interface()
}

// itemsChanged saves the items of the current list after they change.
func (ls *Lists) itemsChanged() {
	if l := ls.Current; l != nil && l.values.IsValid() {
		l.Items = ls.TypeOf(l).savedItems(l.values)
	}
	ls.saveChanges()
}

// SetCurrent sets the current list, whose items are shown.
func (ls *Lists) SetCurrent(l *List) {
	if l == ls.Current {
		return
	}
	ls.Current = l
	ls.Update()
}

// SetType sets the type of the current list, keeping the values
// of the fields with the same names.
func (ls *Lists) SetType(lt *ListType) {
	l := ls.Current
	if l == nil || l.IsGroup() || lt == nil || lt.Name == l.TypeName {
		return
	}
	l.TypeName = lt.Name
	l.values = reflect.Value{}
	ls.saveChanges()
	ls.Update()
}

func (ls *Lists) Init() {
	ls.Frame.Init()
	ls.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
	})
	tree.AddChild(ls, func(w *core.Splits) {
		w.SetSplits(0.2, 0.8)
		tree.AddChild(w, func(w *core.Frame) {
			w.Styler(func(s *styles.Style) {
				s.Direction = styles.Column
				s.Overflow.Set(styles.OverflowAuto)
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
		})
		tree.AddChild(w, func(w *core.Frame) {
			w.Styler(func(s *styles.Style) {
				s.Direction = styles.Column
				s.Grow.Set(1, 1)
			})
			w.Maker(ls.makeCurrent)
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
	tree.Add(p, func(w *core.Frame) {
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
		tree.AddChild(w, func(w *core.Chooser) {
			w.SetTooltip("The type of the list, which has the fields of its items")
			w.OnChange(func(e events.Event) {
				ls.SetType(w.CurrentItem.Value.(*ListType))
			})
			w.Updater(func() {
				items := make([]core.ChooserItem, len(ls.Types))
				for i, lt := range ls.Types {
					items[i] = core.ChooserItem{Value: lt}
				}
				w.SetItems(items...)
				w.SetCurrentValue(ls.TypeOf(ls.Current))
			})
		})
	})
	tree.Add(p, func(w *core.TreeTable) {
		ls.table = w
		w.SetDragMoves(true) // reordering the items, unlike the lists
		w.OnChange(func(e events.Event) {
			ls.Current.ColumnWidths = w.ColumnWidths
			ls.itemsChanged()
		})
		w.Updater(func() {
			w.SetItems(ls.values(ls.Current))
			w.ColumnWidths = ls.Current.ColumnWidths
		})
	})
}

// treeChanged updates after the tree of lists changes.
func (ls *Lists) treeChanged() {
	if l := ls.Current; l != nil && l != ls.Root && l.Parent == nil { // deleted
		ls.Current = nil
		ls.Update()
	}
	if ls.newLists() {
		ls.tree.Resync()
	}
	ls.saveChanges()
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
			l.SetName("New list")
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
			nl.SetName(name)
			if at == nil {
				at = ls.Root
			}
			if at.IsGroup() {
				at.AddChild(nl)
			} else {
				at.Parent.AsTree().InsertChild(nl, at.IndexInParent()+1)
			}
			ls.tree.Resync()
			if tr := ls.tree.FindSyncNode(nl); tr != nil {
				tr.SelectEvent(events.SelectOne)
			}
			ls.SetCurrent(nl)
			ls.saveChanges()
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
			l.SetName(name)
			ls.tree.Resync()
			ls.Update()
			ls.saveChanges()
		})
	})
	d.RunDialog(ls)
}

// MakeToolbar makes the toolbar of the app.
func (ls *Lists) MakeToolbar(p *tree.Plan) {
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
	tree.Add(p, func(w *core.Separator) {})
	table := func(f func(tt *core.TreeTable, idx int)) func(e events.Event) {
		return func(e events.Event) {
			if tt := ls.table; tt != nil {
				f(tt, tt.SelectedIndex)
			}
		}
	}
	tree.Add(p, func(w *core.Button) {
		w.SetText("Add item").SetIcon(icons.Add).
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
		w.SetText("Add sub-item").SetIcon(icons.SubdirectoryArrowRight).
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
		w.SetText("Edit type").SetIcon(icons.Edit).
			SetTooltip("Edit the fields of the type of the current list")
		w.OnClick(func(e events.Event) {
			if lt := ls.TypeOf(ls.Current); lt != nil {
				ls.EditType(lt)
			}
		})
	})
	tree.Add(p, func(w *core.Button) {
		w.SetText("List types").SetIcon(icons.LibraryBooks).
			SetTooltip("Edit the library of list types")
		w.OnClick(func(e events.Event) {
			ls.EditTypes()
		})
	})
}
