// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package codev

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"cogentcore.org/cogent/code/code"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/fileinfo"
	"cogentcore.org/core/filetree"
	"cogentcore.org/core/parse/lexer"
	"cogentcore.org/core/texteditor"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/views"
)

// SaveActiveView saves the contents of the currently active texteditor
func (ge *CodeView) SaveActiveView() { //types:add
	tv := ge.ActiveTextEditor()
	if tv.Buffer != nil {
		ge.LastSaveTStamp = time.Now()
		if tv.Buffer.Filename != "" {
			tv.Buffer.Save()
			ge.SetStatus("File Saved")
			fnm := string(tv.Buffer.Filename)
			fpath, _ := filepath.Split(fnm)
			ge.Files.UpdatePath(fpath) // update everything in dir -- will have removed autosave
			ge.RunPostCmdsActiveView()
		} else {
			views.CallFunc(ge, ge.SaveActiveViewAs)
		}
	}
	ge.SaveProjectIfExists(false) // no saveall
}

// ConfigActiveFilename configures the first arg of given FuncButton to
// use the ActiveFilename
func (ge *CodeView) ConfigActiveFilename(fb *views.FuncButton) *views.FuncButton {
	fb.Args[0].SetValue(ge.ActiveFilename)
	return fb
}

func (ge *CodeView) CallSaveActiveViewAs(ctx core.Widget) {
	ge.ConfigActiveFilename(views.NewSoloFuncButton(ctx, ge.SaveActiveViewAs)).CallFunc()
}

// SaveActiveViewAs save with specified filename the contents of the
// currently active texteditor
func (ge *CodeView) SaveActiveViewAs(filename core.Filename) { //types:add
	tv := ge.ActiveTextEditor()
	if tv.Buffer != nil {
		ge.LastSaveTStamp = time.Now()
		ofn := tv.Buffer.Filename
		tv.Buffer.SaveAsFunc(filename, func(canceled bool) {
			if canceled {
				ge.SetStatus(fmt.Sprintf("File %v NOT Saved As: %v", ofn, filename))
				return
			}
			ge.SetStatus(fmt.Sprintf("File %v Saved As: %v", ofn, filename))
			// ge.RunPostCmdsActiveView() // doesn't make sense..
			ge.Files.UpdatePath(string(filename)) // update everything in dir -- will have removed autosave
			fn, ok := ge.Files.FindFile(string(filename))
			if ok {
				if fn.Buffer != nil {
					fn.Buffer.Revert()
				}
				ge.ViewFileNode(tv, ge.ActiveTextEditorIndex, fn)
			}
		})
	}
	ge.SaveProjectIfExists(false) // no saveall
}

// RevertActiveView revert active view to saved version
func (ge *CodeView) RevertActiveView() { //types:add
	tv := ge.ActiveTextEditor()
	if tv.Buffer != nil {
		ge.ConfigTextBuffer(tv.Buffer)
		tv.Buffer.Revert()
		tv.Buffer.Undos.Reset() // key implication of revert
		fpath, _ := filepath.Split(string(tv.Buffer.Filename))
		ge.Files.UpdatePath(fpath) // update everything in dir -- will have removed autosave
	}
}

// CloseActiveView closes the buffer associated with active view
func (ge *CodeView) CloseActiveView() { //types:add
	tv := ge.ActiveTextEditor()
	ond, _, got := ge.OpenNodeForTextEditor(tv)
	if got {
		ond.Buffer.Close(func(canceled bool) {
			if canceled {
				ge.SetStatus(fmt.Sprintf("File %v NOT closed", ond.FPath))
				return
			}
			ge.SetStatus(fmt.Sprintf("File %v closed", ond.FPath))
			ge.OpenNodes.Delete(ond)
		})
	}
}

// RunPostCmdsActiveView runs any registered post commands on the active view
// -- returns true if commands were run and file was reverted after that --
// uses MainLang to disambiguate if multiple languages associated with extension.
func (ge *CodeView) RunPostCmdsActiveView() bool {
	tv := ge.ActiveTextEditor()
	ond, _, got := ge.OpenNodeForTextEditor(tv)
	if got {
		return ge.RunPostCmdsFileNode(ond)
	}
	return false
}

// RunPostCmdsFileNode runs any registered post commands on the given file node
// -- returns true if commands were run and file was reverted after that --
// uses MainLang to disambiguate if multiple languages associated with extension.
func (ge *CodeView) RunPostCmdsFileNode(fn *filetree.Node) bool {
	lang := fn.Info.Known
	if lopt, has := code.AvailableLangs[lang]; has {
		if len(lopt.PostSaveCmds) > 0 {
			ge.ExecCmdsFileNode(fn, lopt.PostSaveCmds, false, true) // no select, yes clear
			fn.Buffer.Revert()
			return true
		}
	}
	return false
}

// AutoSaveCheck checks for an autosave file and prompts user about opening it
// -- returns true if autosave file does exist for a file that currently
// unchanged (means just opened)
func (ge *CodeView) AutoSaveCheck(tv *code.TextEditor, vidx int, fn *filetree.Node) bool {
	if strings.HasPrefix(fn.Nm, "#") && strings.HasSuffix(fn.Nm, "#") {
		fn.Buffer.Autosave = false
		return false // we are the autosave file
	}
	fn.Buffer.Autosave = true
	if tv.IsNotSaved() || !fn.Buffer.AutoSaveCheck() {
		return false
	}
	ge.DiffFileNode(fn, core.Filename(fn.Buffer.AutoSaveFilename()))
	d := core.NewBody().AddTitle("Autosave file Exists").
		AddText(fmt.Sprintf("An auto-save file for file: %v exists; open it in the other text view (you can then do Save As to replace current file)?  If you don't open it, the next change made will overwrite it with a new one, erasing any changes.", fn.Nm))
	d.AddBottomBar(func(parent core.Widget) {
		core.NewButton(parent).SetText("Ignore and overwrite autosave file").OnClick(func(e events.Event) {
			d.Close()
			fn.Buffer.AutoSaveDelete()
			ge.Files.UpdatePath(fn.Buffer.AutoSaveFilename()) // will update dir
		})
		core.NewButton(parent).SetText("Open autosave file").OnClick(func(e events.Event) {
			d.Close()
			ge.NextViewFile(core.Filename(fn.Buffer.AutoSaveFilename()))
		})
	})
	d.RunDialog(ge)
	return true
}

// OpenFileNode opens file for file node -- returns new bool and error
func (ge *CodeView) OpenFileNode(fn *filetree.Node) (bool, error) {
	if fn.IsDir() {
		return false, fmt.Errorf("cannot open directory: %v", fn.FPath)
	}
	filetree.NodeHiStyle = core.AppearanceSettings.HiStyle // must be set prior to OpenBuf
	nw, err := fn.OpenBuf()
	if err == nil {
		ge.ConfigTextBuffer(fn.Buffer)
		ge.OpenNodes.Add(fn)
		fn.Open()
		fn.UpdateNode()
	}
	return nw, err
}

// ViewFileNode sets the given text view to view file in given node (opens
// buffer if not already opened).  This is the main method for viewing a file.
func (ge *CodeView) ViewFileNode(tv *code.TextEditor, vidx int, fn *filetree.Node) {
	if fn.IsDir() {
		return
	}

	if tv.IsNotSaved() {
		ge.SetStatus(fmt.Sprintf("Note: Changes not yet saved in file: %v", tv.Buffer.Filename))
	}
	nw, err := ge.OpenFileNode(fn)
	if err == nil {
		tv.SetBuffer(fn.Buffer)
		if nw {
			ge.AutoSaveCheck(tv, vidx, fn)
		}
		ge.SetActiveTextEditorIndex(vidx) // this calls FileModCheck
	}
}

// NextViewFileNode sets the next text view to view file in given node (opens
// buffer if not already opened) -- if already being viewed, that is
// activated, returns text view and index
func (ge *CodeView) NextViewFileNode(fn *filetree.Node) (*code.TextEditor, int) {
	tv, idx, ok := ge.TextEditorForFileNode(fn)
	if ok {
		ge.SetActiveTextEditorIndex(idx)
		return tv, idx
	}
	nv, nidx := ge.NextTextEditor()
	// fmt.Println("next idx:", nidx)
	ge.ViewFileNode(nv, nidx, fn)
	return nv, nidx
}

// FileNodeForFile returns file node for given file path
// add: if not found in existing tree and external files, then if add is true,
// it is added to the ExtFiles list.
func (ge *CodeView) FileNodeForFile(fpath string, add bool) *filetree.Node {
	fn, ok := ge.Files.FindFile(fpath)
	if !ok {
		if !add {
			return nil
		}
		if strings.HasSuffix(fpath, "/") {
			log.Printf("CodeView: attempt to add dir to external files: %v\n", fpath)
			return nil
		}
		efn, err := ge.Files.AddExtFile(fpath)
		if err != nil {
			log.Printf("CodeView: cannot add external file: %v\n", err)
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
func (ge *CodeView) TextBufForFile(fpath string, add bool) *texteditor.Buffer {
	fn := ge.FileNodeForFile(fpath, add)
	if fn == nil {
		return nil
	}
	_, err := ge.OpenFileNode(fn)
	if err == nil {
		return fn.Buffer
	}
	return nil
}

// NextViewFile sets the next text view to view given file name -- include as
// much of name as possible to disambiguate -- will use the first matching --
// if already being viewed, that is activated -- returns texteditor and its
// index, false if not found
func (ge *CodeView) NextViewFile(fnm core.Filename) (*code.TextEditor, int, bool) { //types:add
	fn := ge.FileNodeForFile(string(fnm), true)
	if fn == nil {
		return nil, -1, false
	}
	nv, nidx := ge.NextViewFileNode(fn)
	return nv, nidx, true
}

// CallViewFile calls ViewFile with ActiveFilename set as arg
func (ge *CodeView) CallViewFile(ctx core.Widget) {
	ge.ConfigActiveFilename(views.NewSoloFuncButton(ctx, ge.ViewFile)).CallFunc()
}

// ViewFile views file in an existing TextEditor if it is already viewing that
// file, otherwise opens ViewFileNode in active buffer
func (ge *CodeView) ViewFile(fnm core.Filename) (*code.TextEditor, int, bool) { //types:add
	fn := ge.FileNodeForFile(string(fnm), true)
	if fn == nil {
		return nil, -1, false
	}
	tv, idx, ok := ge.TextEditorForFileNode(fn)
	if ok {
		ge.SetActiveTextEditorIndex(idx)
		return tv, idx, ok
	}
	tv = ge.ActiveTextEditor()
	idx = ge.ActiveTextEditorIndex
	ge.ViewFileNode(tv, idx, fn)
	return tv, idx, true
}

// ViewFileInIndex views file in given text view index
func (ge *CodeView) ViewFileInIndex(fnm core.Filename, idx int) (*code.TextEditor, int, bool) {
	fn := ge.FileNodeForFile(string(fnm), true)
	if fn == nil {
		return nil, -1, false
	}
	tv := ge.TextEditorByIndex(idx)
	ge.ViewFileNode(tv, idx, fn)
	return tv, idx, true
}

// LinkViewFileNode opens the file node in the 2nd texteditor, which is next to
// the tabs where links are clicked, if it is not collapsed -- else 1st
func (ge *CodeView) LinkViewFileNode(fn *filetree.Node) (*code.TextEditor, int) {
	if ge.PanelIsOpen(TextEditor2Index) {
		ge.SetActiveTextEditorIndex(1)
	} else {
		ge.SetActiveTextEditorIndex(0)
	}
	tv := ge.ActiveTextEditor()
	idx := ge.ActiveTextEditorIndex
	ge.ViewFileNode(tv, idx, fn)
	return tv, idx
}

// LinkViewFile opens the file in the 2nd texteditor, which is next to
// the tabs where links are clicked, if it is not collapsed -- else 1st
func (ge *CodeView) LinkViewFile(fnm core.Filename) (*code.TextEditor, int, bool) {
	fn := ge.FileNodeForFile(string(fnm), true)
	if fn == nil {
		return nil, -1, false
	}
	tv, idx, ok := ge.TextEditorForFileNode(fn)
	if ok {
		if idx == 1 {
			return tv, idx, true
		}
		if ge.SwapTextEditors() {
			return ge.TextEditorByIndex(1), 1, true
		}
	}
	nv, nidx := ge.LinkViewFileNode(fn)
	return nv, nidx, true
}

// ShowFile shows given file name at given line, returning TextEditor showing it
// or error if not found.
func (ge *CodeView) ShowFile(fname string, ln int) (*code.TextEditor, error) {
	tv, _, ok := ge.LinkViewFile(core.Filename(fname))
	if ok {
		tv.SetCursorTarget(lexer.Pos{Ln: ln - 1})
		return tv, nil
	}
	return nil, fmt.Errorf("ShowFile: file named: %v not found\n", fname)
}

// CodeViewOpenNodes gets list of open nodes for submenu-func
func CodeViewOpenNodes(it any, sc *core.Scene) []string {
	ge, ok := it.(tree.Node).(*CodeView)
	if !ok {
		return nil
	}
	return ge.OpenNodes.Strings()
}

// ViewOpenNodeName views given open node (by name) in active view
func (ge *CodeView) ViewOpenNodeName(name string) {
	nb := ge.OpenNodes.ByStringName(name)
	if nb == nil {
		return
	}
	tv := ge.ActiveTextEditor()
	ge.ViewFileNode(tv, ge.ActiveTextEditorIndex, nb)
}

// SelectOpenNode pops up a menu to select an open node (aka buffer) to view
// in current active texteditor
func (ge *CodeView) SelectOpenNode() {
	if len(ge.OpenNodes) == 0 {
		ge.SetStatus("No open nodes to choose from")
		return
	}
	nl := ge.OpenNodes.Strings()
	tv := ge.ActiveTextEditor() // nl[0] is always currently viewed
	def := nl[0]
	if len(nl) > 1 {
		def = nl[1]
	}
	m := core.NewMenuFromStrings(nl, def, func(idx int) {
		nb := ge.OpenNodes[idx]
		ge.ViewFileNode(tv, ge.ActiveTextEditorIndex, nb)
	})
	core.NewMenuStage(m, tv, tv.ContextMenuPos(nil)).Run()
}

// CloneActiveView sets the next text view to view the same file currently being vieweds
// in the active view. returns text view and index
func (ge *CodeView) CloneActiveView() (*code.TextEditor, int) { //types:add
	tv := ge.ActiveTextEditor()
	if tv == nil {
		return nil, -1
	}
	ond, _, got := ge.OpenNodeForTextEditor(tv)
	if got {
		nv, nidx := ge.NextTextEditor()
		ge.ViewFileNode(nv, nidx, ond)
		return nv, nidx
	}
	return nil, -1
}

// SaveAllOpenNodes saves all of the open filenodes to their current file names
func (ge *CodeView) SaveAllOpenNodes() {
	for _, ond := range ge.OpenNodes {
		if ond.Buffer == nil {
			continue
		}
		if ond.Buffer.IsNotSaved() {
			ond.Buffer.Save()
			ge.RunPostCmdsFileNode(ond)
		}
	}
}

// SaveAll saves all of the open filenodes to their current file names
// and saves the project state if it has been saved before (i.e., the .code file exists)
func (ge *CodeView) SaveAll() { //types:add
	ge.SaveAllOpenNodes()
	ge.SaveProjectIfExists(false)
}

// CloseOpenNodes closes any nodes with open views (including those in directories under nodes).
// called prior to rename.
func (ge *CodeView) CloseOpenNodes(nodes []*code.FileNode) {
	nn := len(ge.OpenNodes)
	for ni := nn - 1; ni >= 0; ni-- {
		ond := ge.OpenNodes[ni]
		if ond.Buffer == nil {
			continue
		}
		path := string(ond.Buffer.Filename)
		for _, cnd := range nodes {
			if strings.HasPrefix(path, string(cnd.FPath)) {
				ond.Buffer.Close(func(canceled bool) {
					if canceled {
						ge.SetStatus(fmt.Sprintf("File %v NOT closed -- recommended as file name changed!", ond.FPath))
						return
					}
					ge.SetStatus(fmt.Sprintf("File %v closed due to file name change", ond.FPath))
				})
				break // out of inner node loop
			}
		}
	}
}

// FileNodeRunExe runs the given executable file node
func (ge *CodeView) FileNodeRunExe(fn *filetree.Node) {
	ge.SetArgVarVals()
	ge.ArgVals["{PromptString1}"] = string(fn.FPath)
	code.CmdNoUserPrompt = true // don't re-prompt!
	cmd, _, ok := code.AvailableCommands.CmdByName(code.CmdName("Build: Run Prompt"), true)
	if ok {
		ge.ArgVals.Set(string(fn.FPath), &ge.Settings, nil)
		cbuf, _, _ := ge.RecycleCmdTab(cmd.Name, true, true)
		cmd.Run(ge, cbuf)
	}
}

// FileNodeOpened is called whenever file node is double-clicked in file tree
func (ge *CodeView) FileNodeOpened(fn *filetree.Node) {
	// todo: could add all these options in LangOpts
	switch fn.Info.Cat {
	case fileinfo.Folder:
	case fileinfo.Exe:
		ge.FileNodeRunExe(fn)
		// this uses exe path for cd to this path!
		return
	case fileinfo.Font, fileinfo.Video, fileinfo.Model, fileinfo.Audio, fileinfo.Sheet, fileinfo.Bin,
		fileinfo.Archive, fileinfo.Image:
		ge.ExecCmdNameFileNode(fn, code.CmdName("File: Open"), true, true) // sel, clear
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
		ge.ExecCmdNameFileNode(fn, code.CmdName("File: Open"), true, true) // sel, clear
		return
	}
	// program, document, data
	if int(fn.Info.Size) > core.SystemSettings.BigFileSize {
		d := core.NewBody().AddTitle("File is relatively large").
			AddText(fmt.Sprintf("The file: %v is relatively large at: %v; really open for editing?", fn.Nm, fn.Info.Size))
		d.AddBottomBar(func(parent core.Widget) {
			d.AddCancel(parent)
			core.NewButton(parent).SetText("Open").OnClick(func(e events.Event) {
				d.Close()
				ge.NextViewFileNode(fn)
			})
		})
		d.RunDialog(ge)
	} else {
		ge.NextViewFileNode(fn)
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
