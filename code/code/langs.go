// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"log"
	"path/filepath"

	"cogentcore.org/core/fi"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/grows/tomls"
)

// LangOpts defines options associated with a given language / file format
// only languages in fi.Known list are supported..
type LangOpts struct {

	// command(s) to run after a file of this type is saved
	PostSaveCmds CmdNames
}

// Langs is a map of language options
type Langs map[fi.Known]*LangOpts

// AvailableLangs is the current set of language options -- can be
// loaded / saved / edited with settings.  This is set to StandardLangs at
// startup.
var AvailableLangs Langs

func init() {
	AvailableLangs.CopyFrom(StandardLangs)
}

// Validate checks to make sure post save command names exist, issuing
// warnings to log for those that don't
func (lt Langs) Validate() bool {
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

// LangSettingsFilename is the name of the settings file in the app settings
// directory for saving / loading the default AvailableLangs languages list
var LangSettingsFilename = "lang-settings.toml"

// Open opens languages from a toml-formatted file.
func (lt *Langs) Open(filename gi.Filename) error {
	*lt = make(Langs) // reset
	return tomls.Open(lt, string(filename))
}

// Save saves languages to a toml-formatted file.
func (lt *Langs) Save(filename gi.Filename) error { //gti:add
	return tomls.Save(lt, string(filename))
}

// OpenSettings opens the Langs from the app settings directory,
// using LangSettingsFilename.
func (lt *Langs) OpenSettings() error { //gti:add
	pdir := gi.TheApp.AppDataDir()
	pnm := filepath.Join(pdir, LangSettingsFilename)
	AvailableLangsChanged = false
	return lt.Open(gi.Filename(pnm))
}

// SaveSettings saves the Langs to the app settings directory,
// using LangSettingsFilename.
func (lt *Langs) SaveSettings() error { //gti:add
	pdir := gi.TheApp.AppDataDir()
	pnm := filepath.Join(pdir, LangSettingsFilename)
	AvailableLangsChanged = false
	return lt.Save(gi.Filename(pnm))
}

// CopyFrom copies languages from given other map
func (lt *Langs) CopyFrom(cp Langs) {
	*lt = make(Langs, len(cp)) // reset
	for ky, val := range cp {
		(*lt)[ky] = val
	}
}

// RevertToStandard reverts this map to using the StdLangs that are compiled into
// the program and have all the lastest standards.
func (lt *Langs) RevertToStandard() { //gti:add
	lt.CopyFrom(StandardLangs)
	AvailableLangsChanged = true
}

// ViewStandard shows the standard langs that are compiled into the program and have
// all the lastest standards.  Useful for comparing against custom lists.
func (lt *Langs) ViewStandard() { //gti:add
	LangsView(&StandardLangs)
}

// AvailableLangsChanged is used to update giv.LangsView toolbars via
// following menu, toolbar properties update methods -- not accurate if editing any
// other map but works for now..
var AvailableLangsChanged = false

// StandardLangs is the original compiled-in set of standard language options.
var StandardLangs = Langs{
	fi.Go: {CmdNames{"Go: Imports File"}},
}
