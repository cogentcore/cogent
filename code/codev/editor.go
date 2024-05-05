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

	"cogentcore.org/cogent/code/code"
	"cogentcore.org/core/base/dirs"
	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/base/strcase"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/filetree"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/parse"
	"cogentcore.org/core/parse/complete"
	"cogentcore.org/core/parse/lexer"
	"cogentcore.org/core/parse/parser"
	"cogentcore.org/core/texteditor"
	"cogentcore.org/core/texteditor/textbuf"
)

// CursorToHistPrev moves back to the previous history item.
func (ge *CodeView) CursorToHistPrev() bool { //types:add
	tv := ge.ActiveTextEditor()
	return tv.CursorToHistPrev()
}

// CursorToHistNext moves forward to the next history item.
func (ge *CodeView) CursorToHistNext() bool { //types:add
	tv := ge.ActiveTextEditor()
	return tv.CursorToHistNext()
}

// LookupFun is the completion system Lookup function that makes a custom
// texteditor dialog that has option to edit resulting file.
func (ge *CodeView) LookupFun(data any, text string, posLn, posCh int) (ld complete.Lookup) {
	sfs := data.(*parse.FileStates)
	if sfs == nil {
		log.Printf("LookupFun: data is nil not FileStates or is nil - can't lookup\n")
		return ld
	}
	lp, err := parse.LangSupport.Properties(sfs.Sup)
	if err != nil {
		log.Printf("LookupFun: %v\n", err)
		return ld
	}
	if lp.Lang == nil {
		return ld
	}

	// note: must have this set to true to allow viewing of AST
	// must set it in pi/parse directly -- so it is changed in the fileparse too
	parser.GuiActive = true // note: this is key for debugging -- runs slower but makes the tree unique

	ld = lp.Lang.Lookup(sfs, text, lexer.Pos{posLn, posCh})
	if len(ld.Text) > 0 {
		texteditor.TextDialog(nil, "Lookup: "+text, string(ld.Text))
		return ld
	}
	if ld.Filename == "" {
		return ld
	}

	if core.RecycleDialog(&ld) {
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

	tb := texteditor.NewBuffer().SetText(txt).SetFilename(ld.Filename)
	tb.Hi.Style = core.AppearanceSettings.HiStyle
	tb.Options.LineNumbers = ge.Settings.Editor.LineNumbers

	d := core.NewBody().AddTitle(title).AddText(prmpt).SetData(&ld)
	tv := texteditor.NewEditor(d).SetBuffer(tb)
	tv.SetReadOnly(true)

	tv.SetCursorTarget(lexer.Pos{Ln: ld.StLine})
	tv.Styles.Font.Family = string(core.AppearanceSettings.MonoFont)
	d.AddBottomBar(func(parent core.Widget) {
		core.NewButton(parent).SetText("Open file").SetIcon(icons.Open).OnClick(func(e events.Event) {
			ge.ViewFile(core.Filename(ld.Filename))
			d.Close()
		})
		core.NewButton(parent).SetText("Copy to clipboard").SetIcon(icons.Copy).
			OnClick(func(e events.Event) {
				d.Clipboard().Write(mimedata.NewTextBytes(txt))
			})
	})
	d.RunWindowDialog(ge.ActiveTextEditor())
	tb.StartDelayedReMarkup() // update markup
	return
}

// ReplaceInActive does query-replace in active file only
func (ge *CodeView) ReplaceInActive() { //types:add
	tv := ge.ActiveTextEditor()
	tv.QReplacePrompt()
}

//////////////////////////////////////////////////////////////////////////////////////
//    Rects, Registers

// CutRect cuts rectangle in active text view
func (ge *CodeView) CutRect() { //types:add
	tv := ge.ActiveTextEditor()
	if tv.Buffer == nil {
		return
	}
	tv.CutRect()
}

// CopyRect copies rectangle in active text view
func (ge *CodeView) CopyRect() { //types:add
	tv := ge.ActiveTextEditor()
	if tv.Buffer == nil {
		return
	}
	tv.CopyRect(true)
}

// PasteRect cuts rectangle in active text view
func (ge *CodeView) PasteRect() { //types:add
	tv := ge.ActiveTextEditor()
	if tv.Buffer == nil {
		return
	}
	tv.PasteRect()
}

// RegisterCopy saves current selection in active text view to register of given name
// returns true if saved
func (ge *CodeView) RegisterCopy(name string) bool { //types:add
	if name == "" {
		return false
	}
	tv := ge.ActiveTextEditor()
	if tv.Buffer == nil {
		return false
	}
	sel := tv.Selection()
	if sel == nil {
		return false
	}
	if code.AvailableRegisters == nil {
		code.AvailableRegisters = make(code.Registers, 100)
	}
	code.AvailableRegisters[name] = string(sel.ToBytes())
	code.AvailableRegisters.SaveSettings()
	ge.Settings.Register = code.RegisterName(name)
	tv.SelectReset()
	return true
}

// RegisterPaste pastes register of given name into active text view
// returns true if pasted
func (ge *CodeView) RegisterPaste(name code.RegisterName) bool { //types:add
	if name == "" {
		return false
	}
	str, ok := code.AvailableRegisters[string(name)]
	if !ok {
		return false
	}
	tv := ge.ActiveTextEditor()
	if tv.Buffer == nil {
		return false
	}
	tv.InsertAtCursor([]byte(str))
	ge.Settings.Register = name
	return true
}

// CommentOut comments-out selected lines in active text view
// and uncomments if already commented
// If multiple lines are selected and any line is uncommented all will be commented
func (ge *CodeView) CommentOut() bool { //types:add
	tv := ge.ActiveTextEditor()
	if tv.Buffer == nil {
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
	tv.Buffer.CommentRegion(stl, etl)
	tv.SelectReset()
	return true
}

// Indent indents selected lines in active view
func (ge *CodeView) Indent() bool { //types:add
	tv := ge.ActiveTextEditor()
	if tv.Buffer == nil {
		return false
	}
	sel := tv.Selection()
	if sel == nil {
		return false
	}
	tv.Buffer.AutoIndentRegion(sel.Reg.Start.Ln, sel.Reg.End.Ln)
	tv.SelectReset()
	return true
}

// ReCase replaces currently selected text in current active view with given case
func (ge *CodeView) ReCase(c strcase.Cases) string { //types:add
	tv := ge.ActiveTextEditor()
	if tv.Buffer == nil {
		return ""
	}
	return tv.ReCaseSelection(c)
}

// JoinParaLines merges sequences of lines with hard returns forming paragraphs,
// separated by blank lines, into a single line per paragraph,
// for given selected region (full text if no selection)
func (ge *CodeView) JoinParaLines() { //types:add
	tv := ge.ActiveTextEditor()
	if tv.Buffer == nil {
		return
	}
	if tv.HasSelection() {
		tv.Buffer.JoinParaLines(tv.SelectRegion.Start.Ln, tv.SelectRegion.End.Ln)
	} else {
		tv.Buffer.JoinParaLines(0, tv.NLines-1)
	}
}

// TabsToSpaces converts tabs to spaces
// for given selected region (full text if no selection)
func (ge *CodeView) TabsToSpaces() { //types:add
	tv := ge.ActiveTextEditor()
	if tv.Buffer == nil {
		return
	}
	if tv.HasSelection() {
		tv.Buffer.TabsToSpacesRegion(tv.SelectRegion.Start.Ln, tv.SelectRegion.End.Ln)
	} else {
		tv.Buffer.TabsToSpacesRegion(0, tv.NLines-1)
	}
}

// SpacesToTabs converts spaces to tabs
// for given selected region (full text if no selection)
func (ge *CodeView) SpacesToTabs() { //types:add
	tv := ge.ActiveTextEditor()
	if tv.Buffer == nil {
		return
	}
	if tv.HasSelection() {
		tv.Buffer.SpacesToTabsRegion(tv.SelectRegion.Start.Ln, tv.SelectRegion.End.Ln)
	} else {
		tv.Buffer.SpacesToTabsRegion(0, tv.NLines-1)
	}
}

// DiffFiles shows the differences between two given files
// in side-by-side DiffView and in the console as a context diff.
// It opens the files as file nodes and uses existing contents if open already.
func (ge *CodeView) DiffFiles(fnmA, fnmB core.Filename) { //types:add
	fna := ge.FileNodeForFile(string(fnmA), true)
	if fna == nil {
		return
	}
	if fna.Buffer == nil {
		ge.OpenFileNode(fna)
	}
	if fna.Buffer == nil {
		return
	}
	ge.DiffFileNode(fna, fnmB)
}

// DiffFileNode shows the differences between given file node as the A file,
// and another given file as the B file,
// in side-by-side DiffView and in the console as a context diff.
func (ge *CodeView) DiffFileNode(fna *filetree.Node, fnmB core.Filename) { //types:add
	fnb := ge.FileNodeForFile(string(fnmB), true)
	if fnb == nil {
		return
	}
	if fnb.Buffer == nil {
		ge.OpenFileNode(fnb)
	}
	if fnb.Buffer == nil {
		return
	}
	dif := fna.Buffer.DiffBuffersUnified(fnb.Buffer, 3)
	cbuf, _, _ := ge.RecycleCmdTab("Diffs", true, true)
	cbuf.SetText(dif)
	cbuf.AutoScrollViews()

	astr := fna.Buffer.Strings(false)
	bstr := fnb.Buffer.Strings(false)
	_, _ = astr, bstr

	texteditor.DiffViewDialog(ge, "Diff File View:", astr, bstr, string(fna.Buffer.Filename), string(fnb.Buffer.Filename), "", "")
}

// CountWords counts number of words (and lines) in active file
// returns a string report thereof.
func (ge *CodeView) CountWords() string { //types:add
	av := ge.ActiveTextEditor()
	if av.Buffer == nil || av.Buffer.NLines <= 0 {
		return "empty"
	}
	av.Buffer.LinesMu.RLock()
	defer av.Buffer.LinesMu.RUnlock()
	ll := av.Buffer.NLines - 1
	reg := textbuf.NewRegion(0, 0, ll, len(av.Buffer.Lines[ll]))
	words, lines := textbuf.CountWordsLinesRegion(av.Buffer.Lines, reg)
	return fmt.Sprintf("File: %s  Words: %d   Lines: %d\n", dirs.DirAndFile(string(av.Buffer.Filename)), words, lines)
}

// CountWordsRegion counts number of words (and lines) in selected region in file
// if no selection, returns numbers for entire file.
func (ge *CodeView) CountWordsRegion() string { //types:add
	av := ge.ActiveTextEditor()
	if av.Buffer == nil || av.Buffer.NLines <= 0 {
		return "empty"
	}
	if !av.HasSelection() {
		return ge.CountWords()
	}
	av.Buffer.LinesMu.RLock()
	defer av.Buffer.LinesMu.RUnlock()
	sel := av.Selection()
	words, lines := textbuf.CountWordsLinesRegion(av.Buffer.Lines, sel.Reg)
	return fmt.Sprintf("File: %s  Words: %d   Lines: %d\n", dirs.DirAndFile(string(av.Buffer.Filename)), words, lines)
}

//////////////////////////////////////////////////////////////////////////////////////
//   Links

// TextLinkHandler is the CodeView handler for text links -- preferred one b/c
// directly connects to correct CodeView project
func TextLinkHandler(tl paint.TextLink) bool {
	// todo:
	// tve := texteditor.AsEditor(tl.Widget)
	// ftv, _ := tl.Widget.Embed(views.KiT_TextEditor).(*texteditor.Editor)
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
	// 		system.TheApp.OpenURL(ur)
	// 	}
	// } else {
	// 	system.TheApp.OpenURL(tl.URL)
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
	if ftv != nil && ftv.Buffer != nil { // get cd path for non-pathed fnames
		cdln := ftv.Buffer.BytesLine(0)
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
	tv, _, ok := ge.LinkViewFile(core.Filename(fpath))
	if !ok {
		_, fnm := filepath.Split(fpath)
		tv, _, ok = ge.LinkViewFile(core.Filename(fnm))
		if !ok {
			core.MessageSnackbar(ge, fmt.Sprintf("Could not find or open file path in project: %v", fpath))
			return false
		}
	}
	if pos == "" {
		return true
	}
	// fmt.Printf("pos: %v\n", pos)
	txpos := lexer.Pos{}
	if txpos.FromString(pos) {
		reg := textbuf.Region{Start: txpos, End: lexer.Pos{Ln: txpos.Ln, Ch: txpos.Ch + 4}}
		// todo: need some way of tagging the time stamp for adjusting!
		// reg = tv.Buf.AdjustReg(reg)
		txpos = reg.Start
		tv.HighlightRegion(reg)
		tv.SetCursorTarget(txpos)
		tv.NeedsLayout()
	}
	return true
}
