// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package photos

import (
	"os"
	"path/filepath"
	"strings"
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

	// Keys are the keyboard shortcuts of the actions in the main window,
	// by their chords, with Command for Meta on the Mac and Control
	// elsewhere.
	Keys KeyMap

	// KeysKnown are the actions whose default keyboard shortcuts have been
	// added to [SettingsData.Keys], so that those of actions added in later
	// versions are added, but not again those that were removed.
	KeysKnown []KeyFunctions `display:"-"`

	// LastTag is the tag chosen last in the Tag dialog, which it starts
	// with the next time, selected, so that typing replaces it.
	LastTag string `display:"-"`

	// Views are where the photos of each tag were viewed last, by the keys
	// of the tags, for going back there.
	Views map[string]TagView `display:"-"`

	// ClosedGroups are the groups closed in the tree of tags.
	ClosedGroups map[string]bool `display:"-"`

	// ClosedYears are the years closed in the tree of tags, before they
	// were groups (see [SettingsData.migrateTags]).
	ClosedYears map[string]bool `display:"-"`
}

// migrateTags changes the keys of tags with years, as tag:2017_Kyoto, and
// of years, as year:2017, from before tags were in groups, into those of
// tags in groups, as tag:2017/Kyoto and group:2017.
func (sd *SettingsData) migrateTags() {
	key := func(k string) string {
		if y, ok := strings.CutPrefix(k, "year:"); ok {
			return groupKey(y)
		}
		if t, ok := strings.CutPrefix(k, "tag:"); ok && !strings.Contains(t, tagSep) {
			if y, n, ok := yearPrefix(t); ok {
				return tagKey(y + tagSep + n)
			}
		}
		return k
	}
	sd.ShownTag = key(sd.ShownTag)
	for k, v := range sd.Views {
		if nk := key(k); nk != k {
			delete(sd.Views, k)
			sd.Views[nk] = v
		}
	}
	if sd.ClosedGroups == nil && sd.ClosedYears != nil {
		sd.ClosedGroups = sd.ClosedYears
	}
	sd.ClosedYears = nil
}

func (sd *SettingsData) Defaults() {
	sd.ThumbSize = 200
}

// Apply adds the default keyboard shortcuts of actions that are new (see
// [SettingsData.addDefaultKeys]). They are not set in Defaults, as the
// saved ones would be merged into them, so that removed ones would come
// back.
func (sd *SettingsData) Apply() {
	sd.addDefaultKeys()
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
