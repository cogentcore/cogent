// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/filetree"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
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
	ge, ok := ParentCode(fn.This)
	if !ok {
		return
	}
	done := false
	fn.SelectedFunc(func(sn *filetree.Node) {
		if !done {
			if sn.IsDir() {
				if !sn.HasChildren() {
					sn.OpenEmptyDir()
				} else {
					sn.ToggleClose()
				}
			} else {
				ge.FileNodeOpened(sn)
			}
			done = true
		}
	})
}

// EditFile pulls up this file in Code
func (fn *FileNode) EditFile() {
	if fn.IsDir() {
		log.Printf("FileNode Edit -- cannot view (edit) directories!\n")
		return
	}
	ge, ok := ParentCode(fn.This)
	if ok {
		ge.NextViewFileNode(&fn.Node)
	}
}

// SetRunExec sets executable as the RunExec executable that will be run with Run / Debug buttons
func (fn *FileNode) SetRunExec() {
	if !fn.IsExec() {
		log.Printf("FileNode SetRunExec -- only works for executable files!\n")
		return
	}
	ge, ok := ParentCode(fn.This)
	if ok {
		ge.Settings.RunExec = fn.FPath
		ge.Settings.BuildDir = core.Filename(filepath.Dir(string(fn.FPath)))
	}
}

// ExecCmdFile pops up a menu to select a command appropriate for the given node,
// and shows output in MainTab with name of command
func (fn *FileNode) ExecCmdFile() { //types:add
	ge, ok := ParentCode(fn.This)
	if ok {
		ge.ExecCmdFileNode(&fn.Node)
	} else {
		fmt.Println("no code!")
	}

}

// ExecCmdNameFile executes given command name on node
func (fn *FileNode) ExecCmdNameFile(cmdNm string) {
	ge, ok := ParentCode(fn.This)
	if ok {
		ge.ExecCmdNameFileNode(&fn.Node, CmdName(cmdNm), true, true)
	}
}

func (fn *FileNode) ContextMenu(m *core.Scene) {
	core.NewButton(m).SetText("Exec Cmd").SetIcon(icons.Terminal).
		SetMenu(CommandMenu(&fn.Node)).Styler(func(s *styles.Style) {
		s.SetState(!fn.HasSelection(), states.Disabled)
	})
	core.NewFuncButton(m).SetFunc(fn.EditFiles).SetText("Edit").SetIcon(icons.Edit).
		Styler(func(s *styles.Style) {
			s.SetState(!fn.HasSelection(), states.Disabled)
		})
	core.NewFuncButton(m).SetFunc(fn.SetRunExecs).SetText("Set Run Exec").SetIcon(icons.PlayArrow).
		Styler(func(s *styles.Style) {
			s.SetState(!fn.HasSelection() || !fn.IsExec(), states.Disabled)
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
		// fn.Buf.TextBufSig.Connect(fn.This, func(recv, send tree.Node, sig int64, data any) {
		// 	if sig == int64(texteditor.BufClosed) {
		// 		fno, _ := recv.Embed(core.KiT_FileNode).(*filetree.Node)
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
		if fn.This == nil || fn.FRoot == nil {
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
		rp = strings.TrimSuffix(rp, fn.Name)
		if rp != "" {
			sl[i] = fn.Name + " - " + rp
		} else {
			sl[i] = fn.Name
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
	fn.SelectedFunc(func(sn *filetree.Node) {
		sn.This.(*FileNode).EditFile()
	})
}

// SetRunExecs sets executable as the RunExec executable that will be run with Run / Debug buttons
func (fn *FileNode) SetRunExecs() { //types:add
	fn.SelectedFunc(func(sn *filetree.Node) {
		sn.This.(*FileNode).SetRunExec()
	})
}

// RenameFiles calls RenameFile on any selected nodes
func (fn *FileNode) RenameFiles() {
	ge, ok := ParentCode(fn.This)
	if !ok {
		return
	}
	ge.SaveAllCheck(true, func() {
		var nodes []*FileNode
		sels := fn.SelectedViews()
		for i := len(sels) - 1; i >= 0; i-- {
			sn := sels[i].(*FileNode)
			nodes = append(nodes, sn)
		}
		ge.CloseOpenNodes(nodes) // close before rename because we are async after this
		for _, sn := range nodes {
			fb := core.NewSoloFuncButton(sn).SetFunc(sn.RenameFile)
			fb.Args[0].SetValue(sn.Name)
			fb.CallFunc()
		}
	})
}
