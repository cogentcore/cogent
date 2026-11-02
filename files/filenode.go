// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"image"

	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/filetree"
	"cogentcore.org/core/icons"
)

// FileNode is the node of the file tree, which shows the files of a
// directory in a directory tab, and opens a file in its default app,
// when double-clicked; View in the context menu of a directory also
// shows its files.
type FileNode struct {
	filetree.Node
}

func (fn *FileNode) Init() {
	fn.Node.Init()
	fn.AddContextMenu(func(m *core.Scene, pos image.Point) {
		if !fn.IsDir() {
			return
		}
		fl, ok := ParentFiles(fn.This)
		if !ok {
			return
		}
		core.NewButton(m).SetText("View").SetIcon(icons.FolderOpen).
			SetTooltip("View the files of this directory in a directory tab, as double-clicking does").
			OnClick(func(e events.Event) {
				fl.OpenDir(string(fn.Filepath))
			})
		fl.dirMenuItems(m, string(fn.Filepath))
		core.NewSeparator(m)
	})
	fn.Parts.OnDoubleClick(func(e events.Event) {
		e.SetHandled()
		if fn.IsDir() {
			if fl, ok := ParentFiles(fn.This); ok {
				fl.OpenDir(string(fn.Filepath))
			}
			return
		}
		core.TheApp.OpenURL(fsx.FileURL(string(fn.Filepath)))
	})
}
