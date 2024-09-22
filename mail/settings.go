// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"path/filepath"
	"slices"
	"strings"

	"cogentcore.org/core/core"
)

func init() {
	core.AllSettings = slices.Insert(core.AllSettings, 1, core.Settings(Settings))
}

// Settings is the currently active global Cogent Mail settings instance.
var Settings = &SettingsData{
	SettingsBase: core.SettingsBase{
		Name: "Mail",
		File: filepath.Join(core.TheApp.DataDir(), "Cogent Mail", "settings.toml"),
	},
}

// SettingsData is the data type for the global Cogent Mail settings.
type SettingsData struct { //types:add
	core.SettingsBase

	// Accounts are the email accounts the user is signed into.
	Accounts []string
}

// friendlyLabelName converts the given label name to a user-friendly version.
func friendlyLabelName(name string) string {
	if f, ok := friendlyLabelNames[name]; ok {
		return f
	}
	name = strings.TrimPrefix(name, "[Gmail]/")
	return name
}

var friendlyLabelNames = map[string]string{
	"INBOX":             "Inbox",
	"[Gmail]/Sent Mail": "Sent",
}

// skipLabels are a temporary set of labels that should not be cached or displayed.
// TODO: figure out a better approach to this.
var skipLabels = map[string]bool{
	"[Gmail]":           true,
	"[Gmail]/All Mail":  true,
	"[Gmail]/Important": true,
	"[Gmail]/Starred":   true,
}
