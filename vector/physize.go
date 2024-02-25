// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vector

import (
	"fmt"

	"cogentcore.org/core/mat32"
	"cogentcore.org/core/units"
)

// PhysSize specifies the physical size of the drawing, when making a new one
type PhysSize struct {

	// select a standard size -- this will set units and size
	StdSize StdSizes

	// for standard size, use first number as width, second as height
	Portrait bool

	// default units to use, e.g., in line widths etc
	Units units.Units

	// drawing size, in Units
	Size mat32.Vec2

	// grid spacing, in units of ViewBox size
	Vector float32
}

func (ps *PhysSize) Defaults() {
	ps.StdSize = Img1280x720
	ps.Units = units.UnitPx
	ps.Size.Set(1280, 720)
	ps.Vector = 12
}

func (ps *PhysSize) Update() {
	if ps.StdSize != CustomSize {
		ps.SetToStdSize()
	}
}

// SetStdSize sets drawing to a standard size
func (ps *PhysSize) SetStdSize(std StdSizes) error {
	ps.StdSize = std
	return ps.SetToStdSize()
}

// SetToStdSize sets drawing to the current standard size value
func (ps *PhysSize) SetToStdSize() error {
	ssv, has := StdSizesMap[ps.StdSize]
	if !has {
		return fmt.Errorf("StdSize: %v not found in StdSizesMap", ps.StdSize)
	}
	ps.Units = ssv.Units
	ps.Size.X = ssv.X
	ps.Size.Y = ssv.Y
	return nil
}

// SetFromSVG sets from svg
func (ps *PhysSize) SetFromSVG(sv *SVGView) {
	ps.Size.X = sv.SSVG().PhysWidth.Val
	ps.Units = sv.SSVG().PhysWidth.Un
	ps.Size.Y = sv.SSVG().PhysHeight.Val
	ps.Vector = sv.Vector
	ps.StdSize = MatchStdSize(ps.Size.X, ps.Size.Y, ps.Units)
}

// SetToSVG sets svg from us
func (ps *PhysSize) SetToSVG(sv *SVGView) {
	sv.SSVG().PhysWidth.Set(ps.Size.X, ps.Units)
	sv.SSVG().PhysHeight.Set(ps.Size.Y, ps.Units)
	sv.Root().ViewBox.Size = ps.Size
	sv.Vector = ps.Vector
}

// StdSizes are standard physical drawing sizes
type StdSizes int32 //enums:enum

func MatchStdSize(wd, ht float32, un units.Units) StdSizes {
	trgl := StdSizeVals{Units: un, X: wd, Y: ht}
	trgp := StdSizeVals{Units: un, X: ht, Y: wd}
	for k, v := range StdSizesMap {
		if *v == trgl || *v == trgp {
			return k
		}
	}
	return CustomSize
}

const (
	// CustomSize =  nonstandard
	CustomSize StdSizes = iota

	// Image 1280x720 Px = 720p
	Img1280x720

	// Image 1920x1080 Px = 1080p HD
	Img1920x1080

	// Image 3840x2160 Px = 4K
	Img3840x2160

	// Image 7680x4320 Px = 8K
	Img7680x4320

	// Image 1024x768 Px = XGA
	Img1024x768

	// Image 720x480 Px = DVD
	Img720x480

	// Image 640x480 Px = VGA
	Img640x480

	// Image 320x240 Px = old CRT
	Img320x240

	// A4 = 210 x 297 mm
	A4

	// USLetter = 8.5 x 11 in = 612 x 792 pt
	USLetter

	// USLegal = 8.5 x 14 in = 612 x 1008 pt
	USLegal

	// A0 = 841 x 1189 mm
	A0

	// A1 = 594 x 841 mm
	A1

	// A2 = 420 x 594 mm
	A2

	// A3 = 297 x 420 mm
	A3

	// A5 = 148 x 210 mm
	A5

	// A6 = 105 x 148 mm
	A6

	// A7 = 74 x 105
	A7

	// A8 = 52 x 74 mm
	A8

	// A9 = 37 x 52
	A9

	// A10 = 26 x 37
	A10
)

// StdSizeVals are values for standard sizes
type StdSizeVals struct {
	Units units.Units
	X     float32
	Y     float32
}

// StdSizesMap is the map of size values for each standard size
var StdSizesMap = map[StdSizes]*StdSizeVals{
	Img1280x720:  {units.UnitPx, 1280, 720},
	Img1920x1080: {units.UnitPx, 1920, 1080},
	Img3840x2160: {units.UnitPx, 3840, 2160},
	Img7680x4320: {units.UnitPx, 7680, 4320},
	Img1024x768:  {units.UnitPx, 1024, 768},
	Img720x480:   {units.UnitPx, 720, 480},
	Img640x480:   {units.UnitPx, 640, 480},
	Img320x240:   {units.UnitPx, 320, 240},
	A4:           {units.UnitMm, 210, 297},
	USLetter:     {units.UnitPt, 612, 792},
	USLegal:      {units.UnitPt, 612, 1008},
	A0:           {units.UnitMm, 841, 1189},
	A1:           {units.UnitMm, 594, 841},
	A2:           {units.UnitMm, 420, 594},
	A3:           {units.UnitMm, 297, 420},
	A5:           {units.UnitMm, 148, 210},
	A6:           {units.UnitMm, 105, 148},
	A7:           {units.UnitMm, 74, 105},
	A8:           {units.UnitMm, 52, 74},
	A9:           {units.UnitMm, 37, 52},
	A10:          {units.UnitMm, 26, 37},
}
