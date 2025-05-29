// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package canvas

import (
	"path/filepath"
	"slices"

	"cogentcore.org/core/base/iox/tomlx"
	"cogentcore.org/core/core"
)

func init() {
	core.TheApp.SetName("Cogent Canvas")
	core.AllSettings = slices.Insert(core.AllSettings, 1, core.Settings(Settings))
	// OpenIcons()
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
	Size PhysSize `display:"add-fields"`

	// turns on the grid display
	GridDisp bool `default:"true"`

	// snap positions and sizes to underlying grid
	SnapGrid bool `default:"true"`

	// snap positions and sizes to line up with other elements
	SnapGuide bool `default:"true"`

	// snap node movements to align with guides
	SnapNodes bool `default:"true"`

	// number of screen pixels around target point (in either direction) to snap
	SnapTol int `min:"1" default:"3"`

	// named-split config in use for configuring the splitters
	SplitName SplitName
}

func (se *SettingsData) Defaults() {
	se.Size.Defaults()
	se.GridDisp = true
	se.SnapGrid = true
	se.SnapGuide = true
	se.SnapNodes = true
	se.SnapTol = 3
}

func (se *SettingsData) Update() {
	se.Size.Update()
}

func (se *SettingsData) Save() error {
	return tomlx.Save(se, se.Filename())
}

func (se *SettingsData) Open() error {
	return tomlx.Open(se, se.Filename())
}

// EditSplits opens the SplitsView editor to customize saved splitter settings
func (se *SettingsData) EditSplits() {
	SplitsView(&AvailableSplits)
}
