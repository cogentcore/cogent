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

// SetValBox sets the relevant value for a given bounding box as a
// mat32.Box2
func (ev BBoxPoints) SetValBox(bb *mat32.Box2, val float32) {
	switch ev {
	case BBLeft:
		bb.Min.X = val
	case BBCenter:
		bb.Min.X = val // not well defined
	case BBRight:
		bb.Max.X = val
	case BBTop:
		bb.Min.Y = val
	case BBMiddle:
		bb.Min.Y = val
	case BBBottom:
		bb.Max.Y = val
	}
}

// Dim returns the relevant dimension for this point
func (ev BBoxPoints) Dim() mat32.Dims {
	switch ev {
	case BBLeft, BBCenter, BBRight:
		return mat32.X
	default:
		return mat32.Y
	}
}

// ReshapeBBoxPoints returns the X and Y BBoxPoints for given sprite Reshape
// control point.
func ReshapeBBoxPoints(reshape Sprites) (bbX, bbY BBoxPoints) {
	switch reshape {
	case ReshapeUpL:
		return BBLeft, BBTop
	case ReshapeUpC:
		return BBCenter, BBTop
	case ReshapeUpR:
		return BBRight, BBTop
	case ReshapeDnL:
		return BBLeft, BBBottom
	case ReshapeDnC:
		return BBCenter, BBBottom
	case ReshapeDnR:
		return BBRight, BBBottom
	case ReshapeLfM:
		return BBLeft, BBMiddle
	case ReshapeRtM:
		return BBRight, BBMiddle
	}
	return
}

// PointRect returns the relevant point for a given bounding box, where
// relevant dimension is from ValRect and other is midpoint -- for drawing lines.
// BBox is an image.Rectangle
func (ev BBoxPoints) PointRect(bb image.Rectangle) mat32.Vec2 {
	val := ev.ValRect(bb)
	switch ev {
	case BBLeft, BBCenter, BBRight:
		return mat32.Vec2{val, 0.5 * float32(bb.Min.Y+bb.Max.Y)}
	default:
		return mat32.Vec2{0.5 * float32(bb.Min.X+bb.Max.X), val}
	}
}

// PointBox returns the relevant point for a given bounding box, where
// relevant dimension is from ValRect and other is midpoint -- for drawing lines.
// BBox is an image.Rectangle
func (ev BBoxPoints) PointBox(bb mat32.Box2) mat32.Vec2 {
	val := ev.ValBox(bb)
	switch ev {
	case BBLeft, BBCenter, BBRight:
		return mat32.Vec2{val, 0.5 * (bb.Min.Y + bb.Max.Y)}
	default:
		return mat32.Vec2{0.5 * (bb.Min.X + bb.Max.X), val}
	}
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

// BBoxReshapeDelta moves given target dimensions by delta amounts
func BBoxReshapeDelta(bb *mat32.Box2, delta float32, bbX, bbY BBoxPoints) {
	switch bbX {
	case BBLeft:
		bb.Min.X += delta
	case BBRight:
		bb.Max.X += delta
	}
	switch bbY {
	case BBTop:
		bb.Min.Y += delta
	case BBBottom:
		bb.Max.Y += delta
	}
}
