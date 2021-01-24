// Copyright (c) 2021, The Grid Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grid

import (
	"fmt"

	"github.com/goki/gi/gist"
	"github.com/goki/gi/units"
	"github.com/goki/mat32"
)

// Preferences for drawing size, etc
type DrawingPrefs struct {
	StdSize  StdSizes `desc:"select a standard size -- this will set units and size"`
	Portrait bool     `desc:"for standard size, use first number as width, second as height"`
	Units    units.Units
	Size     mat32.Vec2 `desc:"drawing size, in Units"`
	Scale    mat32.Vec2 `desc:"drawing scale factor"`
	GridDisp bool       `desc:"turns on the grid display"`
	Grid     mat32.Vec2 `desc:"grid spacing, in *integer* units of basic Units"`
}

func (dp *DrawingPrefs) Defaults() {
	dp.StdSize = CustomSize
	dp.Units = units.Pt
	dp.Size.Set(612, 792)
	dp.Scale.Set(1, 1)
	dp.GridDisp = true
	dp.Grid.Set(12, 12)
}

func (dp *DrawingPrefs) Update() {
	if dp.StdSize != CustomSize {
		dp.SetStdSize(dp.StdSize)
	}
}

// SetStdSize sets drawing to a standard size
func (dp *DrawingPrefs) SetStdSize(std StdSizes) error {
	ssv, has := StdSizesMap[std]
	if !has {
		return fmt.Errorf("StdSize: %v not found in StdSizesMap")
	}
	dp.StdSize = std
	dp.Units = ssv.Units
	dp.Size.X = ssv.X
	dp.Size.Y = ssv.Y
	return nil
}

// Preferences is the overall Grid preferences
type Preferences struct {
	Drawing DrawingPrefs `desc:"default new drawing prefs"`
	Style   gist.Paint   `desc:"default styles"`
}

func (pr *Preferences) Defaults() {
	pr.Drawing.Defaults()
	pr.Style.Defaults()
}

func (pr *Preferences) Update() {
	pr.Drawing.Update()
	// pr.Style.Update()
}
