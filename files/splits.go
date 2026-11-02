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

// applySplits applies the layout of the [Settings] to the splits.
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
	sv.SetSplits(sp.Splits[:]...)
	sv.Update()
}
