// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"image"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/filetree"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/tree"
)

// FileNode is the node of the file tree, which shows the files of a
// directory in a directory tab, and opens a file in its default app,
// when double-clicked; View in the context menu of a directory also
// shows its files (see [initNode]).
type FileNode struct {
	filetree.Node
}

func (fn *FileNode) Init() {
	fn.Node.Init()
	initNode(&fn.Node)
}

// FileTree is the file tree, whose root node is a directory node like
// the [FileNode]s (see [initNode]).
type FileTree struct {
	filetree.Tree
}

func (ft *FileTree) Init() {
	ft.Tree.Init()
	initNode(&ft.Node)
}

// initNode adds the handlers of the nodes of the file tree to the given one:
// double-clicking shows the files of a directory in a directory tab, as the
// right arrow on an open directory does, and opens a file in its default
// app, as Enter does, and the context menu of a directory has
// View, which also shows its files, and the [Files.dirMenuItems], and that
// of a file has the items of [openWithMenuItems]. App bundles (see
// [isApp]) open as apps.
func initNode(fn *filetree.Node) {
	fn.AddContextMenu(func(m *core.Scene, pos image.Point) {
		if !fn.IsDir() {
			openWithMenuItems(m, string(fn.Filepath))
			core.NewSeparator(m)
			return
		}
		if isApp(string(fn.Filepath), true) {
			openAppMenuItem(m, string(fn.Filepath))
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
	fn.OnFirst(events.KeyChord, func(e events.Event) {
		// the keys of the focused node go to the handlers of its parents first
		if !isFocus(fn) {
			return
		}
		switch kf := keymap.Of(e.KeyChord()); {
		case kf == keymap.Enter && (!fn.IsDir() || opensAsApp(string(fn.Filepath), true)):
			// opens the file, as double-clicking does (directories toggle)
			e.SetHandled()
			openFile(string(fn.Filepath))
		case kf == keymap.MoveRight && fn.IsDir() && !fn.Closed:
			// shows the files of the open directory
			if fl, ok := ParentFiles(fn.This); ok {
				e.SetHandled()
				if dt := fl.OpenDir(string(fn.Filepath)); dt != nil {
					dt.FocusTable()
				}
			}
		}
	})
	fn.Parts.OnDoubleClick(func(e events.Event) {
		e.SetHandled()
		if fn.IsDir() && !opensAsApp(string(fn.Filepath), true) {
			if fl, ok := ParentFiles(fn.This); ok {
				fl.OpenDir(string(fn.Filepath))
			}
			return
		}
		openFile(string(fn.Filepath))
	})
}

// openFile opens the given file in its default app.
// It is a variable for the tests, which do not open apps.
var openFile = func(path string) {
	core.TheApp.OpenURL(fsx.FileURL(path))
}

// isFocus returns whether the given node has the keyboard focus itself,
// not one of its children.
func isFocus(fn *filetree.Node) bool {
	return fn.ContainsFocus() && !slices.ContainsFunc(fn.Children, func(c tree.Node) bool {
		return core.AsWidget(c).ContainsFocus()
	})
}

// appBundles is whether directories named *.app are app bundles, which
// are opened as apps (see [isApp]): on macOS. It is a variable for the tests.
var appBundles = runtime.GOOS == "darwin"

// isApp returns whether the given directory is an app bundle (on macOS),
// which is opened as an app by double-clicking, as in the Finder, unless
// hidden files are shown, in which case it is a directory, whose contents
// can be viewed and edited (it can still be opened from its context menu).
func isApp(path string, isDir bool) bool {
	return appBundles && isDir && strings.EqualFold(filepath.Ext(path), ".app")
}

// opensAsApp returns whether double-clicking the given directory opens it
// as an app (see [isApp]).
func opensAsApp(path string, isDir bool) bool {
	return isApp(path, isDir) && !Settings.ShowHidden
}

// openAppMenuItem adds the context menu item for opening the given app
// bundle (see [isApp]).
func openAppMenuItem(m *core.Scene, path string) {
	core.NewButton(m).SetText("Open app").SetIcon(icons.Open).
		SetTooltip("Open " + filepath.Base(path) + " as an app").
		OnClick(func(e events.Event) { openFile(path) })
}
