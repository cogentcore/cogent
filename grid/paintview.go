// Copyright (c) 2021, The Grid Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grid

import (
	"fmt"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/girl"
	"github.com/goki/gi/giv"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// PaintView provides editing of basic Stroke and Fill painting parameters
// for selected items
type PaintView struct {
	gi.Layout
	GridView *GridView `copy:"-" json:"-" xml:"-" view:"-" desc:"the parent gridview"`
}

var KiT_PaintView = kit.Types.AddType(&PaintView{}, PaintViewProps)

func (pv *PaintView) Config(gv *GridView) {
	if pv.HasChildren() {
		return
	}
	updt := pv.UpdateStart()
	pv.GridView = gv
	pv.Lay = gi.LayoutVert
	pv.SetProp("spacing", gi.StdDialogVSpaceUnits)

	spl := gi.AddNewLayout(pv, "stroke-lab", gi.LayoutHoriz)
	gi.AddNewLabel(spl, "stroke-pnt-lab", "<b>Stroke Paint:  </b>")
	spt := gi.AddNewCheckBox(spl, "stroke-on")
	spt.SetText("On")
	spt.Tooltip = "whether to paint stroke"
	spt.SetChecked(true)

	sc := giv.AddNewColorView(pv, "stroke-clr")
	sc.Config()
	sc.SetColor(pv.GridView.Prefs.Style.StrokeStyle.Color.Color)
	sc.ViewSig.Connect(pv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if pv.IsStrokeOn() {
			pv.GridView.SetStrokeColor(pv.StrokeProp(), false) // not manip
		}
	})
	sc.ManipSig.Connect(pv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if pv.IsStrokeOn() {
			pv.GridView.SetStrokeColor(pv.StrokeProp(), true) // manip
		}
	})

	spt.ButtonSig.Connect(pv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.ButtonToggled) {
			pv.GridView.SetStrokeOn(pv.StrokeProp())
		}
	})

	wr := gi.AddNewLayout(pv, "stroke-width", gi.LayoutHoriz)
	gi.AddNewLabel(wr, "width-lab", "Width:  ")
	wsb := gi.AddNewSpinBox(wr, "width")
	wsb.SetProp("min", 0)
	wsb.SetProp("step", 0.05)
	wsb.SetValue(pv.GridView.Prefs.Style.StrokeStyle.Width.Val)
	wsb.SpinBoxSig.Connect(pv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		pv.GridView.SetStrokeWidth(pv.StrokeWidthProp(), false)
	})
	// todo: units from drawing units?

	gi.AddNewSeparator(pv, "fill-sep", true)

	fpl := gi.AddNewLayout(pv, "fill-lab", gi.LayoutHoriz)
	gi.AddNewLabel(fpl, "fill-pnt-lab", "<b>Fill Paint:  </b>")
	fpt := gi.AddNewCheckBox(fpl, "fill-on")
	fpt.SetText("On")
	fpt.Tooltip = "whether to fill paint"
	fpt.SetChecked(true)

	fc := giv.AddNewColorView(pv, "fill-clr")
	fc.Config()
	fc.SetColor(pv.GridView.Prefs.Style.FillStyle.Color.Color)
	fc.ViewSig.Connect(pv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if pv.IsFillOn() {
			pv.GridView.SetFillColor(pv.FillProp(), false)
		}
	})
	fc.ManipSig.Connect(pv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if pv.IsFillOn() {
			pv.GridView.SetFillColor(pv.FillProp(), true) // manip
		}
	})

	fpt.ButtonSig.Connect(pv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.ButtonToggled) {
			pv.GridView.SetFillOn(pv.FillProp())
		}
	})
	pv.UpdateEnd(updt)
}

// IsStrokeOn returns true if Stroke checkbox is on
func (pv *PaintView) IsStrokeOn() bool {
	spt := pv.ChildByName("stroke-lab", 0).ChildByName("stroke-on", 1).(*gi.CheckBox)
	return spt.IsChecked()
}

// StrokeProp returns the stroke property string according to current settings
func (pv *PaintView) StrokeProp() string {
	if !pv.IsStrokeOn() {
		return "none"
	}
	sc := pv.ChildByName("stroke-clr", 1).(*giv.ColorView)
	return sc.Color.HexString()
}

// StrokeWidthProp returns stroke-width property
func (pv *PaintView) StrokeWidthProp() string {
	wsb := pv.ChildByName("stroke-width", 2).ChildByName("width", 1).(*gi.SpinBox)
	return fmt.Sprintf("%gpx", wsb.Value) // todo units
}

// IsFillOn returns true if Fill checkbox is on
func (pv *PaintView) IsFillOn() bool {
	fpt := pv.ChildByName("fill-lab", 0).ChildByName("fill-on", 1).(*gi.CheckBox)
	return fpt.IsChecked()
}

// FillProp returns the fill property string according to current settings
func (pv *PaintView) FillProp() string {
	if !pv.IsFillOn() {
		return "none"
	}
	sc := pv.ChildByName("fill-clr", 1).(*giv.ColorView)
	return sc.Color.HexString()
}

// Update updates the current settings based on the values in the given Paint
func (pv *PaintView) Update(pnt *girl.Paint) {
	spt := pv.ChildByName("stroke-lab", 0).ChildByName("stroke-on", 1).(*gi.CheckBox)
	spt.SetChecked(!pnt.StrokeStyle.Color.IsNil())
	spt.UpdateSig()

	if !pnt.StrokeStyle.Color.IsNil() {
		sc := pv.ChildByName("stroke-clr", 1).(*giv.ColorView)
		sc.SetColor(pnt.StrokeStyle.Color.Color)
	}

	wsb := pv.ChildByName("stroke-width", 2).ChildByName("width", 1).(*gi.SpinBox)
	wsb.SetValue(pnt.StrokeStyle.Width.Val)
	// todo: units

	fpt := pv.ChildByName("fill-lab", 0).ChildByName("fill-on", 1).(*gi.CheckBox)
	fpt.SetChecked(!pnt.FillStyle.Color.IsNil())
	fpt.UpdateSig()

	if !pnt.FillStyle.Color.IsNil() {
		fc := pv.ChildByName("fill-clr", 1).(*giv.ColorView)
		fc.SetColor(pnt.FillStyle.Color.Color)
	}
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
