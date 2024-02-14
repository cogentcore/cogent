// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package codev

import (
	"fmt"
	"log"
	"net/url"
	"strings"

	"cogentcore.org/cogent/code/code"
	"cogentcore.org/core/events"
	"cogentcore.org/core/filetree"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/glop/dirs"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/states"
	"cogentcore.org/core/texteditor"
	"cogentcore.org/core/texteditor/textbuf"
)

// ConfigTextBuf configures the text buf according to prefs
func (ge *CodeView) ConfigTextBuf(tb *texteditor.Buf) {
	tb.SetHiStyle(gi.AppearanceSettings.HiStyle)
	tb.Opts.EditorSettings = ge.Settings.Editor
	tb.ConfigKnown()
	if tb.Complete != nil {
		tb.Complete.LookupFunc = ge.LookupFun
	}

	// these are now set in std textbuf..
	// tb.SetSpellCorrect(tb, giv.SpellCorrectEdit)                    // always set -- option can override
	// tb.SetCompleter(&tb.PiState, pi.CompletePi, giv.CompleteGoEdit) // todo: need pi edit too..
}

// ActiveTextEditor returns the currently-active TextEditor
func (ge *CodeView) ActiveTextEditor() *code.TextEditor {
	//	fmt.Printf("stdout: active text view idx: %v\n", ge.ActiveTextEditorIdx)
	return ge.TextEditorByIndex(ge.ActiveTextEditorIdx)
}

// FocusActiveTextEditor sets focus to active text editor
func (ge *CodeView) FocusActiveTextEditor() *code.TextEditor {
	return ge.SetActiveTextEditorIdx(ge.ActiveTextEditorIdx)
}

// ActiveFileNode returns the file node for the active file -- nil if none
func (ge *CodeView) ActiveFileNode() *filetree.Node {
	return ge.FileNodeForFile(string(ge.ActiveFilename), false)
}

// TextEditorIndex finds index of given texteditor (0 or 1)
func (ge *CodeView) TextEditorIndex(av *code.TextEditor) int {
	for i := 0; i < NTextEditors; i++ {
		tv := ge.TextEditorByIndex(i)
		if tv.This() == av.This() {
			return i
		}
	}
	return -1 // shouldn't happen
}

// TextEditorForFileNode finds a TextEditor that is viewing given FileNode,
// and its index, or false if none is
func (ge *CodeView) TextEditorForFileNode(fn *filetree.Node) (*code.TextEditor, int, bool) {
	if fn.Buf == nil {
		return nil, -1, false
	}
	ge.ConfigTextBuf(fn.Buf)
	for i := 0; i < NTextEditors; i++ {
		tv := ge.TextEditorByIndex(i)
		if tv != nil && tv.Buf != nil && tv.Buf == fn.Buf && ge.PanelIsOpen(i+TextEditor1Idx) {
			return tv, i, true
		}
	}
	return nil, -1, false
}

// OpenNodeForTextEditor finds the FileNode that a given TextEditor is
// viewing, returning its index within OpenNodes list, or false if not found
func (ge *CodeView) OpenNodeForTextEditor(tv *code.TextEditor) (*filetree.Node, int, bool) {
	if tv.Buf == nil {
		return nil, -1, false
	}
	for i, ond := range ge.OpenNodes {
		if ond.Buf == tv.Buf {
			return ond, i, true
		}
	}
	return nil, -1, false
}

// TextEditorForFile finds FileNode for file, and returns TextEditor and index
// that is viewing that FileNode, or false if none is
func (ge *CodeView) TextEditorForFile(fnm gi.Filename) (*code.TextEditor, int, bool) {
	fn, ok := ge.Files.FindFile(string(fnm))
	if !ok {
		return nil, -1, false
	}
	return ge.TextEditorForFileNode(fn)
}

// SetActiveFileInfo sets the active file info from textbuf
func (ge *CodeView) SetActiveFileInfo(buf *texteditor.Buf) {
	ge.ActiveFilename = buf.Filename
	ge.ActiveLang = buf.Info.Known
	ge.ActiveVCSInfo = ""
	ge.ActiveVCS = nil
	fn := ge.FileNodeForFile(string(ge.ActiveFilename), false)
	if fn != nil {
		repo, _ := fn.Repo()
		if repo != nil {
			ge.ActiveVCS = repo
			cur, err := repo.Current()
			if err == nil {
				ge.ActiveVCSInfo = fmt.Sprintf("%s: <i>%s</i>", repo.Vcs(), cur)
			}
		}
		fn.ScrollToMe()
	}
}

// SetActiveTextEditor sets the given texteditor as the active one, and returns its index
func (ge *CodeView) SetActiveTextEditor(av *code.TextEditor) int {
	updt := ge.UpdateStart()
	defer ge.UpdateEndLayout(updt)

	idx := ge.TextEditorIndex(av)
	if idx < 0 {
		fmt.Println("te not found")
		return -1
	}
	ge.ActiveTextEditorIdx = idx
	if av.Buf != nil {
		ge.SetActiveFileInfo(av.Buf)
	}
	ge.SetStatus("")
	return idx
}

// SetActiveTextEditorIdx sets the given view index as the currently-active
// TextEditor -- returns that texteditor.  This is the main method for
// activating a text editor.
func (ge *CodeView) SetActiveTextEditorIdx(idx int) *code.TextEditor {
	updt := ge.UpdateStart()
	defer ge.UpdateEndLayout(updt)

	if idx < 0 || idx >= NTextEditors {
		log.Printf("CodeView SetActiveTextEditorIdx: text view index out of range: %v\n", idx)
		return nil
	}
	ge.ActiveTextEditorIdx = idx
	av := ge.ActiveTextEditor()
	if av.Buf != nil {
		ge.SetActiveFileInfo(av.Buf)
		av.Buf.FileModCheck()
	}
	ge.SetStatus("")
	av.SetFocusEvent()
	av.SetCursorTarget(av.CursorPos)
	av.SetNeedsLayout(true)
	return av
}

// NextTextEditor returns the next text view available for viewing a file and
// its index -- if the active text view is empty, then it is used, otherwise
// it is the next one (if visible)
func (ge *CodeView) NextTextEditor() (*code.TextEditor, int) {
	av := ge.TextEditorByIndex(ge.ActiveTextEditorIdx)
	if av.Buf == nil {
		return av, ge.ActiveTextEditorIdx
	}
	nxt := (ge.ActiveTextEditorIdx + 1) % NTextEditors
	if !ge.PanelIsOpen(nxt + TextEditor1Idx) {
		return av, ge.ActiveTextEditorIdx
	}
	return ge.TextEditorByIndex(nxt), nxt
}

// SwapTextEditors switches the buffers for the two open texteditors
// only operates if both panels are open
func (ge *CodeView) SwapTextEditors() bool {
	if !ge.PanelIsOpen(TextEditor1Idx) || !ge.PanelIsOpen(TextEditor1Idx+1) {
		return false
	}
	updt := ge.UpdateStart()
	defer ge.UpdateEndLayout(updt)

	tva := ge.TextEditorByIndex(0)
	tvb := ge.TextEditorByIndex(1)
	bufa := tva.Buf
	bufb := tvb.Buf
	tva.SetBuf(bufb)
	tvb.SetBuf(bufa)
	ge.SetStatus("swapped buffers")
	return true
}

func (ge *CodeView) OpenFileAtRegion(filename gi.Filename, tr textbuf.Region) (tv *code.TextEditor, ok bool) {
	tv, _, ok = ge.LinkViewFile(filename)
	if tv == nil {
		return nil, false
	}
	updt := tv.UpdateStart()
	defer tv.UpdateEndRender(updt)

	tv.Highlights = tv.Highlights[:0]
	tv.Highlights = append(tv.Highlights, tr)
	tv.SetCursorTarget(tr.Start)
	tv.SetFocusEvent()
	return tv, true
}

// ParseOpenFindURL parses and opens given find:/// url from Find, return text
// region encoded in url, and starting line of results in find buffer, and
// number of results returned -- for parsing all the find results
func (ge *CodeView) ParseOpenFindURL(ur string, ftv *texteditor.Editor) (tv *code.TextEditor, reg textbuf.Region, findBufStLn, findCount int, ok bool) {
	up, err := url.Parse(ur)
	if err != nil {
		log.Printf("FindView OpenFindURL parse err: %v\n", err)
		return
	}
	fpath := up.Path[1:] // has double //
	pos := up.Fragment
	tv, _, ok = ge.LinkViewFile(gi.Filename(fpath))
	if !ok {
		gi.MessageSnackbar(ge, fmt.Sprintf("Could not find or open file path in project: %v", fpath))
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

// OpenFindURL opens given find:/// url from Find -- delegates to FindView
func (ge *CodeView) OpenFindURL(ur string, ftv *texteditor.Editor) bool {
	fvk := ftv.ParentByType(code.FindViewType, ki.NoEmbeds)
	if fvk == nil {
		return false
	}
	fv := fvk.(*code.FindView)
	return fv.OpenFindURL(ur, ftv)
}

// UpdateTextButtons updates texteditor menu buttons
// is called by SetStatus and is generally under cover of TopUpdateStart / End
// doesn't do anything unless a change is required -- safe to call frequently.
func (ge *CodeView) UpdateTextButtons() {
	ati := ge.ActiveTextEditorIdx
	for i := 0; i < NTextEditors; i++ {
		tv := ge.TextEditorByIndex(i)
		mb := ge.TextEditorButtonByIndex(i)
		txnm := "<no file>"
		if tv.Buf != nil {
			txnm = dirs.DirAndFile(string(tv.Buf.Filename))
			if tv.Buf.IsNotSaved() {
				txnm += " <b>*</b>"
			}
		}
		sel := ati == i
		if mb.Text != txnm || sel != mb.StateIs(states.Selected) {
			updt := mb.UpdateStart()
			mb.SetText(txnm)
			mb.SetSelected(sel)
			mb.Update()
			mb.UpdateEndRender(updt)
		}
	}
}

// FileNodeSelected is called whenever tree browser has file node selected
func (ge *CodeView) FileNodeSelected(fn *filetree.Node) {
	// not doing anything with this actually
}

func (ge *CodeView) TextEditorButtonMenu(idx int, m *gi.Scene) {
	tv := ge.TextEditorByIndex(idx)
	opn := ge.OpenNodes.Strings()
	gi.NewButton(m).SetText("Open File...").OnClick(func(e events.Event) {
		ge.CallViewFile(tv)
	})
	gi.NewSeparator(m, "file-sep")
	for i, n := range opn {
		i := i
		n := n
		gi.NewButton(m).SetText(n).OnClick(func(e events.Event) {
			ge.ViewFileNode(tv, idx, ge.OpenNodes[i])
		})
	}
}
