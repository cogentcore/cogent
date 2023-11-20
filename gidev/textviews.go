// Copyright (c) 2023, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gidev

import (
	"fmt"
	"log"
	"net/url"
	"strings"

	"goki.dev/gi/v2/filetree"
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/giv"
	"goki.dev/gi/v2/texteditor"
	"goki.dev/gi/v2/texteditor/textbuf"
	"goki.dev/gide/v2/gide"
	"goki.dev/girl/states"
	"goki.dev/glop/dirs"
	"goki.dev/goosi/events"
	"goki.dev/ki/v2"
)

// ConfigTextBuf configures the text buf according to prefs
func (ge *GideView) ConfigTextBuf(tb *texteditor.Buf) {
	tb.SetHiStyle(gi.Prefs.HiStyle)
	tb.Opts.EditorPrefs = ge.Prefs.Editor
	tb.ConfigSupported()
	if tb.Complete != nil {
		tb.Complete.LookupFunc = ge.LookupFun
	}

	// these are now set in std textbuf..
	// tb.SetSpellCorrect(tb, giv.SpellCorrectEdit)                    // always set -- option can override
	// tb.SetCompleter(&tb.PiState, pi.CompletePi, giv.CompleteGoEdit) // todo: need pi edit too..
}

// ActiveTextView returns the currently-active TextView
func (ge *GideView) ActiveTextView() *gide.TextView {
	//	fmt.Printf("stdout: active text view idx: %v\n", ge.ActiveTextViewIdx)
	return ge.TextViewByIndex(ge.ActiveTextViewIdx)
}

// ActiveFileNode returns the file node for the active file -- nil if none
func (ge *GideView) ActiveFileNode() *filetree.Node {
	return ge.FileNodeForFile(string(ge.ActiveFilename), false)
}

// TextViewIndex finds index of given textview (0 or 1)
func (ge *GideView) TextViewIndex(av *gide.TextView) int {
	for i := 0; i < NTextViews; i++ {
		tv := ge.TextViewByIndex(i)
		if tv.This() == av.This() {
			return i
		}
	}
	return -1 // shouldn't happen
}

// TextViewForFileNode finds a TextView that is viewing given FileNode,
// and its index, or false if none is
func (ge *GideView) TextViewForFileNode(fn *filetree.Node) (*gide.TextView, int, bool) {
	if fn.Buf == nil {
		return nil, -1, false
	}
	ge.ConfigTextBuf(fn.Buf)
	for i := 0; i < NTextViews; i++ {
		tv := ge.TextViewByIndex(i)
		if tv != nil && tv.Buf != nil && tv.Buf == fn.Buf && ge.PanelIsOpen(i+TextView1Idx) {
			return tv, i, true
		}
	}
	return nil, -1, false
}

// OpenNodeForTextView finds the FileNode that a given TextView is
// viewing, returning its index within OpenNodes list, or false if not found
func (ge *GideView) OpenNodeForTextView(tv *gide.TextView) (*filetree.Node, int, bool) {
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

// TextViewForFile finds FileNode for file, and returns TextView and index
// that is viewing that FileNode, or false if none is
func (ge *GideView) TextViewForFile(fnm gi.FileName) (*gide.TextView, int, bool) {
	fn, ok := ge.Files.FindFile(string(fnm))
	if !ok {
		return nil, -1, false
	}
	return ge.TextViewForFileNode(fn)
}

// SetActiveFileInfo sets the active file info from textbuf
func (ge *GideView) SetActiveFileInfo(buf *texteditor.Buf) {
	ge.ActiveFilename = buf.Filename
	ge.ActiveLang = buf.Info.Sup
	fn := ge.FileNodeForFile(string(ge.ActiveFilename), false)
	ge.ActiveVCSInfo = ""
	ge.ActiveVCS = nil
	if fn != nil {
		repo, _ := fn.Repo()
		if repo != nil {
			ge.ActiveVCS = repo
			cur, err := repo.Current()
			if err == nil {
				ge.ActiveVCSInfo = fmt.Sprintf("%s: <i>%s</i>", repo.Vcs(), cur)
			}
		}
	}
}

// SetActiveTextView sets the given textview as the active one, and returns its index
func (ge *GideView) SetActiveTextView(av *gide.TextView) int {
	wupdt := ge.UpdateStart()
	defer ge.UpdateEnd(wupdt)

	idx := ge.TextViewIndex(av)
	if idx < 0 {
		return -1
	}
	if ge.ActiveTextViewIdx == idx {
		return idx
	}
	ge.ActiveTextViewIdx = idx
	if av.Buf != nil {
		ge.SetActiveFileInfo(av.Buf)
	}
	ge.SetStatus("")
	return idx
}

// SetActiveTextViewIdx sets the given view index as the currently-active
// TextView -- returns that textview
func (ge *GideView) SetActiveTextViewIdx(idx int) *gide.TextView {
	wupdt := ge.UpdateStart()
	defer ge.UpdateEnd(wupdt)

	if idx < 0 || idx >= NTextViews {
		log.Printf("GideView SetActiveTextViewIdx: text view index out of range: %v\n", idx)
		return nil
	}
	ge.ActiveTextViewIdx = idx
	av := ge.ActiveTextView()
	if av.Buf != nil {
		ge.SetActiveFileInfo(av.Buf)
		av.Buf.FileModCheck()
	}
	ge.SetStatus("")
	av.GrabFocus()
	return av
}

// NextTextView returns the next text view available for viewing a file and
// its index -- if the active text view is empty, then it is used, otherwise
// it is the next one (if visible)
func (ge *GideView) NextTextView() (*gide.TextView, int) {
	av := ge.TextViewByIndex(ge.ActiveTextViewIdx)
	if av.Buf == nil {
		return av, ge.ActiveTextViewIdx
	}
	nxt := (ge.ActiveTextViewIdx + 1) % NTextViews
	if !ge.PanelIsOpen(nxt + TextView1Idx) {
		return av, ge.ActiveTextViewIdx
	}
	return ge.TextViewByIndex(nxt), nxt
}

// SwapTextViews switches the buffers for the two open textviews
// only operates if both panels are open
func (ge *GideView) SwapTextViews() bool {
	if !ge.PanelIsOpen(TextView1Idx) || !ge.PanelIsOpen(TextView1Idx+1) {
		return false
	}
	wupdt := ge.UpdateStart()
	defer ge.UpdateEnd(wupdt)

	tva := ge.TextViewByIndex(0)
	tvb := ge.TextViewByIndex(1)
	bufa := tva.Buf
	bufb := tvb.Buf
	tva.SetBuf(bufb)
	tvb.SetBuf(bufa)
	ge.SetStatus("swapped buffers")
	return true
}

func (ge *GideView) OpenFileAtRegion(filename gi.FileName, tr textbuf.Region) (tv *gide.TextView, ok bool) {
	tv, _, ok = ge.LinkViewFile(filename)
	if tv != nil {
		tv.UpdateStart()
		tv.Highlights = tv.Highlights[:0]
		tv.Highlights = append(tv.Highlights, tr)
		tv.UpdateEndRender(true)
		tv.SetCursorShow(tr.Start)
		tv.GrabFocus()
		return tv, true

	}
	return nil, false
}

// ParseOpenFindURL parses and opens given find:/// url from Find, return text
// region encoded in url, and starting line of results in find buffer, and
// number of results returned -- for parsing all the find results
func (ge *GideView) ParseOpenFindURL(ur string, ftv *texteditor.Editor) (tv *gide.TextView, reg textbuf.Region, findBufStLn, findCount int, ok bool) {
	up, err := url.Parse(ur)
	if err != nil {
		log.Printf("FindView OpenFindURL parse err: %v\n", err)
		return
	}
	fpath := up.Path[1:] // has double //
	pos := up.Fragment
	tv, _, ok = ge.LinkViewFile(gi.FileName(fpath))
	if !ok {
		gi.NewBody(ge).AddTitle("Could not open file at link").
			AddText(fmt.Sprintf("Could not find or open file path in project: %v", fpath)).Modal(true).Ok().Run()
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
func (ge *GideView) OpenFindURL(ur string, ftv *texteditor.Editor) bool {
	fvk := ftv.ParentByType(gide.FindViewType, ki.Embeds)
	if fvk == nil {
		return false
	}
	fv := fvk.(*gide.FindView)
	return fv.OpenFindURL(ur, ftv)
}

// UpdateTextButtons updates textview menu buttons
// is called by SetStatus and is generally under cover of TopUpdateStart / End
// doesn't do anything unless a change is required -- safe to call frequently.
func (ge *GideView) UpdateTextButtons() {
	ati := ge.ActiveTextViewIdx
	for i := 0; i < NTextViews; i++ {
		tv := ge.TextViewByIndex(i)
		mb := ge.TextViewButtonByIndex(i)
		txnm := "<no file>"
		if tv.Buf != nil {
			txnm = dirs.DirAndFile(string(tv.Buf.Filename))
			if tv.Buf.IsChanged() {
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

// todo:
// TextViewSig handles all signals from the textviews
// func (ge *GideView) TextViewSig(tv *gide.TextView, sig texteditor.EditorSignals) {
// 	ge.SetActiveTextView(tv) // if we're sending signals, we're the active one!
// 	switch sig {
// 	case texteditor.EditorCursorMoved:
// 		ge.SetStatus("") // this really doesn't make any noticeable diff in perf
// 	case texteditor.EditorISearch, texteditor.EditorQReplace:
// 		ge.SetStatus("")
// 	}
// }

// FileNodeSelected is called whenever tree browser has file node selected
func (ge *GideView) FileNodeSelected(fn *filetree.Node) {
	// not doing anything with this actually
}

func (ge *GideView) TextViewButtonMenu(idx int, m *gi.Scene) {
	tv := ge.TextViewByIndex(idx)
	opn := ge.OpenNodes.Strings()
	gi.NewButton(m).SetText("Open File...").OnClick(func(e events.Event) {
		giv.CallFunc(ge, ge.ViewFile)
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
