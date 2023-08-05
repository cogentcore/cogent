// Copyright (c) 2021, The Grid Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grid

import (
	"reflect"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/gist"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/svg"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// TextStyle is text styling info -- using StructView to do text editor
type TextStyle struct {

	// current text to edit
	Text string `desc:"current text to edit"`

	// font family
	Font gi.FontName `xml:"font-family" desc:"font family"`

	// font size
	Size units.Value `xml:"font-size" desc:"font size"`

	// prop: font-style = style -- normal, italic, etc
	Style gist.FontStyles `xml:"font-style" inherit:"true" desc:"prop: font-style = style -- normal, italic, etc"`

	// prop: font-weight = weight: normal, bold, etc
	Weight gist.FontWeights `xml:"font-weight" inherit:"true" desc:"prop: font-weight = weight: normal, bold, etc"`

	// prop: font-stretch = font stretch / condense options
	Stretch gist.FontStretch `xml:"font-stretch" inherit:"true" desc:"prop: font-stretch = font stretch / condense options"`

	// prop: font-variant = normal or small caps
	Variant gist.FontVariants `xml:"font-variant" inherit:"true" desc:"prop: font-variant = normal or small caps"`

	// prop: text-decoration = underline, line-through, etc -- not inherited
	Deco gist.TextDecorations `xml:"text-decoration" desc:"prop: text-decoration = underline, line-through, etc -- not inherited"`

	// prop: baseline-shift = super / sub script -- not inherited
	Shift gist.BaselineShifts `xml:"baseline-shift" desc:"prop: baseline-shift = super / sub script -- not inherited"`

	// prop: text-align (inherited) = how to align text, horizontally. This *only* applies to the text within its containing element, and is typically relevant only for multi-line text: for single-line text, if element does not have a specified size that is different from the text size, then this has *no effect*.
	Align gist.Align `xml:"text-align" inherit:"true" desc:"prop: text-align (inherited) = how to align text, horizontally. This *only* applies to the text within its containing element, and is typically relevant only for multi-line text: for single-line text, if element does not have a specified size that is different from the text size, then this has *no effect*."`

	// font value view for font toolbar
	FontVal giv.FontValueView `view:"-" desc:"font value view for font toolbar"`

	// the parent gridview
	GridView *GridView `copy:"-" json:"-" xml:"-" view:"-" desc:"the parent gridview"`
}

func (ts *TextStyle) Update() {
	// this is called automatically when edited
	if ts.GridView != nil {
		ts.GridView.SetTextProps(ts.TextProps())
		ts.GridView.SetText(ts.Text)
	}
}

func (ts *TextStyle) Defaults() {
	ts.Text = ""
	ts.Font = "Arial"
	ts.Size.SetPx(12)
	ts.Style = gist.FontStyles(0)
	ts.Weight = gist.FontWeights(0)
	ts.Stretch = gist.FontStretch(0)
	ts.Variant = gist.FontVariants(0)
	ts.Deco = gist.TextDecorations(0)
	ts.Shift = gist.BaselineShifts(0)
	ts.Align = gist.AlignLeft

	ts.SetFromFontStyle(&Prefs.TextStyle.FontStyle)
}

// SetFromFontStyle sets from standard gist.Font style
func (ts *TextStyle) SetFromFontStyle(fs *gist.Font) {
	ts.Font = gi.FontName(fs.Family)
	ts.Size = fs.Size
	ts.Weight = fs.Weight
	ts.Stretch = fs.Stretch
	ts.Variant = fs.Variant
	ts.Deco = fs.Deco
	ts.Shift = fs.Shift
}

// SetFromNode sets text style info from given svg.Text node
func (ts *TextStyle) SetFromNode(txt *svg.Text) {
	ts.Defaults()                            // always start fresh
	if txt.Text == "" && txt.HasChildren() { // todo: multi-line text..
		tspan := txt.Kids[0].(*svg.Text)
		ts.Text = tspan.Text
	}
	ts.SetFromFontStyle(&txt.Pnt.FontStyle)
	ts.Align = txt.Pnt.TextStyle.Align
}

// SetTextPropsNode sets the text properties of given Text node
func (gv *GridView) SetTextPropsNode(sii svg.NodeSVG, tps map[string]string) {
	if gp, isgp := sii.(*svg.Group); isgp {
		for _, kid := range gp.Kids {
			gv.SetTextPropsNode(kid.(svg.NodeSVG), tps)
		}
		return
	}
	_, istxt := sii.(*svg.Text)
	if !istxt {
		return
	}
	g := sii.AsSVGNode()
	for k, v := range tps {
		if v == "" {
			g.DeleteProp(k)
		} else {
			g.SetProp(k, v)
		}
	}
}

// SetTextProps sets the text properties of selected Text nodes
func (gv *GridView) SetTextProps(tps map[string]string) {
	es := &gv.EditState
	sv := gv.SVG()
	sv.UndoSave("SetTextProps", "")
	updt := sv.UpdateStart()
	sv.SetFullReRender()
	for itm := range es.Selected {
		gv.SetTextPropsNode(itm.(svg.NodeSVG), tps)
	}
	sv.UpdateEnd(updt)
	gv.ChangeMade()
}

// TextProps returns non-default text properties to set
func (ts *TextStyle) TextProps() map[string]string {
	tps := make(map[string]string)
	tps["font-family"] = string(ts.Font)
	tps["font-size"] = ts.Size.String()
	if int(ts.Weight) != 0 {
		tps["font-weight"] = ts.Weight.String()
	} else {
		tps["font-weight"] = ""
	}
	if int(ts.Stretch) != 0 {
		tps["font-stretch"] = ts.Stretch.String()
	} else {
		tps["font-stretch"] = ""
	}
	if int(ts.Variant) != 0 {
		tps["font-variant"] = ts.Variant.String()
	} else {
		tps["font-variant"] = ""
	}
	if int(ts.Deco) != 0 {
		tps["text-decoration"] = kit.BitFlagsToString(int64(ts.Deco), ts.Deco)
	} else {
		tps["text-decoration"] = ""
	}
	if int(ts.Shift) != 0 {
		tps["baseline-shift"] = ts.Shift.String()
	} else {
		tps["baseline-shift"] = ""
	}
	tps["text-align"] = ts.Align.String()
	return tps
}

// SetTextNode sets the text of given Text node
func (gv *GridView) SetTextNode(sii svg.NodeSVG, txt string) bool {
	if sii.HasChildren() {
		for _, kid := range *sii.Children() {
			if gv.SetTextNode(kid.(svg.NodeSVG), txt) {
				return true
			}
		}
		return false
	}
	tn, istxt := sii.(*svg.Text)
	if !istxt {
		return false
	}
	tn.Text = txt // todo: actually need to deal with multi-line here..
	return true
}

// SetText sets the text of selected Text node
func (gv *GridView) SetText(txt string) {
	es := &gv.EditState
	if len(es.Selected) != 1 { // only if exactly one selected
		return
	}
	sv := gv.SVG()
	sv.UndoSave("SetText", "")
	sv.SetFullReRender()
	for itm := range es.Selected {
		if gv.SetTextNode(itm.(svg.NodeSVG), txt) {
			break // only set first..
		}
	}
	sv.UpdateView(true) // needs full update
	gv.ChangeMade()
}

///////////////////////////////////////////////////////////////////////
// Toolbar

func (gv *GridView) TextToolbar() *gi.ToolBar {
	tbs := gv.ModalToolbarStack()
	tb := tbs.ChildByName("text-tb", 2).(*gi.ToolBar)
	return tb
}

// ConfigTextToolbar configures the text modal toolbar
func (gv *GridView) ConfigTextToolbar() {
	tb := gv.TextToolbar()
	if tb.HasChildren() {
		return
	}
	tb.SetStretchMaxWidth()
	es := &gv.EditState
	ts := &es.Text
	ts.GridView = gv

	txt := gi.AddNewTextField(tb, "text")
	txt.Tooltip = "current text string"
	txt.SetText(ts.Text)
	txt.SetProp("width", units.NewCh(50))
	txt.TextFieldSig.Connect(gv.This(), func(recv, send ki.Ki, sig int64, data any) {
		if sig == int64(gi.TextFieldDone) {
			ts.Text = txt.Text()
			ts.Update()
		}
	})

	ki.InitNode(&ts.FontVal)
	ts.FontVal.SetSoloValue(reflect.ValueOf(&ts.Font))
	fw := tb.AddNewChild(ts.FontVal.WidgetType(), "font").(gi.Node2D)
	ts.FontVal.ConfigWidget(fw)

	fsz := gi.AddNewSpinBox(tb, "size")
	fsz.SetValue(ts.Size.Val)
	fsz.SpinBoxSig.Connect(gv.This(), func(recv, send ki.Ki, sig int64, data any) {
		ts.Size.Val = fsz.Value
		ts.Update()
	})

	fzu := gi.AddNewComboBox(tb, "size-units")
	fzu.ItemsFromEnum(units.KiT_Units, true, 0)
	fzu.SetCurIndex(int(ts.Size.Un))
	fzu.ComboSig.Connect(gv.This(), func(recv, send ki.Ki, sig int64, data any) {
		ts.Size.Un = units.Units(fzu.CurIndex)
		ts.Update()
	})

}

// UpdateTextToolbar updates the select toolbar based on current selection
func (gv *GridView) UpdateTextToolbar() {
	tb := gv.TextToolbar()
	tb.UpdateActions()
	es := &gv.EditState
	ts := &es.Text

	txt := tb.ChildByName("text", 0).(*gi.TextField)
	txt.SetText(ts.Text)

	// fw := tb.ChildByName("font", 0).(gi.Node2D)
	ts.FontVal.UpdateWidget()

	fsz := tb.ChildByName("size", 0).(*gi.SpinBox)
	fsz.SetValue(ts.Size.Val)

	fzu := tb.ChildByName("size-units", 0).(*gi.ComboBox)
	fzu.SetCurIndex(int(ts.Size.Un))
}
