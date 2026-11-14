// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"image"
	"path/filepath"
	"reflect"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
	"github.com/mitchellh/go-homedir"
)

// initFavorites configures the table of the favorite paths, which are those
// of [core.FilePicker]: selecting one makes it the Root of the file tree.
func (fl *Files) initFavorites(w *core.Table) {
	w.SetReadOnly(true)
	w.SelectedIndex = -1
	w.SetSlice(&core.SystemSettings.FavPaths)
	w.Styler(func(s *styles.Style) {
		s.Cursor = cursors.Pointer
		s.Grow.Set(1, 1)
	})
	// a right click selects the row of its context menu, which must not
	// change the Root
	contextClick := false
	tree.AddChildInit(w, "grid", func(g *core.ListGrid) {
		g.On(events.MouseDown, func(e events.Event) {
			contextClick = e.MouseButton() == events.Right
		})
	})
	w.On(events.ContextMenu, func(e events.Event) {
		contextClick = false
	})
	w.OnSelect(func(e events.Event) {
		if contextClick {
			return
		}
		if path := favoriteAt(w.SelectedIndex); path != "" {
			fl.SetRoot(path)
		}
	})
	w.Updater(func() {
		w.ResetSelectedIndexes() // so that the same one can be selected again
	})
	w.AddContextMenu(func(m *core.Scene, pos image.Point) {
		if path := favoriteAt(w.SelectedIndex); path != "" {
			core.NewButton(m).SetText("Terminal here").SetIcon(icons.Terminal).
				SetTooltip("Open a new terminal in " + path).
				OnClick(func(e events.Event) {
					fl.NewTerminal(path)
				})
			core.NewSeparator(m)
		}
		core.NewButton(m).SetText("Edit favorites").SetIcon(icons.Edit).
			SetTooltip("Edit the favorite paths, which are shared by all Cogent Core apps").
			OnClick(func(e events.Event) {
				fl.EditFavorites()
			})
	})
}

// favoriteAt returns the expanded path of the favorite path at the given
// index, or "" if there is none.
func favoriteAt(idx int) string {
	fav := core.SystemSettings.FavPaths
	if idx < 0 || idx >= len(fav) {
		return ""
	}
	return errors.Log1(homedir.Expand(fav[idx].Path))
}

// EditFavorites edits the favorite paths in a window,
// saving them when it is closed.
func (fl *Files) EditFavorites() { //types:add
	d := core.NewBody("Favorite paths")
	fl.makeFavoritesEditor(d)
	d.RunWindowDialog(fl)
}

// makeFavoritesEditor makes the editor of the favorite paths in the given
// body, which saves them when it is closed (see [Files.saveFavorites]).
func (fl *Files) makeFavoritesEditor(d *core.Body) {
	core.NewText(d).SetType(core.TextSupporting).
		SetText("The favorite paths are shared by all Cogent Core apps: they are in the Core system settings.")
	core.NewTable(d).SetSlice(&core.SystemSettings.FavPaths)
	d.OnClose(func(e events.Event) {
		fl.saveFavorites()
	})
}

// saveFavorites saves the favorite paths in the [core.SystemSettings],
// and updates the table of them.
func (fl *Files) saveFavorites() {
	core.ErrorSnackbar(fl, core.SaveSettings(core.SystemSettings), "Error saving settings")
	if fv := fl.Favorites(); fv != nil {
		fv.Update()
	}
}

// AddFavorite adds the given directory to the favorite paths, saving them,
// with ~ for the home directory, as [core.FilePicker] does, unless it is
// one of them already.
func (fl *Files) AddFavorite(dir string) {
	path := dir
	if hd, err := homedir.Dir(); err == nil && strings.HasPrefix(dir, hd+string(filepath.Separator)) {
		path = filepath.Join("~", strings.TrimPrefix(dir, hd+string(filepath.Separator)))
	}
	for _, fav := range core.SystemSettings.FavPaths {
		if fav.Path == path {
			core.MessageSnackbar(fl, path+" is already a favorite")
			return
		}
	}
	addFavorite(filepath.Base(dir), path)
	fl.saveFavorites()
}

// addFavorite appends a favorite path with the given name and path to the
// [core.SystemSettings], whose type of item is not exported.
func addFavorite(name, path string) {
	favs := reflect.ValueOf(&core.SystemSettings.FavPaths).Elem()
	item := reflect.New(favs.Type().Elem()).Elem()
	item.FieldByName("Icon").SetString(string(icons.Folder))
	item.FieldByName("Name").SetString(name)
	item.FieldByName("Path").SetString(path)
	favs.Set(reflect.Append(favs, item))
}
