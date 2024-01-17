// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"path/filepath"

	"cogentcore.org/core/gi"
	"cogentcore.org/core/grows/tomls"
	"cogentcore.org/core/grr"
)

// Registers is a list of named strings
type Registers map[string]string

// RegisterName has an associated ValueView for selecting from the list of
// available named registers
type RegisterName string

// AvailRegisters are available named registers.  can be loaded / saved /
// edited with preferences.
var AvailRegisters Registers

// AvailRegisterNames are the names of the current AvailRegisters -- used for some choosers
var AvailRegisterNames []string

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

// PrefsRegistersFilename is the name of the preferences file in App prefs
// directory for saving / loading the default AvailRegisters
var PrefsRegistersFilename = "registers_prefs.toml"

// Open opens named registers from a toml-formatted file.
func (lt *Registers) Open(filename gi.Filename) error { //gti:add
	*lt = make(Registers) // reset
	return grr.Log(tomls.Open(lt, string(filename)))
}

// Save saves named registers to a toml-formatted file.
func (lt *Registers) Save(filename gi.Filename) error { //gti:add
	return grr.Log(tomls.Save(lt, string(filename)))
}

// OpenSettings opens Registers from App standard prefs directory, using PrefRegistersFilename
func (lt *Registers) OpenSettings() error { //gti:add
	pdir := AppDataDir()
	pnm := filepath.Join(pdir, PrefsRegistersFilename)
	AvailRegistersChanged = false
	err := lt.Open(gi.Filename(pnm))
	if err == nil {
		AvailRegisterNames = lt.Names()
	}
	return err
}

// SavePrefs saves Registers to App standard prefs directory, using PrefRegistersFilename
func (lt *Registers) SavePrefs() error { //gti:add
	pdir := AppDataDir()
	pnm := filepath.Join(pdir, PrefsRegistersFilename)
	AvailRegistersChanged = false
	AvailRegisterNames = lt.Names()
	return lt.Save(gi.Filename(pnm))
}

// AvailRegistersChanged is used to update toolbars via following menu, toolbar
// props update methods -- not accurate if editing any other map but works for
// now..
var AvailRegistersChanged = false
