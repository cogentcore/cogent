// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"strings"

	"cogentcore.org/core/core"
	"cogentcore.org/core/htmlcore"
)

// Settings are the settings of the web browser, which an app adds to
// [core.AllSettings] (with [core.AddAppSettings]) before it makes its
// first body, for them to be loaded and shown in the settings window.
var Settings = &SettingsData{
	SettingsBase: core.SettingsBase{Name: "Web"},
}

// SettingsData is the data type of the [Settings].
type SettingsData struct { //types:add
	core.SettingsBase

	// UserAgent is the User-Agent header that is sent with the requests
	// of pages, and that their scripts get as navigator.userAgent: the
	// requests made after it is changed send it, and the scripts of the
	// pages opened after then get it. Clear it for the default one.
	UserAgent string

	// VerboseConsole makes the console show what the scripts of pages do,
	// for finding why a page does not work (as a captcha that only says
	// that something went wrong): the requests that they make and the
	// scripts that they load, with their statuses and sizes, the messages
	// that they post to other windows (and those that are not delivered),
	// and the properties that they get that the browser does not have,
	// once each. It applies to pages opened (or reloaded) after it is set.
	VerboseConsole bool
}

// defaultUserAgent is the default User-Agent of the requests of pages.
var defaultUserAgent = htmlcore.UserAgent

func (se *SettingsData) Defaults() {
	se.UserAgent = defaultUserAgent
}

// Apply applies the settings.
func (se *SettingsData) Apply() {
	se.UserAgent = strings.TrimSpace(se.UserAgent)
	if se.UserAgent == "" {
		se.UserAgent = defaultUserAgent
	}
	htmlcore.UserAgent = se.UserAgent
}
