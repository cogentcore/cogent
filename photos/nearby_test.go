// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package photos

import (
	"math"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDistance(t *testing.T) {
	boulder := GPSCoord{Lat: 40.015, Long: -105.2705}
	denver := GPSCoord{Lat: 39.7392, Long: -104.9903}
	assert.InDelta(t, 38.9, distance(boulder, denver), 1)
	assert.InDelta(t, 0, distance(boulder, boulder), 1e-9)
	// a degree of latitude is about 111 km
	assert.InDelta(t, 111.2, distance(GPSCoord{Lat: 0}, GPSCoord{Lat: 1}), 0.5)
}

// TestPhotosNearby tests the tab of the photos near a photo, within a
// radius that its slider changes, and closing it.
func TestPhotosNearby(t *testing.T) {
	lib := smallLibrary(t, 5)
	phs := lib.Photos(nil)
	center := GPSCoord{Lat: 40, Long: -105}
	// north of the center by the given km
	north := func(km float64) GPSCoord { return GPSCoord{Lat: 40 + km/111.2, Long: -105} }
	phs[0].GPS = center
	phs[1].GPS = north(0.5)
	phs[2].GPS = north(3)
	phs[3].GPS = north(30) // and phs[4] has no location
	require.NoError(t, lib.Save(phs...))
	assert.Len(t, lib.PhotosNear(center, 1, false), 2)
	assert.Len(t, lib.PhotosNear(center, 5, false), 3)
	assert.Len(t, lib.PhotosNear(center, 50, false), 4)

	b := core.NewBody()
	a := NewApp(b)
	a.Styler(func(s *styles.Style) {
		s.Min.Set(units.Dp(1200), units.Dp(600))
	})
	a.shown = allKey
	core.RenderImage(b, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		a.lib = lib
		a.main.lib = lib
		a.refresh()
	}, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		i := -1
		for j, ph := range a.grid.Photos {
			if ph == phs[0] {
				i = j
			}
		}
		require.GreaterOrEqual(t, i, 0)
		a.grid.SelectIndex(i)
		a.PhotosNearby()
	}, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		require.Len(t, a.nearby, 1)
		nv := a.nearby[0]
		assert.Equal(t, nv.grid, a.grid, "the grid of the tab shown")
		assert.Len(t, a.grid.Photos, 2, "within 1 km")
		assert.Equal(t, phs[0], a.grid.CurrentPhoto(), "the center is current")
		assert.Equal(t, 2, a.tabs.NumTabs())
		nv.radius = 5
		nv.update()
		assert.Len(t, a.grid.Photos, 3, "within 5 km")
		assert.Contains(t, nv.label.Text, "3 photos within 5.0 km")
		nv.radius = math.Pow(10, math.Log10(nearbyMaxRadius))
		nv.update()
		assert.Len(t, a.grid.Photos, 4, "within 50 km")
		a.tabs.DeleteTabIndex(1)
	}, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		assert.Len(t, a.nearby, 0, "closed")
		assert.Equal(t, a.main, a.grid, "back to the main grid")
		assert.Len(t, a.main.Photos, 5)
	})
}
