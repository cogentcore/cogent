// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"io/fs"
	"path/filepath"
	"slices"
	"strings"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/text/terminal"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/types"
)

// Files is the file explorer, laid out as in Cogent Code: a tree of the
// files on the left, the directory tabs with the files of directories,
// and the command tabs, such as terminals and the console, to the right
// of them, or under them in the compact layout (see [SettingsData.Splits]).
type Files struct {
	core.Frame

	// Root is the root directory of the file tree (see [Files.SetRoot]).
	Root string `set:"-"`

	// Tree is the tree of the files in the Root directory.
	Tree *FileTree `set:"-" json:"-"`

	// lastTerminal is the last terminal used.
	lastTerminal *terminal.Terminal

	// terminalDirs are the working directories of the terminals,
	// for the labels of their tabs.
	terminalDirs map[*terminal.Terminal]*terminalDir

	// keySeq1 is the first key of a key sequence being entered,
	// such as Control+X (see [Files.filesKeys]).
	keySeq1 key.Chord
}

// openFiles are the open [Files], to which the settings are applied.
var openFiles []*Files

// These are the indexes of the panels in the [Files.Splits]: the first
// is the [Files.LeftSplits] of the file tree and the favorites.
const (
	FileTreeIndex = iota
	DirTabsIndex
	TabsIndex
)

func (fl *Files) Init() {
	fl.Frame.Init()
	openFiles = append(openFiles, fl)
	fl.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
	})
	fl.OnFirst(events.KeyChord, fl.filesKeys)
	fl.OnShow(func(e events.Event) {
		fl.applySplits()
		fl.Tree.OpenPath(fl.Root)
		fl.OpenConsoleTab()
		fl.recycleTerminal(TerminalTabName, fl.Root)
		if fl.DirTabs().NumTabs() == 0 {
			fl.OpenDir(fl.Root)
		}
	})

	tree.AddChildAt(fl, "splits", func(w *core.Splits) {
		w.OnChange(func(e events.Event) {
			fl.saveSplits()
		})
		tree.AddChildAt(w, "left", func(w *core.Splits) {
			w.Styler(func(s *styles.Style) {
				s.Direction = styles.Column
			})
			w.OnChange(func(e events.Event) {
				fl.saveSplits()
			})
			tree.AddChildAt(w, "filetree", func(w *core.Frame) {
				w.Styler(func(s *styles.Style) {
					s.Direction = styles.Column
					s.Overflow.Set(styles.OverflowAuto)
				})
				tree.AddChildAt(w, "filetree", func(w *FileTree) {
					fl.Tree = w
					w.OpenDepth = 1
					w.FileNodeType = types.For[FileNode]()
					w.FilterFunc = func(path string, info fs.FileInfo) bool {
						return Settings.ShowHidden || !strings.HasPrefix(info.Name(), ".")
					}
				})
			})
			tree.AddChildAt(w, "favorites", fl.initFavorites)
		})
		tree.AddChildAt(w, "dirtabs", func(w *core.Tabs) {
			w.SetType(core.FunctionalTabs)
			w.Styler(func(s *styles.Style) {
				s.Overflow.Set(styles.OverflowHidden)
				s.Grow.Set(1, 1)
			})
		})
		tree.AddChildAt(w, "tabs", func(w *core.Tabs) {
			w.SetType(core.FunctionalTabs)
			w.Styler(func(s *styles.Style) {
				s.Overflow.Set(styles.OverflowHidden)
				s.Grow.Set(1, 1)
			})
		})
	})
}

func (fl *Files) Destroy() {
	openFiles = slices.DeleteFunc(openFiles, func(f *Files) bool { return f == fl })
	fl.Frame.Destroy()
}

// SetRoot sets the root directory of the file tree.
func (fl *Files) SetRoot(root string) *Files {
	root = cleanDir(root)
	fl.Root = root
	if fl.Tree != nil && fl.IsVisible() {
		fl.Tree.OpenPath(root)
	}
	return fl
}

// ParentFiles returns the [Files] of the given node.
func ParentFiles(n tree.Node) (*Files, bool) {
	fl := tree.ParentByType[*Files](n)
	return fl, fl != nil
}

// Splits returns the splits of the panels, nil if the tree has not been built.
func (fl *Files) Splits() *core.Splits {
	sv, _ := fl.ChildByName("splits", 0).(*core.Splits)
	return sv
}

// LeftSplits returns the splits of the file tree and the favorites,
// nil if the tree has not been built.
func (fl *Files) LeftSplits() *core.Splits {
	sv := fl.Splits()
	if sv == nil {
		return nil
	}
	lt, _ := sv.Child(FileTreeIndex).(*core.Splits)
	return lt
}

// Favorites returns the table of the favorite paths, nil if the tree has not been built.
func (fl *Files) Favorites() *core.Table {
	lt := fl.LeftSplits()
	if lt == nil {
		return nil
	}
	tb, _ := lt.Child(1).(*core.Table)
	return tb
}

// DirTabs returns the directory tabs, nil if the tree has not been built.
func (fl *Files) DirTabs() *core.Tabs {
	sv := fl.Splits()
	if sv == nil {
		return nil
	}
	ts, _ := sv.Child(DirTabsIndex).(*core.Tabs)
	return ts
}

// Tabs returns the command tabs, nil if the tree has not been built.
func (fl *Files) Tabs() *core.Tabs {
	sv := fl.Splits()
	if sv == nil {
		return nil
	}
	ts, _ := sv.Child(TabsIndex).(*core.Tabs)
	return ts
}

// dirTableIn returns the [DirTable] in the given tab frame, or nil.
func dirTableIn(fr core.Widget) *DirTable {
	if fr == nil || !fr.AsTree().HasChildren() {
		return nil
	}
	dt, _ := fr.AsTree().Child(0).(*DirTable)
	return dt
}

// DirTables returns the directory tables in the directory tabs.
func (fl *Files) DirTables() []*DirTable {
	ts := fl.DirTabs()
	if ts == nil {
		return nil
	}
	var dts []*DirTable
	for i := range ts.NumTabs() {
		fr, _ := ts.TabAtIndex(i)
		if dt := dirTableIn(fr); dt != nil {
			dts = append(dts, dt)
		}
	}
	return dts
}

// CurrentDirTable returns the directory table in the current directory tab, or nil.
func (fl *Files) CurrentDirTable() *DirTable {
	ts := fl.DirTabs()
	if ts == nil {
		return nil
	}
	fr, _ := ts.CurrentTab()
	return dirTableIn(fr)
}

// CurrentDir returns the directory of the current directory tab,
// or the Root directory if there is none.
func (fl *Files) CurrentDir() string {
	if dt := fl.CurrentDirTable(); dt != nil {
		return dt.Dir
	}
	return fl.Root
}

// OpenDir shows the files of the given directory in a directory tab,
// selecting its tab if it is shown in one already.
func (fl *Files) OpenDir(dir string) *DirTable { //types:add
	ts := fl.DirTabs()
	if ts == nil {
		return nil
	}
	dir = cleanDir(dir)
	for i := range ts.NumTabs() {
		fr, _ := ts.TabAtIndex(i)
		if dt := dirTableIn(fr); dt != nil && dt.Dir == dir {
			ts.SelectTabIndex(i)
			return dt
		}
	}
	fr, tab := ts.NewTab(dir)
	fr.Styler(func(s *styles.Style) {
		s.Overflow.Set(styles.OverflowHidden) // the table scrolls itself
	})
	dt := NewDirTable(fr)
	label := func() {
		tab.SetText(dirLabel(dt.Dir)).SetTooltip(dt.Dir)
		tab.Update()
	}
	dt.OnChange(func(e events.Event) { label() })
	dt.SetDir(dir)
	label()
	ts.SelectTabIndex(ts.NumTabs() - 1)
	return dt
}

// cleanDir returns the absolute path of the given directory, with its
// symbolic links evaluated, as in the file tree, so that a directory has
// one path, and one tab.
func cleanDir(dir string) string {
	if abs, err := filepath.Abs(dir); err == nil {
		dir = abs
	}
	if eval, err := filepath.EvalSymlinks(dir); err == nil {
		dir = eval
	}
	return dir
}

// dirLabel returns the label of the tab of the given directory.
func dirLabel(dir string) string {
	if base := filepath.Base(dir); base != string(filepath.Separator) && base != "." {
		return base
	}
	return dir
}

// DirUp shows the parent directory in the current directory tab.
func (fl *Files) DirUp() { //types:add
	if dt := fl.CurrentDirTable(); dt != nil {
		dt.DirUp()
	}
}

// TerminalHere opens a new terminal in the directory of the current directory tab.
func (fl *Files) TerminalHere() { //types:add
	fl.NewTerminal(fl.CurrentDir())
}

// dirMenuItems adds the context menu items for the given directory,
// in the directory tables and the file tree: Terminal here and Add to favorites.
func (fl *Files) dirMenuItems(m *core.Scene, dir string) {
	core.NewButton(m).SetText("Terminal here").SetIcon(icons.Terminal).
		SetTooltip("Open a new terminal in " + dir).
		OnClick(func(e events.Event) {
			fl.NewTerminal(dir)
		})
	core.NewButton(m).SetText("Add to favorites").SetIcon(icons.Favorite).
		SetTooltip("Add " + dir + " to the favorite paths").
		OnClick(func(e events.Event) {
			fl.AddFavorite(dir)
		})
}

func (fl *Files) MakeToolbar(p *tree.Plan) {
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(fl.DirUp).SetText("Up").SetIcon(icons.ArrowUpward)
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(fl.TerminalHere).SetText("Terminal").SetIcon(icons.Terminal)
	})
	tree.Add(p, func(w *core.Chooser) {
		w.SetStrings(splitNames()...).SetCurrentValue(string(splitByName(Settings.Splits).Name))
		w.SetTooltip("The layout of the panels")
		w.OnChange(func(e events.Event) {
			fl.setSplits(SplitName(w.CurrentItem.Value.(string)))
		})
	})
}
