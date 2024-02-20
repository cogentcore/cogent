// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vector

import (
	"cogentcore.org/core/events"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/giv"
	"cogentcore.org/core/gti"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keyfun"
	"cogentcore.org/core/laser"
	"cogentcore.org/core/styles"
)

//////////////////////////////////////////////////////////////////////////////////////
//  SplitsView

// SplitsView opens a view of a splits table
func SplitsView(pt *Splits) {
	if gi.ActivateExistingMainWindow(pt) {
		return
	}
	d := gi.NewBody().SetTitle("Available Splitter Settings: Can duplicate an existing (using Ctxt Menu) as starting point for new one").SetData(pt)
	tv := giv.NewTableView(d).SetSlice(pt)
	AvailSplitsChanged = false
	tv.OnChange(func(e events.Event) {
		AvailSplitsChanged = true
	})

	d.AddAppBar(func(tb *gi.Toolbar) {
		giv.NewFuncButton(tb, pt.SavePrefs).SetText("Save to prefs").
			SetIcon(icons.Save).SetKey(keyfun.Save).
			StyleFirst(func(s *styles.Style) { s.SetEnabled(AvailSplitsChanged && pt == &StdSplits) })
		oj := giv.NewFuncButton(tb, pt.Open).SetText("Open").SetIcon(icons.Open).SetKey(keyfun.Open)
		oj.Args[0].SetTag("ext", ".toml")
		sj := giv.NewFuncButton(tb, pt.Save).SetText("Save As").SetIcon(icons.SaveAs).SetKey(keyfun.SaveAs)
		sj.Args[0].SetTag("ext", ".toml")
		tb.AddOverflowMenu(func(m *gi.Scene) {
			giv.NewFuncButton(m, pt.OpenSettings).SetIcon(icons.Open).SetKey(keyfun.OpenAlt1)
		})
	})
	d.NewWindow().Run()
}

////////////////////////////////////////////////////////////////////////////////////////
//  SplitValue

// Value registers SplitValue as the viewer of SplitName
func (kn SplitName) Value() giv.Value {
	return &SplitValue{}
}

// SplitValue presents an action for displaying an SplitName and selecting
type SplitValue struct {
	giv.ValueBase
}

func (vv *SplitValue) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.ButtonType
	return vv.WidgetTyp
}

func (vv *SplitValue) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	bt := vv.Widget.(*gi.Button)
	txt := laser.ToString(vv.Value.Interface())
	if txt == "" {
		txt = "(none)"
	}
	bt.SetText(txt)
}

func (vv *SplitValue) ConfigWidget(w gi.Widget) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfigWidget(w)
	bt := vv.Widget.(*gi.Button)
	bt.SetType(gi.ButtonTonal)
	bt.Config()
	bt.OnClick(func(e events.Event) {
		if !vv.IsReadOnly() {
			vv.OpenDialog(bt, nil)
		}
	})
	vv.UpdateWidget()
}

func (vv *SplitValue) HasDialog() bool { return true }

func (vv *SplitValue) OpenDialog(ctx gi.Widget, fun func()) {
	cur := laser.ToString(vv.Value.Interface())
	m := gi.NewMenuFromStrings(AvailSplitNames, cur, func(idx int) {
		nm := AvailSplitNames[idx]
		vv.SetValue(nm)
		vv.UpdateWidget()
		if fun != nil {
			fun()
		}
	})
	gi.NewMenuStage(m, ctx, ctx.ContextMenuPos(nil)).Run()
}
