// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"path/filepath"

	"goki.dev/gi/v2/gi"
	"goki.dev/goosi"
	"goki.dev/grows/jsons"
	"goki.dev/grr"
)

// Registers is a list of named strings
type Registers map[string]string

// RegisterName has an associated ValueView for selecting from the list of
// available named registers
type RegisterName string

func (rn RegisterName) String() string {
	return string(rn)
}

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

// PrefsRegistersFileName is the name of the preferences file in App prefs
// directory for saving / loading the default AvailRegisters
var PrefsRegistersFileName = "registers_prefs.json"

// OpenJSON opens named registers from a JSON-formatted file.
func (lt *Registers) OpenJSON(filename gi.FileName) error { //gti:add
	*lt = make(Registers) // reset
	return grr.Log0(jsons.Open(lt, string(filename)))
}

// SaveJSON saves named registers to a JSON-formatted file.
func (lt *Registers) SaveJSON(filename gi.FileName) error { //gti:add
	return grr.Log0(jsons.Save(lt, string(filename)))
}

// OpenPrefs opens Registers from App standard prefs directory, using PrefRegistersFileName
func (lt *Registers) OpenPrefs() error { //gti:add
	pdir := goosi.TheApp.AppPrefsDir()
	pnm := filepath.Join(pdir, PrefsRegistersFileName)
	AvailRegistersChanged = false
	err := lt.OpenJSON(gi.FileName(pnm))
	if err == nil {
		AvailRegisterNames = lt.Names()
	}
	return err
}

// SavePrefs saves Registers to App standard prefs directory, using PrefRegistersFileName
func (lt *Registers) SavePrefs() error { //gti:add
	pdir := goosi.TheApp.AppPrefsDir()
	pnm := filepath.Join(pdir, PrefsRegistersFileName)
	AvailRegistersChanged = false
	AvailRegisterNames = lt.Names()
	return lt.SaveJSON(gi.FileName(pnm))
}

// AvailRegistersChanged is used to update toolbars via following menu, toolbar
// props update methods -- not accurate if editing any other map but works for
// now..
var AvailRegistersChanged = false
