// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"fmt"
	"log"
	"net/url"
	"strings"

	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/filetree"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/texteditor"
	"cogentcore.org/core/texteditor/textbuf"
	"cogentcore.org/core/tree"
)

// ConfigTextBuffer configures the text buffer according to the settings.
func (cv *Code) ConfigTextBuffer(tb *texteditor.Buffer) {
	tb.SetHiStyle(core.AppearanceSettings.HiStyle)
	tb.Options.EditorSettings = cv.Settings.Editor
	tb.ConfigKnown()
	if tb.Complete != nil {
		tb.Complete.LookupFunc = cv.LookupFun
	}
}

// ActiveTextEditor returns the currently active TextEditor
func (cv *Code) ActiveTextEditor() *TextEditor {
	//	fmt.Printf("stdout: active text view idx: %v\n", ge.ActiveTextEditorIndex)
	return cv.TextEditorByIndex(cv.ActiveTextEditorIndex)
}

// FocusActiveTextEditor sets focus to active text editor
func (cv *Code) FocusActiveTextEditor() *TextEditor {
	return cv.SetActiveTextEditorIndex(cv.ActiveTextEditorIndex)
}

// ActiveFileNode returns the file node for the active file -- nil if none
func (cv *Code) ActiveFileNode() *filetree.Node {
	return cv.FileNodeForFile(string(cv.ActiveFilename), false)
}

// TextEditorIndex finds index of given texteditor (0 or 1)
func (cv *Code) TextEditorIndex(av *TextEditor) int {
	for i := 0; i < NTextEditors; i++ {
		tv := cv.TextEditorByIndex(i)
		if tv.This == av.This {
			return i
		}
	}
	return -1 // shouldn't happen
}

// TextEditorForFileNode finds a TextEditor that is viewing given FileNode,
// and its index, or false if none is
func (cv *Code) TextEditorForFileNode(fn *filetree.Node) (*TextEditor, int, bool) {
	if fn.Buffer == nil {
		return nil, -1, false
	}
	cv.ConfigTextBuffer(fn.Buffer)
	for i := 0; i < NTextEditors; i++ {
		tv := cv.TextEditorByIndex(i)
		if tv != nil && tv.Buffer != nil && tv.Buffer == fn.Buffer && cv.PanelIsOpen(i+TextEditor1Index) {
			return tv, i, true
		}
	}
	return nil, -1, false
}

// OpenNodeForTextEditor finds the FileNode that a given TextEditor is
// viewing, returning its index within OpenNodes list, or false if not found
func (cv *Code) OpenNodeForTextEditor(tv *TextEditor) (*filetree.Node, int, bool) {
	if tv.Buffer == nil {
		return nil, -1, false
	}
	for i, ond := range cv.OpenNodes {
		if ond.Buffer == tv.Buffer {
			return ond, i, true
		}
	}
	return nil, -1, false
}

// TextEditorForFile finds FileNode for file, and returns TextEditor and index
// that is viewing that FileNode, or false if none is
func (cv *Code) TextEditorForFile(fnm core.Filename) (*TextEditor, int, bool) {
	fn, ok := cv.Files.FindFile(string(fnm))
	if !ok {
		return nil, -1, false
	}
	return cv.TextEditorForFileNode(fn)
}

// SetActiveFileInfo sets the active file info from textbuf
func (cv *Code) SetActiveFileInfo(buf *texteditor.Buffer) {
	cv.ActiveFilename = buf.Filename
	cv.ActiveLang = buf.Info.Known
	cv.ActiveVCSInfo = ""
	cv.ActiveVCS = nil
	fn := cv.FileNodeForFile(string(cv.ActiveFilename), false)
	if fn != nil {
		repo, _ := fn.Repo()
		if repo != nil {
			cv.ActiveVCS = repo
			cur, err := repo.Current()
			if err == nil {
				cv.ActiveVCSInfo = fmt.Sprintf("%s: <i>%s</i>", repo.Vcs(), cur)
			}
		}
		fn.ScrollToThis()
	}
}

// SetActiveTextEditor sets the given texteditor as the active one, and returns its index
func (cv *Code) SetActiveTextEditor(av *TextEditor) int {
	idx := cv.TextEditorIndex(av)
	if idx < 0 {
		fmt.Println("te not found")
		return -1
	}
	cv.ActiveTextEditorIndex = idx
	if av.Buffer != nil {
		cv.SetActiveFileInfo(av.Buffer)
	}
	cv.SetStatus("")
	return idx
}

// SetActiveTextEditorIndex sets the given view index as the currently active
// TextEditor -- returns that texteditor.  This is the main method for
// activating a text editor.
func (cv *Code) SetActiveTextEditorIndex(idx int) *TextEditor {
	if idx < 0 || idx >= NTextEditors {
		log.Printf("Code SetActiveTextEditorIndex: text view index out of range: %v\n", idx)
		return nil
	}
	cv.ActiveTextEditorIndex = idx
	av := cv.ActiveTextEditor()
	if av.Buffer != nil {
		cv.SetActiveFileInfo(av.Buffer)
		av.Buffer.FileModCheck()
	}
	cv.SetStatus("")
	av.SetFocusEvent()
	return av
}

// NextTextEditor returns the next text view available for viewing a file and
// its index -- if the active text view is empty, then it is used, otherwise
// it is the next one (if visible)
func (cv *Code) NextTextEditor() (*TextEditor, int) {
	av := cv.TextEditorByIndex(cv.ActiveTextEditorIndex)
	if av.Buffer == nil {
		return av, cv.ActiveTextEditorIndex
	}
	nxt := (cv.ActiveTextEditorIndex + 1) % NTextEditors
	if !cv.PanelIsOpen(nxt + TextEditor1Index) {
		return av, cv.ActiveTextEditorIndex
	}
	return cv.TextEditorByIndex(nxt), nxt
}

// SwapTextEditors switches the buffers for the two open texteditors
// only operates if both panels are open
func (cv *Code) SwapTextEditors() bool {
	if !cv.PanelIsOpen(TextEditor1Index) || !cv.PanelIsOpen(TextEditor1Index+1) {
		return false
	}

	tva := cv.TextEditorByIndex(0)
	tvb := cv.TextEditorByIndex(1)
	bufa := tva.Buffer
	bufb := tvb.Buffer
	tva.SetBuffer(bufb)
	tvb.SetBuffer(bufa)
	cv.SetStatus("swapped buffers")
	return true
}

func (cv *Code) OpenFileAtRegion(filename core.Filename, tr textbuf.Region) (tv *TextEditor, ok bool) {
	tv, _, ok = cv.LinkViewFile(filename)
	if tv == nil {
		return nil, false
	}

	tv.Highlights = tv.Highlights[:0]
	tv.Highlights = append(tv.Highlights, tr)
	tv.SetCursorTarget(tr.Start)
	tv.SetFocusEvent()
	tv.NeedsLayout()
	return tv, true
}

// ParseOpenFindURL parses and opens given find:/// url from Find, return text
// region encoded in url, and starting line of results in find buffer, and
// number of results returned -- for parsing all the find results
func (cv *Code) ParseOpenFindURL(ur string, ftv *texteditor.Editor) (tv *TextEditor, reg textbuf.Region, findBufStLn, findCount int, ok bool) {
	up, err := url.Parse(ur)
	if err != nil {
		log.Printf("FindPanel OpenFindURL parse err: %v\n", err)
		return
	}
	fpath := up.Path[1:] // has double //
	pos := up.Fragment
	tv, _, ok = cv.LinkViewFile(core.Filename(fpath))
	if !ok {
		core.MessageSnackbar(cv, fmt.Sprintf("Could not find or open file path in project: %v", fpath))
		return
	}
	if pos == "" {
		return
	}

	lidx := strings.Index(pos, "L")
	if lidx > 0 {
		reg.FromString(pos[lidx:])
		pos = pos[:lidx]
	}
	fmt.Sscanf(pos, "R%dN%d", &findBufStLn, &findCount)
	return
}

// OpenFindURL opens given find:/// url from Find -- delegates to FindPanel
func (cv *Code) OpenFindURL(ur string, ftv *texteditor.Editor) bool {
	fvk := ftv.ParentByType(FindPanelType, tree.NoEmbeds)
	if fvk == nil {
		return false
	}
	fv := fvk.(*FindPanel)
	return fv.OpenFindURL(ur, ftv)
}

// UpdateTextButtons updates texteditor menu buttons is called by SetStatus.
// Doesn't do anything unless a change is required, so safe to call frequently.
func (cv *Code) UpdateTextButtons() {
	ati := cv.ActiveTextEditorIndex
	for i := 0; i < NTextEditors; i++ {
		tv := cv.TextEditorByIndex(i)
		mb := cv.TextEditorButtonByIndex(i)
		txnm := "<no file>"
		if tv.Buffer != nil {
			txnm = fsx.DirAndFile(string(tv.Buffer.Filename))
			if tv.Buffer.NotSaved {
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

func (cv *Code) TextEditorButtonMenu(idx int, m *core.Scene) {
	tv := cv.TextEditorByIndex(idx)
	opn := cv.OpenNodes.Strings()
	core.NewButton(m).SetText("Open File...").OnClick(func(e events.Event) {
		cv.CallViewFile(tv)
	})
	core.NewSeparator(m)
	for i, n := range opn {
		i := i
		n := n
		core.NewButton(m).SetText(n).OnClick(func(e events.Event) {
			cv.ViewFileNode(tv, idx, cv.OpenNodes[i])
		})
	}
}
