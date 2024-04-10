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
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/filetree"
	"cogentcore.org/core/glop/dirs"
	"cogentcore.org/core/states"
	"cogentcore.org/core/texteditor"
	"cogentcore.org/core/texteditor/textbuf"
	"cogentcore.org/core/tree"
)

// ConfigTextBuf configures the text buf according to prefs
func (ge *CodeView) ConfigTextBuf(tb *texteditor.Buffer) {
	tb.SetHiStyle(core.AppearanceSettings.HiStyle)
	tb.Opts.EditorSettings = ge.Settings.Editor
	tb.ConfigKnown()
	if tb.Complete != nil {
		tb.Complete.LookupFunc = ge.LookupFun
	}

	// these are now set in std textbuf..
	// tb.SetSpellCorrect(tb, views.SpellCorrectEdit)                    // always set -- option can override
	// tb.SetCompleter(&tb.PiState, pi.CompletePi, views.CompleteGoEdit) // todo: need pi edit too..
}

// ActiveTextEditor returns the currently-active TextEditor
func (ge *CodeView) ActiveTextEditor() *code.TextEditor {
	//	fmt.Printf("stdout: active text view idx: %v\n", ge.ActiveTextEditorIndex)
	return ge.TextEditorByIndex(ge.ActiveTextEditorIndex)
}

// FocusActiveTextEditor sets focus to active text editor
func (ge *CodeView) FocusActiveTextEditor() *code.TextEditor {
	return ge.SetActiveTextEditorIndex(ge.ActiveTextEditorIndex)
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
	if fn.Buffer == nil {
		return nil, -1, false
	}
	ge.ConfigTextBuf(fn.Buffer)
	for i := 0; i < NTextEditors; i++ {
		tv := ge.TextEditorByIndex(i)
		if tv != nil && tv.Buffer != nil && tv.Buffer == fn.Buffer && ge.PanelIsOpen(i+TextEditor1Index) {
			return tv, i, true
		}
	}
	return nil, -1, false
}

// OpenNodeForTextEditor finds the FileNode that a given TextEditor is
// viewing, returning its index within OpenNodes list, or false if not found
func (ge *CodeView) OpenNodeForTextEditor(tv *code.TextEditor) (*filetree.Node, int, bool) {
	if tv.Buffer == nil {
		return nil, -1, false
	}
	for i, ond := range ge.OpenNodes {
		if ond.Buffer == tv.Buffer {
			return ond, i, true
		}
	}
	return nil, -1, false
}

// TextEditorForFile finds FileNode for file, and returns TextEditor and index
// that is viewing that FileNode, or false if none is
func (ge *CodeView) TextEditorForFile(fnm core.Filename) (*code.TextEditor, int, bool) {
	fn, ok := ge.Files.FindFile(string(fnm))
	if !ok {
		return nil, -1, false
	}
	return ge.TextEditorForFileNode(fn)
}

// SetActiveFileInfo sets the active file info from textbuf
func (ge *CodeView) SetActiveFileInfo(buf *texteditor.Buffer) {
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
	idx := ge.TextEditorIndex(av)
	if idx < 0 {
		fmt.Println("te not found")
		return -1
	}
	ge.ActiveTextEditorIndex = idx
	if av.Buffer != nil {
		ge.SetActiveFileInfo(av.Buffer)
	}
	ge.SetStatus("")
	return idx
}

// SetActiveTextEditorIndex sets the given view index as the currently-active
// TextEditor -- returns that texteditor.  This is the main method for
// activating a text editor.
func (ge *CodeView) SetActiveTextEditorIndex(idx int) *code.TextEditor {
	if idx < 0 || idx >= NTextEditors {
		log.Printf("CodeView SetActiveTextEditorIndex: text view index out of range: %v\n", idx)
		return nil
	}
	ge.ActiveTextEditorIndex = idx
	av := ge.ActiveTextEditor()
	if av.Buffer != nil {
		ge.SetActiveFileInfo(av.Buffer)
		av.Buffer.FileModCheck()
	}
	ge.SetStatus("")
	av.SetFocusEvent()
	return av
}

// NextTextEditor returns the next text view available for viewing a file and
// its index -- if the active text view is empty, then it is used, otherwise
// it is the next one (if visible)
func (ge *CodeView) NextTextEditor() (*code.TextEditor, int) {
	av := ge.TextEditorByIndex(ge.ActiveTextEditorIndex)
	if av.Buffer == nil {
		return av, ge.ActiveTextEditorIndex
	}
	nxt := (ge.ActiveTextEditorIndex + 1) % NTextEditors
	if !ge.PanelIsOpen(nxt + TextEditor1Index) {
		return av, ge.ActiveTextEditorIndex
	}
	return ge.TextEditorByIndex(nxt), nxt
}

// SwapTextEditors switches the buffers for the two open texteditors
// only operates if both panels are open
func (ge *CodeView) SwapTextEditors() bool {
	if !ge.PanelIsOpen(TextEditor1Index) || !ge.PanelIsOpen(TextEditor1Index+1) {
		return false
	}

	tva := ge.TextEditorByIndex(0)
	tvb := ge.TextEditorByIndex(1)
	bufa := tva.Buffer
	bufb := tvb.Buffer
	tva.SetBuffer(bufb)
	tvb.SetBuffer(bufa)
	ge.SetStatus("swapped buffers")
	return true
}

func (ge *CodeView) OpenFileAtRegion(filename core.Filename, tr textbuf.Region) (tv *code.TextEditor, ok bool) {
	tv, _, ok = ge.LinkViewFile(filename)
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
func (ge *CodeView) ParseOpenFindURL(ur string, ftv *texteditor.Editor) (tv *code.TextEditor, reg textbuf.Region, findBufStLn, findCount int, ok bool) {
	up, err := url.Parse(ur)
	if err != nil {
		log.Printf("FindView OpenFindURL parse err: %v\n", err)
		return
	}
	fpath := up.Path[1:] // has double //
	pos := up.Fragment
	tv, _, ok = ge.LinkViewFile(core.Filename(fpath))
	if !ok {
		core.MessageSnackbar(ge, fmt.Sprintf("Could not find or open file path in project: %v", fpath))
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
	fvk := ftv.ParentByType(code.FindViewType, tree.NoEmbeds)
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
	ati := ge.ActiveTextEditorIndex
	for i := 0; i < NTextEditors; i++ {
		tv := ge.TextEditorByIndex(i)
		mb := ge.TextEditorButtonByIndex(i)
		txnm := "<no file>"
		if tv.Buffer != nil {
			txnm = dirs.DirAndFile(string(tv.Buffer.Filename))
			if tv.Buffer.IsNotSaved() {
				txnm += " <b>*</b>"
			} else {
				txnm += "   "
			}
		}
		sel := ati == i
		if mb.Text != txnm || sel != mb.StateIs(states.Selected) {
			mb.SetText(txnm)
			mb.SetSelected(sel)
			mb.Update()
		}
	}
}

// FileNodeSelected is called whenever tree browser has file node selected
func (ge *CodeView) FileNodeSelected(fn *filetree.Node) {
	// not doing anything with this actually
}

func (ge *CodeView) TextEditorButtonMenu(idx int, m *core.Scene) {
	tv := ge.TextEditorByIndex(idx)
	opn := ge.OpenNodes.Strings()
	core.NewButton(m).SetText("Open File...").OnClick(func(e events.Event) {
		ge.CallViewFile(tv)
	})
	core.NewSeparator(m, "file-sep")
	for i, n := range opn {
		i := i
		n := n
		core.NewButton(m).SetText(n).OnClick(func(e events.Event) {
			ge.ViewFileNode(tv, idx, ge.OpenNodes[i])
		})
	}
}
