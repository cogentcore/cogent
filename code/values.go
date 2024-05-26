// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"strings"

	"cogentcore.org/cogent/code/cdebug"
	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/views"
)

// KeyMapsView opens a view of a key maps table
func KeyMapsView(km *KeyMaps) {
	if core.RecycleMainWindow(km) {
		return
	}
	d := core.NewBody().SetTitle("Available Key Maps: duplicate an existing map (using context menu) as starting point for creating a custom map").SetData(km)
	tv := views.NewTableView(d).SetSlice(km)
	AvailableKeyMapsChanged = false
	tv.OnChange(func(e events.Event) {
		AvailableKeyMapsChanged = true
	})
	d.AddAppBar(func(p *core.Plan) {
		core.Add(p, func(w *views.FuncButton) {
			w.SetFunc(km.SaveSettings).
				SetText("Save to settings").SetIcon(icons.Save).SetKey(keymap.Save).
				StyleFirst(func(s *styles.Style) {
					s.SetEnabled(AvailableKeyMapsChanged && km == &AvailableKeyMaps)
				})
		})
		core.Add(p, func(w *views.FuncButton) {
			w.SetFunc(km.Open).SetText("Open").SetIcon(icons.Open).SetKey(keymap.Open)
			// w.Args[0].SetTag("ext", ".toml") // todo:
		})
		core.Add(p, func(w *views.FuncButton) {
			w.SetFunc(km.Save).SetText("Save As").SetIcon(icons.SaveAs).SetKey(keymap.SaveAs)
			// w.Args[0].SetTag("ext", ".toml")
		})
		core.Add[*core.Separator](p)
		core.Add(p, func(w *views.FuncButton) {
			w.SetFunc(km.ViewStandard).SetConfirm(true).
				SetText("View standard").SetIcon(icons.Visibility).
				StyleFirst(func(s *styles.Style) { s.SetEnabled(km != &StandardKeyMaps) })
		})
		core.Add(p, func(w *views.FuncButton) {
			w.SetFunc(km.RevertToStandard).SetConfirm(true).
				SetText("Revert to standard").SetIcon(icons.DeviceReset).
				StyleFirst(func(s *styles.Style) { s.SetEnabled(km != &StandardKeyMaps) })
		})
		// tb.AddOverflowMenu(func(m *core.Scene) {
		// 	views.NewFuncButton(m, km.OpenSettings).SetIcon(icons.Open).SetKey(keymap.OpenAlt1)
		// })
	})
	d.RunWindow()
}

// ProjectSettingsView opens a view of project settings,
// returns structview if not already open
func ProjectSettingsView(pf *ProjectSettings) *views.StructView {
	if core.RecycleMainWindow(pf) {
		return nil
	}
	d := core.NewBody().SetTitle("Code project settings").SetData(pf)
	core.NewText(d).SetText("Settings are saved in the project .code file, along with other current state (open directories, splitter settings, etc). Do Save All or Save Project to save.")
	tv := views.NewStructView(d).SetStruct(pf)
	tv.OnChange(func(e events.Event) {
		pf.Update()
		core.ErrorSnackbar(d, pf.Save(pf.ProjectFilename), "Error saving "+string(pf.ProjectFilename)+" settings")
	})
	d.RunWindow()
	return tv
}

// DebugSettingsView opens a view of project Debug settings,
// returns structview if not already open
func DebugSettingsView(pf *cdebug.Params) *views.StructView {
	if core.RecycleMainWindow(pf) {
		return nil
	}
	d := core.NewBody().SetTitle("Project debug settings").SetData(pf)
	core.NewText(d).SetText("For args: Use -- double-dash and then add args to pass args to the executable (double-dash is by itself as a separate arg first).  For Debug test, must use -test.run instead of plain -run to specify tests to run")
	tv := views.NewStructView(d).SetStruct(pf)
	d.RunWindow()
	return tv
}

// Value registers [KeyMapValue] as the [views.Value] for [KeyMapName].
func (kn KeyMapName) Value() views.Value {
	return &KeyMapValue{}
}

// KeyMapValue represents a [KeyMapName] value with a button.
type KeyMapValue struct {
	views.ValueBase[*core.Button]
}

func (v *KeyMapValue) Config() {
	v.Widget.SetType(core.ButtonTonal)
	views.ConfigDialogWidget(v, false)
}

func (v *KeyMapValue) Update() {
	txt := reflectx.ToString(v.Value.Interface())
	if txt == "" {
		txt = "(none; click to set)"
	}
	v.Widget.SetText(txt).Update()
}

func (vv *KeyMapValue) ConfigDialog(d *core.Body) (bool, func()) {
	si := 0
	cur := reflectx.ToString(vv.Value.Interface())
	_, curRow, _ := AvailableKeyMaps.MapByName(KeyMapName(cur))
	views.NewTableView(d).SetSlice(&AvailableKeyMaps).SetSelectedIndex(curRow).BindSelect(&si)
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
	if core.RecycleMainWindow(pt) {
		return
	}
	d := core.NewBody().SetTitle("Available Language Opts: add or modify entries to customize options for language / file types").SetData(pt)
	tv := views.NewMapView(d).SetMap(pt)
	AvailableLangsChanged = false
	tv.OnChange(func(e events.Event) {
		AvailableLangsChanged = true
	})

	d.AddAppBar(func(p *core.Plan) {
		core.Add(p, func(w *views.FuncButton) {
			w.SetFunc(pt.SaveSettings).
				SetText("Save to settings").SetIcon(icons.Save).SetKey(keymap.Save).
				StyleFirst(func(s *styles.Style) { s.SetEnabled(AvailableLangsChanged && pt == &AvailableLangs) })
		})
		core.Add(p, func(w *views.FuncButton) {
			w.SetFunc(pt.Open).SetText("Open").SetIcon(icons.Open).SetKey(keymap.Open)
			// w.Args[0].SetTag("ext", ".toml")
		})
		core.Add(p, func(w *views.FuncButton) {
			w.SetFunc(pt.Save).SetText("Save As").SetIcon(icons.SaveAs).SetKey(keymap.SaveAs)
			// w.Args[0].SetTag("ext", ".toml")
		})
		core.Add[*core.Separator](p)
		core.Add(p, func(w *views.FuncButton) {
			w.SetFunc(pt.ViewStandard).SetConfirm(true).
				SetText("View standard").SetIcon(icons.Visibility).
				StyleFirst(func(s *styles.Style) { s.SetEnabled(pt != &StandardLangs) })
		})
		core.Add(p, func(w *views.FuncButton) {
			w.SetFunc(pt.RevertToStandard).SetConfirm(true).
				SetText("Revert to standard").SetIcon(icons.DeviceReset).
				StyleFirst(func(s *styles.Style) { s.SetEnabled(pt != &StandardLangs) })
		})
		// todo:
		// tb.AddOverflowMenu(func(m *core.Scene) {
		// 	views.NewFuncButton(m, pt.OpenSettings).SetIcon(icons.Open).SetKey(keymap.OpenAlt1)
		// })
	})
	d.RunWindow()
}

// CmdsView opens a view of a commands table
func CmdsView(pt *Commands) {
	if core.RecycleMainWindow(pt) {
		return
	}
	d := core.NewBody().SetTitle("Code Commands").SetData(pt)
	tv := views.NewTableView(d).SetSlice(pt)
	CustomCommandsChanged = false
	tv.OnChange(func(e events.Event) {
		CustomCommandsChanged = true
	})
	d.AddAppBar(func(p *core.Plan) {
		core.Add(p, func(w *views.FuncButton) {
			w.SetFunc(pt.SaveSettings).SetText("Save to settings").
				SetIcon(icons.Save).SetKey(keymap.Save).
				StyleFirst(func(s *styles.Style) { s.SetEnabled(CustomCommandsChanged && pt == &CustomCommands) })
		})
		core.Add(p, func(w *views.FuncButton) {
			w.SetFunc(pt.Open).SetText("Open").SetIcon(icons.Open).SetKey(keymap.Open)
			// w.Args[0].SetTag("ext", ".toml")
		})
		core.Add(p, func(w *views.FuncButton) {
			w.SetFunc(pt.Save).SetText("Save As").SetIcon(icons.SaveAs).SetKey(keymap.SaveAs)
			// w.Args[0].SetTag("ext", ".toml")
		})
		core.Add[*core.Separator](p)
		core.Add(p, func(w *views.FuncButton) {
			w.SetFunc(pt.ViewStandard).SetConfirm(true).
				SetText("View standard").SetIcon(icons.Visibility).
				StyleFirst(func(s *styles.Style) { s.SetEnabled(pt != &StandardCommands) })
		})
		// todo:
		// tb.AddOverflowMenu(func(m *core.Scene) {
		// 	views.NewFuncButton(m, pt.OpenSettings).SetIcon(icons.Open).SetKey(keymap.OpenAlt1)
		// })
	})
	d.RunWindow()
}

// Value registers [CmdValue] as the [views.Value] for [CmdName].
func (cn CmdName) Value() views.Value {
	return &CmdValue{}
}

// CmdValue represents a [CmdName] value with a button.
type CmdValue struct {
	views.ValueBase[*core.Button]
}

func (v *CmdValue) Config() {
	v.Widget.SetType(core.ButtonTonal)
	views.ConfigDialogWidget(v, false)
}

func (v *CmdValue) Update() {
	txt := reflectx.ToString(v.Value.Interface())
	if txt == "" {
		txt = "(none)"
	}
	v.Widget.SetText(txt).Update()
}

func (vv *CmdValue) ConfigDialog(d *core.Body) (bool, func()) {
	si := 0
	cur := reflectx.ToString(vv.Value.Interface())
	_, curRow, _ := AvailableCommands.CmdByName(CmdName(cur), false)
	views.NewTableView(d).SetSlice(&AvailableCommands).SetSelectedIndex(curRow).BindSelect(&si)
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
	if core.RecycleMainWindow(pt) {
		return
	}
	d := core.NewBody().SetTitle("Available Splitter Settings: can duplicate an existing (using context menu) as starting point for new one").SetData(pt)
	tv := views.NewTableView(d).SetSlice(pt)
	AvailableSplitsChanged = false
	tv.OnChange(func(e events.Event) {
		AvailableSplitsChanged = true
	})

	d.AddAppBar(func(p *core.Plan) {
		core.Add(p, func(w *views.FuncButton) {
			w.SetFunc(pt.SaveSettings).SetText("Save to settings").
				SetIcon(icons.Save).SetKey(keymap.Save).
				StyleFirst(func(s *styles.Style) {
					s.SetEnabled(AvailableSplitsChanged && pt == &StandardSplits)
				})
		})
		core.Add(p, func(w *views.FuncButton) {
			w.SetFunc(pt.Open).SetText("Open").SetIcon(icons.Open).SetKey(keymap.Open)
			// w.Args[0].SetTag("ext", ".toml")
		})
		core.Add(p, func(w *views.FuncButton) {
			w.SetFunc(pt.Save).SetText("Save As").SetIcon(icons.SaveAs).SetKey(keymap.SaveAs)
			// w.Args[0].SetTag("ext", ".toml")
		})
		// todo:
		// tb.AddOverflowMenu(func(m *core.Scene) {
		// 	views.NewFuncButton(m, pt.OpenSettings).SetIcon(icons.Open).SetKey(keymap.OpenAlt1)
		// })
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

// RegistersView opens a view of a commands table
func RegistersView(pt *Registers) {
	if core.RecycleMainWindow(pt) {
		return
	}
	d := core.NewBody().SetTitle("Cogent Code Registers").SetData(pt)
	d.Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})

	core.NewText(d).SetText("Available Registers: can duplicate an existing (using context menu) as starting point for new one").SetType(core.TextHeadlineSmall)

	tv := views.NewTableView(d).SetSlice(pt)

	AvailableRegistersChanged = false
	tv.OnChange(func(e events.Event) {
		AvailableRegistersChanged = true
	})

	d.AddAppBar(func(p *core.Plan) {
		core.Add(p, func(w *views.FuncButton) {
			w.SetFunc(pt.SaveSettings).SetText("Save to settings").
				SetIcon(icons.Save).SetKey(keymap.Save).
				StyleFirst(func(s *styles.Style) {
					s.SetEnabled(AvailableRegistersChanged && pt == &AvailableRegisters)
				})
		})
		core.Add(p, func(w *views.FuncButton) {
			w.SetFunc(pt.Open).SetText("Open").SetIcon(icons.Open).SetKey(keymap.Open)
			// w.Args[0].SetTag("ext", ".toml")
		})
		core.Add(p, func(w *views.FuncButton) {
			w.SetFunc(pt.Save).SetText("Save As").SetIcon(icons.SaveAs).SetKey(keymap.SaveAs)
			// w.Args[0].SetTag("ext", ".toml")
		})
		// todo:
		// tb.AddOverflowMenu(func(m *core.Scene) {
		// 	views.NewFuncButton(m, pt.OpenSettings).SetIcon(icons.Open).SetKey(keymap.OpenAlt1)
		// })
	})

	d.RunWindow()
}

// Value registers [RegisterValue] as the [views.Value] for [RegisterName].
func (rn RegisterName) Value() views.Value {
	return &RegisterValue{}
}

// RegisterValue represents a [RegisterName] value with a button.
type RegisterValue struct {
	views.ValueBase[*core.Button]
}

func (v *RegisterValue) Config() {
	v.Widget.SetType(core.ButtonTonal)
	views.ConfigDialogWidget(v, false)
}

func (v *RegisterValue) Update() {
	txt := reflectx.ToString(v.Value.Interface())
	if txt == "" {
		txt = "(none)"
	}
	v.Widget.SetText(txt).Update()
}

func (v *RegisterValue) OpenDialog(ctx core.Widget, fun func()) {
	if len(AvailableRegisterNames) == 0 {
		core.MessageSnackbar(ctx, "No registers available")
		return
	}
	cur := reflectx.ToString(v.Value.Interface())
	m := core.NewMenuFromStrings(AvailableRegisterNames, cur, func(idx int) {
		rnm := AvailableRegisterNames[idx]
		if ci := strings.Index(rnm, ":"); ci > 0 {
			rnm = rnm[:ci]
		}
		v.SetValue(rnm)
		v.Update()
		if fun != nil {
			fun()
		}
	})
	core.NewMenuStage(m, ctx, ctx.ContextMenuPos(nil)).Run()
}
