// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package canvas

import (
	"path/filepath"
	"slices"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/base/iox/jsonx"
	"cogentcore.org/core/base/iox/tomlx"
	"cogentcore.org/core/core"
)

func init() {
	core.TheApp.SetName("Cogent Canvas")
	core.AllSettings = slices.Insert(core.AllSettings, 1, core.Settings(Settings))
}

// Settings are the overall Code settings
var Settings = &SettingsData{
	SettingsBase: core.SettingsBase{
		Name: "Canvas",
		File: filepath.Join(core.TheApp.DataDir(), "Cogent Canvas", "settings.toml"),
	},
}

// SettingsData is the overall Vector settings
type SettingsData struct { //types:add
	core.SettingsBase

	// default physical size, when app is started without opening a file
	Size PhysicalSize `display:"add-fields"`

	// turns on the grid display
	ShowGrid bool `default:"true"`

	// snap to grid intervals.
	SnapGrid bool `default:"true"`

	// snap to align with other elements.
	SnapAlign bool `default:"true"`

	// snap individual path nodes using snap settings, while drawing and editing nodes.
	SnapNodes bool `default:"true"`

	// zone of attraction around a potential snapping point (grid, align) where
	// snapping happens. Small values here allow a reasonable balance of snapping
	// and non-snapping, while larger values produce more visible and strong snapping
	// constraints.
	SnapZone int `min:"1" default:"3"`

	// enables saving of metadata about the image (in inkscape-compatible format)
	MetaData bool
}

func (se *SettingsData) Defaults() {
	se.Size.Defaults()
	se.ShowGrid = true
	se.SnapGrid = true
	se.SnapAlign = true
	se.SnapNodes = true
	se.SnapZone = 3
	se.MetaData = true
}

func (se *SettingsData) Update() {
	se.Size.Update()
}

func (se *SettingsData) Save() error {
	SavePaths()
	SaveSplits()
	return tomlx.Save(se, se.Filename())
}

func (se *SettingsData) Open() error {
	OpenPaths()
	OpenSplits()
	return tomlx.Open(se, se.Filename())
}

////////  Recents

var (
	// RecentPaths is a slice of recent file paths
	RecentPaths core.FilePaths

	// SavedPathsFilename is the name of the saved file paths file in Cogent Code data directory
	SavedPathsFilename = "saved-paths.json"
)

// SavePaths saves the active SavedPaths to settings dir
func SavePaths() {
	pdir := core.TheApp.AppDataDir()
	pnm := filepath.Join(pdir, SavedPathsFilename)
	RecentPaths.Save(pnm)
}

// OpenPaths loads the active SavedPaths from settings dir
func OpenPaths() {
	pdir := core.TheApp.AppDataDir()
	pnm := filepath.Join(pdir, SavedPathsFilename)
	RecentPaths.Open(pnm)
}

////////   Splits

var (
	// Splits are the proportions for main window splits, saved and loaded
	Splits = [3]float32{0.15, 0.60, 0.25}

	// SplitsSettingsFilename is the name of the settings file in App prefs
	// directory for saving / loading the current splits
	SplitsSettingsFilename = "splits-settings.json"
)

// OpenSplits opens last saved splits from settings file.
func OpenSplits() {
	pdir := core.TheApp.AppDataDir()
	pnm := filepath.Join(pdir, SplitsSettingsFilename)
	if !errors.Log1(fsx.FileExists(pnm)) {
		return
	}
	jsonx.Open(&Splits, pnm)
}

// SaveSplits saves named splits to a json-formatted file.
func SaveSplits() {
	pdir := core.TheApp.AppDataDir()
	pnm := filepath.Join(pdir, SplitsSettingsFilename)
	jsonx.Save(&Splits, pnm)
}
