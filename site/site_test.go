// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"testing"
	"time"

	"cogentcore.org/core/content"
	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSite tests the website: the grid of the apps on the home page, with
// a tile for each app and a Docs link for the apps with documentation, and
// the documentation of an app, mounted at docs/<app>, with its icon.
func TestSite(t *testing.T) {
	for _, page := range []string{"", "docs/lists"} {
		name := page
		if name == "" {
			name = "home"
		}
		t.Run(name, func(t *testing.T) {
			b := core.NewBody()
			b.Styler(func(s *styles.Style) {
				s.Min.Set(units.Dp(1280), units.Dp(800))
				s.Max = s.Min // (a stable size)
			})
			ct := NewSite(b)
			core.AssertRender(t, b, name, func() {
				ct.AsyncLock()
				ct.Open(page)
				ct.AsyncUnlock()
			}, func() {
				time.Sleep(200 * time.Millisecond) // to give the images time to load
				ct.AsyncLock()
				defer ct.AsyncUnlock()
				checkSite(t, ct, page)
			})
		})
	}
}

// checkSite checks the rendered page with the given URL of the website.
func checkSite(t *testing.T, ct *content.Content, page string) {
	tiles := map[string]int{} // buttons per app tile
	docs := map[string]bool{} // apps with a Docs link
	svgs := 0
	ct.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
		if bt, ok := cw.(*core.Button); ok {
			if tile := bt.Parent; tile != nil {
				for _, a := range Apps {
					if tile.AsTree().Name == a.Path {
						tiles[a.Path]++
						if bt.Text == "Docs" {
							docs[a.Path] = true
						}
					}
				}
			}
		}
		if sv, ok := cw.(*core.SVG); ok && sv.SVG != nil && sv.SVG.Root.NumChildren() > 0 {
			svgs++
		}
		return true
	})
	if page != "" {
		assert.Empty(t, tiles, "no grid on the docs page")
		assert.Equal(t, 1, svgs, "the icon of the app, relative to its docs page")
		return
	}
	require.Len(t, tiles, len(Apps), "a tile for each app")
	for _, a := range Apps {
		assert.Equal(t, a.Docs != nil, docs[a.Path], "the Docs link of %s", a.Name)
		assert.NotEmpty(t, a.Icon(), "the icon of %s", a.Name)
	}
}

// TestSource tests the source of the website: the home page and icon of
// the documentation of each app that has any, at docs/<app>.
func TestSource(t *testing.T) {
	src := Source()
	for _, a := range Apps {
		if a.Docs == nil {
			continue
		}
		_, err := src.Open(a.DocsURL() + "/home.md")
		assert.NoError(t, err, a.Name)
		_, err = src.Open(a.DocsURL() + "/icon.svg")
		assert.NoError(t, err, a.Name)
	}
}
