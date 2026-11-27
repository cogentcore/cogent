// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package photos

import (
	"fmt"
	"math"
	"slices"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
)

// The photos taken near a photo, within a radius of its location, are
// shown in a tab of their own, with a slider for the radius, as the other
// searches that are not of the tag shown, which is in the first tab.

// earthRadius is the radius of the Earth in km.
const earthRadius = 6371.0

// distance returns the distance between the given locations in km, on
// the surface of the Earth (the haversine formula).
func distance(a, b GPSCoord) float64 {
	rad := math.Pi / 180
	dlat := (b.Lat - a.Lat) * rad
	dlong := (b.Long - a.Long) * rad
	h := math.Sin(dlat/2)*math.Sin(dlat/2) +
		math.Cos(a.Lat*rad)*math.Cos(b.Lat*rad)*math.Sin(dlong/2)*math.Sin(dlong/2)
	return 2 * earthRadius * math.Asin(math.Sqrt(min(h, 1)))
}

// PhotosNear returns the photos (not in the trash) taken within the given
// radius in km of the given location, by the date they were taken, the
// newest first if newest.
func (lib *Library) PhotosNear(loc GPSCoord, radius float64, newest bool) []*Photo {
	phs := lib.Photos(func(ph *Photo) bool {
		return ph.Trashed.IsZero() && !ph.GPS.IsZero() && distance(loc, ph.GPS) <= radius
	})
	if newest {
		slices.Reverse(phs)
	}
	return phs
}

// The radius of nearby photos is from 0.1 to 50 km, on a log scale.
const (
	nearbyMinRadius     = 0.1
	nearbyMaxRadius     = 50
	nearbyDefaultRadius = 1
)

// nearbyView is a tab of the photos taken near a photo.
type nearbyView struct {
	app *App

	// center is the photo whose location is the center.
	center *Photo

	// radius is the radius in km.
	radius float64

	grid  *Grid
	label *core.Text
}

// update shows the photos within the radius.
func (nv *nearbyView) update() {
	phs := nv.app.lib.PhotosNear(nv.center.GPS, nv.radius, Settings.Sort == NewestFirst)
	setGridPhotos(nv.grid, phs)
	// the center photo is current at first
	if nv.grid.Current <= 0 {
		if i := slices.Index(phs, nv.center); i >= 0 {
			nv.grid.Current = i
			nv.grid.Selected = map[int]bool{i: true}
		}
	}
	nv.grid.NeedsLayout()
	if nv.label != nil {
		nv.label.SetText(fmt.Sprintf("%d photos within %s", len(phs), formatRadius(nv.radius))).UpdateRender()
	}
}

// formatRadius returns the given radius in km as text.
func formatRadius(r float64) string {
	if r < 1 {
		return fmt.Sprintf("%.0f m", r*1000)
	}
	if r < 10 {
		return fmt.Sprintf("%.1f km", r)
	}
	return fmt.Sprintf("%.0f km", r)
}

// PhotosNearby shows the photos taken near the current photo, within a
// radius of its location, which can be changed, in a tab of their own.
func (a *App) PhotosNearby() { //types:add
	ph := a.current()
	if ph == nil {
		return
	}
	if ph.GPS.IsZero() {
		core.MessageSnackbar(a, "The photo has no location")
		return
	}
	nv := &nearbyView{app: a, center: ph, radius: nearbyDefaultRadius}
	fr, tab := a.tabs.NewTab("Near " + ph.DateTaken.Format("Jan 2, 2006"))
	tab.SetIcon(icons.MyLocation)
	fr.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
	})
	row := core.NewFrame(fr)
	row.Styler(func(s *styles.Style) {
		s.Align.Items = styles.Center
		s.Grow.Set(1, 0)
	})
	core.NewIcon(row).SetIcon(icons.MyLocation)
	sl := core.NewSlider(row)
	// the slider is the log10 of the radius
	sl.SetMin(float32(math.Log10(nearbyMinRadius))).SetMax(float32(math.Log10(nearbyMaxRadius))).SetStep(0.05)
	sl.SetValue(float32(math.Log10(nv.radius)))
	sl.SetTooltip("The radius of the photos nearby")
	sl.FinalStyler(func(s *styles.Style) { // after the 20em of sliders
		s.Min.X.Em(12)
		s.Max.X.Em(12)
		s.Grow.Set(0, 0)
	})
	nv.label = core.NewText(row)
	setRadius := func(e events.Event) {
		nv.radius = math.Pow(10, float64(sl.Value))
		nv.update()
	}
	sl.OnInput(setRadius)
	sl.OnChange(setRadius)
	nv.grid = a.newGridIn(fr)
	nv.grid.onDestroy = func() { // the tab is closed
		a.nearby = slices.DeleteFunc(a.nearby, func(o *nearbyView) bool { return o == nv })
		if a.grid == nv.grid {
			a.grid = a.main
		}
	}
	a.nearby = append(a.nearby, nv)
	nv.update()
	a.tabs.SelectTabIndex(a.tabs.NumTabs() - 1)
}
