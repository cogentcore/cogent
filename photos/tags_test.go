// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package photos

import (
	"image"
	"testing"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCleanTag(t *testing.T) {
	for tag, want := range map[string]string{
		"2017_AlpineVRBOposs":   "2017/AlpineVRBOposs",
		"2026 Groningen":        "2026/Groningen",
		" 2026 / Groningen ":    "2026/Groningen",
		"Donations":             "Donations",
		"People//Family/ Alice": "People/Family/Alice",
		"Trips/2026 Groningen":  "Trips/2026 Groningen", // in a group already
		"2026":                  "2026",
		"20261 Thing":           "20261 Thing",
		"":                      "",
	} {
		assert.Equal(t, want, cleanTag(tag), tag)
	}
	g, n := splitTag("People/Family/Alice")
	assert.Equal(t, "People/Family", g)
	assert.Equal(t, "Alice", n)
	g, n = splitTag("Home")
	assert.Equal(t, "", g)
	assert.Equal(t, "Home", n)
	assert.Equal(t, []string{"People", "People/Family"}, tagGroups("People/Family/Alice"))
	assert.Equal(t, "Alice", joinTag("", "Alice"))
	assert.Equal(t, "2026/Groningen", joinTag("", "2026 Groningen"))
	assert.Equal(t, "People/Alice", joinTag("People", "Alice"))
	assert.True(t, inGroup("People/Family/Alice", "People"))
	assert.False(t, inGroup("Peoples/Alice", "People"))
}

// TestGroups tests making, renaming, moving, and deleting groups, and
// moving tags into them.
func TestGroups(t *testing.T) {
	lib := smallLibrary(t, 3)
	phs := lib.Photos(nil)
	require.NoError(t, lib.AddTag("2026 Groningen", phs[0]))
	require.NoError(t, lib.AddTag("Alice", phs[0], phs[1]))
	require.NoError(t, lib.AddTag("Family/Bob", phs[1]))
	assert.Equal(t, []string{"2026/Groningen", "Alice"}, phs[0].Tags)

	require.NoError(t, lib.AddGroup("People"))
	assert.Equal(t, []string{"2026", "Family", "People"}, lib.Groups(), "an empty group made")

	nt, err := lib.MoveTag("Alice", "People")
	require.NoError(t, err)
	assert.Equal(t, "People/Alice", nt)
	assert.Equal(t, 2, lib.Tags()["People/Alice"])

	require.NoError(t, lib.RenameGroup("Family", "People/Family"), "moved into People")
	assert.Equal(t, []string{"People/Alice", "People/Family/Bob"}, phs[1].Tags)
	assert.Equal(t, []string{"2026", "People", "People/Family"}, lib.Groups())
	assert.Error(t, lib.RenameGroup("People", "People/Family/People"), "into itself")

	require.NoError(t, lib.AddGroup("People/Friends"))
	require.NoError(t, lib.RenameGroup("People", "Persons"))
	assert.Equal(t, []string{"2026", "Persons", "Persons/Family", "Persons/Friends"}, lib.Groups())

	require.NoError(t, lib.DeleteGroup("Persons"))
	assert.Equal(t, []string{"Alice", "Family/Bob"}, phs[1].Tags, "moved up")
	assert.Equal(t, []string{"2026", "Family", "Friends"}, lib.Groups(), "the made group moved up too")
}

// TestMigrateTags tests changing tags of years with an underscore into
// tags in the groups of the years, once.
func TestMigrateTags(t *testing.T) {
	lib := smallLibrary(t, 3)
	phs := lib.Photos(nil)
	require.NoError(t, lib.store.setMeta(tagFormatKey, "")) // as before
	phs[0].Tags = []string{"2017_Kyoto", "Donations", "2017 Not"}
	require.NoError(t, lib.Save(phs[0]))
	require.NoError(t, lib.migrateTags())
	assert.Equal(t, []string{"2017/Kyoto", "Donations", "2017/Not"}, phs[0].Tags)
	phs[0].Tags = []string{"2018_Later"}
	require.NoError(t, lib.migrateTags())
	assert.Equal(t, []string{"2018_Later"}, phs[0].Tags, "only once")

	old := *Settings
	t.Cleanup(func() { *Settings = old })
	Settings.ShownTag = "tag:2017_Kyoto"
	Settings.Views = map[string]TagView{"year:2017": {Current: "x"}, "tag:Donations": {}}
	Settings.ClosedYears = map[string]bool{"2017": false}
	Settings.ClosedGroups = nil
	Settings.migrateTags()
	assert.Equal(t, "tag:2017/Kyoto", Settings.ShownTag)
	assert.Equal(t, map[string]TagView{"group:2017": {Current: "x"}, "tag:Donations": {}}, Settings.Views)
	assert.Equal(t, map[string]bool{"2017": false}, Settings.ClosedGroups)
	assert.Nil(t, Settings.ClosedYears)
}

// TestTagTree tests the tree of tags, with groups in groups, and dragging
// a tag in it onto a group, which moves the tag into it.
func TestTagTree(t *testing.T) {
	lib := smallLibrary(t, 6)
	phs := lib.Photos(nil)
	require.NoError(t, lib.AddTag("2026 Groningen", phs[0]))
	require.NoError(t, lib.AddTag("2017_Kyoto", phs[1]))
	require.NoError(t, lib.AddTag("Alice", phs[2], phs[3]))
	require.NoError(t, lib.AddTag("People/Family/Bob", phs[4]))
	require.NoError(t, lib.AddGroup("Home"))
	b := core.NewBody()
	a := NewApp(b)
	a.Styler(func(s *styles.Style) {
		s.Min.Set(units.Dp(1200), units.Dp(600))
	})
	a.shown = allKey
	old := Settings.ClosedGroups
	Settings.ClosedGroups = map[string]bool{"People": false}
	defer func() { Settings.ClosedGroups = old }()
	names := func() []string {
		var res []string
		for _, r := range *a.tags.Slice.(*[]*tagRow) {
			res = append(res, r.Key)
		}
		return res
	}
	core.RenderImage(b, func() {
		b.AsyncLock()
		a.lib = lib
		a.grid.lib = lib
		a.refresh()
		b.AsyncUnlock()
	}, func() {
		b.AsyncLock()
		assert.Equal(t, []string{"group:2026", "group:2017", "group:Home", "group:People",
			"group:People/Family", "tag:Alice"}, names(), "years first, open People, closed Family")
		var from, to image.Point
		for i, r := range *a.tags.Slice.(*[]*tagRow) {
			nw, _ := a.tags.RowWidgetNs()
			cell := a.tags.ListGrid.Child(i * nw).(core.Widget).AsWidget()
			switch r.Key {
			case "tag:Alice":
				from = cell.Geom.ContentBBox.Min.Add(image.Pt(30, 5))
			case "group:People":
				to = cell.Geom.ContentBBox.Min.Add(image.Pt(30, 5))
			}
		}
		require.NotEqual(t, image.Point{}, from)
		require.NotEqual(t, image.Point{}, to)
		b.AsyncUnlock()
		ev := a.grid.SystemEvents()
		ev.MouseButton(events.MouseDown, events.Left, from, 0)
		time.Sleep(200 * time.Millisecond)
		for i := 1; i <= 10; i++ {
			ev.MouseMove(from.Add(to.Sub(from).Mul(i).Div(10)))
			time.Sleep(30 * time.Millisecond)
		}
		ev.MouseButton(events.MouseUp, events.Left, to, 0)
		waitFor(t, b, func() bool { return lib.Tags()["People/Alice"] == 2 })
	}, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		assert.Equal(t, 2, lib.Tags()["People/Alice"], "moved into People")
		assert.Equal(t, 0, lib.Tags()["Alice"])
		assert.Equal(t, []string{"group:2026", "group:2017", "group:Home", "group:People",
			"group:People/Family", "tag:People/Alice"}, names())
	})
}

// TestTagDialogLast tests that the Tag dialog starts with the tag chosen
// last, with its name selected, so that typing replaces it, and that
// Return tags the photos with it.
func TestTagDialogLast(t *testing.T) {
	lib := smallLibrary(t, 3)
	phs := lib.Photos(nil)
	require.NoError(t, lib.AddTag("2026/Groningen", phs[0]))
	b := core.NewBody()
	a := NewApp(b)
	a.Styler(func(s *styles.Style) {
		s.Min.Set(units.Dp(1200), units.Dp(600))
	})
	a.shown = allKey
	old := Settings.LastTag
	Settings.LastTag = "2026/Groningen"
	defer func() { Settings.LastTag = old }()
	var fields []*core.TextField
	var choosers []*core.Chooser
	core.RenderImage(b, func() {
		b.AsyncLock()
		a.lib = lib
		a.grid.lib = lib
		a.refresh()
		a.grid.Selected = map[int]bool{1: true, 2: true}
		a.Tag()
		b.AsyncUnlock()
	}, func() {
		waitFor(t, b, func() bool {
			fields, choosers = nil, nil
			if sc := b.Scene.RenderWindow().MainScene(); sc != nil && sc != b.Scene {
				sc.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
					if tf, ok := cw.(*core.TextField); ok {
						fields = append(fields, tf)
					}
					if ch, ok := cw.(*core.Chooser); ok {
						choosers = append(choosers, ch)
					}
					return true
				})
			}
			return len(fields) == 2 && fields[1].StateIs(states.Focused)
		})
		b.AsyncLock()
		require.Len(t, fields, 2, "the group and the tag")
		require.Len(t, choosers, 2)
		// (not the Text of the fields, which ends the editing, and the selection)
		assert.Equal(t, "2026", choosers[0].CurrentItem.Value)
		assert.Equal(t, "Groningen", choosers[1].CurrentItem.Value)
		assert.True(t, fields[1].StateIs(states.Focused), "the tag is focused")
		b.AsyncUnlock()
		ev := a.grid.SystemEvents()
		ev.KeyChord('R', 0, 0) // replaces it, as it is selected
		ev.KeyChord('o', 0, 0)
		ev.KeyChord('m', 0, 0)
		ev.KeyChord('e', 0, 0)
		ev.KeyChord(0, key.CodeReturnEnter, 0)
		waitFor(t, b, func() bool { return lib.Tags()["2026/Rome"] == 2 })
	}, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		assert.Equal(t, 2, lib.Tags()["2026/Rome"], "tagged with the new one")
		assert.Equal(t, "2026/Rome", Settings.LastTag)
	})
}
