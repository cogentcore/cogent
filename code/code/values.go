// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"strings"

	"cogentcore.org/cogent/code/cdebug"
	"cogentcore.org/core/events"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/giv"
	"cogentcore.org/core/gti"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keyfun"
	"cogentcore.org/core/laser"
	"cogentcore.org/core/styles"
)

// KeyMapsView opens a view of a key maps table
func KeyMapsView(km *KeyMaps) {
	if gi.ActivateExistingMainWindow(km) {
		return
	}
	d := gi.NewBody().SetTitle("Available Key Maps: Duplicate an existing map (using Ctxt Menu) as starting point for creating a custom map").SetData(km)
	tv := giv.NewTableView(d).SetSlice(km)
	AvailKeyMapsChanged = false
	tv.OnChange(func(e events.Event) {
		AvailKeyMapsChanged = true
	})
	d.AddAppBar(func(tb *gi.Toolbar) {
		giv.NewFuncButton(tb, km.SavePrefs).
			SetText("Save to settings").SetIcon(icons.Save).SetKey(keyfun.Save).
			StyleFirst(func(s *styles.Style) { s.SetEnabled(AvailKeyMapsChanged && km == &AvailKeyMaps) })
		oj := giv.NewFuncButton(tb, km.Open).SetText("Open").SetIcon(icons.Open).SetKey(keyfun.Open)
		oj.Args[0].SetTag("ext", ".toml")
		sj := giv.NewFuncButton(tb, km.Save).SetText("Save As").SetIcon(icons.SaveAs).SetKey(keyfun.SaveAs)
		sj.Args[0].SetTag("ext", ".toml")
		gi.NewSeparator(tb)
		giv.NewFuncButton(tb, km.ViewStd).SetConfirm(true).
			SetText("View standard").SetIcon(icons.Visibility).
			StyleFirst(func(s *styles.Style) { s.SetEnabled(km != &StdKeyMaps) })
		giv.NewFuncButton(tb, km.RevertToStd).SetConfirm(true).
			SetText("Revert to standard").SetIcon(icons.DeviceReset).
			StyleFirst(func(s *styles.Style) { s.SetEnabled(km != &StdKeyMaps) })
		tb.AddOverflowMenu(func(m *gi.Scene) {
			giv.NewFuncButton(m, km.OpenSettings).SetIcon(icons.Open).SetKey(keyfun.OpenAlt1)
		})
	})
	d.NewWindow().Run()
}

//////////////////////////////////////////////////////////////////////////////////////
//  ProjSettingsView

// ProjSettingsView opens a view of project settings,
// returns structview if not already open
func ProjSettingsView(pf *ProjSettings) *giv.StructView {
	if gi.ActivateExistingMainWindow(pf) {
		return nil
	}
	d := gi.NewBody().SetTitle("Code project settings").SetData(pf)
	gi.NewLabel(d).SetText("Settings are saved in the project .code file, along with other current state (open directories, splitter settings, etc). Do Save All or Save Project to save.")
	tv := giv.NewStructView(d).SetStruct(pf)
	tv.OnChange(func(e events.Event) {
		pf.Update()
		gi.ErrorSnackbar(d, pf.Save(pf.ProjFilename), "Error saving "+string(pf.ProjFilename)+" settings")
	})
	d.NewWindow().Run()
	return tv
}

// DebugSettingsView opens a view of project Debug settings,
// returns structview if not already open
func DebugSettingsView(pf *cdebug.Params) *giv.StructView {
	if gi.ActivateExistingMainWindow(pf) {
		return nil
	}
	d := gi.NewBody().SetTitle("Project debug settings").SetData(pf)
	gi.NewLabel(d).SetText("For args: Use -- double-dash and then add args to pass args to the executable (double-dash is by itself as a separate arg first).  For Debug test, must use -test.run instead of plain -run to specify tests to run")
	tv := giv.NewStructView(d).SetStruct(pf)
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
		txt = "(none; click to set)"
	}
	bt.SetText(txt)
}

func (vv *KeyMapValue) ConfigWidget(w gi.Widget) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfigWidget(w)
	bt := vv.Widget.(*gi.Button)
	bt.SetType(gi.ButtonTonal)
	bt.Config()
	giv.ConfigDialogWidget(vv, bt, false)
	vv.UpdateWidget()
}

func (vv *KeyMapValue) HasDialog() bool                      { return true }
func (vv *KeyMapValue) OpenDialog(ctx gi.Widget, fun func()) { giv.OpenValueDialog(vv, ctx, fun) }

func (vv *KeyMapValue) ConfigDialog(d *gi.Body) (bool, func()) {
	si := 0
	cur := laser.ToString(vv.Value.Interface())
	_, curRow, _ := AvailKeyMaps.MapByName(KeyMapName(cur))
	giv.NewTableView(d).SetSlice(&AvailKeyMaps).SetSelIdx(curRow).BindSelect(&si)
	return true, func() {
		if si >= 0 {
			km := AvailKeyMaps[si]
			vv.SetValue(km.Name)
			vv.UpdateWidget()
		}
	}
}

//////////////////////////////////////////////////////////////////////////////////////
//  LangsView

// LangsView opens a view of a languages options map
func LangsView(pt *Langs) {
	if gi.ActivateExistingMainWindow(pt) {
		return
	}
	d := gi.NewBody().SetTitle("Available Language Opts: Add or modify entries to customize options for language / file types").SetData(pt)
	tv := giv.NewMapView(d).SetMap(pt)
	AvailLangsChanged = false
	tv.OnChange(func(e events.Event) {
		AvailLangsChanged = true
	})

	d.AddAppBar(func(tb *gi.Toolbar) {
		giv.NewFuncButton(tb, pt.SavePrefs).
			SetText("Save to settings").SetIcon(icons.Save).SetKey(keyfun.Save).
			StyleFirst(func(s *styles.Style) { s.SetEnabled(AvailLangsChanged && pt == &AvailLangs) })
		oj := giv.NewFuncButton(tb, pt.Open).SetText("Open").SetIcon(icons.Open).SetKey(keyfun.Open)
		oj.Args[0].SetTag("ext", ".toml")
		sj := giv.NewFuncButton(tb, pt.Save).SetText("Save As").SetIcon(icons.SaveAs).SetKey(keyfun.SaveAs)
		sj.Args[0].SetTag("ext", ".toml")
		gi.NewSeparator(tb)
		giv.NewFuncButton(tb, pt.ViewStd).SetConfirm(true).
			SetText("View standard").SetIcon(icons.Visibility).
			StyleFirst(func(s *styles.Style) { s.SetEnabled(pt != &StdLangs) })
		giv.NewFuncButton(tb, pt.RevertToStd).SetConfirm(true).
			SetText("Revert to standard").SetIcon(icons.DeviceReset).
			StyleFirst(func(s *styles.Style) { s.SetEnabled(pt != &StdLangs) })
		tb.AddOverflowMenu(func(m *gi.Scene) {
			giv.NewFuncButton(m, pt.OpenSettings).SetIcon(icons.Open).SetKey(keyfun.OpenAlt1)
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
	d := gi.NewBody().SetTitle("Code Commands").SetData(pt)
	tv := giv.NewTableView(d).SetSlice(pt)
	CustomCmdsChanged = false
	tv.OnChange(func(e events.Event) {
		CustomCmdsChanged = true
	})
	d.AddAppBar(func(tb *gi.Toolbar) {
		giv.NewFuncButton(tb, pt.SavePrefs).SetText("Save to prefs").
			SetIcon(icons.Save).SetKey(keyfun.Save).
			StyleFirst(func(s *styles.Style) { s.SetEnabled(CustomCmdsChanged && pt == &CustomCmds) })
		oj := giv.NewFuncButton(tb, pt.Open).SetText("Open").SetIcon(icons.Open).SetKey(keyfun.Open)
		oj.Args[0].SetTag("ext", ".toml")
		sj := giv.NewFuncButton(tb, pt.Save).SetText("Save As").SetIcon(icons.SaveAs).SetKey(keyfun.SaveAs)
		sj.Args[0].SetTag("ext", ".toml")
		gi.NewSeparator(tb)
		giv.NewFuncButton(tb, pt.ViewStd).SetConfirm(true).
			SetText("View standard").SetIcon(icons.Visibility).
			StyleFirst(func(s *styles.Style) { s.SetEnabled(pt != &StdCmds) })
		tb.AddOverflowMenu(func(m *gi.Scene) {
			giv.NewFuncButton(m, pt.OpenSettings).SetIcon(icons.Open).SetKey(keyfun.OpenAlt1)
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

func (vv *CmdValue) ConfigWidget(w gi.Widget) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfigWidget(w)
	bt := vv.Widget.(*gi.Button)
	bt.SetType(gi.ButtonTonal)
	bt.Config()
	giv.ConfigDialogWidget(vv, bt, false)
	vv.UpdateWidget()
}

func (vv *CmdValue) HasDialog() bool                      { return true }
func (vv *CmdValue) OpenDialog(ctx gi.Widget, fun func()) { giv.OpenValueDialog(vv, ctx, fun) }

func (vv *CmdValue) ConfigDialog(d *gi.Body) (bool, func()) {
	si := 0
	cur := laser.ToString(vv.Value.Interface())
	_, curRow, _ := AvailCmds.CmdByName(CmdName(cur), false)
	giv.NewTableView(d).SetSlice(&AvailCmds).SetSelIdx(curRow).BindSelect(&si)
	return true, func() {
		if si >= 0 {
			pt := AvailCmds[si]
			vv.SetValue(CommandName(pt.Cat, pt.Name))
			vv.UpdateWidget()
		}
	}
}

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

/*
func (vv *SplitValue) OpenDialog(ctx gi.Widget, fun func()) { giv.OpenValueDialog(vv, ctx, fun) }
func (vv *SplitValue) ConfigDialog(d *gi.Body) (bool, func()) {
	si := 0
	cur := laser.ToString(vv.Value.Interface())
	curRow := -1
	if cur != "" {
		_, curRow, _ = AvailSplits.SplitByName(SplitName(cur))
	}
	giv.NewTableView(d).SetSlice(&AvailSplits).SetInitSelIdx(curRow).BindSelectDialog(&si)
	return true, func() {
		if si >= 0 {
			pt := AvailSplits[si]
			vv.SetValue(pt.Name)
			vv.UpdateWidget()
		}
	}
}
*/

//////////////////////////////////////////////////////////////////////////////////////
//  RegistersView

// RegistersView opens a view of a commands table
func RegistersView(pt *Registers) {
	if gi.ActivateExistingMainWindow(pt) {
		return
	}
	d := gi.NewBody().SetTitle("Guide Registers").SetData(pt)
	d.Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})

	gi.NewLabel(d).SetText("Available Registers: Can duplicate an existing (using Ctxt Menu) as starting point for new one").SetType(gi.LabelHeadlineSmall)

	tv := giv.NewTableView(d).SetSlice(pt)

	AvailRegistersChanged = false
	tv.OnChange(func(e events.Event) {
		AvailRegistersChanged = true
	})

	d.AddAppBar(func(tb *gi.Toolbar) {
		giv.NewFuncButton(tb, pt.SavePrefs).SetText("Save to prefs").
			SetIcon(icons.Save).SetKey(keyfun.Save).
			StyleFirst(func(s *styles.Style) { s.SetEnabled(AvailRegistersChanged && pt == &AvailRegisters) })
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

func (vv *RegisterValue) ConfigWidget(w gi.Widget) {
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

func (vv *RegisterValue) HasDialog() bool { return true }

func (vv *RegisterValue) OpenDialog(ctx gi.Widget, fun func()) {
	if len(AvailRegisterNames) == 0 {
		gi.MessageSnackbar(ctx, "No registers available")
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
		if fun != nil {
			fun()
		}
	})
	gi.NewMenuStage(m, ctx, ctx.ContextMenuPos(nil)).Run()
}
