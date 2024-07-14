// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"log"
	"path/filepath"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/iox/tomlx"
	"cogentcore.org/core/core"
)

// LangOpts defines options associated with a given language / file format
// only languages in fileinfo.Known list are supported..
type LangOpts struct {

	// command(s) to run after a file of this type is saved
	PostSaveCmds CmdNames
}

// Languages is a map of language options
type Languages map[fileinfo.Known]*LangOpts

// AvailableLanguages is the current set of language options -- can be
// loaded / saved / edited with settings.  This is set to [StandardLanguages] at
// startup.
var AvailableLanguages Languages

func init() {
	AvailableLanguages.CopyFrom(StandardLanguages)
}

// Validate checks to make sure post save command names exist, issuing
// warnings to log for those that don't
func (lt Languages) Validate() bool {
	ok := true
	for _, lr := range lt {
		for _, cmdnm := range lr.PostSaveCmds {
			if !cmdnm.IsValid() {
				log.Printf("code.Langs Validate: post-save command: %v not found on current AvailCmds list\n", cmdnm)
				ok = false
			}
		}
	}
	return ok
}

// LanguageSettingsFilename is the name of the settings file in the app settings
// directory for saving / loading the default [AvailableLanguages] languages list
var LanguageSettingsFilename = "language-settings.toml"

// Open opens languages from a toml-formatted file.
func (lt *Languages) Open(filename core.Filename) error {
	*lt = make(Languages) // reset
	return tomlx.Open(lt, string(filename))
}

// Save saves languages to a toml-formatted file.
func (lt *Languages) Save(filename core.Filename) error { //types:add
	return tomlx.Save(lt, string(filename))
}

// OpenSettings opens the Langs from the app settings directory,
// using [LanguageSettingsFilename].
func (lt *Languages) OpenSettings() error { //types:add
	pdir := core.TheApp.AppDataDir()
	pnm := filepath.Join(pdir, LanguageSettingsFilename)
	AvailableLanguagesChanged = false
	return lt.Open(core.Filename(pnm))
}

// SaveSettings saves the Langs to the app settings directory,
// using [LanguageSettingsFilename].
func (lt *Languages) SaveSettings() error { //types:add
	pdir := core.TheApp.AppDataDir()
	pnm := filepath.Join(pdir, LanguageSettingsFilename)
	AvailableLanguagesChanged = false
	return lt.Save(core.Filename(pnm))
}

// CopyFrom copies languages from given other map
func (lt *Languages) CopyFrom(cp Languages) {
	*lt = make(Languages, len(cp)) // reset
	for ky, val := range cp {
		(*lt)[ky] = val
	}
}

// RevertToStandard reverts this map to using the StdLangs that are compiled into
// the program and have all the lastest standards.
func (lt *Languages) RevertToStandard() { //types:add
	lt.CopyFrom(StandardLanguages)
	AvailableLanguagesChanged = true
}

// ViewStandard shows the standard languages that are compiled into the program and have
// all the lastest standards.  Useful for comparing against custom lists.
func (lt *Languages) ViewStandard() { //types:add
	LanguagesView(&StandardLanguages)
}

// AvailableLanguagesChanged is used to update core.LangsView toolbars via
// following menu, toolbar properties update methods -- not accurate if editing any
// other map but works for now..
var AvailableLanguagesChanged = false

// StandardLanguages is the original compiled-in set of standard language options.
var StandardLanguages = Languages{
	fileinfo.Go: {CmdNames{"Go: Imports File"}},
}
