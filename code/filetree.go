// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"log"
	"path/filepath"

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
	fn.Parts.OnDoubleClick(func(e events.Event) {
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
						sn.Open()
					} else {
						sn.ToggleClose()
					}
				} else {
					ge.FileNodeOpened(sn)
				}
				done = true
			}
		})
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
		ge.NextViewFile(string(fn.Filepath))
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
		ge.Settings.RunExec = fn.Filepath
		ge.Settings.BuildDir = core.Filename(filepath.Dir(string(fn.Filepath)))
	}
}

// ExecCmdFile pops up a menu to select a command appropriate for the given node,
// and shows output in MainTab with name of command
func (fn *FileNode) ExecCmdFile() { //types:add
	cv, ok := ParentCode(fn.This)
	if ok {
		cv.ExecCmdFileNode(&fn.Node)
	}
}

// ExecCmdNameFile executes given command name on node
func (fn *FileNode) ExecCmdNameFile(cmdNm string) {
	cv, ok := ParentCode(fn.This)
	if ok {
		cv.ExecCmdNameFile(string(fn.Node.Filepath), CmdName(cmdNm))
	}
}

func (fn *FileNode) ContextMenu(m *core.Scene) {
	cv, ok := ParentCode(fn.This)
	if ok {
		core.NewButton(m).SetText("Exec Cmd").SetIcon(icons.Terminal).
			SetMenu(cv.CommandMenuFileNode(&fn.Node)).Styler(func(s *styles.Style) {
			s.SetState(!fn.HasSelection(), states.Disabled)
		})
	}
	core.NewFuncButton(m).SetFunc(fn.EditFiles).SetText("Edit").SetIcon(icons.Edit).
		Styler(func(s *styles.Style) {
			s.SetState(!fn.HasSelection(), states.Disabled)
		})
	core.NewFuncButton(m).SetFunc(fn.SetRunExecs).SetText("Set Run Exec").
		SetIcon(icons.PlayArrow).
		Styler(func(s *styles.Style) {
			s.SetState(!fn.HasSelection() || !fn.IsExec(), states.Disabled)
		})
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
	cv, ok := ParentCode(fn.This)
	if !ok {
		return
	}
	cv.SaveAllCheck(true, func() {
		paths := fn.SelectedPaths()
		cv.CloseOpenFiles(paths) // close before rename because we are async after this
		fn.Node.RenameFiles()
	})
}

// DeleteFiles calls DeleteFile on any selected nodes, after prompting.
func (fn *FileNode) DeleteFiles() {
	cv, ok := ParentCode(fn.This)
	if !ok {
		return
	}
	d := core.NewBody("Delete Files?")
	core.NewText(d).SetType(core.TextSupporting).SetText("OK to delete file(s)?  This is not undoable and files are not moving to trash / recycle bin. If any selections are directories all files and subdirectories will also be deleted.")
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).SetText("Delete Files").OnClick(func(e events.Event) {
			paths := fn.SelectedPaths()
			cv.CloseOpenFiles(paths) // close before rename because we are async after this
			fn.DeleteFilesNoPrompts()
		})
	})
	d.RunDialog(fn)
}
