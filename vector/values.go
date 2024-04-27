// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vector

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/reflectx"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/views"
)

//////////////////////////////////////////////////////////////////////////////////////
//  SplitsView

// SplitsView opens a view of a splits table
func SplitsView(pt *Splits) {
	if core.ActivateExistingMainWindow(pt) {
		return
	}
	d := core.NewBody().SetTitle("Available Splitter Settings: can duplicate an existing ß(using context menu) as starting point for new one").SetData(pt)
	tv := views.NewTableView(d).SetSlice(pt)
	AvailableSplitsChanged = false
	tv.OnChange(func(e events.Event) {
		AvailableSplitsChanged = true
	})

	d.AddAppBar(func(tb *core.Toolbar) {
		views.NewFuncButton(tb, pt.SaveSettings).SetText("Save to settings").
			SetIcon(icons.Save).SetKey(keymap.Save).
			StyleFirst(func(s *styles.Style) { s.SetEnabled(AvailableSplitsChanged && pt == &StandardSplits) })
		oj := views.NewFuncButton(tb, pt.Open).SetText("Open").SetIcon(icons.Open).SetKey(keymap.Open)
		oj.Args[0].SetTag("ext", ".toml")
		sj := views.NewFuncButton(tb, pt.Save).SetText("Save As").SetIcon(icons.SaveAs).SetKey(keymap.SaveAs)
		sj.Args[0].SetTag("ext", ".toml")
		tb.AddOverflowMenu(func(m *core.Scene) {
			views.NewFuncButton(m, pt.OpenSettings).SetIcon(icons.Open).SetKey(keymap.OpenAlt1)
		})
	})
	d.RunWindow()
}

// Value registers [SplitValue] as the [views.Value] for [SplitName].
func (sn SplitName) Value() views.Value {
	return &SplitValue{}
}

// SplitValue represents a [SplitName] value with a button.
type SplitValue struct {
	views.ValueBase[*core.Button]
}

func (v *SplitValue) Config() {
	v.Widget.SetType(core.ButtonTonal)
	views.ConfigDialogWidget(v, false)
}

func (v *SplitValue) Update() {
	txt := reflectx.ToString(v.Value.Interface())
	if txt == "" {
		txt = "(none)"
	}
	v.Widget.SetText(txt).Update()
}

func (v *SplitValue) OpenDialog(ctx core.Widget, fun func()) {
	cur := reflectx.ToString(v.Value.Interface())
	m := core.NewMenuFromStrings(AvailableSplitNames, cur, func(idx int) {
		nm := AvailableSplitNames[idx]
		v.SetValue(nm)
		v.Update()
		if fun != nil {
			fun()
		}
	})
	core.NewMenuStage(m, ctx, ctx.ContextMenuPos(nil)).Run()
}
