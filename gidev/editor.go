// Copyright (c) 2023, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gidev

import (
	"bytes"
	"fmt"
	"log"
	"net/url"
	"path/filepath"

	"goki.dev/gi/v2/filetree"
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/texteditor"
	"goki.dev/gi/v2/texteditor/textbuf"
	"goki.dev/gide/v2/gide"
	"goki.dev/girl/paint"
	"goki.dev/glop/dirs"
	"goki.dev/pi/v2/complete"
	"goki.dev/pi/v2/lex"
)

// CursorToHistPrev moves cursor to previous position on history list --
// returns true if moved
func (ge *GideView) CursorToHistPrev() bool { //gti:add
	tv := ge.ActiveTextView()
	return tv.CursorToHistPrev()
}

// CursorToHistNext moves cursor to next position on history list --
// returns true if moved
func (ge *GideView) CursorToHistNext() bool { //gti:add
	tv := ge.ActiveTextView()
	return tv.CursorToHistNext()
}

// LookupFun is the completion system Lookup function that makes a custom
// textview dialog that has option to edit resulting file.
func (ge *GideView) LookupFun(data any, text string, posLn, posCh int) (ld complete.Lookup) {
	/*
		sfs := data.(*pi.FileStates)
		if sfs == nil {
			log.Printf("LookupFun: data is nil not FileStates or is nil - can't lookup\n")
			return ld
		}
		lp, err := pi.LangSupport.Props(sfs.Sup)
		if err != nil {
			log.Printf("LookupFun: %v\n", err)
			return ld
		}
		if lp.Lang == nil {
			return ld
		}

		// note: must have this set to ture to allow viewing of AST
		// must set it in pi/parse directly -- so it is changed in the fileparse too
		parse.GuiActive = true // note: this is key for debugging -- runs slower but makes the tree unique

		ld = lp.Lang.Lookup(sfs, text, lex.Pos{posLn, posCh})
		if len(ld.Text) > 0 {
			texteditor.EditorDialog(nil, ld.Text, giv.DlgOpts{Title: "Lookup: " + text, Data: text})
			return ld
		}
		if ld.Filename == "" {
			return ld
		}

		txt, err := textbuf.FileBytes(ld.Filename)
		if err != nil {
			return ld
		}
		if ld.StLine > 0 {
			lns := bytes.Split(txt, []byte("\n"))
			comLn, comSt, comEd := textbuf.SupportedComments(ld.Filename)
			ld.StLine = textbuf.PreCommentStart(lns, ld.StLine, comLn, comSt, comEd, 10) // just go back 10 max
		}

		prmpt := ""
		if ld.EdLine > ld.StLine {
			prmpt = fmt.Sprintf("%v [%d -- %d]", ld.Filename, ld.StLine, ld.EdLine)
		} else {
			prmpt = fmt.Sprintf("%v:%d", ld.Filename, ld.StLine)
		}
		opts := giv.DlgOpts{Title: "Lookup: " + text, Prompt: prmpt}

		dlg, recyc := gi.RecycleStdDialog(prmpt, opts.ToGiOpts(), gi.NoOk, gi.NoCancel)
		if recyc {
			return ld
		}
		frame := dlg.Frame()
		_, prIdx := dlg.PromptWidget(frame)

		tb := &texteditor.Buf{}
		tb.InitName(tb, "text-view-dialog-buf")
		tb.Filename = gi.FileName(ld.Filename)
		tb.Hi.Style = gi.Prefs.HiStyle
		tb.Opts.LineNos = ge.Prefs.Editor.LineNos
		tb.Stat() // update markup

		tlv := frame.InsertNewChild(gi.LayoutType, prIdx+1, "text-lay").(*gi.Layout)
		tlv.SetProp("width", units.NewCh(80))
		tlv.SetProp("height", units.NewEm(40))
		tlv.SetStretchMax()
		tv := giv.NewTextView(tlv, "text-view")
		tv.Viewport = dlg.Embed(gi.KiT_Viewport2D).(*gi.Scene)
		tv.SetReadOnly(true)
		tv.SetProp("font-family", gi.Prefs.MonoFont)
		tv.SetBuf(tb)
		tv.ScrollToCursorPos = lex.Pos{Ln: ld.StLine}
		tv.ScrollToCursorOnRender = true

		tb.SetText(txt) // calls remarkup

		bbox, _ := dlg.ButtonBox(frame)
		if bbox == nil {
			bbox = dlg.AddButtonBox(frame)
		}
		ofb := gi.NewButton(bbox, "open-file")
		ofb.SetText("Open File")
		ofb.SetIcon("file-open")
		ofb.ButtonSig.Connect(dlg.This(), func(recv, send ki.Ki, sig int64, data any) {
			if sig == int64(gi.ButtonClicked) {
				ge.ViewFile(gi.FileName(ld.Filename))
				dlg.Close()
			}
		})
		cpb := gi.NewButton(bbox, "copy-to-clip")
		cpb.SetText("Copy To Clipboard")
		cpb.SetIcon("copy")
		cpb.ButtonSig.Connect(dlg.This(), func(recv, send ki.Ki, sig int64, data any) {
			if sig == int64(gi.ButtonClicked) {
				ddlg := recv.Embed(gi.KiT_Dialog).(*gi.Dialog)
				goosi.TheApp.ClipBoard(ddlg.Win.OSWin).Write(mimedata.NewTextBytes(txt))
			}
		})
		dlg.UpdateEndNoSig(true) // going to be shown
		dlg.Open(0, 0, ge.Viewport, nil)
	*/
	return
}

// ReplaceInActive does query-replace in active file only
func (ge *GideView) ReplaceInActive() { //gti:add
	tv := ge.ActiveTextView()
	tv.QReplaceAddText()
}

//////////////////////////////////////////////////////////////////////////////////////
//    Rects, Registers

// CutRect cuts rectangle in active text view
func (ge *GideView) CutRect() { //gti:add
	tv := ge.ActiveTextView()
	if tv.Buf == nil {
		return
	}
	tv.CutRect()
}

// CopyRect copies rectangle in active text view
func (ge *GideView) CopyRect() { //gti:add
	tv := ge.ActiveTextView()
	if tv.Buf == nil {
		return
	}
	tv.CopyRect(true)
}

// PasteRect cuts rectangle in active text view
func (ge *GideView) PasteRect() { //gti:add
	tv := ge.ActiveTextView()
	if tv.Buf == nil {
		return
	}
	tv.PasteRect()
}

// RegisterCopy saves current selection in active text view to register of given name
// returns true if saved
func (ge *GideView) RegisterCopy(name string) bool { //gti:add
	if name == "" {
		return false
	}
	tv := ge.ActiveTextView()
	if tv.Buf == nil {
		return false
	}
	sel := tv.Selection()
	if sel == nil {
		return false
	}
	if gide.AvailRegisters == nil {
		gide.AvailRegisters = make(gide.Registers, 100)
	}
	gide.AvailRegisters[name] = string(sel.ToBytes())
	gide.AvailRegisters.SavePrefs()
	ge.Prefs.Register = gide.RegisterName(name)
	tv.SelectReset()
	return true
}

// RegisterPaste pastes register of given name into active text view
// returns true if pasted
func (ge *GideView) RegisterPaste(name gide.RegisterName) bool { //gti:add
	if name == "" {
		return false
	}
	str, ok := gide.AvailRegisters[string(name)]
	if !ok {
		return false
	}
	tv := ge.ActiveTextView()
	if tv.Buf == nil {
		return false
	}
	tv.InsertAtCursor([]byte(str))
	ge.Prefs.Register = name
	return true
}

// CommentOut comments-out selected lines in active text view
// and uncomments if already commented
// If multiple lines are selected and any line is uncommented all will be commented
func (ge *GideView) CommentOut() bool { //gti:add
	tv := ge.ActiveTextView()
	if tv.Buf == nil {
		return false
	}
	sel := tv.Selection()
	var stl, etl int
	if sel == nil {
		stl = tv.CursorPos.Ln
		etl = stl + 1
	} else {
		stl = sel.Reg.Start.Ln
		etl = sel.Reg.End.Ln
	}
	tv.Buf.CommentRegion(stl, etl)
	tv.SelectReset()
	return true
}

// Indent indents selected lines in active view
func (ge *GideView) Indent() bool { //gti:add
	tv := ge.ActiveTextView()
	if tv.Buf == nil {
		return false
	}
	sel := tv.Selection()
	if sel == nil {
		return false
	}
	tv.Buf.AutoIndentRegion(sel.Reg.Start.Ln, sel.Reg.End.Ln)
	tv.SelectReset()
	return true
}

// ReCase replaces currently selected text in current active view with given case
func (ge *GideView) ReCase(c textbuf.Cases) string { //gti:add
	tv := ge.ActiveTextView()
	if tv.Buf == nil {
		return ""
	}
	return tv.ReCaseSelection(c)
}

// JoinParaLines merges sequences of lines with hard returns forming paragraphs,
// separated by blank lines, into a single line per paragraph,
// for given selected region (full text if no selection)
func (ge *GideView) JoinParaLines() { //gti:add
	tv := ge.ActiveTextView()
	if tv.Buf == nil {
		return
	}
	if tv.HasSelection() {
		tv.Buf.JoinParaLines(tv.SelectReg.Start.Ln, tv.SelectReg.End.Ln)
	} else {
		tv.Buf.JoinParaLines(0, tv.NLines-1)
	}
}

// TabsToSpaces converts tabs to spaces
// for given selected region (full text if no selection)
func (ge *GideView) TabsToSpaces() { //gti:add
	tv := ge.ActiveTextView()
	if tv.Buf == nil {
		return
	}
	if tv.HasSelection() {
		tv.Buf.TabsToSpacesRegion(tv.SelectReg.Start.Ln, tv.SelectReg.End.Ln)
	} else {
		tv.Buf.TabsToSpacesRegion(0, tv.NLines-1)
	}
}

// SpacesToTabs converts spaces to tabs
// for given selected region (full text if no selection)
func (ge *GideView) SpacesToTabs() { //gti:add
	tv := ge.ActiveTextView()
	if tv.Buf == nil {
		return
	}
	if tv.HasSelection() {
		tv.Buf.SpacesToTabsRegion(tv.SelectReg.Start.Ln, tv.SelectReg.End.Ln)
	} else {
		tv.Buf.SpacesToTabsRegion(0, tv.NLines-1)
	}
}

// DiffFiles shows the differences between two given files
// in side-by-side DiffView and in the console as a context diff.
// It opens the files as file nodes and uses existing contents if open already.
func (ge *GideView) DiffFiles(fnmA, fnmB gi.FileName) { //gti:add
	fna := ge.FileNodeForFile(string(fnmA), true)
	if fna == nil {
		return
	}
	if fna.Buf == nil {
		ge.OpenFileNode(fna)
	}
	if fna.Buf == nil {
		return
	}
	ge.DiffFileNode(fna, fnmB)
}

// DiffFileNode shows the differences between given file node as the A file,
// and another given file as the B file,
// in side-by-side DiffView and in the console as a context diff.
func (ge *GideView) DiffFileNode(fna *filetree.Node, fnmB gi.FileName) { //gti:add
	fnb := ge.FileNodeForFile(string(fnmB), true)
	if fnb == nil {
		return
	}
	if fnb.Buf == nil {
		ge.OpenFileNode(fnb)
	}
	if fnb.Buf == nil {
		return
	}
	dif := fna.Buf.DiffBufsUnified(fnb.Buf, 3)
	cbuf, _, _ := ge.RecycleCmdTab("Diffs", true, true)
	cbuf.SetText(dif)
	cbuf.AutoScrollViews()

	astr := fna.Buf.Strings(false)
	bstr := fnb.Buf.Strings(false)
	_, _ = astr, bstr

	// todo!
	// giv.DiffViewDialog(ge.Viewport, astr, bstr, string(fna.Buf.Filename), string(fnb.Buf.Filename), "", "", giv.DlgOpts{Title: "Diff File View:"})
}

// CountWords counts number of words (and lines) in active file
// returns a string report thereof.
func (ge *GideView) CountWords() string { //gti:add
	av := ge.ActiveTextView()
	if av.Buf == nil || av.Buf.NLines <= 0 {
		return "empty"
	}
	av.Buf.LinesMu.RLock()
	defer av.Buf.LinesMu.RUnlock()
	ll := av.Buf.NLines - 1
	reg := textbuf.NewRegion(0, 0, ll, len(av.Buf.Lines[ll]))
	words, lines := textbuf.CountWordsLinesRegion(av.Buf.Lines, reg)
	return fmt.Sprintf("File: %s  Words: %d   Lines: %d\n", dirs.DirAndFile(string(av.Buf.Filename)), words, lines)
}

// CountWordsRegion counts number of words (and lines) in selected region in file
// if no selection, returns numbers for entire file.
func (ge *GideView) CountWordsRegion() string { //gti:add
	av := ge.ActiveTextView()
	if av.Buf == nil || av.Buf.NLines <= 0 {
		return "empty"
	}
	if !av.HasSelection() {
		return ge.CountWords()
	}
	av.Buf.LinesMu.RLock()
	defer av.Buf.LinesMu.RUnlock()
	sel := av.Selection()
	words, lines := textbuf.CountWordsLinesRegion(av.Buf.Lines, sel.Reg)
	return fmt.Sprintf("File: %s  Words: %d   Lines: %d\n", dirs.DirAndFile(string(av.Buf.Filename)), words, lines)
}

//////////////////////////////////////////////////////////////////////////////////////
//   Links

// TextLinkHandler is the GideView handler for text links -- preferred one b/c
// directly connects to correct GideView project
func TextLinkHandler(tl paint.TextLink) bool {
	// todo:
	// tve := texteditor.AsEditor(tl.Widget)
	// ftv, _ := tl.Widget.Embed(giv.KiT_TextView).(*texteditor.Editor)
	// gek := tl.Widget.ParentByType(KiT_GideView, true)
	// if gek != nil {
	// 	ge := gek.Embed(KiT_GideView).(*GideView)
	// 	ur := tl.URL
	// 	// todo: use net/url package for more systematic parsing
	// 	switch {
	// 	case strings.HasPrefix(ur, "find:///"):
	// 		ge.OpenFindURL(ur, ftv)
	// 	case strings.HasPrefix(ur, "file:///"):
	// 		ge.OpenFileURL(ur, ftv)
	// 	default:
	// 		goosi.TheApp.OpenURL(ur)
	// 	}
	// } else {
	// 	goosi.TheApp.OpenURL(tl.URL)
	// }
	return true
}

// // URLHandler is the GideView handler for urls --
// func URLHandler(url string) bool {
// 	return true
// }

// OpenFileURL opens given file:/// url
func (ge *GideView) OpenFileURL(ur string, ftv *texteditor.Editor) bool {
	up, err := url.Parse(ur)
	if err != nil {
		log.Printf("GideView OpenFileURL parse err: %v\n", err)
		return false
	}
	fpath := up.Path[1:] // has double //
	cdpath := ""
	if ftv != nil && ftv.Buf != nil { // get cd path for non-pathed fnames
		cdln := ftv.Buf.BytesLine(0)
		if bytes.HasPrefix(cdln, []byte("cd ")) {
			fmidx := bytes.Index(cdln, []byte(" (from: "))
			if fmidx > 0 {
				cdpath = string(cdln[3:fmidx])
				dr, _ := filepath.Split(fpath)
				if dr == "" || !filepath.IsAbs(dr) {
					fpath = filepath.Join(cdpath, fpath)
				}
			}
		}
	}
	pos := up.Fragment
	tv, _, ok := ge.LinkViewFile(gi.FileName(fpath))
	if !ok {
		_, fnm := filepath.Split(fpath)
		tv, _, ok = ge.LinkViewFile(gi.FileName(fnm))
		if !ok {
			gi.NewBody().AddTitle("Couldn't Open File at Link").
				AddText(fmt.Sprintf("Could not find or open file path in project: %v", fpath)).
				AddOkOnly().NewDialog(ge).Run()
			return false
		}
	}
	if pos == "" {
		return true
	}
	// fmt.Printf("pos: %v\n", pos)
	txpos := lex.Pos{}
	if txpos.FromString(pos) {
		reg := textbuf.Region{Start: txpos, End: lex.Pos{Ln: txpos.Ln, Ch: txpos.Ch + 4}}
		// todo: need some way of tagging the time stamp for adjusting!
		// reg = tv.Buf.AdjustReg(reg)
		txpos = reg.Start
		tv.HighlightRegion(reg)
		tv.SetCursorShow(txpos)
		tv.SetNeedsLayout()
	}
	return true
}
