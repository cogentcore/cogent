// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"bytes"
	"fmt"
	"log"
	"net/url"
	"path/filepath"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/base/strcase"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/parse"
	"cogentcore.org/core/text/parse/complete"
	"cogentcore.org/core/text/parse/parser"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/textcore"
	"cogentcore.org/core/text/textpos"
)

// CursorToHistPrev moves back to the previous history item.
func (cv *Code) CursorToHistPrev() bool { //types:add
	tv := cv.ActiveEditor()
	return tv.CursorToHistoryPrev()
}

// CursorToHistNext moves forward to the next history item.
func (cv *Code) CursorToHistNext() bool { //types:add
	tv := cv.ActiveEditor()
	return tv.CursorToHistoryNext()
}

// LookupFun is the completion system Lookup function that makes a custom
// texteditor dialog that has option to edit resulting file.
func (cv *Code) LookupFun(data any, txt string, posLine, posChar int) (ld complete.Lookup) {
	sfs := data.(*parse.FileStates)
	if sfs == nil {
		log.Printf("LookupFun: data is nil not FileStates or is nil - can't lookup\n")
		return ld
	}
	lp, err := parse.LanguageSupport.Properties(sfs.Known)
	if err != nil {
		log.Printf("LookupFun: %v\n", err)
		return ld
	}
	if lp.Lang == nil {
		return ld
	}

	// note: must have this set to true to allow viewing of AST
	// must set it in pi/parse directly -- so it is changed in the fileparse too
	parser.GUIActive = true // note: this is key for debugging -- runs slower but makes the tree unique

	ld = lp.Lang.Lookup(sfs, txt, textpos.Pos{posLine, posChar})
	if len(ld.Text) > 0 {
		textcore.TextDialog(nil, "Lookup: "+txt, string(ld.Text))
		return ld
	}
	if ld.Filename == "" {
		return ld
	}

	if core.RecycleDialog(&ld) {
		return
	}

	tx, err := lines.FileBytes(ld.Filename)
	if err != nil {
		return ld
	}
	if ld.StLine > 0 {
		lns := bytes.Split(tx, []byte("\n"))
		comLn, comSt, comEd := lines.KnownComments(ld.Filename)
		ld.StLine = lines.PreCommentStart(lns, ld.StLine, comLn, comSt, comEd, 10) // just go back 10 max
	}

	prmpt := ""
	if ld.EdLine > ld.StLine {
		prmpt = fmt.Sprintf("%v [%d -- %d]", ld.Filename, ld.StLine, ld.EdLine)
	} else {
		prmpt = fmt.Sprintf("%v:%d", ld.Filename, ld.StLine)
	}
	title := "Lookup: " + txt

	tb := lines.NewLines().SetText(tx).SetFilename(ld.Filename)
	tb.SetHighlighting(core.AppearanceSettings.Highlighting)
	tb.Settings.LineNumbers = cv.Settings.Editor.LineNumbers

	d := core.NewBody(title).SetData(&ld)
	core.NewText(d).SetType(core.TextSupporting).SetText(prmpt)
	tv := textcore.NewEditor(d).SetLines(tb)
	tv.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
	})
	tv.SetReadOnly(true)

	tv.SetCursorTarget(textpos.Pos{Line: ld.StLine})
	d.AddBottomBar(func(bar *core.Frame) {
		core.NewButton(bar).SetText("Open file").SetIcon(icons.Open).OnClick(func(e events.Event) {
			cv.ViewFile(core.Filename(ld.Filename))
			d.Close()
		})
		core.NewButton(bar).SetText("Copy to clipboard").SetIcon(icons.Copy).
			OnClick(func(e events.Event) {
				d.Clipboard().Write(mimedata.NewTextBytes(tx))
			})
	})
	d.RunWindowDialog(cv.ActiveEditor())
	tb.StartDelayedReMarkup() // update markup
	return
}

// ReplaceInActive does query-replace in active file only
func (cv *Code) ReplaceInActive() { //types:add
	tv := cv.ActiveEditor()
	tv.QReplacePrompt()
}

//////////////////////////////////////////////////////////////////////////////////////
//    Rects, Registers

// CutRect cuts rectangle in active text view
func (cv *Code) CutRect() { //types:add
	tv := cv.ActiveEditor()
	if tv.Lines == nil {
		return
	}
	tv.CutRect()
}

// CopyRect copies rectangle in active text view
func (cv *Code) CopyRect() { //types:add
	tv := cv.ActiveEditor()
	if tv.Lines == nil {
		return
	}
	tv.CopyRect(true)
}

// PasteRect cuts rectangle in active text view
func (cv *Code) PasteRect() { //types:add
	tv := cv.ActiveEditor()
	if tv.Lines == nil {
		return
	}
	tv.PasteRect()
}

// RegisterCopy saves current selection in active text view
// to register of given name returns true if saved.
func (cv *Code) RegisterCopy(regNm RegisterName) { //types:add
	ic := strings.Index(string(regNm), ":")
	if ic > 0 && ic < 4 {
		regNm = regNm[:ic]
	}
	tv := cv.ActiveEditor()
	if tv.Lines == nil {
		return
	}
	sel := tv.Selection()
	if sel == nil {
		return
	}
	if AvailableRegisters == nil {
		AvailableRegisters = make(Registers)
	}
	AvailableRegisters[string(regNm)] = string(sel.ToBytes())
	AvailableRegisters.SaveSettings()
	cv.Settings.Register = RegisterName(regNm)
	tv.SelectReset()
}

// RegisterPaste prompts user for available registers,
// and pastes selected one into active text view
func (cv *Code) RegisterPaste(ctx core.Widget) { //types:add
	RegistersMenu(ctx, string(cv.Settings.Register), func(regNm string) {
		str, ok := AvailableRegisters[regNm]
		if !ok {
			return
		}
		tv := cv.ActiveEditor()
		if tv.Lines == nil {
			return
		}
		tv.InsertAtCursor([]byte(str))
		cv.Settings.Register = RegisterName(regNm)
	})
}

// CommentOut comments-out selected lines in active text view
// and uncomments if already commented
// If multiple lines are selected and any line is uncommented all will be commented
func (cv *Code) CommentOut() bool { //types:add
	tv := cv.ActiveEditor()
	if tv.Lines == nil {
		return false
	}
	sel := tv.Selection()
	var stl, etl int
	if sel == nil {
		stl = tv.CursorPos.Line
		etl = stl + 1
	} else {
		stl = sel.Region.Start.Line
		etl = sel.Region.End.Line
	}
	tv.Lines.CommentRegion(stl, etl)
	tv.SelectReset()
	return true
}

// Indent indents selected lines in active view
func (cv *Code) Indent() bool { //types:add
	tv := cv.ActiveEditor()
	if tv.Lines == nil {
		return false
	}
	sel := tv.Selection()
	if sel == nil {
		return false
	}
	tv.Lines.AutoIndentRegion(sel.Region.Start.Line, sel.Region.End.Line)
	tv.SelectReset()
	return true
}

// ReCase replaces currently selected text in current active view with given case
func (cv *Code) ReCase(c strcase.Cases) string { //types:add
	tv := cv.ActiveEditor()
	if tv.Lines == nil {
		return ""
	}
	return tv.ReCaseSelection(c)
}

// JoinParaLines merges sequences of lines with hard returns forming paragraphs,
// separated by blank lines, into a single line per paragraph,
// for given selected region (full text if no selection)
func (cv *Code) JoinParaLines() { //types:add
	tv := cv.ActiveEditor()
	if tv.Lines == nil {
		return
	}
	if tv.HasSelection() {
		tv.Lines.JoinParaLines(tv.SelectRegion.Start.Line, tv.SelectRegion.End.Line)
	} else {
		tv.Lines.JoinParaLines(0, tv.NumLines()-1)
	}
}

// TabsToSpaces converts tabs to spaces
// for given selected region (full text if no selection)
func (cv *Code) TabsToSpaces() { //types:add
	tv := cv.ActiveEditor()
	if tv.Lines == nil {
		return
	}
	if tv.HasSelection() {
		tv.Lines.TabsToSpaces(tv.SelectRegion.Start.Line, tv.SelectRegion.End.Line)
	} else {
		tv.Lines.TabsToSpaces(0, tv.NumLines()-1)
	}
}

// SpacesToTabs converts spaces to tabs
// for given selected region (full text if no selection)
func (cv *Code) SpacesToTabs() { //types:add
	tv := cv.ActiveEditor()
	if tv.Lines == nil {
		return
	}
	if tv.HasSelection() {
		tv.Lines.SpacesToTabs(tv.SelectRegion.Start.Line, tv.SelectRegion.End.Line)
	} else {
		tv.Lines.SpacesToTabs(0, tv.NumLines()-1)
	}
}

// DiffFiles shows the differences between two given files
// in side-by-side DiffEditor and in the console as a context diff.
// It opens the files as file nodes and uses existing contents if open already.
func (cv *Code) DiffFiles(fnmA, fnmB core.Filename) { //types:add
	lna, _ := cv.RecycleFile(string(fnmA))
	if lna == nil {
		return
	}
	cv.DiffFileLines(lna, string(fnmB))
}

// DiffFileLines shows the differences between given file node as the A file,
// and another given file as the B file,
// in side-by-side DiffEditor and in the console as a context diff.
func (cv *Code) DiffFileLines(lna *lines.Lines, fnmB string) { //types:add
	lnb := lines.NewLines()
	err := lnb.Open(fnmB)
	if errors.Log(err) != nil {
		return
	}
	dif := lna.DiffsUnified(lnb, 3)
	cbuf, _, _ := cv.RecycleCmdTab("Diffs")
	cbuf.SetText(dif)

	astr := lna.Strings(false)
	bstr := lnb.Strings(false)
	textcore.DiffEditorDialog(cv, "Diff File View:", astr, bstr, lna.Filename(), lnb.Filename(), "", "")
}

// CountWords counts number of words (and lines) in active file
// returns a string report thereof.
func (cv *Code) CountWords() string { //types:add
	av := cv.ActiveEditor()
	if av.Lines == nil || av.Lines.NumLines() <= 0 {
		return "empty"
	}
	ll := av.Lines.NumLines() - 1
	reg := textpos.NewRegion(0, 0, ll, av.Lines.NumLines())
	words, lines := av.Lines.CountWordsLinesRegion(reg)
	return fmt.Sprintf("File: %s  Words: %d   Lines: %d\n", fsx.DirAndFile(av.Lines.Filename()), words, lines)
}

// CountWordsRegion counts number of words (and lines) in selected region in file
// if no selection, returns numbers for entire file.
func (cv *Code) CountWordsRegion() string { //types:add
	av := cv.ActiveEditor()
	if av.Lines == nil || av.Lines.NumLines() <= 0 {
		return "empty"
	}
	if !av.HasSelection() {
		return cv.CountWords()
	}
	sel := av.Selection()
	words, lines := av.Lines.CountWordsLinesRegion(sel.Region)
	return fmt.Sprintf("File: %s  Words: %d   Lines: %d\n", fsx.DirAndFile(av.Lines.Filename()), words, lines)
}

////////   Links

// TextLinkHandler is the Code handler for text links -- preferred one b/c
// directly connects to correct Code project
func TextLinkHandler(tl rich.Hyperlink) bool {
	// todo:
	// tve := textcore.AsEditor(tl.Widget)
	// ftv, _ := tl.Widget.Embed(core.KiT_TextEditor).(*textcore.Editor)
	// gek := tl.Widget.ParentByType(KiT_Code, true)
	// if gek != nil {
	// 	ge := gek.Embed(KiT_Code).(*Code)
	// 	ur := tl.URL
	// 	// todo: use net/url package for more systematic parsing
	// 	switch {
	// 	case strings.HasPrefix(ur, "find:///"):
	// 		ge.OpenFindURL(ur, ftv)
	// 	case strings.HasPrefix(ur, "file:///"):
	// 		ge.OpenFileURL(ur, ftv)
	// 	default:
	// 		system.TheApp.OpenURL(ur)
	// 	}
	// } else {
	// 	system.TheApp.OpenURL(tl.URL)
	// }
	return true
}

// // URLHandler is the Code handler for urls --
// func URLHandler(url string) bool {
// 	return true
// }

// OpenFileURL opens given file:/// url
func (cv *Code) OpenFileURL(ur string, ftv *textcore.Editor) bool {
	up, err := url.Parse(ur)
	if err != nil {
		log.Printf("Code OpenFileURL parse err: %v\n", err)
		return false
	}
	fpath := up.Path[1:] // has double //
	cdpath := ""
	if ftv != nil && ftv.Lines != nil { // get cd path for non-pathed fnames
		cdln := ftv.Lines.String()
		if strings.HasPrefix(cdln, "cd ") {
			fmidx := strings.Index(cdln, " (from: ")
			if fmidx > 0 {
				cdpath = cdln[3:fmidx]
				dr, _ := filepath.Split(fpath)
				if dr == "" || !filepath.IsAbs(dr) {
					fpath = filepath.Join(cdpath, fpath)
				}
			}
		}
	}
	pos := up.Fragment

	tv, _, ok := cv.LinkViewFile(fpath)
	if !ok {
		_, fnm := filepath.Split(fpath)
		tv, _, ok = cv.LinkViewFile(fnm)
		if !ok {
			core.MessageSnackbar(cv, fmt.Sprintf("Could not find or open file path in project: %v", fpath))
			return false
		}
	}
	if pos == "" {
		return true
	}
	// fmt.Printf("pos: %v\n", pos)
	txpos := textpos.Pos{}
	if txpos.FromString(pos) {
		reg := textpos.Region{Start: txpos, End: textpos.Pos{Line: txpos.Line, Char: txpos.Char + 4}}
		// todo: need some way of tagging the time stamp for adjusting!
		// reg = tv.Buf.AdjustReg(reg)
		txpos = reg.Start
		tv.HighlightsReset()
		tv.HighlightRegion(reg)
		tv.SetCursorTarget(txpos)
		tv.NeedsLayout()
	}
	return true
}
