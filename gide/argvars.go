// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/goki/gi/giv"
	"github.com/goki/gi/oswin"
	"github.com/goki/ki/kit"
)

// ArgVarInfo has info about argument variables that fill in relevant values
// for commands, used in ArgVars list of variables
type ArgVarInfo struct {
	Desc string      `desc:"description of arg var"`
	Type ArgVarTypes `desc:"type of variable -- used for checking usage and other special features such as prompting"`
}

// ArgVars are variables that can be used for arguments to commands in CmdAndArgs
var ArgVars = map[string]ArgVarInfo{
	/// Current Filename
	"{FilePath}":       ArgVarInfo{"Current file name with full path.", ArgVarFile},
	"{FileName}":       ArgVarInfo{"Current file name only, without path.", ArgVarFile},
	"{FileExt}":        ArgVarInfo{"Extension of current file name.", ArgVarExt},
	"{FileExtLC}":      ArgVarInfo{"Extension of current file name, lowercase.", ArgVarExt},
	"{FileNameNoExt}":  ArgVarInfo{"Current file name without path and extension.", ArgVarFile},
	"{FileDir}":        ArgVarInfo{"Name only of current file's directory.", ArgVarDir},
	"{FileDirPath}":    ArgVarInfo{"Full path to current file's directory.", ArgVarDir},
	"{FileDirProjRel}": ArgVarInfo{"Path to current file's directory relative to project root.", ArgVarDir},

	// Project Root dir
	"{ProjDir}":  ArgVarInfo{"Current project directory name, without full path.", ArgVarDir},
	"{ProjPath}": ArgVarInfo{"Full path to current project directory.", ArgVarDir},

	// BuildDir
	"{BuildDir}":    ArgVarInfo{"Full path to BuildDir specified in project prefs -- the default Build.", ArgVarDir},
	"{BuildDirRel}": ArgVarInfo{"Path to BuildDir relative to project root.", ArgVarDir},

	// BuildTarg
	"{BuildTarg}":           ArgVarInfo{"Build target specified in prefs BuildTarg, just filename by itself, without path.", ArgVarFile},
	"{BuildTargPath}":       ArgVarInfo{"Full path to build target specified in prefs BuildTarg.", ArgVarFile},
	"{BuildTargDirPath}":    ArgVarInfo{"Full path to build target directory, without filename.", ArgVarDir},
	"{BuildTargDirPathRel}": ArgVarInfo{"Project-relative path to build target directory, without filename.", ArgVarDir},

	// RunExec
	"{RunExec}":           ArgVarInfo{"Run-time executable file RunExec specified in project prefs -- just the raw name of the file, without path.", ArgVarFile},
	"{RunExecPath}":       ArgVarInfo{"Full path to the run-time executable file RunExec specified in project prefs.", ArgVarFile},
	"{RunExecDirPath}":    ArgVarInfo{"Full path to the directory of the run-time executable file RunExec specified in project prefs.", ArgVarDir},
	"{RunExecDirPathRel}": ArgVarInfo{"Project-root relative path to the directory of the run-time executable file RunExec specified in project prefs.", ArgVarDir},

	// Cursor, Selection
	"{CurLine}":      ArgVarInfo{"Cursor current line number (starts at 1).", ArgVarPos},
	"{CurCol}":       ArgVarInfo{"Cursor current column number (starts at 0).", ArgVarPos},
	"{SelStartLine}": ArgVarInfo{"Selection starting line (same as CurLine if no selection).", ArgVarPos},
	"{SelStartCol}":  ArgVarInfo{"Selection starting column (same as CurCol if no selection).", ArgVarPos},
	"{SelEndLine}":   ArgVarInfo{"Selection ending line (same as CurLine if no selection).", ArgVarPos},
	"{SelEndCol}":    ArgVarInfo{"Selection ending column (same as CurCol if no selection).", ArgVarPos},

	"{CurSel}":      ArgVarInfo{"Currently selected text.", ArgVarText},
	"{CurLineText}": ArgVarInfo{"Current line text under cursor.", ArgVarText},
	"{CurWord}":     ArgVarInfo{"Current word under cursor.", ArgVarText},

	"{PromptFilePath}":       ArgVarInfo{"Prompt user for a file, and this is the full path to that file.", ArgVarPrompt},
	"{PromptFileName}":       ArgVarInfo{"Prompt user for a file, and this is the filename (only) of that file.", ArgVarPrompt},
	"{PromptFileDir}":        ArgVarInfo{"Prompt user for a file, and this is the directory name (only) of that file.", ArgVarPrompt},
	"{PromptFileDirPath}":    ArgVarInfo{"Prompt user for a file, and this is the full path to that directory.", ArgVarPrompt},
	"{PromptFileDirProjRel}": ArgVarInfo{"Prompt user for a file, and this is the path of that directory relative to the project root.", ArgVarPrompt},
	"{PromptString1}":        ArgVarInfo{"Prompt user for a string -- this is it.", ArgVarPrompt},
	"{PromptString2}":        ArgVarInfo{"Prompt user for another string -- this is it.", ArgVarPrompt},
}

// ArgVarVals are current values of arg var vals -- updated on demand when a
// command is invoked
type ArgVarVals map[string]string

// Set sets the current values for arg variables -- prompts must be already set!
func (avp *ArgVarVals) Set(fpath string, ppref *ProjPrefs, tv *giv.TextView) {
	if *avp == nil {
		*avp = make(ArgVarVals, len(ArgVars))
	}
	av := *avp

	projpath, _ := filepath.Abs(string(ppref.ProjRoot))

	emptyPath := false
	if fpath == "" {
		emptyPath = true
		fpath = projpath
	}
	fpath, _ = filepath.Abs(fpath)

	dirpath, fnm := filepath.Split(fpath)
	if emptyPath {
		dirpath = projpath
	}
	dirpath = filepath.Clean(dirpath)
	_, dir := filepath.Split(dirpath)
	dirrel, _ := filepath.Rel(projpath, dirpath)

	_, projdir := filepath.Split(projpath)

	ext := filepath.Ext(fnm)
	extlc := strings.ToLower(ext)
	fnmnoext := strings.TrimSuffix(fnm, ext)

	bdir, _ := filepath.Abs(string(ppref.BuildDir))
	bdirrel, _ := filepath.Rel(projpath, bdir)

	trgf, _ := filepath.Abs(string(ppref.BuildTarg))
	trgpath, trg := filepath.Split(trgf)
	trgpath = filepath.Clean(trgpath)
	trgrel, _ := filepath.Rel(projpath, trgpath)

	exef, _ := filepath.Abs(string(ppref.RunExec))
	exepath, exe := filepath.Split(exef)
	exepath = filepath.Clean(exepath)
	exerel, _ := filepath.Rel(projpath, exepath)

	av["{FilePath}"] = fpath
	av["{FileName}"] = fnm
	av["{FileExt}"] = ext
	av["{FileExtLC}"] = extlc
	av["{FileNameNoExt}"] = fnmnoext
	av["{FileDir}"] = dir
	av["{FileDirPath}"] = dirpath
	av["{FileDirProjRel}"] = dirrel

	av["{ProjDir}"] = projdir
	av["{ProjPath}"] = projpath

	av["{BuildDir}"] = bdir
	av["{BuildDirRel}"] = bdirrel

	av["{BuildTarg}"] = trg
	av["{BuildTargPath}"] = trgf
	av["{BuildTargDirPath}"] = trgpath
	av["{BuildTargDirPathRel}"] = trgrel

	av["{RunExec}"] = exe
	av["{RunExecPath}"] = exef
	av["{RunExecDirPath}"] = exepath
	av["{RunExecDirPathRel}"] = exerel

	if tv != nil {
		av["{CurLine}"] = fmt.Sprintf("%v", tv.CursorPos.Ln)
		av["{CurCol}"] = fmt.Sprintf("%v", tv.CursorPos.Ch)             // not quite col
		av["{SelStartLine}"] = fmt.Sprintf("%v", tv.SelectReg.Start.Ln) // check for no sel
		av["{SelStartCol}"] = fmt.Sprintf("%v", tv.SelectReg.Start.Ch)
		av["{SelEndLine}"] = fmt.Sprintf("%v", tv.SelectReg.End.Ln)  // check for no sel
		av["{SelEndCol}"] = fmt.Sprintf("%v", tv.SelectReg.Start.Ch) // check for no sel
		av["{CurSel}"] = ""                                          // todo get sel
		av["{CurLineText}"] = ""                                     // todo get cur line
		av["{CurWord}"] = ""                                         // todo get word
	} else {
		av["{CurLine}"] = ""
		av["{CurCol}"] = ""
		av["{SelStartLine}"] = ""
		av["{SelStartCol}"] = ""
		av["{SelEndLine}"] = ""
		av["{SelEndCol}"] = ""
		av["{CurSel}"] = ""
		av["{CurLineText}"] = ""
		av["{CurWord}"] = ""
	}
}

// Bind replaces the variables in the given arg string with their values
func (avp *ArgVarVals) Bind(arg string) string {
	sz := len(arg)
	bs := []byte(arg)
	ci := 0
	gotquote := false
	for ci < sz {
		sb := bytes.Index(bs[ci:], []byte("{"))
		if sb < 0 {
			break
		}
		ci += sb
		if ci-1 >= 0 && bs[ci-1] == '\\' { // quoted
			ci++
			gotquote = true
			continue
		}
		eb := bytes.Index(bs[ci+1:], []byte("}"))
		if eb < 0 {
			break
		}
		eb += ci + 1
		vnm := string(bs[ci : eb+1])
		// fmt.Printf("%v\n", vnm)
		if val, ok := (*avp)[vnm]; ok {
			end := make([]byte, sz-(eb+1))
			copy(end, bs[eb+1:])
			bs = append(bs[:ci], []byte(val)...)
			ci = len(bs)
			bs = append(bs, end...)
		} else {
			break
		}
		sz = len(bs)
	}
	if gotquote {
		bs = bytes.Replace(bs, []byte("\\{"), []byte("{"), -1)
	}

	if oswin.TheApp != nil {
		if oswin.TheApp.Platform() == oswin.Windows {
			bs = bytes.Replace(bs, []byte("}/{"), []byte("}\\{"), -1)
		}
	}
	return string(bs)
}

// ArgVarPrompts returns any Prompt* variables required by this string, false if none
func ArgVarPrompts(arg string) (map[string]struct{}, bool) {
	sz := len(arg)
	var ps map[string]struct{}
	bs := []byte(arg)
	ci := 0
	for ci < sz {
		sb := bytes.Index(bs[ci:], []byte("{"))
		if sb < 0 {
			break
		}
		ci += sb
		if ci-1 >= 0 && bs[ci-1] == '\\' { // quoted
			ci++
			continue
		}
		eb := bytes.Index(bs[ci+1:], []byte("}"))
		if eb < 0 {
			break
		}
		eb += ci + 1
		vnm := string(bs[ci : eb+1])
		// fmt.Printf("%v\n", vnm)
		if strings.HasPrefix(vnm, "{Prompt") {
			if ps == nil {
				ps = make(map[string]struct{})
			}
			if strings.HasPrefix(vnm, "{PromptFile") {
				ps["{PromptFileName}"] = struct{}{}
			} else {
				ps[vnm] = struct{}{}
			}
		}
		ci = eb
	}
	if len(ps) > 0 {
		return ps, true
	}
	return nil, false
}

// ArgVarKeys creates a slice of string to hold the keys
func ArgVarKeys() []string {
	keys := make([]string, 0, len(ArgVars))
	for k := range ArgVars {
		keys = append(keys, k)
	}
	return keys
}

// ArgVarTypes describe the type of information in the arg var -- used for
// checking usage and special features.
type ArgVarTypes int32

const (
	// ArgVarFile is a file name, not a directory
	ArgVarFile ArgVarTypes = iota

	// ArgVarDir is a directory name, not a file
	ArgVarDir

	// ArgVarExt is a file extension
	ArgVarExt

	// ArgVarPos is a text position
	ArgVarPos

	// ArgVarText is text from a buffer
	ArgVarText

	// ArgVarPrompt is a user-prompted variable
	ArgVarPrompt

	// ArgVarTypesN is the number of ArgVarTypes
	ArgVarTypesN
)

//go:generate stringer -type=ArgVarTypes

var KiT_ArgVarTypes = kit.Enums.AddEnumAltLower(ArgVarTypesN, false, nil, "ArgVar")

// MarshalJSON saves arg variables to a JSON-formatted file
func (kf ArgVarTypes) MarshalJSON() ([]byte, error) { return kit.EnumMarshalJSON(kf) }

// UnmarshalJSON decodes arg variables from a JSON-formatted file
func (kf *ArgVarTypes) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(kf, b) }
