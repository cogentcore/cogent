// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lists

import (
	"path/filepath"

	"cogentcore.org/core/core"
)

func init() {
	core.TheApp.SetName("Cogent Lists")
	Settings.File = filepath.Join(core.TheApp.AppDataDir(), "settings.toml") // makes the directory
	core.AddAppSettings(Settings)
}

// Settings are the settings of Cogent Lists.
var Settings = &SettingsData{
	Name: "Lists",
}

// SettingsData is the data type of the [Settings].
type SettingsData struct { //types:add
	core.SettingsBase

	// Splits are the proportions of the tree of lists and the current list,
	// as last set by moving them, which are restored at startup.
	Splits []float32 `display:"-"`

	// Sync has the settings for syncing the lists with a GitHub
	// repository, which makes them the same on all of the devices,
	// with the history of all of the changes in the repository.
	Sync SyncSettings
}

// Apply applies the settings, after they are edited.
func (sd *SettingsData) Apply() {
	if theLists != nil && theLists.Scene != nil {
		theLists.settingsApplied()
	}
}

// theLists is the app, for applying the settings.
var theLists *Lists

// settingsApplied updates the lists after the settings are applied,
// as for syncing with another repository.
func (ls *Lists) settingsApplied() {
	ls.Scene.Update() // for the sync button in the toolbar
	ls.Sync()
}

// splits returns the [core.Splits] of the tree of lists and the current list.
func (ls *Lists) splits() *core.Splits {
	if !ls.HasChildren() {
		return nil
	}
	sv, _ := ls.Child(0).(*core.Splits)
	return sv
}

// applySplits applies the proportions of the splits in the [Settings].
func (ls *Lists) applySplits() {
	if sv := ls.splits(); sv != nil && len(Settings.Splits) == 2 {
		sv.SetSplits(Settings.Splits...)
	}
}

// saveSplits saves the current proportions of the splits in the [Settings].
func (ls *Lists) saveSplits() {
	if sv := ls.splits(); sv != nil {
		Settings.Splits = sv.Splits()
	}
	core.ErrorSnackbar(ls, core.SaveSettings(Settings), "Error saving settings")
}
