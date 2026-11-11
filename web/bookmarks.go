// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"fmt"
	"image"
	"slices"
	"strings"
	"sync"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
)

// Bookmark is a bookmark of a page, or a group of bookmarks, in the
// bookmarks of the [Settings], which are edited in a [core.TreeTable]
// (see [EditBookmarks]). The top-level bookmarks are the buttons of the
// bookmarks bar of a browser: those of groups have menus of their
// bookmarks.
type Bookmark struct {

	// Name is the name of the bookmark, which is shown for it.
	Name string

	// URL is the URL of the page of the bookmark, which is empty for a
	// group of bookmarks.
	URL string `width:"40"`

	// Children are the bookmarks in the group.
	Children []*Bookmark `display:"-"`

	// Closed is whether the group is closed in the editor of bookmarks.
	Closed bool `display:"-" treetable:"closed"`
}

// IsGroup returns whether the bookmark is a group of bookmarks: it has
// children, or no URL.
func (bm *Bookmark) IsGroup() bool {
	return len(bm.Children) > 0 || bm.URL == ""
}

// label returns the text shown for the bookmark: its name, or else its URL.
func (bm *Bookmark) label() string {
	if bm.Name != "" {
		return bm.Name
	}
	if bm.URL != "" {
		return pageName(bm.URL)
	}
	return "Bookmarks"
}

// bookmarkBars are the bookmarks bars of the browsers, which are updated
// when the bookmarks change (see [bookmarksChanged]).
var bookmarkBars struct {
	sync.Mutex
	bars []*core.Toolbar
}

// bookmarksEditor is the tree table of the editor of the bookmarks, while
// its window is open (see [EditBookmarks]), which is updated when the
// bookmarks change elsewhere (see [bookmarksChanged]).
var bookmarksEditor struct {
	sync.Mutex
	table *core.TreeTable
}

// bookmarksChanged saves the settings, with the bookmarks, and updates the
// bookmarks bars of all of the browsers, and the editor of the bookmarks,
// unless the change was made in it, each in its own window.
func bookmarksChanged() {
	changed(nil)
}

// changed is [bookmarksChanged] for a change made in the given editor of
// the bookmarks (or elsewhere, if it is nil), which is not updated.
func changed(editor *core.TreeTable) {
	if Settings.Filename() != "" { // (not in tests)
		errors.Log(core.SaveSettings(Settings))
	}
	bookmarkBars.Lock()
	bars := slices.DeleteFunc(bookmarkBars.bars, func(tb *core.Toolbar) bool { return tb.This == nil })
	bookmarkBars.bars = bars
	bars = slices.Clone(bars)
	bookmarkBars.Unlock()
	var widgets []core.Widget
	for _, tb := range bars {
		widgets = append(widgets, tb)
	}
	bookmarksEditor.Lock()
	if tt := bookmarksEditor.table; tt != nil && tt.This != nil && tt != editor {
		widgets = append(widgets, tt)
	}
	bookmarksEditor.Unlock()
	for _, w := range widgets {
		// (not with the lock of this window, which another window may be
		// waiting on)
		go func() {
			wb := w.AsWidget()
			wb.AsyncLock()
			defer wb.AsyncUnlock()
			if wb.This != nil {
				wb.Update()
			}
		}()
	}
}

// addBookmarksBar adds the bookmarks bar of the given browser to the given
// parent, below its toolbar: a button for each top-level bookmark, which
// opens it, or shows the menu of the bookmarks of a group, and which can be
// dragged to another place in the bar. It is hidden when there are no
// bookmarks.
func addBookmarksBar(parent core.Widget, br *Browser) *core.Toolbar {
	tb := core.NewToolbar(parent)
	tb.SetName("bookmarks")
	tb.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 0)
		if len(Settings.Bookmarks) == 0 {
			s.Display = styles.DisplayNone
		}
	})
	// the bookmarks (and groups) are reordered by dragging them
	tb.MoveFunc = func(from, to int) {
		bms := Settings.Bookmarks
		if from < 0 || from >= len(bms) || to < 0 || to >= len(bms) {
			return
		}
		bm := bms[from]
		bms = slices.Delete(bms, from, from+1)
		Settings.Bookmarks = slices.Insert(bms, to, bm)
		bookmarksChanged()
	}
	tb.Maker(func(p *tree.Plan) {
		for _, bm := range Settings.Bookmarks {
			// (a button for each bookmark, whose changes update it)
			tree.AddAt(p, fmt.Sprintf("bookmark-%p", bm), func(w *core.Button) {
				w.SetType(core.ButtonAction)
				w.OnClick(func(e events.Event) {
					if !bm.IsGroup() {
						br.openBookmark(bm.URL, e)
					}
				})
				w.Updater(func() { configBookmarkButton(w, bm, br) })
			})
		}
	})
	bookmarkBars.Lock()
	bookmarkBars.bars = append(bookmarkBars.bars, tb)
	bookmarkBars.Unlock()
	return tb
}

// configBookmarkButton configures the given button of the bookmarks bar for
// the given bookmark of the given browser: a group has the menu of its
// bookmarks.
func configBookmarkButton(w *core.Button, bm *Bookmark, br *Browser) {
	w.SetText(bm.label())
	if bm.IsGroup() {
		w.SetIcon(icons.Folder).SetTooltip(bm.label())
		w.SetMenu(func(m *core.Scene, _ image.Point) { bookmarksMenu(m, bm.Children, br) })
		return
	}
	w.SetIcon(icons.Bookmark).SetTooltip(bm.URL)
	w.Menu = nil
}

// bookmarksMenu adds the buttons of the given bookmarks to the given menu:
// a group has a submenu of its bookmarks.
func bookmarksMenu(m *core.Scene, bms []*Bookmark, br *Browser) {
	if len(bms) == 0 {
		core.NewText(m).SetText("(no bookmarks)")
		return
	}
	for _, bm := range bms {
		b := core.NewButton(m).SetText(bm.label())
		if bm.IsGroup() {
			b.SetIcon(icons.Folder)
			b.SetMenu(func(sm *core.Scene, _ image.Point) { bookmarksMenu(sm, bm.Children, br) })
			continue
		}
		b.SetIcon(icons.Bookmark).SetTooltip(bm.URL)
		b.OnClick(func(e events.Event) { br.openBookmark(bm.URL, e) })
	}
}

// openBookmark opens the page at the given URL of a bookmark in the current
// tab, or in a new one if there is none, or with the meta or control key.
func (br *Browser) openBookmark(url string, e events.Event) {
	if pg := br.CurrentPage(); pg != nil {
		pg.OpenURL(LocationURL(url), e)
		return
	}
	br.NewTab(LocationURL(url))
}

// AddBookmark adds a bookmark of the given name and URL to the given group
// of bookmarks, or at the top level (in the bookmarks bar) if it is nil,
// and saves it.
func AddBookmark(group *Bookmark, name, url string) *Bookmark {
	bm := &Bookmark{Name: name, URL: url}
	if group == nil {
		Settings.Bookmarks = append(Settings.Bookmarks, bm)
	} else {
		group.Children = append(group.Children, bm)
	}
	bookmarksChanged()
	return bm
}

// bookmarkGroups returns the groups of bookmarks in the given bookmarks, in
// order (each followed by those in it), with their paths of names.
func bookmarkGroups(bms []*Bookmark, path string) (groups []*Bookmark, paths []string) {
	for _, bm := range bms {
		if !bm.IsGroup() {
			continue
		}
		p := bm.label()
		if path != "" {
			p = path + " / " + p
		}
		groups = append(groups, bm)
		paths = append(paths, p)
		g, ps := bookmarkGroups(bm.Children, p)
		groups = append(groups, g...)
		paths = append(paths, ps...)
	}
	return
}

// bookmarkMenu adds the items of the menu of the bookmark button of the
// toolbar to the given menu, for the page of the current tab of the given
// browser: adding it to the bookmarks bar or to a group, and editing the
// bookmarks.
func (br *Browser) bookmarkMenu(m *core.Scene) {
	pg := br.CurrentPage()
	url, name := "", ""
	if pg != nil {
		url = pg.currentURL()
		name = strings.Join(strings.Fields(pg.Title), " ")
	}
	if name == "" {
		name = pageName(url)
	}
	add := func(text string, group *Bookmark) {
		b := core.NewButton(m).SetText(text).SetIcon(icons.BookmarkAdd)
		b.SetEnabled(url != "")
		b.OnClick(func(e events.Event) { AddBookmark(group, name, url) })
	}
	add("Add to bookmarks bar", nil)
	groups, paths := bookmarkGroups(Settings.Bookmarks, "")
	for i, g := range groups {
		add("Add to "+paths[i], g)
	}
	core.NewSeparator(m)
	core.NewButton(m).SetText("Edit bookmarks").SetIcon(icons.Edit).OnClick(func(e events.Event) {
		EditBookmarks(br)
	})
}

// EditBookmarks opens the editor of the bookmarks, a [core.TreeTable] of
// them in a new window, in which they are saved as they are edited.
func EditBookmarks(ctx core.Widget) {
	if core.RecycleMainWindow(&Settings.Bookmarks) {
		return
	}
	b := core.NewBody("Bookmarks")
	b.SetData(&Settings.Bookmarks)
	tt := core.NewTreeTable(b)
	tt.SetDragMoves(true)
	tt.SetItems(&Settings.Bookmarks)
	tt.OnChange(func(e events.Event) { changed(tt) })
	bookmarksEditor.Lock()
	bookmarksEditor.table = tt
	bookmarksEditor.Unlock()
	b.OnClose(func(e events.Event) {
		bookmarksEditor.Lock()
		if bookmarksEditor.table == tt {
			bookmarksEditor.table = nil
		}
		bookmarksEditor.Unlock()
	})
	b.AddTopBar(func(bar *core.Frame) {
		tb := core.NewToolbar(bar)
		tb.Maker(func(p *tree.Plan) {
			tree.Add(p, func(w *core.Button) {
				w.SetText("Add group").SetIcon(icons.CreateNewFolder).SetTooltip("Add a group of bookmarks at the end")
				w.OnClick(func(e events.Event) {
					Settings.Bookmarks = append(Settings.Bookmarks, &Bookmark{Name: "New group"})
					tt.Update()
					changed(tt)
				})
			})
			tree.Add(p, func(w *core.Button) {
				w.SetText("Add bookmark").SetIcon(icons.BookmarkAdd).SetTooltip("Add a bookmark at the end, for its name and URL to be entered")
				w.OnClick(func(e events.Event) {
					Settings.Bookmarks = append(Settings.Bookmarks, &Bookmark{Name: "New bookmark", URL: "https://"})
					tt.Update()
					changed(tt)
				})
			})
		})
	})
	b.RunWindow()
}
