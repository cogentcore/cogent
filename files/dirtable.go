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

	"cogentcore.org/core/base/dirsort"
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/core"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/system"
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
	// (final, for those of the table of the files, which has the focus)
	dt.OnFinal(events.KeyChord, func(e events.Event) {
		switch keymap.Of(e.KeyChord()) {
		case keymap.Jump, keymap.WordLeft:
			e.SetHandled()
			dt.DirUp()
		case keymap.Accept, keymap.Open:
			e.SetHandled()
			dt.openSelected()
		case keymap.Delete, keymap.Backspace, keymap.DeleteWord, keymap.BackspaceWord:
			// (BackspaceWord is Command+Backspace on macOS, as in the Finder)
			if fis := dt.SelectedFiles(); len(fis) > 0 {
				e.SetHandled()
				dt.deleteFiles(fis)
			}
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
		// several files can be selected, with the modifier keys, to drag,
		// delete or duplicate them together
		w.ReadOnlyMultiSelect = true
		w.SetSlice(&dt.files)
		// sorted in the sort order of the directory, with the directories
		// first, which the user sets for it, for all apps, by sorting by a
		// column (see [dirsort.For])
		w.SortFunc = dt.sortFiles
		w.OnSort = func(field string, down bool) {
			if s, ok := dirsort.Parse(field); ok {
				s.Down = down
				errors.Log(dirsort.Set(dt.Dir, s))
			}
		}
		w.Updater(func() {
			w.SetSortFieldName(dirsort.For(dt.Dir).String())
		})
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

// SelectedFiles returns the selected files, in the order of the table.
func (dt *DirTable) SelectedFiles() []*fileinfo.FileInfo {
	if dt.table == nil {
		return nil
	}
	var fis []*fileinfo.FileInfo
	for _, idx := range dt.table.SelectedIndexesList(false) {
		if idx >= 0 && idx < len(dt.files) {
			fis = append(fis, dt.files[idx])
		}
	}
	return fis
}

// deleteFiles moves the given files to the trash / recycling bin, after
// asking whether to.
func (dt *DirTable) deleteFiles(fis []*fileinfo.FileInfo) {
	names := make([]string, len(fis))
	for i, fi := range fis {
		names[i] = fi.Name
	}
	d := core.NewBody("Delete?")
	core.NewText(d).SetType(core.TextSupporting).SetText("Move to the trash: " + strings.Join(names, ", ") + "?")
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).SetText("Delete").OnClick(func(e events.Event) {
			var errs []error
			for _, fi := range fis {
				errs = append(errs, fi.Delete())
			}
			dt.Update()
			core.ErrorSnackbar(dt, errors.Join(errs...), "Error deleting file")
		})
	})
	d.RunDialog(dt)
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
	if fi.IsDir() && !opensAsApp(fi.Path, true) {
		dt.SetDir(fi.Path)
		return
	}
	openFile(fi.Path)
}

// readFiles reads the files in the directory (which the table sorts).
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
}

// sortFiles sorts the files by the field of the given name, up or down, in
// the order of [dirsort.Sort], with the directories first, and returns
// whether it is a field of it.
func (dt *DirTable) sortFiles(field string, down bool) bool {
	s, ok := dirsort.Parse(field)
	if !ok {
		return false
	}
	s.Down = down
	s.SortFiles(dt.files)
	slices.SortStableFunc(dt.files, func(a, b *fileinfo.FileInfo) int {
		if a.IsDir() == b.IsDir() {
			return 0
		}
		if a.IsDir() {
			return -1
		}
		return 1
	})
	return true
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
	if fi != nil && isApp(fi.Path, fi.IsDir()) {
		openAppMenuItem(m, fi.Path)
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
	fis := dt.SelectedFiles()
	single := len(fis) <= 1
	core.NewSeparator(m)
	core.NewButton(m).SetText("Duplicate").SetIcon(icons.FileCopy).
		SetTooltip("Make a copy of each of the selected files").
		OnClick(func(e events.Event) {
			for _, fi := range fis {
				errors.Log1(fi.Duplicate())
			}
			dt.Update()
		})
	core.NewButton(m).SetText("Delete").SetIcon(icons.Delete).SetShortcut(deleteShortcut()).
		SetTooltip("Delete moves the selected files to the trash / recycling bin").
		OnClick(func(e events.Event) {
			dt.deleteFiles(fis)
		})
	core.NewButton(m).SetText("Rename").SetIcon(icons.EditNote).SetEnabled(single).
		SetTooltip("Rename the selected file").
		OnClick(func(e events.Event) {
			core.NewSoloFuncButton(dt).SetFunc(fi.Rename).SetAfterFunc(dt.Update).CallFunc()
		})
	core.NewButton(m).SetText("Info").SetIcon(icons.Info).SetEnabled(single).
		SetTooltip("View information about the selected file").
		OnClick(func(e events.Event) {
			d := core.NewBody("Info: " + fi.Name)
			core.NewForm(d).SetStruct(fi).SetReadOnly(true)
			d.AddOKOnly().RunWindowDialog(dt)
		})
	core.NewSeparator(m)
	core.NewFuncButton(m).SetFunc(dt.NewFolder).SetIcon(icons.CreateNewFolder)
}

// deleteShortcut returns the shortcut for deleting the selected files
// that is shown in the context menu: Command+Backspace on macOS, as in the
// Finder, or else the delete key (all of them work: see [DirTable.Init]).
func deleteShortcut() key.Chord {
	if core.TheApp.SystemPlatform() == system.MacOS {
		return "Meta+Backspace"
	}
	return keymap.Delete.Chord()
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
