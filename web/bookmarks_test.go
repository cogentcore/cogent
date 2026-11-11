// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"path/filepath"
	"strings"
	"testing"

	"cogentcore.org/core/base/iox/tomlx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withBookmarks sets the bookmarks of the settings to the given ones for
// the test.
func withBookmarks(t *testing.T, bms []*Bookmark) {
	prev := Settings.Bookmarks
	Settings.Bookmarks = bms
	t.Cleanup(func() { Settings.Bookmarks = prev })
}

// testBookmarks returns bookmarks with a group, in which there is another
// group, and a bookmark at the top level.
func testBookmarks() []*Bookmark {
	return []*Bookmark{
		{Name: "News", Children: []*Bookmark{
			{Name: "One", URL: "https://example.com/one"},
			{Name: "Tech", Children: []*Bookmark{{Name: "Two", URL: "https://example.com/two"}}},
		}},
		{Name: "Three", URL: "https://example.com/three"},
	}
}

// barButtons returns the texts of the buttons of the bookmarks bar of the
// window of the given body (in its top bar).
func barButtons(b *core.Body) (texts []string, bar *core.Toolbar) {
	b.Scene.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
		if tb, ok := cw.(*core.Toolbar); ok && tb.Name == "bookmarks" {
			bar = tb
			for _, k := range tb.Children {
				if bt, ok := k.(*core.Button); ok && strings.HasPrefix(bt.Name, "bookmark-") {
					texts = append(texts, bt.Text)
				}
			}
			return tree.Break
		}
		return tree.Continue
	})
	return
}

// TestBookmarksBar tests that the top-level bookmarks are the buttons of
// the bookmarks bar, below the toolbar, that one opens its page, and that
// the bar follows the changes of the bookmarks.
func TestBookmarksBar(t *testing.T) {
	withBookmarks(t, testBookmarks())
	b := core.NewBody()
	br := InitBody(b)
	var opened string
	br.PageFunc = func(pg *Page) {}
	pg := br.NewTab("")
	b.RenderImage(func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		texts, bar := barButtons(b)
		require.NotNil(t, bar)
		assert.Equal(t, []string{"News", "Three"}, texts)
		// (the page of the current tab opens it)
		pg.OnOpen = func(url string) { opened = url }
		bar.Child(1).(*core.Button).Send(events.Click)
	}, func() {
		pg.wait()
		assert.Contains(t, opened, "example.com/three")
		AddBookmark(nil, "Four", "https://example.com/four")
	}, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		texts, _ := barButtons(b)
		assert.Equal(t, []string{"News", "Three", "Four"}, texts, "updated for a new bookmark")
	})
}

// TestBookmarkGroups tests the groups that a page can be bookmarked in,
// with their paths, and adding a bookmark to one.
func TestBookmarkGroups(t *testing.T) {
	withBookmarks(t, testBookmarks())
	groups, paths := bookmarkGroups(Settings.Bookmarks, "")
	assert.Equal(t, []string{"News", "News / Tech"}, paths)
	AddBookmark(groups[1], "Five", "https://example.com/five")
	assert.Equal(t, "Five", Settings.Bookmarks[0].Children[1].Children[1].Name)
	assert.False(t, Settings.Bookmarks[1].IsGroup())
	assert.True(t, (&Bookmark{Name: "empty group"}).IsGroup())
}

// TestBookmarksSaved tests that the bookmarks, with their groups, are saved
// in the settings file and loaded from it.
func TestBookmarksSaved(t *testing.T) {
	withBookmarks(t, testBookmarks())
	prev := Settings.File
	Settings.File = filepath.Join(t.TempDir(), "settings.toml")
	defer func() { Settings.File = prev }()
	require.NoError(t, core.SaveSettings(Settings))
	Settings.Bookmarks = nil
	require.NoError(t, tomlx.Open(Settings, Settings.File))
	require.Len(t, Settings.Bookmarks, 2)
	assert.Equal(t, "Two", Settings.Bookmarks[0].Children[1].Children[0].Name)
	assert.Equal(t, "https://example.com/three", Settings.Bookmarks[1].URL)
}

// TestEditBookmarks tests that the editor of the bookmarks shows them in a
// tree table.
func TestEditBookmarks(t *testing.T) {
	withBookmarks(t, testBookmarks())
	b := core.NewBody()
	b.RenderImage(func() {
		b.AsyncLock()
		EditBookmarks(b)
		b.AsyncUnlock()
	}, func() {
		var tt *core.TreeTable
		for _, w := range core.AllRenderWindows {
			if sc := w.MainScene(); sc != nil && sc.Body != nil && sc.Body.Title == "Bookmarks" {
				sc.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
					if t, ok := cw.(*core.TreeTable); ok {
						tt = t
					}
					return tt == nil
				})
			}
		}
		require.NotNil(t, tt)
		assert.Same(t, &Settings.Bookmarks, tt.Items)
	})
}

// TestBookmarksBarMove tests that a bookmark of the bookmarks bar (here a
// group) moves to another place in the bookmarks when it is dragged there.
func TestBookmarksBarMove(t *testing.T) {
	withBookmarks(t, testBookmarks())
	b := core.NewBody()
	br := InitBody(b)
	br.NewTab("")
	b.RenderImage(func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		_, bar := barButtons(b)
		require.NotNil(t, bar)
		require.NotNil(t, bar.MoveFunc)
		bar.MoveFunc(0, 1) // (as by dragging News onto Three)
	}, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		assert.Equal(t, "Three", Settings.Bookmarks[0].Name)
		assert.Equal(t, "News", Settings.Bookmarks[1].Name)
		texts, _ := barButtons(b)
		assert.Equal(t, []string{"Three", "News"}, texts)
	})
}
