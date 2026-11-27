// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package photos implements Cogent Photos, an app for a library of
// photos, in one directory, named by the dates they were taken, with tags
// as albums, which a photo can have any number of, as the successor of
// GoPix.
package photos

import (
	"cmp"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"

	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/tree"
)

// App is the Cogent Photos app: the tags on the left, the grid of the
// photos with the selected tag in the middle, and the current photo on
// the right.
type App struct {
	core.Frame

	// Dir, if set, is the directory of the library instead of that in
	// the [Settings], as given on the command line.
	Dir string

	// lib is the library of photos, once it is open.
	lib *Library

	// shown is the key of the tag row shown (see [tagRow.key]).
	shown string

	// searchText is the text of the current search, and searchResults
	// are the names of the photos it found, or nil for no search.
	searchText    string
	searchResults map[string]bool

	// tagRows are the rows of the tree of tags.
	tagRows []*tagRow

	splits  *core.Splits
	tags    *core.TreeTable
	grid    *Grid
	viewer  *Viewer
	status  *core.Text
	search  *core.TextField
	toolbar *core.Toolbar

	// statusText is the text of the status, as of the progress of a scan.
	statusText string

	// windows are the viewers of the windows of photos open, which
	// follow the current photo in the grid (see [App.OpenWindow]).
	windows []*Viewer

	// syncCancel stops the sync with Google Photos in progress, if any.
	syncCancel func()

	// busy is held during long operations on the library, as scans.
	busy sync.Mutex
}

// tagRow is a row of the tree of tags: All, Trash, a year, or a tag.
type tagRow struct {

	// Name is the name shown: of a tag, without its year.
	Name string `edit:"-"`

	// Photos is the number of photos.
	Photos int `edit:"-"`

	// Children are the tags of a year.
	Children []*tagRow `display:"-"`

	// Closed is whether a year is closed.
	Closed bool `display:"-" treetable:"closed"`

	// key is the key of the row: all, trash, year:2017, or tag:2017_Kyoto.
	key string

	// tag is the tag of a tag row.
	tag string

	// year is the year of a year row.
	year string
}

// The keys of the special rows of the tree of tags.
const (
	allKey   = "all"
	trashKey = "trash"
)

func (a *App) Init() {
	a.Frame.Init()
	a.shown = cmp.Or(Settings.ShownTag, allKey)
	a.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
	})
	a.OnFirst(events.KeyChord, a.appKeys)

	tree.AddChild(a, func(w *core.Splits) {
		a.splits = w
		w.SetSplits(0.2, 0.5, 0.3)
		if len(Settings.Splits) == 3 {
			w.SetSplits(Settings.Splits...)
		}
		w.OnChange(func(e events.Event) {
			Settings.Splits = w.Splits()
			saveSettings(a)
		})
		tree.AddChild(w, func(w *core.Frame) { // the sidebar
			w.Styler(func(s *styles.Style) {
				s.Direction = styles.Column
			})
			tree.AddChild(w, a.initTags)
			tree.AddChild(w, func(w *core.Separator) {})
			tree.AddChild(w, func(w *core.Text) {
				a.status = w
				w.SetType(core.TextSupporting)
				w.Updater(func() {
					w.SetText(a.statusText)
				})
			})
		})
		tree.AddChild(w, func(w *core.Frame) { // the photos
			w.Styler(func(s *styles.Style) {
				s.Direction = styles.Column
			})
			tree.AddChild(w, func(w *core.Frame) { // the search and sort
				w.Styler(func(s *styles.Style) {
					s.Align.Items = styles.Center
				})
				tree.AddChild(w, func(w *core.TextField) {
					a.search = w
					w.SetPlaceholder("Search, as kyoto 2017").SetLeadingIcon(icons.Search)
					w.SetType(core.TextFieldOutlined)
					w.Styler(func(s *styles.Style) {
						s.Min.X.Em(10)
						s.Max.X.Zero() // stretched
						s.Grow.Set(1, 0)
					})
					w.OnInput(func(e events.Event) {
						a.SearchPhotos(w.Text())
					})
					w.OnChange(func(e events.Event) {
						a.SearchPhotos(w.Text())
					})
				})
				tree.AddChild(w, func(w *core.Icon) {
					w.SetIcon(icons.PhotoSizeSelectLarge)
				})
				tree.AddChild(w, func(w *core.Slider) {
					w.SetMin(48).SetMax(512).SetStep(8)
					w.SetTooltip("The size of the photos")
					w.FinalStyler(func(s *styles.Style) { // after the 20em of sliders
						s.Min.X.Em(8)
						s.Max.X.Em(8)
						s.Grow.Set(0, 0)
					})
					w.Updater(func() {
						w.SetValue(Settings.ThumbSize)
					})
					w.OnInput(func(e events.Event) {
						a.setThumbSize(w.Value)
					})
					w.OnChange(func(e events.Event) {
						a.setThumbSize(w.Value)
						saveSettings(a)
					})
				})
				tree.AddChild(w, func(w *core.Chooser) {
					core.Bind(&Settings.Sort, w)
					w.SetTooltip("The order of the photos, by the date they were taken")
					w.OnChange(func(e events.Event) {
						saveSettings(a)
						a.grid.SetPhotos(nil) // from the start
						a.refresh()
					})
				})
			})
			tree.AddChild(w, func(w *core.Frame) {
				w.Styler(func(s *styles.Style) {
					s.Direction = styles.Column
					s.Grow.Set(1, 1)
					s.Overflow.Set(styles.OverflowAuto)
					s.Min.Set(units.Em(10))
				})
				tree.AddChild(w, func(w *Grid) {
					a.grid = w
					w.CellSize = units.Dp(Settings.ThumbSize)
					w.OnSelect(func(e events.Event) {
						a.viewer.SetPhoto(w.CurrentPhoto())
						for _, v := range a.windows {
							v.SetPhoto(w.CurrentPhoto())
						}
						a.updateToolbar()
					})
					w.OnDoubleClick(func(e events.Event) {
						a.OpenWindow()
					})
					w.AddContextMenu(a.photoMenu)
					a.dropFiles(w, a.shownTag)
				})
			})
		})
		tree.AddChild(w, func(w *Viewer) {
			a.viewer = w
			w.app = a
			a.dropFiles(w, a.shownTag)
		})
	})
}

// MakeToolbar makes the toolbar of the app.
func (a *App) MakeToolbar(p *tree.Plan) {
	tree.Add(p, func(w *core.FuncButton) {
		a.toolbar, _ = w.Parent.(*core.Toolbar)
		w.SetFunc(core.SettingsWindow).SetText("").SetIcon(icons.Settings)
		w.SetTooltip("Settings")
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(a.Rescan).SetIcon(icons.Refresh)
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(a.ImportFolder).SetIcon(icons.Download)
	})
	if a.syncCancel != nil {
		tree.Add(p, func(w *core.FuncButton) {
			w.SetFunc(a.CancelSync).SetIcon(icons.Cancel)
		})
	} else {
		tree.Add(p, func(w *core.FuncButton) {
			w.SetFunc(a.SyncGoogle).SetText("Sync").SetIcon(icons.Sync).SetTooltip("Add photos from Google Photos, picked there")
		})
	}
	tree.Add(p, func(w *core.Separator) {})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(a.RotateLeft).SetIcon(icons.RotateLeft).SetShortcut("Command+L")
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(a.RotateRight).SetIcon(icons.RotateRight).SetShortcut("Command+R")
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(a.Tag).SetIcon(icons.Label).SetShortcut("Command+T")
	})
	if strings.HasPrefix(a.shown, "tag:") {
		tree.Add(p, func(w *core.FuncButton) {
			w.SetFunc(a.Untag).SetIcon(icons.LabelOff)
		})
	}
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(a.SetDate).SetIcon(icons.CalendarMonth)
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(a.SaveMetadata).SetIcon(icons.Save)
	})
	tree.Add(p, func(w *core.Separator) {})
	if a.shown == trashKey {
		tree.Add(p, func(w *core.FuncButton) {
			w.SetFunc(a.Restore).SetIcon(icons.RestoreFromTrash)
		})
		tree.Add(p, func(w *core.FuncButton) {
			w.SetFunc(a.EmptyTrash).SetConfirm(true).SetIcon(icons.DeleteForever)
		})
	} else {
		tree.Add(p, func(w *core.FuncButton) {
			w.SetFunc(a.Trash).SetIcon(icons.Delete)
		})
	}
	tree.Add(p, func(w *core.Separator) {})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(a.Info).SetIcon(icons.Info)
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(a.Map).SetIcon(icons.Map)
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(a.Open).SetIcon(icons.OpenInNew)
	})
}

// updateToolbar updates the toolbar, for the photos selected and the tag shown.
func (a *App) updateToolbar() {
	if a.toolbar != nil {
		a.toolbar.Update()
	}
}

// Start opens the library and shows its photos, scanning it for changes
// in the background, and first importing GoPix if the library is new.
func (a *App) Start() {
	go func() {
		a.busy.Lock()
		defer a.busy.Unlock()
		dir := cmp.Or(a.Dir, Settings.libraryDir())
		lib, err := OpenLibrary(dir)
		if err != nil {
			a.AsyncLock()
			core.ErrorDialog(a, err, "Error opening the library of photos in "+dir)
			a.AsyncUnlock()
			return
		}
		core.TheApp.AddQuitCleanFunc(func() {
			a.saveView() // for the next start
			core.SaveSettings(Settings)
			lib.Close()
		})
		a.AsyncLock()
		a.lib = lib
		a.grid.lib = lib
		a.refresh()
		a.restoreView()
		a.AsyncUnlock()

		a.setStatus("Scanning the photos…")
		progress := a.progress("Scanning")
		if lib.Len() == 0 && GoPixDir() != "" && !lib.GoPixImported() {
			a.setStatus("Importing the photos of GoPix…")
			err = lib.ScanGoPix(GoPixDir(), progress)
		} else {
			err = lib.Scan(nil, progress)
			if err == nil && GoPixDir() != "" {
				err = lib.ImportGoPixVideos(GoPixDir())
			}
		}
		a.AsyncLock()
		if err != nil {
			core.ErrorSnackbar(a, err, "Error scanning the photos")
		}
		a.statusText = ""
		a.refresh()
		a.AsyncUnlock()

		if err := lib.updateVideoThumbs(); err != nil {
			slog.Error("photos: updating the thumbnails of videos", "err", err)
		}
		lib.MakeThumbs(a.progress("Thumbnails"))
		a.setStatus("")
	}()
}

// progress returns a [ScanProgress] that shows the progress in the status,
// at most every quarter second.
func (a *App) progress(what string) ScanProgress {
	var last time.Time
	return func(done, total int) {
		if time.Since(last) < 250*time.Millisecond && done < total {
			return
		}
		last = time.Now()
		a.setStatus(fmt.Sprintf("%s: %d of %d", what, done, total))
	}
}

// setStatus sets the text of the status, from any goroutine.
func (a *App) setStatus(text string) {
	a.AsyncLock()
	defer a.AsyncUnlock()
	a.statusText = text
	if a.status != nil {
		a.status.Update()
	}
}

// refresh updates the tags and the photos shown, as after they change,
// keeping the current photo and the selected ones.
func (a *App) refresh() {
	if a.lib == nil {
		return
	}
	a.updateTags()
	cur := a.grid.CurrentPhoto()
	sel := a.grid.SelectedPhotos()
	phs := a.shownPhotos()
	a.grid.SetPhotos(phs)
	if len(sel) > 0 {
		for i, ph := range phs {
			if slices.Contains(sel, ph) {
				a.grid.Selected[i] = true
			}
			if ph == cur {
				a.grid.Current = i
			}
		}
	}
	if a.grid.Current < 0 && len(phs) > 0 {
		a.grid.Current = 0
		a.grid.Selected[0] = true
	}
	a.viewer.SetPhoto(a.grid.CurrentPhoto())
	a.updateToolbar()
	a.Update()
	a.grid.Defer(func() { a.grid.scrollToIndex(max(a.grid.Current, 0)) })
}

// shownPhotos returns the photos of the tag shown, that match the search.
func (a *App) shownPhotos() []*Photo {
	key := a.shown
	var fun func(ph *Photo) bool
	switch {
	case key == trashKey:
		fun = func(ph *Photo) bool { return !ph.Trashed.IsZero() }
	case strings.HasPrefix(key, "tag:"):
		tag := strings.TrimPrefix(key, "tag:")
		fun = func(ph *Photo) bool { return ph.Trashed.IsZero() && ph.HasTag(tag) }
	case strings.HasPrefix(key, "year:"):
		year := strings.TrimPrefix(key, "year:")
		fun = func(ph *Photo) bool {
			if !ph.Trashed.IsZero() {
				return false
			}
			for _, t := range ph.Tags {
				if y, _ := splitYearTag(t); y == year {
					return true
				}
			}
			return false
		}
	default:
		fun = func(ph *Photo) bool { return ph.Trashed.IsZero() }
	}
	if a.searchResults != nil {
		inner := fun
		fun = func(ph *Photo) bool { return a.searchResults[ph.Name] && inner(ph) }
	}
	phs := a.lib.Photos(fun)
	if Settings.Sort == NewestFirst {
		slices.Reverse(phs)
	}
	return phs
}

// SearchPhotos shows the photos of the tag shown that match the given search
// (see [searchIndex.query]): words that match the start of words in the
// tags, dates (as 2017 or february), descriptions, cameras, and names.
func (a *App) SearchPhotos(search string) {
	if a.lib == nil || search == a.searchText {
		return
	}
	a.searchText = search
	res, err := a.lib.Search(search)
	if err != nil {
		slog.Error("photos: search", "err", err)
	}
	a.searchResults = res
	if res == nil && strings.TrimSpace(search) != "" {
		a.searchResults = map[string]bool{}
	}
	a.refresh()
}

// showTag shows the photos of the tag row with the given key.
func (a *App) showTag(key string) {
	if key == a.shown {
		return
	}
	a.saveView()
	a.shown = key
	Settings.ShownTag = key
	saveSettings(a)
	a.grid.SetPhotos(nil)
	a.refresh()
	a.restoreView()
}

// TagView is where the photos of a tag were viewed last: the scroll
// position of the grid, and the current photo, for going back there.
type TagView struct {

	// Scroll is the scroll position of the grid, in dots.
	Scroll float32

	// Current is the name of the current photo.
	Current string
}

// gridFrame returns the scrolling frame of the grid.
func (a *App) gridFrame() *core.Frame {
	fr, _ := a.grid.Parent.(*core.Frame)
	return fr
}

// saveView saves where the photos of the tag shown are viewed (see [TagView]).
func (a *App) saveView() {
	fr := a.gridFrame()
	if fr == nil || a.lib == nil {
		return
	}
	if Settings.Views == nil {
		Settings.Views = map[string]TagView{}
	}
	tv := TagView{Scroll: fr.Geom.Scroll.Y}
	if ph := a.grid.CurrentPhoto(); ph != nil {
		tv.Current = ph.Name
	}
	Settings.Views[a.shown] = tv
}

// restoreView goes back to where the photos of the tag shown were viewed
// last, if they were (see [TagView]).
func (a *App) restoreView() {
	tv, ok := Settings.Views[a.shown]
	fr := a.gridFrame()
	if !ok || fr == nil {
		return
	}
	g := a.grid
	for i, ph := range g.Photos {
		if ph.Name == tv.Current {
			g.Current = i
			g.Selected = map[int]bool{i: true}
			a.viewer.SetPhoto(ph)
			a.updateToolbar()
			break
		}
	}
	g.Defer(func() { // after the scroll to the current photo in refresh
		fr.Geom.Scroll.Y = tv.Scroll
		fr.ScrollUpdateFromGeom(math32.Y)
		g.NeedsRender()
	})
}

// initTags initializes the tree of tags.
func (a *App) initTags(w *core.TreeTable) {
	a.tags = w
	w.SetReadOnly(true)
	w.ReadOnlyKeyNav = true
	w.FitWidth = true // the names in the width of the sidebar
	w.HeaderIcons = map[string]icons.Icon{"Photos": icons.Photo}
	w.ContextMenus = nil
	w.AddContextMenu(a.tagMenu)
	w.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
	})
	w.TableStyler = func(w core.Widget, s *styles.Style, row, col int) {
		r := a.tagRowAt(row)
		if r == nil {
			return
		}
		if r.key == allKey || r.key == trashKey {
			s.Font.Weight = rich.Bold
		}
		if r.key == trashKey && r.Photos == 0 {
			s.Color = colors.Scheme.OnSurfaceVariant
		}
	}
	w.OnSelect(func(e events.Event) {
		if r := a.tagRowAt(w.SelectedIndex); r != nil {
			a.showTag(r.key)
		}
	})
	w.On(events.Drop, a.dropOnTags) // before that of the tree table
}

// tagRowAt returns the visible row of the tree of tags at the given index,
// or nil.
func (a *App) tagRowAt(idx int) *tagRow {
	if a.tags == nil {
		return nil
	}
	rows, ok := a.tags.Slice.(*[]*tagRow)
	if !ok || idx < 0 || idx >= len(*rows) {
		return nil
	}
	return (*rows)[idx]
}

// updateTags makes the rows of the tree of tags: All and Trash, the years
// with their tags, newest first, and then the tags without years.
func (a *App) updateTags() {
	if a.lib == nil {
		return
	}
	for _, r := range a.tagRows { // the years closed now
		if r.year != "" {
			if Settings.ClosedYears == nil {
				Settings.ClosedYears = map[string]bool{}
			}
			Settings.ClosedYears[r.year] = r.Closed
		}
	}
	all, trash := 0, 0
	yearCounts := map[string]int{}
	for _, ph := range a.lib.Photos(nil) {
		if !ph.Trashed.IsZero() {
			trash++
			continue
		}
		all++
		years := map[string]bool{}
		for _, t := range ph.Tags {
			if y, _ := splitYearTag(t); y != "" {
				years[y] = true
			}
		}
		for y := range years {
			yearCounts[y]++
		}
	}
	rows := []*tagRow{
		{Name: "All", Photos: all, key: allKey},
		{Name: "Trash", Photos: trash, key: trashKey},
	}
	years := map[string]*tagRow{}
	var other []*tagRow
	for tag, n := range a.lib.Tags() {
		year, name := splitYearTag(tag)
		r := &tagRow{Name: name, Photos: n, key: "tag:" + tag, tag: tag}
		if year == "" {
			other = append(other, r)
			continue
		}
		yr := years[year]
		if yr == nil {
			closed, ok := Settings.ClosedYears[year]
			yr = &tagRow{Name: year, Photos: yearCounts[year], key: "year:" + year, year: year, Closed: !ok || closed}
			years[year] = yr
		}
		yr.Children = append(yr.Children, r)
	}
	byName := func(a, b *tagRow) int { return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)) }
	var yrs []*tagRow
	for _, yr := range years {
		slices.SortFunc(yr.Children, byName)
		yrs = append(yrs, yr)
	}
	slices.SortFunc(yrs, func(a, b *tagRow) int { return strings.Compare(b.Name, a.Name) })
	slices.SortFunc(other, byName)
	rows = append(rows, yrs...)
	rows = append(rows, other...)
	a.tagRows = rows
	if a.tags != nil {
		a.tags.SetItems(&a.tagRows)
		a.tags.Update()
		a.selectShownTag()
	}
}

// selectShownTag selects the row of the tag shown in the tree of tags,
// opening its year if needed.
func (a *App) selectShownTag() {
	for _, yr := range a.tagRows {
		for _, r := range yr.Children {
			if r.key == a.shown && yr.Closed {
				yr.Closed = false
				a.tags.Update()
			}
		}
	}
	rows, ok := a.tags.Slice.(*[]*tagRow)
	if !ok {
		return
	}
	for i, r := range *rows {
		if r.key == a.shown {
			a.tags.SelectedIndex = i
			a.tags.ResetSelectedIndexes()
			a.tags.SelectedIndexes[i] = struct{}{}
			a.tags.NeedsRender()
			return
		}
	}
}

// dropOnTags handles a drop of photos onto the tree of tags, which tags
// them with the tag they are dropped on, moves them to the trash, or out
// of it onto All.
func (a *App) dropOnTags(e events.Event) {
	de, ok := e.(*events.DragDrop)
	if !ok || a.lib == nil {
		return
	}
	md, ok := de.Data.(mimedata.Mimes)
	if !ok {
		return
	}
	phs := a.lib.draggedPhotos(md)
	files := a.droppedMedia(de)
	if len(phs) == 0 && len(files) == 0 {
		return // something else, for the tree table
	}
	e.SetHandled()
	a.tags.Scene.Events.DropFinalize(de)
	idx, ok := a.tags.IndexFromEvent(e)
	r := a.tagRowAt(idx)
	if len(files) > 0 { // from other apps: added, with the tag, if any
		tag := ""
		if ok && r != nil {
			tag = r.tag
		}
		a.addFiles(files, tag)
		return
	}
	if !ok || r == nil {
		return
	}
	var err error
	switch {
	case r.key == trashKey:
		err = a.lib.Trash(phs...)
	case r.key == allKey:
		err = a.lib.Restore(phs...)
	case r.tag != "":
		err = a.lib.AddTag(r.tag, phs...)
	default:
		return
	}
	core.ErrorSnackbar(a, err)
	if r.tag != "" {
		core.MessageSnackbar(a, fmt.Sprintf("Tagged %d photos %s", len(phs), r.tag))
	}
	a.refresh()
}

// appKeys handles the keys of the app.
func (a *App) appKeys(e events.Event) {
	switch keymap.Of(e.KeyChord()) {
	case keymap.Find, keymap.Search:
		e.SetHandled()
		a.search.SetFocus()
	case keymap.Delete, keymap.Backspace:
		if a.grid.StateIs(states.Focused) || a.viewer.StateIs(states.Focused) {
			e.SetHandled()
			if a.shown == trashKey {
				return
			}
			a.Trash()
		}
	case keymap.ZoomIn:
		e.SetHandled()
		a.zoomThumbs(1.25)
	case keymap.ZoomOut:
		e.SetHandled()
		a.zoomThumbs(0.8)
	}
}

// zoomThumbs changes the size of the thumbnails by the given factor.
func (a *App) zoomThumbs(factor float32) {
	a.setThumbSize(Settings.ThumbSize * factor)
	saveSettings(a)
	a.Update() // the slider
}

// setThumbSize sets the size of the thumbnails, in dp, keeping the
// current photo in view.
func (a *App) setThumbSize(size float32) {
	Settings.ThumbSize = max(48, min(512, size))
	a.grid.CellSize = units.Dp(Settings.ThumbSize)
	a.grid.NeedsLayout()
	a.grid.Defer(func() { a.grid.scrollToIndex(max(a.grid.Current, 0)) })
}
