// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"reflect"
	"time"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/giv/textbuf"
	"github.com/goki/pi/complete"
	"github.com/goki/pi/filecat"
)

// Gide provides the interface for the GideView functionality that is needed
// by the core gide infrastructure, to allow GideView to be in a separate package.
// It is not intended to be the full functionality of the GideView.
type Gide interface {
	gi.Node2D

	// VPort returns the viewport for the view
	VPort() *gi.Viewport2D

	// ProjPrefs returns the gide.ProjPrefs
	ProjPrefs() *ProjPrefs

	// FileTree returns the gide.Files file tree
	FileTree() *giv.FileTree

	// LastSaveTime returns the time stamp when a file was last saved within project --
	// can be used for dirty flag state relative to other time stamps.
	LastSaveTime() time.Time

	// VersCtrl returns the version control system in effect, using the file tree detected
	// version or whatever is set in project preferences
	VersCtrl() giv.VersCtrlName

	// CmdRuns returns the CmdRuns manager of running commands, used extensively
	// in commands.go
	CmdRuns() *CmdRuns

	// ArgVarVals returns the ArgVarVals argument variable values
	ArgVarVals() *ArgVarVals

	// SetStatus updates the statusbar label with given message, along with other status info
	SetStatus(msg string)

	// SelectTabByName Selects given main tab, and returns all of its contents as well.
	SelectTabByName(label string) gi.Node2D

	// FocusOnTabs moves keyboard focus to Tabs panel -- returns false if nothing at that tab
	FocusOnTabs() bool

	// ShowFile shows given file name at given line, returning TextView showing it
	// or error if not found.
	ShowFile(fname string, ln int) (*TextView, error)

	// FileNodeForFile returns file node for given file path.
	// add: if not found in existing tree and external files, then if add is true,
	// it is added to the ExtFiles list.
	FileNodeForFile(fpath string, add bool) *giv.FileNode

	// TextBufForFile returns the TextBuf for given file path
	// add: if not found in existing tree and external files, then if add is true,
	// it is added to the ExtFiles list.
	TextBufForFile(fpath string, add bool) *giv.TextBuf

	// NextViewFileNode sets the next text view to view file in given node (opens
	// buffer if not already opened) -- if already being viewed, that is
	// activated, returns text view and index
	NextViewFileNode(fn *giv.FileNode) (*TextView, int)

	// ActiveTextView returns the currently-active TextView
	ActiveTextView() *TextView

	// SetActiveTextView sets the given textview as the active one, and returns its index
	SetActiveTextView(av *TextView) int

	// ExecCmdFileNode pops up a menu to select a command appropriate for the given node,
	// and shows output in Tab with name of command
	ExecCmdFileNode(fn *giv.FileNode)

	// ExecCmdNameFileNode executes command of given name on given node
	ExecCmdNameFileNode(fn *giv.FileNode, cmdNm CmdName, sel bool, clearBuf bool)

	// ExecCmdNameFileName executes command of given name on given file name
	ExecCmdNameFileName(fn string, cmdNm CmdName, sel bool, clearBuf bool)

	// Find does Find / Replace in files, using given options and filters -- opens up a
	// main tab with the results and further controls.
	Find(find, repl string, ignoreCase, regExp bool, loc FindLoc, langs []filecat.Supported)

	// ParseOpenFindURL parses and opens given find:/// url from Find, return text
	// region encoded in url, and starting line of results in find buffer, and
	// number of results returned -- for parsing all the find results
	ParseOpenFindURL(ur string, ftv *giv.TextView) (tv *TextView, reg textbuf.Region, findBufStLn, findCount int, ok bool)

	// OpenFileAtRegion opens the specified file, highlights the region and sets the cursor
	OpenFileAtRegion(filename gi.FileName, reg textbuf.Region) (tv *TextView, ok bool)

	// SaveAllCheck checks if any files have not been saved, and prompt to save them.
	// returns true if there were unsaved files, false otherwise.
	// cancelOpt presents an option to cancel current command,
	// in which case function is not called.
	// if function is passed, then it is called in all cases except
	// if the user selects cancel.
	SaveAllCheck(cancelOpt bool, fun func()) bool

	// SaveAllOpenNodes saves all of the open filenodes to their current file names
	SaveAllOpenNodes()

	// CloseOpenNodes closes any nodes with open views (including those in directories under nodes).
	// called prior to rename.
	CloseOpenNodes(nodes []*FileNode)

	// LookupFun is the completion system Lookup function that makes a custom
	// textview dialog that has option to edit resulting file.
	LookupFun(data interface{}, text string, posLn, posCh int) (ld complete.Lookup)

	// Spell checks spelling in active text view
	Spell()

	// Symbols calls a function to parse file or package
	Symbols()

	// Debug runs debugger on default exe
	Debug()

	// CurDebug returns the current debug view, or nil
	CurDebug() *DebugView

	// ClearDebug clears the current debugger setting -- no more debugger active.
	ClearDebug()
}

// GideType is a Gide reflect.Type, suitable for checking for Type.Implements.
var GideType = reflect.TypeOf((*Gide)(nil)).Elem()
