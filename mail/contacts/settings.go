// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package contacts

import (
	"os"
	"path/filepath"
	"sync"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/core"
)

// SettingsData are the settings of the contacts, as the layout of their
// browser, used by the apps that show them (Cogent Contacts and Mail), and
// saved in the data folder of Cogent Contacts.
type SettingsData struct { //types:add
	core.SettingsBase

	// Splits are the proportions of the list of the contacts and the
	// editor of the selected one in the browser, as set by the user.
	Splits []float32 `display:"-"`
}

// Settings are the settings of the contacts (see [SettingsData]).
var Settings = &SettingsData{
	SettingsBase: core.SettingsBase{
		Name: "Contacts",
		File: filepath.Join(core.TheApp.DataDir(), "Cogent Contacts", "settings.toml"),
	},
}

// loadSettingsOnce loads the [Settings] once, by the first browser.
var loadSettingsOnce sync.Once

// loadSettings loads the [Settings], if they have not been loaded.
func loadSettings() {
	loadSettingsOnce.Do(func() {
		errors.Log(core.LoadSettings(Settings))
	})
}

// saveSettings saves the [Settings], logging any error.
func saveSettings() {
	errors.Log(os.MkdirAll(filepath.Dir(Settings.Filename()), 0o700))
	errors.Log(core.SaveSettings(Settings))
}
