// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/base/keylist"
	"cogentcore.org/core/base/metadata"
	"cogentcore.org/core/base/vcs"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/filetree"
	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/search"
	"cogentcore.org/core/text/textcore"
	"cogentcore.org/core/text/textpos"
)

// OpenFiles is the ordered list of open files that are being edited.
// The key is the full filepath, and a recency order is maintained.
type OpenFiles struct {
	keylist.List[string, *lines.Lines]
}

// Add adds given Lines buffer to list of open files.
// If already on the list, moves it to the top. Returns the Lines.
func (of *OpenFiles) Add(ln *lines.Lines) *lines.Lines {
	fpath := ln.Filename()
	if ix := of.IndexByKey(fpath); ix >= 0 {
		ln = of.Values[ix]
		of.Move(0, ix)
		return ln
	}
	of.List.Add(fpath, ln)
	return ln
}

// Move moves item at given index to destination index.
// Item is deleted first and then inserted at given index.
func (of *OpenFiles) Move(to, from int) {
	ln := of.Values[from]
	of.DeleteByIndex(from, from+1)
	of.Insert(to, ln.Filename(), ln)
}

// Paths returns the list of full paths of open files.
func (of *OpenFiles) Paths() []string {
	sl := make([]string, of.Len())
	for i, ln := range of.Values {
		sl[i] = ln.Filename()
	}
	return sl
}

// Strings returns a string list of nodes, with paths relative to proj root.
func (of *OpenFiles) Strings(root string) []string {
	sl := make([]string, of.Len())
	for i, ln := range of.Values {
		fpath := ln.Filename()
		_, fn := filepath.Split(fpath)
		rp := fsx.RelativeFilePath(fpath, root)
		rp = strings.TrimSuffix(rp, fn)
		if rp != "" {
			sl[i] = fn + " - " + rp
		} else {
			sl[i] = fn
		}
		if ln.IsNotSaved() {
			sl[i] += " *"
		}
	}
	return sl
}

// NChanged returns number of changed open files.
func (of *OpenFiles) NChanged() int {
	cnt := 0
	for _, ln := range of.Values {
		if ln.IsNotSaved() {
			cnt++
		}
	}
	return cnt
}

// SearchInPaths returns search results for given string literal with ignoreCase
// setting, in all open files contained directly in given paths. If paths is nil
// then all open files are searched.
func (of *OpenFiles) SearchInPaths(paths []string, find string, ignoreCase bool, langs []fileinfo.Known) []search.Results {
	var res []search.Results
	for _, ln := range of.Values {
		path := filepath.Dir(ln.Filename())
		if !(paths == nil || slices.Contains(paths, path)) {
			continue
		}
		if !search.LangCheck(ln.FileInfo(), langs) {
			continue
		}
		cnt, matches := ln.Search([]byte(find), ignoreCase, false)
		if cnt > 0 {
			res = append(res, search.Results{ln.Filename(), cnt, matches})
		}
	}
	return res
}

// SearchRegexpInPaths returns search results for given string literal with ignoreCase
// setting, in all open files contained directly in given paths. If paths is nil
// then all open files are searched.
func (of *OpenFiles) SearchRegexpInPaths(paths []string, re *regexp.Regexp, langs []fileinfo.Known) []search.Results {
	var res []search.Results
	for _, ln := range of.Values {
		path, _ := filepath.Split(ln.Filename())
		if !(paths == nil || slices.Contains(paths, path)) {
			continue
		}
		if !search.LangCheck(ln.FileInfo(), langs) {
			continue
		}
		cnt, matches := ln.SearchRegexp(re)
		if cnt > 0 {
			res = append(res, search.Results{ln.Filename(), cnt, matches})
		}
	}
	return res
}

// ReMarkup all open files: when mode color changes etc, during rebuild
func (of *OpenFiles) ReMarkup() {
	for _, ln := range of.Values {
		ln.SetHighlighting(core.AppearanceSettings.Highlighting)
		ln.ReMarkup()
	}
}

// SetVCSRepo sets a pointer to the [vcs.Repo] into given object's [metadata],
// used on Lines.
func SetVCSRepo(obj any, repo vcs.Repo) {
	metadata.Set(obj, "VCSRepo", repo)
}

// GetVCSRepo returns [vcs.Repo] from given object's [metadata].
// Returns nil if none or no metadata.
func GetVCSRepo(obj any) vcs.Repo {
	st, _ := metadata.Get[vcs.Repo](obj, "VCSRepo")
	return st
}

////////  Code file methods

// OpenLines opens a Lines buffer for given file path.
// Must already have determined that it is not in the list of OpenFiles,
// and it must be an Abs path.
// If file is outside the current root path, it is also added to
// external files in the file browser.
func (cv *Code) OpenLines(fpath string) *lines.Lines {
	ln := lines.NewLines()
	err := ln.Open(fpath)
	if errors.Log(err) != nil {
		return nil
	}
	cv.ConfigLines(ln)
	cv.OpenFiles.Add(ln)
	if !cv.InRootPath(fpath) {
		cv.Files.AddExternalFile(fpath)
	}
	fn := cv.FileNodeForFile(fpath)
	if fn != nil {
		if repo, _ := fn.Repo(); repo != nil {
			SetVCSRepo(ln, repo)
		}
		fn.FileIsOpen = true
	}
	return ln
}

// InRootPath returns true if the given path, which must be an absolute path,
// is under the current file browser root path.
func (cv *Code) InRootPath(fpath string) bool {
	return strings.HasPrefix(fpath, string(cv.Files.Filepath))
}

// RecycleFile either opens given file or returns already open one.
// Returns true if it is a new file, false otherwise.
// If file is outside the current root path, it is also added to
// external files in the file browser. Can return nil if not openable.
func (cv *Code) RecycleFile(fpath string) (*lines.Lines, bool) {
	fpath, _ = filepath.Abs(fpath)
	ln := cv.OpenFiles.At(fpath)
	if ln == nil {
		return cv.OpenLines(fpath), true
	}
	return cv.OpenFiles.Add(ln), false // moves to top
}

// GetOpenFile either returns the Lines for a file in OpenFiles.
func (cv *Code) GetOpenFile(fpath string) *lines.Lines {
	return cv.OpenFiles.At(fpath)
}

// SaveActiveView saves the contents of the currently active texteditor.
func (cv *Code) SaveActiveView() { //types:add
	tv := cv.ActiveEditor()
	if tv.Lines == nil {
		return
	}
	cv.LastSaveTStamp = time.Now()
	if tv.Lines.Filename() != "" {
		tv.Save()
		fname := tv.Lines.Filename()
		cv.SetStatus("File Saved: " + fname)
		fpath, _ := filepath.Split(fname)
		cv.Files.UpdatePath(fpath) // update everything in dir -- will have removed autosave
		cv.RunPostCmds(tv.Lines)
		cv.updatePreviewPanel()
	} else {
		core.CallFunc(cv, cv.SaveActiveViewAs)
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
	tv := cv.ActiveEditor()
	if tv.Lines == nil {
		return
	}
	cv.LastSaveTStamp = time.Now()
	ofn := tv.Lines.Filename()
	textcore.SaveAs(tv.Scene, tv.Lines, string(filename), func(canceled bool) {
		if canceled {
			cv.SetStatus(fmt.Sprintf("File %q NOT Saved As: %q", ofn, filename))
			return
		}
		cv.SetStatus(fmt.Sprintf("File %q Saved As: %q", ofn, filename))
		cv.Files.UpdatePath(string(filename)) // update everything in dir -- will have removed autosave
		if ofn != string(filename) {
			cv.OpenFiles.DeleteByKey(ofn)
			cv.OpenFiles.Add(tv.Lines)
		}
	})
	cv.SaveProjectIfExists(false) // no saveall
}

// RevertActiveView revert active view to saved version.
func (cv *Code) RevertActiveView() { //types:add
	tv := cv.ActiveEditor()
	if tv.Lines == nil {
		return
	}
	// cv.ConfigLines(tv.Lines) // why here?
	tv.Lines.Revert()
	tv.Lines.UndoReset() // key implication of revert
	fpath, _ := filepath.Split(tv.Lines.Filename())
	cv.Files.UpdatePath(fpath) // update everything in dir -- will have removed autosave
}

// CloseActiveView closes the buffer associated with active view.
func (cv *Code) CloseActiveView() { //types:add
	tv := cv.ActiveEditor()
	fpath := tv.Lines.Filename()
	tv.Close(func(canceled bool) {
		if canceled {
			cv.SetStatus("File NOT closed: " + fpath)
			return
		}
		cv.SetStatus("File closed: " + fpath)
		cv.OpenFiles.DeleteByKey(fpath)
	})
}

// RunPostCmds runs any registered post commands on the given open file.
// Returns true if commands were run and file was reverted after that.
// Uses MainLang to disambiguate if multiple languages associated with extension.
func (cv *Code) RunPostCmds(ln *lines.Lines) bool {
	if ln == nil {
		return false
	}
	lang := ln.FileInfo().Known
	lopt, has := AvailableLanguages[lang]
	if !has {
		return false
	}
	if len(lopt.PostSaveCmds) == 0 {
		return false
	}
	_, ptab := cv.Tabs().CurrentTab()
	cv.ExecCmdsFile(ln.Filename(), lopt.PostSaveCmds)
	if ptab >= 0 {
		cv.Tabs().SelectTabIndex(ptab) // we stay at the previous tab
	}
	ln.Revert()
	return true
}

// AutosaveCheck checks for an autosave file and prompts user about opening it.
// Returns true if autosave file does exist for a file that currently
// unchanged (means just opened).
func (cv *Code) AutosaveCheck(tv *TextEditor, vidx int, ln *lines.Lines) bool {
	fname := ln.Filename()
	if strings.HasPrefix(fname, "#") && strings.HasSuffix(fname, "#") {
		ln.Autosave = false
		return false // we are the autosave file
	}
	ln.Autosave = true
	if tv.IsNotSaved() || !ln.AutosaveCheck() {
		return false
	}
	cv.DiffFileLines(ln, ln.AutosaveFilename())
	d := core.NewBody("Autosave file exists")
	core.NewText(d).SetType(core.TextSupporting).SetText(fmt.Sprintf("An auto-save file for file: %v exists; open it in the other text view (you can then do Save As to replace current file)?  If you don't open it, the next change made will overwrite it with a new one, erasing any changes.", fname))
	d.AddBottomBar(func(bar *core.Frame) {
		core.NewButton(bar).SetText("Ignore and overwrite autosave file").OnClick(func(e events.Event) {
			d.Close()
			ln.AutosaveDelete()
			cv.Files.UpdatePath(ln.AutosaveFilename()) // will update dir
		})
		core.NewButton(bar).SetText("Open autosave file").OnClick(func(e events.Event) {
			d.Close()
			cv.NextViewFile(ln.AutosaveFilename())
		})
	})
	d.RunDialog(cv)
	return true
}

// CallViewFile calls ViewFile with ActiveFilename set as arg
func (cv *Code) CallViewFile(ctx core.Widget) {
	cv.ConfigActiveFilename(core.NewSoloFuncButton(ctx).SetFunc(cv.ViewFile)).CallFunc()
}

// ViewFile views file in an existing TextEditor if it is already viewing that
// file, otherwise opens ViewLines in active buffer.
func (cv *Code) ViewFile(fnm core.Filename) (*TextEditor, int, bool) { //types:add
	ln, nw := cv.RecycleFile(string(fnm))
	if ln == nil {
		return nil, -1, false
	}
	tv, idx, ok := cv.EditorForLines(ln)
	if ok {
		cv.SetActiveEditorIndex(idx)
		return tv, idx, ok
	}
	tv = cv.ActiveEditor()
	idx = cv.ActiveEditorIndex
	if nw {
		cv.AutosaveCheck(tv, idx, ln)
	}
	cv.ViewLines(tv, idx, ln)
	return tv, idx, true
}

// ViewLines sets the given text view to view file lines.
func (cv *Code) ViewLines(tv *TextEditor, vidx int, ln *lines.Lines) {
	if tv.IsNotSaved() {
		cv.SetStatus(fmt.Sprintf("Note: Changes not yet saved in file: %v", tv.Lines.Filename()))
	}
	tv.SetLines(ln)
	cv.OpenFiles.Add(ln)
	cv.SetActiveEditorIndex(vidx) // this calls FileModCheck
}

// NextViewFile sets the next text view to view given file name.
// Will use a more robust search of file tree if file path is not
// directly openable. Returns texteditor and its index, false if not found.
func (cv *Code) NextViewFile(fnm string) (*TextEditor, int, bool) { //types:add
	ln, nw := cv.RecycleFile(fnm)
	if ln == nil {
		fn, ok := cv.Files.FindFile(fnm)
		if ok {
			fnm = string(fn.Filepath)
			ln, nw = cv.RecycleFile(fnm)
			if ln == nil {
				return nil, -1, false
			}
		}
	}
	nv, nidx := cv.NextViewLines(ln)
	if nw {
		cv.AutosaveCheck(nv, nidx, ln)
	}
	return nv, nidx, true
}

// NextViewLines sets the next text view to view file in given lines.
// If already being viewed, that is activated, returns text view and index.
func (cv *Code) NextViewLines(ln *lines.Lines) (*TextEditor, int) {
	tv, idx, ok := cv.EditorForLines(ln)
	if ok {
		cv.SetActiveEditorIndex(idx)
		return tv, idx
	}
	nv, nidx := cv.NextTextEditor()
	// fmt.Println("next idx:", nidx)
	cv.ViewLines(nv, nidx, ln)
	return nv, nidx
}

// ViewFileInIndex views file in given text view index
func (cv *Code) ViewFileInIndex(fnm string, idx int) (*TextEditor, int, bool) {
	ln, _ := cv.RecycleFile(fnm)
	if ln == nil {
		return nil, -1, false
	}
	tv := cv.EditorByIndex(idx)
	cv.ViewLines(tv, idx, ln)
	return tv, idx, true
}

// SourceOfGenerated returns the source of the given generated file.
// This should be listed in a //line comment right after the first comment
// line that indicates that it is generated. Returns the input lines if
// source comment not found.
func (cv *Code) SourceOfGenerated(ln *lines.Lines) *lines.Lines {
	if ln.NumLines() < 2 {
		return ln
	}
	txt := string(ln.Line(1))
	lm := "//line "
	if !strings.HasPrefix(txt, lm) {
		return ln
	}
	src := txt[len(lm):]
	ci := strings.Index(src, ":")
	if ci < 0 {
		return ln
	}
	src = src[:ci]
	fpath, _ := filepath.Split(ln.Filename())
	sfp := filepath.Join(fpath, src)
	sln, _ := cv.RecycleFile(sfp)
	if sln != nil {
		return sln
	}
	return ln
}

// LinkViewFile opens the file in the 2nd texteditor, which is next to
// the tabs where links are clicked, if it is not collapsed, else 1st.
// It uses SourceOfGenerated to get the source of any generated files.
func (cv *Code) LinkViewFile(fnm string) (*TextEditor, int, bool) {
	ln, _ := cv.RecycleFile(fnm)
	if ln == nil {
		return nil, -1, false
	}
	if ln.FileInfo().Generated {
		ln = cv.SourceOfGenerated(ln)
	}
	tv, idx, ok := cv.EditorForLines(ln)
	if ok {
		if idx == 1 {
			return tv, idx, true
		}
		if cv.SwapTextEditors() {
			return cv.EditorByIndex(1), 1, true
		}
	}
	nv, nidx := cv.LinkViewLines(ln)
	return nv, nidx, true
}

// LinkViewLines opens the file Lines in the 2nd texteditor, which is next to
// the tabs where links are clicked, if it is not collapsed, else 1st.
func (cv *Code) LinkViewLines(ln *lines.Lines) (*TextEditor, int) {
	if cv.PanelIsOpen(TextEditor2Index) {
		cv.SetActiveEditorIndex(1)
	} else {
		cv.SetActiveEditorIndex(0)
	}
	tv := cv.ActiveEditor()
	idx := cv.ActiveEditorIndex
	cv.ViewLines(tv, idx, ln)
	return tv, idx
}

// ShowFile shows given file name at given line, returning TextEditor showing it
// or error if not found.
func (cv *Code) ShowFile(fpath string, ln int) (*TextEditor, error) {
	tv, _, ok := cv.LinkViewFile(fpath)
	if ok {
		tv.SetCursorTarget(textpos.Pos{Line: ln - 1})
		return tv, nil
	}
	return nil, fmt.Errorf("ShowFile: file named: %v not found\n", fpath)
}

// SelectOpenFile pops up a menu to select an open file to view
// in current active texteditor.
func (cv *Code) SelectOpenFile() {
	if cv.OpenFiles.Len() == 0 {
		cv.SetStatus("No open nodes to choose from")
		return
	}
	nl := cv.OpenFiles.Strings(string(cv.Files.Filepath))
	if len(nl) == 0 {
		return
	}
	tv := cv.ActiveEditor() // nl[0] is always currently viewed
	def := nl[0]
	if len(nl) > 1 {
		def = nl[1]
	}
	m := core.NewMenuFromStrings(nl, def, func(idx int) {
		ln := cv.OpenFiles.Values[idx]
		cv.ViewLines(tv, cv.ActiveEditorIndex, ln)
	})
	core.NewMenuStage(m, tv, tv.ContextMenuPos(nil)).Run()
}

// CloneActiveView sets the next text view to view the same file currently being vieweds
// in the active view. returns text view and index
func (cv *Code) CloneActiveView() (*TextEditor, int) { //types:add
	tv := cv.ActiveEditor()
	if tv == nil || tv.Lines == nil {
		return nil, -1
	}
	nv, nidx := cv.NextTextEditor()
	cv.ViewLines(nv, nidx, tv.Lines)
	return nv, nidx
}

// SaveAllOpenFiles saves all of the open filenodes to their current file names
func (cv *Code) SaveAllOpenFiles() {
	for _, ln := range cv.OpenFiles.Values {
		if ln.IsNotSaved() {
			textcore.Save(cv.Scene, ln)
			cv.RunPostCmds(ln)
		}
	}
}

// SaveAll saves all of the open filenodes to their current file names
// and saves the project state if it has been saved before (i.e., the .code file exists)
func (cv *Code) SaveAll() { //types:add
	cv.SaveAllOpenFiles()
	cv.SaveProjectIfExists(false)
}

// CloseOpenFiles closes any open files on the given list of filenames.
func (cv *Code) CloseOpenFiles(fnames []string) {
	for _, fnm := range fnames {
		fi, err := os.Stat(fnm)
		if errors.Log(err) != nil {
			cv.CloseOpenFile(fnm) // try anyway
			continue
		}
		if !fi.IsDir() {
			cv.CloseOpenFile(fnm)
			continue
		}
		// todo: all files in dir
	}
}

// CloseOpenFile closes given file if it is open.
func (cv *Code) CloseOpenFile(fname string) {
	ln := cv.OpenFiles.At(fname)
	if ln == nil {
		return
	}
	textcore.Close(cv.Scene, ln, func(canceled bool) {
		if canceled {
			cv.SetStatus(fmt.Sprintf("File %q NOT closed", fname))
			return
		}
		cv.SetStatus(fmt.Sprintf("File %q closed", fname))
		cv.OpenFiles.DeleteByKey(fname)
	})
}

////////  FileNode stuff

// FileNodeForFile returns file node for given file path.
// nil if not found or is a directory.
func (cv *Code) FileNodeForFile(fpath string) *filetree.Node {
	fn, ok := cv.Files.FindFile(fpath)
	if !ok {
		return nil
	}
	if fn.IsDir() {
		return nil
	}
	return fn
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
		cv.ExecCmdNameFile(string(fn.Filepath), CmdName("File: Open"))
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
		cv.ExecCmdNameFile(string(fn.Filepath), CmdName("File: Open"))
		return
	}
	// program, document, data
	if int(fn.Info.Size) > core.SystemSettings.BigFileSize {
		d := core.NewBody("File is relatively large")
		core.NewText(d).SetType(core.TextSupporting).SetText(fmt.Sprintf("The file: %v is relatively large at: %v; really open for editing?", fn.Name, fn.Info.Size))
		d.AddBottomBar(func(bar *core.Frame) {
			d.AddCancel(bar)
			core.NewButton(bar).SetText("Open").OnClick(func(e events.Event) {
				d.Close()
				cv.NextViewFile(string(fn.Filepath))
			})
		})
		d.RunDialog(cv)
	} else {
		cv.NextViewFile(string(fn.Filepath))
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
