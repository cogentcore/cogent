// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"image"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// toolbarWidget returns the widget of the toolbar of the page of the given
// body with the given name.
func toolbarWidget[T tree.Node](b *core.Body, name string) T {
	var res T
	for n := range b.Scene.All() {
		if w, ok := n.(T); ok && n.AsTree().Name == name {
			res = w
		}
	}
	return res
}

// TestToolbarActions tests the actions of the toolbar of a page: the
// address field, back, forward, reload, the zoom, and the menu of the
// scripts (the console, saving the page, clearing the cookies, and turning
// off JavaScript), and the cookies of the site.
func TestToolbarActions(t *testing.T) {
	// the files that the actions write are in temporary directories
	home := t.TempDir()
	t.Setenv("HOME", home)
	require.NoError(t, os.MkdirAll(filepath.Join(home, "Downloads"), 0o755))
	cookieDir := t.TempDir()
	savedFile := cookiesFile
	cookiesFile = func() string { return filepath.Join(cookieDir, "cookies.json") }
	t.Cleanup(func() { cookiesFile = savedFile })
	t.Cleanup(core.TrackWindows())

	var b *core.Body
	b, pg := newScriptPageWith(t, map[string]string{
		"/":    `<html><body><p>one</p><script>console.error("bad")</script></body></html>`,
		"/two": `<html><body><p>two</p></body></html>`,
	}, func(pg *Page) {
		b := pg.Parent.(*core.Body)
		b.AddTopBar(func(bar *core.Frame) {
			tb := core.NewToolbar(bar)
			tb.Maker(pg.MakeToolbar)
			pg.OnOpen = func(url string) { tb.Update() }
		})
	})
	button := func(name string) *core.Button {
		t.Helper()
		bt := toolbarWidget[*core.Button](b, name)
		require.NotNil(t, bt, name)
		return bt
	}
	var menu *core.Scene
	core.RenderImage(b, func() { settle(pg) }, core.Locked(b, func() {
		tb := button("scripts").Parent.(*core.Toolbar)
		tb.Update()
		assert.Contains(t, button("scripts").Tooltip, "1 script errors")
		button("zoom-in").Send(events.Click)
		assert.Greater(t, pg.zoom, float32(1))
		tb.Update()
		assert.NotEqual(t, "100%", button("zoom").Text)
		button("zoom").Send(events.Click)
		assert.Equal(t, float32(1), pg.zoomFactor())
		button("zoom-out").Send(events.Click)
		assert.Less(t, pg.zoom, float32(1))
		button("zoom").Send(events.Click)

		addr := toolbarWidget[*core.TextField](b, "url")
		require.NotNil(t, addr)
		addr.SetText("https://example.com/two")
		addr.SendChange()
	}), func() { settle(pg) }, core.Locked(b, func() {
		assert.Equal(t, "https://example.com/two", pg.currentURL())
		button("back").Send(events.Click)
	}), func() { settle(pg) }, core.Locked(b, func() {
		assert.Equal(t, "https://example.com/", pg.currentURL())
		button("forward").Send(events.Click)
	}), func() { settle(pg) }, core.Locked(b, func() {
		assert.Equal(t, "https://example.com/two", pg.currentURL())
		button("reload").Send(events.Click)
	}), func() { settle(pg) }, core.Locked(b, func() {
		menu = core.NewScene()
		button("scripts").Menu(menu, image.Point{})
		core.ButtonByText(menu, "Script console").Send(events.Click) // a window, closed at the end

		// (only in the temporary directories)
		dir, err := os.UserHomeDir()
		require.NoError(t, err)
		require.Equal(t, home, dir)
		require.True(t, strings.HasPrefix(cookiesFile(), cookieDir))
		core.ButtonByText(menu, "Save page").Send(events.Click)
		core.ButtonByText(menu, "Clear cookies").Send(events.Click)
		button("cookies").Send(events.Click) // a window, closed at the end
		core.ButtonByText(menu, "Turn off JavaScript").Send(events.Click)
	}), func() { settle(pg) }, core.Locked(b, func() {
		assert.False(t, pg.Scripts)
	}))
	assert.Eventually(t, func() bool {
		files, _ := filepath.Glob(filepath.Join(home, "Downloads", "example.com-*.html"))
		return len(files) == 1
	}, 5*time.Second, 10*time.Millisecond, "the saved page")
}

// TestScriptsVerbose tests the verbose console of a page with scripts:
// the requests of the page and the times of reading it, the messages of
// its frames, a page that redirects with a meta refresh, and a fragment.
func TestScriptsVerbose(t *testing.T) {
	saved := Settings.VerboseConsole
	Settings.VerboseConsole = true
	t.Cleanup(func() { Settings.VerboseConsole = saved })
	// (a frame gets its page from the network: a local server)
	frame := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `<html><body><script>console.log("in the frame")</script></body></html>`)
	}))
	defer frame.Close()
	b, pg := newScriptPage(t, map[string]string{
		"/":      `<html><head><meta http-equiv="refresh" content="0; url=/dest#end"></head><body><script>1</script></body></html>`,
		"/dest":  `<html><head><link rel="stylesheet" href="/s.css"></head><body><p>dest</p><iframe src="` + frame.URL + `/frame" width="100" height="50"></iframe><div style="height:2000px"></div><p id="end">end</p><script>console.log("top")</script></body></html>`,
		"/s.css": `p { color: blue }`,
	})
	console := func() string {
		b.AsyncLock()
		defer b.AsyncUnlock()
		return strings.Join(pg.ScriptConsole(), "\n")
	}
	core.RenderImage(b, func() { settle(pg) }, func() {
		require.True(t, eventually(func() bool {
			b.AsyncLock()
			defer b.AsyncUnlock()
			return pg.currentURL() == "https://example.com/dest#end"
		}), "redirected")
		settle(pg)
		require.True(t, eventually(func() bool {
			all := console()
			return strings.Contains(all, "in the frame") && strings.Contains(all, "got https://example.com/s.css")
		}), "the console")
	})
}
