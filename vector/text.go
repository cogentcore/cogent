// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vector

import (
	"cogentcore.org/core/gi"
	"cogentcore.org/core/giv"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/svg"
	"cogentcore.org/core/units"
)

// TextStyle is text styling info -- using StructView to do text editor
type TextStyle struct {

	// current text to edit
	Text string

	// font family
	Font gi.FontName `xml:"font-family"`

	// font size
	Size units.Value `xml:"font-size"`

	// prop: font-style = style -- normal, italic, etc
	Style styles.FontStyles `xml:"font-style" inherit:"true"`

	// prop: font-weight = weight: normal, bold, etc
	Weight styles.FontWeights `xml:"font-weight" inherit:"true"`

	// prop: font-stretch = font stretch / condense options
	Stretch styles.FontStretch `xml:"font-stretch" inherit:"true"`

	// prop: font-variant = normal or small caps
	Variant styles.FontVariants `xml:"font-variant" inherit:"true"`

	// prop: text-decoration = underline, line-through, etc -- not inherited
	Deco styles.TextDecorations `xml:"text-decoration"`

	// prop: baseline-shift = super / sub script -- not inherited
	Shift styles.BaselineShifts `xml:"baseline-shift"`

	// prop: text-align (inherited) = how to align text, horizontally. This *only* applies to the text within its containing element, and is typically relevant only for multi-line text: for single-line text, if element does not have a specified size that is different from the text size, then this has *no effect*.
	Align styles.Aligns `xml:"text-align" inherit:"true"`

	// font value view for font toolbar
	FontVal giv.FontValue `view:"-"`

	// the parent vectorview
	VectorView *VectorView `copier:"-" json:"-" xml:"-" view:"-"`
}

func (ts *TextStyle) Update() {
	// this is called automatically when edited
	if ts.VectorView != nil {
		ts.VectorView.SetTextProps(ts.TextProps())
		ts.VectorView.SetText(ts.Text)
	}
}

func (ts *TextStyle) Defaults() {
	ts.Text = ""
	ts.Font = "Arial"
	ts.Size.Dp(16)
	ts.Style = 0
	ts.Weight = 0
	ts.Stretch = 0
	ts.Variant = 0
	ts.Deco = 0
	ts.Shift = 0
	ts.Align = 0

	// ts.SetFromFontStyle(&Prefs.TextStyle.FontStyle)
}

// SetFromFontStyle sets from standard styles.Font style
func (ts *TextStyle) SetFromFontStyle(fs *styles.Font) {
	ts.Font = gi.FontName(fs.Family)
	ts.Size = fs.Size
	ts.Weight = fs.Weight
	ts.Stretch = fs.Stretch
	ts.Variant = fs.Variant
	ts.Deco = fs.Decoration
	ts.Shift = fs.Shift
}

// SetFromNode sets text style info from given svg.Text node
func (ts *TextStyle) SetFromNode(txt *svg.Text) {
	ts.Defaults()                            // always start fresh
	if txt.Text == "" && txt.HasChildren() { // todo: multi-line text..
		tspan := txt.Kids[0].(*svg.Text)
		ts.Text = tspan.Text
	}
	// ts.SetFromFontStyle(&txt.Paint.FontStyle)
	ts.Align = txt.Paint.TextStyle.Align
}

// SetTextPropsNode sets the text properties of given Text node
func (gv *VectorView) SetTextPropsNode(sii svg.Node, tps map[string]string) {
	if gp, isgp := sii.(*svg.Group); isgp {
		for _, kid := range gp.Kids {
			gv.SetTextPropsNode(kid.(svg.Node), tps)
		}
		return
	}
	_, istxt := sii.(*svg.Text)
	if !istxt {
		return
	}
	g := sii.AsNodeBase()
	for k, v := range tps {
		if v == "" {
			g.DeleteProp(k)
		} else {
			g.SetProp(k, v)
		}
	}
}

// SetTextProps sets the text properties of selected Text nodes
func (gv *VectorView) SetTextProps(tps map[string]string) {
	es := &gv.EditState
	sv := gv.SVG()
	sv.UndoSave("SetTextProps", "")
	// sv.SetFullReRender()
	for itm := range es.Selected {
		gv.SetTextPropsNode(itm.(svg.Node), tps)
	}
	sv.NeedsRender()
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
		tps["text-decoration"] = ts.Deco.String()
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
func (gv *VectorView) SetTextNode(sii svg.Node, txt string) bool {
	if sii.HasChildren() {
		for _, kid := range *sii.Children() {
			if gv.SetTextNode(kid.(svg.Node), txt) {
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
func (gv *VectorView) SetText(txt string) {
	es := &gv.EditState
	if len(es.Selected) != 1 { // only if exactly one selected
		return
	}
	sv := gv.SVG()
	sv.UndoSave("SetText", "")
	// sv.SetFullReRender()
	for itm := range es.Selected {
		if gv.SetTextNode(itm.(svg.Node), txt) {
			break // only set first..
		}
	}
	sv.UpdateView(true) // needs full update
	gv.ChangeMade()
}

///////////////////////////////////////////////////////////////////////
// Toolbar

func (gv *VectorView) TextToolbar() *gi.Toolbar {
	tbs := gv.ModalToolbarStack()
	tb := tbs.ChildByName("text-tb", 2).(*gi.Toolbar)
	return tb
}

// ConfigTextToolbar configures the text modal toolbar
func (gv *VectorView) ConfigTextToolbar() {
	tb := gv.TextToolbar()
	if tb.HasChildren() {
		return
	}
	es := &gv.EditState
	ts := &es.Text
	ts.VectorView = gv

	txt := gi.NewTextField(tb, "text")
	txt.Tooltip = "current text string"
	txt.SetText(ts.Text)
	// txt.SetProp("width", units.NewCh(50))
	// txt.TextFieldSig.Connect(gv.This(), func(recv, send ki.Ki, sig int64, data any) {
	// 	if sig == int64(gi.TextFieldDone) {
	// 		ts.Text = txt.Text()
	// 		ts.Update()
	// 	}
	// })

	// ki.InitNode(&ts.FontVal)
	// ts.FontVal.SetSoloValue(reflect.ValueOf(&ts.Font))
	// fw := tb.AddNewChild(ts.FontVal.WidgetType(), "font").(gi.Node2D)
	// ts.FontVal.ConfigWidget(fw)

	// fsz := gi.NewSpinner(tb, "size")
	// fsz.SetValue(ts.Size.Val)
	// fsz.SpinnerSig.Connect(gv.This(), func(recv, send ki.Ki, sig int64, data any) {
	// 	ts.Size.Val = fsz.Value
	// 	ts.Update()
	// })

	// fzu := gi.NewChooser(tb, "size-units")
	// fzu.ItemsFromEnum(units.KiT_Units, true, 0)
	// fzu.SetCurIndex(int(ts.Size.Un))
	// fzu.ComboSig.Connect(gv.This(), func(recv, send ki.Ki, sig int64, data any) {
	// 	ts.Size.Un = units.Units(fzu.CurIndex)
	// 	ts.Update()
	// })

}

// UpdateTextToolbar updates the select toolbar based on current selection
func (gv *VectorView) UpdateTextToolbar() {
	tb := gv.TextToolbar()
	es := &gv.EditState
	ts := &es.Text

	txt := tb.ChildByName("text", 0).(*gi.TextField)
	txt.SetText(ts.Text)

	// fw := tb.ChildByName("font", 0).(gi.Node2D)
	// ts.FontVal.UpdateWidget()

	// fsz := tb.ChildByName("size", 0).(*gi.Spinner)
	// fsz.SetValue(ts.Size.Val)

	// fzu := tb.ChildByName("size-units", 0).(*gi.Chooser)
	// fzu.SetCurrentIndex(int(ts.Size.Un))
}
