// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"fmt"
	"image"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/parse/complete"
	"cogentcore.org/core/text/parse/lexer"
	"cogentcore.org/core/text/parse/syms"
	"cogentcore.org/core/text/search"
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

	// sigStage is the live signature help popup, anchored near the
	// cursor like the completion popup but display-only. Any existing
	// popup is canceled before a new one is shown, following the
	// completer's logic, so exactly one can exist at a time.
	sigStage *core.Stage

	quit chan struct{}
}

// sigHelpCache is a short-lived single-entry cache for signature help,
// keyed by the exact text prefix before the cursor, which changes on
// every keystroke, so re-requests only happen when the prefix recurs.
type sigHelpCache struct {
	valid  bool
	time   time.Time
	fname  string
	prefix string
	sh     *lsp.SignatureHelp
	act    int
}

// sigHelpCacheTime is how long cached signature help remains valid.
// It is short since signatures are only shown while actively typing,
// and the cache key is the exact text prefix before the cursor.
const sigHelpCacheTime = 5 * time.Second

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
	for _, t := range ls.semantic {
		t.Stop()
	}
	ls.semantic = make(map[string]*time.Timer)
	ls.mu.Unlock()
	ls.clearSigHelp() // close any open signature popup
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
	ls.SemanticRefresh(ln)
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

// Lookup shows the lookup / quick-fix chooser menu at the cursor, the
// Ctrl+. action: quick fixes and refactorings from the language server,
// plus symbol lookup (docs or go-to-definition). For LSP-backed files
// it uses the language-server lookup; otherwise it falls back to the
// editor's own parse-based lookup.
func (cv *Code) Lookup() { //types:add
	tv := cv.ActiveEditor()
	if tv == nil || tv.Lines == nil {
		return
	}
	ls := cv.LangSvc()
	if ls.LSPActive(tv.Lines) && tv.Complete != nil && tv.Complete.LookupFunc != nil {
		// same setup the editor's own Lookup does: resolve the word end
		// and the line-prefix text, then run the lookup function.
		ln := tv.Lines
		pos := tv.CursorPos
		if tv.HasSelection() {
			pos.Line = tv.SelectRegion.Start.Line
			if tv.SelectRegion.End.Line != pos.Line {
				return
			}
			pos.Char = tv.SelectRegion.End.Char
		} else if !ln.IsWordEnd(pos) {
			pos.Char = ln.WordAt(pos).End.Char
		}
		if !ln.IsValidLine(pos.Line) {
			return
		}
		ls.Change(ln)
		st := textpos.Pos{Line: pos.Line, Char: 0}
		en := textpos.Pos{Line: pos.Line, Char: pos.Char}
		txt := ""
		if tbe := ln.Region(st, en); tbe != nil {
			txt = strings.TrimLeft(string(tbe.ToBytes()), " \t")
		}
		tv.Complete.Cancel()
		tv.Complete.LookupFunc(tv.Complete.Context, txt, pos.Line, pos.Char)
		return
	}
	tv.Lookup()
}

// lookupFun returns a LookupFunc that shows a completer-style chooser
// menu at the cursor: the quick fixes / refactorings available from the
// language server (applied on click), followed by the symbol lookup
// actions -- showing hover docs if the symbol is defined in the current
// file, and opening the defining file otherwise. This is the Ctrl+.
// action, restoring the old lookup menu with Quick fix as an option.
func (ls *LangService) lookupFun(ln *lines.Lines) complete.LookupFunc {
	return func(data any, txt string, posLine, posChar int) complete.Lookup {
		var ld complete.Lookup
		pos := textpos.Pos{Line: posLine, Char: posChar}
		if !ln.IsValidLine(pos.Line) {
			return ld
		}
		tv, _, has := ls.cv.EditorForLines(ln)
		if !has || tv == nil || tv.Scene == nil {
			return ld
		}
		// cancel any existing completion popup before showing ours,
		// like the completer does before offering a new one:
		if tv.Complete != nil {
			tv.Complete.Cancel()
		}
		ls.Change(ln) // make sure the server has the current text

		// quick fixes / refactorings first, if any:
		ls.mu.Lock()
		all := ls.diagCache[ln.Filename()]
		ls.mu.Unlock()
		var diags []lang.Diagnostic
		for _, d := range all {
			if diagCovers(d, pos) {
				diags = append(diags, d)
			}
		}
		reg := textpos.Region{Start: pos, End: pos}
		if w := ln.WordAt(pos); w.Start.Line == pos.Line {
			reg = w
		}
		actions, _ := ls.lspB.CodeActionsQuick(ln, reg, diags) // 2s bound; none on failure

		var items []core.ChooserItem
		for i := range actions {
			ca := &actions[i]
			t := ca.Title
			switch {
			case ca.Disabled != nil:
				t += " -- " + ca.Disabled.Reason
			case ca.IsPreferred:
				t += " (preferred)"
			}
			items = append(items, core.ChooserItem{Text: "Fix: " + t, Value: *ca})
		}
		locs := ls.lspB.Definition(ln, pos)
		switch {
		case len(locs) > 0 && locs[0].Filename != ln.Filename():
			items = append(items, core.ChooserItem{
				Text:  "Go to definition: " + locString(ls.cv, locs[0]),
				Value: locs[0],
			})
		default:
			docs := ls.lspB.Hover(ln, pos)
			if docs != "" {
				items = append(items, core.ChooserItem{
					Text:  "Docs: " + strings.ReplaceAll(firstLine(docs), "\n", " "),
					Value: docs,
				})
			}
		}
		if len(items) == 0 {
			ls.cv.SetStatus("Lookup: nothing available at cursor")
			return ld
		}
		m := core.NewMenu(func(m *core.Scene, mp image.Point) {
			for _, it := range items {
				it := it
				b := core.NewButton(m).SetText(it.Text)
				b.OnClick(func(e events.Event) {
					m.Stage.ClosePopup()
					switch v := it.Value.(type) {
					case lsp.CodeAction:
						ls.applyCodeAction(ln, &v)
					case lang.Location:
						ls.cv.openLoc(v)
						ld.Filename = v.Filename
						ld.StLine = v.Region.Start.Line
						ld.EdLine = v.Region.End.Line
					case string:
						textcore.TextDialog(nil, "Lookup: "+txt, v)
						ld.Text = []byte(v)
					}
				})
			}
		}, tv, editorCursorPos(tv, pos))
		if m != nil {
			m.Run()
		}
		return ld
	}
}

// firstLine returns the first line of the given text, for compact menu
// labels from multi-line hover docs.
func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// applyCodeAction runs a code action chosen from the lookup menu,
// reporting the result in the status bar.
func (ls *LangService) applyCodeAction(ln *lines.Lines, ca *lsp.CodeAction) {
	out, err := ls.lspB.RunCodeAction(ln, ca)
	if err != nil {
		ls.cv.SetStatus("Quick fix: " + err.Error())
		return
	}
	n := 0
	for _, f := range out {
		n += len(f.Edits)
		if fln := ls.cv.GetOpenFile(f.Filename); fln != nil {
			if ftv, _, has := ls.cv.EditorForLines(fln); has && ftv != nil {
				ftv.NeedsRender()
			}
		}
	}
	ls.cv.SetStatus(fmt.Sprintf("Applied: %s (%d edits in %d files)", ca.Title, n, len(out)))
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
		// semantic token regions are stale once the text changes:
		ls.bumpSemantic(ln.Filename())
		ls.scheduleSemantic(ln)
	}
}

// semanticChangeDelay is how long to wait after the last change before
// re-requesting semantic tokens, so a burst of typing is one request.
const semanticChangeDelay = 750 * time.Millisecond

// scheduleSemantic starts (or resets) the debounce timer for a semantic
// tokens refresh following an edit.
func (ls *LangService) scheduleSemantic(ln *lines.Lines) {
	fname := ln.Filename()
	ls.mu.Lock()
	t, has := ls.semantic[fname]
	if !has {
		t = time.AfterFunc(semanticChangeDelay, func() { ls.SemanticRefresh(ln) })
		ls.semantic[fname] = t
	} else {
		t.Reset(semanticChangeDelay)
	}
	ls.mu.Unlock()
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
	if ls.sigStage != nil {
		ls.clearSigHelp() // also closes any signature popup for this buffer
	}
	ls.clearDiags(ln)
	ls.bumpSemantic(ln.Filename())
	ls.clearSemanticTags(ln)
	ls.lspB.Close(ln)
	ls.mu.Lock()
	if t, has := ls.debounce[ln.Filename()]; has {
		t.Stop()
		delete(ls.debounce, ln.Filename())
	}
	if t, has := ls.semantic[ln.Filename()]; has {
		t.Stop()
		delete(ls.semantic, ln.Filename())
	}
	delete(ls.semGen, ln.Filename())
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
		case fe, ok := <-ls.lspB.AppliedEdits():
			if !ok {
				return
			}
			// edits are already applied to the buffers and files by the
			// backend; here we only refresh the affected views:
			if ls.cv.Scene == nil {
				continue
			}
			ls.cv.AsyncLock()
			for _, f := range fe {
				ln := ls.cv.GetOpenFile(f.Filename)
				if ln == nil {
					continue
				}
				if tv, _, hasTv := ls.cv.EditorForLines(ln); hasTv && tv != nil {
					tv.NeedsRender()
				}
			}
			ls.cv.AsyncUnlock()
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
				delete(ls.diagCache, fd.Filename)
			} else {
				ls.diagStatus[fd.Filename] = summary
				ls.diagCache[fd.Filename] = fd.Diags
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
	delete(ls.diagCache, fname)
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

////////   Semantic tokens

// semanticMaxLines is the maximum buffer size for which we request
// semantic tokens; larger files keep whatever markup they have.
const semanticMaxLines = 8192

// SemanticRefresh requests semantic tokens for the given buffer from
// the server asynchronously, replacing any prior semantic tags.
// The request runs in its own goroutine with the server's bounded
// timeout; results are applied with self-locking Lines methods, and
// discarded if a newer pass (or a change/close) has superseded this one.
func (ls *LangService) SemanticRefresh(ln *lines.Lines) {
	if ln == nil || ln.Filename() == "" || !ls.LSPActive(ln) {
		return
	}
	if !ls.lspB.Ready(ln) {
		return // server still loading: the next change re-triggers
	}
	switch ln.FileInfo().Known {
	case fileinfo.Markdown, fileinfo.Html, fileinfo.Xml, fileinfo.Json, fileinfo.Toml, fileinfo.Yaml, fileinfo.Css, fileinfo.Svg:
		return // markup-based files keep their own markup
	}
	if ln.NumLines() > semanticMaxLines {
		return
	}
	fname := ln.Filename()
	ls.mu.Lock()
	gen := ls.semGen[fname] + 1
	ls.semGen[fname] = gen
	ls.mu.Unlock()
	go ls.semanticPass(ln, fname, gen)
}

// bumpSemantic marks any in-flight semantic pass for this file stale.
func (ls *LangService) bumpSemantic(fname string) {
	ls.mu.Lock()
	ls.semGen[fname]++
	ls.mu.Unlock()
}

// semanticPass runs one semantic tokens request/apply cycle for a file.
func (ls *LangService) semanticPass(ln *lines.Lines, fname string, gen int) {
	stale := func() bool {
		ls.mu.Lock()
		defer ls.mu.Unlock()
		return ls.semGen[fname] != gen || ls.cv.Scene == nil
	}
	toks, err := ls.lspB.SemanticTokensFull(ln)
	if stale() {
		return // file changed or a newer pass started while we waited
	}
	if err != nil || len(toks) == 0 {
		if len(toks) == 0 {
			// no tokens: remove any we previously applied, and remember
			// to keep trying (server may have just gained support).
			ls.clearSemanticTags(ln)
		}
		return
	}
	ls.applySemanticTags(ln, toks)
	if stale() {
		return
	}
	ls.cv.AsyncLock()
	if ls.cv.Scene != nil {
		if tv, _, hasTv := ls.cv.EditorForLines(ln); hasTv && tv != nil {
			tv.NeedsRender()
		}
	}
	ls.cv.AsyncUnlock()
}

// applySemanticTags replaces the semantic token tags we own on the
// buffer with new ones derived from the server's tokens, and marks up
// the affected lines. Only uses self-locking Lines methods.
func (ls *LangService) applySemanticTags(ln *lines.Lines, toks []lsp.SemanticToken) {
	fname := ln.Filename()

	// collect the lines we previously tagged so we can clean them:
	ls.mu.Lock()
	tagged := ls.semanticTagged[fname]
	ls.mu.Unlock()

	// bucket new tags per line, dropping tokens on invalid lines:
	perLine := make(map[int]lexer.Line, 64)
	for _, t := range toks {
		if !ln.IsValidLine(t.Start.Line) {
			continue
		}
		tok, ok := semanticTokenTypeMap[t.Type]
		if !ok || tok == token.None {
			continue
		}
		st := max(0, t.Start.Char)
		ed := min(ln.LineLen(t.Start.Line), t.End.Char)
		if ed <= st {
			continue
		}
		if t.Deprecated {
			tok = token.KeywordReserved // distinct style, not owned by diagnostics
		}
		lx := lexer.NewLex(token.KeyToken{Token: tok}, st, ed)
		lx.Time.Now()
		perLine[t.Start.Line] = append(perLine[t.Start.Line], lx)
	}

	// lines to update = union of old-tagged and new-tagged lines.
	// we own the tokens in semanticTokenMap in LSP mode, so cleaning
	// removes exactly our previous semantic tags (Error tags from
	// diagnostics pass through untouched):
	lnum := ln.NumLines()
	changed := make(map[int]bool, len(perLine))
	if tagged {
		for i := range lnum {
			changed[i] = true
		}
	}
	for i := range perLine {
		changed[i] = true
	}
	for i := range changed {
		if !ln.IsValidLine(i) {
			continue
		}
		tags := ln.AdjustedTags(i)
		out := make(lexer.Line, 0, len(tags)+len(perLine[i]))
		for _, t := range tags {
			if semanticOwnedTokens[t.Token.Token] {
				continue
			}
			out = append(out, t)
		}
		out = append(out, perLine[i]...)
		ln.SetTags(i, out)
	}
	for i := range changed {
		ln.MarkupLines(i, i)
	}

	ls.mu.Lock()
	ls.semanticTagged[fname] = true
	ls.mu.Unlock()
}

// clearSemanticTags removes any semantic token tags we applied.
func (ls *LangService) clearSemanticTags(ln *lines.Lines) {
	fname := ln.Filename()
	ls.mu.Lock()
	tagged := ls.semanticTagged[fname]
	delete(ls.semanticTagged, fname)
	ls.mu.Unlock()
	if !tagged {
		return
	}
	lnum := ln.NumLines()
	for i := range lnum {
		if !ln.IsValidLine(i) {
			continue
		}
		tags := ln.AdjustedTags(i)
		out := make(lexer.Line, 0, len(tags))
		changed := false
		for _, t := range tags {
			if semanticOwnedTokens[t.Token.Token] {
				changed = true
				continue
			}
			out = append(out, t)
		}
		if changed {
			ln.SetTags(i, out)
			ln.MarkupLines(i, i)
		}
	}
}

// semanticTokenMap maps LSP semantic token type names to the token
// tags we apply for them. The set of types is what we declare in our
// client capabilities, so decoding and rendering stay in sync.
// We own these tokens in LSP mode for tag cleanup purposes.
var semanticTokenTypeMap = map[string]token.Tokens{
	"namespace":     token.NameNamespace,
	"type":          token.NameType,
	"class":         token.NameStruct,
	"enum":          token.NameEnum,
	"interface":     token.NameInterface,
	"struct":        token.NameStruct,
	"typeParameter": token.NameTypeParam,
	"parameter":     token.NameVarParam,
	"variable":      token.NameVar,
	"property":      token.NameProperty,
	"enumMember":    token.NameEnumMember,
	"event":         token.NameEvent,
	"function":      token.NameFunction,
	"method":        token.NameMethod,
	"keyword":       token.Keyword,
	"modifier":      token.KeywordDeclaration,
	"comment":       token.Comment,
	"string":        token.LitStr,
	"number":        token.LitNum,
	"regexp":        token.LitStrRegex,
	"operator":      token.Operator,
	"decorator":     token.NameDecorator,
}

// semanticOwnedTokens is the set of token tags we own for semantic
// highlighting, used to clean up our previous tags: cleanup must not
// remove diagnostic (Error) tags or tags from other sources.
var semanticOwnedTokens = func() map[token.Tokens]bool {
	m := make(map[token.Tokens]bool, len(semanticTokenTypeMap)+1)
	for _, t := range semanticTokenTypeMap {
		m[t] = true
	}
	m[token.KeywordReserved] = true // our deprecated-token style
	return m
}()

////////   Signature help

// signatureStatusPrefix marks status-bar messages that are signature
// help, so they can be recognized and cleared.
const signatureStatusPrefix = "call: "

// SigHelpUpdate checks whether the editor cursor is in a function-call
// parameter context and updates the signature display accordingly: a
// display-only popup anchored below the cursor (like the completion
// popup, but nothing is ever inserted) plus the status-bar line.
// Should be called on editor Input events, from the event loop.
func (ls *LangService) SigHelpUpdate(tv *TextEditor) {
	if tv == nil || tv.Lines == nil {
		ls.clearSigHelp()
		return
	}
	ln := tv.Lines
	if !ls.LSPActive(ln) || !ls.lspB.Ready(ln) {
		ls.clearSigHelp()
		return
	}
	pos := tv.CursorPos
	if !ln.IsValidLine(pos.Line) {
		ls.clearSigHelp()
		return
	}
	text := string(ln.Line(pos.Line))
	rn := []rune(text)
	if pos.Char > len(rn) {
		pos.Char = len(rn)
	}
	prefix := string(rn[:pos.Char])
	if !inCallContext(prefix) {
		ls.clearSigHelp()
		return
	}

	fname := ln.Filename()
	ls.mu.Lock()
	c := &ls.sigHelp
	if c.valid && c.fname == fname && c.prefix == prefix && time.Since(c.time) < sigHelpCacheTime {
		sh, act := c.sh, c.act
		ls.mu.Unlock()
		ls.showSigHelp(tv, sh, act)
		return
	}
	c.valid = false
	retrigger := ls.sigStatus != ""
	ls.mu.Unlock()

	sh, act, err := ls.lspB.SignatureHelp(ln, pos, retrigger)
	if err != nil || sh == nil || act < 0 {
		ls.clearSigHelp()
		return
	}
	ls.mu.Lock()
	c.valid = true
	c.time = time.Now()
	c.fname = fname
	c.prefix = prefix
	c.sh = sh
	c.act = act
	ls.mu.Unlock()
	ls.showSigHelp(tv, sh, act)
}

// inCallContext reports whether the text before the cursor ends inside
// an unclosed "(" (possibly after typing an argument, i.e. any of the
// signature trigger characters appear after the innermost open paren).
// This is the client-side equivalent of the server returning no help:
// it keeps us from requesting signatures while the cursor sits on a
// plain line, and from leaving a stale signature up after ")" closes
// the call.
func inCallContext(prefix string) bool {
	rn := []rune(prefix)
	depth := 0
	paren := -1
	for i := len(rn) - 1; i >= 0; i-- {
		switch rn[i] {
		case ')', ']', '}':
			depth++
		case '(', '[', '{':
			if depth > 0 {
				depth--
			} else {
				if rn[i] == '(' {
					paren = i
				}
				return paren >= 0 // unclosed non-paren bracket: no
			}
		}
	}
	return paren >= 0 || depth > 0 // unterminated open paren
}

// showSigHelp formats and displays the active signature: in a popup
// anchored near the cursor (like the completion popup, but strictly
// display-only: nothing is ever inserted), and in the status bar as a
// persistent record. The active parameter is marked with brackets.
func (ls *LangService) showSigHelp(tv *TextEditor, sh *lsp.SignatureHelp, act int) {
	actP := -1
	if sh != nil && act >= 0 && act < len(sh.Signatures) {
		sig := &sh.Signatures[act]
		actP = 0
		if sh.ActiveParameter != nil && *sh.ActiveParameter >= 0 {
			actP = *sh.ActiveParameter
		}
		if actP >= len(sig.Parameters) {
			actP = len(sig.Parameters) - 1
		}
	}
	txt := lsp.SignatureText(sh, act, actP, "[", "]")
	if txt == "" {
		ls.clearSigHelp()
		return
	}
	ls.showSigPopup(tv, txt)
	msg := signatureStatusPrefix + txt
	ls.mu.Lock()
	ls.sigStatus = msg
	ls.mu.Unlock()
	// go through the official status pathway: UpdateStatusText
	// regenerates the line from scratch, picking up sigStatus:
	ls.cv.UpdateStatusText()
}

// showSigPopup shows the display-only signature popup anchored below
// the cursor, in the style of the completion popup. It must be called
// from the event loop. Following the completer's logic (Complete.Show
// cancels any existing stage before showing a new one), any existing
// popup is canceled first and a fresh one is always shown, so popups
// can never accumulate or overlap.
func (ls *LangService) showSigPopup(tv *TextEditor, txt string) {
	ls.closeSigPopup() // like Complete.Cancel before each Show
	if tv == nil || tv.Lines == nil || tv.Scene == nil {
		return
	}
	p := editorCursorPos(tv, tv.CursorPos)
	if p == (image.Point{}) {
		return
	}
	p.Y -= 16 // todo: make this the font height
	fname := tv.Lines.Filename()
	sc := core.NewScene(fname + "-sig")
	core.StyleMenuScene(sc)
	ls.sigStage = core.NewPopupStage(core.CompleterStage, sc, tv).SetPos(p)
	ls.sigStage.SetClickOff(false) // typing continues in the editor; we manage closing
	sc.OnFocus(func(e events.Event) {
		tv.SetFocus() // never take focus: just return it
	})
	// match the editor's font size so the signature reads as part of the buffer:
	fsz := tv.Styles.Font.Size
	core.NewText(sc).SetType(core.TextSupporting).
		SetText(txt).Styler(func(s *styles.Style) {
		s.SetTextWrap(false)
		s.Font.Size = fsz
	})
	ls.sigStage.Run()
}

// closeSigPopup closes the signature popup if one is open.
// Safe to call from the event loop.
func (ls *LangService) closeSigPopup() {
	ls.mu.Lock()
	st := ls.sigStage
	ls.sigStage = nil
	ls.mu.Unlock()
	if st != nil && st.Scene != nil {
		st.ClosePopup()
	}
}

// editorCursorPos returns the window-level pixel position just below
// the cursor at the given position, for anchoring popups, mirroring
// how the completion popup computes its position via charStartPos.
// Returns the zero point if the editor or position is not available.
func editorCursorPos(tv *TextEditor, pos textpos.Pos) image.Point {
	if tv == nil || tv.Lines == nil || tv.Scene == nil {
		return image.Point{}
	}
	return tv.CharStartPos(pos).Add(math32.Vec2(5, 10)).ToPoint()
}

// SigStatusMsg returns the current signature help status message, or ""
// if none. Called from UpdateStatusText when regenerating the official
// status line.
func (ls *LangService) SigStatusMsg() string {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	return ls.sigStatus
}

// clearSigHelp removes the signature help from the status bar if one
// was being shown.
func (ls *LangService) clearSigHelp() {
	ls.mu.Lock()
	had := ls.sigStatus != ""
	st := ls.sigStage
	ls.sigHelp.valid = false
	ls.sigStatus = ""
	ls.sigStage = nil
	ls.mu.Unlock()
	if had {
		ls.cv.UpdateStatusText() // regenerate the official status line
	}
	if st != nil && st.Scene != nil {
		st.ClosePopup()
	}
}

////////   Inlay hints

// goplsInlayHints are the gopls inlay hint categories to enable when
// the InlayHints setting is on, from gopls internal settings
// (ui.inlayhint.hints map). gopls only returns hints for categories
// explicitly enabled, and its default is an empty map.
var goplsInlayHints = map[string]any{
	"parameterNames":         true,
	"assignVariableTypes":    true,
	"constantValues":         true,
	"rangeVariableTypes":     true,
	"compositeLiteralTypes":  true,
	"compositeLiteralFields": true,
	"functionTypeParameters": true,
	"ignoredError":           true,
}

// serverOptions returns the additional per-server options for the
// given inlay-hints setting (server ID -> options). Only gopls is
// configured currently; other servers gate hints differently and
// return none, which is a valid empty result.
func serverOptions(inlayHints bool) map[string]map[string]any {
	if !inlayHints {
		return nil
	}
	return map[string]map[string]any{
		"gopls": {"hints": goplsInlayHints},
	}
}

// SetInlayHints turns the inlay hints setting on or off, applying it to
// the running language servers: already-running gopls re-fetches its
// settings through workspace/configuration (gopls supports dynamic
// configuration), so no restart is needed for it; servers that only
// read options at startup pick the change up when they next start.
// The server work runs in the background.
func (cv *Code) SetInlayHints(on bool) { //types:add
	if cv.Settings.InlayHints == on {
		return
	}
	cv.Settings.InlayHints = on
	ls := cv.LangSvc()
	go func() {
		ls.lspB.SetOptions(serverOptions(on))
		ls.lspB.Reconfigure()
	}()
	cv.SetStatus(fmt.Sprintf("Inlay hints: %v (displayed by the Show Inlay Hints command)", on))
}

// ShowInlayHints requests the language server's inlay hints for the
// current file and lists them in a popup, as a display-only overview:
// nothing is inserted into the buffer (the per-hint TextEdits that would
// insert the label text are ignored). Requires the InlayHints setting to
// be on for servers like gopls that gate hints behind settings.
func (cv *Code) ShowInlayHints() { //types:add
	tv := cv.ActiveEditor()
	if tv == nil || tv.Lines == nil {
		return
	}
	ln := tv.Lines
	ls := cv.LangSvc()
	if !ls.LSPActive(ln) {
		cv.SetStatus("Inlay hints: requires an LSP language server for this file")
		return
	}
	if !cv.Settings.InlayHints {
		cv.SetStatus("Inlay hints: enable the InlayHints setting first (servers gate these behind settings)")
		return
	}
	ls.Change(ln)
	numLines := ln.NumLines()
	hints, err := ls.lspB.InlayHints(ln, textpos.Region{
		Start: textpos.Pos{Line: 0, Char: 0},
		End:   textpos.Pos{Line: numLines, Char: 0},
	})
	if err != nil {
		cv.SetStatus("Inlay hints: " + err.Error())
		return
	}
	var sb strings.Builder
	n := 0
	for _, h := range hints {
		txt := h.LabelText()
		if txt == "" {
			continue
		}
		fmt.Fprintf(&sb, "%d:%d: %s\n", h.Position.Line+1, h.Position.Character+1, txt)
		n++
	}
	if n == 0 {
		cv.SetStatus("Inlay hints: none returned by the server for this file")
		return
	}
	d := core.NewBody("Inlay Hints").SetTitle(fmt.Sprintf("Inlay hints for %s (%d)", filepath.Base(ln.Filename()), n))
	core.NewText(d).SetType(core.TextSupporting).SetText(sb.String())
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar).SetText("Close")
	})
	d.RunDialog(tv)
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
	if diag != "" {
		// pair fixes with the diagnostic: list available quick fixes
		// right in the hover, on the quick timeout so hover stays snappy:
		txt += ls.quickFixHint(ln, pos)
	}

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

// wordAtCursor returns the word at the cursor for use as a rename
// placeholder, preferring a single-line selection if one exists.
// Returns "" if there is no word at the cursor.
func wordAtCursor(tv *TextEditor) string {
	ln := tv.Lines
	if ln == nil {
		return ""
	}
	if tv.HasSelection() {
		r := tv.SelectRegion
		if r.Start.Line == r.End.Line && r.End.Char > r.Start.Char {
			return string(ln.Line(r.Start.Line)[r.Start.Char:r.End.Char])
		}
		return ""
	}
	pos := tv.CursorPos
	if !ln.IsValidLine(pos.Line) {
		return ""
	}
	if ln.IsWordEnd(pos) && pos.Char > 0 {
		pos.Char-- // move inside the word so WordAt finds it
	}
	w := ln.WordAt(pos)
	if w.Start.Line != pos.Line || w.End.Char <= w.Start.Char {
		return ""
	}
	ed := min(w.End.Char, ln.LineLen(pos.Line))
	if ed <= w.Start.Char {
		return ""
	}
	return string(ln.Line(pos.Line)[w.Start.Char:ed])
}

// RenameSymbol prompts for a new name and renames the symbol at the
// cursor across the workspace, using the language server. The proposed
// edits are shown in the Find tab for review before applying: click a
// result to jump to it, Replace for one at a time, or All for all
// (open buffers are edited, unopened files are opened; save to write).
func (cv *Code) RenameSymbol() { //types:add
	tv := cv.ActiveEditor()
	if tv == nil || tv.Lines == nil {
		return
	}
	pos, ok := cursorWordPos(tv)
	if !ok {
		return
	}
	ln := tv.Lines
	ls := cv.LangSvc()
	if !ls.LSPActive(ln) {
		cv.SetStatus("Rename: requires an LSP language server for this file")
		return
	}
	word := wordAtCursor(tv)
	if word == "" {
		cv.SetStatus("Rename: no symbol at cursor")
		return
	}
	d := core.NewBody("Rename " + word + " to")
	tf := core.NewTextField(d).SetPlaceholder("new name")
	tf.SetText(word)
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).SetText("Rename").OnClick(func(e events.Event) {
			newName := strings.TrimSpace(tf.Text())
			d.Close()
			if newName == "" || newName == word {
				return
			}
			ls.previewRename(ln, pos, word, newName)
		})
	})
	d.RunWindowDialog(tv)
}

// previewRename computes the workspace edits that rename the symbol at
// pos to newName, and shows them in the Find tab for review: clicking
// a result jumps to it, Replace applies one at a time, and All applies
// them all (with confirmation). This reuses [FindPanel] in rename mode.
func (ls *LangService) previewRename(ln *lines.Lines, pos textpos.Pos, old, newName string) {
	cv := ls.cv
	// synchronous, as for GotoDefinition: bounded server timeouts.
	cv.SetStatus("Renaming " + old + "...")
	we, err := ls.lspB.RenameEdits(ln, pos, newName)
	if err != nil {
		cv.SetStatus("Rename: " + err.Error())
		return
	}
	prev := ls.lspB.PreviewWorkspaceEdit(we)
	if len(prev) == 0 {
		cv.SetStatus("Rename: nothing to rename")
		return
	}
	tv := cv.Tabs()
	if tv == nil {
		cv.SetStatus("Rename: no tabs panel")
		return
	}
	fv := core.RecycleTabWidget[FindPanel](tv, "Find")
	fv.UpdateTree()
	fbuf, _ := cv.RecycleCmdBuf("Find")
	ftv := fv.TextEditor()
	ftv.SetLines(fbuf)
	fv.Time = time.Now()
	fv.Renaming = true
	fv.Renamed = 0
	fv.OnReplace = func(etv *TextEditor) {
		ls.Change(etv.Lines) // keep the language server in sync
	}
	fp := &cv.Settings.Find
	fp.Find = old
	fp.Replace = newName
	fp.IgnoreCase = false
	res := renameResults(prev)
	fv.ShowResults(res)
	cv.FocusOnPanel(TabsIndex)
	cv.SetStatus(fmt.Sprintf("Rename %s to %s: review %d edits in %d files -- Replace steps through, All applies", old, newName, len(prev), len(res)))
}

// renameResults converts rename edit previews into [search.Results]
// for display and stepwise application in the Find panel. The previews
// must be sorted by filename, as [lsp.Backend.PreviewWorkspaceEdit]
// returns them, so that the matches group by file.
func renameResults(prev []*lsp.EditPreview) []search.Results {
	var res []search.Results
	var cur *search.Results
	for _, it := range prev {
		if cur == nil || cur.Filepath != it.Filename {
			res = append(res, search.Results{Filepath: it.Filename})
			cur = &res[len(res)-1]
		}
		txt := []rune(it.LineText)
		st := min(it.Region.Start.Char, len(txt))
		ed := it.Region.End.Char
		if it.Region.End.Line != it.Region.Start.Line || ed > len(txt) {
			ed = len(txt) // multi-line region: show to end of first line
		}
		m := textpos.Match{Region: it.Region, Text: txt}
		if st <= ed {
			m.TextMatch = textpos.Range{Start: st, End: ed}
		}
		cur.Matches = append(cur.Matches, m)
		cur.Count++
	}
	return res
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
		d.AddCancel(bar)
		d.AddOK(bar).SetText("Search").OnClick(func(e events.Event) {
			search()
		})
	})
	d.RunWindowDialog(tv)
}

// diagCovers reports whether the diagnostic region contains pos,
// with inclusive boundaries matching [LangService.diagAt].
func diagCovers(d lang.Diagnostic, pos textpos.Pos) bool {
	if pos.Line < d.Region.Start.Line || pos.Line > d.Region.End.Line {
		return false
	}
	if pos.Line == d.Region.Start.Line && pos.Char < d.Region.Start.Char {
		return false
	}
	if pos.Line == d.Region.End.Line && pos.Char > d.Region.End.Char {
		return false
	}
	return true
}

// diagsIn returns the cached diagnostics for this file whose regions
// overlap the given region, for use as code action context.
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

// quickFixHint returns a hover text suffix listing the titles of the
// quick fixes available for the diagnostic at pos, pairing fixes with
// diagnostics on hover. It uses the quick timeout and returns "" on any
// failure, so hover never blocks or misbehaves because of it.
func (ls *LangService) quickFixHint(ln *lines.Lines, pos textpos.Pos) string {
	ls.mu.Lock()
	all := ls.diagCache[ln.Filename()]
	ls.mu.Unlock()
	var atPos []lang.Diagnostic
	for _, d := range all {
		if diagCovers(d, pos) {
			atPos = append(atPos, d)
		}
	}
	if len(atPos) == 0 {
		return ""
	}
	actions, err := ls.lspB.CodeActionsQuick(ln, atPos[0].Region, atPos)
	if err != nil || len(actions) == 0 {
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

// actionRegion returns the region to request code actions for:
// a single-line selection if there is one, otherwise the word at
// the cursor, falling back to the bare cursor position.
func actionRegion(tv *TextEditor) textpos.Region {
	ln := tv.Lines
	if tv.HasSelection() && tv.SelectRegion.End.Line == tv.SelectRegion.Start.Line {
		return tv.SelectRegion
	}
	pos := tv.CursorPos
	if !ln.IsValidLine(pos.Line) {
		return textpos.Region{Start: pos, End: pos}
	}
	w := ln.WordAt(pos)
	if w.Start.Line == pos.Line && w.End.Char >= w.Start.Char {
		return w
	}
	return textpos.Region{Start: pos, End: pos}
}

// QuickFix prompts for and applies a quick fix or refactoring from the
// language server at the cursor or selection, like other editors'
// Ctrl+. (Ctrl+period). The diagnostics cached for the position are
// sent as context, which is what turns the quick fixes on: organize
// imports, missing returns, error-check wrapping, typo fixes, etc,
// along with refactorings like extract function/variable.
func (cv *Code) QuickFix() { //types:add
	tv := cv.ActiveEditor()
	if tv == nil || tv.Lines == nil {
		return
	}
	ln := tv.Lines
	ls := cv.LangSvc()
	if !ls.LSPActive(ln) {
		cv.SetStatus("Quick fix: requires an LSP language server for this file")
		return
	}
	reg := actionRegion(tv)
	diags := ls.diagsIn(ln, reg)
	cv.SetStatus("Quick fix: consulting server...")
	actions, err := ls.lspB.CodeActions(ln, reg, diags)
	if err != nil {
		cv.SetStatus("Quick fix: " + err.Error())
		return
	}
	if len(actions) == 0 {
		cv.SetStatus("Quick fix: no actions available here")
		return
	}
	// preferred, enabled actions first for a useful default selection:
	sort.SliceStable(actions, func(i, j int) bool {
		a, b := &actions[i], &actions[j]
		ad, bd := a.Disabled != nil, b.Disabled != nil
		if ad != bd {
			return !ad
		}
		if a.IsPreferred != b.IsPreferred {
			return a.IsPreferred
		}
		return false
	})
	d := core.NewBody("Quick fix / refactor")
	c := core.NewChooser(d)
	items := make([]core.ChooserItem, 0, len(actions))
	for _, ca := range actions {
		t := ca.Title
		switch {
		case ca.Disabled != nil:
			t += " -- " + ca.Disabled.Reason
		case ca.IsPreferred:
			t += " (preferred)"
		}
		items = append(items, core.ChooserItem{Text: t, Value: ca.Title})
	}
	c.SetItems(items...)
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).SetText("Apply").OnClick(func(e events.Event) {
			idx := int(c.CurrentIndex)
			d.Close()
			if idx < 0 || idx >= len(actions) {
				idx = 0
			}
			ca := &actions[idx]
			if ca.Disabled != nil {
				cv.SetStatus("Quick fix: " + ca.Disabled.Reason)
				return
			}
			out, err := ls.lspB.RunCodeAction(ln, ca)
			if err != nil {
				cv.SetStatus("Quick fix: " + err.Error())
				return
			}
			n := 0
			for _, f := range out {
				n += len(f.Edits)
				if fln := cv.GetOpenFile(f.Filename); fln != nil {
					if ftv, _, has := cv.EditorForLines(fln); has && ftv != nil {
						ftv.NeedsRender()
					}
				}
			}
			cv.SetStatus(fmt.Sprintf("Applied: %s (%d edits in %d files)", ca.Title, n, len(out)))
		})
	})
	d.RunWindowDialog(tv)
}

// ShowSignatureHelp explicitly requests signature help at the cursor
// and shows it in a dialog, for cases where the automatic status-bar
// display is not desired or the cursor is not in a call context.
func (cv *Code) ShowSignatureHelp() { //types:add
	tv := cv.ActiveEditor()
	if tv == nil || tv.Lines == nil {
		return
	}
	ln := tv.Lines
	ls := cv.LangSvc()
	if !ls.LSPActive(ln) {
		cv.SetStatus("Signature help: requires an LSP language server for this file")
		return
	}
	ls.Change(ln)
	pos := tv.CursorPos
	if !ln.IsValidLine(pos.Line) {
		return
	}
	sh, act, err := ls.lspB.SignatureHelp(ln, pos, false)
	if err != nil {
		cv.SetStatus("Signature help: " + err.Error())
		return
	}
	if sh == nil || act < 0 {
		cv.SetStatus("Signature help: no callable at cursor")
		return
	}
	actP := 0
	if sh.ActiveParameter != nil && *sh.ActiveParameter >= 0 {
		actP = *sh.ActiveParameter
	}
	var sb strings.Builder
	for i, sig := range sh.Signatures {
		mark, unmark := "", ""
		if i == act {
			mark, unmark = "[", "]"
		}
		sb.WriteString(lsp.SignatureText(sh, i, actP, mark, unmark))
		if sig.Documentation != nil && sig.Documentation.Value != "" {
			sb.WriteString("\n")
			sb.WriteString(sig.Documentation.Value)
		}
		if i < len(sh.Signatures)-1 {
			sb.WriteString("\n\n")
		}
	}
	textcore.TextDialog(nil, "Signature help", sb.String())
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
