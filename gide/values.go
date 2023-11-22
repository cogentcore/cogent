// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"strings"

	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/giv"
	"goki.dev/gi/v2/keyfun"
	"goki.dev/girl/styles"
	"goki.dev/goosi/events"
	"goki.dev/gti"
	"goki.dev/icons"
	"goki.dev/laser"
)

// KeyMapsView opens a view of a key maps table
func KeyMapsView(km *KeyMaps) {
	if gi.ActivateExistingMainWindow(km) {
		return
	}
	d := gi.NewBody().SetTitle("Available Key Maps: Duplicate an existing map (using Ctxt Menu) as starting point for creating a custom map")
	d.Sc.Data = km
	tv := giv.NewTableView(d).SetSlice(km)
	AvailKeyMapsChanged = false
	tv.OnChange(func(e events.Event) {
		AvailKeyMapsChanged = true
	})
	d.AddTopBar(func(pw gi.Widget) {
		tb := d.DefaultTopAppBar(pw)
		sp := giv.NewFuncButton(tb, km.SavePrefs).SetText("Save to preferences").SetIcon(icons.Save).SetKey(keyfun.Save)
		sp.SetUpdateFunc(func() {
			sp.SetEnabled(AvailKeyMapsChanged && km == &AvailKeyMaps)
		})
		oj := giv.NewFuncButton(tb, km.OpenJSON).SetText("Open").SetIcon(icons.Open).SetKey(keyfun.Open)
		oj.Args[0].SetTag("ext", ".json")
		sj := giv.NewFuncButton(tb, km.SaveJSON).SetText("Save As").SetIcon(icons.SaveAs).SetKey(keyfun.SaveAs)
		sj.Args[0].SetTag("ext", ".json")
		gi.NewSeparator(tb)
		vs := giv.NewFuncButton(tb, km.ViewStd).SetConfirm(true).SetText("View standard").SetIcon(icons.Visibility)
		vs.SetUpdateFunc(func() {
			vs.SetEnabledUpdt(km != &StdKeyMaps)
		})
		rs := giv.NewFuncButton(tb, km.RevertToStd).SetConfirm(true).SetText("Revert to standard").SetIcon(icons.DeviceReset)
		rs.SetUpdateFunc(func() {
			rs.SetEnabledUpdt(km != &StdKeyMaps)
		})
		tb.AddOverflowMenu(func(m *gi.Scene) {
			giv.NewFuncButton(m, km.OpenPrefs).SetIcon(icons.Open).SetKey(keyfun.OpenAlt1)
		})
	})
	d.NewWindow().Run()
}

//////////////////////////////////////////////////////////////////////////////////////
//  PrefsView

// PrefsView opens a view of user preferences,
// returns structview if new (nil if recycled)
func PrefsView(pf *Preferences) *giv.StructView {
	if gi.ActivateExistingMainWindow(pf) {
		return nil
	}
	d := gi.NewBody().SetTitle("Gide Preferences")
	tv := giv.NewStructView(d).SetStruct(pf)
	tv.OnChange(func(e events.Event) {
		pf.Changed = true
	})
	d.Sc.Data = pf

	d.AddTopBar(func(pw gi.Widget) {
		tb := d.DefaultTopAppBar(pw)
		giv.NewFuncButton(tb, pf.Apply).SetIcon(icons.Done)
		sp := giv.NewFuncButton(tb, pf.Save).SetText("Save to prefs").SetIcon(icons.Save).SetKey(keyfun.Save)
		sp.SetUpdateFunc(func() {
			sp.SetEnabled(pf.Changed)
		})
		giv.NewFuncButton(tb, pf.VersionInfo).SetShowReturn(true).SetIcon(icons.Info)
		giv.NewFuncButton(tb, pf.EditKeyMaps).SetIcon(icons.Keyboard)
		giv.NewFuncButton(tb, pf.EditLangOpts).SetIcon(icons.Subtitles)
		giv.NewFuncButton(tb, pf.EditCmds).SetIcon(icons.KeyboardCommandKey)
		giv.NewFuncButton(tb, pf.EditSplits).SetIcon(icons.VerticalSplit)
		giv.NewFuncButton(tb, pf.EditRegisters).SetIcon(icons.Variables)

		tb.AddOverflowMenu(func(m *gi.Scene) {
			giv.NewFuncButton(m, pf.Open).SetText("Open prefs").SetIcon(icons.Open).SetKey(keyfun.OpenAlt1)
		})
	})

	d.NewWindow().Run()
	return tv
}

//////////////////////////////////////////////////////////////////////////////////////
//  ProjPrefsView

// ProjPrefsView opens a view of project preferences (settings),
// returns structview if not already open
func ProjPrefsView(pf *ProjPrefs) *giv.StructView {
	if gi.ActivateExistingMainWindow(pf) {
		return nil
	}
	d := gi.NewBody().SetTitle("Project preferences are saved in the project .gide file, along with other current state (open directories, splitter settings, etc) -- do Save Project to save.")
	d.Sc.Data = pf
	tv := giv.NewStructView(d).SetStruct(pf)
	tv.OnChange(func(e events.Event) {
		pf.Changed = true
	})
	d.NewWindow().Run()
	return tv
}

////////////////////////////////////////////////////////////////////////////////////////
//  KeyMapValue

// Value registers KeyMapValue as the viewer of KeyMapName
func (kn KeyMapName) Value() giv.Value {
	return &KeyMapValue{}
}

// KeyMapValue presents an action for displaying an KeyMapName and selecting
// from KeyMapChooserDialog
type KeyMapValue struct {
	giv.ValueBase
}

func (vv *KeyMapValue) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.ButtonType
	return vv.WidgetTyp
}

func (vv *KeyMapValue) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	bt := vv.Widget.(*gi.Button)
	txt := laser.ToString(vv.Value.Interface())
	if txt == "" {
		txt = "(none -- click to set)"
	}
	bt.SetText(txt)
}

func (vv *KeyMapValue) ConfigWidget(w gi.Widget, sc *gi.Scene) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfigWidget(w)
	bt := vv.Widget.(*gi.Button)
	bt.SetType(gi.ButtonTonal)
	bt.Config(sc)
	bt.OnClick(func(e events.Event) {
		vv.OpenDialog(bt)
	})
	vv.UpdateWidget()
}

func (vv *KeyMapValue) HasDialog() bool {
	return true
}

func (vv *KeyMapValue) OpenDialog(ctx gi.Widget) {
	if vv.IsReadOnly() {
		return
	}
	si := 0
	cur := laser.ToString(vv.Value.Interface())
	_, curRow, _ := AvailKeyMaps.MapByName(KeyMapName(cur))
	d := gi.NewBody().AddTitle("Select a key map").AddText(vv.Doc())
	giv.NewTableView(d).SetSlice(&AvailKeyMaps).SetSelIdx(curRow).BindSelectDialog(d.Sc, &si)
	d.AddBottomBar(func(pw gi.Widget) {
		d.AddCancel(pw)
		d.AddOk(pw).OnClick(func(e events.Event) {
			if si >= 0 {
				km := AvailKeyMaps[si]
				vv.SetValue(km.Name)
				vv.UpdateWidget()
			}
		})
	})
	d.NewFullDialog(ctx).Run()
}

//////////////////////////////////////////////////////////////////////////////////////
//  LangsView

// LangsView opens a view of a languages options map
func LangsView(pt *Langs) {
	if gi.ActivateExistingMainWindow(pt) {
		return
	}
	d := gi.NewBody().SetTitle("Available Language Opts: Add or modify entries to customize options for language / file types")
	tv := giv.NewMapView(d).SetMap(pt)
	d.Sc.Data = pt
	AvailLangsChanged = false
	tv.OnChange(func(e events.Event) {
		AvailLangsChanged = true
	})

	d.AddTopBar(func(pw gi.Widget) {
		tb := d.DefaultTopAppBar(pw)
		sp := giv.NewFuncButton(tb, pt.SavePrefs).SetText("Save to preferences").SetIcon(icons.Save).SetKey(keyfun.Save)
		sp.SetUpdateFunc(func() {
			sp.SetEnabled(AvailLangsChanged && pt == &AvailLangs)
		})
		oj := giv.NewFuncButton(tb, pt.OpenJSON).SetText("Open").SetIcon(icons.Open).SetKey(keyfun.Open)
		oj.Args[0].SetTag("ext", ".json")
		sj := giv.NewFuncButton(tb, pt.SaveJSON).SetText("Save As").SetIcon(icons.SaveAs).SetKey(keyfun.SaveAs)
		sj.Args[0].SetTag("ext", ".json")
		gi.NewSeparator(tb)
		vs := giv.NewFuncButton(tb, pt.ViewStd).SetConfirm(true).SetText("View standard").SetIcon(icons.Visibility)
		vs.SetUpdateFunc(func() {
			vs.SetEnabledUpdt(pt != &StdLangs)
		})
		rs := giv.NewFuncButton(tb, pt.RevertToStd).SetConfirm(true).SetText("Revert to standard").SetIcon(icons.DeviceReset)
		rs.SetUpdateFunc(func() {
			rs.SetEnabledUpdt(pt != &StdLangs)
		})
		tb.AddOverflowMenu(func(m *gi.Scene) {
			giv.NewFuncButton(m, pt.OpenPrefs).SetIcon(icons.Open).SetKey(keyfun.OpenAlt1)
		})
	})
	d.NewWindow().Run()
}

//////////////////////////////////////////////////////////////////////////////////////
//  CmdsView

// CmdsView opens a view of a commands table
func CmdsView(pt *Commands) {
	if gi.ActivateExistingMainWindow(pt) {
		return
	}
	d := gi.NewBody().SetTitle("Gide Commands")
	tv := giv.NewTableView(d).SetSlice(pt)
	d.Sc.Data = pt
	CustomCmdsChanged = false
	tv.OnChange(func(e events.Event) {
		CustomCmdsChanged = true
	})
	d.AddTopBar(func(pw gi.Widget) {
		tb := d.DefaultTopAppBar(pw)
		sp := giv.NewFuncButton(tb, pt.SavePrefs).SetText("Save to prefs").SetIcon(icons.Save).SetKey(keyfun.Save)
		sp.SetUpdateFunc(func() {
			sp.SetEnabled(CustomCmdsChanged && pt == &CustomCmds)
		})
		oj := giv.NewFuncButton(tb, pt.OpenJSON).SetText("Open").SetIcon(icons.Open).SetKey(keyfun.Open)
		oj.Args[0].SetTag("ext", ".json")
		sj := giv.NewFuncButton(tb, pt.SaveJSON).SetText("Save As").SetIcon(icons.SaveAs).SetKey(keyfun.SaveAs)
		sj.Args[0].SetTag("ext", ".json")
		gi.NewSeparator(tb)
		vs := giv.NewFuncButton(tb, pt.ViewStd).SetConfirm(true).SetText("View standard").SetIcon(icons.Visibility)
		vs.SetUpdateFunc(func() {
			vs.SetEnabledUpdt(pt != &StdCmds)
		})
		tb.AddOverflowMenu(func(m *gi.Scene) {
			giv.NewFuncButton(m, pt.OpenPrefs).SetIcon(icons.Open).SetKey(keyfun.OpenAlt1)
		})
	})
	d.NewWindow().Run()
}

////////////////////////////////////////////////////////////////////////////////////////
//  CmdValue

// Value registers CmdValue as the viewer of CmdName
func (kn CmdName) Value() giv.Value {
	return &CmdValue{}
}

// CmdValue presents an action for displaying an CmdName and selecting
type CmdValue struct {
	giv.ValueBase
}

func (vv *CmdValue) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.ButtonType
	return vv.WidgetTyp
}

func (vv *CmdValue) UpdateWidget() {
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

func (vv *CmdValue) ConfigWidget(w gi.Widget, sc *gi.Scene) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfigWidget(w)
	bt := vv.Widget.(*gi.Button)
	bt.SetType(gi.ButtonTonal)
	bt.Config(sc)
	bt.OnClick(func(e events.Event) {
		vv.OpenDialog(bt)
	})
	vv.UpdateWidget()
}

func (vv *CmdValue) HasDialog() bool {
	return true
}

func (vv *CmdValue) OpenDialog(ctx gi.Widget) {
	if vv.IsReadOnly() {
		return
	}
	si := 0
	cur := laser.ToString(vv.Value.Interface())
	_, curRow, _ := AvailCmds.CmdByName(CmdName(cur), false)
	d := gi.NewBody().AddTitle("Select a command").AddText(vv.Doc())
	giv.NewTableView(d).SetSlice(&AvailCmds).SetSelIdx(curRow).BindSelectDialog(d.Sc, &si)
	d.AddBottomBar(func(pw gi.Widget) {
		d.AddCancel(pw)
		d.AddOk(pw).SetText("Replace All").OnClick(func(e events.Event) {
			if si >= 0 {
				pt := AvailCmds[si]
				vv.SetValue(CommandName(pt.Cat, pt.Name))
				vv.UpdateWidget()
			}
		})
	})
	d.NewFullDialog(ctx).Run()
}

//////////////////////////////////////////////////////////////////////////////////////
//  SplitsView

// SplitsView opens a view of a splits table
func SplitsView(pt *Splits) {
	if gi.ActivateExistingMainWindow(pt) {
		return
	}
	d := gi.NewBody().SetTitle("Available Splitter Settings: Can duplicate an existing (using Ctxt Menu) as starting point for new one")
	d.Sc.Data = pt
	tv := giv.NewTableView(d).SetSlice(pt)
	AvailSplitsChanged = false
	tv.OnChange(func(e events.Event) {
		AvailSplitsChanged = true
	})

	d.AddTopBar(func(pw gi.Widget) {
		tb := d.DefaultTopAppBar(pw)
		sp := giv.NewFuncButton(tb, pt.SavePrefs).SetText("Save to prefs").SetIcon(icons.Save).SetKey(keyfun.Save)
		sp.SetUpdateFunc(func() {
			sp.SetEnabled(AvailSplitsChanged && pt == &StdSplits)
		})
		oj := giv.NewFuncButton(tb, pt.OpenJSON).SetText("Open").SetIcon(icons.Open).SetKey(keyfun.Open)
		oj.Args[0].SetTag("ext", ".json")
		sj := giv.NewFuncButton(tb, pt.SaveJSON).SetText("Save As").SetIcon(icons.SaveAs).SetKey(keyfun.SaveAs)
		sj.Args[0].SetTag("ext", ".json")
		tb.AddOverflowMenu(func(m *gi.Scene) {
			giv.NewFuncButton(m, pt.OpenPrefs).SetIcon(icons.Open).SetKey(keyfun.OpenAlt1)
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

func (vv *SplitValue) ConfigWidget(w gi.Widget, sc *gi.Scene) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfigWidget(w)
	bt := vv.Widget.(*gi.Button)
	bt.SetType(gi.ButtonTonal)
	bt.Config(sc)
	bt.OnClick(func(e events.Event) {
		vv.OpenDialog(bt)
	})
	vv.UpdateWidget()
}

func (vv *SplitValue) HasDialog() bool {
	return true
}

func (vv *SplitValue) OpenDialog(ctx gi.Widget) {
	if vv.IsReadOnly() {
		return
	}
	si := 0
	cur := laser.ToString(vv.Value.Interface())
	curRow := -1
	if cur != "" {
		_, curRow, _ = AvailSplits.SplitByName(SplitName(cur))
	}
	d := gi.NewBody().AddTitle("Select a Named Splitter Config").AddText(vv.Doc())
	giv.NewTableView(d).SetSlice(&AvailSplits).SetSelIdx(curRow).BindSelectDialog(d.Sc, &si)
	d.AddBottomBar(func(pw gi.Widget) {
		d.AddCancel(pw)
		d.AddOk(pw).SetText("Replace All").OnClick(func(e events.Event) {
			if si >= 0 {
				pt := AvailSplits[si]
				vv.SetValue(pt.Name)
				vv.UpdateWidget()
			}
		})
	})
	d.NewFullDialog(ctx).Run()
}

//////////////////////////////////////////////////////////////////////////////////////
//  RegistersView

// RegistersView opens a view of a commands table
func RegistersView(pt *Registers) {
	if gi.ActivateExistingMainWindow(pt) {
		return
	}
	d := gi.NewBody().SetTitle("Guide Registers")
	d.Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})
	d.Sc.Data = pt

	gi.NewLabel(d).SetText("Available Registers: Can duplicate an existing (using Ctxt Menu) as starting point for new one").SetType(gi.LabelHeadlineSmall)

	tv := giv.NewTableView(d).SetSlice(pt)

	AvailRegistersChanged = false
	tv.OnChange(func(e events.Event) {
		AvailRegistersChanged = true
	})

	d.AddTopBar(func(pw gi.Widget) {
		tb := d.DefaultTopAppBar(pw)
		sp := giv.NewFuncButton(tb, pt.SavePrefs).SetText("Save to prefs").SetIcon(icons.Save).SetKey(keyfun.Save)
		sp.SetUpdateFunc(func() {
			sp.SetEnabled(AvailRegistersChanged && pt == &AvailRegisters)
		})
		oj := giv.NewFuncButton(tb, pt.OpenJSON).SetText("Open").SetIcon(icons.Open).SetKey(keyfun.Open)
		oj.Args[0].SetTag("ext", ".json")
		sj := giv.NewFuncButton(tb, pt.SaveJSON).SetText("Save As").SetIcon(icons.SaveAs).SetKey(keyfun.SaveAs)
		sj.Args[0].SetTag("ext", ".json")
		tb.AddOverflowMenu(func(m *gi.Scene) {
			giv.NewFuncButton(m, pt.OpenPrefs).SetIcon(icons.Open).SetKey(keyfun.OpenAlt1)
		})
	})

	d.NewWindow().Run()
}

////////////////////////////////////////////////////////////////////////////////////////
//  RegisterValue

// Value registers RegisterValue as the viewer of RegisterName
func (kn RegisterName) Value() giv.Value {
	return &RegisterValue{}
}

// RegisterValue presents an action for displaying an RegisterName and selecting
type RegisterValue struct {
	giv.ValueBase
}

func (vv *RegisterValue) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.ButtonType
	return vv.WidgetTyp
}

func (vv *RegisterValue) UpdateWidget() {
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

func (vv *RegisterValue) ConfigWidget(w gi.Widget, sc *gi.Scene) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfigWidget(w)
	bt := vv.Widget.(*gi.Button)
	bt.SetType(gi.ButtonTonal)
	bt.Config(sc)
	bt.OnClick(func(e events.Event) {
		vv.OpenDialog(bt)
	})
	vv.UpdateWidget()
}

func (vv *RegisterValue) HasDialog() bool {
	return true
}

func (vv *RegisterValue) OpenDialog(ctx gi.Widget) {
	if vv.IsReadOnly() {
		return
	}
	cur := laser.ToString(vv.Value.Interface())
	m := gi.NewMenuFromStrings(AvailRegisterNames, cur, func(idx int) {
		rnm := AvailRegisterNames[idx]
		if ci := strings.Index(rnm, ":"); ci > 0 {
			rnm = rnm[:ci]
		}
		vv.SetValue(rnm)
		vv.UpdateWidget()
	})
	gi.NewMenuFromScene(m, ctx, ctx.ContextMenuPos(nil)).Run()
}