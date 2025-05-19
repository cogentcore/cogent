// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package canvas

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/svg"
	"cogentcore.org/core/tree"
)

// TextStyle is text styling info -- using Form to do text editor
type TextStyle struct {
	// current text to edit
	Text string

	// FontStyle styling properties.
	FontStyle styles.Font `new-window:"+"`

	// TextStyle styling properties.
	TextStyle styles.Text `new-window:"+"`

	// the parent [Canvas]
	Canvas *Canvas `copier:"-" json:"-" xml:"-" display:"-"`
}

func (ts *TextStyle) Update() {
	// this is called automatically when edited (TODO: not anymore)
	if ts.Canvas != nil {
		// ts.Canvas.SetTextProperties(ts.TextProperties())
		ts.Canvas.SetText(ts.Text)
	}
}

func (ts *TextStyle) Defaults() {
	ts.Text = ""
}

// SetFromFontStyle sets from standard styles.Font style
func (ts *TextStyle) SetFromFontStyle(fs *styles.Font) {
	ts.FontStyle = *fs
}

// SetFromNode sets text style info from given svg.Text node
func (ts *TextStyle) SetFromNode(txt *svg.Text) {
	ts.Defaults()                            // always start fresh
	if txt.Text == "" && txt.HasChildren() { // todo: multi-line text..
		tspan := txt.Children[0].(*svg.Text)
		ts.Text = tspan.Text
	}
	// ts.SetFromFontStyle(&txt.Paint.FontStyle)
	// ts.Align = txt.Paint.TextStyle.Align
}

// SetTextPropertiesNode sets the text properties of given Text node
func (gv *Canvas) SetTextPropertiesNode(sii svg.Node, tps map[string]string) {
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
func (gv *Canvas) SetTextProperties(tps map[string]string) {
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
// func (ts *TextStyle) TextProperties() map[string]string {
// tps := make(map[string]string)
// // tps["font-family"] = string(ts.Font)
// tps["font-size"] = ts.Size.String()
// if int(ts.Weight) != 0 {
// 	tps["font-weight"] = ts.Weight.String()
// } else {
// 	tps["font-weight"] = ""
// }
// if int(ts.Stretch) != 0 {
// 	tps["font-stretch"] = ts.Stretch.String()
// } else {
// 	tps["font-stretch"] = ""
// }
// if int(ts.Variant) != 0 {
// 	tps["font-variant"] = ts.Variant.String()
// } else {
// 	tps["font-variant"] = ""
// }
// if int(ts.Deco) != 0 {
// 	tps["text-decoration"] = ts.Deco.String()
// } else {
// 	tps["text-decoration"] = ""
// }
// if int(ts.Shift) != 0 {
// 	tps["baseline-shift"] = ts.Shift.String()
// } else {
// 	tps["baseline-shift"] = ""
// }
// tps["text-align"] = ts.Align.String()
// return tps
// }

// SetTextNode sets the text of given Text node
func (gv *Canvas) SetTextNode(sii svg.Node, txt string) bool {
	if sii.AsTree().HasChildren() {
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
func (gv *Canvas) SetText(txt string) {
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

func (vc *Canvas) MakeTextToolbar(p *tree.Plan) {
	es := &vc.EditState
	ts := &es.Text
	ts.Canvas = vc

	tree.Add(p, func(w *core.TextField) {
		core.Bind(&ts.Text, w)
		w.SetTooltip("Current text")
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
func (gv *Canvas) UpdateTextToolbar() {
	// fw := tb.ChildByName("font", 0).(core.Node2D)
	// ts.FontVal.UpdateWidget()

	// fsz := tb.ChildByName("size", 0).(*core.Spinner)
	// fsz.SetValue(ts.Size.Val)

	// fzu := tb.ChildByName("size-units", 0).(*core.Chooser)
	// fzu.SetCurrentIndex(int(ts.Size.Un))
}
