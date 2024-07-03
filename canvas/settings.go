// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package canvas

import (
	"image/color"
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

	// active color settings
	Colors ColorSettings

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
	se.Colors.Defaults()
	se.ShapeStyle.Defaults()
	se.ShapeStyle.FontStyle.Family = "Arial"
	se.ShapeStyle.FontStyle.Size.Px(12)
	// pf.ShapeStyle.FillStyle.Color.SetName("blue")
	// pf.ShapeStyle.StrokeStyle.On = true // todo: image
	// pf.ShapeStyle.FillStyle.On = true
	se.TextStyle.Defaults()
	se.TextStyle.FontStyle.Family = "Arial"
	se.TextStyle.FontStyle.Size.Px(12)
	// pf.TextStyle.StrokeStyle.On = false
	// pf.TextStyle.FillStyle.On = true
	se.PathStyle.Defaults()
	se.PathStyle.FontStyle.Family = "Arial"
	se.PathStyle.FontStyle.Size.Px(12)
	// pf.PathStyle.StrokeStyle.On = true
	// pf.PathStyle.FillStyle.On = false
	se.LineStyle.Defaults()
	se.LineStyle.FontStyle.Family = "Arial"
	se.LineStyle.FontStyle.Size.Px(12)
	// pf.LineStyle.StrokeStyle.On = true
	// pf.LineStyle.FillStyle.On = false
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

/////////////////////////////////////////////////////////////////////////////////
//   ColorSettings

// ColorSettings for
type ColorSettings struct { //types:add

	// drawing background color
	Background color.Color

	// border color of the drawing
	Border color.Color

	// grid line color
	Grid color.Color
}

// todo: replace with color tone defaults

func (se *ColorSettings) Defaults() {
	se.Background = colors.White
	se.Border = colors.Black
	se.Grid = color.RGBA{220, 220, 220, 255}
}

func (se *ColorSettings) DarkDefaults() {
	se.Background = colors.Black
	se.Border = color.RGBA{102, 102, 102, 255}
	se.Grid = color.RGBA{40, 40, 40, 255}
}
