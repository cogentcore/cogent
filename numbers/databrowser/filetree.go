// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package databrowser

import (
	"fmt"
	"log"
	"strings"

	"cogentcore.org/core/base/dirs"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/filetree"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/tensor/table"
	"cogentcore.org/core/texteditor"
	"cogentcore.org/core/texteditor/diffbrowser"
	"cogentcore.org/core/views"
)

// FileNode is Code version of FileNode for FileTree view
type FileNode struct {
	filetree.Node
}

func (fn *FileNode) OnInit() {
	fn.Node.OnInit()
	fn.AddContextMenu(fn.ContextMenu)
}

func (fn *FileNode) OnDoubleClick(e events.Event) {
	e.SetHandled()
	br, ok := ParentBrowser(fn.This())
	if !ok {
		return
	}
	sels := fn.SelectedViews()
	if len(sels) > 0 {
		sn := filetree.AsNode(sels[len(sels)-1])
		if sn != nil {
			if sn.IsDir() {
				if !sn.HasChildren() {
					sn.OpenEmptyDir()
				} else {
					sn.ToggleClose()
				}
			} else {
				br.FileNodeOpened(sn)
			}
		}
	}
}

func (br *Browser) FileNodeOpened(fn *filetree.Node) {
	fmt.Println("opened:", fn.FPath)
	df := dirs.DirAndFile(string(fn.FPath))
	switch {
	case IsTableFile(string(fn.Name())):
		df := dirs.DirAndFile(string(fn.FPath))
		tv := br.NewTabTable(df)
		dt := tv.Table.Table
		err := dt.OpenCSV(fn.FPath, table.Tab)
		tv.Table.Sequential()
		br.Update()
		if err != nil {
			core.ErrorSnackbar(br, err)
		}
	default:
		br.NewTabEditor(df, string(fn.FPath))
	}
}

func (br *Browser) FileNodeSelected(fn *filetree.Node) {

}

// NewTabEditor opens an editor tab for given file
func (br *Browser) NewTabEditor(label, filename string) *texteditor.Editor {
	tabs := br.Tabs()
	tab := tabs.RecycleTab(label, true)
	if tab.HasChildren() {
		ed := tab.Child(0).(*texteditor.Editor)
		ed.Buffer.Open(core.Filename(filename))
		return ed
	}
	ed := texteditor.NewSoloEditor(tab)
	ed.Buffer.Open(core.Filename(filename))
	br.Update()
	return ed
}

// EditFiles calls EditFile on selected files
func (fn *FileNode) EditFiles() { //types:add
	fn.SelectedFunc(func(sn *filetree.Node) {
		sn.This().(*FileNode).EditFile()
	})
}

// EditFile pulls up this file in Code
func (fn *FileNode) EditFile() {
	if fn.IsDir() {
		log.Printf("FileNode Edit -- cannot view (edit) directories!\n")
		return
	}
	br, ok := ParentBrowser(fn.This())
	if ok {
		df := dirs.DirAndFile(string(fn.FPath))
		br.NewTabEditor(df, string(fn.FPath))
	}
}

// PlotFiles calls PlotFile on selected files
func (fn *FileNode) PlotFiles() { //types:add
	fn.SelectedFunc(func(sn *filetree.Node) {
		sn.This().(*FileNode).PlotFile()
	})
}

// PlotFile pulls up this file in Code
func (fn *FileNode) PlotFile() {
	if fn.IsDir() {
		return
	}
	br, ok := ParentBrowser(fn.This())
	if ok {
		df := dirs.DirAndFile(string(fn.FPath))
		pl := br.NewTabPlot(df)

		dt := table.NewTable()
		err := dt.OpenCSV(fn.FPath, table.Tab)
		if err != nil {
			core.ErrorSnackbar(br, err)
		}
		pl.SetTable(dt)
		pl.Params.Title = df
		br.Update()
	}
}

// DiffDirs displays a browser with differences between two selected directories
func (fn *FileNode) DiffDirs() { //types:add
	var da, db *filetree.Node
	fn.SelectedFunc(func(sn *filetree.Node) {
		if sn.IsDir() {
			if da == nil {
				da = sn
			} else if db == nil {
				db = sn
			}
		}
	})
	if da == nil || db == nil {
		core.MessageSnackbar(fn, "DiffDirs requires two selected directories")
		return
	}
	NewDiffBrowserDirs(string(da.FPath), string(db.FPath))
}

// NewDiffBrowserDirs returns a new diff browser for files that differ
// within the two given directories.  Excludes Job and .tsv data files.
func NewDiffBrowserDirs(pathA, pathB string) {
	brow, b := diffbrowser.NewBrowserWindow()
	brow.DiffDirs(pathA, pathB, func(fname string) bool {
		if IsTableFile(fname) {
			return true
		}
		if strings.HasPrefix(fname, "job.") || fname == "dbmeta.toml" {
			return true
		}
		return false
	})
	b.RunWindow()
}

func IsTableFile(fname string) bool {
	return strings.HasSuffix(fname, ".tsv")
}

func (fn *FileNode) ContextMenu(m *core.Scene) {
	views.NewFuncButton(m, fn.EditFiles).SetText("Edit").SetIcon(icons.Edit).
		Style(func(s *styles.Style) {
			s.SetState(!fn.HasSelection(), states.Disabled)
		})
	views.NewFuncButton(m, fn.PlotFiles).SetText("Plot").SetIcon(icons.Edit).
		Style(func(s *styles.Style) {
			s.SetState(!fn.HasSelection() || !IsTableFile(fn.Name()), states.Disabled)
		})
	views.NewFuncButton(m, fn.DiffDirs).SetText("Diff Dirs").SetIcon(icons.Edit).
		Style(func(s *styles.Style) {
			s.SetState(!fn.HasSelection() || !fn.IsDir(), states.Disabled)
		})
}
