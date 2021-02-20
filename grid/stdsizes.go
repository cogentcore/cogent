// Copyright (c) 2021, The Grid Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grid

import (
	"github.com/goki/gi/units"
	"github.com/goki/ki/kit"
)

type StdSizes int

const (
	// CustomSize =  nonstandard
	CustomSize StdSizes = iota

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
	A4:       &StdSizeVals{units.Mm, 210, 297},
	USLetter: &StdSizeVals{units.Pt, 612, 792},
}
