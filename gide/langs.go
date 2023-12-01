// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"log"
	"path/filepath"

	"goki.dev/fi"
	"goki.dev/gi/v2/gi"
	"goki.dev/grows/tomls"
)

// LangOpts defines options associated with a given language / file format
// only languages in fi.Supported list are supported..
type LangOpts struct {

	// command(s) to run after a file of this type is saved
	PostSaveCmds CmdNames
}

// Langs is a map of language options
type Langs map[fi.Supported]*LangOpts

// AvailLangs is the current set of language options -- can be
// loaded / saved / edited with preferences.  This is set to StdLangs at
// startup.
var AvailLangs Langs

func init() {
	AvailLangs.CopyFrom(StdLangs)
}

// Validate checks to make sure post save command names exist, issuing
// warnings to log for those that don't
func (lt Langs) Validate() bool {
	ok := true
	for _, lr := range lt {
		for _, cmdnm := range lr.PostSaveCmds {
			if !cmdnm.IsValid() {
				log.Printf("gide.Langs Validate: post-save command: %v not found on current AvailCmds list\n", cmdnm)
				ok = false
			}
		}
	}
	return ok
}

// PrefsLangsFileName is the name of the preferences file in App prefs
// directory for saving / loading the default AvailLangs languages list
var PrefsLangsFileName = "lang_prefs.toml"

// Open opens languages from a toml-formatted file.
func (lt *Langs) Open(filename gi.FileName) error {
	*lt = make(Langs) // reset
	return tomls.Open(lt, string(filename))
}

// Save saves languages to a toml-formatted file.
func (lt *Langs) Save(filename gi.FileName) error { //gti:add
	return tomls.Save(lt, string(filename))
}

// OpenPrefs opens Langs from App standard prefs directory, using PrefsLangsFileName
func (lt *Langs) OpenPrefs() error { //gti:add
	pdir := gi.AppPrefsDir()
	pnm := filepath.Join(pdir, PrefsLangsFileName)
	AvailLangsChanged = false
	return lt.Open(gi.FileName(pnm))
}

// SavePrefs saves Langs to App standard prefs directory, using PrefsLangsFileName
func (lt *Langs) SavePrefs() error { //gti:add
	pdir := gi.AppPrefsDir()
	pnm := filepath.Join(pdir, PrefsLangsFileName)
	AvailLangsChanged = false
	return lt.Save(gi.FileName(pnm))
}

// CopyFrom copies languages from given other map
func (lt *Langs) CopyFrom(cp Langs) {
	*lt = make(Langs, len(cp)) // reset
	for ky, val := range cp {
		(*lt)[ky] = val
	}
}

// RevertToStd reverts this map to using the StdLangs that are compiled into
// the program and have all the lastest standards.
func (lt *Langs) RevertToStd() { //gti:add
	lt.CopyFrom(StdLangs)
	AvailLangsChanged = true
}

// ViewStd shows the standard langs that are compiled into the program and have
// all the lastest standards.  Useful for comparing against custom lists.
func (lt *Langs) ViewStd() { //gti:add
	LangsView(&StdLangs)
}

// AvailLangsChanged is used to update giv.LangsView toolbars via
// following menu, toolbar props update methods -- not accurate if editing any
// other map but works for now..
var AvailLangsChanged = false

// StdLangs is the original compiled-in set of standard language options.
var StdLangs = Langs{
	fi.Go: {CmdNames{"Go: Imports File"}},
}
