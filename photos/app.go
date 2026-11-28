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
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"

	"cogentcore.org/cogent/photos/imagecapture"
	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
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

	splits *core.Splits
	tags   *core.TreeTable

	// grid is the grid of the tab shown, which the actions on photos
	// act on, and main is that of the first tab, of the tag shown, in
	// mainTab, of tabs. The other tabs have the photos of other searches,
	// as of those nearby (see [App.PhotosNearby]).
	grid    *Grid
	main    *Grid
	tabs    *core.Tabs
	mainTab *core.Tab

	// nearby are the tabs of photos nearby others.
	nearby []*nearbyView

	viewer  *Viewer
	status  *core.Text
	search  *core.TextField
	toolbar *core.Toolbar

	// statusText is the text of the status, as of the progress of a scan.
	statusText string

	// allCount and trashCount are the numbers of photos in All and Trash.
	allCount, trashCount int

	// windows are the viewers of the windows of photos open, which
	// follow the current photo in the grid (see [App.OpenWindow]).
	windows []*Viewer

	// syncCancel stops the sync with Google Photos in progress, if any.
	syncCancel func()

	// keySeq1 is the first chord of a key sequence being typed, as
	// Control+X on the Mac (see [App.appKeys]).
	keySeq1 key.Chord

	// importCancel stops the import from a camera or phone in progress, if any.
	importCancel func()

	// busy is held during long operations on the library, as scans.
	busy sync.Mutex
}

// tagRow is a row of the tree of tags: All, Trash, a year, or a tag.
type tagRow struct {

	// Name is the name shown: of a tag or group, without its groups.
	Name string `edit:"-"`

	// Photos is the number of photos.
	Photos int `edit:"-"`

	// Children are the groups and tags in a group.
	Children []*tagRow `display:"-"`

	// Closed is whether a group is closed.
	Closed bool `display:"-" treetable:"closed"`

	// Key is the key of the row: all, trash, group:2017, or tag:2017/Kyoto
	// (see [groupKey] and [tagKey]). It is exported for dragging rows,
	// whose data is JSON.
	Key string `display:"-"`

	// tag is the tag of a tag row.
	tag string

	// group is the group of a group row.
	group string
}

// groupKey and tagKey return the keys of the rows of the given group and
// tag (see [tagRow.Key]).
func groupKey(group string) string { return "group:" + group }
func tagKey(tag string) string     { return "tag:" + tag }

// The keys of the special rows of the tree of tags.
const (
	allKey   = "all"
	trashKey = "trash"
)

func (a *App) Init() {
	a.Frame.Init()
	Settings.migrateTags()
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
			tree.AddChild(w, func(w *core.Frame) { // All and Trash, always there
				w.Styler(func(s *styles.Style) {
					s.Direction = styles.Column
					s.Gap.Zero()
					s.Grow.Set(1, 0)
				})
				w.Maker(func(p *tree.Plan) {
					a.fixedButton(p, allKey, "All", icons.PhotoLibrary, func() int { return a.allCount })
					a.fixedButton(p, trashKey, "Trash", icons.Delete, func() int { return a.trashCount })
				})
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
					s.Grow.Set(1, 0) // the search field fills the rest
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
						a.main.SetPhotos(nil) // from the start
						a.refresh()
					})
				})
			})
			tree.AddChild(w, func(w *core.Tabs) {
				a.tabs = w
				w.SetType(core.FunctionalTabs)
				w.Styler(func(s *styles.Style) {
					s.Grow.Set(1, 1)
				})
				fr, tab := w.NewTab("All")
				a.mainTab = tab
				tab.SetCloseIcon(icons.None) // always there
				g := a.newGridIn(fr)
				a.main, a.grid = g, g
				a.dropFiles(g, a.shownTag)
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
	if a.importCancel != nil {
		tree.Add(p, func(w *core.FuncButton) {
			w.SetFunc(a.CancelImport).SetIcon(icons.Cancel)
		})
	} else {
		tree.Add(p, func(w *core.FuncButton) {
			w.SetFunc(a.ImportFromCamera).SetText("Camera").SetIcon(icons.PhotoCamera)
		})
	}
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
		w.SetFunc(a.RotateLeft).SetIcon(icons.RotateLeft).SetShortcut(shortcutOf("RotateLeft"))
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(a.RotateRight).SetIcon(icons.RotateRight).SetShortcut(shortcutOf("RotateRight"))
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(a.Tag).SetIcon(icons.Label).SetShortcut(shortcutOf("Tag"))
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
		if err := lib.removeGoPixThumbs(); err != nil { // before they are shown
			slog.Error("photos: removing the thumbnails of GoPix", "err", err)
		}
		imagecapture.Start() // finding devices takes a moment
		core.TheApp.AddQuitCleanFunc(func() {
			a.saveView() // for the next start
			core.SaveSettings(Settings)
			lib.Close()
		})
		a.AsyncLock()
		a.lib = lib
		a.main.lib = lib
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
// keeping the current photo and the selected ones, in all of the tabs.
func (a *App) refresh() {
	if a.lib == nil {
		return
	}
	a.updateTags()
	setGridPhotos(a.main, a.shownPhotos())
	if a.mainTab != nil {
		a.mainTab.SetText(a.shownName())
	}
	for _, nv := range a.nearby {
		nv.update()
	}
	a.viewer.SetPhoto(a.grid.CurrentPhoto())
	a.updateToolbar()
	a.Update()
	g := a.main
	g.Defer(func() { g.scrollToIndex(max(g.Current, 0)) })
}

// setGridPhotos sets the photos of the given grid, keeping its current
// photo and the selected ones, or the first one current if there were none.
func setGridPhotos(g *Grid, phs []*Photo) {
	cur := g.CurrentPhoto()
	sel := g.SelectedPhotos()
	g.SetPhotos(phs)
	if len(sel) > 0 {
		for i, ph := range phs {
			if slices.Contains(sel, ph) {
				g.Selected[i] = true
			}
			if ph == cur {
				g.Current = i
			}
		}
	}
	if g.Current < 0 && len(phs) > 0 {
		g.Current = 0
		g.Selected[0] = true
	}
}

// shownName returns the name of the tag shown, for its tab.
func (a *App) shownName() string {
	switch {
	case a.shown == trashKey:
		return "Trash"
	case strings.HasPrefix(a.shown, "tag:"):
		return tagLabel(strings.TrimPrefix(a.shown, "tag:"))
	case strings.HasPrefix(a.shown, "group:"):
		return tagLabel(strings.TrimPrefix(a.shown, "group:"))
	}
	return "All"
}

// newGridIn adds a new grid of photos in a scrolling frame to the given
// frame of a tab, which is the grid of the actions when the tab is shown.
func (a *App) newGridIn(tab *core.Frame) *Grid {
	tab.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
	})
	fr := core.NewFrame(tab)
	fr.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
		s.Overflow.Set(styles.OverflowAuto)
		s.Min.Set(units.Em(10))
	})
	g := NewGrid(fr)
	g.lib = a.lib
	g.CellSize = units.Dp(Settings.ThumbSize)
	g.OnSelect(func(e events.Event) {
		if a.grid != g {
			return
		}
		a.viewer.SetPhoto(g.CurrentPhoto())
		for _, v := range a.windows {
			v.SetPhoto(g.CurrentPhoto())
		}
		a.updateToolbar()
	})
	g.OnDoubleClick(func(e events.Event) {
		a.OpenWindow()
	})
	g.AddContextMenu(a.photoMenu)
	tab.OnShow(func(e events.Event) { // its grid is the one of the actions
		if a.grid != g {
			a.grid = g
			a.viewer.SetPhoto(g.CurrentPhoto())
			a.updateToolbar()
		}
	})
	return g
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
	case strings.HasPrefix(key, "group:"):
		group := strings.TrimPrefix(key, "group:")
		fun = func(ph *Photo) bool {
			return ph.Trashed.IsZero() && slices.ContainsFunc(ph.Tags, func(t string) bool { return inGroup(t, group) })
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
	a.main.SetPhotos(nil)
	a.refresh()
	a.restoreView()
}

// TagView is where the photos of a tag were viewed last: the scroll
// position of the grid, and the current photo, for going back there.
type TagView struct {

	// Scroll is the scroll position of the grid, in dots, or a positive
	// value for the current photo in the middle of the view.
	Scroll float32

	// Current is the name of the current photo.
	Current string
}

// gridFrame returns the scrolling frame of the grid.
func (a *App) gridFrame() *core.Frame {
	fr, _ := a.main.Parent.(*core.Frame)
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
	if ph := a.main.CurrentPhoto(); ph != nil {
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
	g := a.main
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
		if tv.Scroll > 0 { // to the current photo, in the middle
			g.scrollToCenter(g.Current)
			return
		}
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
	w.ReadOnlyDrag = true // for moving tags into groups (see App.moveTagRows)
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
	}
	w.OnSelect(func(e events.Event) {
		if r := a.tagRowAt(w.SelectedIndex); r != nil {
			a.showTag(r.Key)
		}
	})
	w.Styler(func(s *styles.Style) { // (not as a read-only tree table)
		s.SetAbilities(true, abilities.Droppable)
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

// updateTags makes the rows of the tree of tags: the groups, with the
// groups and tags in them, and the tags at the top (see [sortTagRows]).
func (a *App) updateTags() {
	if a.lib == nil {
		return
	}
	if Settings.ClosedGroups == nil {
		Settings.ClosedGroups = map[string]bool{}
	}
	walkTagRows(a.tagRows, func(r *tagRow) { // the groups closed now
		if r.group != "" {
			Settings.ClosedGroups[r.group] = r.Closed
		}
	})
	all, trash := 0, 0
	groupCounts := map[string]int{}
	for _, ph := range a.lib.Photos(nil) {
		if !ph.Trashed.IsZero() {
			trash++
			continue
		}
		all++
		groups := map[string]bool{}
		for _, t := range ph.Tags {
			for _, g := range tagGroups(t) {
				groups[g] = true
			}
		}
		for g := range groups {
			groupCounts[g]++
		}
	}
	a.allCount, a.trashCount = all, trash
	groups := map[string]*tagRow{}
	var rows []*tagRow
	add := func(r *tagRow, group string) {
		if group == "" {
			rows = append(rows, r)
			return
		}
		g := groups[group]
		g.Children = append(g.Children, r)
	}
	gs := a.lib.Groups() // sorted, so groups are before those in them
	for _, g := range gs {
		_, name := splitTag(g)
		closed, ok := Settings.ClosedGroups[g]
		groups[g] = &tagRow{Name: name, Photos: groupCounts[g], Key: groupKey(g), group: g, Closed: !ok || closed}
	}
	for _, g := range gs {
		parent, _ := splitTag(g)
		add(groups[g], parent)
	}
	for tag, n := range a.lib.Tags() {
		group, name := splitTag(tag)
		add(&tagRow{Name: name, Photos: n, Key: tagKey(tag), tag: tag}, group)
	}
	sortTagRows(rows)
	a.tagRows = rows
	if a.tags != nil {
		a.tags.SetItems(&a.tagRows)
		a.tags.Update()
		a.selectShownTag()
	}
}

// sortTagRows sorts the given rows, and those in them: the groups of years
// first, the newest first, then the other groups, and then the tags, by
// their names.
func sortTagRows(rows []*tagRow) {
	rank := func(r *tagRow) int {
		switch {
		case r.group != "" && isYear(r.Name):
			return 0
		case r.group != "":
			return 1
		}
		return 2
	}
	slices.SortFunc(rows, func(a, b *tagRow) int {
		ra, rb := rank(a), rank(b)
		switch {
		case ra != rb:
			return ra - rb
		case ra == 0:
			return strings.Compare(b.Name, a.Name)
		}
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})
	for _, r := range rows {
		sortTagRows(r.Children)
	}
}

// walkTagRows calls the given function on the given rows, and on those
// in them.
func walkTagRows(rows []*tagRow, fun func(r *tagRow)) {
	for _, r := range rows {
		fun(r)
		walkTagRows(r.Children, fun)
	}
}

// selectShownTag selects the row of the tag shown in the tree of tags,
// opening the groups it is in if needed.
func (a *App) selectShownTag() {
	var in []string
	if tag, ok := strings.CutPrefix(a.shown, "tag:"); ok {
		in = tagGroups(tag)
	} else if group, ok := strings.CutPrefix(a.shown, "group:"); ok {
		in = tagGroups(group)
	}
	opened := false
	walkTagRows(a.tagRows, func(r *tagRow) {
		if r.Closed && slices.Contains(in, r.group) {
			r.Closed = false
			opened = true
		}
	})
	if opened {
		a.tags.Update()
	}
	rows, ok := a.tags.Slice.(*[]*tagRow)
	if !ok {
		return
	}
	a.tags.ResetSelectedIndexes() // none for All and Trash
	a.tags.SelectedIndex = -1
	a.tags.NeedsRender()
	for i, r := range *rows {
		if r.Key == a.shown {
			a.tags.SelectedIndex = i
			a.tags.ResetSelectedIndexes()
			a.tags.SelectedIndexes[i] = struct{}{}
			a.tags.NeedsRender()
			return
		}
	}
}

// dropOnTags handles a drop of photos onto the tree of tags, which tags
// them with the tag they are dropped on (see [App.dropOn]).
func (a *App) dropOnTags(e events.Event) {
	idx, ok := a.tags.IndexFromEvent(e)
	var r *tagRow
	if ok {
		r = a.tagRowAt(idx)
	}
	if a.moveTagRows(e, r) {
		return
	}
	tag := ""
	if r != nil {
		tag = r.tag
	}
	a.dropOn(e, a.tags, "", tag)
}

// moveTagRows handles a drop of tags or groups dragged in the tree of tags
// onto the given row, if any, returning whether it was one: they are
// moved into the group dropped on, or the group of the tag dropped on,
// or to the top, when dropped below the rows.
func (a *App) moveTagRows(e events.Event, onto *tagRow) bool {
	de, ok := e.(*events.DragDrop)
	if !ok || a.lib == nil {
		return false
	}
	src, ok := de.Source.(tree.Node)
	for ok && src != nil && src != tree.Node(a.tags) {
		src = src.AsTree().Parent
	}
	md, isMimes := de.Data.(mimedata.Mimes)
	if src == nil || !isMimes {
		return false
	}
	e.SetHandled()
	a.tags.Scene.Events.DropFinalize(de)
	group := ""
	switch {
	case onto == nil:
	case onto.group != "":
		group = onto.group
	default:
		group, _ = splitTag(onto.tag)
	}
	var errs []error
	for _, d := range md {
		var r tagRow
		if json.Unmarshal(d.Data, &r) != nil {
			continue
		}
		if t, ok := strings.CutPrefix(r.Key, "tag:"); ok {
			if g, _ := splitTag(t); g == group {
				continue
			}
			nt, err := a.lib.MoveTag(t, group)
			errs = append(errs, err)
			a.movedKeys(r.Key, tagKey(nt), false)
		} else if g, ok := strings.CutPrefix(r.Key, "group:"); ok {
			if p, _ := splitTag(g); p == group || g == group || inGroup(group, g) {
				continue // where it is, or into itself
			}
			_, name := splitTag(g)
			ng := name
			if group != "" {
				ng = group + tagSep + name
			}
			errs = append(errs, a.lib.RenameGroup(g, ng))
			a.movedKeys(r.Key, groupKey(ng), false)
		}
	}
	if group != "" {
		Settings.ClosedGroups[group] = false
	}
	core.ErrorSnackbar(a, errors.Join(errs...))
	a.refresh()
	return true
}

// dropOn handles a drop onto the given widget for the given key of All or
// Trash, or the given tag: photos of the app dropped onto All are moved
// out of the trash, onto Trash to the trash, and onto a tag are tagged
// with it; files from other apps are added, with the tag, if any. It does
// nothing for other drops, as of tags within the tree of tags.
func (a *App) dropOn(e events.Event, w core.Widget, key, tag string) {
	de, ok := e.(*events.DragDrop)
	if !ok || a.lib == nil {
		return
	}
	var phs []*Photo
	if md, ok := de.Data.(mimedata.Mimes); ok {
		phs = a.lib.draggedPhotos(md)
	}
	files := a.droppedMedia(de)
	if len(phs) == 0 && len(files) == 0 {
		return
	}
	e.SetHandled()
	w.AsWidget().Scene.Events.DropFinalize(de)
	if len(files) > 0 { // from other apps
		if key != trashKey {
			a.addFiles(files, tag)
		}
		return
	}
	var err error
	switch {
	case key == trashKey:
		err = a.lib.Trash(phs...)
	case key == allKey:
		err = a.lib.Restore(phs...)
	case tag != "":
		err = a.lib.AddTag(tag, phs...)
		core.MessageSnackbar(a, fmt.Sprintf("Tagged %d photos %s", len(phs), tag))
	default:
		return
	}
	core.ErrorSnackbar(a, err)
	a.refresh()
}

// fixedButton adds the button of All or Trash, with the given key, above
// the tree of tags, which stays there when the tree is scrolled.
func (a *App) fixedButton(p *tree.Plan, key, name string, icon icons.Icon, count func() int) {
	tree.AddAt(p, key, func(w *core.Button) {
		w.SetType(core.ButtonAction).SetIcon(icon)
		w.Styler(func(s *styles.Style) {
			s.Grow.Set(1, 0)
			s.Min.X.Pw(100) // the width of the sidebar
			s.Justify.Content = styles.Start
			s.SetAbilities(true, abilities.Droppable)
			if a.shown == key {
				s.Background = colors.Scheme.Select.Container
				s.Font.Weight = rich.Bold
			}
		})
		w.Updater(func() {
			w.SetText(fmt.Sprintf("%s  %d", name, count()))
		})
		w.OnClick(func(e events.Event) {
			a.showTag(key)
		})
		w.On(events.Drop, func(e events.Event) {
			a.dropOn(e, w, key, "")
		})
	})
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
	grids := []*Grid{a.main}
	for _, nv := range a.nearby {
		grids = append(grids, nv.grid)
	}
	for _, g := range grids {
		g.CellSize = units.Dp(Settings.ThumbSize)
		g.NeedsLayout()
		g.Defer(func() { g.scrollToIndex(max(g.Current, 0)) })
	}
}
