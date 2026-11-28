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
// the given tag, if any, as 2026/Trip.
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
		if !e.IsDir() && isMediaFile(e.Name()) {
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
// Videos are rotated in the background, as their files are copied.
func (a *App) rotate(degrees int) {
	phs := a.selected()
	if len(phs) == 0 {
		return
	}
	a.viewer.stopVideo() // (its file is replaced)
	go func() {
		a.busy.Lock()
		defer a.busy.Unlock()
		var errs []error
		progress := a.progress("Rotating")
		for i, ph := range phs {
			if len(phs) > 1 {
				progress(i, len(phs))
			}
			errs = append(errs, a.lib.Rotate(ph, degrees))
		}
		a.AsyncLock()
		defer a.AsyncUnlock()
		a.statusText = ""
		for _, ph := range phs {
			a.grid.Forget(ph)
		}
		core.ErrorSnackbar(a, errors.Join(errs...), "Error rotating")
		a.viewer.SetPhoto(a.grid.CurrentPhoto())
		a.status.Update()
	}()
}

// Tag adds a tag to the selected photos, chosen from the tags, or a new
// one, in a group, as 2026 or People, or none. A new tag that starts with
// a year, as 2026 Trip, is in the group of that year when it is in none.
func (a *App) Tag() { //types:add
	phs := a.selected()
	if len(phs) == 0 {
		return
	}
	group, name := splitTag(Settings.LastTag) // the one chosen last, at first
	if Settings.LastTag == "" {
		if g, ok := strings.CutPrefix(a.shown, "group:"); ok {
			group = g
		} else if t, ok := strings.CutPrefix(a.shown, "tag:"); ok {
			group, _ = splitTag(t)
		}
	}
	d := core.NewBody("Tag")
	core.NewText(d).SetType(core.TextSupporting).SetText(fmt.Sprintf("Tag %d photos with a tag, or a new one, in a group, or none:", len(phs)))
	var tag func() string
	tag = a.tagFields(d, group, name, func() { // Return
		if t := tag(); t != "" {
			d.Close()
			a.tagWith(t, phs)
		}
	})
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).SetText("Tag").OnClick(func(e events.Event) {
			if t := tag(); t != "" {
				a.tagWith(t, phs)
			}
		})
	})
	d.RunDialog(a)
}

// tagWith tags the given photos with the given tag, saving it as the tag
// chosen last (see [SettingsData.LastTag]).
func (a *App) tagWith(tag string, phs []*Photo) {
	Settings.LastTag = tag
	saveSettings(a)
	core.ErrorSnackbar(a, a.lib.AddTag(tag, phs...))
	a.refresh()
}

// tagFields adds the fields for choosing a tag to the given dialog: a
// group, from those there are, or a new one, or none, and the name of the
// tag, from those in the group, or a new one, starting with the given
// group and name, with the name selected, so that typing replaces it,
// and Return in it calling accept. It returns a function that returns
// the tag chosen.
func (a *App) tagFields(d *core.Body, group, name string, accept func()) func() string {
	row := core.NewFrame(d)
	row.Styler(func(s *styles.Style) {
		s.Align.Items = styles.Center
	})
	// new ones are typed without completion popups, which would take
	// Return, and those there are are in the menu
	gc := core.NewChooser(row).SetEditable(true).SetDefaultNew(true)
	gc.SetPlaceholder("Group (none)")
	gc.SetTooltip("The group of the tag, as 2026 or People/Family, or none, for the top")
	for _, g := range a.lib.Groups() {
		gc.Items = append(gc.Items, core.ChooserItem{Value: g})
	}
	if group != "" {
		gc.SetCurrentValue(group)
	}
	gc.Styler(func(s *styles.Style) { s.Min.X.Em(12) })
	core.NewText(row).SetText(tagSep)
	tc := core.NewChooser(row).SetEditable(true).SetDefaultNew(true)
	tc.SetPlaceholder("Tag")
	tc.Styler(func(s *styles.Style) { s.Min.X.Em(16) })
	groupText := func() string {
		if tf := gc.TextField(); tf != nil {
			return cleanTag(tf.Text())
		}
		return group
	}
	setNames := func() { // those in the group
		g := groupText()
		tc.Items = nil
		for t := range a.lib.Tags() {
			if tg, n := splitTag(t); tg == g {
				tc.Items = append(tc.Items, core.ChooserItem{Value: n})
			}
		}
		slices.SortFunc(tc.Items, func(a, b core.ChooserItem) int {
			return strings.Compare(strings.ToLower(fmt.Sprint(a.Value)), strings.ToLower(fmt.Sprint(b.Value)))
		})
	}
	setNames()
	if name != "" {
		tc.SetCurrentValue(name)
	}
	gc.OnChange(func(e events.Event) { setNames() })
	hooked := false
	tc.OnShow(func(e events.Event) {
		tf := tc.TextField()
		if tf == nil {
			return
		}
		d.Scene.Events.SetStartFocus(tf) // rather than the group
		if !hooked {
			hooked = true
			selected := false
			tf.On(events.Focus, func(e events.Event) { // selected when focused, at first
				if !selected {
					selected = true
					tf.Defer(tf.SelectAll)
				}
			})
			// Return accepts the tag as it is
			tf.OnFirst(events.KeyChord, func(e events.Event) {
				if keymap.Of(e.KeyChord()) == keymap.Enter {
					e.SetHandled()
					accept()
				}
			})
		}
	})
	return func() string {
		n := ""
		if tf := tc.TextField(); tf != nil {
			n = tf.Text()
		}
		if strings.TrimSpace(n) == "" {
			return ""
		}
		return joinTag(groupText(), n)
	}
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
	if a.shown != allKey || a.searchResults != nil || a.grid != a.main {
		core.NewFuncButton(m).SetFunc(a.ShowInAll).SetIcon(icons.PhotoLibrary)
	}
	if ph := a.grid.CurrentPhoto(); ph != nil && !ph.GPS.IsZero() {
		core.NewFuncButton(m).SetFunc(a.PhotosNearby).SetIcon(icons.MyLocation)
	}
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
	in := "" // the group of a new group
	switch {
	case r != nil && r.group != "":
		in = r.group
	case r != nil && r.tag != "":
		in, _ = splitTag(r.tag)
	}
	text := "New group"
	if in != "" {
		text = "New group in " + tagLabel(in)
	}
	core.NewButton(m).SetText(text).SetIcon(icons.CreateNewFolder).OnClick(func(e events.Event) {
		a.newGroupDialog(in)
	})
	if in != "" {
		core.NewButton(m).SetText("New group at the top").SetIcon(icons.CreateNewFolder).OnClick(func(e events.Event) {
			a.newGroupDialog("")
		})
	}
	if r == nil {
		return
	}
	core.NewSeparator(m)
	if r.group != "" {
		core.NewButton(m).SetText("Rename group").SetIcon(icons.Edit).OnClick(func(e events.Event) {
			a.renameGroupDialog(r.group)
		})
		core.NewButton(m).SetText("Delete group").SetIcon(icons.Delete).
			SetTooltip("Deletes the group, moving the tags and groups in it up a level").OnClick(func(e events.Event) {
			parent, _ := splitTag(r.group)
			core.ErrorSnackbar(a, a.lib.DeleteGroup(r.group))
			a.movedKeys(groupKey(r.group), groupKey(parent), true)
			a.refresh()
		})
		return
	}
	if r.tag == "" {
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

// newGroupDialog opens a dialog for making a new group in the given one,
// or at the top.
func (a *App) newGroupDialog(in string) {
	title := "New group"
	if in != "" {
		title += " in " + tagLabel(in)
	}
	d := core.NewBody(title)
	tf := core.NewTextField(d).SetPlaceholder("Name, as People or 2026")
	tf.Styler(func(s *styles.Style) { s.Min.X.Em(20) })
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).SetText("Make").OnClick(func(e events.Event) {
			name := strings.ReplaceAll(strings.TrimSpace(tf.Text()), tagSep, " ")
			if name == "" {
				return
			}
			g := joinTag(in, name)
			if in == "" {
				g = name // not split as a tag of a year
			}
			core.ErrorSnackbar(a, a.lib.AddGroup(g))
			Settings.ClosedGroups[g] = false
			a.refresh()
		})
	})
	d.RunDialog(a)
}

// renameGroupDialog opens a dialog for renaming the given group, in the
// group it is in.
func (a *App) renameGroupDialog(group string) {
	parent, name := splitTag(group)
	d := core.NewBody("Rename group")
	tf := core.NewTextField(d).SetText(name)
	tf.Styler(func(s *styles.Style) { s.Min.X.Em(20) })
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).SetText("Rename").OnClick(func(e events.Event) {
			nn := strings.ReplaceAll(strings.TrimSpace(tf.Text()), tagSep, " ")
			if nn == "" || nn == name {
				return
			}
			ng := joinTag(parent, nn)
			if parent == "" {
				ng = nn
			}
			core.ErrorSnackbar(a, a.lib.RenameGroup(group, ng))
			a.movedKeys(groupKey(group), groupKey(ng), false)
			a.refresh()
		})
	})
	d.RunDialog(a)
}

// renameTagDialog opens a dialog for renaming the given tag, which can
// also move it into another group.
func (a *App) renameTagDialog(tag string) {
	group, name := splitTag(tag)
	d := core.NewBody("Rename tag")
	var nt func() string
	rename := func() {
		t := nt()
		if t == "" || t == tag {
			return
		}
		core.ErrorSnackbar(a, a.lib.RenameTag(tag, t))
		a.movedKeys(tagKey(tag), tagKey(t), false)
		a.refresh()
	}
	nt = a.tagFields(d, group, name, func() {
		d.Close()
		rename()
	})
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).SetText("Rename").OnClick(func(e events.Event) { rename() })
	})
	d.RunDialog(a)
}

// movedKeys changes the given key of a tag or group that was renamed or
// moved into the new one, as the key shown and the key of its view, and
// those of the tags and groups in a group. If gone, a group is gone,
// with those in it moved into the new one.
func (a *App) movedKeys(old, key string, gone bool) {
	oldGroup, isGroup := strings.CutPrefix(old, "group:")
	newGroup := strings.TrimPrefix(key, "group:")
	move := func(k string) string {
		if k == old {
			return key
		}
		if !isGroup {
			return k
		}
		for _, p := range []string{"tag:", "group:"} {
			if t, ok := strings.CutPrefix(k, p); ok && inGroup(t, oldGroup) {
				rest := strings.TrimPrefix(t, oldGroup+tagSep)
				if gone {
					return p + joinTag(newGroup, rest)
				}
				return p + newGroup + tagSep + rest
			}
		}
		return k
	}
	if gone && a.shown == old {
		a.shown = allKey
	}
	a.shown = move(a.shown)
	Settings.ShownTag = a.shown
	for k, v := range Settings.Views {
		if nk := move(k); nk != k {
			delete(Settings.Views, k)
			Settings.Views[nk] = v
		}
	}
	if isGroup && !gone {
		if c, ok := Settings.ClosedGroups[oldGroup]; ok {
			Settings.ClosedGroups[newGroup] = c
		}
	}
	saveSettings(a)
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

// ShowInAll shows the current photo in All, with the photos taken at
// about the same time around it.
func (a *App) ShowInAll() { //types:add
	ph := a.current()
	if ph == nil {
		return
	}
	a.searchText, a.searchResults = "", nil // all of them
	if a.search != nil {
		a.search.SetText("")
	}
	if Settings.Views == nil {
		Settings.Views = map[string]TagView{}
	}
	// the photo is current in All, which is scrolled to it (see [App.restoreView])
	Settings.Views[allKey] = TagView{Current: ph.Name, Scroll: 1}
	a.tabs.SelectTabIndex(0)
	if a.shown == allKey {
		a.refresh()
		a.restoreView()
		return
	}
	a.showTag(allKey)
}
