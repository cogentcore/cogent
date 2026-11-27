// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package photos

import (
	"errors"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
)

// selected returns the selected photos, or shows a message and returns
// nil if there are none.
func (a *App) selected() []*Photo {
	if a.lib == nil {
		return nil
	}
	phs := a.grid.SelectedPhotos()
	if len(phs) == 0 {
		if ph := a.grid.CurrentPhoto(); ph != nil {
			return []*Photo{ph}
		}
		core.MessageSnackbar(a, "Select one or more photos first")
	}
	return phs
}

// current returns the current photo, or shows a message and returns nil.
func (a *App) current() *Photo {
	ph := a.grid.CurrentPhoto()
	if ph == nil && a.lib != nil {
		core.MessageSnackbar(a, "Select a photo first")
	}
	return ph
}

// Rescan scans the photos for new, changed, and deleted files in the
// library, as after adding files to its All directory.
func (a *App) Rescan() { //types:add
	if a.lib == nil {
		return
	}
	go func() {
		a.busy.Lock()
		defer a.busy.Unlock()
		err := a.lib.Scan(nil, a.progress("Scanning"))
		a.AsyncLock()
		defer a.AsyncUnlock()
		a.statusText = ""
		core.ErrorSnackbar(a, err, "Error scanning the photos")
		a.refresh()
	}()
}

// ImportFolder imports the photos in the given folder into the library,
// copying them, or moving them if move, renamed by their dates, and with
// the given tag, if any, as 2026_Trip.
func (a *App) ImportFolder(folder core.Filename, move bool, tag string) { //types:add
	if a.lib == nil {
		return
	}
	ents, err := os.ReadDir(string(folder))
	if err != nil {
		core.ErrorSnackbar(a, err)
		return
	}
	var files []string
	for _, e := range ents {
		if !e.IsDir() && isImageFile(e.Name()) {
			files = append(files, filepath.Join(string(folder), e.Name()))
		}
	}
	var tags []string
	if tag = strings.TrimSpace(tag); tag != "" {
		tags = []string{tag}
	}
	go func() {
		a.busy.Lock()
		defer a.busy.Unlock()
		phs, err := a.lib.Import(files, move, tags, a.progress("Importing"))
		a.AsyncLock()
		defer a.AsyncUnlock()
		a.statusText = ""
		core.ErrorSnackbar(a, err, "Error importing photos")
		core.MessageSnackbar(a, fmt.Sprintf("Imported %d photos", len(phs)))
		a.refresh()
	}()
}

// RotateLeft rotates the selected photos 90 degrees counterclockwise.
func (a *App) RotateLeft() { //types:add
	a.rotate(270)
}

// RotateRight rotates the selected photos 90 degrees clockwise.
func (a *App) RotateRight() { //types:add
	a.rotate(90)
}

// rotate rotates the selected photos by the given degrees clockwise.
func (a *App) rotate(degrees int) {
	var errs []error
	for _, ph := range a.selected() {
		errs = append(errs, a.lib.Rotate(ph, degrees))
		a.grid.Forget(ph)
	}
	core.ErrorSnackbar(a, errors.Join(errs...))
	a.viewer.SetPhoto(a.grid.CurrentPhoto())
}

// Tag adds a tag to the selected photos, chosen from the tags, or a new
// one. A tag that starts with a year and an underscore, as 2026_Trip,
// is shown in that year.
func (a *App) Tag() { //types:add
	phs := a.selected()
	if len(phs) == 0 {
		return
	}
	d := core.NewBody("Tag")
	core.NewText(d).SetType(core.TextSupporting).SetText(fmt.Sprintf("Tag %d photos with a tag, or a new one (as 2026_Trip, in 2026):", len(phs)))
	ch := core.NewChooser(d).SetEditable(true).SetAllowNew(true)
	for _, t := range a.sortedTags() {
		ch.Items = append(ch.Items, core.ChooserItem{Value: t})
	}
	ch.Styler(func(s *styles.Style) {
		s.Min.X.Em(20)
	})
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).SetText("Tag").OnClick(func(e events.Event) {
			tag := strings.TrimSpace(fmt.Sprint(ch.CurrentItem.Value))
			if ch.CurrentItem.Value == nil || tag == "" {
				return
			}
			core.ErrorSnackbar(a, a.lib.AddTag(tag, phs...))
			a.refresh()
		})
	})
	d.RunDialog(a)
}

// sortedTags returns all of the tags, the newest years first.
func (a *App) sortedTags() []string {
	var tags []string
	for _, r := range a.tagRows {
		if r.tag != "" {
			tags = append(tags, r.tag)
		}
		for _, c := range r.Children {
			tags = append(tags, c.tag)
		}
	}
	return tags
}

// Untag removes the tag shown from the selected photos.
func (a *App) Untag() { //types:add
	tag, ok := strings.CutPrefix(a.shown, "tag:")
	if !ok {
		return
	}
	phs := a.selected()
	if len(phs) == 0 {
		return
	}
	core.ErrorSnackbar(a, a.lib.RemoveTag(tag, phs...))
	a.refresh()
}

// SetDate sets the date the selected photos were taken, as for scans of
// old photos, which renames them by it. Each photo after the first is
// taken the given number of days and minutes after the one before it.
func (a *App) SetDate(date time.Time, dayIncrement, minuteIncrement int) { //types:add
	phs := a.selected()
	inc := time.Duration(dayIncrement)*24*time.Hour + time.Duration(minuteIncrement)*time.Minute
	var errs []error
	for _, ph := range phs {
		errs = append(errs, a.lib.SetDate(ph, date))
		a.grid.Forget(ph)
		date = date.Add(inc)
	}
	core.ErrorSnackbar(a, errors.Join(errs...))
	a.refresh()
}

// Trash moves the selected photos to the trash, where they are deleted
// for good after 30 days.
func (a *App) Trash() { //types:add
	phs := a.selected()
	if len(phs) == 0 {
		return
	}
	next := a.grid.Current + 1 - len(phs)
	core.ErrorSnackbar(a, a.lib.Trash(phs...))
	a.grid.Selected = map[int]bool{}
	a.refresh()
	if len(a.grid.Photos) > 0 {
		a.grid.SelectIndex(max(0, min(next, len(a.grid.Photos)-1)))
	}
}

// Restore moves the selected photos out of the trash.
func (a *App) Restore() { //types:add
	phs := a.selected()
	if len(phs) == 0 {
		return
	}
	core.ErrorSnackbar(a, a.lib.Restore(phs...))
	a.grid.Selected = map[int]bool{}
	a.refresh()
}

// EmptyTrash deletes all of the photos in the trash for good.
func (a *App) EmptyTrash() { //types:add
	if a.lib == nil {
		return
	}
	core.ErrorSnackbar(a, a.lib.EmptyTrash())
	a.refresh()
}

// Info shows the information about the current photo, with all of the
// metadata in its file. The description and tags can be edited.
func (a *App) Info() { //types:add
	ph := a.current()
	if ph == nil {
		return
	}
	d := core.NewBody("Photo " + ph.Filename())
	core.NewForm(d).SetStruct(ph).OnChange(func(e events.Event) {
		core.ErrorSnackbar(a, a.lib.Save(ph))
		a.refresh()
	})
	if tags, err := exifTags(a.lib.PhotoFile(ph)); err == nil && len(tags) > 0 {
		core.NewText(d).SetType(core.TextTitleSmall).SetText("Metadata")
		var b strings.Builder
		for _, t := range tags {
			fmt.Fprintf(&b, "%s: %s\n", t[0], t[1])
		}
		core.NewText(d).SetText(b.String())
	}
	d.AddOKOnly().RunWindowDialog(a)
}

// Map shows the location of the current photo in Google Maps.
func (a *App) Map() { //types:add
	ph := a.current()
	if ph == nil {
		return
	}
	if ph.GPS.IsZero() {
		core.MessageSnackbar(a, "The photo has no location")
		return
	}
	core.TheApp.OpenURL(fmt.Sprintf("https://www.google.com/maps/search/?api=1&query=%g,%g", ph.GPS.Lat, ph.GPS.Long))
}

// Open opens the current photo in the default app for it, as for editing.
func (a *App) Open() { //types:add
	ph := a.current()
	if ph == nil {
		return
	}
	core.TheApp.OpenURL("file://" + a.lib.PhotoFile(ph))
}

// photoMenu is the context menu of the photos.
func (a *App) photoMenu(m *core.Scene, pos image.Point) {
	if idx, ok := a.grid.indexAt(pos); ok && !a.grid.Selected[idx] {
		a.grid.SelectIndex(idx)
	}
	core.NewFuncButton(m).SetFunc(a.OpenWindow).SetIcon(icons.OpenInFull)
	core.NewFuncButton(m).SetFunc(a.Info).SetIcon(icons.Info)
	core.NewFuncButton(m).SetFunc(a.Tag).SetIcon(icons.Label)
	if strings.HasPrefix(a.shown, "tag:") {
		core.NewFuncButton(m).SetFunc(a.Untag).SetIcon(icons.LabelOff)
	}
	core.NewFuncButton(m).SetFunc(a.SetDate).SetIcon(icons.CalendarMonth)
	core.NewFuncButton(m).SetFunc(a.SaveMetadata).SetIcon(icons.Save)
	core.NewSeparator(m)
	core.NewFuncButton(m).SetFunc(a.RotateLeft).SetIcon(icons.RotateLeft)
	core.NewFuncButton(m).SetFunc(a.RotateRight).SetIcon(icons.RotateRight)
	core.NewSeparator(m)
	if a.shown == trashKey {
		core.NewFuncButton(m).SetFunc(a.Restore).SetIcon(icons.RestoreFromTrash)
	} else {
		core.NewFuncButton(m).SetFunc(a.Trash).SetIcon(icons.Delete)
	}
	core.NewFuncButton(m).SetFunc(a.Open).SetIcon(icons.OpenInNew)
}

// tagMenu is the context menu of the tree of tags.
func (a *App) tagMenu(m *core.Scene, pos image.Point) {
	r := a.tagRowAt(a.tags.SelectedIndex)
	if r == nil || r.tag == "" {
		return
	}
	core.NewButton(m).SetText("Rename tag").SetIcon(icons.Edit).OnClick(func(e events.Event) {
		a.renameTagDialog(r.tag)
	})
	core.NewButton(m).SetText("Delete tag").SetIcon(icons.Delete).OnClick(func(e events.Event) {
		core.ErrorSnackbar(a, a.lib.RenameTag(r.tag, ""))
		a.showTag(allKey)
		a.refresh()
	})
}

// renameTagDialog opens a dialog for renaming the given tag.
func (a *App) renameTagDialog(tag string) {
	d := core.NewBody("Rename tag")
	tf := core.NewTextField(d).SetText(tag)
	tf.Styler(func(s *styles.Style) { s.Min.X.Em(20) })
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).SetText("Rename").OnClick(func(e events.Event) {
			nt := strings.TrimSpace(tf.Text())
			if nt == "" || nt == tag {
				return
			}
			core.ErrorSnackbar(a, a.lib.RenameTag(tag, nt))
			a.shown = "tag:" + nt
			a.refresh()
		})
	})
	d.RunDialog(a)
}

// OpenWindow opens the current photo in its own window, as large as the
// screen allows, where the left and right keys go to the previous and
// next photos.
func (a *App) OpenWindow() { //types:add
	ph := a.current()
	if ph == nil {
		return
	}
	b := core.NewBody("Photo")
	tree.AddChild(b, func(w *Viewer) {
		w.app = a
		a.windows = append(a.windows, w)
		w.Styler(func(s *styles.Style) {
			s.Min.Set(units.Vw(60), units.Vh(60))
		})
		w.OnShow(func(e events.Event) {
			w.SetPhoto(a.grid.CurrentPhoto())
			w.SetFocus()
		})
		w.On(events.KeyChord, func(e events.Event) {
			if keymap.Of(e.KeyChord()) == keymap.Abort {
				e.SetHandled()
				w.Scene.Close()
			}
		})
	})
	b.OnClose(func(e events.Event) {
		a.windows = slices.DeleteFunc(a.windows, func(v *Viewer) bool { return v == b.Child(0) })
	})
	b.RunWindow()
}

// SaveMetadata writes the dates and orientations of the selected photos,
// or of all of them, into the metadata of their files where they differ,
// so that other apps, such as Google Photos, show the dates set here, as
// for scans of old photos. Set date and rotating JPEG photos write them
// already. Only JPEG files can be written: other files, other than HEIC
// photos from phones, are converted to JPEG if convert, with the original
// files kept in Trash/Converted, and otherwise skipped.
func (a *App) SaveMetadata(allPhotos, convert bool) { //types:add
	if a.lib == nil {
		return
	}
	var phs []*Photo
	if allPhotos {
		phs = a.lib.Photos(func(ph *Photo) bool { return ph.Trashed.IsZero() })
	} else {
		phs = a.selected()
	}
	if len(phs) == 0 {
		return
	}
	go func() {
		a.busy.Lock()
		defer a.busy.Unlock()
		n, skipped, err := a.lib.WriteMetadata(phs, convert, a.progress("Saving metadata"))
		a.AsyncLock()
		defer a.AsyncUnlock()
		a.statusText = ""
		core.ErrorSnackbar(a, err, "Error saving metadata")
		msg := fmt.Sprintf("Saved the metadata of %d photos", n)
		if skipped > 0 {
			msg += fmt.Sprintf("; skipped %d that are not JPEG (convert them to save it)", skipped)
		}
		core.MessageSnackbar(a, msg)
		a.refresh()
	}()
}
