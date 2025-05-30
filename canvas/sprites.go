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
type Sprites int32 //enums:enum -transform kebab -trim-prefix Sp

const (
	// SpNone is used for subtypes
	SpNone Sprites = iota

	// SpReshapeBBox is a reshape bbox -- the overall active selection BBox
	// for active manipulation
	SpReshapeBBox

	// SpSelBBox is a selection bounding box -- display only
	SpSelBBox

	// SpNodePoint is a main coordinate point for path node
	SpNodePoint

	// SpNodeCtrl is a control coordinate point for path node
	SpNodeCtrl

	// SpRubberBand is the draggable selection box
	SpRubberBand

	// SpAlignMatch is an alignment match (n of these),
	SpAlignMatch

	// SpLineAdd is preview of new line to add
	SpLineAdd

	// below are subtypes:

	// Sprite bounding boxes are set as a "bbox" property on sprites
	SpUpL
	SpUpC
	SpUpR
	SpDnL
	SpDnC
	SpDnR
	SpLfM
	SpRtM

	// Node points
	SpMoveTo
	SpLineTo
	SpCubeTo
	SpQuadTo
	SpArcTo
	SpClose

	SpStart
	SpEnd
	SpQuad1
	SpCube1
	SpCube2
)

// SpriteName returns the unique name of the sprite based
// on main type, subtype (e.g., bbox) if relevant, and index if relevant
func SpriteName(typ, subtyp Sprites, idx int) string {
	nm := typ.String()
	switch typ {
	case SpReshapeBBox:
		nm += "-" + subtyp.String()
	case SpSelBBox:
		nm += fmt.Sprintf("-%d-%s", idx, subtyp.String())
	case SpNodePoint:
		nm += fmt.Sprintf("-%d", idx)
	case SpNodeCtrl:
		nm += fmt.Sprintf("-%d-%s", idx, subtyp.String())
	case SpRubberBand:
		nm += "-" + subtyp.String()
	case SpAlignMatch:
		nm += fmt.Sprintf("-%d", idx)
	}
	return nm
}

// SetSpriteProperties sets sprite properties
func SetSpriteProperties(sp *core.Sprite, typ, subtyp Sprites, idx int) {
	sp.InitProperties()
	sp.Active = true
	sp.Name = SpriteName(typ, subtyp, idx)
	sp.Properties["grid-type"] = typ
	sp.Properties["grid-sub"] = subtyp
	sp.Properties["grid-idx"] = idx
}

// SpriteProperties reads the sprite properties -- returns SpNone if
// not one of our sprites.
func SpriteProperties(sp *core.Sprite) (typ, subtyp Sprites, idx int) {
	if sp.Properties == nil {
		return
	}
	typi, has := sp.Properties["grid-type"]
	if !has {
		typ = SpNone
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
func (sv *SVG) Sprite(typ, subtyp Sprites, idx int, trgsz image.Point, init func(sp *core.Sprite)) *core.Sprite {
	sprites := sv.SpritesNolock()
	es := sv.EditState()
	spnm := SpriteName(typ, subtyp, idx)
	var sp *core.Sprite
	ok := false
	if typ == SpReshapeBBox {
		sp, ok = sprites.Final.AtTry(spnm)
	} else {
		sp, ok = sprites.SpriteByNameLocked(spnm)
	}
	if ok {
		sp.Active = true
		sprites.SetModifiedLocked()
		return sp
	}
	sp = core.NewSprite(spnm, nil)
	SetSpriteProperties(sp, typ, subtyp, idx)
	final := false
	first := false
	switch typ {
	case SpReshapeBBox:
		final = true
		sp.Draw = sv.DrawSpriteReshape(sp, subtyp)
	case SpSelBBox:
		sp.Draw = sv.DrawSpriteSelect(sp, subtyp)
	case SpNodePoint:
		sp.Draw = sv.DrawSpriteNodePoint(es, sp, subtyp, idx)
	case SpNodeCtrl:
		first = true
		sp.Draw = sv.DrawSpriteNodeCtrl(es, sp, subtyp, idx, trgsz)
	case SpRubberBand:
		sp.Draw = sv.DrawRubberBand(sp, trgsz)
	case SpAlignMatch:
		sp.Draw = sv.DrawAlignMatch(sp, trgsz)
	case SpLineAdd:
		sp.Draw = sv.DrawLineAdd(sp, trgsz)
	}
	if init != nil {
		init(sp)
	}
	switch {
	case first:
		sprites.First.Add(sp.Name, sp)
	case final:
		sprites.Final.Add(sp.Name, sp)
	default:
		sprites.AddLocked(sp)
	}
	return sp
}

// SetSpritePos sets sprite position, taking into account relative offsets
func SetSpritePos(sp *core.Sprite, x, y int) {
	typ, subtyp, _ := SpriteProperties(sp)
	pos := image.Point{x, y}
	switch {
	case typ == SpNodePoint || typ == SpNodeCtrl:
		spbb, _ := HandleSpriteSize(1, image.Point{})
		pos.X -= spbb.Dx() / 2
		pos.Y -= spbb.Dy() / 2
	case subtyp >= SpUpL && subtyp <= SpRtM: // Reshape, Sel BBox
		sc := float32(1)
		if typ == SpSelBBox {
			sc = .8
		}
		spbb, _ := HandleSpriteSize(sc, image.Point{})
		if subtyp == SpDnL || subtyp == SpUpL || subtyp == SpLfM {
			pos.X -= spbb.Dx()
		}
		if subtyp == SpUpL || subtyp == SpUpC || subtyp == SpUpR {
			pos.Y -= spbb.Dy()
		}
	}
	sp.SetPos(pos)
}

// InactivateSprites inactivates sprites of given type; must be locked.
func InactivateSprites(sprites *core.Sprites, typ Sprites) {
	sprites.Do(func(sl core.SpriteList) {
		for _, sp := range sl.Values {
			if sp == nil {
				continue
			}
			st, _, _ := SpriteProperties(sp)
			if st == typ {
				sp.Active = false
			}
		}

	})
	sprites.SetModifiedLocked()
}

////////  Sprite rendering

const (
	HandleSpriteScale = 12
	HandleSizeMin     = 12
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
	rdraw = math32.Box2{Min: fp.AddScalar(bsz), Max: fp.AddScalar(sz).SubScalar(bsz)}
	return
}

// DrawSpriteReshape returns sprite Draw function for reshape points
func (sv *SVG) DrawSpriteReshape(sp *core.Sprite, bbtyp Sprites) func(pc *paint.Painter) {
	return func(pc *paint.Painter) {
		bb, rdraw := HandleSpriteSize(1, sp.EventBBox.Min)
		sp.EventBBox = bb
		if sv.Geom.ContentBBox.Intersect(bb) == (image.Rectangle{}) {
			return
		}
		pc.BlitBox(rdraw.Min, rdraw.Size(), colors.Scheme.Primary.Base)
	}
}

// DrawSpriteSelect renders a Select sprite handle -- smaller
func (sv *SVG) DrawSpriteSelect(sp *core.Sprite, bbtyp Sprites) func(pc *paint.Painter) {
	return func(pc *paint.Painter) {
		bb, rdraw := HandleSpriteSize(.8, sp.EventBBox.Min)
		sp.EventBBox = bb
		if sv.Geom.ContentBBox.Intersect(bb) == (image.Rectangle{}) {
			return
		}
		pc.BlitBox(math32.FromPoint(bb.Min), math32.FromPoint(bb.Size()), colors.Scheme.Surface)
		pc.BlitBox(rdraw.Min, rdraw.Size(), colors.Scheme.OnSurface)
	}
}

// DrawSpriteNodePoint renders a NodePoint sprite handle
func (sv *SVG) DrawSpriteNodePoint(es *EditState, sp *core.Sprite, bbtyp Sprites, idx int) func(pc *paint.Painter) {
	return func(pc *paint.Painter) {
		bbi, _ := HandleSpriteSize(1, sp.EventBBox.Min)
		bb := math32.B2FromRect(bbi)
		ctr := bb.Center()
		sp.EventBBox = bbi
		if sv.Geom.ContentBBox.Intersect(bbi) == (image.Rectangle{}) {
			return
		}
		pc.BlitBox(math32.FromPoint(bbi.Min), math32.FromPoint(bbi.Size()), colors.Scheme.Surface)
		switch {
		case es.NodeIsSelected(idx):
			pc.Fill.Color = colors.Scheme.Primary.Base
		case es.NodeHover == idx:
			pc.Fill.Color = colors.Scheme.Select.Container
		default:
			pc.Fill.Color = nil
		}
		pc.Stroke.Color = colors.Scheme.OnSurface
		pc.Stroke.Width.Dp(1)
		pc.Polygon(math32.Vec2(bb.Min.X, ctr.Y), math32.Vec2(ctr.X, bb.Min.Y), math32.Vec2(bb.Max.X, ctr.Y), math32.Vec2(ctr.X, bb.Max.Y))
		pc.Draw()
	}
}

// DrawSpriteNodeCtrl renders a NodeControl sprite handle
func (sv *SVG) DrawSpriteNodeCtrl(es *EditState, sp *core.Sprite, subtyp Sprites, idx int, nodepos image.Point) func(pc *paint.Painter) {
	sp.Properties["nodePoint"] = nodepos
	return func(pc *paint.Painter) {
		bbi, _ := HandleSpriteSize(1, sp.EventBBox.Min)
		if sv.Geom.ContentBBox.Intersect(bbi) == (image.Rectangle{}) {
			return
		}

		bb := math32.B2FromRect(bbi)
		ctr := bb.Center()
		sz := bb.Size()
		ept := sp.Properties["nodePoint"].(image.Point)

		pc.Stroke.Color = colors.Scheme.OnSurface
		pc.Stroke.Width.Dp(1)
		pc.Line(float32(ept.X), float32(ept.Y), ctr.X, ctr.Y)
		pc.Draw()

		sp.EventBBox = bbi
		pc.BlitBox(math32.FromPoint(bbi.Min), math32.FromPoint(bbi.Size()), colors.Scheme.Surface)

		switch {
		case idx == es.CtrlDragIndex && subtyp == es.CtrlDrag:
			pc.Fill.Color = colors.Scheme.Primary.Base
		case es.CtrlHover == idx && subtyp == es.CtrlHoverType:
			pc.Fill.Color = colors.Scheme.Select.Container
		case subtyp == SpCube2:
			pc.Fill.Color = colors.Scheme.Warn.Container
		default:
			pc.Fill.Color = nil
		}
		pc.Circle(ctr.X, ctr.Y, 0.5*sz.X)
		pc.Draw()
	}
}

const (
	SpriteLineBorderWidth = 4
	SpriteLineWidth       = 2
)

// DrawRubberBand renders the rubber-band box.
func (sv *SVG) DrawRubberBand(sp *core.Sprite, trgsz image.Point) func(pc *paint.Painter) {
	sp.Properties["size"] = trgsz
	return func(pc *paint.Painter) {
		trgsz := sp.Properties["size"].(image.Point)
		sp.EventBBox.Max = sp.EventBBox.Min.Add(trgsz)
		bb := math32.B2FromRect(sp.EventBBox)
		fsz := math32.FromPoint(trgsz)
		pc.Fill.Color = nil
		pc.Stroke.Dashes = []float32{4, 4}
		pc.Stroke.Width.Dp(SpriteLineBorderWidth)
		pc.Stroke.Color = colors.Scheme.Surface
		pc.Rectangle(bb.Min.X, bb.Min.Y, fsz.X, fsz.Y)
		pc.Draw()
		pc.Stroke.Width.Dp(SpriteLineWidth)
		pc.Stroke.Color = colors.Scheme.Primary.Base
		pc.Rectangle(bb.Min.X, bb.Min.Y, fsz.X, fsz.Y)
		pc.Draw()
		pc.Stroke.Dashes = nil
	}
}

// DrawAlignMatch renders an alignment line
func (sv *SVG) DrawAlignMatch(sp *core.Sprite, trgsz image.Point) func(pc *paint.Painter) {
	sp.Properties["size"] = trgsz
	return func(pc *paint.Painter) {
		trgsz := sp.Properties["size"].(image.Point)
		sp.EventBBox.Max = sp.EventBBox.Min // no events
		bbi := image.Rectangle{Min: sp.EventBBox.Min, Max: sp.EventBBox.Min.Add(trgsz)}
		bb := math32.B2FromRect(bbi)
		pc.Fill.Color = nil
		pc.Stroke.Dashes = nil
		pc.Stroke.Width.Dp(SpriteLineBorderWidth)
		pc.Stroke.Color = colors.Scheme.Surface
		pc.Line(bb.Min.X, bb.Min.Y, bb.Max.X, bb.Max.Y)
		pc.Draw()
		pc.Stroke.Width.Dp(SpriteLineWidth)
		pc.Stroke.Color = colors.Scheme.Success.Container
		pc.Line(bb.Min.X, bb.Min.Y, bb.Max.X, bb.Max.Y)
		pc.Draw()
	}
}

// DrawLineAdd where new line would go.
func (sv *SVG) DrawLineAdd(sp *core.Sprite, trgsz image.Point) func(pc *paint.Painter) {
	sp.Properties["lastPoint"] = trgsz
	return func(pc *paint.Painter) {
		trgsz := sp.Properties["lastPoint"].(image.Point)
		sp.EventBBox.Max = sp.EventBBox.Min // no events
		bbi := image.Rectangle{Min: sp.EventBBox.Min, Max: trgsz}
		bb := math32.B2FromRect(bbi)
		pc.Fill.Color = nil
		pc.Stroke.Dashes = nil
		pc.Stroke.Width.Dp(SpriteLineBorderWidth)
		pc.Stroke.Color = colors.Scheme.Surface
		pc.Line(bb.Min.X, bb.Min.Y, bb.Max.X, bb.Max.Y)
		pc.Draw()
		pc.Stroke.Width.Dp(SpriteLineWidth)
		pc.Stroke.Color = colors.Scheme.Error.Base
		pc.Line(bb.Min.X, bb.Min.Y, bb.Max.X, bb.Max.Y)
		pc.Draw()
	}
}
