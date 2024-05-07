// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"path/filepath"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/iox/tomlx"
	"cogentcore.org/core/core"
)

// Registers is a list of named strings
type Registers map[string]string

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
