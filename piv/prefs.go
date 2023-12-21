// Copyright (c) 2018, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package piv

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"

	"goki.dev/gi/v2/gi"
	"goki.dev/pi/v2/parse"
)

// ProjPrefs are the preferences for saving for a project -- this IS the project file
type ProjPrefs struct {

	// filename for project (i.e, these preference)
	ProjFile gi.FileName

	// filename for parser
	ParserFile gi.FileName

	// the file for testing
	TestFile gi.FileName

	// the options for tracing parsing
	TraceOpts parse.TraceOpts
}

// Open open from  file
func (pf *ProjPrefs) Open(filename gi.FileName) error {
	b, err := os.ReadFile(string(filename))
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, pf)
	if err == nil {
		pf.ProjFile = filename
	}
	return err
}

// Save save to  file
func (pf *ProjPrefs) Save(filename gi.FileName) error {
	pf.ProjFile = filename
	b, err := json.MarshalIndent(pf, "", "  ")
	if err != nil {
		log.Println(err)
		return err
	}
	err = os.WriteFile(string(filename), b, 0644)
	if err != nil {
		log.Println(err)
	}
	return err
}

// InitPrefs is the overall init at startup for PiView project
func InitPrefs() {
	OpenPaths()
}

//////////////////////////////////////////////////////////////////////////////////////
//   Saved Projects / Paths

// SavedPaths is a slice of strings that are file paths
var SavedPaths gi.FilePaths

// SavedPathsFileName is the name of the saved file paths file in GoPi prefs directory
var SavedPathsFileName = "gopi_saved_paths.toml"

// SavePaths saves the active SavedPaths to prefs dir
func SavePaths() {
	pdir := gi.AppDataDir()
	pnm := filepath.Join(pdir, SavedPathsFileName)
	SavedPaths.Save(pnm)
}

// OpenPaths loads the active SavedPaths from prefs dir
func OpenPaths() {
	pdir := gi.AppDataDir()
	pnm := filepath.Join(pdir, SavedPathsFileName)
	SavedPaths.Open(pnm)
}
