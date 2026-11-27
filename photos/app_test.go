// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package photos

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"github.com/stretchr/testify/require"
)

// TestMain uses temporary settings.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "photos-test")
	if err != nil {
		panic(err)
	}
	Settings.File = filepath.Join(dir, "settings.toml")
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// TestAppShot renders the app with the library in PHOTOS_SHOT, saving
// the image in PHOTOS_SHOT_IMAGE, for looking at it.
func TestAppShot(t *testing.T) {
	dir := os.Getenv("PHOTOS_SHOT")
	if dir == "" {
		t.Skip("PHOTOS_SHOT not set")
	}
	b := core.NewBody("Cogent Photos")
	a := NewApp(b)
	a.Styler(func(s *styles.Style) {
		s.Min.Set(units.Dp(1400), units.Dp(850))
	})
	b.AddTopBar(func(bar *core.Frame) {
		core.NewToolbar(bar).Maker(a.MakeToolbar)
	})
	if tag := os.Getenv("PHOTOS_SHOT_TAG"); tag != "" {
		a.shown = tag
	}
	img := core.RenderImage(b, func() {
		lib, err := OpenLibrary(dir)
		require.NoError(t, err)
		lib.ThumbDir = filepath.Join(t.TempDir(), "thumbs") // not the real ones
		if lib.Len() == 0 {
			require.NoError(t, lib.Scan(nil, nil))
		}
		b.AsyncLock()
		a.lib = lib
		a.grid.lib = lib
		a.refresh()
		b.AsyncUnlock()
		time.Sleep(3 * time.Second) // the thumbnails
		b.AsyncLock()
		a.grid.NeedsRender()
		b.AsyncUnlock()
	})
	require.NotNil(t, img)
	require.NoError(t, imagex.Save(img, os.Getenv("PHOTOS_SHOT_IMAGE")))
}
