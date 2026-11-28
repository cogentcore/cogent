// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package photos

import (
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestKeys tests the keys for moving among the panels, Control+X o and
// Control+X p, and Control+T and Control+O for tagging, as in Cogent Mail.
func TestKeys(t *testing.T) {
	lib := smallLibrary(t, 3)
	require.NoError(t, lib.AddTag("Home", lib.Photos(nil)[0]))
	b := core.NewBody()
	a := NewApp(b)
	a.Styler(func(s *styles.Style) {
		s.Min.Set(units.Dp(1200), units.Dp(600))
	})
	a.shown = allKey
	old := keymap.ActiveMapName            // after the settings are applied
	keymap.SetActiveMapName("MacStandard") // with Control+X for key sequences
	t.Cleanup(func() { keymap.SetActiveMapName(old) })
	var ctrl key.Modifiers
	ctrl.SetFlag(true, key.Control)
	core.RenderImage(b, func() {
		b.AsyncLock()
		a.lib = lib
		a.grid.lib = lib
		a.refresh()
		a.grid.SetFocusQuiet()
		b.AsyncUnlock()
	}, func() {
		ev := a.grid.SystemEvents()
		seq := func(c rune, code key.Codes) {
			ev.KeyChord('x', key.CodeX, ctrl)
			ev.KeyChord(c, code, 0)
		}
		focused := func(w core.Widget) func() bool {
			return func() bool { return w.AsWidget().ContainsFocus() }
		}
		seq('o', key.CodeO) // from the photos to the photo shown
		waitFor(t, b, focused(a.viewer))
		b.AsyncLock()
		assert.True(t, a.viewer.ContainsFocus(), "the photo shown")
		b.AsyncUnlock()
		seq('o', key.CodeO) // around to the tags
		waitFor(t, b, focused(a.tags))
		b.AsyncLock()
		assert.True(t, a.tags.ContainsFocus(), "the tags")
		b.AsyncUnlock()
		seq('p', key.CodeP) // back to the photo shown
		waitFor(t, b, focused(a.viewer))
		b.AsyncLock()
		assert.True(t, a.viewer.ContainsFocus(), "back to the photo shown")
		b.AsyncUnlock()
		seq('p', key.CodeP)
		waitFor(t, b, focused(a.grid))
		b.AsyncLock()
		assert.True(t, a.grid.ContainsFocus(), "back to the photos")
		b.AsyncUnlock()

		for _, c := range []struct {
			r    rune
			code key.Codes
		}{{'t', key.CodeT}, {'o', key.CodeO}} {
			ev.KeyChord(c.r, c.code, ctrl)
			dialog := func() bool { return b.Scene.RenderWindow().MainScene() != b.Scene }
			waitFor(t, b, dialog)
			b.AsyncLock()
			sc := b.Scene.RenderWindow().MainScene()
			require.NotEqual(t, b.Scene, sc, "the Tag dialog, for Control+%c", c.r)
			assert.Equal(t, "Tag", sc.Body.Title)
			sc.Close()
			b.AsyncUnlock()
			waitFor(t, b, func() bool { return !dialog() })
		}
	})
}
