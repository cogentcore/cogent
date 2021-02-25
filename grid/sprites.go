// Copyright (c) 2021, The Grid Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grid

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"

	"github.com/chewxy/math32"
	"github.com/goki/gi/gi"
	"github.com/goki/gi/gist"
	"github.com/goki/gi/oswin"
	"github.com/goki/ki/ints"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

type Sprites int

const (
	ReshapeUpL Sprites = iota
	ReshapeUpC
	ReshapeUpR
	ReshapeDnL
	ReshapeDnC
	ReshapeDnR
	ReshapeLfM
	ReshapeRtM

	RubberBandT
	RubberBandR
	RubberBandL
	RubberBandB

	AlignMatch1
	AlignMatch2
	AlignMatch3
	AlignMatch4
	AlignMatch5
	AlignMatch6
	AlignMatch7
	AlignMatch8

	SpritesN
	// beyond this are the node markers!
)

//go:generate stringer -type=Sprites

var KiT_Sprites = kit.Enums.AddEnum(SpritesN, kit.NotBitFlag, nil)

func (ev Sprites) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *Sprites) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

var SpriteNames = map[Sprites]string{
	ReshapeUpL: "grid-size-up-l",
	ReshapeUpC: "grid-size-up-m",
	ReshapeUpR: "grid-size-up-r",
	ReshapeDnL: "grid-size-dn-l",
	ReshapeDnC: "grid-size-dn-m",
	ReshapeDnR: "grid-size-dn-r",
	ReshapeLfM: "grid-size-lf-c",
	ReshapeRtM: "grid-size-rt-c",

	RubberBandT: "rubber-band-t",
	RubberBandR: "rubber-band-r",
	RubberBandL: "rubber-band-l",
	RubberBandB: "rubber-band-b",

	AlignMatch1: "align-match-1",
	AlignMatch2: "align-match-2",
	AlignMatch3: "align-match-3",
	AlignMatch4: "align-match-4",
	AlignMatch5: "align-match-5",
	AlignMatch6: "align-match-6",
	AlignMatch7: "align-match-7",
	AlignMatch8: "align-match-8",
}

var (
	HandleSpriteScale = float32(18)
	HandleSizeMin     = 4
	HandleBorderMin   = 2
)

// HandleSpriteSize returns the border size and overall size of handle-type sprites
func HandleSpriteSize() (int, image.Point) {
	sz := int(math32.Ceil(gi.Prefs.LogicalDPIScale * HandleSpriteScale))
	sz = ints.MaxInt(sz, HandleSizeMin)
	bsz := ints.MaxInt(sz/6, HandleBorderMin)
	bbsz := image.Point{sz, sz}
	return bsz, bbsz
}

// DrawSpriteSize renders a Size sprite handle
func DrawSpriteSize(spi Sprites, sp *gi.Sprite, bsz int, bbsz image.Point) {
	ibd := sp.Pixels.Bounds()
	bbd := ibd
	bbd.Min.X += bsz
	bbd.Min.Y += bsz
	bbd.Max.X -= bsz
	bbd.Max.Y -= bsz
	draw.Draw(sp.Pixels, ibd, &image.Uniform{color.White}, image.ZP, draw.Src)
	draw.Draw(sp.Pixels, bbd, &image.Uniform{color.Black}, image.ZP, draw.Src)
}

var (
	LineSpriteScale = float32(8)
	LineSizeMin     = 3
	LineBorderMin   = 1
)

// LineSpriteSize returns the border size and overall size of line-type sprites
func LineSpriteSize() (int, int) {
	sz := int(math32.Ceil(gi.Prefs.LogicalDPIScale * LineSpriteScale))
	sz = ints.MaxInt(sz, LineSizeMin)
	bsz := ints.MaxInt(sz/6, LineBorderMin)
	return bsz, sz
}

// DrawRubberBandHoriz renders a horizontal rubber band line
func DrawRubberBandHoriz(spi Sprites, sp *gi.Sprite, bsz, sz int, trgsz image.Point) {
	ssz := image.Point{trgsz.X, sz}
	if !sp.SetSize(ssz) { // already set
		return
	}
	ibd := sp.Pixels.Bounds()
	bbd := ibd
	bbd.Min.Y += bsz
	bbd.Max.Y -= bsz
	draw.Draw(sp.Pixels, ibd, &image.Uniform{color.White}, image.ZP, draw.Src)
	for x := 0; x < ssz.X; x += sz * 2 {
		bbd.Min.X = x
		bbd.Max.X = x + sz
		draw.Draw(sp.Pixels, bbd, &image.Uniform{color.Black}, image.ZP, draw.Src)
	}
}

// DrawRubberBandVert renders a vertical rubber band line
func DrawRubberBandVert(spi Sprites, sp *gi.Sprite, bsz, sz int, trgsz image.Point) {
	ssz := image.Point{sz, trgsz.Y}
	if !sp.SetSize(ssz) { // already set
		return
	}
	ibd := sp.Pixels.Bounds()
	bbd := ibd
	bbd.Min.X += bsz
	bbd.Max.X -= bsz
	draw.Draw(sp.Pixels, ibd, &image.Uniform{color.White}, image.ZP, draw.Src)
	for y := sz; y < ssz.Y; y += sz * 2 {
		bbd.Min.Y = y
		bbd.Max.Y = y + sz
		draw.Draw(sp.Pixels, bbd, &image.Uniform{color.Black}, image.ZP, draw.Src)
	}
}

// DrawAlignMatchHoriz renders a horizontal alignment line
func DrawAlignMatchHoriz(spi Sprites, sp *gi.Sprite, bsz, sz int, trgsz image.Point) {
	ssz := image.Point{trgsz.X, sz}
	if !sp.SetSize(ssz) { // already set
		return
	}
	ibd := sp.Pixels.Bounds()
	bbd := ibd
	bbd.Min.Y += bsz
	bbd.Max.Y -= bsz
	clr := gist.Color{0, 200, 200, 255}
	draw.Draw(sp.Pixels, ibd, &image.Uniform{color.White}, image.ZP, draw.Src)
	draw.Draw(sp.Pixels, bbd, &image.Uniform{clr}, image.ZP, draw.Src)
}

// DrawAlignMatchVert renders a vertical alignment line
func DrawAlignMatchVert(spi Sprites, sp *gi.Sprite, bsz, sz int, trgsz image.Point) {
	ssz := image.Point{sz, trgsz.Y}
	if !sp.SetSize(ssz) { // already set
		return
	}
	ibd := sp.Pixels.Bounds()
	bbd := ibd
	bbd.Min.X += bsz
	bbd.Max.X -= bsz
	clr := gist.Color{0, 200, 200, 255}
	draw.Draw(sp.Pixels, ibd, &image.Uniform{color.White}, image.ZP, draw.Src)
	draw.Draw(sp.Pixels, bbd, &image.Uniform{clr}, image.ZP, draw.Src)
}

func SpriteName(spi Sprites) string {
	if spi < SpritesN {
		return SpriteNames[spi]
	}
	return fmt.Sprintf("path-point-%d", spi-SpritesN)
}

// Sprite returns given sprite -- renders to window if not yet made.
// trgsz is the target size (e.g., for rubber band boxes)
func Sprite(spi Sprites, win *gi.Window, trgsz image.Point) *gi.Sprite {
	spnm := SpriteName(spi)
	sp, ok := win.SpriteByName(spnm)
	switch {
	case spi >= ReshapeUpL && spi <= ReshapeRtM:
		if !ok {
			bsz, bbsz := HandleSpriteSize()
			sp = win.AddNewSprite(spnm, bbsz, image.ZP)
			DrawSpriteSize(spi, sp, bsz, bbsz)
		}
	case spi >= RubberBandT && spi <= RubberBandB:
		bsz, sz := LineSpriteSize()
		switch spi {
		case RubberBandT, RubberBandB:
			if !ok {
				sp = win.AddNewSprite(spnm, image.Point{trgsz.X, sz}, image.ZP)
			}
			DrawRubberBandHoriz(spi, sp, bsz, sz, trgsz)
		case RubberBandR, RubberBandL:
			if !ok {
				sp = win.AddNewSprite(spnm, image.Point{sz, trgsz.Y}, image.ZP)
			}
			DrawRubberBandVert(spi, sp, bsz, sz, trgsz)
		}
	case spi >= AlignMatch1 && spi <= AlignMatch8:
		bsz, sz := LineSpriteSize()
		switch {
		case trgsz.X > trgsz.Y:
			if !ok {
				sp = win.AddNewSprite(spnm, image.Point{trgsz.X, sz}, image.ZP)
			}
			DrawAlignMatchHoriz(spi, sp, bsz, sz, trgsz)
		default:
			if !ok {
				sp = win.AddNewSprite(spnm, image.Point{sz, trgsz.Y}, image.ZP)
			}
			DrawAlignMatchVert(spi, sp, bsz, sz, trgsz)
		}
	case spi >= SpritesN:
		if !ok {
			bsz, bbsz := HandleSpriteSize()
			sp = win.AddNewSprite(spnm, bbsz, image.ZP)
			DrawSpriteSize(spi, sp, bsz, bbsz)
		}
	}
	return sp
}

// SpriteConnectEvent activates and sets mouse event functions to given function
func SpriteConnectEvent(spi Sprites, win *gi.Window, trgsz image.Point, recv ki.Ki, fun ki.RecvFunc) *gi.Sprite {
	sp := Sprite(spi, win, trgsz)
	if recv != nil {
		sp.ConnectEvent(recv, oswin.MouseEvent, fun)
		sp.ConnectEvent(recv, oswin.MouseDragEvent, fun)
	}
	win.ActivateSprite(sp.Name)
	return sp
}

// SetSpritePos sets sprite position, taking into account relative offsets
func SetSpritePos(spi Sprites, sp *gi.Sprite, pos image.Point) {
	switch {
	case spi >= ReshapeUpL && spi <= ReshapeRtM:
		_, sz := HandleSpriteSize()
		if spi == ReshapeDnL || spi == ReshapeUpL || spi == ReshapeLfM {
			pos.X -= sz.X
		}
		if spi == ReshapeUpL || spi == ReshapeUpC || spi == ReshapeUpR {
			pos.Y -= sz.Y
		}
		sp.Geom.Pos = pos
	case spi >= RubberBandT && spi <= RubberBandB:
		_, sz := LineSpriteSize()
		switch spi {
		case RubberBandT:
			pos.Y -= sz
		case RubberBandL:
			pos.X -= sz
		}
		sp.Geom.Pos = pos
	case spi >= AlignMatch1 && spi <= AlignMatch8:
		_, sz := LineSpriteSize()
		bbt := sp.Props.Prop("bbox").(BBoxPoints)
		switch bbt {
		case BBLeft:
			pos.X -= sz
		case BBCenter:
			pos.X -= sz / 2
		case BBTop:
			pos.Y -= sz
		case BBMiddle:
			pos.Y -= sz / 2
		}
		sp.Geom.Pos = pos
	case spi >= SpritesN:
		_, sz := HandleSpriteSize()
		pos.X -= sz.X / 2
		pos.Y -= sz.Y / 2
		sp.Geom.Pos = pos
	}
}

// InactivateSprites inactivates all of our sprites
func InactivateSprites(win *gi.Window) {
	for spi := Sprites(0); spi < SpritesN; spi++ {
		spnm := SpriteNames[spi]
		win.InactivateSprite(spnm)
	}
}

// InactivateSpriteRange inactivates range of sprites (end is inclusive)
func InactivateSpriteRange(win *gi.Window, st, end Sprites) {
	for spi := st; spi <= end; spi++ {
		spnm := SpriteNames[spi]
		win.InactivateSprite(spnm)
	}
}
