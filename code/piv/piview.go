// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package piv provides the PiView object for the full GUI view of the
// interactive parser (pi) system.
package piv

// TODO: piv

//go:generate core generate

/*
import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"cogentcore.org/core/gi/filetree"
	"cogentcore.org/core/core"
	"cogentcore.org/core/giv"
	"cogentcore.org/core/gi/texteditor"
	"cogentcore.org/cogent/code/code"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/units"
	"cogentcore.org/core/system"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/parse/lex"
	"cogentcore.org/core/parse/parse"
	"cogentcore.org/core/parse"
)

// These are then the fixed indices of the different elements in the splitview
const (
	LexRulesIndex = iota
	ParseRulesIndex
	StructViewIndex
	AstOutIndex
	MainTabsIndex
)

// PiView provides the interactive GUI view for constructing and testing the
// lexer and parser
type PiView struct {
	core.Frame

	// the parser we are viewing
	Parser pi.Parser

	// project settings -- this IS the project file
	Settings ProjSettings

	// has the root changed?  we receive update signals from root for changes
	Changed bool `json:"-"`

	// our own dedicated filestate for controlled parsing
	FileState pi.FileState `json:"-"`

	// test file buffer
	TestBuf texteditor.Buf `json:"-"`

	// output buffer -- shows all errors, tracing
	OutputBuffer texteditor.Buf `json:"-"`

	// buffer of lexified tokens
	LexBuf texteditor.Buf `json:"-"`

	// buffer of parse info
	ParseBuf texteditor.Buf `json:"-"`

	// first key in sequence if needs2 key pressed
	KeySeq1 key.Chord

	// is the output monitor running?
	OutMonRunning bool `json:"-"`

	// mutex for updating, checking output monitor run status
	OutMonMu sync.Mutex `json:"-"`
}

// IsEmpty returns true if current project is empty
func (pv *PiView) IsEmpty() bool {
	return (!pv.Parser.Lexer.HasChildren() && !pv.Parser.Parser.HasChildren())
}

// OpenRecent opens a recently-used project
func (pv *PiView) OpenRecent(filename core.Filename) { //types:add
	pv.OpenProj(filename)
}

// OpenProj opens lexer and parser rules to current filename, in a standard JSON-formatted file
// if current is not empty, opens in a new window
func (pv *PiView) OpenProj(filename core.Filename) *PiView { //types:add
	if !pv.IsEmpty() {
		_, nprj := NewPiView()
		nprj.OpenProj(filename)
		return nprj
	}
	pv.Settings.OpenJSON(filename)
	pv.Config()
	pv.ApplySettings()
	SavedPaths.AddPath(string(filename), core.Settings.Params.SavedPathsMax)
	SavePaths()
	return pv
}

// NewProj makes a new project in a new window
func (pv *PiView) NewProj() (*core.Window, *PiView) { //types:add
	return NewPiView()
}

// SaveProj saves project prefs to current filename, in a standard JSON-formatted file
// also saves the current parser
func (pv *PiView) SaveProj() { //types:add
	if pv.Settings.ProjectFile == "" {
		return
	}
	pv.SaveParser()
	pv.GetSettings()
	pv.Settings.SaveJSON(pv.Settings.ProjectFile)
	pv.Changed = false
	pv.SetStatus(fmt.Sprintf("Project Saved to: %v", pv.Settings.ProjectFile))
	pv.UpdateSig() // notify our editor
}

// SaveProjAs saves lexer and parser rules to current filename, in a standard JSON-formatted file
// also saves the current parser
func (pv *PiView) SaveProjAs(filename core.Filename) { //types:add
	SavedPaths.AddPath(string(filename), core.Settings.Params.SavedPathsMax)
	SavePaths()
	pv.SaveParser()
	pv.GetSettings()
	pv.Settings.SaveJSON(filename)
	pv.Changed = false
	pv.SetStatus(fmt.Sprintf("Project Saved to: %v", pv.Settings.ProjectFile))
	pv.UpdateSig() // notify our editor
}

// ApplySettings applies project-level prefs (e.g., after opening)
func (pv *PiView) ApplySettings() { //types:add
	fs := &pv.FileState
	fs.ParseState.Trace.CopyOpts(&pv.Settings.TraceOpts)
	if pv.Settings.ParserFile != "" {
		pv.OpenParser(pv.Settings.ParserFile)
	}
	if pv.Settings.TestFile != "" {
		pv.OpenTest(pv.Settings.TestFile)
	}
}

// GetSettings gets the current values of things for prefs
func (pv *PiView) GetSettings() {
	fs := &pv.FileState
	pv.Settings.TraceOpts.CopyOpts(&fs.ParseState.Trace)
}

/////////////////////////////////////////////////////////////////////////
//  other IO

// OpenParser opens lexer and parser rules to current filename, in a standard JSON-formatted file
func (pv *PiView) OpenParser(filename core.Filename) { //types:add
	pv.Parser.OpenJSON(string(filename))
	pv.Settings.ParserFile = filename
	pv.Config()
}

// SaveParser saves lexer and parser rules to current filename, in a standard JSON-formatted file
func (pv *PiView) SaveParser() { //types:add
	if pv.Settings.ParserFile == "" {
		return
	}
	pv.Parser.SaveJSON(string(pv.Settings.ParserFile))

	ext := filepath.Ext(string(pv.Settings.ParserFile))
	pigfn := strings.TrimSuffix(string(pv.Settings.ParserFile), ext) + ".parsegrammar"
	pv.Parser.SaveGrammar(pigfn)

	pv.Changed = false
	pv.SetStatus(fmt.Sprintf("Parser Saved to: %v", pv.Settings.ParserFile))
	pv.UpdateSig() // notify our editor
}

// SaveParserAs saves lexer and parser rules to current filename, in a standard JSON-formatted file
func (pv *PiView) SaveParserAs(filename core.Filename) { //types:add
	pv.Parser.SaveJSON(string(filename))

	ext := filepath.Ext(string(pv.Settings.ParserFile))
	pigfn := strings.TrimSuffix(string(pv.Settings.ParserFile), ext) + ".parsegrammar"
	pv.Parser.SaveGrammar(pigfn)

	pv.Changed = false
	pv.Settings.ParserFile = filename
	pv.SetStatus(fmt.Sprintf("Parser Saved to: %v", pv.Settings.ParserFile))
	pv.UpdateSig() // notify our editor
}

// OpenTest opens test file
func (pv *PiView) OpenTest(filename core.Filename) { //types:add
	pv.TestBuf.OpenFile(filename)
	pv.Settings.TestFile = filename
}

// SaveTestAs saves the test file as..
func (pv *PiView) SaveTestAs(filename core.Filename) {
	pv.TestBuf.EditDone()
	pv.TestBuf.SaveFile(filename)
	pv.Settings.TestFile = filename
	pv.SetStatus(fmt.Sprintf("TestFile Saved to: %v", pv.Settings.TestFile))
}

// SetStatus updates the statusbar label with given message, along with other status info
func (pv *PiView) SetStatus(msg string) {
	sb := pv.StatusBar()
	if sb == nil {
		return
	}
	// pv.UpdateMu.Lock()
	// defer pv.UpdateMu.Unlock()

	updt := sb.UpdateStart()
	lbl := pv.StatusLabel()
	fnm := ""
	ln := 0
	ch := 0
	if tv, ok := pv.TestTextEditor(); ok {
		ln = tv.CursorPos.Ln + 1
		ch = tv.CursorPos.Ch
		if tv.ISearch.On {
			msg = fmt.Sprintf("\tISearch: %v (n=%v)\t%v", tv.ISearch.Find, len(tv.ISearch.Matches), msg)
		}
		if tv.QReplace.On {
			msg = fmt.Sprintf("\tQReplace: %v -> %v (n=%v)\t%v", tv.QReplace.Find, tv.QReplace.Replace, len(tv.QReplace.Matches), msg)
		}
	}

	str := fmt.Sprintf("%v\t<b>%v:</b>\t(%v,%v)\t%v", pv.Nm, fnm, ln, ch, msg)
	lbl.SetText(str)
	sb.UpdateEnd(updt)
}

////////////////////////////////////////////////////////////////////////////////////////
//  Lexing

// LexInit initializes / restarts lexing process for current test file
func (pv *PiView) LexInit() {
	pv.OutputBuffer.New(0)
	go pv.MonitorOut()
	fs := &pv.FileState
	fs.SetSrc(pv.TestBuf.Lines, string(pv.TestBuf.Filename), "", pv.TestBuf.Info.Sup)
	// pv.Hi.SetParser(&pv.Parser)
	pv.Parser.Lexer.CompileAll(&fs.LexState)
	pv.Parser.Lexer.Validate(&fs.LexState)
	pv.Parser.LexInit(fs)
	if fs.LexHasErrs() {
		errs := fs.LexErrReport()
		fs.ParseState.Trace.OutWrite.Write([]byte(errs)) // goes to outbuf
		core.PromptDialog(pv.Viewport, core.DlgOpts{Title: "Lex Error",
			Prompt: "The Lexer validation has errors<br>\n" + errs}, core.AddOK, core.NoCancel, nil, nil)
	}
	pv.UpdateLexBuf()
}

// LexStopped tells the user why the lexer stopped
func (pv *PiView) LexStopped() {
	fs := &pv.FileState
	if fs.LexAtEnd() {
		pv.SetStatus("The Lexer is now at the end of available text")
	} else {
		errs := fs.LexErrReport()
		if errs != "" {
			fs.ParseState.Trace.OutWrite.Write([]byte(errs)) // goes to outbuf
			pv.SetStatus("Lexer Errors!")
			core.PromptDialog(pv.Viewport, core.DlgOpts{Title: "Lex Error",
				Prompt: "The Lexer has stopped due to errors<br>\n" + errs}, core.AddOK, core.NoCancel, nil, nil)
		} else {
			pv.SetStatus("Lexer Missing Rules!")
			core.PromptDialog(pv.Viewport, core.DlgOpts{Title: "Lex Error",
				Prompt: "The Lexer has stopped because it cannot process the source at this point:<br>\n" + fs.LexNextSrcLine()}, core.AddOK, core.NoCancel, nil, nil)
		}
	}
}

// LexNext does next step of lexing
func (pv *PiView) LexNext() *lexer.Rule {
	fs := &pv.FileState
	mrule := pv.Parser.LexNext(fs)
	if mrule == nil {
		pv.LexStopped()
	} else {
		pv.SetStatus(mrule.Nm + ": " + fs.LexLineString())
		pv.SelectLexRule(mrule)
	}
	pv.UpdateLexBuf()
	return mrule
}

// LexLine does next line of lexing
func (pv *PiView) LexNextLine() *lexer.Rule {
	fs := &pv.FileState
	mrule := pv.Parser.LexNextLine(fs)
	if mrule == nil && fs.LexHasErrs() {
		pv.LexStopped()
	} else if mrule != nil {
		pv.SetStatus(mrule.Nm + ": " + fs.LexLineString())
		pv.SelectLexRule(mrule)
	}
	pv.UpdateLexBuf()
	return mrule
}

// LexAll does all remaining lexing until end or error
func (pv *PiView) LexAll() {
	fs := &pv.FileState
	for {
		mrule := pv.Parser.LexNext(fs)
		if mrule == nil {
			if !fs.LexAtEnd() {
				pv.LexStopped()
			}
			break
		}
	}
	pv.UpdateLexBuf()
}

// SelectLexRule selects given lex rule in Lexer
func (pv *PiView) SelectLexRule(rule *lexer.Rule) {
	lt := pv.LexTree()
	lt.UnselectAll()
	lt.FuncDownMeFirst(0, lt.This(), func(k tree.Node, level int, d any) bool {
		lnt := k.Embed(views.KiT_TreeView)
		if lnt == nil {
			return true
		}
		ln := lnt.(*views.TreeView)
		if ln.SrcNode == rule.This() {
			ln.Select()
			return false
		}
		return true
	})
}

// UpdateLexBuf sets the LexBuf to current lex content
func (pv *PiView) UpdateLexBuf() {
	fs := &pv.FileState
	txt := fs.Src.LexTagSrc()
	pv.LexBuf.SetText([]byte(txt))
	pv.TestBuf.HiTags = fs.Src.Lexs
	pv.TestBuf.MarkupFromTags()
	pv.TestBuf.Update()
}

////////////////////////////////////////////////////////////////////////////////////////
//  PassTwo

// EditPassTwo shows the PassTwo settings to edit -- does nest depth and finds the EOS end-of-statements
func (pv *PiView) EditPassTwo() {
	sv := pv.StructView()
	if sv != nil {
		sv.SetStruct(&pv.Parser.PassTwo)
	}
}

// PassTwo does the second pass after lexing, per current settings
func (pv *PiView) PassTwo() {
	pv.OutputBuffer.New(0)
	fs := &pv.FileState
	pv.Parser.DoPassTwo(fs)
	if fs.PassTwoHasErrs() {
		errs := fs.PassTwoErrReport()
		fs.ParseState.Trace.OutWrite.Write([]byte(errs)) // goes to outbuf
		core.PromptDialog(pv.Viewport, core.DlgOpts{Title: "PassTwo Error",
			Prompt: "The PassTwo had the following errors<br>\n" + errs}, core.AddOK, core.NoCancel, nil, nil)
	}
}

////////////////////////////////////////////////////////////////////////////////////////
//  Parsing

// EditTrace shows the parser.Trace options for detailed tracing output
func (pv *PiView) EditTrace() {
	sv := pv.StructView()
	if sv != nil {
		fs := &pv.FileState
		sv.SetStruct(&fs.ParseState.Trace)
	}
}

// ParseInit initializes / restarts lexing process for current test file
func (pv *PiView) ParseInit() {
	fs := &pv.FileState
	pv.OutputBuffer.New(0)
	go pv.MonitorOut()
	pv.LexInit()
	pv.Parser.LexAll(fs)
	pv.Parser.Parser.CompileAll(&fs.ParseState)
	pv.Parser.Parser.Validate(&fs.ParseState)
	pv.Parser.ParserInit(fs)
	pv.UpdateLexBuf()
	if fs.ParseHasErrs() {
		errs := fs.ParseErrReportDetailed()
		core.PromptDialog(pv.Viewport, core.DlgOpts{Title: "Parse Error",
			Prompt: "The Parser validation has errors<br>\n" + errs}, core.AddOK, core.NoCancel, nil, nil)
	}
}

// ParseStopped tells the user why the lexer stopped
func (pv *PiView) ParseStopped() {
	fs := &pv.FileState
	if fs.ParseAtEnd() && !fs.ParseHasErrs() {
		pv.SetStatus("The Parser is now at the end of available text")
	} else {
		errs := fs.ParseErrReportDetailed()
		if errs != "" {
			pv.SetStatus("Parse Error!")
			core.PromptDialog(pv.Viewport, core.DlgOpts{Title: "Parse Error",
				Prompt: "The Parser has the following errors (see Output tab for full list)<br>\n" + errs}, core.AddOK, core.NoCancel, nil, nil)
		} else {
			pv.SetStatus("Parse Missing Rules!")
			core.PromptDialog(pv.Viewport, core.DlgOpts{Title: "Parse Error",
				Prompt: "The Parser has stopped because it cannot process the source at this point:<br>\n" + fs.ParseNextSrcLine()}, core.AddOK, core.NoCancel, nil, nil)
		}
	}
}

// ParseNext does next step of lexing
func (pv *PiView) ParseNext() *parser.Rule {
	fs := &pv.FileState
	at := pv.AstTree()
	updt := at.UpdateStart()
	mrule := pv.Parser.ParseNext(fs)
	at.UpdateEnd(updt)
	at.OpenAll()
	pv.AstTreeToEnd()
	pv.UpdateLexBuf()
	pv.UpdateParseBuf()
	if mrule == nil {
		pv.ParseStopped()
	} else {
		// pv.SelectParseRule(mrule) // not that informative
		if fs.ParseHasErrs() { // can have errs even when matching..
			pv.ParseStopped()
		}
	}
	return mrule
}

// ParseAll does all remaining lexing until end or error
func (pv *PiView) ParseAll() {
	fs := &pv.FileState
	at := pv.AstTree()
	updt := at.UpdateStart()
	for {
		mrule := pv.Parser.ParseNext(fs)
		if mrule == nil || fs.ParseState.AtEofNext() {
			break
		}
	}
	at.UpdateEnd(updt)
	// at.OpenAll()
	// pv.AstTreeToEnd()
	pv.UpdateLexBuf()
	pv.UpdateParseBuf()
	pv.ParseStopped()
}

// SelectParseRule selects given lex rule in Parser
func (pv *PiView) SelectParseRule(rule *parser.Rule) {
	lt := pv.ParseTree()
	lt.UnselectAll()
	lt.FuncDownMeFirst(0, lt.This(), func(k tree.Node, level int, d any) bool {
		lnt := k.Embed(views.KiT_TreeView)
		if lnt == nil {
			return true
		}
		ln := lnt.(*views.TreeView)
		if ln.SrcNode == rule.This() {
			ln.Select()
			return false
		}
		return true
	})
}

// AstTreeToEnd
func (pv *PiView) AstTreeToEnd() {
	lt := pv.AstTree()
	lt.MoveEndAction(events.SelectOne)
}

// UpdateParseBuf sets the ParseBuf to current parse rule output
func (pv *PiView) UpdateParseBuf() {
	fs := &pv.FileState
	txt := fs.ParseRuleString(fs.ParseState.Trace.FullStackOut)
	pv.ParseBuf.SetText([]byte(txt))
}

// ViewParseState
func (pv *PiView) ViewParseState() {
	sv := pv.StructView()
	if sv != nil {
		sv.SetStruct(&pv.FileState.ParseState)
	}
}

//////////////////////////////////////////////////////////////////////////////////////
//   Panels

// CurPanel returns the splitter panel that currently has keyboard focus
func (pv *PiView) CurPanel() int {
	sv := pv.Splits()
	if sv == nil {
		return -1
	}
	for i, ski := range sv.Kids {
		_, sk := core.KiToNode2D(ski)
		if sk.ContainsFocus() {
			return i
		}
	}
	return -1 // nobody
}

// FocusOnPanel moves keyboard focus to given panel -- returns false if nothing at that tab
func (pv *PiView) FocusOnPanel(panel int) bool {
	sv := pv.Splits()
	if sv == nil {
		return false
	}
	win := pv.ParentWindow()
	ski := sv.Kids[panel]
	win.EventMgr.FocusNext(ski)
	return true
}

// FocusNextPanel moves the keyboard focus to the next panel to the right
func (pv *PiView) FocusNextPanel() {
	sv := pv.Splits()
	if sv == nil {
		return
	}
	cp := pv.CurPanel()
	cp++
	np := len(sv.Kids)
	if cp >= np {
		cp = 0
	}
	for sv.Splits[cp] <= 0.01 {
		cp++
		if cp >= np {
			cp = 0
		}
	}
	pv.FocusOnPanel(cp)
}

// FocusPrevPanel moves the keyboard focus to the previous panel to the left
func (pv *PiView) FocusPrevPanel() {
	sv := pv.Splits()
	if sv == nil {
		return
	}
	cp := pv.CurPanel()
	cp--
	np := len(sv.Kids)
	if cp < 0 {
		cp = np - 1
	}
	for sv.Splits[cp] <= 0.01 {
		cp--
		if cp < 0 {
			cp = np - 1
		}
	}
	pv.FocusOnPanel(cp)
}

//////////////////////////////////////////////////////////////////////////////////////
//   Tabs

// MainTabByName returns a MainTabs (first set of tabs) tab with given name
func (pv *PiView) MainTabByName(label string) core.Widget {
	tv := pv.MainTabs()
	return tv.TabByName(label)
}

// MainTabByNameTry returns a MainTabs (first set of tabs) tab with given name, err if not found
func (pv *PiView) MainTabByNameTry(label string) (core.Widget, error) {
	tv := pv.MainTabs()
	return tv.TabByNameTry(label)
}

// SelectMainTabByName Selects given main tab, and returns all of its contents as well.
func (pv *PiView) SelectMainTabByName(label string) core.Widget {
	tv := pv.MainTabs()
	widg, err := pv.MainTabByNameTry(label)
	if err == nil {
		tv.SelectTabByName(label)
	}
	return widg
}

// RecycleMainTab returns a MainTabs (first set of tabs) tab with given
// name, first by looking for an existing one, and if not found, making a new
// one with widget of given type.  if sel, then select it.  returns widget
func (pv *PiView) RecycleMainTab(label string, typ reflect.Type, sel bool) core.Widget {
	tv := pv.MainTabs()
	widg, err := pv.MainTabByNameTry(label)
	if err == nil {
		if sel {
			tv.SelectTabByName(label)
		}
		return widg
	}
	widg = tv.NewTab(typ, label)
	if sel {
		tv.SelectTabByName(label)
	}
	return widg
}

// ConfigTextEditor configures text view
func (pv *PiView) ConfigTextEditor(ly *core.Layout, out bool) *texteditor.Editor {
	ly.Lay = core.LayoutVert
	ly.SetStretchMaxWidth()
	ly.SetStretchMaxHeight()
	ly.SetMinPrefWidth(units.NewValue(20, units.Ch))
	ly.SetMinPrefHeight(units.NewValue(10, units.Ch))
	var tv *texteditor.Editor
	updt := false
	if ly.HasChildren() {
		tv = ly.Child(0).Embed(views.KiT_TextEditor).(*texteditor.Editor)
	} else {
		updt = ly.UpdateStart()
		tv = ly.NewChild(views.KiT_TextEditor, ly.Nm).(*texteditor.Editor)
	}

	if core.Settings.Editor.WordWrap {
		tv.SetProp("white-space", styles.WhiteSpacePreWrap)
	} else {
		tv.SetProp("white-space", styles.WhiteSpacePre)
	}
	tv.SetProp("tab-size", 4)
	tv.SetProp("font-family", core.Settings.MonoFont)
	if out {
		tv.SetInactive()
	}
	ly.UpdateEnd(updt)
	return tv
}

// RecycleMainTabTextEditor returns a MainTabs (first set of tabs) tab with given
// name, first by looking for an existing one, and if not found, making a new
// one with a Layout and then a TextEditor in it.  if sel, then select it.
// returns widget
func (pv *PiView) RecycleMainTabTextEditor(label string, sel bool, out bool) *texteditor.Editor {
	ly := pv.RecycleMainTab(label, core.LayoutType, sel).Embed(core.LayoutType).(*core.Layout)
	tv := pv.ConfigTextEditor(ly, out)
	return tv
}

// MainTabTextEditorByName returns the texteditor for given main tab, if it exists
func (pv *PiView) MainTabTextEditorByName(tabnm string) (*texteditor.Editor, bool) {
	lyk, err := pv.MainTabByNameTry(tabnm)
	if err != nil {
		return nil, false
	}
	ctv := lyk.Child(0).Embed(views.KiT_TextEditor).(*texteditor.Editor)
	return ctv, true
}

// TextTextEditor returns the texteditor for TestBuf TextEditor
func (pv *PiView) TestTextEditor() (*texteditor.Editor, bool) {
	return pv.MainTabTextEditorByName("TestText")
}

// OpenConsoleTab opens a main tab displaying console output (stdout, stderr)
func (pv *PiView) OpenConsoleTab() {
	ctv := pv.RecycleMainTabTextEditor("Console", true, true)
	ctv.SetInactive()
	ctv.SetProp("white-space", styles.WhiteSpacePre) // no word wrap
	if ctv.Buf == nil || ctv.Buf != code.TheConsole.Buf {
		ctv.SetBuf(code.TheConsole.Buf)
		code.TheConsole.Buf.TextBufSig.Connect(pv.This(), func(recv, send tree.Node, sig int64, data any) {
			pve, _ := recv.Embed(KiT_PiView).(*PiView)
			pve.SelectMainTabByName("Console")
		})
	}
}

// OpenTestTextTab opens a main tab displaying test text
func (pv *PiView) OpenTestTextTab() {
	ctv := pv.RecycleMainTabTextEditor("TestText", true, false)
	if ctv.Buf == nil || ctv.Buf != &pv.TestBuf {
		ctv.SetBuf(&pv.TestBuf)
	}
}

// OpenOutTab opens a main tab displaying all output
func (pv *PiView) OpenOutTab() {
	ctv := pv.RecycleMainTabTextEditor("Output", true, true)
	ctv.SetInactive()
	ctv.SetProp("white-space", styles.WhiteSpacePre) // no word wrap
	if ctv.Buf == nil || ctv.Buf != &pv.OutputBuffer {
		ctv.SetBuf(&pv.OutputBuffer)
	}
}

// OpenLexTab opens a main tab displaying lexer output
func (pv *PiView) OpenLexTab() {
	ctv := pv.RecycleMainTabTextEditor("LexOut", true, true)
	if ctv.Buf == nil || ctv.Buf != &pv.LexBuf {
		ctv.SetBuf(&pv.LexBuf)
	}
}

// OpenParseTab opens a main tab displaying parser output
func (pv *PiView) OpenParseTab() {
	ctv := pv.RecycleMainTabTextEditor("ParseOut", true, true)
	if ctv.Buf == nil || ctv.Buf != &pv.ParseBuf {
		ctv.SetBuf(&pv.ParseBuf)
	}
}

//////////////////////////////////////////////////////////////////////////////////////
//   GUI configs

// Config configures the view
func (pv *PiView) Config() {
	parser.GuiActive = true
	fmt.Printf("PiView enabling GoPi parser output\n")
	pv.Parser.Init()
	pv.Lay = core.LayoutVert
	pv.SetProp("spacing", core.StdDialogVSpaceUnits)
	config := tree.Config{}
	config.Add(core.ToolbarType, "toolbar")
	config.Add(core.SplitsType, "splitview")
	config.Add(core.FrameType, "statusbar")
	mods, updt := pv.ConfigChildren(config)
	if !mods {
		updt = pv.UpdateStart()
	}
	pv.ConfigSplits()
	pv.ConfigStatusBar()
	pv.ConfigToolbar()
	pv.UpdateEnd(updt)
	go pv.MonitorOut()
}

// IsConfiged returns true if the view is fully configured
func (pv *PiView) IsConfiged() bool {
	if len(pv.Kids) == 0 {
		return false
	}
	sv := pv.Splits()
	if len(sv.Kids) == 0 {
		return false
	}
	return true
}

// Splits returns the main Splits
func (pv *PiView) Splits() *core.Splits {
	return pv.ChildByName("splitview", 4).(*core.Splits)
}

// LexTree returns the lex rules tree view
func (pv *PiView) LexTree() *views.TreeView {
	return pv.Splits().Child(LexRulesIndex).Child(0).(*views.TreeView)
}

// ParseTree returns the parse rules tree view
func (pv *PiView) ParseTree() *views.TreeView {
	return pv.Splits().Child(ParseRulesIndex).Child(0).(*views.TreeView)
}

// AstTree returns the Ast output tree view
func (pv *PiView) AstTree() *views.TreeView {
	return pv.Splits().Child(AstOutIndex).Child(0).(*views.TreeView)
}

// StructView returns the StructView for editing rules
func (pv *PiView) StructView() *views.StructView {
	return pv.Splits().Child(StructViewIndex).(*views.StructView)
}

// MainTabs returns the main TabView
func (pv *PiView) MainTabs() *core.TabView {
	return pv.Splits().Child(MainTabsIndex).Embed(core.KiT_TabView).(*core.TabView)
}

// StatusBar returns the statusbar widget
func (pv *PiView) StatusBar() *core.Frame {
	return pv.ChildByName("statusbar", 2).(*core.Frame)
}

// StatusLabel returns the statusbar label widget
func (pv *PiView) StatusLabel() *core.Label {
	return pv.StatusBar().Child(0).Embed(core.LabelType).(*core.Label)
}

// Toolbar returns the toolbar widget
func (pv *PiView) Toolbar() *core.Toolbar {
	return pv.ChildByName("toolbar", 0).(*core.Toolbar)
}

// ConfigStatusBar configures statusbar with label
func (pv *PiView) ConfigStatusBar() {
	sb := pv.StatusBar()
	if sb == nil || sb.HasChildren() {
		return
	}
	sb.SetStretchMaxWidth()
	sb.SetMinPrefHeight(units.NewValue(1.2, units.Em))
	sb.SetProp("overflow", "hidden") // no scrollbars!
	sb.SetProp("margin", 0)
	sb.SetProp("padding", 0)
	lbl := sb.NewChild(core.LabelType, "sb-lbl").(*core.Label)
	lbl.SetStretchMaxWidth()
	lbl.SetMinPrefHeight(units.NewValue(1, units.Em))
	lbl.SetProp("vertical-align", styles.AlignTop)
	lbl.SetProp("margin", 0)
	lbl.SetProp("padding", 0)
	lbl.SetProp("tab-size", 4)
}

// ConfigToolbar adds a PiView toolbar.
func (pv *PiView) ConfigToolbar() {
	tb := pv.Toolbar()
	if tb.HasChildren() {
		return
	}
	tb.SetStretchMaxWidth()
	views.ToolbarView(pv, pv.Viewport, tb)
}

// SplitsConfig returns a TypeAndNameList for configuring the Splits
func (pv *PiView) SplitsConfig() tree.Config {
	config := tree.Config{}
	config.Add(core.FrameType, "lex-tree-fr")
	config.Add(core.FrameType, "parse-tree-fr")
	config.Add(views.KiT_StructView, "struct-view")
	config.Add(core.FrameType, "ast-tree-fr")
	config.Add(core.KiT_TabView, "main-tabs")
	return config
}

// MonitorOut sets up the OutputBuffer monitor -- must call as separate goroutine using go
func (pv *PiView) MonitorOut() {
	pv.OutMonMu.Lock()
	if pv.OutMonRunning {
		pv.OutMonMu.Unlock()
		return
	}
	pv.OutMonRunning = true
	pv.OutMonMu.Unlock()
	obuf := texteditor.OutputBuffer{}
	fs := &pv.FileState
	obuf.Init(fs.ParseState.Trace.OutRead, &pv.OutputBuffer, 0, code.MarkupCmdOutput)
	obuf.MonOut()
	pv.OutMonMu.Lock()
	pv.OutMonRunning = false
	pv.OutMonMu.Unlock()
}

// ConfigSplits configures the Splits.
func (pv *PiView) ConfigSplits() {
	fs := &pv.FileState
	split := pv.Splits()
	if split == nil {
		return
	}
	split.Dim = core.X

	split.SetProp("white-space", styles.WhiteSpacePreWrap)
	split.SetProp("tab-size", 4)

	config := pv.SplitsConfig()
	mods, updt := split.ConfigChildren(config)
	if mods {
		lxfr := split.Child(LexRulesIndex).(*core.Frame)
		lxt := lxfr.NewChild(views.KiT_TreeView, "lex-tree").(*views.TreeView)
		lxt.SetRootNode(&pv.Parser.Lexer)

		prfr := split.Child(ParseRulesIndex).(*core.Frame)
		prt := prfr.NewChild(views.KiT_TreeView, "parse-tree").(*views.TreeView)
		prt.SetRootNode(&pv.Parser.Parser)

		astfr := split.Child(AstOutIndex).(*core.Frame)
		astt := astfr.NewChild(views.KiT_TreeView, "ast-tree").(*views.TreeView)
		astt.SetRootNode(&fs.Ast)

		pv.TestBuf.SetHiStyle(core.Settings.Colors.HiStyle)
		pv.TestBuf.Hi.Off = true // prevent auto-hi

		pv.OutputBuffer.SetHiStyle(core.Settings.Colors.HiStyle)
		pv.OutputBuffer.Opts.LineNos = false

		fs.ParseState.Trace.Init()
		fs.ParseState.Trace.PipeOut()
		go pv.MonitorOut()

		pv.LexBuf.SetHiStyle(core.Settings.Colors.HiStyle)
		pv.ParseBuf.SetHiStyle(core.Settings.Colors.HiStyle)

		split.SetSplits(.15, .15, .2, .15, .35)
		split.UpdateEnd(updt)

		pv.OpenConsoleTab()
		pv.OpenTestTextTab()
		pv.OpenOutTab()
		pv.OpenLexTab()
		pv.OpenParseTab()

	} else {
		pv.LexTree().SetRootNode(&pv.Parser.Lexer)
		pv.LexTree().Open()
		pv.ParseTree().SetRootNode(&pv.Parser.Parser)
		pv.ParseTree().Open()
		pv.AstTree().SetRootNode(&fs.Ast)
		pv.AstTree().Open()
		pv.StructView().SetStruct(&pv.Parser.Lexer)
	}

	pv.LexTree().TreeViewSig.Connect(pv.This(), func(recv, send tree.Node, sig int64, data any) {
		if data == nil {
			return
		}
		tvn, _ := data.(tree.Node).Embed(views.KiT_TreeView).(*views.TreeView)
		pvb, _ := recv.Embed(KiT_PiView).(*PiView)
		switch sig {
		case int64(views.TreeViewSelected):
			pvb.ViewNode(tvn)
		case int64(views.TreeViewChanged):
			pvb.SetChanged()
		}
	})

	pv.ParseTree().TreeViewSig.Connect(pv.This(), func(recv, send tree.Node, sig int64, data any) {
		if data == nil {
			return
		}
		tvn, _ := data.(tree.Node).Embed(views.KiT_TreeView).(*views.TreeView)
		pvb, _ := recv.Embed(KiT_PiView).(*PiView)
		switch sig {
		case int64(views.TreeViewSelected):
			pvb.ViewNode(tvn)
		case int64(views.TreeViewChanged):
			pvb.SetChanged()
		}
	})

	pv.AstTree().TreeViewSig.Connect(pv.This(), func(recv, send tree.Node, sig int64, data any) {
		if data == nil {
			return
		}
		tvn, _ := data.(tree.Node).Embed(views.KiT_TreeView).(*views.TreeView)
		pvb, _ := recv.Embed(KiT_PiView).(*PiView)
		switch sig {
		case int64(views.TreeViewSelected):
			pvb.ViewNode(tvn)
		case int64(views.TreeViewChanged):
			pvb.SetChanged()
		}
	})

}

// ViewNode sets the StructView view to src node for given treeview
func (pv *PiView) ViewNode(tv *views.TreeView) {
	sv := pv.StructView()
	if sv != nil {
		sv.SetStruct(tv.SrcNode)
	}
}

func (pv *PiView) SetChanged() {
	pv.Changed = true
	pv.Toolbar().UpdateActions() // nil safe
}

func (pv *PiView) FileNodeOpened(fn *filetree.Node) {
}

func (pv *PiView) FileNodeClosed(fn *filetree.Node) {
}

func (ge *PiView) PiViewKeys(kt *key.ChordEvent) {
	var kf code.KeyFuns
	kc := kt.Chord()
	if core.DebugSettings.KeyEventTrace {
		fmt.Printf("PiView KeyInput: %v\n", ge.Path())
	}
	// gkf := keyfun.(kc)
	if ge.KeySeq1 != "" {
		kf = code.KeyFun(ge.KeySeq1, kc)
		seqstr := string(ge.KeySeq1) + " " + string(kc)
		if kf == code.KeyFunNil || kc == "Escape" {
			if core.DebugSettings.KeyEventTrace {
				fmt.Printf("code.KeyFun sequence: %v aborted\n", seqstr)
			}
			ge.SetStatus(seqstr + " -- aborted")
			kt.SetProcessed() // abort key sequence, don't send esc to anyone else
			ge.KeySeq1 = ""
			return
		}
		ge.SetStatus(seqstr)
		ge.KeySeq1 = ""
		// gkf = keyfun.Nil // override!
	} else {
		kf = code.KeyFun(kc, "")
		if kf == code.KeyFunNeeds2 {
			kt.SetProcessed()
			ge.KeySeq1 = kt.Chord()
			ge.SetStatus(string(ge.KeySeq1))
			if core.DebugSettings.KeyEventTrace {
				fmt.Printf("code.KeyFun sequence needs 2 after: %v\n", ge.KeySeq1)
			}
			return
		} else if kf != code.KeyFunNil {
			if core.DebugSettings.KeyEventTrace {
				fmt.Printf("code.KeyFun got in one: %v = %v\n", ge.KeySeq1, kf)
			}
			// gkf = keyfun.Nil // override!
		}
	}

	// switch gkf {
	// case keyfun.Find:
	// 	kt.SetProcessed()
	// 	tv := ge.ActiveTextEditor()
	// 	if tv.HasSelection() {
	// 		ge.Settings.Find.Find = string(tv.Selection().ToBytes())
	// 	}
	// 	views.CallMethod(ge, "Find", ge.Viewport)
	// }
	// if kt.IsProcessed() {
	// 	return
	// }
	switch kf {
	case code.KeyFunNextPanel:
		kt.SetProcessed()
		ge.FocusNextPanel()
	case code.KeyFunPrevPanel:
		kt.SetProcessed()
		ge.FocusPrevPanel()
	case code.KeyFunFileOpen:
		kt.SetProcessed()
		views.CallMethod(ge, "OpenTest", ge.Viewport)
	// case code.KeyFunBufSelect:
	// 	kt.SetProcessed()
	// 	ge.SelectOpenNode()
	// case code.KeyFunBufClone:
	// 	kt.SetProcessed()
	// 	ge.CloneActiveView()
	case code.KeyFunBufSave:
		kt.SetProcessed()
		views.CallMethod(ge, "SaveTestAs", ge.Viewport)
	case code.KeyFunBufSaveAs:
		kt.SetProcessed()
		views.CallMethod(ge, "SaveActiveViewAs", ge.Viewport)
		// case code.KeyFunBufClose:
		// 	kt.SetProcessed()
		// 	ge.CloseActiveView()
		// case code.KeyFunExecCmd:
		// 	kt.SetProcessed()
		// 	views.CallMethod(ge, "ExecCmd", ge.Viewport)
		// case code.KeyFunCommentOut:
		// 	kt.SetProcessed()
		// 	ge.CommentOut()
		// case code.KeyFunIndent:
		// 	kt.SetProcessed()
		// 	ge.Indent()
		// case code.KeyFunSetSplit:
		// 	kt.SetProcessed()
		// 	views.CallMethod(ge, "SplitsSetView", ge.Viewport)
		// case code.KeyFunBuildProj:
		// 	kt.SetProcessed()
		// 	ge.Build()
		// case code.KeyFunRunProj:
		// 	kt.SetProcessed()
		// 	ge.Run()
	}
}

func (ge *PiView) KeyChordEvent() {
	// need hipri to prevent 2-seq guys from being captured by others
	ge.ConnectEvent(events.KeyChordEvent, core.HiPri, func(recv, send tree.Node, sig int64, d any) {
		gee := recv.Embed(KiT_PiView).(*PiView)
		kt := d.(*key.ChordEvent)
		gee.PiViewKeys(kt)
	})
}

func (ge *PiView) ConnectEvents2D() {
	if ge.HasAnyScroll() {
		ge.LayoutScrollEvents()
	}
	ge.KeyChordEvent()
}

func (pv *PiView) Render2D() {
	if len(pv.Kids) > 0 {
		pv.Toolbar().UpdateActions()
		if win := pv.ParentWindow(); win != nil {
			if !win.IsResizing() {
				win.MainMenuUpdateActives()
			}
		}
	}
	pv.Frame.Render2D()
}

var PiViewProperties = tree.Properties{
	"EnumType:Flag":    core.KiT_NodeFlags,
	"background-color": &core.Settings.Colors.Background,
	"color":            &core.Settings.Colors.Font,
	"max-width":        -1,
	"max-height":       -1,
	"#title": tree.Properties{
		"max-width":        -1,
		"horizontal-align": styles.AlignCenter,
		"vertical-align":   styles.AlignTop,
	},
	"Toolbar": tree.Propertieslice{
		{"SaveProj", tree.Properties{
			"shortcut": keyfun.MenuSave,
			"label":    "Save Project",
			"desc":     "Save GoPi project file to standard JSON-formatted file",
			"updtfunc": views.ActionUpdateFunc(func(pvi any, act *core.Button) {
				pv := pvi.(*PiView)
				act.SetActiveState( pv.Changed && pv.Settings.ProjectFile != "")
			}),
		}},
		{"sep-parse", tree.BlankProp{}},
		{"OpenParser", tree.Properties{
			"label": "Open Parser...",
			"icon":  "file-open",
			"desc":  "Open lexer and parser rules from standard JSON-formatted file",
			"Args": tree.Propertieslice{
				{"File Name", tree.Properties{
					"default-field": "Settings.ParserFile",
					"ext":           ".parse",
				}},
			},
		}},
		{"SaveParser", tree.Properties{
			"icon": "file-save",
			"desc": "Save lexer and parser rules from file standard JSON-formatted file",
			"updtfunc": views.ActionUpdateFunc(func(pvi any, act *core.Button) {
				pv := pvi.(*PiView)
				act.SetActiveStateUpdate( pv.Changed && pv.Settings.ParserFile != "")
			}),
		}},
		{"SaveParserAs", tree.Properties{
			"label": "Save Parser As...",
			"icon":  "file-save",
			"desc":  "Save As lexer and parser rules from file standard JSON-formatted file",
			"Args": tree.Propertieslice{
				{"File Name", tree.Properties{
					"default-field": "Settings.ParserFile",
					"ext":           ".parse",
				}},
			},
		}},
		{"sep-file", tree.BlankProp{}},
		{"OpenTest", tree.Properties{
			"label": "Open Test",
			"icon":  "file-open",
			"desc":  "Open test file",
			"Args": tree.Propertieslice{
				{"File Name", tree.Properties{
					"default-field": "Settings.TestFile",
				}},
			},
		}},
		{"SaveTestAs", tree.Properties{
			"label": "Save Test As",
			"icon":  "file-save",
			"desc":  "Save current test file as",
			"Args": tree.Propertieslice{
				{"File Name", tree.Properties{
					"default-field": "Settings.TestFile",
				}},
			},
		}},
		{"sep-lex", tree.BlankProp{}},
		{"LexInit", tree.Properties{
			"icon": "update",
			"desc": "Init / restart lexer",
		}},
		{"LexNext", tree.Properties{
			"icon": "play",
			"desc": "do next single step of lexing",
		}},
		{"LexNextLine", tree.Properties{
			"icon": "play",
			"desc": "do next line of lexing",
		}},
		{"LexAll", tree.Properties{
			"icon": "fast-fwd",
			"desc": "do all remaining lexing",
		}},
		{"sep-passtwo", tree.BlankProp{}},
		{"EditPassTwo", tree.Properties{
			"icon": "edit",
			"desc": "edit the settings of the PassTwo -- second pass after lexing",
		}},
		{"PassTwo", tree.Properties{
			"icon": "play",
			"desc": "perform second pass after lexing -- computes nesting depth globally and finds EOS tokens",
		}},
		{"sep-parse", tree.BlankProp{}},
		{"EditTrace", tree.Properties{
			"icon": "edit",
			"desc": "edit the parse tracing options for seeing how the parsing process is working",
		}},
		{"ParseInit", tree.Properties{
			"icon": "update",
			"desc": "initialize parser -- this also performs lexing, PassTwo, assuming that is all working",
		}},
		{"ParseNext", tree.Properties{
			"icon": "play",
			"desc": "do next step of parsing",
		}},
		{"ParseAll", tree.Properties{
			"icon": "fast-fwd",
			"desc": "do remaining parsing",
		}},
		{"ViewParseState", tree.Properties{
			"icon": "edit",
			"desc": "view the parser state, including symbols recorded etc",
		}},
	},
	"MainMenu": tree.Propertieslice{
		{"AppMenu", tree.BlankProp{}},
		{"File", tree.Propertieslice{
			{"OpenRecent", tree.Properties{
				"submenu": &SavedPaths,
				"Args": tree.Propertieslice{
					{"File Name", tree.Properties{}},
				},
			}},
			{"OpenProj", tree.Properties{
				"shortcut": keyfun.MenuOpen,
				"label":    "Open Project...",
				"desc":     "open a GoPi project that has full settings",
				"Args": tree.Propertieslice{
					{"File Name", tree.Properties{
						"default-field": "Settings.ProjectFile",
						"ext":           ".parseproject",
					}},
				},
			}},
			{"NewProj", tree.Properties{
				"shortcut": keyfun.MenuNew,
				"label":    "New Project...",
				"desc":     "create a new project",
			}},
			{"SaveProj", tree.Properties{
				"shortcut": keyfun.MenuSave,
				"label":    "Save Project",
				"desc":     "Save GoPi project file to standard JSON-formatted file",
				"updtfunc": views.ActionUpdateFunc(func(pvi any, act *core.Button) {
					pv := pvi.(*PiView)
					act.SetActiveState( pv.Changed && pv.Settings.ProjectFile != "")
				}),
			}},
			{"SaveProjAs", tree.Properties{
				"shortcut": keyfun.MenuSaveAs,
				"label":    "Save Project As...",
				"desc":     "Save GoPi project to file standard JSON-formatted file",
				"Args": tree.Propertieslice{
					{"File Name", tree.Properties{
						"default-field": "Settings.ProjectFile",
						"ext":           ".parseproject",
					}},
				},
			}},
			{"sep-parse", tree.BlankProp{}},
			{"OpenParser", tree.Properties{
				"shortcut": keyfun.MenuOpenAlt1,
				"label":    "Open Parser...",
				"desc":     "Open lexer and parser rules from standard JSON-formatted file",
				"Args": tree.Propertieslice{
					{"File Name", tree.Properties{
						"default-field": "Settings.ParserFile",
						"ext":           ".parse",
					}},
				},
			}},
			{"SaveParser", tree.Properties{
				"shortcut": keyfun.MenuSaveAlt,
				"desc":     "Save lexer and parser rules to file standard JSON-formatted file",
				"updtfunc": views.ActionUpdateFunc(func(pvi any, act *core.Button) {
					pv := pvi.(*PiView)
					act.SetActiveState( pv.Changed && pv.Settings.ParserFile != "")
				}),
			}},
			{"SaveParserAs", tree.Properties{
				"label": "Save Parser As...",
				"desc":  "Save As lexer and parser rules to file standard JSON-formatted file",
				"Args": tree.Propertieslice{
					{"File Name", tree.Properties{
						"default-field": "Settings.ParserFile",
						"ext":           ".parse",
					}},
				},
			}},
			{"sep-close", tree.BlankProp{}},
			{"Close Window", tree.BlankProp{}},
			{"OpenConsoleTab", tree.Properties{}},
		}},
		{"Edit", "Copy Cut Paste"},
		{"Window", "Windows"},
	},
}

//////////////////////////////////////////////////////////////////////////////////////
//   Project window

// CloseWindowReq is called when user tries to close window -- we
// automatically save the project if it already exists (no harm), and prompt
// to save open files -- if this returns true, then it is OK to close --
// otherwise not
func (pv *PiView) CloseWindowReq() bool {
	if !pv.Changed {
		return true
	}
	core.ChoiceDialog(pv.Viewport, core.DlgOpts{Title: "Close Project: There are Unsaved Changes",
		Prompt: fmt.Sprintf("In Project: %v There are <b>unsaved changes</b> -- do you want to save or cancel closing this project and review?", pv.Nm)},
		[]string{"Cancel", "Save Proj", "Close Without Saving"},
		pv.This(), func(recv, send tree.Node, sig int64, data any) {
			switch sig {
			case 0:
				// do nothing, will have returned false already
			case 1:
				pv.SaveProj()
			case 2:
				pv.ParentWindow().OSWin.Close() // will not be prompted again!
			}
		})
	return false // not yet
}

// QuitReq is called when user tries to quit the app -- we go through all open
// main windows and look for code windows and call their CloseWindowReq
// functions!
func QuitReq() bool {
	for _, win := range core.MainWindows {
		if !strings.HasPrefix(win.Nm, "Pie") {
			continue
		}
		mfr, err := win.MainWidget()
		if err != nil {
			continue
		}
		gek := mfr.ChildByName("piview", 0)
		if gek == nil {
			continue
		}
		ge := gek.Embed(KiT_PiView).(*PiView)
		if !ge.CloseWindowReq() {
			return false
		}
	}
	return true
}

// NewPiView creates a new PiView window
func NewPiView() (*core.Window, *PiView) {
	winm := "Pie Interactive Parser Editor"

	width := 1600
	height := 1280
	sc := system.TheApp.Screen(0)
	if sc != nil {
		scsz := sc.SceneGeometry.Size()
		width = int(.9 * float64(scsz.X))
		height = int(.8 * float64(scsz.Y))
	}

	win := core.NewMainWindow(winm, winm, width, height)

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()

	pv := mfr.NewChild(KiT_PiView, "piview").(*PiView)
	pv.Viewport = vp

	// mmen := win.MainMenu
	// views.MainMenuView(pv, win, mmen)
	//
	// inClosePrompt := false
	// win.OSWin.SetCloseReqFunc(func(w oswin.Window) {
	// 	if !inClosePrompt {
	// 		inClosePrompt = true
	// 		if pv.Changed {
	// 			core.ChoiceDialog(vp, core.DlgOpts{Title: "Close Without Saving?",
	// 				Prompt: "Do you want to save your changes?  If so, Cancel and then Save"},
	// 				[]string{"Close Without Saving", "Cancel"},
	// 				win.This(), func(recv, send tree.Node, sig int64, data any) {
	// 					switch sig {
	// 					case 0:
	// 						w.Close()
	// 					case 1:
	// 						// default is to do nothing, i.e., cancel
	// 					}
	// 				})
	// 		} else {
	// 			w.Close()
	// 		}
	// 	}
	// })
	//
	// inQuitPrompt := false
	// system.TheApp.SetQuitReqFunc(func() {
	// 	if !inQuitPrompt {
	// 		inQuitPrompt = true
	// 		core.PromptDialog(vp, core.DlgOpts{Title: "Really Quit?",
	// 			Prompt: "Are you <i>sure</i> you want to quit?"}, true, true,
	// 			win.This(), func(recv, send tree.Node, sig int64, data any) {
	// 				if sig == int64(core.DialogAccepted) {
	// 					system.TheApp.Quit()
	// 				} else {
	// 					inQuitPrompt = false
	// 				}
	// 			})
	// 	}
	// })

	// win.OSWin.SetCloseCleanFunc(func(w oswin.Window) {
	// 	fmt.Printf("Doing final Close cleanup here..\n")
	// })

	// win.OSWin.SetCloseCleanFunc(func(w oswin.Window) {
	// 	if core.MainWindows.Len() <= 1 {
	// 		go system.TheApp.Quit() // once main window is closed, quit
	// 	}
	// })

	// win.MainMenuUpdated()

	pv.Config()

	vp.UpdateEndNoSig(updt)

	win.GoStartEventLoop()
	return win, pv
}

*/
