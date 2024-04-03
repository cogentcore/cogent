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
	d := gi.NewBody().SetTitle("Available Key Maps: duplicate an existing map (using context menu) as starting point for creating a custom map").SetData(km)
	tv := giv.NewTableView(d).SetSlice(km)
	AvailableKeyMapsChanged = false
	tv.OnChange(func(e events.Event) {
		AvailableKeyMapsChanged = true
	})
	d.AddAppBar(func(tb *gi.Toolbar) {
		giv.NewFuncButton(tb, km.SaveSettings).
			SetText("Save to settings").SetIcon(icons.Save).SetKey(keyfun.Save).
			StyleFirst(func(s *styles.Style) { s.SetEnabled(AvailableKeyMapsChanged && km == &AvailableKeyMaps) })
		oj := giv.NewFuncButton(tb, km.Open).SetText("Open").SetIcon(icons.Open).SetKey(keyfun.Open)
		oj.Args[0].SetTag("ext", ".toml")
		sj := giv.NewFuncButton(tb, km.Save).SetText("Save As").SetIcon(icons.SaveAs).SetKey(keyfun.SaveAs)
		sj.Args[0].SetTag("ext", ".toml")
		gi.NewSeparator(tb)
		giv.NewFuncButton(tb, km.ViewStandard).SetConfirm(true).
			SetText("View standard").SetIcon(icons.Visibility).
			StyleFirst(func(s *styles.Style) { s.SetEnabled(km != &StandardKeyMaps) })
		giv.NewFuncButton(tb, km.RevertToStandard).SetConfirm(true).
			SetText("Revert to standard").SetIcon(icons.DeviceReset).
			StyleFirst(func(s *styles.Style) { s.SetEnabled(km != &StandardKeyMaps) })
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

// Value registers [KeyMapValue] as the [giv.Value] for [KeyMapName].
func (kn KeyMapName) Value() giv.Value {
	return &KeyMapValue{}
}

// KeyMapValue represents a [KeyMapName] value with a button.
type KeyMapValue struct {
	giv.ValueBase[*gi.Button]
}

func (v *KeyMapValue) Config() {
	v.Widget.SetType(gi.ButtonTonal)
	giv.ConfigDialogWidget(v, false)
}

func (v *KeyMapValue) Update() {
	txt := laser.ToString(v.Value.Interface())
	if txt == "" {
		txt = "(none; click to set)"
	}
	v.Widget.SetText(txt).Update()
}

func (vv *KeyMapValue) ConfigDialog(d *gi.Body) (bool, func()) {
	si := 0
	cur := laser.ToString(vv.Value.Interface())
	_, curRow, _ := AvailableKeyMaps.MapByName(KeyMapName(cur))
	giv.NewTableView(d).SetSlice(&AvailableKeyMaps).SetSelectedIndex(curRow).BindSelect(&si)
	return true, func() {
		if si >= 0 {
			km := AvailableKeyMaps[si]
			vv.SetValue(km.Name)
			vv.Update()
		}
	}
}

// LangsView opens a view of a languages options map
func LangsView(pt *Langs) {
	if gi.ActivateExistingMainWindow(pt) {
		return
	}
	d := gi.NewBody().SetTitle("Available Language Opts: add or modify entries to customize options for language / file types").SetData(pt)
	tv := giv.NewMapView(d).SetMap(pt)
	AvailableLangsChanged = false
	tv.OnChange(func(e events.Event) {
		AvailableLangsChanged = true
	})

	d.AddAppBar(func(tb *gi.Toolbar) {
		giv.NewFuncButton(tb, pt.SaveSettings).
			SetText("Save to settings").SetIcon(icons.Save).SetKey(keyfun.Save).
			StyleFirst(func(s *styles.Style) { s.SetEnabled(AvailableLangsChanged && pt == &AvailableLangs) })
		oj := giv.NewFuncButton(tb, pt.Open).SetText("Open").SetIcon(icons.Open).SetKey(keyfun.Open)
		oj.Args[0].SetTag("ext", ".toml")
		sj := giv.NewFuncButton(tb, pt.Save).SetText("Save As").SetIcon(icons.SaveAs).SetKey(keyfun.SaveAs)
		sj.Args[0].SetTag("ext", ".toml")
		gi.NewSeparator(tb)
		giv.NewFuncButton(tb, pt.ViewStandard).SetConfirm(true).
			SetText("View standard").SetIcon(icons.Visibility).
			StyleFirst(func(s *styles.Style) { s.SetEnabled(pt != &StandardLangs) })
		giv.NewFuncButton(tb, pt.RevertToStandard).SetConfirm(true).
			SetText("Revert to standard").SetIcon(icons.DeviceReset).
			StyleFirst(func(s *styles.Style) { s.SetEnabled(pt != &StandardLangs) })
		tb.AddOverflowMenu(func(m *gi.Scene) {
			giv.NewFuncButton(m, pt.OpenSettings).SetIcon(icons.Open).SetKey(keyfun.OpenAlt1)
		})
	})
	d.NewWindow().Run()
}

// CmdsView opens a view of a commands table
func CmdsView(pt *Commands) {
	if gi.ActivateExistingMainWindow(pt) {
		return
	}
	d := gi.NewBody().SetTitle("Code Commands").SetData(pt)
	tv := giv.NewTableView(d).SetSlice(pt)
	CustomCommandsChanged = false
	tv.OnChange(func(e events.Event) {
		CustomCommandsChanged = true
	})
	d.AddAppBar(func(tb *gi.Toolbar) {
		giv.NewFuncButton(tb, pt.SavePrefs).SetText("Save to settings").
			SetIcon(icons.Save).SetKey(keyfun.Save).
			StyleFirst(func(s *styles.Style) { s.SetEnabled(CustomCommandsChanged && pt == &CustomCommands) })
		oj := giv.NewFuncButton(tb, pt.Open).SetText("Open").SetIcon(icons.Open).SetKey(keyfun.Open)
		oj.Args[0].SetTag("ext", ".toml")
		sj := giv.NewFuncButton(tb, pt.Save).SetText("Save As").SetIcon(icons.SaveAs).SetKey(keyfun.SaveAs)
		sj.Args[0].SetTag("ext", ".toml")
		gi.NewSeparator(tb)
		giv.NewFuncButton(tb, pt.ViewStandard).SetConfirm(true).
			SetText("View standard").SetIcon(icons.Visibility).
			StyleFirst(func(s *styles.Style) { s.SetEnabled(pt != &StandardCommands) })
		tb.AddOverflowMenu(func(m *gi.Scene) {
			giv.NewFuncButton(m, pt.OpenSettings).SetIcon(icons.Open).SetKey(keyfun.OpenAlt1)
		})
	})
	d.NewWindow().Run()
}

// Value registers [CmdValue] as the [giv.Value] for [CmdName].
func (cn CmdName) Value() giv.Value {
	return &CmdValue{}
}

// CmdValue represents a [CmdName] value with a button.
type CmdValue struct {
	giv.ValueBase[*gi.Button]
}

func (v *CmdValue) Config() {
	v.Widget.SetType(gi.ButtonTonal)
	giv.ConfigDialogWidget(v, false)
}

func (v *CmdValue) Update() {
	txt := laser.ToString(v.Value.Interface())
	if txt == "" {
		txt = "(none)"
	}
	v.Widget.SetText(txt).Update()
}

func (vv *CmdValue) ConfigDialog(d *gi.Body) (bool, func()) {
	si := 0
	cur := laser.ToString(vv.Value.Interface())
	_, curRow, _ := AvailableCommands.CmdByName(CmdName(cur), false)
	giv.NewTableView(d).SetSlice(&AvailableCommands).SetSelectedIndex(curRow).BindSelect(&si)
	return true, func() {
		if si >= 0 {
			pt := AvailableCommands[si]
			vv.SetValue(CommandName(pt.Cat, pt.Name))
			vv.Update()
		}
	}
}

// SplitsView opens a view of a splits table
func SplitsView(pt *Splits) {
	if gi.ActivateExistingMainWindow(pt) {
		return
	}
	d := gi.NewBody().SetTitle("Available Splitter Settings: can duplicate an existing (using context menu) as starting point for new one").SetData(pt)
	tv := giv.NewTableView(d).SetSlice(pt)
	AvailableSplitsChanged = false
	tv.OnChange(func(e events.Event) {
		AvailableSplitsChanged = true
	})

	d.AddAppBar(func(tb *gi.Toolbar) {
		giv.NewFuncButton(tb, pt.SavePrefs).SetText("Save to settings").
			SetIcon(icons.Save).SetKey(keyfun.Save).
			StyleFirst(func(s *styles.Style) { s.SetEnabled(AvailableSplitsChanged && pt == &StandardSplits) })
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

// RegistersView opens a view of a commands table
func RegistersView(pt *Registers) {
	if gi.ActivateExistingMainWindow(pt) {
		return
	}
	d := gi.NewBody().SetTitle("Cogent Code Registers").SetData(pt)
	d.Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})

	gi.NewLabel(d).SetText("Available Registers: can duplicate an existing (using context menu) as starting point for new one").SetType(gi.LabelHeadlineSmall)

	tv := giv.NewTableView(d).SetSlice(pt)

	AvailRegistersChanged = false
	tv.OnChange(func(e events.Event) {
		AvailRegistersChanged = true
	})

	d.AddAppBar(func(tb *gi.Toolbar) {
		giv.NewFuncButton(tb, pt.SavePrefs).SetText("Save to settings").
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

// Value registers [RegisterValue] as the [giv.Value] for [RegisterName].
func (rn RegisterName) Value() giv.Value {
	return &RegisterValue{}
}

// RegisterValue represents a [RegisterName] value with a button.
type RegisterValue struct {
	giv.ValueBase[*gi.Button]
}

func (v *RegisterValue) Config() {
	v.Widget.SetType(gi.ButtonTonal)
	giv.ConfigDialogWidget(v, false)
}

func (v *RegisterValue) Update() {
	txt := laser.ToString(v.Value.Interface())
	if txt == "" {
		txt = "(none)"
	}
	v.Widget.SetText(txt).Update()
}

func (v *RegisterValue) OpenDialog(ctx gi.Widget, fun func()) {
	if len(AvailRegisterNames) == 0 {
		gi.MessageSnackbar(ctx, "No registers available")
		return
	}
	cur := laser.ToString(v.Value.Interface())
	m := gi.NewMenuFromStrings(AvailRegisterNames, cur, func(idx int) {
		rnm := AvailRegisterNames[idx]
		if ci := strings.Index(rnm, ":"); ci > 0 {
			rnm = rnm[:ci]
		}
		v.SetValue(rnm)
		v.Update()
		if fun != nil {
			fun()
		}
	})
	gi.NewMenuStage(m, ctx, ctx.ContextMenuPos(nil)).Run()
}
