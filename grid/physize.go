// Copyright (c) 2021, The Grid Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grid

import (
	"fmt"

	"github.com/goki/gi/units"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
)

// PhysSize specifies the physical size of the drawing, when making a new one
type PhysSize struct {
	StdSize  StdSizes    `desc:"select a standard size -- this will set units and size"`
	Portrait bool        `desc:"for standard size, use first number as width, second as height"`
	Units    units.Units `desc:"default units to use, e.g., in line widths etc"`
	Size     mat32.Vec2  `desc:"drawing size, in Units"`
}

var KiT_PhysSize = kit.Types.AddType(&PhysSize{}, nil)

func (dp *PhysSize) Defaults() {
	dp.StdSize = Img1280x720
	dp.Units = units.Px
	dp.Size.Set(1280, 720)
}

func (dp *PhysSize) Update() {
	if dp.StdSize != CustomSize {
		dp.SetToStdSize()
	}
}

// SetStdSize sets drawing to a standard size
func (dp *PhysSize) SetStdSize(std StdSizes) error {
	dp.StdSize = std
	return dp.SetToStdSize()
}

// SetToStdSize sets drawing to the current standard size value
func (dp *PhysSize) SetToStdSize() error {
	ssv, has := StdSizesMap[dp.StdSize]
	if !has {
		return fmt.Errorf("StdSize: %v not found in StdSizesMap")
	}
	dp.Units = ssv.Units
	dp.Size.X = ssv.X
	dp.Size.Y = ssv.Y
	return nil
}

// StdSizes are standard physical drawing sizes
type StdSizes int

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

	StdSizesN
)

//go:generate stringer -type=StdSizes

var KiT_StdSizes = kit.Enums.AddEnum(StdSizesN, kit.NotBitFlag, nil)

func (ev StdSizes) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *StdSizes) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// StdSizeVals are values for standard sizes
type StdSizeVals struct {
	Units units.Units
	X     float32
	Y     float32
}

// StdSizesMap is the map of size values for each standard size
var StdSizesMap = map[StdSizes]*StdSizeVals{
	Img1280x720:  &StdSizeVals{units.Px, 1280, 720},
	Img1920x1080: &StdSizeVals{units.Px, 1920, 1080},
	Img3840x2160: &StdSizeVals{units.Px, 3840, 2160},
	Img7680x4320: &StdSizeVals{units.Px, 7680, 4320},
	Img1024x768:  &StdSizeVals{units.Px, 1024, 768},
	Img720x480:   &StdSizeVals{units.Px, 720, 480},
	Img640x480:   &StdSizeVals{units.Px, 640, 480},
	Img320x240:   &StdSizeVals{units.Px, 320, 240},
	A4:           &StdSizeVals{units.Mm, 210, 297},
	USLetter:     &StdSizeVals{units.Pt, 612, 792},
	USLegal:      &StdSizeVals{units.Pt, 612, 1008},
	A0:           &StdSizeVals{units.Mm, 841, 1189},
	A1:           &StdSizeVals{units.Mm, 594, 841},
	A2:           &StdSizeVals{units.Mm, 420, 594},
	A3:           &StdSizeVals{units.Mm, 297, 420},
	A5:           &StdSizeVals{units.Mm, 148, 210},
	A6:           &StdSizeVals{units.Mm, 105, 148},
	A7:           &StdSizeVals{units.Mm, 74, 105},
	A8:           &StdSizeVals{units.Mm, 52, 74},
	A9:           &StdSizeVals{units.Mm, 37, 52},
	A10:          &StdSizeVals{units.Mm, 26, 37},
}
