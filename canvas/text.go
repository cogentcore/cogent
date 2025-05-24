// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package canvas

import (
	"image/color"
	"maps"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/svg"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/text"
	"cogentcore.org/core/tree"
)

// TextStyle is text styling info -- using Form to do text editor
type TextStyle struct {
	// current text to edit
	String string `label:"Text"`

	// Color is the fill color to render the text in.
	Color color.RGBA

	// Stroke is the stroke color to render the text in: this is typically
	// not set for standard text rendering, but can be used to render an outline
	// around the text glyphs.
	Stroke color.RGBA

	// Opacity is the overall opacity multiplier on the text, between 0-1.
	Opacity float32 `min:"0" max:"1", step:"0.1"`

	// Font styling properties.
	Font styles.Font `display:"add-fields"`

	// Text styling properties.
	Text styles.Text `display:"add-fields"`

	// the parent [Canvas]
	Canvas *Canvas `copier:"-" json:"-" xml:"-" display:"-"`
}

func (ts *TextStyle) Defaults() {
	ts.Color = colors.Black
	ts.Stroke = colors.Transparent
	ts.Opacity = 1
	ts.Font.Defaults()
	ts.Text.Defaults()
	ts.Font.Size.Px(32)
}

// Update updates any selected text from updated settings in TextStyle.
func (ts *TextStyle) Update() {
	ts.SetTextProperties()
	ts.Canvas.SetText(ts.String)
}

// SetTextProperties sets the text properties of selected Text nodes.
func (ts *TextStyle) SetTextProperties() {
	ts.Canvas.SetTextProperties(ts.TextProperties())
}

// SetTextProperties sets the text properties of selected Text nodes.
func (cv *Canvas) SetTextProperties(tps map[string]any) {
	cv.setPropsOnSelected("SetTextProperties", "", func(nd svg.Node) {
		nb := nd.AsNodeBase()
		nb.Properties = maps.Clone(tps)
	})
}

// TextProperties returns non-default text properties to set
func (ts *TextStyle) TextProperties() map[string]any {
	sty := rich.NewStyle()
	tsty := text.NewStyle()
	clr := colors.Uniform(ts.Color)
	styles.SetRichText(sty, tsty, &ts.Font, &ts.Text, clr, ts.Opacity)
	if ts.Stroke != colors.Transparent {
		sty.SetStrokeColor(ts.Stroke)
	}
	tps := map[string]any{}
	tsty.ToProperties(sty, tps)
	return tps
}

// SetTextNode sets the text of given Text node
func (cv *Canvas) SetTextNode(sii svg.Node, txt string) bool {
	return true
}

// SetText sets the text of selected Text node
func (cv *Canvas) SetText(txt string) {
	if txt == "" {
		return
	}
	es := &cv.EditState
	if len(es.Selected) != 1 { // only if exactly one selected
		return
	}
	tn, ok := es.SelectedList(false)[0].(*svg.Text)
	if !ok {
		return
	}
	if tn.IsParText() {
		tn, ok = tn.Child(0).(*svg.Text)
	}
	if !ok {
		return
	}
	sv := cv.SVG
	sv.UndoSave("SetText", "")
	tn.Text = txt
	cv.ChangeMade()
	sv.UpdateView()
}

// SetFromNode sets text style info from given svg.Text node
func (ts *TextStyle) SetFromNode(txt *svg.Text) {
	ts.Defaults() // always start fresh
	if txt.IsParText() {
		tspan := txt.Children[0].(*svg.Text)
		ts.String = tspan.Text
	} else {
		ts.String = txt.Text
	}
	styles.SetFromRichText(&txt.Paint.Font, &txt.Paint.Text, &ts.Font, &ts.Text)
	ts.Color = colors.AsRGBA(colors.ToUniform(txt.Paint.Fill.Color)) // this is where it goes
	ts.Opacity = txt.Paint.Opacity
	if txt.Paint.HasStroke() {
		ts.Stroke = colors.AsRGBA(colors.ToUniform(txt.Paint.Stroke.Color))
	}

	ts.Canvas.UpdateText()
}

//////// Toolbar

func (cv *Canvas) MakeTextToolbar(p *tree.Plan) {
	es := &cv.EditState
	ts := &es.Text
	ts.Canvas = cv

	tree.Add(p, func(w *core.TextField) {
		core.Bind(&ts.String, w)
		w.SetTooltip("Current text")
		w.OnChange(func(e events.Event) {
			cv.SetText(ts.String)
			ts.Canvas.UpdateText()
		})
	})

	tree.Add(p, func(w *core.Spinner) {
		core.Bind(&ts.Font.Size.Value, w)
		w.Step = 1
		w.SetTooltip("Current text size")
		w.OnChange(func(e events.Event) {
			ts.SetTextProperties()
			ts.Canvas.UpdateText()
		})
	})

	tree.Add(p, func(w *core.Chooser) {
		core.Bind(&ts.Font.Size.Unit, w)
		w.SetTooltip("Current text size units")
		w.OnChange(func(e events.Event) {
			ts.SetTextProperties()
			ts.Canvas.UpdateText()
		})
	})

	// txt.SetProp("width", units.NewCh(50))
	// txt.TextFieldSig.Connect(gv.This, func(recv, send tree.Node, sig int64, data any) {
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
	// fsz.SpinnerSig.Connect(gv.This, func(recv, send tree.Node, sig int64, data any) {
	// 	ts.Size.Val = fsz.Value
	// 	ts.Update()
	// })

	// fzu := core.NewChooser(tb, "size-units")
	// fzu.ItemsFromEnum(units.KiT_Units, true, 0)
	// fzu.SetCurIndex(int(ts.Size.Un))
	// fzu.ComboSig.Connect(gv.This, func(recv, send tree.Node, sig int64, data any) {
	// 	ts.Size.Un = units.Units(fzu.CurIndex)
	// 	ts.Update()
	// })

}

// UpdateTextToolbar updates the select toolbar based on current selection
func (cv *Canvas) UpdateTextToolbar() {
	// fw := tb.ChildByName("font", 0).(core.Node2D)
	// ts.FontVal.UpdateWidget()

	// fsz := tb.ChildByName("size", 0).(*core.Spinner)
	// fsz.SetValue(ts.Size.Val)

	// fzu := tb.ChildByName("size-units", 0).(*core.Chooser)
	// fzu.SetCurrentIndex(int(ts.Size.Un))
}
