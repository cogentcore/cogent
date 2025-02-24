// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package piv

/*

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"

	"cogentcore.org/core/core"
	"cogentcore.org/core/text/parse/parse"
)

// ProjSettings are the settings for saving for a project -- this IS the project file
type ProjSettings struct {

	// filename for project (i.e, these preference)
	ProjectFile core.Filename

	// filename for parser
	ParserFile core.Filename

	// the file for testing
	TestFile core.Filename

	// the options for tracing parsing
	TraceOpts parser.TraceOpts
}

// Open open from  file
func (pf *ProjSettings) Open(filename core.Filename) error {
	b, err := os.ReadFile(string(filename))
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, pf)
	if err == nil {
		pf.ProjectFile = filename
	}
	return err
}

// Save save to  file
func (pf *ProjSettings) Save(filename core.Filename) error {
	pf.ProjectFile = filename
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

// InitSettings is the overall init at startup for PiView project
func InitSettings() {
	OpenPaths()
}

//////////////////////////////////////////////////////////////////////////////////////
//   Saved Projects / Paths

// SavedPaths is a slice of strings that are file paths
var SavedPaths core.FilePaths

// SavedPathsFilename is the name of the saved file paths file in GoPi prefs directory
var SavedPathsFilename = "gopi_saved_paths.toml"

// SavePaths saves the active SavedPaths to prefs dir
func SavePaths() {
	pdir := core.AppDataDir()
	pnm := filepath.Join(pdir, SavedPathsFilename)
	SavedPaths.Save(pnm)
}

// OpenPaths loads the active SavedPaths from prefs dir
func OpenPaths() {
	pdir := core.AppDataDir()
	pnm := filepath.Join(pdir, SavedPathsFilename)
	SavedPaths.Open(pnm)
}

*/
