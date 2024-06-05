// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"strings"

	"cogentcore.org/cogent/code/cdebug"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/views"
)

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
			w.Args[0].SetTag(`ext:".toml"`)
		})
		core.Add(p, func(w *views.FuncButton) {
			w.SetFunc(pt.Save).SetText("Save As").SetIcon(icons.SaveAs).SetKey(keymap.SaveAs)
			w.Args[0].SetTag(`ext:".toml"`)
		})
		core.Add(p, func(w *core.Separator) {})
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
			w.Args[0].SetTag(`ext:".toml"`)
		})
		core.Add(p, func(w *views.FuncButton) {
			w.SetFunc(pt.Save).SetText("Save As").SetIcon(icons.SaveAs).SetKey(keymap.SaveAs)
			w.Args[0].SetTag(`ext:".toml"`)
		})
		core.Add(p, func(w *core.Separator) {})
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

/*
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
*/

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
			w.Args[0].SetTag(`ext:".toml"`)
		})
		core.Add(p, func(w *views.FuncButton) {
			w.SetFunc(pt.Save).SetText("Save As").SetIcon(icons.SaveAs).SetKey(keymap.SaveAs)
			w.Args[0].SetTag(`ext:".toml"`)
		})
		// todo:
		// tb.AddOverflowMenu(func(m *core.Scene) {
		// 	views.NewFuncButton(m, pt.OpenSettings).SetIcon(icons.Open).SetKey(keymap.OpenAlt1)
		// })
	})
	d.RunWindow()
}

// Value registers [core.Chooser] as the [core.Value] widget
// for [SplitName]
func (sn SplitName) Value() core.Value {
	return core.NewChooser().SetStrings(AvailableSplitNames...)
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
			w.Args[0].SetTag(`ext:".toml"`)
		})
		core.Add(p, func(w *views.FuncButton) {
			w.SetFunc(pt.Save).SetText("Save As").SetIcon(icons.SaveAs).SetKey(keymap.SaveAs)
			w.Args[0].SetTag(`ext:".toml"`)
		})
		// todo:
		// tb.AddOverflowMenu(func(m *core.Scene) {
		// 	views.NewFuncButton(m, pt.OpenSettings).SetIcon(icons.Open).SetKey(keymap.OpenAlt1)
		// })
	})

	d.RunWindow()
}

// Value registers [core.Chooser] as the [core.Value] widget
// for [RegisterName]
func (rn RegisterName) Value() core.Value {
	ch := core.NewChooser().SetStrings(AvailableRegisterNames...)
	ch.SetEditable(true).SetAllowNew(true)
	return ch
}

// RegistersMenu presents a menu of existing registers,
// calling the given function with the selected register name
func RegistersMenu(ctx core.Widget, curVal string, fun func(regNm string)) {
	m := core.NewMenuFromStrings(AvailableRegisterNames, curVal, func(idx int) {
		rnm := AvailableRegisterNames[idx]
		if ci := strings.Index(rnm, ":"); ci > 0 {
			rnm = rnm[:ci]
		}
		if fun != nil {
			fun(rnm)
		}
	})
	core.NewMenuStage(m, ctx, ctx.ContextMenuPos(nil)).Run()
}
