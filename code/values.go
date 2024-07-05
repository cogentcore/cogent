// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"cogentcore.org/cogent/code/cdebug"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
)

// ProjectSettingsEditor opens a view of project settings,
// returns form if not already open
func ProjectSettingsEditor(pf *ProjectSettings) *core.Form {
	if core.RecycleMainWindow(pf) {
		return nil
	}
	d := core.NewBody().SetTitle("Code project settings").SetData(pf)
	core.NewText(d).SetText("Settings are saved in the project .code file, along with other current state (open directories, splitter settings, etc). Do Save All or Save Project to save.")
	tv := core.NewForm(d).SetStruct(pf)
	tv.OnChange(func(e events.Event) {
		pf.Update()
		core.ErrorSnackbar(d, pf.Save(pf.ProjectFilename), "Error saving "+string(pf.ProjectFilename)+" settings")
	})
	d.RunWindow()
	return tv
}

// DebugSettingsEditor opens a view of project Debug settings,
// returns form if not already open
func DebugSettingsEditor(pf *cdebug.Params) *core.Form {
	if core.RecycleMainWindow(pf) {
		return nil
	}
	d := core.NewBody().SetTitle("Project debug settings").SetData(pf)
	core.NewText(d).SetText("For args: Use -- double-dash and then add args to pass args to the executable (double-dash is by itself as a separate arg first).  For Debug test, must use -test.run instead of plain -run to specify tests to run")
	tv := core.NewForm(d).SetStruct(pf)
	d.RunWindow()
	return tv
}

// LangsView opens a view of a languages options map
func LangsView(pt *Langs) {
	if core.RecycleMainWindow(pt) {
		return
	}
	d := core.NewBody().SetTitle("Available Language Opts: add or modify entries to customize options for language / file types").SetData(pt)
	tv := core.NewKeyedList(d).SetMap(pt)
	AvailableLangsChanged = false
	tv.OnChange(func(e events.Event) {
		AvailableLangsChanged = true
	})

	d.AddAppBar(func(p *tree.Plan) {
		tree.Add(p, func(w *core.FuncButton) {
			w.SetFunc(pt.SaveSettings).
				SetText("Save to settings").SetIcon(icons.Save).SetKey(keymap.Save).
				FirstStyler(func(s *styles.Style) { s.SetEnabled(AvailableLangsChanged && pt == &AvailableLangs) })
		})
		tree.Add(p, func(w *core.FuncButton) {
			w.SetFunc(pt.Open).SetText("Open").SetIcon(icons.Open).SetKey(keymap.Open)
			w.Args[0].SetTag(`ext:".toml"`)
		})
		tree.Add(p, func(w *core.FuncButton) {
			w.SetFunc(pt.Save).SetText("Save as").SetIcon(icons.SaveAs).SetKey(keymap.SaveAs)
			w.Args[0].SetTag(`ext:".toml"`)
		})
		tree.Add(p, func(w *core.Separator) {})
		tree.Add(p, func(w *core.FuncButton) {
			w.SetFunc(pt.ViewStandard).SetConfirm(true).
				SetText("View standard").SetIcon(icons.Visibility).
				FirstStyler(func(s *styles.Style) { s.SetEnabled(pt != &StandardLangs) })
		})
		tree.Add(p, func(w *core.FuncButton) {
			w.SetFunc(pt.RevertToStandard).SetConfirm(true).
				SetText("Revert to standard").SetIcon(icons.DeviceReset).
				FirstStyler(func(s *styles.Style) { s.SetEnabled(pt != &StandardLangs) })
		})
	})
	d.RunWindow()
}

// CmdsView opens a view of a commands table
func CmdsView(pt *Commands) {
	if core.RecycleMainWindow(pt) {
		return
	}
	d := core.NewBody().SetTitle("Code Commands").SetData(pt)
	tv := core.NewTable(d).SetSlice(pt)
	CustomCommandsChanged = false
	tv.OnChange(func(e events.Event) {
		CustomCommandsChanged = true
	})
	d.AddAppBar(func(p *tree.Plan) {
		tree.Add(p, func(w *core.FuncButton) {
			w.SetFunc(pt.SaveSettings).SetText("Save to settings").
				SetIcon(icons.Save).SetKey(keymap.Save).
				FirstStyler(func(s *styles.Style) { s.SetEnabled(CustomCommandsChanged && pt == &CustomCommands) })
		})
		tree.Add(p, func(w *core.FuncButton) {
			w.SetFunc(pt.Open).SetText("Open").SetIcon(icons.Open).SetKey(keymap.Open)
			w.Args[0].SetTag(`ext:".toml"`)
		})
		tree.Add(p, func(w *core.FuncButton) {
			w.SetFunc(pt.Save).SetText("Save as").SetIcon(icons.SaveAs).SetKey(keymap.SaveAs)
			w.Args[0].SetTag(`ext:".toml"`)
		})
		tree.Add(p, func(w *core.Separator) {})
		tree.Add(p, func(w *core.FuncButton) {
			w.SetFunc(pt.ViewStandard).SetConfirm(true).
				SetText("View standard").SetIcon(icons.Visibility).
				FirstStyler(func(s *styles.Style) { s.SetEnabled(pt != &StandardCommands) })
		})
	})
	d.RunWindow()
}

func (cn CmdName) Value() core.Value {
	return NewCmdButton()
}

// CmdButton represents a [CmdName] value with a button that opens a [CmdView].
type CmdButton struct {
	core.Button
}

func (cb *CmdButton) WidgetValue() any { return &cb.Text }

func (cb *CmdButton) Init() {
	cb.Button.Init()
	cb.SetType(core.ButtonTonal).SetIcon(icons.PlayArrow)
	core.InitValueButton(cb, false, func(d *core.Body) {
		d.SetTitle("Select a command")
		si := 0
		cl := AvailableCommands
		tv := core.NewTable(d)
		// todo: not a single entry: SetSelectedField("Name").SetSelectedValue(cb.Text)
		tv.SetSlice(&cl).BindSelect(&si)
		tv.OnChange(func(e events.Event) {
			cb.Text = CommandName(cl[si].Cat, cl[si].Name)
		})
	})
}
