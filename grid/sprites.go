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
	SpritesN
)

//go:generate stringer -type=Sprites

var KiT_Sprites = kit.Enums.AddEnumAltLower(SpritesN, kit.NotBitFlag, nil, "")

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
}

var SpriteScale = float32(24)

func SpriteSize() int {
	sz := int(math32.Ceil(gi.Prefs.LogicalDPIScale * SpriteScale))
	if sz < 4 {
		sz = 4
	}
	return sz
}

// todo: store this on EditState, pass editstate to Sprite method
// ActiveSprites are cached only for sprites last accessed by Sprite() method
var ActiveSprites = map[Sprites]*gi.Sprite{}

// Sprite returns given sprite -- renders to window if not yet made
func Sprite(spi Sprites, win *gi.Window) *gi.Sprite {
	spnm := SpriteNames[spi]
	sp, ok := win.SpriteByName(spnm)
	if !ok {
		sz := SpriteSize()
		bsz := ints.MaxInt(sz/6, 1)
		bbsz := image.Point{sz, sz}
		sp = win.AddNewSprite(spnm, bbsz, image.ZP)
		ibd := sp.Pixels.Bounds()
		bbd := ibd
		bbd.Min.X += bsz
		bbd.Min.Y += bsz
		bbd.Max.X -= bsz
		bbd.Max.Y -= bsz
		draw.Draw(sp.Pixels, bbd, &image.Uniform{color.Black}, image.ZP, draw.Src)
		sp.Bg = image.NewRGBA(ibd)
		draw.Draw(sp.Bg, ibd, &image.Uniform{color.White}, image.ZP, draw.Src)
	}
	ActiveSprites[spi] = sp
	return sp
}

// SetSpritePos activates and sets sprite position
func SetSpritePos(spi Sprites, pos image.Point, win *gi.Window, recv ki.Ki, fun ki.RecvFunc) *gi.Sprite {
	sp := Sprite(spi, win)
	sz := SpriteSize()
	if spi == SizeDnL || spi == SizeUpL || spi == SizeLfC {
		pos.X -= sz
	}
	if spi == SizeUpL || spi == SizeUpM || spi == SizeUpR {
		pos.Y -= sz
	}
	sp.Geom.Pos = pos
	sp.ConnectEvent(recv, oswin.MouseEvent, fun)
	sp.ConnectEvent(recv, oswin.MouseDragEvent, fun)
	win.ActivateSprite(sp.Name)
	return sp
}

// InactivateSprites inactivates all of our sprites
func InactivateSprites(win *gi.Window) {
	for spi := Sprites(0); spi < SpritesN; spi++ {
		sp := Sprite(spi, win)
		win.InactivateSprite(sp.Name)
	}
}
