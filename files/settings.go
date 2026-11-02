// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"path/filepath"

	"cogentcore.org/core/core"
)

func init() {
	core.TheApp.SetName("Cogent Files")
	core.AddAppSettings(Settings)
}

// Settings are the settings of Cogent Files.
var Settings = &SettingsData{
	Name: "Files",
	File: filepath.Join(core.TheApp.DataDir(), "Cogent Files", "settings.toml"),
}

// SettingsData is the data type of the [Settings].
type SettingsData struct { //types:add
	core.SettingsBase

	// Splits is the layout of the panels: the file tree, the directory tabs,
	// and the command tabs, which are to the right of the directory tabs,
	// or under them in a compact layout.
	Splits SplitName `default:"Wide"`

	// ShowHidden shows the hidden files, whose names start with a dot.
	ShowHidden bool

	// Terminal is the command to run in terminals, with its arguments,
	// such as goal; the default is the user's shell.
	Terminal string
}

// Apply applies the settings to the open [Files].
func (se *SettingsData) Apply() {
	for _, fl := range openFiles {
		fl.applySplits()
		if fl.Tree != nil && fl.Root != "" {
			fl.Tree.OpenPath(fl.Root) // for ShowHidden
		}
		for _, dt := range fl.DirTables() {
			dt.Update()
		}
	}
}
