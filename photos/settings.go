// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package photos

import (
	"os"
	"path/filepath"
	"time"

	"cogentcore.org/core/core"
)

func init() {
	core.AddAppSettings(Settings)
}

// Settings are the settings of Cogent Photos.
var Settings = &SettingsData{
	SettingsBase: core.SettingsBase{
		Name: "Photos",
		File: filepath.Join(core.TheApp.DataDir(), "Cogent Photos", "settings.toml"),
	},
}

// SettingsData is the type of the settings of Cogent Photos.
type SettingsData struct { //types:add
	core.SettingsBase

	// Library is the directory of the library of photos, with the photos
	// in its All directory. It is ~/Pictures/CogentPhotos by default.
	// The app must be started again for a change to take effect.
	Library core.Filename

	// Sort is the order of the photos, by the date they were taken.
	Sort SortOrders

	// ThumbSize is the size of the thumbnails in the grid, in dp.
	ThumbSize float32 `default:"200" min:"48" max:"512"`

	// GoogleAccount is the Google account whose Google Photos photos are
	// synced, which is chosen when signing in the first time.
	GoogleAccount string

	// GoogleSynced is when photos were last synced from Google Photos.
	GoogleSynced time.Time `display:"-"`

	// Splits are the proportions of the panels.
	Splits []float32 `display:"-"`

	// ShownTag is the tag shown last (see [tagRow.key]), which is shown
	// when the app starts again.
	ShownTag string `display:"-"`

	// Views are where the photos of each tag were viewed last, by the keys
	// of the tags, for going back there.
	Views map[string]TagView `display:"-"`

	// ClosedYears are the years closed in the tree of tags.
	ClosedYears map[string]bool `display:"-"`
}

func (sd *SettingsData) Defaults() {
	sd.ThumbSize = 200
}

// libraryDir returns the directory of the library.
func (sd *SettingsData) libraryDir() string {
	if sd.Library != "" {
		return string(sd.Library)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "CogentPhotos"
	}
	return filepath.Join(home, "Pictures", "CogentPhotos")
}

// saveSettings saves the settings, showing any error on the given widget.
func saveSettings(w core.Widget) {
	os.MkdirAll(filepath.Dir(Settings.Filename()), 0o755)
	core.ErrorSnackbar(w, core.SaveSettings(Settings), "Error saving settings")
}

// SortOrders are the orders of the photos in the grid.
type SortOrders int32 //enums:enum

const (
	// NewestFirst shows the newest photos first, at the top.
	NewestFirst SortOrders = iota

	// OldestFirst shows the oldest photos first, at the top.
	OldestFirst
)
