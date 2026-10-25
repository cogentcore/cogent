// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"log"
	"path/filepath"
	"slices"
	"sync/atomic"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/iox/tomlx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/text/highlighting"
)

// LanguageOptions defines options associated with a given language / file format
// only languages in fileinfo.Known list are supported..
type LanguageOptions struct {

	// command(s) to run after a file of this type is saved
	PostSaveCmds CmdNames

	// Parser uses the built-in parser for this language, for highlighting,
	// completion, lookup, symbols, and indentation, instead of a language
	// server, when the project uses the LSP language backend. Other
	// languages use a language server when one is available, and otherwise
	// just chroma highlighting. With the Parser backend, the parser is used
	// for all languages that it supports.
	Parser bool
}

// Languages is a map of language options
type Languages map[fileinfo.Known]*LanguageOptions

// AvailableLanguages is the current set of language options -- can be
// loaded / saved / edited with settings.  This is set to [StandardLanguages] at
// startup.
var AvailableLanguages Languages

func init() {
	AvailableLanguages.CopyFrom(StandardLanguages)
	highlighting.ParseFilter = useParser
	updateParserLangs()
}

// parserLangs is a snapshot of the languages with [LanguageOptions.Parser]
// set, for [useParser], which runs on markup goroutines.
var parserLangs atomic.Pointer[map[fileinfo.Known]bool]

// parserBackend is whether any open project uses [LangBackendParser],
// in which case the parser is used for all the languages it supports.
var parserBackend atomic.Bool

// updateParserLangs updates the [parserLangs] snapshot from
// [AvailableLanguages].
func updateParserLangs() {
	pl := map[fileinfo.Known]bool{}
	for kn, lo := range AvailableLanguages {
		if lo != nil && lo.Parser {
			pl[kn] = true
		}
	}
	parserLangs.Store(&pl)
}

// parserLanguage returns whether [LanguageOptions.Parser] is set for
// the given language.
func parserLanguage(kn fileinfo.Known) bool {
	pl := parserLangs.Load()
	return pl != nil && (*pl)[kn]
}

// useParser is the [highlighting.ParseFilter], which determines whether
// the built-in parser is used for the given language.
func useParser(kn fileinfo.Known) bool {
	return parserBackend.Load() || parserLanguage(kn)
}

// UpdateParserUse applies changes in the use of the built-in parser,
// from the language backend of the projects and the Parser language
// options, to the files open in all Code windows.
// Any given Code windows are included even if they are not yet
// in [core.AllRenderWindows].
func UpdateParserUse(cvs ...*Code) {
	updateParserLangs()
	for _, win := range core.AllRenderWindows {
		if cv := CodeInScene(win.MainScene()); cv != nil && !slices.Contains(cvs, cv) {
			cvs = append(cvs, cv)
		}
	}
	pb := false
	for _, cv := range cvs {
		if cv.Settings.LangBackend == LangBackendParser {
			pb = true
		}
	}
	parserBackend.Store(pb)
	for _, cv := range cvs {
		cv.updateParserUse()
	}
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
	err := tomlx.Open(lt, string(filename))
	if lt == &AvailableLanguages {
		UpdateParserUse()
	}
	return err
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
		lo := *val // copy, so editing does not change cp
		lo.PostSaveCmds = slices.Clone(lo.PostSaveCmds)
		(*lt)[ky] = &lo
	}
}

// RevertToStandard reverts this map to using the StdLangs that are compiled into
// the program and have all the lastest standards.
func (lt *Languages) RevertToStandard() { //types:add
	lt.CopyFrom(StandardLanguages)
	AvailableLanguagesChanged = true
	if lt == &AvailableLanguages {
		UpdateParserUse()
	}
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
	fileinfo.Go:       {PostSaveCmds: CmdNames{"Go: Imports File"}},
	fileinfo.Markdown: {Parser: true},
	fileinfo.TeX:      {Parser: true},
}
