// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
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
	prevDiags map[string][]tagPos    // filename -> currently tagged regions

	// diagStatus is a short status-bar summary of diagnostics per file,
	// used by [Code.UpdateStatusText] when generating the official status
	// line, which it regenerates from scratch on every cursor movement.
	diagStatus map[string]string

	// diagCache is the full diagnostic list per file, from the latest
	// publish, used as code action context for quick fixes.
	diagCache map[string][]lang.Diagnostic

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
		diagCache:  make(map[string][]lang.Diagnostic),
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
