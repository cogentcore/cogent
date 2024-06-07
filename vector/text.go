// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vector

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/svg"
	"cogentcore.org/core/views"
)

// TextStyle is text styling info -- using StructView to do text editor
type TextStyle struct {

	// current text to edit
	Text string

	// font family
	Font core.FontName `xml:"font-family"`

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
	FontButton views.FontButton `view:"-"`

	// the parent vectorview
	VectorView *VectorView `copier:"-" json:"-" xml:"-" view:"-"`
}

func (ts *TextStyle) Update() {
	// this is called augtomatically when edited
	if ts.VectorView != nil {
		ts.VectorView.SetTextProperties(ts.TextProperties())
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

	// ts.SetFromFontStyle(&Settings.TextStyle.FontStyle)
}

// SetFromFontStyle sets from standard styles.Font style
func (ts *TextStyle) SetFromFontStyle(fs *styles.Font) {
	ts.Font = core.FontName(fs.Family)
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
		tspan := txt.Children[0].(*svg.Text)
		ts.Text = tspan.Text
	}
	// ts.SetFromFontStyle(&txt.Paint.FontStyle)
	ts.Align = txt.Paint.TextStyle.Align
}

// SetTextPropertiesNode sets the text properties of given Text node
func (gv *VectorView) SetTextPropertiesNode(sii svg.Node, tps map[string]string) {
	if gp, isgp := sii.(*svg.Group); isgp {
		for _, kid := range gp.Children {
			gv.SetTextPropertiesNode(kid.(svg.Node), tps)
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
			g.DeleteProperty(k)
		} else {
			g.SetProperty(k, v)
		}
	}
}

// SetTextProperties sets the text properties of selected Text nodes
func (gv *VectorView) SetTextProperties(tps map[string]string) {
	es := &gv.EditState
	sv := gv.SVG()
	sv.UndoSave("SetTextProperties", "")
	// sv.SetFullReRender()
	for itm := range es.Selected {
		gv.SetTextPropertiesNode(itm.(svg.Node), tps)
	}
	sv.NeedsRender()
	gv.ChangeMade()
}

// TextProperties returns non-default text properties to set
func (ts *TextStyle) TextProperties() map[string]string {
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
		for _, kid := range sii.AsTree().Children {
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

func (gv *VectorView) TextToolbar() *core.Toolbar {
	tbs := gv.ModalToolbarStack()
	tb := tbs.ChildByName("text-tb", 2).(*core.Toolbar)
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

	txt := core.NewTextField(tb)
	txt.SetName("text")
	txt.Tooltip = "current text string"
	txt.SetText(ts.Text)
	// txt.SetProp("width", units.NewCh(50))
	// txt.TextFieldSig.Connect(gv.This(), func(recv, send tree.Node, sig int64, data any) {
	// 	if sig == int64(core.TextFieldDone) {
	// 		ts.Text = txt.Text()
	// 		ts.Update()
	// 	}
	// })

	// tree.InitNode(&ts.FontVal)
	// ts.FontVal.SetSoloValue(reflect.ValueOf(&ts.Font))
	// fw := tb.AddNewChild(ts.FontVal.WidgetType(), "font").(core.Node2D)
	// ts.FontVal.Config(fw)

	// fsz := core.NewSpinner(tb, "size")
	// fsz.SetValue(ts.Size.Val)
	// fsz.SpinnerSig.Connect(gv.This(), func(recv, send tree.Node, sig int64, data any) {
	// 	ts.Size.Val = fsz.Value
	// 	ts.Update()
	// })

	// fzu := core.NewChooser(tb, "size-units")
	// fzu.ItemsFromEnum(units.KiT_Units, true, 0)
	// fzu.SetCurIndex(int(ts.Size.Un))
	// fzu.ComboSig.Connect(gv.This(), func(recv, send tree.Node, sig int64, data any) {
	// 	ts.Size.Un = units.Units(fzu.CurIndex)
	// 	ts.Update()
	// })

}

// UpdateTextToolbar updates the select toolbar based on current selection
func (gv *VectorView) UpdateTextToolbar() {
	tb := gv.TextToolbar()
	es := &gv.EditState
	ts := &es.Text

	txt := tb.ChildByName("text", 0).(*core.TextField)
	txt.SetText(ts.Text)

	// fw := tb.ChildByName("font", 0).(core.Node2D)
	// ts.FontVal.UpdateWidget()

	// fsz := tb.ChildByName("size", 0).(*core.Spinner)
	// fsz.SetValue(ts.Size.Val)

	// fzu := tb.ChildByName("size-units", 0).(*core.Chooser)
	// fzu.SetCurrentIndex(int(ts.Size.Un))
}
