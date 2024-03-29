// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vector

import (
	"cogentcore.org/core/events"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/giv"
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
	d := gi.NewBody().SetTitle("Available Splitter Settings: can duplicate an existing ß(using context menu) as starting point for new one").SetData(pt)
	tv := giv.NewTableView(d).SetSlice(pt)
	AvailSplitsChanged = false
	tv.OnChange(func(e events.Event) {
		AvailSplitsChanged = true
	})

	d.AddAppBar(func(tb *gi.Toolbar) {
		giv.NewFuncButton(tb, pt.SavePrefs).SetText("Save to settings").
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

// Value registers [SplitValue] as the [giv.Value] for [SplitName].
func (sn SplitName) Value() giv.Value {
	return &SplitValue{}
}

// SplitValue represents a [SplitName] value with a button.
type SplitValue struct {
	giv.ValueBase[*gi.Button]
}

func (v *SplitValue) Config() {
	v.Widget.SetType(gi.ButtonTonal)
	giv.ConfigDialogWidget(v, false)
}

func (v *SplitValue) Update() {
	txt := laser.ToString(v.Value.Interface())
	if txt == "" {
		txt = "(none)"
	}
	v.Widget.SetText(txt).Update()
}

func (v *SplitValue) OpenDialog(ctx gi.Widget, fun func()) {
	cur := laser.ToString(v.Value.Interface())
	m := gi.NewMenuFromStrings(AvailSplitNames, cur, func(idx int) {
		nm := AvailSplitNames[idx]
		v.SetValue(nm)
		v.Update()
		if fun != nil {
			fun()
		}
	})
	gi.NewMenuStage(m, ctx, ctx.ContextMenuPos(nil)).Run()
}
