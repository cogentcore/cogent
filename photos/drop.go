// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package photos

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
)

// Photos and videos dragged onto the app from other apps, as the Finder,
// are added to the library, as copies, renamed by their dates: onto the
// photos or the viewer, with the tag shown, if any, and onto a tag, with
// that tag. The photos of the library dragged within the app are not.

// dropFiles makes the given widget take drops of files from other apps,
// which are added with the tag that the given function returns.
func (a *App) dropFiles(w core.Widget, tag func() string) {
	wb := w.AsWidget()
	wb.Styler(func(s *styles.Style) { s.SetAbilities(true, abilities.Droppable) })
	wb.On(events.Drop, func(e events.Event) {
		de, ok := e.(*events.DragDrop)
		if !ok || a.lib == nil {
			return
		}
		files := a.droppedMedia(de)
		if len(files) == 0 {
			return // as photos dragged within the app
		}
		e.SetHandled()
		a.addFiles(files, tag())
	})
}

// shownTag returns the tag shown, or "" for All and the others.
func (a *App) shownTag() string {
	tag, _ := strings.CutPrefix(a.shown, "tag:")
	if tag == a.shown {
		return ""
	}
	return tag
}

// droppedMedia returns the files of photos and videos in the given drop,
// with those in the folders in it, except for those in the library.
func (a *App) droppedMedia(de *events.DragDrop) []string {
	lib := filepath.Clean(a.lib.Dir) + string(filepath.Separator)
	var files []string
	add := func(f string) {
		if isMediaFile(f) && !strings.HasPrefix(filepath.Clean(f), lib) {
			files = append(files, f)
		}
	}
	for _, f := range core.DroppedFiles(de) {
		filepath.WalkDir(f, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() && strings.HasPrefix(d.Name(), ".") && path != f {
				return filepath.SkipDir
			}
			if !d.IsDir() {
				add(path)
			}
			return nil
		})
	}
	return files
}

// addFiles adds copies of the given files of photos and videos to the
// library, with the given tag, if any, in the background.
func (a *App) addFiles(files []string, tag string) {
	var tags []string
	if tag != "" {
		tags = []string{tag}
	}
	go func() {
		a.busy.Lock()
		defer a.busy.Unlock()
		phs, err := a.lib.Import(files, false, tags, a.progress("Adding"))
		a.AsyncLock()
		defer a.AsyncUnlock()
		a.statusText = ""
		core.ErrorSnackbar(a, err, "Error adding photos")
		msg := fmt.Sprintf("Added %d photos", len(phs))
		if tag != "" {
			msg += " tagged " + tag
		}
		core.MessageSnackbar(a, msg)
		a.refresh()
	}()
}
