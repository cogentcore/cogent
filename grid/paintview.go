// Copyright (c) 2021, The Grid Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grid

import (
	"fmt"
	"strings"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/gist"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/svg"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// PaintView provides editing of basic Stroke and Fill painting parameters
// for selected items
type PaintView struct {
	gi.Layout
	StrokeType  PaintTypes `desc:"paint type for stroke"`
	StrokeStops string     `desc:"name of gradient with stops"`
	FillType    PaintTypes `desc:"paint type for fill"`
	FillStops   string     `desc:"name of gradient with stops"`
	GridView    *GridView  `copy:"-" json:"-" xml:"-" view:"-" desc:"the parent gridview"`
}

var KiT_PaintView = kit.Types.AddType(&PaintView{}, PaintViewProps)

// GradStopsName returns the stopsname for gradient from url
func (pv *PaintView) GradStopsName(g *svg.NodeBase, url string) string {
	gr := svg.GradientByName(g, url)
	if gr.StopsName != "" {
		return gr.StopsName
	}
	return gr.Nm
}

// DecodeType decodes the paint type from paint and props
// also returns the name of the gradient if using one.
func (pv *PaintView) DecodeType(g *svg.NodeBase, cs *gist.ColorSpec, prop string) (PaintTypes, string) {
	pstr := ""
	if p := g.Prop(prop); p != nil {
		pstr = p.(string)
	}
	switch {
	case pstr == "inherit":
		return PaintInherit, ""
	case pstr == "none" || cs.IsNil():
		return PaintOff, ""
	case strings.HasPrefix(pstr, "url(#linear") || (cs.Gradient != nil && !cs.Gradient.IsRadial):
		return PaintLinear, pv.GradStopsName(g, pstr)
	case strings.HasPrefix(pstr, "url(#radial") || (cs.Gradient != nil && cs.Gradient.IsRadial):
		return PaintRadial, pv.GradStopsName(g, pstr)
	}
	return PaintSolid, ""
}

// Update updates the current settings based on the values in the given Paint
func (pv *PaintView) Update(g *svg.NodeBase) {
	pv.StrokeType, pv.StrokeStops = pv.DecodeType(g, &g.Pnt.StrokeStyle.Color, "stroke")
	pv.FillType, pv.FillStops = pv.DecodeType(g, &g.Pnt.FillStyle.Color, "fill")

	spt := pv.ChildByName("stroke-lab", 0).ChildByName("stroke-type", 1).(*gi.ButtonBox)
	spt.SelectItem(int(pv.StrokeType))

	// todo: stack
	if pv.StrokeType == PaintSolid {
		sc := pv.ChildByName("stroke-clr", 1).(*giv.ColorView)
		sc.SetColor(g.Pnt.StrokeStyle.Color.Color)
	}

	wsb := pv.ChildByName("stroke-width", 2).ChildByName("width", 1).(*gi.SpinBox)
	wsb.SetValue(g.Pnt.StrokeStyle.Width.Val)
	// todo: units

	fpt := pv.ChildByName("fill-lab", 0).ChildByName("fill-type", 1).(*gi.ButtonBox)
	fpt.SelectItem(int(pv.FillType))

	// todo: stack
	if pv.FillType == PaintSolid {
		fc := pv.ChildByName("fill-clr", 1).(*giv.ColorView)
		fc.SetColor(g.Pnt.FillStyle.Color.Color)
	}
}

func (pv *PaintView) Config(gv *GridView) {
	if pv.HasChildren() {
		return
	}
	updt := pv.UpdateStart()
	pv.StrokeType = PaintSolid
	pv.FillType = PaintSolid

	pv.GridView = gv
	pv.Lay = gi.LayoutVert
	pv.SetProp("spacing", gi.StdDialogVSpaceUnits)

	spl := gi.AddNewLayout(pv, "stroke-lab", gi.LayoutHoriz)
	gi.AddNewLabel(spl, "stroke-pnt-lab", "<b>Stroke Paint:  </b>")

	spt := gi.AddNewButtonBox(spl, "stroke-type")
	spt.ItemsFromStringList(PaintTypeNames)
	spt.SelectItem(int(pv.StrokeType))
	spt.Mutex = true
	spt.ButtonSig.Connect(pv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		pv.StrokeType = PaintTypes(sig)
		pv.GridView.SetStroke(pv.StrokeType, pv.StrokeProp())
	})

	sc := giv.AddNewColorView(pv, "stroke-clr")
	sc.Config()
	sc.SetColor(pv.GridView.Prefs.Style.StrokeStyle.Color.Color)
	sc.ViewSig.Connect(pv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if pv.StrokeType == PaintSolid {
			pv.GridView.SetStrokeColor(pv.StrokeProp(), false) // not manip
		}
	})
	sc.ManipSig.Connect(pv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if pv.StrokeType == PaintSolid {
			pv.GridView.SetStrokeColor(pv.StrokeProp(), true) // manip
		}
	})

	wr := gi.AddNewLayout(pv, "stroke-width", gi.LayoutHoriz)
	gi.AddNewLabel(wr, "width-lab", "Width:  ")
	wsb := gi.AddNewSpinBox(wr, "width")
	wsb.SetProp("min", 0)
	wsb.SetProp("step", 0.05)
	wsb.SetValue(pv.GridView.Prefs.Style.StrokeStyle.Width.Val)
	wsb.SpinBoxSig.Connect(pv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if pv.IsStrokeOn() {
			pv.GridView.SetStrokeWidth(pv.StrokeWidthProp(), false)
		}
	})
	// todo: units from drawing units?

	gi.AddNewSeparator(pv, "fill-sep", true)

	fpl := gi.AddNewLayout(pv, "fill-lab", gi.LayoutHoriz)
	gi.AddNewLabel(fpl, "fill-pnt-lab", "<b>Fill Paint:  </b>")

	fpt := gi.AddNewButtonBox(fpl, "fill-type")
	fpt.ItemsFromStringList(PaintTypeNames)
	fpt.SelectItem(int(pv.FillType))
	fpt.Mutex = true
	fpt.ButtonSig.Connect(pv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		pv.FillType = PaintTypes(sig)
		pv.GridView.SetFill(pv.FillType, pv.FillProp())
	})

	fc := giv.AddNewColorView(pv, "fill-clr")
	fc.Config()
	fc.SetColor(pv.GridView.Prefs.Style.FillStyle.Color.Color)
	fc.ViewSig.Connect(pv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if pv.FillType == PaintSolid {
			pv.GridView.SetFillColor(pv.FillProp(), false)
		}
	})
	fc.ManipSig.Connect(pv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if pv.FillType == PaintSolid {
			pv.GridView.SetFillColor(pv.FillProp(), true) // manip
		}
	})

	pv.UpdateEnd(updt)
}

// StrokeProp returns the stroke property string according to current settings
func (pv *PaintView) StrokeProp() string {
	switch pv.StrokeType {
	case PaintOff:
		return "none"
	case PaintSolid:
		sc := pv.ChildByName("stroke-clr", 1).(*giv.ColorView)
		return sc.Color.HexString()
	case PaintLinear:
		return pv.StrokeStops
	case PaintRadial:
		return pv.StrokeStops
	case PaintInherit:
		return "inherit"
	}
	return "none"
}

// IsStrokeOn returns true if stroke is active
func (pv *PaintView) IsStrokeOn() bool {
	return pv.StrokeType >= PaintSolid && pv.StrokeType < PaintInherit
}

// StrokeWidthProp returns stroke-width property
func (pv *PaintView) StrokeWidthProp() string {
	wsb := pv.ChildByName("stroke-width", 2).ChildByName("width", 1).(*gi.SpinBox)
	return fmt.Sprintf("%gpx", wsb.Value) // todo units
}

// IsFillOn returns true if Fill is active
func (pv *PaintView) IsFillOn() bool {
	return pv.FillType >= PaintSolid && pv.FillType < PaintInherit
}

// FillProp returns the fill property string according to current settings
func (pv *PaintView) FillProp() string {
	switch pv.FillType {
	case PaintOff:
		return "none"
	case PaintSolid:
		sc := pv.ChildByName("fill-clr", 1).(*giv.ColorView)
		return sc.Color.HexString()
	case PaintLinear:
		return pv.FillStops
	case PaintRadial:
		return pv.FillStops
	case PaintInherit:
		return "inherit"
	}
	return "none"
}

// SetProps sets the props for given node according to current settings
func (pv *PaintView) SetProps(kn ki.Ki) {
	kn.SetProp("stroke", pv.StrokeProp())
	if pv.IsStrokeOn() {
		kn.SetProp("stroke-width", pv.StrokeWidthProp())
	}
	kn.SetProp("fill", pv.FillProp())
}

var PaintViewProps = ki.Props{
	"EnumType:Flag":    gi.KiT_VpFlags,
	"background-color": &gi.Prefs.Colors.Background,
	"color":            &gi.Prefs.Colors.Font,
	"max-width":        -1,
	"max-height":       -1,
}

type PaintTypes int

const (
	PaintOff PaintTypes = iota
	PaintSolid
	PaintLinear
	PaintRadial
	PaintInherit
	PaintTypesN
)

var PaintTypeNames = []string{"Off", "Solid", "Linear", "Radial", "Inherit"}

//go:generate stringer -type=PaintTypes

var KiT_PaintTypes = kit.Enums.AddEnumAltLower(PaintTypesN, kit.NotBitFlag, nil, "")

func (ev PaintTypes) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *PaintTypes) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }
