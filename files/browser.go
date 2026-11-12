// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"image"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
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

	// dragWindow is whether the Files is in a new window that a tab is
	// dragged into, which does not open a default terminal.
	dragWindow bool

	// splitsChooser is the chooser of the layout of the panels in the toolbar.
	splitsChooser *core.Chooser

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
		if !fl.dragWindow {
			fl.recycleTerminal(TerminalTabName, fl.Root)
		}
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
					w.AppsAsDirs = Settings.ShowHidden // (see opensAsApp)
				})
			})
			tree.AddChildAt(w, "favorites", fl.initFavorites)
		})
		tree.AddChildAt(w, "dirtabs", func(w *core.Tabs) {
			w.SetType(core.FunctionalTabs)
			fl.dragTabs(w, (*Files).DirTabs)
			w.Styler(func(s *styles.Style) {
				s.Overflow.Set(styles.OverflowHidden)
				s.Grow.Set(1, 1)
			})
		})
		tree.AddChildAt(w, "tabs", func(w *core.Tabs) {
			w.SetType(core.FunctionalTabs)
			fl.dragTabs(w, (*Files).Tabs)
			// a new tab is a new terminal
			w.NewTabButton = true
			w.NewTabFunc = fl.newTerminalTab
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
	fr, _ := ts.NewTab(dir)
	fr.Styler(func(s *styles.Style) {
		s.Overflow.Set(styles.OverflowHidden) // the table scrolls itself
	})
	dt := NewDirTable(fr)
	label := func() {
		// (the tab of its frame, which it may have been dragged into)
		if tab := tabOfFrame(fr); tab != nil {
			tab.SetText(dirLabel(dt.Dir)).SetTooltip(dt.Dir)
			tab.Update()
		}
	}
	dt.OnChange(func(e events.Event) { label() })
	dt.SetDir(dir)
	label()
	fl.dirTabTerminal(fr)
	ts.SelectTabIndex(ts.NumTabs() - 1)
	return dt
}

// dirTabTerminal makes the tab button of the given frame of a directory
// tab open a new terminal in its directory when it is double-clicked, or
// from its context menu.
func (fl *Files) dirTabTerminal(fr *core.Frame) {
	tab, dt := tabOfFrame(fr), dirTableIn(fr)
	if tab == nil || dt == nil {
		return
	}
	tab.Styler(func(s *styles.Style) {
		// (a tab does not get double clicks otherwise)
		s.SetAbilities(true, abilities.DoubleClickable)
	})
	tab.OnDoubleClick(func(e events.Event) {
		fl.NewTerminal(dt.Dir)
	})
	tab.AddContextMenu(func(m *core.Scene, _ image.Point) {
		core.NewButton(m).SetText("Terminal here").SetIcon(icons.Terminal).
			SetTooltip("Open a new terminal in " + dt.Dir).
			OnClick(func(e events.Event) {
				fl.NewTerminal(dt.Dir)
			})
	})
}

// tabOfFrame returns the tab button of the given frame of a tab, in the
// tabs that it is in, or nil.
func tabOfFrame(fr *core.Frame) *core.Tab {
	ts := tree.ParentByType[*core.Tabs](fr)
	if ts == nil {
		return nil
	}
	idx := fr.IndexInParent()
	if idx < 0 || idx >= ts.NumTabs() {
		return nil
	}
	_, tab := ts.TabAtIndex(idx)
	return tab
}

// dragTabs makes the given tabs of the Files, which the given function
// returns of a Files, draggable out of the window into a new Files window
// with the same root (see [core.Tabs.DragToWindow]), and into those of
// other Files windows, with what goes with them (as the working
// directories of terminals).
func (fl *Files) dragTabs(ts *core.Tabs, tabsOf func(f *Files) *core.Tabs) {
	ts.DragToWindow = func(b *core.Body) *core.Tabs {
		nf := InitBody(b, fl.Root)
		nf.dragWindow = true
		nf.Update() // (for its tabs, which the tab is moved into)
		return tabsOf(nf)
	}
	ts.MoveTabFunc = func(from int, to *core.Tabs, at int) {
		if to == ts {
			return
		}
		// moved out, to the tabs of another Files
		if fl.lastTerminal != nil && !fl.hasTerminal(fl.lastTerminal) {
			fl.lastTerminal = nil
		}
		fl.updateTerminalLabels()
	}
	ts.MoveInFunc = func(index int) {
		fr, _ := ts.TabAtIndex(index)
		if fr == nil || !fr.HasChildren() {
			return
		}
		switch w := fr.Child(0).(type) {
		case *terminal.Terminal:
			fl.focusOnTab(fr)
			fl.lastTerminal = w
			fl.updateTerminalLabels()
		case *DirTable:
			if tab := tabOfFrame(fr); tab != nil {
				tab.SetText(dirLabel(w.Dir)).SetTooltip(w.Dir)
			}
			fl.dirTabTerminal(fr) // (its tab button is new)
		}
	}
}

// hasTerminal returns whether the given terminal is in a tab of the
// command tabs of the Files.
func (fl *Files) hasTerminal(tm *terminal.Terminal) bool {
	ts := fl.Tabs()
	if ts == nil {
		return false
	}
	for i := range ts.NumTabs() {
		if fr, _ := ts.TabAtIndex(i); fr != nil && fr.HasChildren() && fr.Child(0) == tm.This {
			return true
		}
	}
	return false
}

// newTerminalTab makes a new terminal in the new tab of the command tabs at
// the given index (see [core.Tabs.NewTabFunc]), in the current directory,
// and moves the keyboard focus to it.
func (fl *Files) newTerminalTab(index int) {
	ts := fl.Tabs()
	fr, tab := ts.TabAtIndex(index)
	if fr == nil {
		return
	}
	n := 2
	for ts.TabByName(terminalTabName(n)) != nil {
		n++
	}
	name := terminalTabName(n)
	fr.SetName(name)
	tab.SetName(name)
	tab.SetText(name)
	tm := fl.initTerminal(fr, name, fl.CurrentDir())
	tm.Defer(func() { tm.SetFocus() }) // (once it is shown)
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
		fl.splitsChooser = w
		w.SetStrings(splitNames()...).SetCurrentValue(string(splitByName(Settings.Splits).Name))
		chord, _, _ := strings.Cut(string(KeySetSplit.Chord()), "\n")
		w.SetTooltip("The layout of the panels (" + chord + ")")
		w.OnChange(func(e events.Event) {
			fl.setSplits(SplitName(w.CurrentItem.Value.(string)))
		})
	})
}
