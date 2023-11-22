// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"log"
	"path/filepath"

	"goki.dev/gi/v2/gi"
	"goki.dev/grows/jsons"
	"goki.dev/pi/v2/filecat"
)

// LangOpts defines options associated with a given language / file format
// only languages in filecat.Supported list are supported..
type LangOpts struct {

	// command(s) to run after a file of this type is saved
	PostSaveCmds CmdNames
}

// Langs is a map of language options
type Langs map[filecat.Supported]*LangOpts

// AvailLangs is the current set of language options -- can be
// loaded / saved / edited with preferences.  This is set to StdLangs at
// startup.
var AvailLangs Langs

func init() {
	AvailLangs.CopyFrom(StdLangs)
}

// Validate checks to make sure post save command names exist, issuing
// warnings to log for those that don't
func (lt Langs) Validate() bool {
	ok := true
	for _, lr := range lt {
		for _, cmdnm := range lr.PostSaveCmds {
			if !cmdnm.IsValid() {
				log.Printf("gide.Langs Validate: post-save command: %v not found on current AvailCmds list\n", cmdnm)
				ok = false
			}
		}
	}
	return ok
}

// PrefsLangsFileName is the name of the preferences file in App prefs
// directory for saving / loading the default AvailLangs languages list
var PrefsLangsFileName = "lang_prefs.json"

// OpenJSON opens languages from a JSON-formatted file.
func (lt *Langs) OpenJSON(filename gi.FileName) error {
	*lt = make(Langs) // reset
	return jsons.Open(lt, string(filename))
}

// SaveJSON saves languages to a JSON-formatted file.
func (lt *Langs) SaveJSON(filename gi.FileName) error { //gti:add
	return jsons.Save(lt, string(filename))
}

// OpenPrefs opens Langs from App standard prefs directory, using PrefsLangsFileName
func (lt *Langs) OpenPrefs() error { //gti:add
	pdir := gi.AppPrefsDir()
	pnm := filepath.Join(pdir, PrefsLangsFileName)
	AvailLangsChanged = false
	return lt.OpenJSON(gi.FileName(pnm))
}

// SavePrefs saves Langs to App standard prefs directory, using PrefsLangsFileName
func (lt *Langs) SavePrefs() error { //gti:add
	pdir := gi.AppPrefsDir()
	pnm := filepath.Join(pdir, PrefsLangsFileName)
	AvailLangsChanged = false
	return lt.SaveJSON(gi.FileName(pnm))
}

// CopyFrom copies languages from given other map
func (lt *Langs) CopyFrom(cp Langs) {
	*lt = make(Langs, len(cp)) // reset
	for ky, val := range cp {
		(*lt)[ky] = val
	}
}

// RevertToStd reverts this map to using the StdLangs that are compiled into
// the program and have all the lastest standards.
func (lt *Langs) RevertToStd() { //gti:add
	lt.CopyFrom(StdLangs)
	AvailLangsChanged = true
}

// ViewStd shows the standard langs that are compiled into the program and have
// all the lastest standards.  Useful for comparing against custom lists.
func (lt *Langs) ViewStd() { //gti:add
	LangsView(&StdLangs)
}

// AvailLangsChanged is used to update giv.LangsView toolbars via
// following menu, toolbar props update methods -- not accurate if editing any
// other map but works for now..
var AvailLangsChanged = false

/*
// LangsProps define the Toolbar and MenuBar for TableView of Langs, e.g., giv.LangsView
var LangsProps = ki.Props{
	"MainMenu": ki.PropSlice{
		{"AppMenu", ki.BlankProp{}},
		{"File", ki.PropSlice{
			{"OpenPrefs", ki.Props{}},
			{"SavePrefs", ki.Props{
				"shortcut": "Command+S",
				"updtfunc": giv.ActionUpdateFunc(func(lti any, act *gi.Button) {
					act.SetActiveState(AvailLangsChanged && lti.(*Langs) == &AvailLangs)
				}),
			}},
			{"sep-file", ki.BlankProp{}},
			{"OpenJSON", ki.Props{
				"label":    "Open from file",
				"desc":     "You can save and open language options to / from files to share, experiment, transfer, etc",
				"shortcut": "Command+O",
				"Args": ki.PropSlice{
					{"File Name", ki.Props{
						"ext": ".json",
					}},
				},
			}},
			{"SaveJSON", ki.Props{
				"label": "Save to file",
				"desc":  "You can save and open language options to / from files to share, experiment, transfer, etc",
				"Args": ki.PropSlice{
					{"File Name", ki.Props{
						"ext": ".json",
					}},
				},
			}},
			{"RevertToStd", ki.Props{
				"desc":    "This reverts the language options to using the StdLangs that are compiled into the program and have all the lastest standards. <b>Your current edits will be lost if you proceed!</b>  Continue?",
				"confirm": true,
			}},
		}},
		{"Edit", "Copy Cut Paste Dupe"},
		{"Window", "Windows"},
	},
	"Toolbar": ki.PropSlice{
		{"SavePrefs", ki.Props{
			"desc": "saves Langs to App standard prefs directory, in file lang_prefs.json, which will be loaded automatically at startup if prefs SaveLangs is checked (should be if you're using custom language options)",
			"icon": "file-save",
			"updtfunc": giv.ActionUpdateFunc(func(lti any, act *gi.Button) {
				act.SetActiveState(AvailLangsChanged && lti.(*Langs) == &AvailLangs)
			}),
		}},
		{"sep-file", ki.BlankProp{}},
		{"OpenJSON", ki.Props{
			"label": "Open from file",
			"icon":  "file-open",
			"desc":  "You can save and open language options to / from files to share, experiment, transfer, etc",
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"ext": ".json",
				}},
			},
		}},
		{"SaveJSON", ki.Props{
			"label": "Save to file",
			"icon":  "file-save",
			"desc":  "You can save and open language options to / from files to share, experiment, transfer, etc",
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"ext": ".json",
				}},
			},
		}},
		{"sep-std", ki.BlankProp{}},
		{"ViewStd", ki.Props{
			"desc": "Shows the standard language options that are compiled into the program and have all the latest changes.  Useful for comparing against custom langs.",
			"updtfunc": giv.ActionUpdateFunc(func(lti any, act *gi.Button) {
				act.SetActiveState(lti.(*Langs) != &StdLangs)
			}),
		}},
		{"RevertToStd", ki.Props{
			"icon":    "update",
			"desc":    "This reverts the language options to using the StdLangs that are compiled into the program and have all the lastest standards.  <b>Your current edits will be lost if you proceed!</b>  Continue?",
			"confirm": true,
			"updtfunc": giv.ActionUpdateFunc(func(lti any, act *gi.Button) {
				act.SetActiveState(lti.(*Langs) != &StdLangs)
			}),
		}},
	},
}
*/

// StdLangs is the original compiled-in set of standard language options.
var StdLangs = Langs{
	filecat.Go: {CmdNames{"Go: Imports File"}},
}
