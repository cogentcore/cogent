// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"cogentcore.org/core/core"
)

// Split is a layout of the panels: the file tree, the directory tabs,
// and the command tabs, as in Cogent Code.
type Split struct {

	// Name is the name of the layout.
	Name string

	// Splits are the proportions of the file tree, the directory tabs,
	// and the command tabs. With TabsUnder, those of the directory tabs
	// and the command tabs are of the column that they share.
	Splits [3]float32

	// TabsUnder puts the command tabs under the directory tabs,
	// for a more compact layout, suitable for smaller displays.
	TabsUnder bool
}

// StandardSplits are the available layouts.
var StandardSplits = []Split{
	{"Wide", [3]float32{.15, .55, .3}, false},
	{"Compact", [3]float32{.2, .6, .4}, true},
	{"Files", [3]float32{.2, .8, 0}, false},
}

// SplitName is the name of one of the [StandardSplits].
type SplitName string

// Widget makes a chooser of the [StandardSplits].
func (sn SplitName) Widget() core.Value {
	return core.NewChooser().SetStrings(splitNames()...)
}

// splitNames returns the names of the [StandardSplits].
func splitNames() []string {
	names := make([]string, len(StandardSplits))
	for i, sp := range StandardSplits {
		names[i] = sp.Name
	}
	return names
}

// splitByName returns the split with the given name, or the first one.
func splitByName(name SplitName) Split {
	for _, sp := range StandardSplits {
		if sp.Name == string(name) {
			return sp
		}
	}
	return StandardSplits[0]
}

// treeSplits are the default proportions of the file tree and the favorites.
var treeSplits = []float32{.75, .25}

// applySplits applies the layout of the [Settings] to the splits:
// the Splits layout, with its SplitValues if set, and the TreeSplits.
func (fl *Files) applySplits() {
	sv := fl.Splits()
	if sv == nil {
		return
	}
	sp := splitByName(Settings.Splits)
	if sp.TabsUnder {
		sv.SetTiles(core.TileSpan, core.TileSplit)
	} else {
		sv.SetTiles(core.TileSpan, core.TileSpan, core.TileSpan)
	}
	if len(Settings.SplitValues) == len(sp.Splits) {
		sv.SetSplits(Settings.SplitValues...)
	} else {
		sv.SetSplits(sp.Splits[:]...)
	}
	sv.Update()
	if lt := fl.LeftSplits(); lt != nil {
		if len(Settings.TreeSplits) == len(treeSplits) {
			lt.SetSplits(Settings.TreeSplits...)
		} else {
			lt.SetSplits(treeSplits...)
		}
		lt.Update()
	}
}

// setSplits sets the Splits layout of the [Settings] to the given one,
// with its proportions, applies it, and saves the settings.
func (fl *Files) setSplits(name SplitName) {
	Settings.Splits = name
	Settings.SplitValues = nil
	fl.applySplits()
	fl.saveSplits()
}

// saveSplits saves the current proportions of the splits in the [Settings].
func (fl *Files) saveSplits() {
	if sv := fl.Splits(); sv != nil {
		Settings.SplitValues = sv.Splits()
	}
	if lt := fl.LeftSplits(); lt != nil {
		Settings.TreeSplits = lt.Splits()
	}
	core.ErrorSnackbar(fl, core.SaveSettings(Settings), "Error saving settings")
}
