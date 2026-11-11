// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles/states"
)

// handleDND sets up drag-and-drop for the table of files, as in
// [core.FilePicker]: files can be dragged out of it, to other widgets,
// windows and apps, and dropped files are copied (or moved) into the
// directory.
func (dt *DirTable) handleDND(w *core.Table) {
	w.On(events.DragStart, func(e events.Event) {
		dt.dragStart(e)
	})
	w.On(events.DragEnter, func(e events.Event) {
		w.SetState(true, states.DragHovered)
		w.NeedsRender()
		e.SetHandled()
	})
	w.On(events.DragLeave, func(e events.Event) {
		w.SetState(false, states.DragHovered)
		w.NeedsRender()
		e.SetHandled()
	})
	w.On(events.Drop, func(e events.Event) {
		dt.drop(e)
	})
}

// rowAt returns the index of the file in the row of the table at the given
// position, and a widget of the row, or -1 if there is no row there.
func (dt *DirTable) rowAt(pos image.Point) (int, core.Widget) {
	w := dt.table
	nw, _ := w.RowWidgetNs()
	for i, c := range w.ListGrid.Children {
		cw := c.(core.Widget)
		if !pos.In(cw.AsWidget().Geom.TotalBBox) {
			continue
		}
		idx := i/nw + w.StartIndex
		if idx >= len(dt.files) {
			return -1, nil
		}
		return idx, cw
	}
	return -1, nil
}

// dragStart starts dragging the selected files, if the file under the
// mouse is one of them, or else the file under the mouse, selecting it.
func (dt *DirTable) dragStart(e events.Event) {
	idx, rw := dt.rowAt(e.Pos())
	if idx < 0 {
		return
	}
	if _, selected := dt.table.SelectedIndexes[idx]; !selected {
		dt.table.SelectIndexEvent(idx, events.SelectOne)
	}
	var paths []string
	md := mimedata.Mimes{}
	for _, fi := range dt.SelectedFiles() {
		paths = append(paths, fi.Path)
		md = append(md, mimedata.NewTextData(fi.Path))
	}
	if len(paths) == 0 {
		return
	}
	dt.Scene.Events.DragStart(rw, md, e)
	dt.Scene.Events.SetDragFiles(paths...)
	e.SetHandled()
}

// drop copies the files dropped on the table into the directory, or moves
// them for [events.DropMove], first asking whether to overwrite any
// existing files of the same names.
func (dt *DirTable) drop(e events.Event) {
	de := e.(*events.DragDrop)
	e.SetHandled()
	dt.table.SetState(false, states.DragHovered)
	paths := core.DroppedFiles(de)
	var existing []string
	for _, p := range paths {
		tp := filepath.Join(dt.Dir, filepath.Base(p))
		if tp != p && fsx.Exists(tp) {
			existing = append(existing, filepath.Base(p))
		}
	}
	move := de.DropMod == events.DropMove
	drop := func() {
		var errs []error
		for _, p := range paths {
			tp := filepath.Join(dt.Dir, filepath.Base(p))
			if tp == p {
				continue
			}
			var err error
			if move {
				err = movePath(tp, p)
			} else {
				err = fsx.CopyPath(tp, p)
			}
			errs = append(errs, err)
		}
		dt.Update()
		core.ErrorSnackbar(dt, errors.Join(errs...), "Error copying file")
	}
	if len(existing) == 0 {
		drop()
		return
	}
	d := core.NewBody("Overwrite?")
	core.NewText(d).SetType(core.TextSupporting).SetText(fmt.Sprintf("Overwrite existing file(s) in %s: %s?", dt.Dir, strings.Join(existing, ", ")))
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).SetText("Overwrite").OnClick(func(e events.Event) {
			drop()
		})
	})
	d.RunDialog(dt)
}

// movePath moves the file or directory at src to dst, overwriting an
// existing file. If it can't be renamed, for example on another drive,
// it is copied, and then deleted if the copy succeeds.
func movePath(dst, src string) error {
	if os.Rename(src, dst) == nil {
		return nil
	}
	if err := fsx.CopyPath(dst, src); err != nil {
		return err
	}
	return os.RemoveAll(src)
}
