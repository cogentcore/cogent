// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"path/filepath"
	"slices"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/iox/tomlx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
)

// Registers is a list of named strings
type Registers map[string]string //types:add

// RegisterName has an associated ValueView for selecting from the list of
// available named registers
type RegisterName string

// AvailableRegisters are available named registers.  can be loaded / saved /
// edited with settings.
var AvailableRegisters Registers

// AvailableRegisterNames are the names of the current AvailRegisters -- used for some choosers
var AvailableRegisterNames []string

// Names returns a slice of current register names
func (lt *Registers) Names() []string {
	nms := make([]string, len(*lt))
	i := 0
	for key, val := range *lt {
		if len(val) > 20 {
			val = val[:20]
		}
		nms[i] = key + ": " + val
		i++
	}
	slices.Sort(nms)
	return nms
}

// RegisterSettingsFilename is the name of the settings file in the app settings
// directory for saving / loading the default AvailableRegisters
var RegisterSettingsFilename = "register-settings.toml"

// Open opens named registers from a toml-formatted file.
func (lt *Registers) Open(filename core.Filename) error { //types:add
	*lt = make(Registers) // reset
	return errors.Log(tomlx.Open(lt, string(filename)))
}

// Save saves named registers to a toml-formatted file.
func (lt *Registers) Save(filename core.Filename) error { //types:add
	return errors.Log(tomlx.Save(lt, string(filename)))
}

// OpenSettings opens the Registers from the app settings directory,
// using RegisterSettingsFilename.
func (lt *Registers) OpenSettings() error { //types:add
	pdir := core.TheApp.AppDataDir()
	pnm := filepath.Join(pdir, RegisterSettingsFilename)
	AvailableRegistersChanged = false
	err := lt.Open(core.Filename(pnm))
	if err == nil {
		AvailableRegisterNames = lt.Names()
	}
	return err
}

// SaveSettings saves the Registers to the app settings directory,
// using RegisterSettingsFilename.
func (lt *Registers) SaveSettings() error { //types:add
	pdir := core.TheApp.AppDataDir()
	pnm := filepath.Join(pdir, RegisterSettingsFilename)
	AvailableRegistersChanged = false
	AvailableRegisterNames = lt.Names()
	return lt.Save(core.Filename(pnm))
}

// AvailableRegistersChanged is used to update toolbars via following menu, toolbar
// properties update methods -- not accurate if editing any other map but works for
// now..
var AvailableRegistersChanged = false

// RegistersView opens a view of a commands table
func RegistersView(pt *Registers) {
	if core.RecycleMainWindow(pt) {
		return
	}
	d := core.NewBody().SetTitle("Cogent Code Registers").SetData(pt)
	d.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
	})

	core.NewText(d).SetText("Available Registers: can duplicate an existing (using context menu) as starting point for new one").SetType(core.TextHeadlineSmall)

	tv := core.NewKeyedList(d).SetMap(pt)

	AvailableRegistersChanged = false
	tv.OnChange(func(e events.Event) {
		AvailableRegistersChanged = true
	})

	d.AddAppBar(func(p *tree.Plan) {
		tree.Add(p, func(w *core.FuncButton) {
			w.SetFunc(pt.SaveSettings).SetText("Save to settings").
				SetIcon(icons.Save).SetKey(keymap.Save).
				FirstStyler(func(s *styles.Style) {
					s.SetEnabled(AvailableRegistersChanged && pt == &AvailableRegisters)
				})
		})
		tree.Add(p, func(w *core.FuncButton) {
			w.SetFunc(pt.Open).SetText("Open").SetIcon(icons.Open).SetKey(keymap.Open)
			w.Args[0].SetTag(`ext:".toml"`)
		})
		tree.Add(p, func(w *core.FuncButton) {
			w.SetFunc(pt.Save).SetText("Save as").SetIcon(icons.SaveAs).SetKey(keymap.SaveAs)
			w.Args[0].SetTag(`ext:".toml"`)
		})
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
