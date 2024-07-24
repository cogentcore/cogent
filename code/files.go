// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/filetree"
	"cogentcore.org/core/parse/lexer"
	"cogentcore.org/core/texteditor"
	"cogentcore.org/core/tree"
)

// SaveActiveView saves the contents of the currently active texteditor
func (cv *Code) SaveActiveView() { //types:add
	tv := cv.ActiveTextEditor()
	if tv.Buffer != nil {
		cv.LastSaveTStamp = time.Now()
		if tv.Buffer.Filename != "" {
			tv.Buffer.Save()
			cv.SetStatus("File Saved")
			fnm := string(tv.Buffer.Filename)
			fpath, _ := filepath.Split(fnm)
			cv.Files.UpdatePath(fpath) // update everything in dir -- will have removed autosave
			cv.RunPostCmdsActiveView()
		} else {
			core.CallFunc(cv, cv.SaveActiveViewAs)
		}
	}
	cv.SaveProjectIfExists(false) // no saveall
}

// ConfigActiveFilename configures the first arg of given FuncButton to
// use the ActiveFilename
func (cv *Code) ConfigActiveFilename(fb *core.FuncButton) *core.FuncButton {
	fb.Args[0].SetValue(cv.ActiveFilename)
	return fb
}

func (cv *Code) CallSaveActiveViewAs(ctx core.Widget) {
	cv.ConfigActiveFilename(core.NewSoloFuncButton(ctx).SetFunc(cv.SaveActiveViewAs)).CallFunc()
}

// SaveActiveViewAs save with specified filename the contents of the
// currently active texteditor
func (cv *Code) SaveActiveViewAs(filename core.Filename) { //types:add
	tv := cv.ActiveTextEditor()
	if tv.Buffer != nil {
		cv.LastSaveTStamp = time.Now()
		ofn := tv.Buffer.Filename
		tv.Buffer.SaveAsFunc(filename, func(canceled bool) {
			if canceled {
				cv.SetStatus(fmt.Sprintf("File %v NOT Saved As: %v", ofn, filename))
				return
			}
			cv.SetStatus(fmt.Sprintf("File %v Saved As: %v", ofn, filename))
			// ge.RunPostCmdsActiveView() // doesn't make sense..
			cv.Files.UpdatePath(string(filename)) // update everything in dir -- will have removed autosave
			fn, ok := cv.Files.FindFile(string(filename))
			if ok {
				if fn.Buffer != nil {
					fn.Buffer.Revert()
				}
				cv.ViewFileNode(tv, cv.ActiveTextEditorIndex, fn)
			}
		})
	}
	cv.SaveProjectIfExists(false) // no saveall
}

// RevertActiveView revert active view to saved version
func (cv *Code) RevertActiveView() { //types:add
	tv := cv.ActiveTextEditor()
	if tv.Buffer != nil {
		cv.ConfigTextBuffer(tv.Buffer)
		tv.Buffer.Revert()
		tv.Buffer.Undos.Reset() // key implication of revert
		fpath, _ := filepath.Split(string(tv.Buffer.Filename))
		cv.Files.UpdatePath(fpath) // update everything in dir -- will have removed autosave
	}
}

// CloseActiveView closes the buffer associated with active view
func (cv *Code) CloseActiveView() { //types:add
	tv := cv.ActiveTextEditor()
	ond, _, got := cv.OpenNodeForTextEditor(tv)
	if got {
		ond.Buffer.Close(func(canceled bool) {
			if canceled {
				cv.SetStatus(fmt.Sprintf("File %v NOT closed", ond.Filepath))
				return
			}
			cv.SetStatus(fmt.Sprintf("File %v closed", ond.Filepath))
			cv.OpenNodes.Delete(ond)
		})
	}
}

// RunPostCmdsActiveView runs any registered post commands on the active view
// -- returns true if commands were run and file was reverted after that --
// uses MainLang to disambiguate if multiple languages associated with extension.
func (cv *Code) RunPostCmdsActiveView() bool {
	tv := cv.ActiveTextEditor()
	ond, _, got := cv.OpenNodeForTextEditor(tv)
	if got {
		return cv.RunPostCmdsFileNode(ond)
	}
	return false
}

// RunPostCmdsFileNode runs any registered post commands on the given file node
// -- returns true if commands were run and file was reverted after that --
// uses MainLang to disambiguate if multiple languages associated with extension.
func (cv *Code) RunPostCmdsFileNode(fn *filetree.Node) bool {
	lang := fn.Info.Known
	if lopt, has := AvailableLanguages[lang]; has {
		if len(lopt.PostSaveCmds) > 0 {
			_, ptab := cv.Tabs().CurrentTab()
			cv.ExecCmdsFileNode(fn, lopt.PostSaveCmds)
			if ptab >= 0 {
				cv.Tabs().SelectTabIndex(ptab) // we stay at the previous tab
			}
			fn.Buffer.Revert()
			return true
		}
	}
	return false
}

// AutoSaveCheck checks for an autosave file and prompts user about opening it
// -- returns true if autosave file does exist for a file that currently
// unchanged (means just opened)
func (cv *Code) AutoSaveCheck(tv *TextEditor, vidx int, fn *filetree.Node) bool {
	if strings.HasPrefix(fn.Name, "#") && strings.HasSuffix(fn.Name, "#") {
		fn.Buffer.Autosave = false
		return false // we are the autosave file
	}
	fn.Buffer.Autosave = true
	if tv.IsNotSaved() || !fn.Buffer.AutoSaveCheck() {
		return false
	}
	cv.DiffFileNode(fn, core.Filename(fn.Buffer.AutoSaveFilename()))
	d := core.NewBody().AddTitle("Autosave file Exists").
		AddText(fmt.Sprintf("An auto-save file for file: %v exists; open it in the other text view (you can then do Save As to replace current file)?  If you don't open it, the next change made will overwrite it with a new one, erasing any changes.", fn.Name))
	d.AddBottomBar(func(parent core.Widget) {
		core.NewButton(parent).SetText("Ignore and overwrite autosave file").OnClick(func(e events.Event) {
			d.Close()
			fn.Buffer.AutoSaveDelete()
			cv.Files.UpdatePath(fn.Buffer.AutoSaveFilename()) // will update dir
		})
		core.NewButton(parent).SetText("Open autosave file").OnClick(func(e events.Event) {
			d.Close()
			cv.NextViewFile(core.Filename(fn.Buffer.AutoSaveFilename()))
		})
	})
	d.RunDialog(cv)
	return true
}

// OpenFileNode opens file for file node -- returns new bool and error
func (cv *Code) OpenFileNode(fn *filetree.Node) (bool, error) {
	if fn.IsDir() {
		return false, fmt.Errorf("cannot open directory: %v", fn.Filepath)
	}
	filetree.NodeHighlighting = core.AppearanceSettings.Highlighting // must be set prior to OpenBuf
	nw, err := fn.OpenBuf()
	if err == nil {
		cv.ConfigTextBuffer(fn.Buffer)
		cv.OpenNodes.Add(fn)
		fn.Open()
		fn.Update()
	}
	return nw, err
}

// ViewFileNode sets the given text view to view file in given node (opens
// buffer if not already opened).  This is the main method for viewing a file.
func (cv *Code) ViewFileNode(tv *TextEditor, vidx int, fn *filetree.Node) {
	if fn.IsDir() {
		return
	}

	if tv.IsNotSaved() {
		cv.SetStatus(fmt.Sprintf("Note: Changes not yet saved in file: %v", tv.Buffer.Filename))
	}
	nw, err := cv.OpenFileNode(fn)
	if err == nil {
		tv.SetBuffer(fn.Buffer)
		if nw {
			cv.AutoSaveCheck(tv, vidx, fn)
		}
		cv.SetActiveTextEditorIndex(vidx) // this calls FileModCheck
	}
}

// NextViewFileNode sets the next text view to view file in given node (opens
// buffer if not already opened) -- if already being viewed, that is
// activated, returns text view and index
func (cv *Code) NextViewFileNode(fn *filetree.Node) (*TextEditor, int) {
	tv, idx, ok := cv.TextEditorForFileNode(fn)
	if ok {
		cv.SetActiveTextEditorIndex(idx)
		return tv, idx
	}
	nv, nidx := cv.NextTextEditor()
	// fmt.Println("next idx:", nidx)
	cv.ViewFileNode(nv, nidx, fn)
	return nv, nidx
}

// FileNodeForFile returns file node for given file path
// add: if not found in existing tree and external files, then if add is true,
// it is added to the ExtFiles list.
func (cv *Code) FileNodeForFile(fpath string, add bool) *filetree.Node {
	fn, ok := cv.Files.FindFile(fpath)
	if !ok {
		if !add {
			return nil
		}
		if strings.HasSuffix(fpath, "/") {
			log.Printf("Code: attempt to add dir to external files: %v\n", fpath)
			return nil
		}
		efn, err := cv.Files.AddExternalFile(fpath)
		if err != nil {
			log.Printf("Code: cannot add external file: %v\n", err)
			return nil
		}
		return efn
	}
	if fn.IsDir() {
		return nil
	}
	return fn
}

// TextBufForFile returns TextBuf for given file path.
// add: if not found in existing tree and external files, then if add is true,
// it is added to the ExtFiles list.
func (cv *Code) TextBufForFile(fpath string, add bool) *texteditor.Buffer {
	fn := cv.FileNodeForFile(fpath, add)
	if fn == nil {
		return nil
	}
	_, err := cv.OpenFileNode(fn)
	if err == nil {
		return fn.Buffer
	}
	return nil
}

// NextViewFile sets the next text view to view given file name -- include as
// much of name as possible to disambiguate -- will use the first matching --
// if already being viewed, that is activated -- returns texteditor and its
// index, false if not found
func (cv *Code) NextViewFile(fnm core.Filename) (*TextEditor, int, bool) { //types:add
	fn := cv.FileNodeForFile(string(fnm), true)
	if fn == nil {
		return nil, -1, false
	}
	nv, nidx := cv.NextViewFileNode(fn)
	return nv, nidx, true
}

// CallViewFile calls ViewFile with ActiveFilename set as arg
func (cv *Code) CallViewFile(ctx core.Widget) {
	cv.ConfigActiveFilename(core.NewSoloFuncButton(ctx).SetFunc(cv.ViewFile)).CallFunc()
}

// ViewFile views file in an existing TextEditor if it is already viewing that
// file, otherwise opens ViewFileNode in active buffer
func (cv *Code) ViewFile(fnm core.Filename) (*TextEditor, int, bool) { //types:add
	fn := cv.FileNodeForFile(string(fnm), true)
	if fn == nil {
		return nil, -1, false
	}
	tv, idx, ok := cv.TextEditorForFileNode(fn)
	if ok {
		cv.SetActiveTextEditorIndex(idx)
		return tv, idx, ok
	}
	tv = cv.ActiveTextEditor()
	idx = cv.ActiveTextEditorIndex
	cv.ViewFileNode(tv, idx, fn)
	return tv, idx, true
}

// ViewFileInIndex views file in given text view index
func (cv *Code) ViewFileInIndex(fnm core.Filename, idx int) (*TextEditor, int, bool) {
	fn := cv.FileNodeForFile(string(fnm), true)
	if fn == nil {
		return nil, -1, false
	}
	tv := cv.TextEditorByIndex(idx)
	cv.ViewFileNode(tv, idx, fn)
	return tv, idx, true
}

// LinkViewFileNode opens the file node in the 2nd texteditor, which is next to
// the tabs where links are clicked, if it is not collapsed -- else 1st
func (cv *Code) LinkViewFileNode(fn *filetree.Node) (*TextEditor, int) {
	if cv.PanelIsOpen(TextEditor2Index) {
		cv.SetActiveTextEditorIndex(1)
	} else {
		cv.SetActiveTextEditorIndex(0)
	}
	tv := cv.ActiveTextEditor()
	idx := cv.ActiveTextEditorIndex
	cv.ViewFileNode(tv, idx, fn)
	return tv, idx
}

// LinkViewFile opens the file in the 2nd texteditor, which is next to
// the tabs where links are clicked, if it is not collapsed -- else 1st
func (cv *Code) LinkViewFile(fnm core.Filename) (*TextEditor, int, bool) {
	fn := cv.FileNodeForFile(string(fnm), true)
	if fn == nil {
		return nil, -1, false
	}
	tv, idx, ok := cv.TextEditorForFileNode(fn)
	if ok {
		if idx == 1 {
			return tv, idx, true
		}
		if cv.SwapTextEditors() {
			return cv.TextEditorByIndex(1), 1, true
		}
	}
	nv, nidx := cv.LinkViewFileNode(fn)
	return nv, nidx, true
}

// ShowFile shows given file name at given line, returning TextEditor showing it
// or error if not found.
func (cv *Code) ShowFile(fname string, ln int) (*TextEditor, error) {
	tv, _, ok := cv.LinkViewFile(core.Filename(fname))
	if ok {
		tv.SetCursorTarget(lexer.Pos{Ln: ln - 1})
		return tv, nil
	}
	return nil, fmt.Errorf("ShowFile: file named: %v not found\n", fname)
}

// CodeOpenNodes gets list of open nodes for submenu-func
func CodeOpenNodes(it any, sc *core.Scene) []string {
	cv, ok := it.(tree.Node).(*Code)
	if !ok {
		return nil
	}
	return cv.OpenNodes.Strings()
}

// ViewOpenNodeName views given open node (by name) in active view
func (cv *Code) ViewOpenNodeName(name string) {
	nb := cv.OpenNodes.ByStringName(name)
	if nb == nil {
		return
	}
	tv := cv.ActiveTextEditor()
	cv.ViewFileNode(tv, cv.ActiveTextEditorIndex, nb)
}

// SelectOpenNode pops up a menu to select an open node (aka buffer) to view
// in current active texteditor
func (cv *Code) SelectOpenNode() {
	if len(cv.OpenNodes) == 0 {
		cv.SetStatus("No open nodes to choose from")
		return
	}
	nl := cv.OpenNodes.Strings()
	if len(nl) == 0 {
		return
	}
	tv := cv.ActiveTextEditor() // nl[0] is always currently viewed
	def := nl[0]
	if len(nl) > 1 {
		def = nl[1]
	}
	m := core.NewMenuFromStrings(nl, def, func(idx int) {
		nb := cv.OpenNodes[idx]
		cv.ViewFileNode(tv, cv.ActiveTextEditorIndex, nb)
	})
	core.NewMenuStage(m, tv, tv.ContextMenuPos(nil)).Run()
}

// CloneActiveView sets the next text view to view the same file currently being vieweds
// in the active view. returns text view and index
func (cv *Code) CloneActiveView() (*TextEditor, int) { //types:add
	tv := cv.ActiveTextEditor()
	if tv == nil {
		return nil, -1
	}
	ond, _, got := cv.OpenNodeForTextEditor(tv)
	if got {
		nv, nidx := cv.NextTextEditor()
		cv.ViewFileNode(nv, nidx, ond)
		return nv, nidx
	}
	return nil, -1
}

// SaveAllOpenNodes saves all of the open filenodes to their current file names
func (cv *Code) SaveAllOpenNodes() {
	for _, ond := range cv.OpenNodes {
		if ond.Buffer == nil {
			continue
		}
		if ond.Buffer.IsNotSaved() {
			ond.Buffer.Save()
			cv.RunPostCmdsFileNode(ond)
		}
	}
}

// SaveAll saves all of the open filenodes to their current file names
// and saves the project state if it has been saved before (i.e., the .code file exists)
func (cv *Code) SaveAll() { //types:add
	cv.SaveAllOpenNodes()
	cv.SaveProjectIfExists(false)
}

// CloseOpenNodes closes any nodes with open views (including those in directories under nodes).
// called prior to rename.
func (cv *Code) CloseOpenNodes(nodes []*FileNode) {
	nn := len(cv.OpenNodes)
	for ni := nn - 1; ni >= 0; ni-- {
		ond := cv.OpenNodes[ni]
		if ond.Buffer == nil {
			continue
		}
		path := string(ond.Buffer.Filename)
		for _, cnd := range nodes {
			if strings.HasPrefix(path, string(cnd.Filepath)) {
				ond.Buffer.Close(func(canceled bool) {
					if canceled {
						cv.SetStatus(fmt.Sprintf("File %v NOT closed -- recommended as file name changed!", ond.Filepath))
						return
					}
					cv.SetStatus(fmt.Sprintf("File %v closed due to file name change", ond.Filepath))
				})
				break // out of inner node loop
			}
		}
	}
}

// FileNodeRunExe runs the given executable file node
func (cv *Code) FileNodeRunExe(fn *filetree.Node) {
	cv.SetArgVarVals()
	cv.ArgVals["{PromptString1}"] = string(fn.Filepath)
	CmdNoUserPrompt = true // don't re-prompt!
	cmd, _, ok := AvailableCommands.CmdByName(CmdName("Build: Run Prompt"), true)
	if ok {
		cv.ArgVals.Set(string(fn.Filepath), &cv.Settings, nil)
		cbuf, _, _ := cv.RecycleCmdTab(cmd.Name)
		cmd.Run(cv, cbuf)
	}
}

// FileNodeOpened is called whenever file node is double-clicked in file tree
func (cv *Code) FileNodeOpened(fn *filetree.Node) {
	// todo: could add all these options in LangOpts
	switch fn.Info.Cat {
	case fileinfo.Folder:
	case fileinfo.Exe:
		cv.FileNodeRunExe(fn)
		// this uses exe path for cd to this path!
		return
	case fileinfo.Font, fileinfo.Video, fileinfo.Model, fileinfo.Audio, fileinfo.Sheet, fileinfo.Bin,
		fileinfo.Archive, fileinfo.Image:
		cv.ExecCmdNameFileNode(fn, CmdName("File: Open"))
		return
	}

	edit := true
	switch fn.Info.Cat {
	case fileinfo.Code:
		edit = true
	case fileinfo.Text:
		edit = true
	default:
		if _, noed := CatNoEdit[fn.Info.Known]; noed {
			edit = false
		}
	}
	if !edit {
		cv.ExecCmdNameFileNode(fn, CmdName("File: Open"))
		return
	}
	// program, document, data
	if int(fn.Info.Size) > core.SystemSettings.BigFileSize {
		d := core.NewBody().AddTitle("File is relatively large").
			AddText(fmt.Sprintf("The file: %v is relatively large at: %v; really open for editing?", fn.Name, fn.Info.Size))
		d.AddBottomBar(func(parent core.Widget) {
			d.AddCancel(parent)
			core.NewButton(parent).SetText("Open").OnClick(func(e events.Event) {
				d.Close()
				cv.NextViewFileNode(fn)
			})
		})
		d.RunDialog(cv)
	} else {
		cv.NextViewFileNode(fn)
	}

}

// CatNoEdit are the files to NOT edit from categories: Doc, Data
var CatNoEdit = map[fileinfo.Known]bool{
	fileinfo.Rtf:          true,
	fileinfo.MSWord:       true,
	fileinfo.OpenText:     true,
	fileinfo.OpenPres:     true,
	fileinfo.MSPowerpoint: true,
	fileinfo.EBook:        true,
	fileinfo.EPub:         true,
}
