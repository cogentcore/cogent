// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/filetree"
	"cogentcore.org/core/text/lines"

	"cogentcore.org/cogent/code/lsp"
)

// filesChanged handles the changes to files on disk detected by the
// file tree watcher ([filetree.Tree.WatchFunc]), made by other programs
// or by a version control checkout, for example. Open files that have
// not been edited are reverted to the new contents, and those that
// have been edited prompt about it if they are shown in an editor (and
// otherwise when they are next shown). Deleted open files are closed,
// prompting first if they have been edited, and the language servers
// are notified of all of the changes.
func (cv *Code) filesChanged(changes []filetree.FileChange) {
	var lspChanges []lsp.FileChange
	for _, c := range changes {
		switch {
		case c.Removed:
			lspChanges = append(lspChanges, lsp.FileChange{Path: c.Path, Type: lsp.FileDeleted})
		case !c.IsDir:
			lspChanges = append(lspChanges, lsp.FileChange{Path: c.Path, Type: lsp.FileChanged})
		}
	}
	cv.LangSvc().FilesChanged(lspChanges)

	var deleted []*lines.Lines
	for _, ln := range slices.Clone(cv.OpenFiles.Values) { // closing deletes
		fname := ln.Filename()
		if fname == "" || !fileAffected(realPath(fname), changes) {
			continue
		}
		if _, err := os.Stat(fname); errors.Is(err, fs.ErrNotExist) {
			switch {
			case !ln.IsNotSaved():
				cv.CloseOpenFile(fname)
				cv.SetStatus("Closed file deleted on disk: " + cv.relPath(fname))
			case !cv.deletedAsked[ln]:
				deleted = append(deleted, ln)
			}
			continue
		}
		delete(cv.deletedAsked, ln)
		cv.fileModified(ln)
	}
	for ln := range cv.deletedAsked { // forget closed files
		if cv.OpenFiles.IndexByKey(ln.Filename()) < 0 {
			delete(cv.deletedAsked, ln)
		}
	}
	if len(deleted) > 0 {
		cv.promptDeleted(deleted)
	}
}

// realPath returns the given absolute file path with the symbolic
// links in its directory evaluated, as in the paths of the file tree,
// which evaluates those of its root.
func realPath(fname string) string {
	dir, base := filepath.Split(fname)
	if d, err := filepath.EvalSymlinks(dir); err == nil {
		return filepath.Join(d, base)
	}
	return fname
}

// fileAffected returns whether the given file may have been changed
// by one of the given changes: a change to it, or to a directory
// containing it (including a removed path, which may have been one).
func fileAffected(fname string, changes []filetree.FileChange) bool {
	for _, c := range changes {
		if c.Path == fname {
			return true
		}
		if (c.IsDir || c.Removed) && strings.HasPrefix(fname, c.Path+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

// fileModified handles an open file that may have been changed on
// disk: if it has not been edited, it is reverted to the contents on
// disk, and otherwise the user is prompted about it if it is shown in
// an editor; editors check that whenever they show a file.
func (cv *Code) fileModified(ln *lines.Lines) {
	if !ln.IsNotSaved() {
		if ln.FileModCheck() { // reverts it
			cv.LangSvc().Change(ln)
			cv.SetStatus("Reloaded file changed on disk: " + cv.relPath(ln.Filename()))
		}
		return
	}
	if cv.editorForLines(ln) != nil {
		ln.FileModCheck() // prompts
	}
}

// editorForLines returns the text editor showing the given lines, or nil.
func (cv *Code) editorForLines(ln *lines.Lines) *TextEditor {
	for i := range NTextEditors {
		if tv := cv.EditorByIndex(i); tv != nil && tv.Lines == ln {
			return tv
		}
	}
	return nil
}

// relPath returns the given path relative to the project root.
func (cv *Code) relPath(fname string) string {
	return fsx.RelativeFilePath(fname, string(cv.ProjectRoot))
}

// promptDeleted prompts the user to close the given edited open files,
// which have been deleted on disk, or to keep them open.
// Each file is only prompted about once while it is deleted.
func (cv *Code) promptDeleted(lns []*lines.Lines) {
	if cv.deletedAsked == nil {
		cv.deletedAsked = make(map[*lines.Lines]bool)
	}
	names := make([]string, len(lns))
	for i, ln := range lns {
		cv.deletedAsked[ln] = true
		names[i] = "<code>" + cv.relPath(ln.Filename()) + "</code>"
	}
	title, these, it := "File deleted on disk", "This edited file was", "it"
	if len(lns) > 1 {
		title, these, it = "Files deleted on disk", "These edited files were", "them"
	}
	d := core.NewBody(title)
	core.NewText(d).SetType(core.TextSupporting).SetText(fmt.Sprintf("%s deleted or renamed on disk: %s. Do you want to close %s, or keep %s open? Saving writes the file to disk again.", these, strings.Join(names, ", "), it, it))
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar).SetText("Keep open")
		d.AddOK(bar).SetText("Close").OnClick(func(e events.Event) {
			// closing prompts to save the edits
			for _, ln := range lns {
				if fname := ln.Filename(); fname != "" {
					cv.CloseOpenFile(fname)
				}
			}
		})
	})
	d.RunDialog(cv)
}
