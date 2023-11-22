// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"

	"goki.dev/gi/v2/gi"
	"goki.dev/grows/jsons"
	"goki.dev/grr"
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

func (sn SplitName) String() string {
	return string(sn)
}

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
func (lt *Splits) OpenJSON(filename gi.FileName) error { //gti:add
	*lt = make(Splits, 0, 10) // reset
	err := grr.Log0(jsons.Open(lt, string(filename)))
	lt.FixLen()
	return err
}

// SaveJSON saves named splits to a JSON-formatted file.
func (lt *Splits) SaveJSON(filename gi.FileName) error { //gti:add
	return grr.Log0(jsons.Save(lt, string(filename)))
}

// OpenPrefs opens Splits from App standard prefs directory, using PrefSplitsFileName
func (lt *Splits) OpenPrefs() error { //gti:add
	pdir := gi.AppPrefsDir()
	pnm := filepath.Join(pdir, PrefsSplitsFileName)
	AvailSplitsChanged = false
	err := lt.OpenJSON(gi.FileName(pnm))
	if err == nil {
		AvailSplitNames = lt.Names()
	}
	return err
}

// SavePrefs saves Splits to App standard prefs directory, using PrefSplitsFileName
func (lt *Splits) SavePrefs() error { //gti:add
	lt.FixLen()
	pdir := gi.AppPrefsDir()
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

// StdSplits is the original compiled-in set of standard named splits.
var StdSplits = Splits{
	{"Code", "2 text views, tabs", []float32{.1, .325, .325, .25}},
	{"Small", "1 text view, tabs", []float32{.1, .5, 0, .4}},
	{"BigTabs", "1 text view, big tabs", []float32{.1, .3, 0, .6}},
}
