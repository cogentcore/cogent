// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"image"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/core"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/tree"
)

// DirTable is a table of the files in a directory, under its full path:
// the table of files of [core.FilePicker] without its other parts. Double-clicking a directory
// shows its files, and a file opens it in its default app.
type DirTable struct {
	core.Frame

	// Dir is the absolute path of the directory (see [DirTable.SetDir]).
	Dir string `set:"-"`

	// files are the files in the directory.
	files []*fileinfo.FileInfo

	// table is the table of the files.
	table *core.Table

	// watched is the directory being watched for changes.
	watched string

	// stopWatch stops watching the directory for changes, if set.
	stopWatch context.CancelFunc
}

func (dt *DirTable) Init() {
	dt.Frame.Init()
	dt.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
	})
	dt.Updater(func() {
		dt.readFiles()
		if dt.watched != dt.Dir {
			dt.watchDir()
		}
	})
	dt.OnKeyChord(func(e events.Event) {
		switch keymap.Of(e.KeyChord()) {
		case keymap.Jump, keymap.WordLeft:
			e.SetHandled()
			dt.DirUp()
		case keymap.Accept, keymap.Open:
			e.SetHandled()
			dt.openSelected()
		}
	})
	tree.AddChildAt(dt, "path", func(w *core.Text) {
		w.SetType(core.TextTitleSmall)
		w.Styler(func(s *styles.Style) {
			s.Grow.Set(1, 0)
		})
		w.Updater(func() {
			w.SetText(dt.Dir)
		})
	})
	tree.AddChildAt(dt, "files", func(w *core.Table) {
		dt.table = w
		w.SetReadOnly(true)
		w.SetSlice(&dt.files)
		w.Styler(func(s *styles.Style) {
			s.Cursor = cursors.Pointer
			s.Grow.Set(1, 1)
			s.SetAbilities(true, abilities.Draggable, abilities.Droppable)
		})
		w.OnDoubleClick(func(e events.Event) {
			e.SetHandled()
			dt.openSelected()
		})
		dt.handleDND(w)
		w.ContextMenus = nil
		w.AddContextMenu(dt.contextMenu)
	})
}

func (dt *DirTable) Destroy() {
	if dt.stopWatch != nil {
		dt.stopWatch()
		dt.stopWatch = nil
	}
	dt.Frame.Destroy()
}

// SetDir sets the directory to show the files of, sending a change event.
func (dt *DirTable) SetDir(dir string) *DirTable {
	dt.Dir = cleanDir(dir)
	if dt.table != nil {
		dt.table.ResetSelectedIndexes()
	}
	dt.Update()
	dt.SendChange()
	return dt
}

// DirUp shows the files of the parent directory.
func (dt *DirTable) DirUp() {
	dt.SetDir(filepath.Dir(dt.Dir))
}

// SelectedFile returns the selected file, or nil.
func (dt *DirTable) SelectedFile() *fileinfo.FileInfo {
	if dt.table == nil {
		return nil
	}
	idx := dt.table.SelectedIndex
	if idx < 0 || idx >= len(dt.files) {
		return nil
	}
	return dt.files[idx]
}

// FocusTable moves the keyboard focus to the table of the files, after
// it is next rendered, as it may have just been made or shown.
func (dt *DirTable) FocusTable() {
	dt.Defer(func() {
		if dt.table != nil {
			dt.table.SetFocus()
		}
	})
	dt.NeedsRender()
}

// openSelected shows the files of the selected directory,
// or opens the selected file in its default app.
func (dt *DirTable) openSelected() {
	fi := dt.SelectedFile()
	if fi == nil {
		return
	}
	if fi.IsDir() {
		dt.SetDir(fi.Path)
		return
	}
	openFile(fi.Path)
}

// readFiles reads the files in the directory, with the directories first.
func (dt *DirTable) readFiles() {
	dt.files = dt.files[:0]
	ents, err := os.ReadDir(dt.Dir)
	if err != nil {
		slog.Error("Files: could not read the directory", "Dir", dt.Dir, "Error", err)
		return
	}
	for _, ent := range ents {
		if !Settings.ShowHidden && strings.HasPrefix(ent.Name(), ".") {
			continue
		}
		fi, err := fileinfo.NewFileInfo(filepath.Join(dt.Dir, ent.Name()))
		if err == nil {
			dt.files = append(dt.files, fi)
		}
	}
	slices.SortStableFunc(dt.files, func(a, b *fileinfo.FileInfo) int {
		if a.IsDir() == b.IsDir() {
			return 0
		}
		if a.IsDir() {
			return -1
		}
		return 1
	})
}

// watchDir watches the directory for changes, updating the files.
func (dt *DirTable) watchDir() {
	if dt.stopWatch != nil {
		dt.stopWatch()
		dt.stopWatch = nil
	}
	dt.watched = dt.Dir
	ctx, cancel := context.WithCancel(context.Background())
	err := watchDir(ctx, dt.Dir, func() {
		// not in the event loop, so update with the lock, if still shown
		if dt.AsyncLockIfShown() {
			dt.Update()
			dt.AsyncUnlock()
		}
	})
	if err != nil {
		cancel()
		return // e.g., a directory that does not exist
	}
	dt.stopWatch = cancel
}

// contextMenu adds the items of the context menu of the table.
func (dt *DirTable) contextMenu(m *core.Scene, pos image.Point) {
	fi := dt.SelectedFile()
	core.NewButton(m).SetText("Open").SetIcon(icons.Open).
		SetTooltip("Open the selected directory, or the selected file in its default app").
		SetState(fi == nil, states.Disabled).
		OnClick(func(e events.Event) {
			dt.openSelected()
		})
	if fi != nil && !fi.IsDir() {
		openWithMenuItems(m, fi.Path)
	}
	if fi == nil || fi.IsDir() { // the selected directory, or this one
		dir := dt.Dir
		if fi != nil {
			dir = fi.Path
		}
		if fl, ok := ParentFiles(dt); ok {
			fl.dirMenuItems(m, dir)
		}
	}
	if fi == nil {
		core.NewSeparator(m)
		core.NewFuncButton(m).SetFunc(dt.NewFolder).SetIcon(icons.CreateNewFolder)
		return
	}
	core.NewSeparator(m)
	core.NewButton(m).SetText("Duplicate").SetIcon(icons.FileCopy).
		SetTooltip("Make a copy of the selected file").
		OnClick(func(e events.Event) {
			errors.Log1(fi.Duplicate())
			dt.Update()
		})
	tip := "Delete moves the selected file to the trash / recycling bin"
	core.NewButton(m).SetText("Delete").SetIcon(icons.Delete).
		SetTooltip(tip).
		OnClick(func(e events.Event) {
			fb := core.NewSoloFuncButton(dt).SetFunc(fi.Delete).SetConfirm(true).SetAfterFunc(dt.Update)
			fb.SetTooltip(tip)
			fb.CallFunc()
		})
	core.NewButton(m).SetText("Rename").SetIcon(icons.EditNote).
		SetTooltip("Rename the selected file").
		OnClick(func(e events.Event) {
			core.NewSoloFuncButton(dt).SetFunc(fi.Rename).SetAfterFunc(dt.Update).CallFunc()
		})
	core.NewButton(m).SetText("Info").SetIcon(icons.Info).
		SetTooltip("View information about the selected file").
		OnClick(func(e events.Event) {
			d := core.NewBody("Info: " + fi.Name)
			core.NewForm(d).SetStruct(fi).SetReadOnly(true)
			d.AddOKOnly().RunWindowDialog(dt)
		})
	core.NewSeparator(m)
	core.NewFuncButton(m).SetFunc(dt.NewFolder).SetIcon(icons.CreateNewFolder)
}

// NewFolder makes a new folder with the given name in the directory.
func (dt *DirTable) NewFolder(name string) error { //types:add
	if name == "" {
		return errors.New("the name of the new folder is empty")
	}
	err := os.MkdirAll(filepath.Join(dt.Dir, name), 0750)
	if err != nil {
		return err
	}
	dt.Update()
	return nil
}
