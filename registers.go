// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"path/filepath"

	"github.com/goki/gi"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/oswin"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
)

// Registers is a list of named strings
type Registers map[string]string

var KiT_Registers = kit.Types.AddType(&Registers{}, RegistersProps)

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

// PrefsRegistersFileName is the name of the preferences file in App prefs
// directory for saving / loading the default AvailRegisters
var PrefsRegistersFileName = "registers_prefs.json"

// OpenJSON opens named registers from a JSON-formatted file.
func (lt *Registers) OpenJSON(filename gi.FileName) error {
	b, err := ioutil.ReadFile(string(filename))
	if err != nil {
		// gi.PromptDialog(nil, gi.DlgOpts{Title: "File Not Found", Prompt: err.Error()}, true, false, nil, nil)
		// log.Println(err)
		return err
	}
	*lt = make(Registers) // reset
	rval := json.Unmarshal(b, lt)
	return rval
}

// SaveJSON saves named registers to a JSON-formatted file.
func (lt *Registers) SaveJSON(filename gi.FileName) error {
	b, err := json.MarshalIndent(lt, "", "  ")
	if err != nil {
		log.Println(err) // unlikely
		return err
	}
	err = ioutil.WriteFile(string(filename), b, 0644)
	if err != nil {
		gi.PromptDialog(nil, gi.DlgOpts{Title: "Could not Save to File", Prompt: err.Error()}, true, false, nil, nil)
		log.Println(err)
	}
	return err
}

// OpenPrefs opens Registers from App standard prefs directory, using PrefRegistersFileName
func (lt *Registers) OpenPrefs() error {
	pdir := oswin.TheApp.AppPrefsDir()
	pnm := filepath.Join(pdir, PrefsRegistersFileName)
	AvailRegistersChanged = false
	err := lt.OpenJSON(gi.FileName(pnm))
	if err == nil {
		AvailRegisterNames = lt.Names()
	}
	return err
}

// SavePrefs saves Registers to App standard prefs directory, using PrefRegistersFileName
func (lt *Registers) SavePrefs() error {
	pdir := oswin.TheApp.AppPrefsDir()
	pnm := filepath.Join(pdir, PrefsRegistersFileName)
	AvailRegistersChanged = false
	AvailRegisterNames = lt.Names()
	return lt.SaveJSON(gi.FileName(pnm))
}

// AvailRegistersChanged is used to update toolbars via following menu, toolbar
// props update methods -- not accurate if editing any other map but works for
// now..
var AvailRegistersChanged = false

// RegistersProps define the ToolBar and MenuBar for TableView of Registers
var RegistersProps = ki.Props{
	"MainMenu": ki.PropSlice{
		{"AppMenu", ki.BlankProp{}},
		{"File", ki.PropSlice{
			{"OpenPrefs", ki.Props{}},
			{"SavePrefs", ki.Props{
				"shortcut": "Command+S",
				"updtfunc": giv.ActionUpdateFunc(func(ari interface{}, act *gi.Action) {
					act.SetActiveState(AvailRegistersChanged && ari.(*Registers) == &AvailRegisters)
				}),
			}},
			{"sep-file", ki.BlankProp{}},
			{"OpenJSON", ki.Props{
				"label":    "Open from file",
				"desc":     "You can save and open named registers to / from files to share, experiment, transfer, etc",
				"shortcut": "Command+O",
				"Args": ki.PropSlice{
					{"File Name", ki.Props{
						"ext": ".json",
					}},
				},
			}},
			{"SaveJSON", ki.Props{
				"label": "Save to file",
				"desc":  "You can save and open named registers to / from files to share, experiment, transfer, etc",
				"Args": ki.PropSlice{
					{"File Name", ki.Props{
						"ext": ".json",
					}},
				},
			}},
		}},
		{"Edit", "Copy Cut Paste Dupe"},
		{"Window", "Windows"},
	},
	"ToolBar": ki.PropSlice{
		{"SavePrefs", ki.Props{
			"desc": "saves Registers to App standard prefs directory, in file registers_prefs.json, which will be loaded automatically at startup)",
			"icon": "file-save",
			"updtfunc": giv.ActionUpdateFunc(func(ari interface{}, act *gi.Action) {
				act.SetActiveState(AvailRegistersChanged && ari.(*Registers) == &AvailRegisters)
			}),
		}},
		{"sep-file", ki.BlankProp{}},
		{"OpenJSON", ki.Props{
			"label": "Open from file",
			"icon":  "file-open",
			"desc":  "You can save and open named registers to / from files to share, experiment, transfer, etc",
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"ext": ".json",
				}},
			},
		}},
		{"SaveJSON", ki.Props{
			"label": "Save to file",
			"icon":  "file-save",
			"desc":  "You can save and open named registers to / from files to share, experiment, transfer, etc",
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"ext": ".json",
				}},
			},
		}},
	},
}
