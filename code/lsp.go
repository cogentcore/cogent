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
	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/parse/complete"
	"cogentcore.org/core/text/parse/lexer"
	"cogentcore.org/core/text/textpos"
	"cogentcore.org/core/text/token"

	"cogentcore.org/cogent/code/lang"
	"cogentcore.org/cogent/code/lsp"
)

// LangService manages the language services for a Code window, routing
// editor actions to the [lsp.Backend] (external language servers such
// as gopls) when the [ProjectSettings.LangBackend] setting selects it
// and a server is available for a file, and otherwise leaving the
// built-in parse system to provide what it can through the standard
// editor machinery.
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
	semantic  map[string]*time.Timer // filename -> pending semantic refresh
	prevDiags map[string][]tagPos    // filename -> currently tagged regions

	// diagStatus is a short status-bar summary of diagnostics per file,
	// used by [Code.UpdateStatusText] when generating the official status
	// line, which it regenerates from scratch on every cursor movement.
	diagStatus map[string]string

	// diagCache is the full diagnostic list per file, from the latest
	// publish, used as code action context for quick fixes.
	diagCache map[string][]lang.Diagnostic

	hover hoverCache

	// semanticTagged records which buffers currently carry semantic
	// token highlighting tags.
	semanticTagged map[string]bool

	// semGen is bumped whenever a semantic tokens pass is superseded
	// (new pass started, file changed or closed), so in-flight async
	// passes can see that their results are stale.
	semGen map[string]int

	// sigHelp is the single-entry signature help cache for the typing
	// hot path, and sigStatus is the last signature shown in the status
	// bar, so it can be cleared when the parameter context is exited.
	sigHelp   sigHelpCache
	sigStatus string

	// sigPending is set while a deferred SigHelpUpdate is scheduled.
	sigPending bool

	// sigStage is the live signature help popup, anchored near the
	// cursor like the completion popup but display-only. Any existing
	// popup is closed before a new one is shown, so exactly one can
	// exist at a time.
	sigStage *core.Stage

	quit     chan struct{}
	quitOnce sync.Once
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
		cv:             cv,
		lspB:           lsp.NewBackend(Settings.LSExecs),
		debounce:       make(map[string]*time.Timer),
		semantic:       make(map[string]*time.Timer),
		prevDiags:      make(map[string][]tagPos),
		diagStatus:     make(map[string]string),
		diagCache:      make(map[string][]lang.Diagnostic),
		semanticTagged: make(map[string]bool),
		semGen:         make(map[string]int),
		quit:           make(chan struct{}),
	}
	ls.lspB.SetOptions(serverOptions(cv.Settings.InlayHints))
	ls.lspB.RootFunc = func(fpath string) string {
		root := filepath.Clean(string(cv.Settings.ProjectRoot))
		if root != "." && strings.HasPrefix(fpath, root+string(filepath.Separator)) {
			return root
		}
		return ""
	}
	go ls.consumeEvents()
	return ls
}

// LangSvc returns the language service for this Code window,
// creating it if needed.
func (cv *Code) LangSvc() *LangService {
	if cv.langSvc == nil {
		cv.langSvc = NewLangService(cv)
	}
	return cv.langSvc
}

// LSPActive returns whether the language server backend is being used
// for the given buffer: the LSP backend must be selected in settings and
// a server must be registered for the file's language. Otherwise, the
// built-in parse system provides whatever features it can through the
// standard editor machinery.
func (ls *LangService) LSPActive(ln *lines.Lines) bool {
	return ls.cv.Settings.LangBackend == LangBackendLSP && ls.lspB.Supported(ln)
}

// Stop halts all language servers and ends the event loop.
// The service cannot be used after this.
func (ls *LangService) Stop() {
	ls.quitOnce.Do(func() { close(ls.quit) })
	ls.mu.Lock()
	for _, t := range ls.debounce {
		t.Stop()
	}
	clear(ls.debounce)
	for _, t := range ls.semantic {
		t.Stop()
	}
	clear(ls.semantic)
	ls.mu.Unlock()
	ls.clearSigHelp()
	ls.lspB.Stop()
}

////////   File lifecycle

// Activated should be called when a texteditor becomes active,
// opening its file with the backend and configuring completion
// functions to use the active backend.
func (ls *LangService) Activated(tv *TextEditor) {
	ln := tv.Lines
	if ln == nil || !ls.LSPActive(ln) {
		return // textcore.UpdateNewFile already set up parse completers
	}
	ls.lspB.Open(ln)
	ls.SemanticRefresh(ln)
	if tv.Complete != nil {
		tv.Complete.MatchFunc = func(data any, text string, posLine, posChar int) complete.Matches {
			return ls.lspB.Completions(ln, text, posLine, posChar)
		}
		tv.Complete.EditFunc = func(data any, text string, cursorPos int, comp complete.Completion, seed string) complete.Edit {
			return ls.lspB.CompleteEdit(ln, text, cursorPos, comp, seed)
		}
		tv.Complete.LookupFunc = ls.lookupFunc(ln)
	}
}

// Input should be called on editor Input events, to sync changed
// text to the language server with a debounce delay.
func (ls *LangService) Input(ln *lines.Lines) {
	if ln == nil || ln.Filename() == "" || !ls.LSPActive(ln) {
		return
	}
	ls.mu.Lock()
	resetTimer(ls.debounce, ln.Filename(), changeDelay, func() { ls.Change(ln) })
	ls.mu.Unlock()
}

// resetTimer starts or resets the timer for the given key in the map,
// calling fn after the delay. Must be called under ls.mu.
func resetTimer(timers map[string]*time.Timer, key string, delay time.Duration, fn func()) {
	if t, has := timers[key]; has {
		t.Reset(delay)
		return
	}
	timers[key] = time.AfterFunc(delay, fn)
}

// stopTimer stops and removes the timer for the given key in the map.
// Must be called under ls.mu.
func stopTimer(timers map[string]*time.Timer, key string) {
	if t, has := timers[key]; has {
		t.Stop()
		delete(timers, key)
	}
}

// Change syncs the current contents of the buffer to the backend now.
func (ls *LangService) Change(ln *lines.Lines) {
	if ln == nil || ln.Filename() == "" {
		return
	}
	ls.mu.Lock()
	stopTimer(ls.debounce, ln.Filename())
	ls.mu.Unlock()
	if !ls.LSPActive(ln) {
		return
	}
	ls.lspB.Change(ln)
	// semantic token regions are stale once the text changes:
	ls.bumpSemantic(ln.Filename())
	ls.scheduleSemantic(ln)
}

// Saved should be called after a file has been saved.
func (ls *LangService) Saved(ln *lines.Lines) {
	if ln == nil || ln.Filename() == "" || !ls.LSPActive(ln) {
		return
	}
	ls.Change(ln)
	ls.lspB.Save(ln)
}

// Closed should be called when a file is closed in the editor.
// Language servers are shut down automatically when the last of
// their files closes.
func (ls *LangService) Closed(ln *lines.Lines) {
	if ln == nil {
		return
	}
	fname := ln.Filename()
	ls.clearSigHelp()
	ls.clearDiags(ln)
	ls.bumpSemantic(fname)
	ls.clearSemanticTags(ln)
	ls.lspB.Close(ln)
	ls.mu.Lock()
	stopTimer(ls.debounce, fname)
	stopTimer(ls.semantic, fname)
	delete(ls.semGen, fname)
	ls.mu.Unlock()
}

////////   Diagnostics

// consumeEvents receives diagnostics and applied edit notifications
// from the language server backend, and applies them to the
// corresponding buffers and views. Runs in its own goroutine until quit.
func (ls *LangService) consumeEvents() {
	for {
		select {
		case <-ls.quit:
			return
		case fe, ok := <-ls.lspB.AppliedEdits():
			if !ok {
				return
			}
			// edits are already applied to the buffers and files by the
			// backend; here we only refresh the affected views:
			ls.cv.AsyncLock()
			ls.renderEdited(fe)
			ls.cv.AsyncUnlock()
		case fd, ok := <-ls.lspB.Diagnostics():
			if !ok {
				return
			}
			ls.receiveDiags(fd)
		}
	}
}

// renderEdited triggers rendering of the editors showing the
// given edited files.
func (ls *LangService) renderEdited(fe []lang.FileEdits) {
	if ls.cv.Scene == nil {
		return
	}
	for _, f := range fe {
		if ln := ls.cv.GetOpenFile(f.Filename); ln != nil {
			ls.needsRender(ln)
		}
	}
}

// needsRender triggers rendering of the editor showing the given buffer.
func (ls *LangService) needsRender(ln *lines.Lines) {
	if tv, _, has := ls.cv.EditorForLines(ln); has && tv != nil {
		tv.NeedsRender()
	}
}

// receiveDiags applies newly published diagnostics for a file.
// Called from the consumeEvents goroutine.
func (ls *LangService) receiveDiags(fd lang.FileDiags) {
	ln := ls.cv.GetOpenFile(fd.Filename)
	if ln == nil {
		return
	}
	// applyDiags only uses self-locking Lines methods, which is safe from
	// this goroutine (same pattern as the async markup in lines itself):
	ls.applyDiags(ln, fd.Diags)
	summary := diagSummary(fd.Diags)
	ls.mu.Lock()
	if summary == "" {
		delete(ls.diagStatus, fd.Filename)
		delete(ls.diagCache, fd.Filename)
	} else {
		ls.diagStatus[fd.Filename] = summary
		ls.diagCache[fd.Filename] = fd.Diags
	}
	ls.mu.Unlock()
	if ls.cv.Scene == nil {
		return // not attached to a shown window: tagging is done
	}
	ls.cv.AsyncLock()
	ls.needsRender(ln)
	// the official status line is generated by UpdateStatusText,
	// which picks up the diagStatus summary for this file:
	ls.cv.UpdateStatusText()
	ls.cv.AsyncUnlock()
}

// diagSummary returns a short status-bar summary of diagnostics,
// or "" if there are no errors or warnings.
func diagSummary(diags []lang.Diagnostic) string {
	nerr, nwarn := 0, 0
	for _, d := range diags {
		switch d.Severity {
		case lang.DiagErr:
			nerr++
		case lang.DiagWarn:
			nwarn++
		}
	}
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
	return "lsp: " + strings.Join(ps, ", ")
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

// applyDiags replaces the prior diagnostic tags on the buffer with
// new ones. Only uses self-locking Lines methods, safe to call from
// any goroutine (not under AsyncLock or ln.Lock).
func (ls *LangService) applyDiags(ln *lines.Lines, diags []lang.Diagnostic) {
	fname := ln.Filename()
	ls.mu.Lock()
	prev := ls.prevDiags[fname]
	delete(ls.prevDiags, fname)
	ls.mu.Unlock()
	removeErrorTags(ln, prev)

	cur := make([]tagPos, 0, len(diags))
	for _, d := range diags {
		// info and hints are not tagged in the buffer:
		if d.Severity != lang.DiagErr && d.Severity != lang.DiagWarn {
			continue
		}
		lnum := d.Region.Start.Line
		if !ln.IsValidLine(lnum) {
			continue
		}
		llen := ln.LineLen(lnum)
		st := max(0, d.Region.Start.Char)
		ed := llen // multi-line regions are tagged through the end of the first line
		if d.Region.End.Line == lnum {
			ed = min(llen, max(st+1, d.Region.End.Char))
		}
		ln.AddTag(lnum, st, ed, token.Error)
		cur = append(cur, tagPos{line: lnum, st: st, ed: ed, msg: d.Msg, sev: diagWord(d.Severity)})
	}
	if len(cur) > 0 {
		ls.mu.Lock()
		ls.prevDiags[fname] = cur
		ls.mu.Unlock()
	}
}

// removeErrorTags removes the Error tags from the lines of the given
// tag records: we own the Error tokens in LSP mode (spell-check uses
// a distinct TextSpellErr token).
func removeErrorTags(ln *lines.Lines, tags []tagPos) {
	done := make(map[int]bool)
	for _, p := range tags {
		if done[p.line] || !ln.IsValidLine(p.line) {
			continue
		}
		done[p.line] = true
		removeTags(ln, p.line, func(tk token.Tokens) bool { return tk == token.Error })
	}
}

// removeTags removes the tags on the given line for which the
// given function returns true, returning whether any were removed,
// in which case the line is marked up again.
func removeTags(ln *lines.Lines, line int, remove func(tk token.Tokens) bool) bool {
	tags := ln.AdjustedTags(line)
	out := make(lexer.Line, 0, len(tags))
	for _, t := range tags {
		if !remove(t.Token.Token) {
			out = append(out, t)
		}
	}
	if len(out) == len(tags) {
		return false
	}
	ln.SetTags(line, out)
	ln.MarkupLines(line, line)
	return true
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
	delete(ls.diagCache, fname)
	ls.mu.Unlock()
	removeErrorTags(ln, prev)
	ls.needsRender(ln)
}

// diagCovers reports whether the diagnostic region contains pos,
// with inclusive boundaries matching [LangService.diagAt].
func diagCovers(d lang.Diagnostic, pos textpos.Pos) bool {
	return !pos.IsLess(d.Region.Start) && !d.Region.End.IsLess(pos)
}

// diagsAt returns the cached diagnostics for this file that cover pos.
func (ls *LangService) diagsAt(ln *lines.Lines, pos textpos.Pos) []lang.Diagnostic {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	var out []lang.Diagnostic
	for _, d := range ls.diagCache[ln.Filename()] {
		if diagCovers(d, pos) {
			out = append(out, d)
		}
	}
	return out
}

// diagsIn returns the cached diagnostics for this file whose regions
// overlap the lines of the given region, for use as code action context.
func (ls *LangService) diagsIn(ln *lines.Lines, reg textpos.Region) []lang.Diagnostic {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	var out []lang.Diagnostic
	for _, d := range ls.diagCache[ln.Filename()] {
		if d.Region.End.Line >= reg.Start.Line && d.Region.Start.Line <= reg.End.Line {
			out = append(out, d)
		}
	}
	return out
}

////////   Hover

// HoverAt returns hover documentation for the word at the given
// position, using a short-lived cache. The request runs synchronously
// on the UI goroutine with the server's bounded quick timeout: tooltips
// are generated once, at LongHoverStart, so there is no second request
// that an async result could arrive on. Any diagnostic message tagged
// at that position is included first, from the local record, followed
// by the available quick fixes.
func (ls *LangService) HoverAt(tv *TextEditor, pos textpos.Pos) string {
	ln := tv.Lines
	if ln == nil || !ls.LSPActive(ln) || !ln.IsValidLine(pos.Line) {
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
	if diag != "" {
		txt += ls.quickFixHint(ln, pos)
	}

	ls.mu.Lock()
	ls.hover = hoverCache{valid: true, time: time.Now(), key: key, text: txt}
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

// quickFixHint returns a hover text suffix listing the titles of the
// quick fixes available for the diagnostic at pos, pairing fixes with
// diagnostics on hover. It uses the quick timeout and returns "" on any
// failure, so hover never blocks or misbehaves because of it.
func (ls *LangService) quickFixHint(ln *lines.Lines, pos textpos.Pos) string {
	diags := ls.diagsAt(ln, pos)
	if len(diags) == 0 {
		return ""
	}
	actions, err := ls.lspB.CodeActionsQuick(ln, diags[0].Region, diags)
	if err != nil {
		return ""
	}
	var ts []string
	for _, ca := range actions {
		if ca.Disabled != nil || ca.Kind == "refactor" || strings.HasPrefix(ca.Kind, "refactor.") {
			continue // fixes only on hover; refactorings via the prompt
		}
		ts = append(ts, ca.Title)
		if len(ts) >= 3 {
			break
		}
	}
	if len(ts) == 0 {
		return ""
	}
	return "\n⚡ quick fix: " + strings.Join(ts, "; ") + "  (" + string(KeyQuickFix.Chord()) + ")"
}
