// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"fmt"
	"log"

	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/filetree"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/textcore"
	"cogentcore.org/core/text/textpos"
	"cogentcore.org/core/tree"
)

// ConfigLines configures the text buffer according to the settings.
func (cv *Code) ConfigLines(tb *lines.Lines) {
	tb.SetHighlighting(core.AppearanceSettings.Highlighting)
	tb.Settings.EditorSettings = cv.Settings.Editor
	tb.ConfigKnown()
}

// ActiveEditor returns the currently active TextEditor
func (cv *Code) ActiveEditor() *TextEditor {
	//	fmt.Printf("stdout: active text view idx: %v\n", ge.ActiveEditorIndex)
	return cv.EditorByIndex(cv.ActiveEditorIndex)
}

// FocusActiveEditor sets focus to active text editor
func (cv *Code) FocusActiveEditor() *TextEditor {
	return cv.SetActiveEditorIndex(cv.ActiveEditorIndex)
}

// ActiveLines returns the Lines for the active file; nil if none
func (cv *Code) ActiveLines() *lines.Lines {
	return cv.GetOpenFile(string(cv.ActiveFilename))
}

// EditorIndex finds index of given texteditor (0 or 1)
func (cv *Code) EditorIndex(av *TextEditor) int {
	for i := 0; i < NTextEditors; i++ {
		tv := cv.EditorByIndex(i)
		if tv.This == av.This {
			return i
		}
	}
	return -1 // shouldn't happen
}

// EditorForLines finds a Editor that is viewing given Lines,
// and its index, or false if none is
func (cv *Code) EditorForLines(ln *lines.Lines) (*TextEditor, int, bool) {
	for i := 0; i < NTextEditors; i++ {
		tv := cv.EditorByIndex(i)
		if tv != nil && tv.Lines == ln && cv.PanelIsOpen(i+TextEditor1Index) {
			return tv, i, true
		}
	}
	return nil, -1, false
}

// SetActiveFileInfo sets the active file info from Lines.
func (cv *Code) SetActiveFileInfo(ln *lines.Lines) {
	cv.ActiveFilename = fsx.Filename(ln.Filename())
	cv.ActiveLang = ln.FileInfo().Known
	cv.ActiveVCSInfo = ""
	cv.ActiveVCS = nil
	repo := GetVCSRepo(ln)
	if repo != nil {
		cv.ActiveVCS = repo
		cur, err := repo.Current()
		if err == nil {
			cv.ActiveVCSInfo = fmt.Sprintf("%s: <i>%s</i>", repo.Vcs(), cur)
		}
	}
	fn := cv.FileNodeForFile(ln.Filename())
	if fn != nil {
		fn.ScrollToThis()
	}
}

// SetActiveEditor sets the given texteditor as the active one,
// configures it, and returns its index.
func (cv *Code) SetActiveEditor(av *TextEditor) int {
	cv.ActiveEditorIndex = cv.EditorIndex(av)
	if av.Lines != nil {
		if av.Complete != nil {
			av.Complete.LookupFunc = cv.LookupFun
		}
		cv.SetActiveFileInfo(av.Lines)
		av.Lines.FileModCheck()
	}
	cv.SetStatus("")
	return cv.ActiveEditorIndex
}

// SetActiveEditorIndex sets the given view index as the currently active
// Editor -- returns that texteditor.  This is the main method for
// activating a text editor.
func (cv *Code) SetActiveEditorIndex(idx int) *TextEditor {
	if idx < 0 || idx >= NTextEditors {
		log.Printf("Code SetActiveEditorIndex: text view index out of range: %v\n", idx)
		return nil
	}
	cv.ActiveEditorIndex = idx
	av := cv.ActiveEditor()
	cv.SetActiveEditor(av)
	av.SetFocus()
	return av
}

// NextTextEditor returns the next text view available for viewing a file and
// its index -- if the active text view is empty, then it is used, otherwise
// it is the next one (if visible)
func (cv *Code) NextTextEditor() (*TextEditor, int) {
	av := cv.EditorByIndex(cv.ActiveEditorIndex)
	if av.Lines == nil {
		return av, cv.ActiveEditorIndex
	}
	nxt := (cv.ActiveEditorIndex + 1) % NTextEditors
	if !cv.PanelIsOpen(nxt + TextEditor1Index) {
		return av, cv.ActiveEditorIndex
	}
	return cv.EditorByIndex(nxt), nxt
}

// SwapTextEditors switches the buffers for the two open texteditors
// only operates if both panels are open.
func (cv *Code) SwapTextEditors() bool {
	if !cv.PanelIsOpen(TextEditor1Index) || !cv.PanelIsOpen(TextEditor1Index+1) {
		return false
	}
	tva := cv.EditorByIndex(0)
	tvb := cv.EditorByIndex(1)
	bufa := tva.Lines
	bufb := tvb.Lines
	tva.SetLines(bufb)
	tvb.SetLines(bufa)
	cv.SetStatus("swapped buffers")
	return true
}

func (cv *Code) OpenFileAtRegion(filename string, tr textpos.Region) (tv *TextEditor, ok bool) {
	tv, _, ok = cv.LinkViewFile(filename)
	if tv == nil {
		return nil, false
	}
	tv.Highlights = tv.Highlights[:0]
	tv.Highlights = append(tv.Highlights, tr)
	tv.SetCursorTarget(tr.Start)
	tv.SetFocus()
	tv.NeedsLayout()
	return tv, true
}

// OpenFindURL opens given find:/// url from Find; delegates to FindPanel
func (cv *Code) OpenFindURL(ur string, ftv *textcore.Editor) bool {
	fv := tree.ParentByType[*FindPanel](ftv)
	if fv == nil {
		return false
	}
	return fv.OpenFindURL(ur, ftv)
}

// UpdateTextButtons updates texteditor menu buttons is called by SetStatus.
// Doesn't do anything unless a change is required, so safe to call frequently.
func (cv *Code) UpdateTextButtons() {
	ati := cv.ActiveEditorIndex
	for i := 0; i < NTextEditors; i++ {
		tv := cv.EditorByIndex(i)
		mb := cv.EditorButtonByIndex(i)
		txnm := "<no file>"
		if tv.Lines != nil {
			txnm = fsx.DirAndFile(tv.Lines.Filename())
			if tv.Lines.IsNotSaved() {
				txnm += " <b>*</b>"
			} else {
				txnm += "   "
			}
		}
		sel := ati == i
		if sel != mb.StateIs(states.Selected) {
			mb.SetSelected(sel)
		}
		if mb.Text != txnm {
			mb.SetText(txnm)
			mb.Update()
		}
	}
}

// FileNodeSelected is called whenever tree browser has file node selected
func (cv *Code) FileNodeSelected(fn *filetree.Node) {
	// not doing anything with this actually
}

func (cv *Code) EditorButtonMenu(idx int, m *core.Scene) {
	tv := cv.EditorByIndex(idx)
	opn := cv.OpenFiles.Strings(string(cv.Files.Filepath))
	core.NewButton(m).SetText("Open File...").OnClick(func(e events.Event) {
		cv.CallViewFile(tv)
	})
	core.NewSeparator(m)
	for i, n := range opn {
		core.NewButton(m).SetText(n).OnClick(func(e events.Event) {
			cv.ViewLines(tv, idx, cv.OpenFiles.Values[i])
		})
	}
}
