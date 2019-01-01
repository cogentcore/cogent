// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"reflect"

	"github.com/goki/gi/filecat"
	"github.com/goki/gi/gi"
	"github.com/goki/gi/giv"
)

// Gide provides the interface for the GideView functionality that is needed
// by the core gide infrastructure, to allow GideView to be in a separate package.
// It is not intended to be the full functionality of the GideView.
type Gide interface {
	gi.Node2D

	// VPort returns the viewport for the view
	VPort() *gi.Viewport2D

	// ProjPrefs() returns the gide.ProjPrefs
	ProjPrefs() *ProjPrefs

	// CmdRuns returns the CmdRuns manager of running commands, used extensively
	// in commands.go
	CmdRuns() *CmdRuns

	// ArgVarVals returns the ArgVarVals argument variable values
	ArgVarVals() *ArgVarVals

	// SetStatus updates the statusbar label with given message, along with other status info
	SetStatus(msg string)

	// SelectMainTabByName Selects given main tab, and returns all of its contents as well.
	SelectMainTabByName(label string) (gi.Node2D, int, bool)

	// FocusOnMainTabs moves keyboard focus to MainTabs panel -- returns false if nothing at that tab
	FocusOnMainTabs() bool

	// NextViewFileNode sets the next text view to view file in given node (opens
	// buffer if not already opened) -- if already being viewed, that is
	// activated, returns text view and index
	NextViewFileNode(fn *giv.FileNode) (*giv.TextView, int)

	// ActiveTextView returns the currently-active TextView
	ActiveTextView() *giv.TextView

	// ConfigOutputTextView configures a command-output textview within given parent layout
	ConfigOutputTextView(ly *gi.Layout) *giv.TextView

	// ExecCmdFileNode pops up a menu to select a command appropriate for the given node,
	// and shows output in MainTab with name of command
	ExecCmdFileNode(fn *giv.FileNode)

	// ExecCmdNameFileNode executes command of given name on given node
	ExecCmdNameFileNode(fn *giv.FileNode, cmdNm CmdName, sel bool, clearBuf bool)

	// ExecCmdNameFileName executes command of given name on given file name
	ExecCmdNameFileName(fn string, cmdNm CmdName, sel bool, clearBuf bool)

	// Find does Find / Replace in files, using given options and filters -- opens up a
	// main tab with the results and further controls.
	Find(find, repl string, ignoreCase bool, loc FindLoc, langs []filecat.Supported)

	// ParseOpenFindURL parses and opens given find:/// url from Find, return text
	// region encoded in url, and starting line of results in find buffer, and
	// number of results returned -- for parsing all the find results
	ParseOpenFindURL(ur string, ftv *giv.TextView) (tv *giv.TextView, reg giv.TextRegion, findBufStLn, findCount int, ok bool)

	// Spell checks spelling in files
	Spell()

	// Structure calls a function to parse file or package
	Structure()
}

// GideType is a Gide reflect.Type, suitable for checking for Type.Implements.
var GideType = reflect.TypeOf((*Gide)(nil)).Elem()
