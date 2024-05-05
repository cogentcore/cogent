// Copyright (c) 2023, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"path/filepath"

	"cogentcore.org/core/core"
)

// Settings is the currently active global Cogent Mail settings instance.
var Settings = &SettingsData{
	SettingsBase: core.SettingsBase{
		Name: "Mail",
		File: filepath.Join(core.TheApp.AppDataDir(), "settings.toml"),
	},
}

// SettingsData is the data type for the global Cogent Mail settings.
type SettingsData struct { //types:add
	core.SettingsBase

	// Accounts are the email accounts the user is signed into.
	Accounts []string
}
