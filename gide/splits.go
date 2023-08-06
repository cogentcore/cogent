// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/oswin"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/ki/sliceclone"
)

// Split is a named splitter configuration
type Split struct {

	// name of splitter config
	Name string `desc:"name of splitter config"`

	// brief description
	Desc string `desc:"brief description"`

	// [min: 0] [max: 1] [step: .05] splitter panel proportions
	Splits []float32 `min:"0" max:"1" step:".05" fixed-len:"4" desc:"splitter panel proportions"`
}

// Label satisfies the Labeler interface
func (sp Split) Label() string {
	return sp.Name
}

// SaveSplits saves given splits to this setting -- must use copy!
func (lt *Split) SaveSplits(sp []float32) {
	copy(lt.Splits, sp)
}

// Splits is a list of named splitter configurations
type Splits []*Split

var KiT_Splits = kit.Types.AddType(&Splits{}, SplitsProps)

// SplitName has an associated ValueView for selecting from the list of
// available named splits
type SplitName string

// AvailSplits are available named splitter settings.  can be loaded / saved /
// edited with preferences.  This is set to StdSplits at startup.
var AvailSplits Splits

// AvailSplitNames are the names of the current AvailSplits -- used for some choosers
var AvailSplitNames []string

func init() {
	AvailSplits.CopyFrom(StdSplits)
	AvailSplitNames = AvailSplits.Names()
}

// SplitByName returns a named split and index by name -- returns false and emits a
// message to stdout if not found
func (lt *Splits) SplitByName(name SplitName) (*Split, int, bool) {
	if name == "" {
		return nil, -1, false
	}
	for i, sp := range *lt {
		if sp.Name == string(name) {
			return sp, i, true
		}
	}
	fmt.Printf("gide.SplitByName: split named: %v not found\n", name)
	return nil, -1, false
}

// Add adds a new splitter setting, returns split and index
func (lt *Splits) Add(name, desc string, splits []float32) (*Split, int) {
	sp := &Split{Name: name, Desc: desc, Splits: sliceclone.Float32(splits)}
	*lt = append(*lt, sp)
	return sp, len(*lt) - 1
}

// Names returns a slice of current names
func (lt *Splits) Names() []string {
	nms := make([]string, len(*lt))
	for i, sp := range *lt {
		nms[i] = sp.Name
	}
	return nms
}

// PrefsSplitsFileName is the name of the preferences file in App prefs
// directory for saving / loading the default AvailSplits
var PrefsSplitsFileName = "splits_prefs.json"

// FixLen ensures that there are exactly 4 splits in each
func (lt *Splits) FixLen() {
	for _, sp := range *lt {
		ln := len(sp.Splits)
		if ln > 4 {
			sp.Splits = sp.Splits[:4]
		} else if ln < 4 {
			sp.Splits = append(sp.Splits, make([]float32, 4-ln)...)
		}
	}
}

// OpenJSON opens named splits from a JSON-formatted file.
func (lt *Splits) OpenJSON(filename gi.FileName) error {
	b, err := ioutil.ReadFile(string(filename))
	if err != nil {
		// gi.PromptDialog(nil, gi.DlgOpts{Title: "File Not Found", Prompt: err.Error()}, gi.AddOk, gi.NoCancel, nil, nil)
		// log.Println(err)
		return err
	}
	*lt = make(Splits, 0, 10) // reset
	err = json.Unmarshal(b, lt)
	if err != nil {
		return err
	}
	lt.FixLen()
	return err
}

// SaveJSON saves named splits to a JSON-formatted file.
func (lt *Splits) SaveJSON(filename gi.FileName) error {
	b, err := json.MarshalIndent(lt, "", "  ")
	if err != nil {
		log.Println(err) // unlikely
		return err
	}
	err = ioutil.WriteFile(string(filename), b, 0644)
	if err != nil {
		gi.PromptDialog(nil, gi.DlgOpts{Title: "Could not Save to File", Prompt: err.Error()}, gi.AddOk, gi.NoCancel, nil, nil)
		log.Println(err)
	}
	return err
}

// OpenPrefs opens Splits from App standard prefs directory, using PrefSplitsFileName
func (lt *Splits) OpenPrefs() error {
	pdir := oswin.TheApp.AppPrefsDir()
	pnm := filepath.Join(pdir, PrefsSplitsFileName)
	AvailSplitsChanged = false
	err := lt.OpenJSON(gi.FileName(pnm))
	if err == nil {
		AvailSplitNames = lt.Names()
	}
	return err
}

// SavePrefs saves Splits to App standard prefs directory, using PrefSplitsFileName
func (lt *Splits) SavePrefs() error {
	lt.FixLen()
	pdir := oswin.TheApp.AppPrefsDir()
	pnm := filepath.Join(pdir, PrefsSplitsFileName)
	AvailSplitsChanged = false
	AvailSplitNames = lt.Names()
	return lt.SaveJSON(gi.FileName(pnm))
}

// CopyFrom copies named splits from given other map
func (lt *Splits) CopyFrom(cp Splits) {
	*lt = make(Splits, 0, len(cp)) // reset
	b, err := json.Marshal(cp)
	if err != nil {
		fmt.Printf("json err: %v\n", err.Error())
	}
	json.Unmarshal(b, lt)
	lt.FixLen()
}

// AvailSplitsChanged is used to update toolbars via following menu, toolbar
// props update methods -- not accurate if editing any other map but works for
// now..
var AvailSplitsChanged = false

// SplitsProps define the ToolBar and MenuBar for TableView of Splits
var SplitsProps = ki.Props{
	"MainMenu": ki.PropSlice{
		{"AppMenu", ki.BlankProp{}},
		{"File", ki.PropSlice{
			{"OpenPrefs", ki.Props{}},
			{"SavePrefs", ki.Props{
				"shortcut": "Command+S",
				"updtfunc": giv.ActionUpdateFunc(func(spi any, act *gi.Action) {
					act.SetActiveState(AvailSplitsChanged && spi.(*Splits) == &AvailSplits)
				}),
			}},
			{"sep-file", ki.BlankProp{}},
			{"OpenJSON", ki.Props{
				"label":    "Open from file",
				"desc":     "You can save and open named splits to / from files to share, experiment, transfer, etc",
				"shortcut": "Command+O",
				"Args": ki.PropSlice{
					{"File Name", ki.Props{
						"ext": ".json",
					}},
				},
			}},
			{"SaveJSON", ki.Props{
				"label": "Save to file",
				"desc":  "You can save and open named splits to / from files to share, experiment, transfer, etc",
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
			"desc": "saves Splits to App standard prefs directory, in file splits_prefs.json, which will be loaded automatically at startup)",
			"icon": "file-save",
			"updtfunc": giv.ActionUpdateFunc(func(spi any, act *gi.Action) {
				act.SetActiveState(AvailSplitsChanged && spi.(*Splits) == &AvailSplits)
			}),
		}},
		{"sep-file", ki.BlankProp{}},
		{"OpenJSON", ki.Props{
			"label": "Open from file",
			"icon":  "file-open",
			"desc":  "You can save and open named splits to / from files to share, experiment, transfer, etc",
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"ext": ".json",
				}},
			},
		}},
		{"SaveJSON", ki.Props{
			"label": "Save to file",
			"icon":  "file-save",
			"desc":  "You can save and open named splits to / from files to share, experiment, transfer, etc",
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"ext": ".json",
				}},
			},
		}},
	},
}

// StdSplits is the original compiled-in set of standard named splits.
var StdSplits = Splits{
	{"Code", "2 text views, tabs", []float32{.1, .325, .325, .25}},
	{"Small", "1 text view, tabs", []float32{.1, .5, 0, .4}},
	{"BigTabs", "1 text view, big tabs", []float32{.1, .3, 0, .6}},
}
