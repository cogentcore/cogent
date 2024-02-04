// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package codev

import (
	"bytes"
	"fmt"
	"log"
	"net/url"
	"path/filepath"
	"reflect"

	"cogentcore.org/cogent/code/code"
	"cogentcore.org/core/events"
	"cogentcore.org/core/filetree"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/glop/dirs"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/mimedata"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/pi"
	"cogentcore.org/core/pi/complete"
	"cogentcore.org/core/pi/lex"
	"cogentcore.org/core/pi/parse"
	"cogentcore.org/core/texteditor"
	"cogentcore.org/core/texteditor/textbuf"
)

// CursorToHistPrev moves back to the previous history item.
func (ge *CodeView) CursorToHistPrev() bool { //gti:add
	tv := ge.ActiveTextEditor()
	return tv.CursorToHistPrev()
}

// CursorToHistNext moves forward to the next history item.
func (ge *CodeView) CursorToHistNext() bool { //gti:add
	tv := ge.ActiveTextEditor()
	return tv.CursorToHistNext()
}

// LookupFun is the completion system Lookup function that makes a custom
// texteditor dialog that has option to edit resulting file.
func (ge *CodeView) LookupFun(data any, text string, posLn, posCh int) (ld complete.Lookup) {
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

	// note: must have this set to true to allow viewing of AST
	// must set it in pi/parse directly -- so it is changed in the fileparse too
	parse.GuiActive = true // note: this is key for debugging -- runs slower but makes the tree unique

	ld = lp.Lang.Lookup(sfs, text, lex.Pos{posLn, posCh})
	if len(ld.Text) > 0 {
		tev := texteditor.Value{}
		tev.SetSoloValue(reflect.ValueOf(ld.Text))
		tev.OpenDialog(ge, nil)
		// todo: title: "Lookup: " + text
		return ld
	}
	if ld.Filename == "" {
		return ld
	}

	if gi.RecycleDialog(ld) {
		return
	}

	txt, err := textbuf.FileBytes(ld.Filename)
	if err != nil {
		return ld
	}
	if ld.StLine > 0 {
		lns := bytes.Split(txt, []byte("\n"))
		comLn, comSt, comEd := textbuf.KnownComments(ld.Filename)
		ld.StLine = textbuf.PreCommentStart(lns, ld.StLine, comLn, comSt, comEd, 10) // just go back 10 max
	}

	prmpt := ""
	if ld.EdLine > ld.StLine {
		prmpt = fmt.Sprintf("%v [%d -- %d]", ld.Filename, ld.StLine, ld.EdLine)
	} else {
		prmpt = fmt.Sprintf("%v:%d", ld.Filename, ld.StLine)
	}
	title := "Lookup: " + text

	tb := texteditor.NewBuf().SetText(txt).SetFilename(ld.Filename)
	tb.Hi.Style = gi.AppearanceSettings.HiStyle
	tb.Opts.LineNos = ge.Settings.Editor.LineNos

	d := gi.NewBody().AddTitle(title).AddText(prmpt)
	tv := texteditor.NewEditor(d).SetBuf(tb)
	tv.SetReadOnly(true)

	tv.SetCursorTarget(lex.Pos{Ln: ld.StLine})
	tv.Styles.Font.Family = string(gi.AppearanceSettings.MonoFont)
	d.AddBottomBar(func(pw gi.Widget) {
		gi.NewButton(pw).SetText("Open File").SetIcon(icons.Open).OnClick(func(e events.Event) {
			ge.ViewFile(gi.Filename(ld.Filename))
			d.Close()
		})
		gi.NewButton(pw).SetText("Copy To Clipboard").SetIcon("copy").
			OnClick(func(e events.Event) {
				d.Clipboard().Write(mimedata.NewTextBytes(txt))
			})
	})
	d.NewFullDialog(ge.ActiveTextEditor()).SetNewWindow(true).Run()
	tb.StartDelayedReMarkup() // update markup
	return
}

// ReplaceInActive does query-replace in active file only
func (ge *CodeView) ReplaceInActive() { //gti:add
	tv := ge.ActiveTextEditor()
	tv.QReplaceAddText()
}

//////////////////////////////////////////////////////////////////////////////////////
//    Rects, Registers

// CutRect cuts rectangle in active text view
func (ge *CodeView) CutRect() { //gti:add
	tv := ge.ActiveTextEditor()
	if tv.Buf == nil {
		return
	}
	tv.CutRect()
}

// CopyRect copies rectangle in active text view
func (ge *CodeView) CopyRect() { //gti:add
	tv := ge.ActiveTextEditor()
	if tv.Buf == nil {
		return
	}
	tv.CopyRect(true)
}

// PasteRect cuts rectangle in active text view
func (ge *CodeView) PasteRect() { //gti:add
	tv := ge.ActiveTextEditor()
	if tv.Buf == nil {
		return
	}
	tv.PasteRect()
}

// RegisterCopy saves current selection in active text view to register of given name
// returns true if saved
func (ge *CodeView) RegisterCopy(name string) bool { //gti:add
	if name == "" {
		return false
	}
	tv := ge.ActiveTextEditor()
	if tv.Buf == nil {
		return false
	}
	sel := tv.Selection()
	if sel == nil {
		return false
	}
	if code.AvailRegisters == nil {
		code.AvailRegisters = make(code.Registers, 100)
	}
	code.AvailRegisters[name] = string(sel.ToBytes())
	code.AvailRegisters.SavePrefs()
	ge.Settings.Register = code.RegisterName(name)
	tv.SelectReset()
	return true
}

// RegisterPaste pastes register of given name into active text view
// returns true if pasted
func (ge *CodeView) RegisterPaste(name code.RegisterName) bool { //gti:add
	if name == "" {
		return false
	}
	str, ok := code.AvailRegisters[string(name)]
	if !ok {
		return false
	}
	tv := ge.ActiveTextEditor()
	if tv.Buf == nil {
		return false
	}
	tv.InsertAtCursor([]byte(str))
	ge.Settings.Register = name
	return true
}

// CommentOut comments-out selected lines in active text view
// and uncomments if already commented
// If multiple lines are selected and any line is uncommented all will be commented
func (ge *CodeView) CommentOut() bool { //gti:add
	tv := ge.ActiveTextEditor()
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
func (ge *CodeView) Indent() bool { //gti:add
	tv := ge.ActiveTextEditor()
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
func (ge *CodeView) ReCase(c textbuf.Cases) string { //gti:add
	tv := ge.ActiveTextEditor()
	if tv.Buf == nil {
		return ""
	}
	return tv.ReCaseSelection(c)
}

// JoinParaLines merges sequences of lines with hard returns forming paragraphs,
// separated by blank lines, into a single line per paragraph,
// for given selected region (full text if no selection)
func (ge *CodeView) JoinParaLines() { //gti:add
	tv := ge.ActiveTextEditor()
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
func (ge *CodeView) TabsToSpaces() { //gti:add
	tv := ge.ActiveTextEditor()
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
func (ge *CodeView) SpacesToTabs() { //gti:add
	tv := ge.ActiveTextEditor()
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
func (ge *CodeView) DiffFiles(fnmA, fnmB gi.Filename) { //gti:add
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
func (ge *CodeView) DiffFileNode(fna *filetree.Node, fnmB gi.Filename) { //gti:add
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

	texteditor.DiffViewDialog(ge, "Diff File View:", astr, bstr, string(fna.Buf.Filename), string(fnb.Buf.Filename), "", "")
}

// CountWords counts number of words (and lines) in active file
// returns a string report thereof.
func (ge *CodeView) CountWords() string { //gti:add
	av := ge.ActiveTextEditor()
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
func (ge *CodeView) CountWordsRegion() string { //gti:add
	av := ge.ActiveTextEditor()
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

// TextLinkHandler is the CodeView handler for text links -- preferred one b/c
// directly connects to correct CodeView project
func TextLinkHandler(tl paint.TextLink) bool {
	// todo:
	// tve := texteditor.AsEditor(tl.Widget)
	// ftv, _ := tl.Widget.Embed(giv.KiT_TextEditor).(*texteditor.Editor)
	// gek := tl.Widget.ParentByType(KiT_CodeView, true)
	// if gek != nil {
	// 	ge := gek.Embed(KiT_CodeView).(*CodeView)
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

// // URLHandler is the CodeView handler for urls --
// func URLHandler(url string) bool {
// 	return true
// }

// OpenFileURL opens given file:/// url
func (ge *CodeView) OpenFileURL(ur string, ftv *texteditor.Editor) bool {
	up, err := url.Parse(ur)
	if err != nil {
		log.Printf("CodeView OpenFileURL parse err: %v\n", err)
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
	tv, _, ok := ge.LinkViewFile(gi.Filename(fpath))
	if !ok {
		_, fnm := filepath.Split(fpath)
		tv, _, ok = ge.LinkViewFile(gi.Filename(fnm))
		if !ok {
			gi.MessageSnackbar(ge, fmt.Sprintf("Could not find or open file path in project: %v", fpath))
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
		tv.SetCursorTarget(txpos)
		tv.SetNeedsLayout(true)
	}
	return true
}
