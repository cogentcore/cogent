// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package canvas

import (
	"fmt"
	"image"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
)

// note: all sprite functions assume overall sprites are locked.
// which keeps everything consistent under async rendering.

// Sprites are the type of sprite
type Sprites int32 //enums:enum

const (
	// SpUnk is an unknown sprite type
	SpUnk Sprites = iota

	// SpReshapeBBox is a reshape bbox -- the overall active selection BBox
	// for active manipulation
	SpReshapeBBox

	// SpSelBBox is a selection bounding box -- display only
	SpSelBBox

	// SpNodePoint is a main coordinate point for path node
	SpNodePoint

	// SpNodeCtrl is a control coordinate point for path node
	SpNodeCtrl

	// SpRubberBand is the draggable sel box
	// subtyp = UpC, LfM, RtM, DnC for sides
	SpRubberBand

	// SpAlignMatch is an alignment match (n of these),
	// subtyp is actually BBoxPoints so we just hack cast that
	SpAlignMatch

	// below are subtypes:

	// Sprite bounding boxes are set as a "bbox" property on sprites
	SpBBoxUpL
	SpBBoxUpC
	SpBBoxUpR
	SpBBoxDnL
	SpBBoxDnC
	SpBBoxDnR
	SpBBoxLfM
	SpBBoxRtM

	// todo: add nodectrl subtypes
)

// SpriteNames are name strings to use for naming sprites
var SpriteNames = map[Sprites]string{
	SpBBoxUpL: "up-l",
	SpBBoxUpC: "up-c",
	SpBBoxUpR: "up-r",
	SpBBoxDnL: "dn-l",
	SpBBoxDnC: "dn-c",
	SpBBoxDnR: "dn-r",
	SpBBoxLfM: "lf-m",
	SpBBoxRtM: "rt-m",

	SpReshapeBBox: "reshape-bbox",

	SpSelBBox: "sel-bbox",

	SpNodePoint: "node-point",
	SpNodeCtrl:  "node-ctrl",

	SpRubberBand: "rubber-band",

	SpAlignMatch: "align-match",
}

// SpriteName returns the unique name of the sprite based
// on main type, subtype (e.g., bbox) if relevant, and index if relevant
func SpriteName(typ, subtyp Sprites, idx int) string {
	nm := SpriteNames[typ]
	switch typ {
	case SpReshapeBBox:
		nm += "-" + SpriteNames[subtyp]
	case SpSelBBox:
		nm += fmt.Sprintf("-%d-%s", idx, SpriteNames[subtyp])
	case SpNodePoint:
		nm += fmt.Sprintf("-%d", idx)
	case SpNodeCtrl: // todo: subtype
		nm += fmt.Sprintf("-%d", idx)
	case SpRubberBand:
		nm += "-" + SpriteNames[subtyp]
	case SpAlignMatch:
		nm += fmt.Sprintf("-%d", idx)
	}
	return nm
}

// SetSpriteProperties sets sprite properties
func SetSpriteProperties(sp *core.Sprite, typ, subtyp Sprites, idx int) {
	sp.InitProperties()
	sp.Name = SpriteName(typ, subtyp, idx)
	sp.Properties["grid-type"] = typ
	sp.Properties["grid-sub"] = subtyp
	sp.Properties["grid-idx"] = idx
}

// SpriteProperties reads the sprite properties -- returns SpUnk if
// not one of our sprites.
func SpriteProperties(sp *core.Sprite) (typ, subtyp Sprites, idx int) {
	typi, has := sp.Properties["grid-type"]
	if !has {
		typ = SpUnk
		return
	}
	typ = typi.(Sprites)
	subtyp = sp.Properties["grid-sub"].(Sprites)
	idx = sp.Properties["grid-idx"].(int)
	return
}

// SpriteByName returns the given sprite in the context of the given widget,
// returning nil, false if not yet made.
func SpriteByName(ctx core.Widget, typ, subtyp Sprites, idx int) (*core.Sprite, bool) {
	sprites := &ctx.AsWidget().Scene.Stage.Sprites
	spnm := SpriteName(typ, subtyp, idx)
	return sprites.SpriteByName(spnm)
}

// Sprite returns the given sprite in the context of the given widget,
// making it if not yet made. trgsz is the target size (e.g., for rubber
// band boxes).  Init function is called on new sprites.
func Sprite(sprites *core.Sprites, typ, subtyp Sprites, idx int, trgsz image.Point, init func(sp *core.Sprite)) *core.Sprite {
	spnm := SpriteName(typ, subtyp, idx)
	sp, ok := sprites.SpriteByNameLocked(spnm)
	if ok {
		sprites.ActivateSpriteLocked(sp.Name)
		return sp
	}
	sp = core.NewSprite(spnm, nil)
	SetSpriteProperties(sp, typ, subtyp, idx)
	switch typ {
	case SpReshapeBBox:
		sp.Draw = DrawSpriteReshape(sp, subtyp)
	case SpSelBBox:
		sp.Draw = DrawSpriteSelect(sp, subtyp)
	case SpNodePoint:
		sp.Draw = DrawSpriteNodePoint(sp, subtyp)
	case SpNodeCtrl:
		sp.Draw = DrawSpriteNodeCtrl(sp, subtyp)
	case SpRubberBand:
		switch subtyp {
		case SpBBoxUpC, SpBBoxDnC:
			DrawRubberBandHoriz(sp, trgsz)
		case SpBBoxLfM, SpBBoxRtM:
			DrawRubberBandVert(sp, trgsz)
		}
	case SpAlignMatch:
		switch {
		case trgsz.X > trgsz.Y:
			DrawAlignMatchHoriz(sp, trgsz)
		default:
			DrawAlignMatchVert(sp, trgsz)
		}
	}
	if init != nil {
		init(sp)
	}
	sprites.AddLocked(sp)
	return sp
}

// SetSpritePos sets sprite position, taking into account relative offsets
func SetSpritePos(sp *core.Sprite, x, y int) {
	typ, subtyp, _ := SpriteProperties(sp)
	pos := image.Point{x, y}
	switch {
	// case typ == SpRubberBand:
	// 	_, sz := LineSpriteSize()
	// 	switch subtyp {
	// 	case SpBBoxUpC:
	// 		pos.Y -= sz
	// 	case SpBBoxLfM:
	// 		pos.X -= sz
	// 	}
	// case typ == SpAlignMatch:
	// 	_, sz := LineSpriteSize()
	// 	bbtp := BBoxPoints(subtyp) // just hack it
	// 	switch bbtp {
	// 	case BBLeft:
	// 		pos.X -= sz
	// 	case BBCenter:
	// 		pos.X -= sz / 2
	// 	case BBTop:
	// 		pos.Y -= sz
	// 	case BBMiddle:
	// 		pos.Y -= sz / 2
	// 	}
	case typ == SpNodePoint || typ == SpNodeCtrl:
		spbb, _ := HandleSpriteSize(1, image.Point{})
		pos.X -= spbb.Dx() / 2
		pos.Y -= spbb.Dy() / 2
	case subtyp >= SpBBoxUpL && subtyp <= SpBBoxRtM: // Reshape, Sel BBox
		sc := float32(1)
		if typ == SpSelBBox {
			sc = .8
		}
		spbb, _ := HandleSpriteSize(sc, image.Point{})
		if subtyp == SpBBoxDnL || subtyp == SpBBoxUpL || subtyp == SpBBoxLfM {
			pos.X -= spbb.Dx()
		}
		if subtyp == SpBBoxUpL || subtyp == SpBBoxUpC || subtyp == SpBBoxUpR {
			pos.Y -= spbb.Dy()
		}
	}
	sp.SetPos(pos)
}

// InactivateSprites inactivates sprites of given type; must be locked.
func InactivateSprites(sprites *core.Sprites, typ Sprites) {
	nms := []string{}
	for _, spkv := range sprites.Order {
		sp := spkv.Value
		st, _, _ := SpriteProperties(sp)
		if st == typ {
			nms = append(nms, sp.Name)
		}
	}
	sprites.InactivateSpriteLocked(nms...)
}

////////  Sprite rendering

const (
	HandleSpriteScale = 12
	HandleSizeMin     = 4
	HandleBorderMin   = 2
)

// HandleSpriteSize returns the bounding box and rect draw coords
// for handle-type sprites.
func HandleSpriteSize(scale float32, pos image.Point) (bb image.Rectangle, rdraw math32.Box2) {
	sz := math32.Ceil(scale * core.AppearanceSettings.Zoom * HandleSpriteScale / 100)
	sz = max(sz, HandleSizeMin)
	bsz := max(sz/6, HandleBorderMin)
	bb = image.Rectangle{Min: pos, Max: pos.Add(image.Pt(int(sz), int(sz)))}
	fp := math32.FromPoint(pos).AddScalar(bsz)
	rdraw = math32.Box2{Min: fp, Max: fp.AddScalar(sz).SubScalar(bsz)}
	return
}

// DrawSpriteReshape returns sprite Draw function for reshape points
func DrawSpriteReshape(sp *core.Sprite, bbtyp Sprites) func(pc *paint.Painter) {
	return func(pc *paint.Painter) {
		bb, rdraw := HandleSpriteSize(1, sp.EventBBox.Min)
		sp.EventBBox = bb
		pc.BlitBox(rdraw.Min, rdraw.Size(), colors.Scheme.Primary.Base)
	}
}

// DrawSpriteSelect renders a Select sprite handle -- smaller
func DrawSpriteSelect(sp *core.Sprite, bbtyp Sprites) func(pc *paint.Painter) {
	return func(pc *paint.Painter) {
		bb, rdraw := HandleSpriteSize(.8, sp.EventBBox.Min)
		sp.EventBBox = bb
		pc.BlitBox(math32.FromPoint(bb.Min), math32.FromPoint(bb.Size()), colors.Scheme.Surface)
		pc.BlitBox(rdraw.Min, rdraw.Size(), colors.Scheme.OnSurface)
	}
}

// DrawSpriteNodePoint renders a NodePoint sprite handle
func DrawSpriteNodePoint(sp *core.Sprite, bbtyp Sprites) func(pc *paint.Painter) {
	return func(pc *paint.Painter) {
		bb, rdraw := HandleSpriteSize(1, sp.EventBBox.Min)
		sp.EventBBox = bb
		pc.BlitBox(math32.FromPoint(bb.Min), math32.FromPoint(bb.Size()), colors.Scheme.Surface)
		pc.BlitBox(rdraw.Min, rdraw.Size(), colors.Scheme.OnSurface)
	}
}

// DrawSpriteNodeCtrl renders a NodePoint sprite handle
func DrawSpriteNodeCtrl(sp *core.Sprite, subtyp Sprites) func(pc *paint.Painter) {
	return func(pc *paint.Painter) {
		bb, rdraw := HandleSpriteSize(1, sp.EventBBox.Min)
		sp.EventBBox = bb
		pc.BlitBox(math32.FromPoint(bb.Min), math32.FromPoint(bb.Size()), colors.Scheme.Surface)
		pc.BlitBox(rdraw.Min, rdraw.Size(), colors.Scheme.OnSurface)
	}
}

const (
	LineSpriteScale = 4
	LineSizeMin     = 3
	LineBorderMin   = 1
)

// LineSpriteSize returns the border size and overall size of line-type sprites
func LineSpriteSize() (int, int) {
	sz := int(math32.Ceil(core.AppearanceSettings.Zoom * LineSpriteScale / 100))
	sz = max(sz, LineSizeMin)
	bsz := max(sz/6, LineBorderMin)
	return bsz, sz
}

// DrawRubberBandHoriz renders a horizontal rubber band line
func DrawRubberBandHoriz(sp *core.Sprite, trgsz image.Point) {
	// bsz, sz := LineSpriteSize()
	// ssz := image.Point{trgsz.X, sz}
	// ibd := sp.Pixels.Bounds()
	// bbd := ibd
	// bbd.Min.Y += bsz
	// bbd.Max.Y -= bsz
	// for x := 0; x < ssz.X; x += sz * 2 {
	// 	bbd.Min.X = x
	// 	bbd.Max.X = x + sz
	// 	draw.Draw(sp.Pixels, bbd, colors.Scheme.Primary.Base, image.ZP, draw.Src)
	// }
}

// DrawRubberBandVert renders a vertical rubber band line
func DrawRubberBandVert(sp *core.Sprite, trgsz image.Point) {
	// bsz, sz := LineSpriteSize()
	// ssz := image.Point{sz, trgsz.Y}
	// if !sp.SetSize(ssz) { // already set
	// 	return
	// }
	// ibd := sp.Pixels.Bounds()
	// bbd := ibd
	// bbd.Min.X += bsz
	// bbd.Max.X -= bsz
	// for y := sz; y < ssz.Y; y += sz * 2 {
	// 	bbd.Min.Y = y
	// 	bbd.Max.Y = y + sz
	// 	draw.Draw(sp.Pixels, bbd, colors.Scheme.Primary.Base, image.ZP, draw.Src)
	// }
}

// DrawAlignMatchHoriz renders a horizontal alignment line
func DrawAlignMatchHoriz(sp *core.Sprite, trgsz image.Point) {
	// bsz, sz := LineSpriteSize()
	// ssz := image.Point{trgsz.X, sz}
	// if !sp.SetSize(ssz) { // already set
	// 	return
	// }
	// ibd := sp.Pixels.Bounds()
	// bbd := ibd
	// bbd.Min.Y += bsz
	// bbd.Max.Y -= bsz
	// clr := color.RGBA{0, 200, 200, 255}
	// draw.Draw(sp.Pixels, ibd, &image.Uniform{color.White}, image.ZP, draw.Src)
	// draw.Draw(sp.Pixels, bbd, &image.Uniform{clr}, image.ZP, draw.Src)
}

// DrawAlignMatchVert renders a vertical alignment line
func DrawAlignMatchVert(sp *core.Sprite, trgsz image.Point) {
	// bsz, sz := LineSpriteSize()
	// ssz := image.Point{sz, trgsz.Y}
	// if !sp.SetSize(ssz) { // already set
	// 	return
	// }
	// ibd := sp.Pixels.Bounds()
	// bbd := ibd
	// bbd.Min.X += bsz
	// bbd.Max.X -= bsz
	// clr := color.RGBA{0, 200, 200, 255}
	// draw.Draw(sp.Pixels, ibd, &image.Uniform{color.White}, image.ZP, draw.Src)
	// draw.Draw(sp.Pixels, bbd, &image.Uniform{clr}, image.ZP, draw.Src)
}
