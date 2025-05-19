// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package canvas

import (
	"os"
	"path/filepath"
	"slices"

	"cogentcore.org/core/base/iox/tomlx"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
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
	Size PhysSize

	// default shape styles
	ShapeStyle styles.Paint

	// default text styles
	TextStyle styles.Paint

	// default line styles
	PathStyle styles.Paint

	// default line styles
	LineStyle styles.Paint

	// turns on the grid display
	GridDisp bool

	// snap positions and sizes to underlying grid
	SnapGrid bool

	// snap positions and sizes to line up with other elements
	SnapGuide bool

	// snap node movements to align with guides
	SnapNodes bool

	// number of screen pixels around target point (in either direction) to snap
	SnapTol int `min:"1"`

	// named-split config in use for configuring the splitters
	SplitName SplitName

	// environment variables to set for this app -- if run from the command line, standard shell environment variables are inherited, but on some OS's (Mac), they are not set when run as a gui app
	EnvVars map[string]string
}

func (se *SettingsData) Defaults() {
	se.Size.Defaults()
	se.ShapeStyle.Defaults()
	// se.ShapeStyle.Font.Family = string(core.AppearanceSettings.Font)
	// se.ShapeStyle.Font.Size = 1
	se.ShapeStyle.Fill.Color = colors.Scheme.OnSurface
	se.TextStyle.Defaults()
	// se.TextStyle.Font.Family = string(core.AppearanceSettings.Font)
	// se.TextStyle.Font.Size.Dp(16)
	se.TextStyle.Fill.Color = colors.Scheme.OnSurface
	se.PathStyle.Defaults()
	// se.PathStyle.Font.Family = string(core.AppearanceSettings.Font)
	// se.PathStyle.Font.Size.Dp(16)
	se.PathStyle.Stroke.Color = colors.Scheme.OnSurface
	se.LineStyle.Defaults()
	// se.LineStyle.Font.Family = string(core.AppearanceSettings.Font)
	// se.LineStyle.Font.Size.Dp(16)
	se.LineStyle.Stroke.Color = colors.Scheme.OnSurface
	se.GridDisp = true
	se.SnapTol = 3
	se.SnapGrid = true
	se.SnapGuide = true
	se.SnapNodes = true
	home := core.SystemSettings.User.HomeDir
	se.EnvVars = map[string]string{
		"PATH": home + "/bin:" + home + "/go/bin:/usr/local/bin:/opt/homebrew/bin:/opt/homebrew/shbin:/Library/TeX/texbin:/usr/bin:/bin:/usr/sbin:/sbin",
	}
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

// Apply settings updates things according with settings
func (se *SettingsData) Apply() { //types:add
	for k, v := range se.EnvVars {
		os.Setenv(k, v)
	}
}

// EditSplits opens the SplitsView editor to customize saved splitter settings
func (se *SettingsData) EditSplits() {
	SplitsView(&AvailableSplits)
}
