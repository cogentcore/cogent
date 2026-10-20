// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/parse/complete"
	"cogentcore.org/core/text/parse/lexer"
	"cogentcore.org/core/text/parse/syms"
	"cogentcore.org/core/text/textcore"
	"cogentcore.org/core/text/textpos"
	"cogentcore.org/core/text/token"

	"cogentcore.org/cogent/code/lang"
	"cogentcore.org/cogent/code/lsp"
)

// LangService manages the language service backends for a Code window,
// routing editor actions to either the [lsp.Backend] (external language
// servers such as gopls) or the [lang.ParseBackend] (built-in parser),
// according to the [ProjectSettings.LangBackend] setting.
//
// It syncs open files with the servers, applies diagnostics as buffer
// tags, provides completions, hover docs, definitions, references, and
// symbols to the editor and panels.
type LangService struct {

	// cv is the Code window this service serves.
	cv *Code

	// lspB is the language server backend, which is always present,
	// but only used when supported for a given file and enabled
	// in settings.
	lspB *lsp.Backend

	mu        sync.Mutex
	debounce  map[string]*time.Timer // filename -> pending didChange
	prevDiags map[string][]tagPos    // filename -> currently tagged regions

	// diagStatus is a short status-bar summary of diagnostics per file,
	// used by [Code.UpdateStatusText] when generating the official status
	// line, which it regenerates from scratch on every cursor movement.
	diagStatus map[string]string

	hover hoverCache

	quit chan struct{}
}

// tagPos records one diagnostic tag applied to a buffer,
// for later removal, and for showing the message on hover.
type tagPos struct {
	line int
	st   int
	ed   int
	msg  string
	sev  string // "error", "warning", etc
}

// hoverCache is a simple single-entry cache for hover docs,
// which avoids re-querying the server for the same word.
type hoverCache struct {
	valid bool
	time  time.Time
	key   hoverKey
	text  string
}

type hoverKey struct {
	fname string
	line  int
	char  int
}

// hoverCacheTime is how long cached hover docs remain valid.
const hoverCacheTime = 10 * time.Second

// changeDelay is how long to wait after the last keystroke before
// syncing changes to the language server.
const changeDelay = 250 * time.Millisecond

// NewLangService returns a new language service for given Code window.
func NewLangService(cv *Code) *LangService {
	ls := &LangService{
		cv:         cv,
		lspB:       lsp.NewBackend(Settings.LSExecs),
		debounce:   make(map[string]*time.Timer),
		prevDiags:  make(map[string][]tagPos),
		diagStatus: make(map[string]string),
		quit:       make(chan struct{}),
	}
	ls.lspB.RootFunc = func(fpath string) string {
		root := string(cv.Settings.ProjectRoot)
		if root != "" && strings.HasPrefix(fpath, root) {
			return root
		}
		return ""
	}
	go ls.consumeDiags()
	return ls
}

// LSP returns the language server backend, for registry access etc.
func (ls *LangService) LSP() *lsp.Backend { return ls.lspB }

// LSPActive returns whether the language server backend is being used
// for the given buffer: the LSP backend must be selected in settings and
// a server must be registered for the file's language. Otherwise, the
// built-in parse system provides whatever features it can through the
// standard editor machinery.
func (ls *LangService) LSPActive(ln *lines.Lines) bool {
	return ls.cv.Settings.LangBackend == LangBackendLSP && ls.lspB.Supported(ln)
}

// Stop halts all language servers and ends the diagnostics loop.
func (ls *LangService) Stop() {
	select {
	case <-ls.quit:
	default:
		close(ls.quit)
	}
	ls.mu.Lock()
	for _, t := range ls.debounce {
		t.Stop()
	}
	ls.debounce = make(map[string]*time.Timer)
	ls.mu.Unlock()
	ls.lspB.Stop()
}

////////   File lifecycle

// Activated should be called when a texteditor becomes active,
// opening its file with the backend and configuring completion
// functions to use the active backend.
func (ls *LangService) Activated(tv *TextEditor) {
	ln := tv.Lines
	if ln == nil {
		return
	}
	if !ls.LSPActive(ln) {
		return // textcore.UpdateNewFile already set up parse completers
	}
	ls.lspB.Open(ln)
	if tv.Complete != nil {
		tv.Complete.MatchFunc = func(data any, text string, posLine, posChar int) complete.Matches {
			return ls.lspB.Completions(ln, text, posLine, posChar)
		}
		tv.Complete.EditFunc = func(data any, text string, cursorPos int, comp complete.Completion, seed string) complete.Edit {
			return ls.lspB.CompleteEdit(ln, text, cursorPos, comp, seed)
		}
		tv.Complete.LookupFunc = ls.lookupFun(ln)
	}
}

// lookupFun returns a LookupFunc that resolves the symbol at the given
// position using the language server: showing hover docs if the symbol
// is defined in the current file, and opening the defining file otherwise.
func (ls *LangService) lookupFun(ln *lines.Lines) complete.LookupFunc {
	return func(data any, txt string, posLine, posChar int) complete.Lookup {
		var ld complete.Lookup
		pos := textpos.Pos{Line: posLine, Char: posChar}
		locs := ls.lspB.Definition(ln, pos)
		if len(locs) > 0 {
			lc := locs[0]
			if lc.Filename == ln.Filename() {
				// same file: show hover docs about the symbol
				if docs := ls.lspB.Hover(ln, pos); docs != "" {
					textcore.TextDialog(nil, "Lookup: "+txt, docs)
					ld.Text = []byte(docs)
					return ld
				}
				return ld
			}
			ld.Filename = lc.Filename
			ld.StLine = lc.Region.Start.Line
			ld.EdLine = lc.Region.End.Line
			_, _ = ls.cv.OpenFileAtRegion(ld.Filename, lc.Region)
			return ld
		}
		if docs := ls.lspB.Hover(ln, pos); docs != "" {
			textcore.TextDialog(nil, "Lookup: "+txt, docs)
			ld.Text = []byte(docs)
		}
		return ld
	}
}

// Input should be called on editor Input events, to sync changed
// text to the language server with a debounce delay.
func (ls *LangService) Input(ln *lines.Lines) {
	if ln == nil {
		return
	}
	if !ls.LSPActive(ln) {
		return
	}
	fname := ln.Filename()
	if fname == "" {
		return
	}
	ls.mu.Lock()
	t, has := ls.debounce[fname]
	if !has {
		t = time.AfterFunc(changeDelay, func() { ls.Change(ln) })
		ls.debounce[fname] = t
	} else {
		t.Reset(changeDelay)
	}
	ls.mu.Unlock()
}

// Change syncs the current contents of the buffer to the backend now.
func (ls *LangService) Change(ln *lines.Lines) {
	if ln == nil || ln.Filename() == "" {
		return
	}
	ls.mu.Lock()
	if t, has := ls.debounce[ln.Filename()]; has {
		t.Stop()
		delete(ls.debounce, ln.Filename())
	}
	ls.mu.Unlock()
	if ls.LSPActive(ln) {
		ls.lspB.Change(ln)
	}
}

// Saved should be called after a file has been saved.
func (ls *LangService) Saved(ln *lines.Lines) {
	if ln == nil || ln.Filename() == "" {
		return
	}
	ls.Change(ln)
	if ls.LSPActive(ln) {
		ls.lspB.Save(ln)
	}
}

// Closed should be called when a file is closed in the editor.
// Language servers are shut down automatically when the last of
// their files closes.
func (ls *LangService) Closed(ln *lines.Lines) {
	if ln == nil {
		return
	}
	ls.clearDiags(ln)
	ls.lspB.Close(ln)
	ls.mu.Lock()
	if t, has := ls.debounce[ln.Filename()]; has {
		t.Stop()
		delete(ls.debounce, ln.Filename())
	}
	ls.mu.Unlock()
}

// CloseAll closes all open files with the backends, shutting down
// any servers that were serving only this window's files, without
// stopping the service itself, so it can restart servers if files
// are reopened.
func (ls *LangService) CloseAll() {
	for _, ln := range ls.cv.OpenFiles.Values {
		ls.Closed(ln)
	}
}

////////   Diagnostics

// consumeDiags receives diagnostics from the language server backend
// and applies them to the corresponding buffers. Runs in its own
// goroutine until quit.
func (ls *LangService) consumeDiags() {
	for {
		select {
		case <-ls.quit:
			return
		case fd, ok := <-ls.lspB.Diagnostics():
			if !ok {
				return
			}
			ln := ls.cv.GetOpenFile(fd.Filename)
			if ln == nil {
				continue
			}
			// applyDiags only uses self-locking Lines methods,
			// which is safe from this goroutine (same pattern as
			// the async markup in the lines package itself):
			ls.applyDiags(ln, fd.Diags)
			nerr, nwarn := countDiags(fd.Diags)
			summary := diagSummary(ls.lspB.Name(), nerr, nwarn)
			ls.mu.Lock()
			if summary == "" {
				delete(ls.diagStatus, fd.Filename)
			} else {
				ls.diagStatus[fd.Filename] = summary
			}
			ls.mu.Unlock()
			if ls.cv.Scene == nil {
				continue // not attached to a shown window: tagging is done
			}
			ls.cv.AsyncLock()
			tv, _, hasTv := ls.cv.EditorForLines(ln)
			if hasTv && tv != nil {
				tv.NeedsRender()
			}
			// the official status line is generated by UpdateStatusText,
			// which picks up the diagStatus summary for this file:
			ls.cv.UpdateStatusText()
			ls.cv.AsyncUnlock()
		}
	}
}

// diagSummary returns a short status-bar summary of diagnostics,
// or "" if there are no errors or warnings.
func diagSummary(name string, nerr, nwarn int) string {
	var ps []string
	if nerr > 0 {
		ps = append(ps, fmt.Sprintf("%d error%s", nerr, plural(nerr)))
	}
	if nwarn > 0 {
		ps = append(ps, fmt.Sprintf("%d warning%s", nwarn, plural(nwarn)))
	}
	if len(ps) == 0 {
		return ""
	}
	return name + ": " + strings.Join(ps, ", ")
}

func plural(n int) string {
	if n > 1 {
		return "s"
	}
	return ""
}

// DiagStatusMsg returns the current status-bar summary of diagnostics
// for the given buffer, or "" if none. Called from UpdateStatusText
// when regenerating the official status line.
func (ls *LangService) DiagStatusMsg(ln *lines.Lines) string {
	if ln == nil {
		return ""
	}
	ls.mu.Lock()
	defer ls.mu.Unlock()
	return ls.diagStatus[ln.Filename()]
}

func countDiags(diags []lang.Diagnostic) (nerr, nwarn int) {
	for _, d := range diags {
		switch d.Severity {
		case lang.DiagErr:
			nerr++
		case lang.DiagWarn:
			nwarn++
		}
	}
	return
}

// applyDiags replaces the prior diagnostic tags on the buffer with
// new ones. Only uses self-locking Lines methods, safe to call from
// any goroutine (not under AsyncLock or ln.Lock).
func (ls *LangService) applyDiags(ln *lines.Lines, diags []lang.Diagnostic) {
	fname := ln.Filename()
	ls.mu.Lock()
	prev := ls.prevDiags[fname]
	delete(ls.prevDiags, fname)
	ls.mu.Unlock()

	// remove our previous tags: we own the Error tokens in LSP mode
	// (spell-check uses a distinct TextSpellErr token):
	done := make(map[int]bool)
	for _, p := range prev {
		if done[p.line] || !ln.IsValidLine(p.line) {
			continue
		}
		done[p.line] = true
		tags := ln.AdjustedTags(p.line)
		out := make(lexer.Line, 0, len(tags))
		for _, t := range tags {
			if t.Token.Token == token.Error {
				continue
			}
			out = append(out, t)
		}
		ln.SetTags(p.line, out)
		ln.MarkupLines(p.line, p.line)
	}

	cur := make([]tagPos, 0, len(diags))
	for _, d := range diags {
		// info and hints are not tagged in the buffer yet:
		if d.Severity != lang.DiagErr && d.Severity != lang.DiagWarn {
			continue
		}
		lnum := d.Region.Start.Line
		if !ln.IsValidLine(lnum) {
			continue
		}
		st := max(0, d.Region.Start.Char)
		ed := min(ln.LineLen(lnum), max(st+1, d.Region.End.Char))
		ln.AddTag(lnum, st, ed, token.Error)
		cur = append(cur, tagPos{line: lnum, st: st, ed: ed, msg: d.Msg, sev: diagWord(d.Severity)})
	}
	if len(cur) > 0 {
		ls.mu.Lock()
		ls.prevDiags[fname] = cur
		ls.mu.Unlock()
	}
}

// diagWord returns a lowercase word for the severity, for hover text.
func diagWord(s lang.DiagSeverity) string {
	switch s {
	case lang.DiagErr:
		return "error"
	case lang.DiagWarn:
		return "warning"
	case lang.DiagInfo:
		return "info"
	}
	return "hint"
}

// clearDiags removes all diagnostic tags previously applied to buffer.
// Called from the UI goroutine; Lines methods lock internally.
func (ls *LangService) clearDiags(ln *lines.Lines) {
	fname := ln.Filename()
	ls.mu.Lock()
	prev := ls.prevDiags[fname]
	delete(ls.prevDiags, fname)
	delete(ls.diagStatus, fname)
	ls.mu.Unlock()
	done := make(map[int]bool)
	for _, p := range prev {
		if done[p.line] || !ln.IsValidLine(p.line) {
			continue
		}
		done[p.line] = true
		tags := ln.AdjustedTags(p.line)
		out := make(lexer.Line, 0, len(tags))
		for _, t := range tags {
			if t.Token.Token == token.Error {
				continue
			}
			out = append(out, t)
		}
		ln.SetTags(p.line, out)
		ln.MarkupLines(p.line, p.line)
	}
	if tv, _, hasTv := ls.cv.EditorForLines(ln); hasTv && tv != nil {
		tv.NeedsRender()
	}
}

////////   Hover

// HoverAt returns hover documentation for the word at the given
// position, using a short-lived cache. The request runs synchronously
// on the UI goroutine with the server's bounded quick timeout: tooltips
// are generated once, at LongHoverStart, so there is no second request
// that an async result could arrive on. Any diagnostic message tagged
// at that position is included first, from the local record.
func (ls *LangService) HoverAt(tv *TextEditor, pos textpos.Pos) string {
	ln := tv.Lines
	if ln == nil || !ls.LSPActive(ln) {
		return ""
	}
	if !ln.IsValidLine(pos.Line) {
		return ""
	}
	diag := ls.diagAt(ln, pos)
	key := hoverKey{fname: ln.Filename(), line: pos.Line, char: pos.Char}
	ls.mu.Lock()
	hc := &ls.hover
	if hc.valid && hc.key == key && time.Since(hc.time) < hoverCacheTime {
		txt := hc.text
		ls.mu.Unlock()
		return joinHover(diag, txt)
	}
	ls.mu.Unlock()

	txt := ls.lspB.Hover(ln, pos)

	ls.mu.Lock()
	hc = &ls.hover
	hc.key = key
	hc.text = txt
	hc.time = time.Now()
	hc.valid = true
	ls.mu.Unlock()
	return joinHover(diag, txt)
}

// diagAt returns the formatted message of any diagnostic currently
// tagged at the given position, or "" if none. The end is inclusive
// since hover positions are often the word-end position, which matches
// the exclusive end of the diagnostic region.
func (ls *LangService) diagAt(ln *lines.Lines, pos textpos.Pos) string {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	for _, p := range ls.prevDiags[ln.Filename()] {
		if p.line == pos.Line && pos.Char >= p.st && pos.Char <= p.ed {
			if p.sev == "" {
				return p.msg
			}
			return p.sev + ": " + p.msg
		}
	}
	return ""
}

// joinHover combines a diagnostic message with symbol docs.
func joinHover(diag, docs string) string {
	switch {
	case diag == "":
		return docs
	case docs == "":
		return diag
	}
	return diag + "\n\n" + docs
}

////////   Editor commands

// cursorWordPos returns a position on the word at the cursor,
// preferring the end of the word, for definition-type queries.
func cursorWordPos(tv *TextEditor) (textpos.Pos, bool) {
	ln := tv.Lines
	if ln == nil {
		return textpos.Pos{}, false
	}
	pos := tv.CursorPos
	if tv.HasSelection() {
		if tv.SelectRegion.End.Line != tv.SelectRegion.Start.Line {
			return textpos.Pos{}, false
		}
		pos = tv.SelectRegion.End
	} else if !ln.IsWordEnd(pos) {
		w := ln.WordAt(pos)
		if w.Start.Line == pos.Line && w.End.Char > 0 {
			pos = w.End
		}
	}
	if !ln.IsValidLine(pos.Line) {
		return textpos.Pos{}, false
	}
	return pos, true
}

// GotoDefinition jumps to the definition of the symbol at the cursor,
// using the active language backend.
func (cv *Code) GotoDefinition() { //types:add
	tv := cv.ActiveEditor()
	if tv == nil || tv.Lines == nil {
		return
	}
	pos, ok := cursorWordPos(tv)
	if !ok {
		return
	}
	ln := tv.Lines
	if !cv.LangSvc().LSPActive(ln) {
		cv.SetStatus("Goto definition: requires an LSP language server for this file")
		return
	}
	b := cv.LangSvc().lspB
	cv.SetStatus("Goto definition...")
	locs := b.Definition(ln, pos)
	switch len(locs) {
	case 0:
		cv.SetStatus("Goto definition: not found")
	case 1:
		cv.openLoc(locs[0])
	default:
		texts := make([]string, len(locs))
		for i, lc := range locs {
			texts[i] = locString(cv, lc)
		}
		m := core.NewMenuFromStrings(texts, texts[0], func(idx int) {
			cv.openLoc(locs[idx])
		})
		core.NewMenuStage(m, tv, tv.ContextMenuPos(nil)).Run()
	}
}

// FindReferences lists all references to the symbol at the cursor,
// using the active language backend, in a menu for selection.
func (cv *Code) FindReferences() { //types:add
	tv := cv.ActiveEditor()
	if tv == nil || tv.Lines == nil {
		return
	}
	pos, ok := cursorWordPos(tv)
	if !ok {
		return
	}
	ln := tv.Lines
	if !cv.LangSvc().LSPActive(ln) {
		cv.SetStatus("Find references: requires an LSP language server for this file")
		return
	}
	b := cv.LangSvc().lspB
	cv.SetStatus("Finding references...")
	locs := b.References(ln, pos)
	if len(locs) == 0 {
		cv.SetStatus("Find references: none found")
		return
	}
	texts := make([]string, len(locs))
	for i, lc := range locs {
		texts[i] = locString(cv, lc)
	}
	m := core.NewMenuFromStrings(texts, texts[0], func(idx int) {
		cv.openLoc(locs[idx])
	})
	core.NewMenuStage(m, tv, tv.ContextMenuPos(nil)).Run()
}

// LangFormat formats the current file using the active language
// backend, if it supports formatting (e.g., gopls for Go).
func (cv *Code) LangFormat() { //types:add
	tv := cv.ActiveEditor()
	if tv == nil || tv.Lines == nil {
		return
	}
	ln := tv.Lines
	ls := cv.LangSvc()
	if !ls.LSPActive(ln) {
		cv.SetStatus("Format: no formatting backend for this file -- try the goimports command")
		return
	}
	ls.Change(ln)
	// run synchronously on the UI goroutine, as for GotoDefinition and
	// FindReferences: b.Format applies edits with self-locking Lines
	// methods, and the server round trip is bounded by its own timeout,
	// so we never block the render loop under AsyncLock to do this.
	ok := ls.lspB.Format(ln)
	if ok {
		ls.Change(ln) // keep server in sync with new text
		if ed, _, has := cv.EditorForLines(ln); has && ed != nil {
			ed.NeedsRender()
		}
		cv.SetStatus("Formatted: " + filepath.Base(ln.Filename()))
	} else {
		cv.SetStatus("Format: no edits (or not supported yet by server)")
	}
}

// openLoc opens a file at the given location, selecting the region.
func (cv *Code) openLoc(lc lang.Location) {
	_, ok := cv.OpenFileAtRegion(lc.Filename, lc.Region)
	if !ok {
		cv.SetStatus("Could not open file: " + lc.Filename)
	}
}

// locString returns a display string for a location,
// with file:line plus a trimmed preview of the line.
func locString(cv *Code, lc lang.Location) string {
	base := filepath.Base(lc.Filename)
	rel := strings.TrimPrefix(lc.Filename, string(cv.Settings.ProjectRoot)+"/")
	nm := rel
	if len(rel) > len(base) && strings.Count(rel, "/") > 2 {
		nm = ".../" + filepath.Base(filepath.Dir(rel)) + "/" + base
	} else if rel == lc.Filename {
		nm = lc.Filename
	}
	txt := ""
	if ln := cv.GetOpenFile(lc.Filename); ln != nil && ln.IsValidLine(lc.Region.Start.Line) {
		txt = strings.TrimSpace(string(ln.Line(lc.Region.Start.Line)))
		if len(txt) > 60 {
			txt = txt[:60] + "…"
		}
	}
	if txt != "" {
		return fmt.Sprintf("%s:%d: %s", nm, lc.Region.Start.Line+1, txt)
	}
	return fmt.Sprintf("%s:%d", nm, lc.Region.Start.Line+1)
}

// WorkspaceSymbolPrompt prompts for a symbol name and shows matching
// symbols from the entire workspace project, using the language server.
func (cv *Code) WorkspaceSymbolPrompt() { //types:add
	tv := cv.ActiveEditor()
	if tv == nil || tv.Lines == nil {
		return
	}
	ln := tv.Lines
	if !cv.LangSvc().LSPActive(ln) {
		cv.SetStatus("Workspace symbols: requires an LSP language server for this file")
		return
	}
	b := cv.LangSvc().lspB
	d := core.NewBody("Workspace Symbols")
	lst := core.NewChooser(d)
	tf := core.NewTextField(d).SetPlaceholder("symbol name to search")
	search := func() {
		symsL := b.WorkspaceSymbols(ln, tf.Text())
		lst.SetItems(langSymItems(cv, symsL)...)
	}
	d.AddBottomBar(func(bar *core.Frame) {
		core.NewButton(bar).SetText("Search").OnClick(func(e events.Event) {
			search()
		})
	})
	d.RunWindowDialog(tv)
}

func langSymItems(cv *Code, sl []lang.Symbol) []core.ChooserItem {
	items := make([]core.ChooserItem, 0, len(sl))
	for _, sy := range sl {
		sy2 := sy
		txt := sy2.Name
		if sy2.Detail != "" {
			txt += "  [" + sy2.Detail + "]"
		}
		items = append(items, core.ChooserItem{
			Text: txt,
			Icon: sy2.Kind.Icon(),
			Func: func() {
				SelectSymbol(cv, LangSymbolToSyms(sy2))
			},
		})
	}
	return items
}

// LangSymbolToSyms converts a [lang.Symbol] to the [syms.Symbol]
// representation used by the symbols panels, recursively.
func LangSymbolToSyms(sy lang.Symbol) syms.Symbol {
	s := syms.Symbol{
		Name:      sy.Name,
		Detail:    sy.Detail,
		Kind:      sy.Kind,
		Filename:  sy.Filename,
		Region:    sy.Region,
		SelectReg: sy.Region,
	}
	for _, ch := range sy.Children {
		c := LangSymbolToSyms(ch)
		s.Children.Add(&c)
	}
	return s
}

// LangSvc returns the language service for this Code window,
// creating it if needed.
func (cv *Code) LangSvc() *LangService {
	if cv.langSvc == nil {
		cv.langSvc = NewLangService(cv)
	}
	return cv.langSvc
}
