// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"unicode/utf8"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/parse/complete"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/textpos"
	"cogentcore.org/lab/goal/transpile"

	"cogentcore.org/cogent/code/lang"
	"cogentcore.org/cogent/code/lsp"
)

// The language services for Goal files come from the Go language server
// (gopls), for the Go code that each open Goal file is transpiled to, as
// `goal build` does, which is kept in a hidden buffer that is not shown,
// and updated as the Goal file changes. The positions of the requests and
// results are mapped between the Goal file and its Go code (see
// [goalCode]). The Go file is written when the Goal file is saved.

// goalLayer is the virtual text layer of the Go code of the lines of a
// Goal file with errors, which is shown after them.
const goalLayer = "goal"

// goalDoc is an open Goal file, with the hidden buffer of its Go code.
type goalDoc struct {

	// ln is the buffer of the Goal file.
	ln *lines.Lines

	// filename is the name of the Goal file.
	filename string

	// goLines is the hidden buffer of the Go code, which has the name of
	// the Go file, and is opened with the language server.
	goLines *lines.Lines

	// mu protects goLines across updating it and sending it to the
	// language server, which are done in order of gen.
	mu sync.Mutex

	// The fields below are protected by [LangService.mu].

	// code is the Go code of the Goal file in goLines.
	code *goalCode

	// gen is incremented for each update of the Go code.
	gen int

	// opened is whether goLines has been opened with the language server.
	opened bool

	// closed is whether the Goal file has been closed.
	closed bool

	// diags are the diagnostics of the Go code, in the Goal file.
	diags []lang.Diagnostic
}

// goFilename returns the name of the Go file of the given Goal file.
func goFilename(goalFile string) string {
	return strings.TrimSuffix(goalFile, filepath.Ext(goalFile)) + ".go"
}

// isGoalFile returns whether the given buffer is of a Goal file, which
// has the .goal extension (its file type can be Go, for highlighting).
func isGoalFile(ln *lines.Lines) bool {
	return ln != nil && (ln.FileInfo().Known == fileinfo.Goal || strings.EqualFold(filepath.Ext(ln.Filename()), ".goal"))
}

// goalActive returns whether the language services of the given buffer
// of a Goal file come from the Go language server, which they do with
// the LSP backend, if there is a Go language server.
func (ls *LangService) goalActive(ln *lines.Lines) bool {
	if ls.cv.Settings.LangBackend != LangBackendLSP || !isGoalFile(ln) || ln.Filename() == "" {
		return false
	}
	_, has := ls.lspB.Reg.Server(fileinfo.Go)
	return has
}

// goalDocFor returns the open Goal file of the given buffer, or nil.
func (ls *LangService) goalDocFor(ln *lines.Lines) *goalDoc {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	return ls.goals[ln]
}

// goalDocForGo returns the open Goal file whose Go code is in the Go
// file of the given name, or nil.
func (ls *LangService) goalDocForGo(gofile string) *goalDoc {
	if !strings.HasSuffix(gofile, ".go") {
		return nil
	}
	ls.mu.Lock()
	defer ls.mu.Unlock()
	for _, doc := range ls.goals {
		if lsp.SameFile(doc.goLines.Filename(), gofile) {
			return doc
		}
	}
	return nil
}

// openGoal opens the given buffer of a Goal file, if it is not open,
// transpiling it to Go code that is opened with the language server.
func (ls *LangService) openGoal(ln *lines.Lines) {
	ls.mu.Lock()
	if ls.goals[ln] != nil {
		ls.mu.Unlock()
		return
	}
	fname := ln.Filename()
	doc := &goalDoc{ln: ln, filename: fname, goLines: lines.NewLines()}
	doc.goLines.SetFilename(goFilename(fname))
	ls.goals[ln] = doc
	ls.mu.Unlock()
	ls.goalUpdate(doc)
}

// goalUpdate transpiles the Goal file again, in the background, and
// sends its Go code to the language server.
func (ls *LangService) goalUpdate(doc *goalDoc) {
	ls.mu.Lock()
	doc.gen++
	gen := doc.gen
	var prev []string
	if doc.code != nil {
		prev = doc.code.imports
	}
	ls.mu.Unlock()
	src := doc.ln.String()
	go func() {
		gc := transpileGoal(src, doc.goLines.Filename(), prev, true)
		doc.mu.Lock()
		defer doc.mu.Unlock()
		ls.mu.Lock()
		if doc.gen != gen || doc.closed {
			ls.mu.Unlock()
			return
		}
		doc.code = gc
		opened := doc.opened
		doc.opened = true
		ls.mu.Unlock()
		doc.goLines.SetText([]byte(gc.code))
		if opened {
			ls.lspB.Change(doc.goLines)
		} else {
			ls.lspB.Open(doc.goLines)
		}
		ls.applyGoalDiags(doc) // the errors of transpiling it
		if ls.cv.Settings.InlayHints {
			ls.InlayRefresh(doc.ln)
		}
	}()
}

// goalSync transpiles the Goal file again now, with the imports it had,
// and updates the Go code in its hidden buffer, which the language
// server gets with the next request, returning the Go code. It is for
// requests right after each change, such as completions, which must be
// fast, so the imports are found by the next update.
func (ls *LangService) goalSync(doc *goalDoc) *goalCode {
	ls.mu.Lock()
	var prev []string
	if doc.code != nil {
		prev = doc.code.imports
	}
	ls.mu.Unlock()
	gc := transpileGoal(doc.ln.String(), doc.goLines.Filename(), prev, false)
	doc.mu.Lock()
	defer doc.mu.Unlock()
	ls.mu.Lock()
	doc.code = gc
	doc.gen++ // any update in progress is older
	ls.mu.Unlock()
	doc.goLines.SetText([]byte(gc.code))
	ls.goalInput(doc) // for the imports, and to open it if it is not open
	return gc
}

// goalCompleter sets the completion functions of the given editor of
// the given Goal file, to use the completions of the language server for
// its Go code, at the position of the cursor in it.
func (ls *LangService) goalCompleter(tv *TextEditor, doc *goalDoc) {
	c := tv.Complete
	if c == nil {
		return
	}
	c.MatchFunc = func(data any, text string, posLine, posChar int) complete.Matches {
		gc := ls.goalSync(doc)
		gp, ok := gc.toGo(textpos.Pos{Line: posLine, Char: posChar})
		if !ok {
			return complete.Matches{}
		}
		return ls.lspB.Completions(doc.goLines, text, gp.Line, gp.Char)
	}
	c.EditFunc = func(data any, text string, cursorPos int, comp complete.Completion, seed string) complete.Edit {
		return ls.lspB.CompleteEdit(doc.goLines, text, cursorPos, comp, seed)
	}
}

// goalInput schedules an update of the Go code of the given Goal file
// after a change to it, with the debounce delay of the language servers.
func (ls *LangService) goalInput(doc *goalDoc) {
	ls.mu.Lock()
	resetTimer(ls.debounce, doc.filename, changeDelay, func() { ls.goalUpdate(doc) })
	ls.mu.Unlock()
}

// goalSaved writes the Go file of the given saved Goal file, in the
// background, as `goal build` does, after updating its Go code.
func (ls *LangService) goalSaved(doc *goalDoc) {
	ls.mu.Lock()
	stopTimer(ls.debounce, doc.filename)
	ls.mu.Unlock()
	ls.goalUpdate(doc)
	go func() {
		st := transpile.NewState()
		if err := st.TranspileFile(doc.filename, doc.goLines.Filename()); err != nil && ls.cv.Scene != nil {
			ls.cv.AsyncLock()
			ls.cv.SetStatus("Writing the Go code of " + filepath.Base(doc.filename) + ": " + err.Error())
			ls.cv.AsyncUnlock()
		}
	}()
}

// closeGoal closes the given Goal file, and its Go code with the
// language server.
func (ls *LangService) closeGoal(doc *goalDoc) {
	ls.mu.Lock()
	doc.closed = true
	delete(ls.goals, doc.ln)
	delete(ls.diagCache, doc.filename)
	stopTimer(ls.debounce, doc.filename)
	ls.mu.Unlock()
	ls.forgetFile(doc.ln, doc.filename)
	doc.ln.ClearVirtualText(goalLayer, -1)
	doc.mu.Lock()
	ls.lspB.Close(doc.goLines)
	doc.mu.Unlock()
}

// goalTarget returns the hidden buffer of the Go code of the given Goal
// file, and the position in it of the given position in the Goal file,
// for a request to the language server, and whether it has one.
func (ls *LangService) goalTarget(doc *goalDoc, pos textpos.Pos) (*lines.Lines, textpos.Pos, bool) {
	ls.mu.Lock()
	gc := doc.code
	ls.mu.Unlock()
	if gc == nil {
		return nil, textpos.Pos{}, false
	}
	gp, ok := gc.toGo(pos)
	return doc.goLines, gp, ok
}

// lspTarget returns the buffer and position for a request to the language
// server for the given position in the given buffer: the hidden buffer of
// the Go code of a Goal file, and the position in it, or else the buffer
// and position themselves. It returns false if there is no language
// server for the buffer.
func (ls *LangService) lspTarget(ln *lines.Lines, pos textpos.Pos) (*lines.Lines, textpos.Pos, bool) {
	if doc := ls.goalDocFor(ln); doc != nil {
		return ls.goalTarget(doc, pos)
	}
	return ln, pos, ls.LSPActive(ln)
}

// fromTargets returns the given locations from the language server, with
// those in the Go code of an open Goal file in the Goal file instead,
// except for those in the code that the transpiler adds, which are omitted.
func (ls *LangService) fromTargets(locs []lang.Location) []lang.Location {
	out := make([]lang.Location, 0, len(locs))
	for _, loc := range locs {
		doc := ls.goalDocForGo(loc.Filename)
		if doc == nil {
			out = append(out, loc)
			continue
		}
		ls.mu.Lock()
		gc := doc.code
		ls.mu.Unlock()
		if reg, ok := gc.toGoalRegion(loc.Region); ok {
			out = append(out, lang.Location{Filename: doc.filename, Region: reg})
		}
	}
	return out
}

// receiveGoalDiags handles the diagnostics of the Go code of the given
// Goal file, which are shown in the Goal file. Those in the code that the
// transpiler adds are at the start of the Goal file. Called from the
// consumeEvents goroutine.
func (ls *LangService) receiveGoalDiags(doc *goalDoc, diags []lang.Diagnostic) {
	ls.mu.Lock()
	gc := doc.code
	ls.mu.Unlock()
	if gc == nil {
		return
	}
	mapped := make([]lang.Diagnostic, 0, len(diags))
	for _, d := range diags {
		if reg, ok := gc.toGoalRegion(d.Region); ok {
			d.Region = reg
		} else {
			d.Region = textpos.Region{}
			d.Msg = "in the Go code: " + d.Msg
		}
		mapped = append(mapped, d)
	}
	ls.mu.Lock()
	doc.diags = mapped
	ls.mu.Unlock()
	ls.applyGoalDiags(doc)
}

// applyGoalDiags shows the diagnostics of the given Goal file: those of
// its Go code, and the errors of transpiling it, with the Go code of the
// lines with errors after them, when it is not the same as the source.
// They are also the problems of the Goal file in the Problems tab.
func (ls *LangService) applyGoalDiags(doc *goalDoc) {
	ls.mu.Lock()
	gc := doc.code
	all := append([]lang.Diagnostic{}, doc.diags...)
	ls.mu.Unlock()
	if gc == nil {
		return
	}
	for _, e := range gc.errs {
		all = append(all, transpileDiag(gc, e))
	}
	ls.mu.Lock()
	if doc.closed {
		ls.mu.Unlock()
		return
	}
	if len(all) == 0 {
		delete(ls.diagCache, doc.filename)
	} else {
		ls.diagCache[doc.filename] = all
	}
	shown := filterIgnored(all, ls.ignore)
	if summary := diagSummary(shown); summary == "" {
		delete(ls.diagStatus, doc.filename)
	} else {
		ls.diagStatus[doc.filename] = summary
	}
	ls.mu.Unlock()
	ls.applyDiags(doc.ln, shown)
	if ls.cv.Scene == nil {
		return
	}
	ls.cv.AsyncLock()
	defer ls.cv.AsyncUnlock()
	ls.setGoalCode(doc, gc, shown)
	ls.needsRender(doc.ln)
	ls.cv.UpdateStatusText()
	ls.updateProblems(false)
}

// transpileDiag returns the diagnostic of the given error of
// transpiling the Goal code.
func transpileDiag(gc *goalCode, e *transpile.Error) lang.Diagnostic {
	pos := textpos.Pos{Line: max(e.Line-1, 0)}
	if pos.Line < len(gc.src) {
		src := gc.src[pos.Line].Src
		pos.Char = utf8.RuneCountInString(src[:min(max(e.Col-1, 0), len(src))])
	}
	end := pos
	end.Char++
	return lang.Diagnostic{Region: textpos.Region{Start: pos, End: end}, Msg: e.Msg, Severity: lang.DiagErr, Source: "goal"}
}

// setGoalCode shows the Go code of the lines of the given Goal file
// with errors in the given diagnostics after them, as virtual text, when
// it is not the same as their source. Must be called under AsyncLock.
func (ls *LangService) setGoalCode(doc *goalDoc, gc *goalCode, diags []lang.Diagnostic) {
	var sty *rich.Style
	if tv, _, has := ls.cv.EditorForLines(doc.ln); has && tv != nil {
		sty = tv.HintStyle()
	} else {
		sty = rich.NewStyle()
	}
	done := map[int]bool{}
	var vts []lines.VirtualText
	for _, d := range diags {
		ln := d.Region.Start.Line
		if d.Severity != lang.DiagErr || d.Source == "goal" || done[ln] || ln >= len(gc.src) || !doc.ln.IsValidLine(ln) {
			continue
		}
		done[ln] = true
		goCode := strings.TrimSpace(gc.goLines[gc.offset+ln])
		if goCode == strings.TrimSpace(gc.src[ln].Src) {
			continue
		}
		pos := textpos.Pos{Line: ln, Char: doc.ln.LineLen(ln)}
		vts = append(vts, lines.VirtualText{Pos: pos, Text: rich.NewText(sty, []rune(fmt.Sprintf("  Go: %s", goCode)))})
	}
	doc.ln.SetVirtualText(goalLayer, -1, vts...)
}

// goalHints returns the given inlay hints of the Go code of the given
// Goal file at their positions in it, without those in the code that
// the transpiler adds.
func (ls *LangService) goalHints(doc *goalDoc, hints []lang.InlayHint) []lang.InlayHint {
	ls.mu.Lock()
	gc := doc.code
	ls.mu.Unlock()
	out := make([]lang.InlayHint, 0, len(hints))
	for _, h := range hints {
		if p, ok := gc.toGoal(h.Pos); ok {
			h.Pos = p
			out = append(out, h)
		}
	}
	return out
}
