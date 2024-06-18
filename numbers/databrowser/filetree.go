// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package databrowser

import (
	"log"
	"strings"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/filetree"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/tensor/table"
	"cogentcore.org/core/texteditor"
	"cogentcore.org/core/texteditor/diffbrowser"
)

// FileNode is Code version of FileNode for FileTree
type FileNode struct {
	filetree.Node
}

func (fn *FileNode) Init() {
	fn.Node.Init()
	fn.AddContextMenu(fn.ContextMenu)
}

func (fn *FileNode) OnDoubleClick(e events.Event) {
	e.SetHandled()
	br, ok := ParentBrowser(fn.This)
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
	// fmt.Println("opened:", fn.FPath)
	df := fsx.DirAndFile(string(fn.Filepath))
	switch {
	case fn.Info.Cat == fileinfo.Data:
		df := fsx.DirAndFile(string(fn.Filepath))
		tv := br.NewTabTensorTable(df)
		dt := tv.Table.Table
		err := dt.OpenCSV(fn.Filepath, table.Tab) // todo: need more flexible data handling mode
		tv.Table.Sequential()
		br.Update()
		if err != nil {
			core.ErrorSnackbar(br, err)
		}
	case fn.IsExec(): // todo: use exec?
		fn.This.(filetree.Filer).OpenFilesDefault()
	case fn.Info.Cat == fileinfo.Video: // todo: use our video viewer
		fn.This.(filetree.Filer).OpenFilesDefault()
	case fn.Info.Cat == fileinfo.Audio: // todo: use our audio viewer
		fn.This.(filetree.Filer).OpenFilesDefault()
	case fn.Info.Cat == fileinfo.Image: // todo: use our image viewer
		fn.This.(filetree.Filer).OpenFilesDefault()
	case fn.Info.Cat == fileinfo.Model: // todo: use xyz
		fn.This.(filetree.Filer).OpenFilesDefault()
	case fn.Info.Cat == fileinfo.Sheet: // todo: use our spreadsheet :)
		fn.This.(filetree.Filer).OpenFilesDefault()
	case fn.Info.Cat == fileinfo.Bin: // don't edit
		fn.This.(filetree.Filer).OpenFilesDefault()
	case fn.Info.Cat == fileinfo.Archive || fn.Info.Cat == fileinfo.Backup: // don't edit
		fn.This.(filetree.Filer).OpenFilesDefault()
	default:
		br.NewTabEditor(df, string(fn.Filepath))
	}
}

func (br *Browser) FileNodeSelected(fn *filetree.Node) {
	// todo: anything?
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
		sn.This.(*FileNode).EditFile()
	})
}

// EditFile pulls up this file in Code
func (fn *FileNode) EditFile() {
	if fn.IsDir() {
		log.Printf("FileNode Edit -- cannot view (edit) directories!\n")
		return
	}
	br, ok := ParentBrowser(fn.This)
	if ok {
		df := fsx.DirAndFile(string(fn.Filepath))
		br.NewTabEditor(df, string(fn.Filepath))
	}
}

// PlotFiles calls PlotFile on selected files
func (fn *FileNode) PlotFiles() { //types:add
	fn.SelectedFunc(func(sn *filetree.Node) {
		sn.This.(*FileNode).PlotFile()
	})
}

// PlotFile pulls up this file in Code
func (fn *FileNode) PlotFile() {
	if fn.IsDir() {
		return
	}
	br, ok := ParentBrowser(fn.This)
	if ok {
		df := fsx.DirAndFile(string(fn.Filepath))
		pl := br.NewTabPlot(df)

		dt := table.NewTable()
		err := dt.OpenCSV(fn.Filepath, table.Tab)
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
	NewDiffBrowserDirs(string(da.Filepath), string(db.Filepath))
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
	return strings.HasSuffix(fname, ".tsv") || strings.HasSuffix(fname, ".csv")
}

func (fn *FileNode) ContextMenu(m *core.Scene) {
	core.NewFuncButton(m).SetFunc(fn.EditFiles).SetText("Edit").SetIcon(icons.Edit).
		Styler(func(s *styles.Style) {
			s.SetState(!fn.HasSelection(), states.Disabled)
		})
	core.NewFuncButton(m).SetFunc(fn.PlotFiles).SetText("Plot").SetIcon(icons.Edit).
		Styler(func(s *styles.Style) {
			s.SetState(!fn.HasSelection() || fn.Info.Cat != fileinfo.Data, states.Disabled)
		})
	core.NewFuncButton(m).SetFunc(fn.DiffDirs).SetText("Diff Dirs").SetIcon(icons.Edit).
		Styler(func(s *styles.Style) {
			s.SetState(!fn.HasSelection() || !fn.IsDir(), states.Disabled)
		})
}
