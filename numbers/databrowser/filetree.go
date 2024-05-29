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
	case strings.HasSuffix(df, ".tsv"):
		f := strings.TrimSuffix(df, ".tsv")
		tv := br.NewTabTable(f)
		dt := tv.Table.Table
		err := dt.OpenCSV(fn.FPath, table.Tab)
		tv.Table.Sequential()
		br.Update()
		if err != nil {
			fmt.Println(err.Error())
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

func (fn *FileNode) ContextMenu(m *core.Scene) {
	views.NewFuncButton(m, fn.EditFiles).SetText("Edit").SetIcon(icons.Edit).
		Style(func(s *styles.Style) {
			s.SetState(!fn.HasSelection(), states.Disabled)
		})
}

/////////////////////////////////////////////////////////////////////
//   OpenNodes

// OpenNodes is a list of file nodes that have been opened for editing -- it
// is maintained in recency order -- most recent on top -- call Add every time
// a node is opened / visited for editing
type OpenNodes []*filetree.Node

// Add adds given node to list of open nodes -- if already on the list it is
// moved to the top -- returns true if actually added.
// Connects to fn.TextBuf signal and auto-closes when buffer closes.
func (on *OpenNodes) Add(fn *filetree.Node) bool {
	added := on.AddImpl(fn)
	if !added {
		return added
	}
	if fn.Buffer != nil {
		// fn.Buf.TextBufSig.Connect(fn.This(), func(recv, send tree.Node, sig int64, data any) {
		// 	if sig == int64(texteditor.BufClosed) {
		// 		fno, _ := recv.Embed(views.KiT_FileNode).(*filetree.Node)
		// 		on.Delete(fno)
		// 	}
		// })
	}
	return added
}

// AddImpl adds given node to list of open nodes -- if already on the list it is
// moved to the top -- returns true if actually added.
func (on *OpenNodes) AddImpl(fn *filetree.Node) bool {
	sz := len(*on)

	for i, f := range *on {
		if f == fn {
			if i == 0 {
				return false
			}
			copy((*on)[1:i+1], (*on)[0:i])
			(*on)[0] = fn
			return false
		}
	}

	*on = append(*on, nil)
	if sz > 0 {
		copy((*on)[1:], (*on)[0:sz])
	}
	(*on)[0] = fn
	return true
}

// Delete deletes given node in list of open nodes, returning true if found and deleted
func (on *OpenNodes) Delete(fn *filetree.Node) bool {
	for i, f := range *on {
		if f == fn {
			on.DeleteIndex(i)
			return true
		}
	}
	return false
}

// DeleteIndex deletes at given index
func (on *OpenNodes) DeleteIndex(idx int) {
	*on = append((*on)[:idx], (*on)[idx+1:]...)
}

// DeleteDeleted deletes deleted nodes on list
func (on *OpenNodes) DeleteDeleted() {
	sz := len(*on)
	for i := sz - 1; i >= 0; i-- {
		fn := (*on)[i]
		if fn.This() == nil || fn.FRoot == nil {
			on.DeleteIndex(i)
		}
	}
}

// Strings returns a string list of nodes, with paths relative to proj root
func (on *OpenNodes) Strings() []string {
	on.DeleteDeleted()
	sl := make([]string, len(*on))
	for i, fn := range *on {
		rp := fn.FRoot.RelPath(fn.FPath)
		rp = strings.TrimSuffix(rp, fn.Nm)
		if rp != "" {
			sl[i] = fn.Nm + " - " + rp
		} else {
			sl[i] = fn.Nm
		}
		if fn.IsNotSaved() {
			sl[i] += " *"
		}
	}
	return sl
}

// ByStringName returns the open node with given strings name
func (on *OpenNodes) ByStringName(name string) *filetree.Node {
	sl := on.Strings()
	for i, ns := range sl {
		if ns == name {
			return (*on)[i]
		}
	}
	return nil
}

// NChanged returns number of changed open files
func (on *OpenNodes) NChanged() int {
	cnt := 0
	for _, fn := range *on {
		if fn.IsNotSaved() {
			cnt++
		}
	}
	return cnt
}

// FindPath finds node for given path, nil if not found
func (on *OpenNodes) FindPath(path string) *filetree.Node {
	for _, f := range *on {
		if f.FPath == core.Filename(path) {
			return f
		}
	}
	return nil
}

// EditFiles calls EditFile on selected files
func (fn *FileNode) EditFiles() { //types:add
	sels := fn.SelectedViews()
	for i := len(sels) - 1; i >= 0; i-- {
		sn := sels[i].This().(*FileNode)
		sn.EditFile()
	}
}
