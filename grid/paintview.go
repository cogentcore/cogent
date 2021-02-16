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
	pstr := kit.ToString(g.Prop(prop))
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
	updt := pv.UpdateStart()
	defer pv.UpdateEnd(updt)

	pv.StrokeType, pv.StrokeStops = pv.DecodeType(g, &g.Pnt.StrokeStyle.Color, "stroke")
	pv.FillType, pv.FillStops = pv.DecodeType(g, &g.Pnt.FillStyle.Color, "fill")

	es := &pv.GridView.EditState
	grl := &es.Gradients

	spt := pv.ChildByName("stroke-lab", 0).ChildByName("stroke-type", 1).(*gi.ButtonBox)
	spt.SelectItem(int(pv.StrokeType))

	ss := pv.ChildByName("stroke-stack", 1).(*gi.Frame)
	ss.SetFullReRender()

	switch pv.StrokeType {
	case PaintSolid:
		ss.StackTop = 1
		sc := ss.ChildByName("stroke-clr", 1).(*giv.ColorView)
		sc.SetColor(g.Pnt.StrokeStyle.Color.Color)
	case PaintLinear, PaintRadial:
		ss.StackTop = 2
		sg := ss.ChildByName("stroke-grad", 1).(*giv.TableView)
		sg.SetSlice(grl)
		pv.SelectStrokeGrad()
	default:
		ss.StackTop = 0
	}

	wsb := pv.ChildByName("stroke-width", 2).ChildByName("width", 1).(*gi.SpinBox)
	wsb.SetValue(g.Pnt.StrokeStyle.Width.Val)
	// todo: units

	fpt := pv.ChildByName("fill-lab", 0).ChildByName("fill-type", 1).(*gi.ButtonBox)
	fpt.SelectItem(int(pv.FillType))

	fs := pv.ChildByName("fill-stack", 1).(*gi.Frame)
	fs.SetFullReRender()

	switch pv.FillType {
	case PaintSolid:
		fs.StackTop = 1
		fc := fs.ChildByName("fill-clr", 1).(*giv.ColorView)
		fc.SetColor(g.Pnt.FillStyle.Color.Color)
	case PaintLinear, PaintRadial:
		fs.StackTop = 2
		fg := fs.ChildByName("fill-grad", 1).(*giv.TableView)
		fg.SetSlice(grl)
		pv.SelectFillGrad()
	default:
		fs.StackTop = 0
	}
}

func (pv *PaintView) SelectStrokeGrad() {
	es := &pv.GridView.EditState
	grl := &es.Gradients
	ss := pv.ChildByName("stroke-stack", 1).(*gi.Frame)
	sg := ss.ChildByName("stroke-grad", 1).(*giv.TableView)
	sg.UnselectAllIdxs()
	for i, g := range *grl {
		if g.Name == pv.StrokeStops {
			sg.SelectIdx(i)
			break
		}
	}
}

func (pv *PaintView) SelectFillGrad() {
	es := &pv.GridView.EditState
	grl := &es.Gradients
	fs := pv.ChildByName("fill-stack", 1).(*gi.Frame)
	fg := fs.ChildByName("fill-grad", 1).(*giv.TableView)
	fg.UnselectAllIdxs()
	for i, g := range *grl {
		if g.Name == pv.FillStops {
			fg.SelectIdx(i)
			break
		}
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

	ss := gi.AddNewFrame(pv, "stroke-stack", gi.LayoutStacked)
	ss.StackTop = 1
	ss.SetStretchMax()
	ss.SetReRenderAnchor()
	// ss.StackTopOnly = true

	gi.AddNewFrame(ss, "stroke-blank", gi.LayoutHoriz) // nothing

	spt.ButtonSig.Connect(pv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		prev := pv.StrokeType
		pv.StrokeType = PaintTypes(sig)
		updt := ss.UpdateStart()
		ss.SetFullReRender()
		sp := pv.StrokeProp()
		switch pv.StrokeType {
		case PaintOff, PaintInherit:
			ss.StackTop = 0
		case PaintSolid:
			ss.StackTop = 1
		case PaintLinear, PaintRadial:
			if pv.StrokeStops == "" {
				pv.StrokeStops = pv.GridView.DefaultGradient()
			}
			sp = pv.StrokeStops
			ss.StackTop = 2
			pv.SelectStrokeGrad()
		}
		ss.UpdateEnd(updt)
		pv.GridView.SetStroke(prev, pv.StrokeType, sp)
	})

	sc := giv.AddNewColorView(ss, "stroke-clr")
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

	sg := giv.AddNewTableView(ss, "stroke-grad")
	sg.SetProp("index", true)
	sg.SetProp("toolbar", true)
	sg.SelectedIdx = -1
	sg.SetSlice(&pv.GridView.EditState.Gradients)
	sg.WidgetSig.Connect(pv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.WidgetSelected) {
			svv, _ := send.(*giv.TableView)
			if svv.SelectedIdx >= 0 {
				pv.StrokeStops = pv.GridView.EditState.Gradients[svv.SelectedIdx].Name
				pv.GridView.SetStroke(pv.StrokeType, pv.StrokeType, pv.StrokeStops) // handles full updating
			}
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

	fs := gi.AddNewFrame(pv, "fill-stack", gi.LayoutStacked)
	fs.SetStretchMax()
	fs.StackTop = 1
	fs.SetReRenderAnchor()
	fs.StackTopOnly = true

	gi.AddNewFrame(fs, "fill-blank", gi.LayoutHoriz) // nothing

	fpt.ButtonSig.Connect(pv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		prev := pv.FillType
		pv.FillType = PaintTypes(sig)
		updt := fs.UpdateStart()
		fs.SetFullReRender()
		fp := pv.FillProp()
		switch pv.FillType {
		case PaintOff, PaintInherit:
			fs.StackTop = 0
		case PaintSolid:
			fs.StackTop = 1
		case PaintLinear, PaintRadial:
			if pv.FillStops == "" {
				pv.FillStops = pv.GridView.DefaultGradient()
			}
			fp = pv.FillStops
			fs.StackTop = 2
			pv.SelectFillGrad()
		}
		fs.UpdateEnd(updt)
		pv.GridView.SetFill(prev, pv.FillType, fp)
	})

	fc := giv.AddNewColorView(fs, "fill-clr")
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

	fg := giv.AddNewTableView(fs, "fill-grad")
	fg.SetProp("index", true)
	fg.SetProp("toolbar", true)
	fg.SelectedIdx = -1
	fg.SetSlice(&pv.GridView.EditState.Gradients)
	fg.WidgetSig.Connect(pv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.WidgetSelected) {
			svv, _ := send.(*giv.TableView)
			if svv.SelectedIdx >= 0 {
				pv.FillStops = pv.GridView.EditState.Gradients[svv.SelectedIdx].Name
				pv.GridView.SetFill(pv.FillType, pv.FillType, pv.FillStops) // this handles updating gradients etc to use stops
			}
		}
	})
	fg.SliceViewSig.Connect(pv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		// svv, _ := send.(*giv.TableView)
		if sig == int64(giv.SliceViewDeleted) { // not clear what we can do here
		} else {
			pv.GridView.UpdateGradients()
		}
	})
	fg.ViewSig.Connect(pv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		// svv, _ := send.(*giv.TableView)
		pv.GridView.UpdateGradients()
	})

	pv.UpdateEnd(updt)
}

// StrokeProp returns the stroke property string according to current settings
func (pv *PaintView) StrokeProp() string {
	ss := pv.ChildByName("stroke-stack", 1).(*gi.Frame)
	switch pv.StrokeType {
	case PaintOff:
		return "none"
	case PaintSolid:
		sc := ss.ChildByName("stroke-clr", 1).(*giv.ColorView)
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
	fs := pv.ChildByName("fill-stack", 1).(*gi.Frame)
	switch pv.FillType {
	case PaintOff:
		return "none"
	case PaintSolid:
		sc := fs.ChildByName("fill-clr", 1).(*giv.ColorView)
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
func (pv *PaintView) SetProps(sii svg.NodeSVG) {
	pv.GridView.SetStrokeNode(sii, pv.StrokeType, pv.StrokeType, pv.StrokeProp())
	if pv.IsStrokeOn() {
		sii.SetProp("stroke-width", pv.StrokeWidthProp())
	}
	pv.GridView.SetFillNode(sii, pv.FillType, pv.FillType, pv.FillProp())
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
