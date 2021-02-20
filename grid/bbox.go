// Copyright (c) 2021, The Grid Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grid

import (
	"image"

	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
)

// BBoxPoints are the different control points within a bounding box
type BBoxPoints int

const (
	BBLeft BBoxPoints = iota
	BBCenter
	BBRight
	BBTop
	BBMiddle
	BBBottom
	BBoxPointsN
)

//go:generate stringer -type=BBoxPoints

var KiT_BBoxPoints = kit.Enums.AddEnum(BBoxPointsN, kit.NotBitFlag, nil)

func (ev BBoxPoints) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *BBoxPoints) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// ValRect returns the relevant value for a given bounding box
// as an image.Rectangle
func (ev BBoxPoints) ValRect(bb image.Rectangle) float32 {
	switch ev {
	case BBLeft:
		return float32(bb.Min.X)
	case BBCenter:
		return 0.5 * float32(bb.Min.X+bb.Max.X)
	case BBRight:
		return float32(bb.Max.X)
	case BBTop:
		return float32(bb.Min.Y)
	case BBMiddle:
		return 0.5 * float32(bb.Min.Y+bb.Max.Y)
	case BBBottom:
		return float32(bb.Max.Y)
	}
	return 0
}

// ValBox returns the relevant value for a given bounding box as a
// mat32.Box2
func (ev BBoxPoints) ValBox(bb mat32.Box2) float32 {
	switch ev {
	case BBLeft:
		return bb.Min.X
	case BBCenter:
		return 0.5 * (bb.Min.X + bb.Max.X)
	case BBRight:
		return bb.Max.X
	case BBTop:
		return bb.Min.Y
	case BBMiddle:
		return 0.5 * (bb.Min.Y + bb.Max.Y)
	case BBBottom:
		return bb.Max.Y
	}
	return 0
}

// MoveDelta moves overall bbox (Max and Min) by delta (X or Y depending on pt)
func (ev BBoxPoints) MoveDelta(bb *mat32.Box2, delta float32) {
	switch ev {
	case BBLeft, BBCenter, BBRight:
		bb.Min.X += delta
		bb.Max.X += delta
	default:
		bb.Min.Y += delta
		bb.Max.Y += delta
	}
}
