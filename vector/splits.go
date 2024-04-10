// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vector

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"

	"cogentcore.org/core/core"
	"cogentcore.org/core/grows/jsons"
	"cogentcore.org/core/grr"
	"cogentcore.org/core/xgo/dirs"
)

// Split is a named splitter configuration
type Split struct {

	// name of splitter config
	Name string

	// brief description
	Desc string

	// splitter panel proportions
	Splits []float32 `min:"0" max:"1" step:".05" fixed-len:"4"`
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

// SplitName has an associated ValueView for selecting from the list of
// available named splits
type SplitName string

// AvailableSplits are available named splitter settings.  can be loaded / saved /
// edited with settings.  This is set to StandardSplits at startup.
var AvailableSplits Splits

// AvailableSplitNames are the names of the current AvailableSplits -- used for some choosers
var AvailableSplitNames []string

func init() {
	AvailableSplits.CopyFrom(StandardSplits)
	AvailableSplitNames = AvailableSplits.Names()
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
	fmt.Printf("code.SplitByName: split named: %v not found\n", name)
	return nil, -1, false
}

// Add adds a new splitter setting, returns split and index
func (lt *Splits) Add(name, desc string, splits []float32) (*Split, int) {
	sp := &Split{Name: name, Desc: desc, Splits: slices.Clone(splits)}
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

// SplitsSettingsFilename is the name of the settings file in App prefs
// directory for saving / loading the default AvailSplits
var SplitsSettingsFilename = "splits-settings.json"

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

// Open opens named splits from a json-formatted file.
func (lt *Splits) Open(filename core.Filename) error { //gti:add
	if grr.Ignore1(dirs.FileExists(string(filename))) {
		*lt = make(Splits, 0, 10) // reset
		err := grr.Log(jsons.Open(lt, string(filename)))
		lt.FixLen()
		return err
	}
	return nil
}

// Save saves named splits to a json-formatted file.
func (lt *Splits) Save(filename core.Filename) error { //gti:add
	return grr.Log(jsons.Save(lt, string(filename)))
}

// OpenSettings opens Splits from App standard prefs directory, using PrefSplitsFilename
func (lt *Splits) OpenSettings() error { //gti:add
	pdir := core.TheApp.AppDataDir()
	pnm := filepath.Join(pdir, SplitsSettingsFilename)
	AvailableSplitsChanged = false
	err := lt.Open(core.Filename(pnm))
	if err == nil {
		AvailableSplitNames = lt.Names()
	}
	return err
}

// SaveSettings saves Splits to App standard prefs directory, using PrefSplitsFilename
func (lt *Splits) SaveSettings() error { //gti:add
	lt.FixLen()
	pdir := core.TheApp.AppDataDir()
	pnm := filepath.Join(pdir, SplitsSettingsFilename)
	AvailableSplitsChanged = false
	AvailableSplitNames = lt.Names()
	return lt.Save(core.Filename(pnm))
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

// AvailableSplitsChanged is used to update toolbars via following menu, toolbar
// properties update methods -- not accurate if editing any other map but works for
// now..
var AvailableSplitsChanged = false

// StandardSplits is the original compiled-in set of standard named splits.
var StandardSplits = Splits{
	{"Code", "2 text views, tabs", []float32{.1, .325, .325, .25}},
	{"Small", "1 text view, tabs", []float32{.1, .5, 0, .4}},
	{"BigTabs", "1 text view, big tabs", []float32{.1, .3, 0, .6}},
	{"Debug", "big command panel for debugging", []float32{0.1, 0.29539588, 0.2949658, 0.30963832}},
}
