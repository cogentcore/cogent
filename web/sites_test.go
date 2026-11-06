// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
)

// tempSites makes the settings of the sites be saved in a temporary
// file for the test, and loaded from it again.
func tempSites(t *testing.T) string {
	file := filepath.Join(t.TempDir(), "web", "sites.json")
	saved := sitesFile
	sitesFile = func() string { return file }
	reset := func() {
		sites.Lock()
		sites.loaded = false
		sites.settings = nil
		sites.Unlock()
	}
	reset()
	t.Cleanup(func() { sitesFile = saved; reset() })
	return file
}

// TestSiteSettings tests that the settings of sites are saved by host,
// loaded again, and not saved when they are the defaults.
func TestSiteSettings(t *testing.T) {
	file := tempSites(t)
	SetSite("https://www.Example.com/a/page?x=1", func(ss *SiteSettings) { ss.Zoom = 1.5 })
	SetSite("https://other.org/", func(ss *SiteSettings) { ss.Zoom = 2 })
	SetSite("https://other.org/", func(ss *SiteSettings) { ss.Zoom = 0 })
	SetSite("file:///tmp/x.html", func(ss *SiteSettings) { ss.Zoom = 3 })
	b, err := os.ReadFile(file)
	assert.NoError(t, err)
	assert.Contains(t, string(b), "www.example.com")
	assert.NotContains(t, string(b), "other.org", "default settings are not saved")
	sites.Lock()
	sites.loaded = false // as when the app starts again
	sites.Unlock()
	assert.Equal(t, float32(1.5), Site("https://www.example.com/other").Zoom)
	assert.Equal(t, float32(0), Site("https://other.org/").Zoom)
}

// TestPageZoom tests that zooming a page multiplies its sizes, with and
// without scripts, and that the zoom is kept for the pages of its site.
func TestPageZoom(t *testing.T) {
	for _, scripts := range []bool{true, false} {
		t.Run(fmt.Sprint("scripts=", scripts), func(t *testing.T) {
			tempSites(t)
			b, pg := newScriptPage(t, map[string]string{
				"/":      `<html><body><div id="box" style="width: 100px; height: 20px">box</div></body></html>`,
				"/other": `<html><body><div id="box" style="width: 100px; height: 20px">other</div></body></html>`,
			})
			pg.Scripts = scripts
			pg.Reload()
			width := func() float32 {
				b.AsyncLock()
				defer b.AsyncUnlock()
				w := float32(0)
				pg.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
					if cwb.Property("id") == "box" {
						w = cwb.Geom.Size.Actual.Content.X
					}
					return tree.Continue
				})
				return w
			}
			var w1 float32
			b.RenderImage(func() { settle(pg) }, func() {
				w1 = width()
				assert.Greater(t, w1, float32(0))
				b.AsyncLock()
				pg.SetZoom(2)
				b.AsyncUnlock()
				settle(pg)
			}, func() {
				assert.True(t, eventually(func() bool { return width() > 1.9*w1 }), "zoomed: %g from %g", width(), w1)
				assert.Equal(t, float32(2), Site("https://example.com/").Zoom)
				b.AsyncLock()
				pg.OpenURL("https://example.com/other", nil)
				b.AsyncUnlock()
				settle(pg)
			}, func() {
				assert.True(t, eventually(func() bool { return width() > 1.9*w1 }), "another page of the site is zoomed: %g", width())
				b.AsyncLock()
				pg.StepZoom(0)
				b.AsyncUnlock()
				settle(pg)
			}, func() {
				assert.True(t, eventually(func() bool { return width() < 1.1*w1 }), "back to 100%%: %g", width())
				assert.Equal(t, float32(0), Site("https://example.com/").Zoom)
			})
		})
	}
}
