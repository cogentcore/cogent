// Copyright (c) 2021, The Grid Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grid

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/chewxy/math32"
	"github.com/goki/gi/gi"
	"github.com/goki/gi/oswin"
	"github.com/goki/ki/ints"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

type Sprites int

const (
	SizeUpL Sprites = iota
	SizeUpM
	SizeUpR
	SizeDnL
	SizeDnM
	SizeDnR
	SizeLfC
	SizeRtC

	RubberBandT
	RubberBandR
	RubberBandL
	RubberBandB

	SpritesN
)

//go:generate stringer -type=Sprites

var KiT_Sprites = kit.Enums.AddEnum(SpritesN, kit.NotBitFlag, nil)

func (ev Sprites) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *Sprites) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

var SpriteNames = map[Sprites]string{
	SizeUpL: "grid-size-up-l",
	SizeUpM: "grid-size-up-m",
	SizeUpR: "grid-size-up-r",
	SizeDnL: "grid-size-dn-l",
	SizeDnM: "grid-size-dn-m",
	SizeDnR: "grid-size-dn-r",
	SizeLfC: "grid-size-lf-c",
	SizeRtC: "grid-size-rt-c",

	RubberBandT: "rubber-band-t",
	RubberBandR: "rubber-band-r",
	RubberBandL: "rubber-band-l",
	RubberBandB: "rubber-band-b",
}

var HandleSpriteScale = float32(18)

// HandleSpriteSize returns the border size and overall size of handle-type sprites
func HandleSpriteSize() (int, image.Point) {
	sz := int(math32.Ceil(gi.Prefs.LogicalDPIScale * HandleSpriteScale))
	if sz < 4 {
		sz = 4
	}
	bsz := ints.MaxInt(sz/6, 2)
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

var LineSpriteScale = float32(10)

// LineSpriteSize returns the border size and overall size of line-type sprites
func LineSpriteSize() (int, int) {
	sz := int(math32.Ceil(gi.Prefs.LogicalDPIScale * LineSpriteScale))
	if sz < 4 {
		sz = 4
	}
	bsz := ints.MaxInt(sz/6, 2)
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

// Sprite returns given sprite -- renders to window if not yet made.
// trgsz is the target size (e.g., for rubber band boxes)
func Sprite(spi Sprites, win *gi.Window, trgsz image.Point) *gi.Sprite {
	spnm := SpriteNames[spi]
	sp, ok := win.SpriteByName(spnm)
	switch {
	case spi >= SizeUpL && spi <= SizeRtC:
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
	case spi >= SizeUpL && spi <= SizeRtC:
		_, sz := HandleSpriteSize()
		if spi == SizeDnL || spi == SizeUpL || spi == SizeLfC {
			pos.X -= sz.X
		}
		if spi == SizeUpL || spi == SizeUpM || spi == SizeUpR {
			pos.Y -= sz.Y
		}
		sp.Geom.Pos = pos
	case spi >= RubberBandT && spi <= RubberBandB:
		_, sz := LineSpriteSize()
		switch spi {
		case RubberBandT:
			pos.Y -= sz
		// case RubberBandB:
		// 	pos.Y += sz
		// case RubberBandR:
		// 	pos.X += sz
		case RubberBandL:
			pos.X -= sz
		}
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
