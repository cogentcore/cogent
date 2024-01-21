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
	"cogentcore.org/core/events"
	"cogentcore.org/core/fi"
	"cogentcore.org/core/filetree"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/giv"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/pi/lex"
	"cogentcore.org/core/texteditor"
)

// SaveActiveView saves the contents of the currently-active textview
func (ge *CodeView) SaveActiveView() { //gti:add
	tv := ge.ActiveTextEditor()
	if tv.Buf != nil {
		ge.LastSaveTStamp = time.Now()
		if tv.Buf.Filename != "" {
			tv.Buf.Save()
			ge.SetStatus("File Saved")
			fnm := string(tv.Buf.Filename)
			fpath, _ := filepath.Split(fnm)
			ge.Files.UpdatePath(fpath) // update everything in dir -- will have removed autosave
			ge.RunPostCmdsActiveView()
		} else {
			giv.CallFunc(ge, ge.SaveActiveViewAs)
		}
	}
	ge.SaveProjIfExists(false) // no saveall
}

// ConfigActiveFilename configures the first arg of given FuncButton to
// use the ActiveFilename
func (ge *CodeView) ConfigActiveFilename(fb *giv.FuncButton) *giv.FuncButton {
	fb.Args[0].SetValue(ge.ActiveFilename)
	return fb
}

func (ge *CodeView) CallSaveActiveViewAs(ctx gi.Widget) {
	ge.ConfigActiveFilename(giv.NewSoloFuncButton(ctx, ge.SaveActiveViewAs)).CallFunc()
}

// SaveActiveViewAs save with specified filename the contents of the
// currently-active textview
func (ge *CodeView) SaveActiveViewAs(filename gi.Filename) { //gti:add
	tv := ge.ActiveTextEditor()
	if tv.Buf != nil {
		ge.LastSaveTStamp = time.Now()
		ofn := tv.Buf.Filename
		tv.Buf.SaveAsFunc(filename, func(canceled bool) {
			if canceled {
				ge.SetStatus(fmt.Sprintf("File %v NOT Saved As: %v", ofn, filename))
				return
			}
			ge.SetStatus(fmt.Sprintf("File %v Saved As: %v", ofn, filename))
			// ge.RunPostCmdsActiveView() // doesn't make sense..
			ge.Files.UpdatePath(string(filename)) // update everything in dir -- will have removed autosave
			fn, ok := ge.Files.FindFile(string(filename))
			if ok {
				if fn.Buf != nil {
					fn.Buf.Revert()
				}
				ge.ViewFileNode(tv, ge.ActiveTextEditorIdx, fn)
			}
		})
	}
	ge.SaveProjIfExists(false) // no saveall
}

// RevertActiveView revert active view to saved version
func (ge *CodeView) RevertActiveView() { //gti:add
	tv := ge.ActiveTextEditor()
	if tv.Buf != nil {
		ge.ConfigTextBuf(tv.Buf)
		tv.Buf.Revert()
		tv.Buf.Undos.Reset() // key implication of revert
		fpath, _ := filepath.Split(string(tv.Buf.Filename))
		ge.Files.UpdatePath(fpath) // update everything in dir -- will have removed autosave
	}
}

// CloseActiveView closes the buffer associated with active view
func (ge *CodeView) CloseActiveView() { //gti:add
	tv := ge.ActiveTextEditor()
	ond, _, got := ge.OpenNodeForTextEditor(tv)
	if got {
		ond.Buf.Close(func(canceled bool) {
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
	if lopt, has := code.AvailLangs[lang]; has {
		if len(lopt.PostSaveCmds) > 0 {
			ge.ExecCmdsFileNode(fn, lopt.PostSaveCmds, false, true) // no select, yes clear
			fn.Buf.Revert()
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
		fn.Buf.Autosave = false
		return false // we are the autosave file
	}
	fn.Buf.Autosave = true
	if tv.IsNotSaved() || !fn.Buf.AutoSaveCheck() {
		return false
	}
	ge.DiffFileNode(fn, gi.Filename(fn.Buf.AutoSaveFilename()))
	d := gi.NewBody().AddTitle("Autosave file Exists").
		AddText(fmt.Sprintf("An auto-save file for file: %v exists -- open it in the other text view (you can then do Save As to replace current file)?  If you don't open it, the next change made will overwrite it with a new one, erasing any changes.", fn.Nm))
	d.AddBottomBar(func(pw gi.Widget) {
		gi.NewButton(pw).SetText("Ignore and overwrite autosave file").OnClick(func(e events.Event) {
			d.Close()
			fn.Buf.AutoSaveDelete()
			ge.Files.UpdatePath(fn.Buf.AutoSaveFilename()) // will update dir
		})
		gi.NewButton(pw).SetText("Open autosave file").OnClick(func(e events.Event) {
			d.Close()
			ge.NextViewFile(gi.Filename(fn.Buf.AutoSaveFilename()))
		})
	})
	d.NewDialog(ge).Run()
	return true
}

// OpenFileNode opens file for file node -- returns new bool and error
func (ge *CodeView) OpenFileNode(fn *filetree.Node) (bool, error) {
	if fn.IsDir() {
		return false, fmt.Errorf("cannot open directory: %v", fn.FPath)
	}
	filetree.NodeHiStyle = gi.AppearanceSettings.HiStyle // must be set prior to OpenBuf
	nw, err := fn.OpenBuf()
	if err == nil {
		ge.ConfigTextBuf(fn.Buf)
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
	updt := ge.UpdateStart()
	defer ge.UpdateEndLayout(updt)

	if tv.IsNotSaved() {
		ge.SetStatus(fmt.Sprintf("Note: Changes not yet saved in file: %v", tv.Buf.Filename))
	}
	nw, err := ge.OpenFileNode(fn)
	if err == nil {
		tv.SetBuf(fn.Buf)
		if nw {
			ge.AutoSaveCheck(tv, vidx, fn)
		}
		ge.SetActiveTextEditorIdx(vidx) // this calls FileModCheck
	}
}

// NextViewFileNode sets the next text view to view file in given node (opens
// buffer if not already opened) -- if already being viewed, that is
// activated, returns text view and index
func (ge *CodeView) NextViewFileNode(fn *filetree.Node) (*code.TextEditor, int) {
	tv, idx, ok := ge.TextEditorForFileNode(fn)
	if ok {
		ge.SetActiveTextEditorIdx(idx)
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
func (ge *CodeView) TextBufForFile(fpath string, add bool) *texteditor.Buf {
	fn := ge.FileNodeForFile(fpath, add)
	if fn == nil {
		return nil
	}
	_, err := ge.OpenFileNode(fn)
	if err == nil {
		return fn.Buf
	}
	return nil
}

// NextViewFile sets the next text view to view given file name -- include as
// much of name as possible to disambiguate -- will use the first matching --
// if already being viewed, that is activated -- returns textview and its
// index, false if not found
func (ge *CodeView) NextViewFile(fnm gi.Filename) (*code.TextEditor, int, bool) { //gti:add
	fn := ge.FileNodeForFile(string(fnm), true)
	if fn == nil {
		return nil, -1, false
	}
	nv, nidx := ge.NextViewFileNode(fn)
	return nv, nidx, true
}

// CallViewFile calls ViewFile with ActiveFilename set as arg
func (ge *CodeView) CallViewFile(ctx gi.Widget) {
	ge.ConfigActiveFilename(giv.NewSoloFuncButton(ctx, ge.ViewFile)).CallFunc()
}

// ViewFile views file in an existing TextEditor if it is already viewing that
// file, otherwise opens ViewFileNode in active buffer
func (ge *CodeView) ViewFile(fnm gi.Filename) (*code.TextEditor, int, bool) { //gti:add
	fn := ge.FileNodeForFile(string(fnm), true)
	if fn == nil {
		return nil, -1, false
	}
	tv, idx, ok := ge.TextEditorForFileNode(fn)
	if ok {
		ge.SetActiveTextEditorIdx(idx)
		return tv, idx, ok
	}
	tv = ge.ActiveTextEditor()
	idx = ge.ActiveTextEditorIdx
	ge.ViewFileNode(tv, idx, fn)
	return tv, idx, true
}

// ViewFileInIdx views file in given text view index
func (ge *CodeView) ViewFileInIdx(fnm gi.Filename, idx int) (*code.TextEditor, int, bool) {
	fn := ge.FileNodeForFile(string(fnm), true)
	if fn == nil {
		return nil, -1, false
	}
	tv := ge.TextEditorByIndex(idx)
	ge.ViewFileNode(tv, idx, fn)
	return tv, idx, true
}

// LinkViewFileNode opens the file node in the 2nd textview, which is next to
// the tabs where links are clicked, if it is not collapsed -- else 1st
func (ge *CodeView) LinkViewFileNode(fn *filetree.Node) (*code.TextEditor, int) {
	updt := ge.UpdateStart()
	defer ge.UpdateEndLayout(updt)

	if ge.PanelIsOpen(TextEditor2Idx) {
		ge.SetActiveTextEditorIdx(1)
	} else {
		ge.SetActiveTextEditorIdx(0)
	}
	tv := ge.ActiveTextEditor()
	idx := ge.ActiveTextEditorIdx
	ge.ViewFileNode(tv, idx, fn)
	return tv, idx
}

// LinkViewFile opens the file in the 2nd textview, which is next to
// the tabs where links are clicked, if it is not collapsed -- else 1st
func (ge *CodeView) LinkViewFile(fnm gi.Filename) (*code.TextEditor, int, bool) {
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
	tv, _, ok := ge.LinkViewFile(gi.Filename(fname))
	if ok {
		tv.SetCursorTarget(lex.Pos{Ln: ln - 1})
		return tv, nil
	}
	return nil, fmt.Errorf("ShowFile: file named: %v not found\n", fname)
}

// CodeViewOpenNodes gets list of open nodes for submenu-func
func CodeViewOpenNodes(it any, sc *gi.Scene) []string {
	ge, ok := it.(ki.Ki).(*CodeView)
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
	ge.ViewFileNode(tv, ge.ActiveTextEditorIdx, nb)
}

// SelectOpenNode pops up a menu to select an open node (aka buffer) to view
// in current active textview
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
	m := gi.NewMenuFromStrings(nl, def, func(idx int) {
		nb := ge.OpenNodes[idx]
		ge.ViewFileNode(tv, ge.ActiveTextEditorIdx, nb)
	})
	gi.NewMenuStage(m, tv, tv.ContextMenuPos(nil)).Run()
}

// CloneActiveView sets the next text view to view the same file currently being vieweds
// in the active view. returns text view and index
func (ge *CodeView) CloneActiveView() (*code.TextEditor, int) { //gti:add
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
		if ond.Buf == nil {
			continue
		}
		if ond.Buf.IsNotSaved() {
			ond.Buf.Save()
			ge.RunPostCmdsFileNode(ond)
		}
	}
}

// SaveAll saves all of the open filenodes to their current file names
// and saves the project state if it has been saved before (i.e., the .code file exists)
func (ge *CodeView) SaveAll() { //gti:add
	ge.SaveAllOpenNodes()
	ge.SaveProjIfExists(false)
}

// CloseOpenNodes closes any nodes with open views (including those in directories under nodes).
// called prior to rename.
func (ge *CodeView) CloseOpenNodes(nodes []*code.FileNode) {
	nn := len(ge.OpenNodes)
	for ni := nn - 1; ni >= 0; ni-- {
		ond := ge.OpenNodes[ni]
		if ond.Buf == nil {
			continue
		}
		path := string(ond.Buf.Filename)
		for _, cnd := range nodes {
			if strings.HasPrefix(path, string(cnd.FPath)) {
				ond.Buf.Close(func(canceled bool) {
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
	cmd, _, ok := code.AvailCmds.CmdByName(code.CmdName("Build: Run Prompt"), true)
	if ok {
		ge.ArgVals.Set(string(fn.FPath), &ge.Prefs, nil)
		cbuf, _, _ := ge.RecycleCmdTab(cmd.Name, true, true)
		cmd.Run(ge, cbuf)
	}
}

// FileNodeOpened is called whenever file node is double-clicked in file tree
func (ge *CodeView) FileNodeOpened(fn *filetree.Node) {
	// todo: could add all these options in LangOpts
	switch fn.Info.Cat {
	case fi.Folder:
	case fi.Exe:
		ge.FileNodeRunExe(fn)
		// this uses exe path for cd to this path!
		return
	case fi.Font, fi.Video, fi.Model, fi.Audio, fi.Sheet, fi.Bin,
		fi.Archive, fi.Image:
		ge.ExecCmdNameFileNode(fn, code.CmdName("File: Open"), true, true) // sel, clear
		return
	}

	edit := true
	switch fn.Info.Cat {
	case fi.Code:
		edit = true
	case fi.Text:
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
	if int(fn.Info.Size) > gi.SystemSettings.BigFileSize {
		d := gi.NewBody().AddTitle("File is relatively large").
			AddText(fmt.Sprintf("The file: %v is relatively large at: %v -- really open for editing?", fn.Nm, fn.Info.Size))
		d.AddBottomBar(func(pw gi.Widget) {
			d.AddCancel(pw)
			gi.NewButton(pw).SetText("Open").OnClick(func(e events.Event) {
				d.Close()
				ge.NextViewFileNode(fn)
			})
		})
		d.NewDialog(ge).Run()
	} else {
		ge.NextViewFileNode(fn)
	}

}

// CatNoEdit are the files to NOT edit from categories: Doc, Data
var CatNoEdit = map[fi.Known]bool{
	fi.Rtf:          true,
	fi.MSWord:       true,
	fi.OpenText:     true,
	fi.OpenPres:     true,
	fi.MSPowerpoint: true,
	fi.EBook:        true,
	fi.EPub:         true,
}
