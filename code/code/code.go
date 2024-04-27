// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

//go:generate core generate

import (
	"embed"
	"io/fs"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/fileinfo"
	"cogentcore.org/core/filetree"
	"cogentcore.org/core/gox/errors"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/parse/complete"
	"cogentcore.org/core/texteditor"
	"cogentcore.org/core/texteditor/textbuf"
	"cogentcore.org/core/tree"
)

// Code provides the interface for the CodeView functionality that is needed
// by the core code infrastructure, to allow CodeView to be in a separate package.
// It is not intended to be the full functionality of the CodeView.
type Code interface {
	core.Widget

	// ProjectSettings returns the [ProjectSettings]
	ProjectSettings() *ProjectSettings

	// FileTree returns the code.Files file tree
	FileTree() *filetree.Tree

	// LastSaveTime returns the time stamp when a file was last saved within project --
	// can be used for dirty flag state relative to other time stamps.
	LastSaveTime() time.Time

	// VersionControl returns the version control system in effect, using the file tree detected
	// version or whatever is set in project settings
	VersionControl() filetree.VersionControlName

	// CmdRuns returns the CmdRuns manager of running commands, used extensively
	// in commands.go
	CmdRuns() *CmdRuns

	// history of commands executed in this session
	CmdHist() *CmdNames

	// ArgVarVals returns the ArgVarVals argument variable values
	ArgVarVals() *ArgVarVals

	// SetStatus updates the statusbar label with given message, to be rendered next time
	SetStatus(msg string)

	// UpdateStatusText updates the status bar text with current data
	UpdateStatusText()

	// SelectTabByName Selects given main tab, and returns all of its contents as well.
	SelectTabByName(label string) core.Widget

	// FocusOnTabs moves keyboard focus to Tabs panel -- returns false if nothing at that tab
	FocusOnTabs() bool

	// ShowFile shows given file name at given line, returning TextEditor showing it
	// or error if not found.
	ShowFile(fname string, ln int) (*TextEditor, error)

	// FileNodeOpened is called whenever file node is double-clicked in file tree
	FileNodeOpened(fn *filetree.Node)

	// FileNodeForFile returns file node for given file path.
	// add: if not found in existing tree and external files, then if add is true,
	// it is added to the ExtFiles list.
	FileNodeForFile(fpath string, add bool) *filetree.Node

	// TextBufForFile returns the TextBuf for given file path
	// add: if not found in existing tree and external files, then if add is true,
	// it is added to the ExtFiles list.
	TextBufForFile(fpath string, add bool) *texteditor.Buffer

	// NextViewFileNode sets the next text view to view file in given node (opens
	// buffer if not already opened) -- if already being viewed, that is
	// activated, returns text view and index
	NextViewFileNode(fn *filetree.Node) (*TextEditor, int)

	// ActiveTextEditor returns the currently active TextEditor
	ActiveTextEditor() *TextEditor

	// SetActiveTextEditor sets the given texteditor as the active one, and returns its index
	SetActiveTextEditor(av *TextEditor) int

	// ActiveFileNode returns the file node for the active file -- nil if none
	ActiveFileNode() *filetree.Node

	// ExecCmdFileNode pops up a menu to select a command appropriate for the given node,
	// and shows output in Tab with name of command
	ExecCmdFileNode(fn *filetree.Node)

	// ExecCmdNameFileNode executes command of given name on given node
	ExecCmdNameFileNode(fn *filetree.Node, cmdNm CmdName, sel bool, clearBuf bool)

	// ExecCmdNameFilename executes command of given name on given file name
	ExecCmdNameFilename(fn string, cmdNm CmdName, sel bool, clearBuf bool)

	// Find does Find / Replace in files, using given options and filters -- opens up a
	// main tab with the results and further controls.
	Find(find, repl string, ignoreCase, regExp bool, loc FindLoc, langs []fileinfo.Known)

	// ParseOpenFindURL parses and opens given find:/// url from Find, return text
	// region encoded in url, and starting line of results in find buffer, and
	// number of results returned -- for parsing all the find results
	ParseOpenFindURL(ur string, ftv *texteditor.Editor) (tv *TextEditor, reg textbuf.Region, findBufStLn, findCount int, ok bool)

	// OpenFileAtRegion opens the specified file, highlights the region and sets the cursor
	OpenFileAtRegion(filename core.Filename, reg textbuf.Region) (tv *TextEditor, ok bool)

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

	// CurOpenNodes returns the current open nodes list
	CurOpenNodes() *OpenNodes

	// LookupFun is the completion system Lookup function that makes a custom
	// texteditor dialog that has option to edit resulting file.
	LookupFun(data any, text string, posLn, posCh int) (ld complete.Lookup)

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

// ParentCode returns the Code parent of given node
func ParentCode(tn tree.Node) (Code, bool) {
	var res Code
	tn.WalkUp(func(n tree.Node) bool {
		if c, ok := n.This().(Code); ok {
			res = c
			return false
		}
		return true
	})
	return res, res != nil
}

//go:embed icons/*.svg
var Icons embed.FS

func init() {
	icons.AddFS(errors.Log1(fs.Sub(Icons, "icons")))
}
