// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tempSettings makes the settings of the test saved in a temporary file.
func tempSettings(t *testing.T) {
	saved := Settings.File
	Settings.File = filepath.Join(t.TempDir(), "settings.toml")
	t.Cleanup(func() { Settings.File = saved })
}

// TestEditBookmarksWindow tests the editor of the bookmarks, in its window:
// adding a group and a bookmark, which are saved, and opening it again,
// which raises its window.
func TestEditBookmarksWindow(t *testing.T) {
	withBookmarks(t, testBookmarks())
	tempSettings(t)
	t.Cleanup(core.TrackWindows())
	b := core.NewBody()
	core.NewText(b).SetText("app")
	var tt *core.TreeTable
	core.RenderImage(b, core.Locked(b, func() { EditBookmarks(b) }), func() {
		require.Eventually(t, func() bool {
			bookmarksEditor.Lock()
			defer bookmarksEditor.Unlock()
			tt = bookmarksEditor.table
			return tt != nil && tt.Scene != nil && tt.Scene.RenderWindow() != nil
		}, 5*time.Second, 10*time.Millisecond)
	}, func() {
		core.Locked(tt, func() {
			core.ButtonByText(tt.Scene, "Add group").Send(events.Click)
			core.ButtonByText(tt.Scene, "Add bookmark").Send(events.Click)
			n := len(Settings.Bookmarks)
			require.GreaterOrEqual(t, n, 4)
			assert.Equal(t, "New group", Settings.Bookmarks[n-2].Name)
			assert.Equal(t, "https://", Settings.Bookmarks[n-1].URL)
		})()
	}, core.Locked(b, func() {
		EditBookmarks(b) // raises the window
	}))
	assert.FileExists(t, Settings.File, "saved")
}

// TestConsolePanelButtons tests the buttons of the console panel: the
// verbose switch, which is saved, copying, clearing, and closing.
func TestConsolePanelButtons(t *testing.T) {
	tempSettings(t)
	saved := Settings.VerboseConsole
	t.Cleanup(func() { Settings.VerboseConsole = saved })
	b := core.NewBody()
	b.Styler(func(s *styles.Style) { s.Min.X.Dp(800); s.Min.Y.Dp(600) })
	br := NewBrowser(b)
	br.PageFunc = func(pg *Page) {
		pg.Context.GetURL = func(u string) (*http.Response, error) {
			pu, _ := url.Parse(u)
			r := strings.NewReader(`<html><body><p>page</p><script>console.log("hello")</script></body></html>`)
			return &http.Response{StatusCode: 200, Status: "200 OK", Request: &http.Request{URL: pu}, Body: io.NopCloser(r)}, nil
		}
	}
	br.NewTab("https://example.com/one")
	button := func(icon icons.Icon) *core.Button {
		for w := range br.Console.AllWidgets() {
			if bt, ok := w.(*core.Button); ok && bt.Icon == icon {
				return bt
			}
		}
		return nil
	}
	core.RenderImage(b, func() {
		for _, pg := range br.Pages() {
			settle(pg)
		}
	}, core.Locked(b, func() {
		br.ShowConsole(true)
	}), core.Locked(b, func() {
		assert.Contains(t, br.Console.allText(), "hello")
		var sw *core.Switch
		for w := range br.Console.AllWidgets() {
			if x, ok := w.(*core.Switch); ok {
				sw = x
			}
		}
		require.NotNil(t, sw)
		sw.SetChecked(!Settings.VerboseConsole)
		sw.SendChange()
		assert.Equal(t, sw.IsChecked(), Settings.VerboseConsole)
		button(icons.ContentCopy).Send(events.Click)
		button(icons.Delete).Send(events.Click)
		assert.NotContains(t, br.Console.allText(), "hello", "cleared")
		button(icons.Close).Send(events.Click)
		assert.False(t, br.ConsoleShown())
	}))
	assert.FileExists(t, Settings.File, "saved")
}

// TestPushStateShown tests that the URL that a script pushes once the page
// is shown is that of the page in its history, and its address.
func TestPushStateShown(t *testing.T) {
	b, pg := newScriptPage(t, map[string]string{"/": `<html><body><p>page</p><script>1</script></body></html>`})
	core.RenderImage(b, func() { settle(pg) }, func() {
		pg.scripts.engine.Eval(`history.pushState({}, "", "/pushed?x=1")`)
		require.Eventually(t, func() bool {
			b.AsyncLock()
			defer b.AsyncUnlock()
			return pg.History[pg.Current] == "https://example.com/pushed?x=1"
		}, 5*time.Second, 10*time.Millisecond)
		b.AsyncLock()
		assert.Equal(t, "https://example.com/pushed?x=1", pg.Context.PageURL)
		b.AsyncUnlock()
	})
}
