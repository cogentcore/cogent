// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"image"
	"time"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/parse/lexer"
	"cogentcore.org/core/text/textpos"
	"cogentcore.org/core/text/token"

	"cogentcore.org/cogent/code/lsp"
)

////////   Semantic tokens

// semanticMaxLines is the maximum buffer size for which we request
// semantic tokens; larger files keep whatever markup they have.
const semanticMaxLines = 8192

// semanticChangeDelay is how long to wait after the last change before
// re-requesting semantic tokens, so a burst of typing is one request.
const semanticChangeDelay = 750 * time.Millisecond

// scheduleSemantic starts (or resets) the debounce timer for a semantic
// tokens refresh following an edit.
func (ls *LangService) scheduleSemantic(ln *lines.Lines) {
	ls.mu.Lock()
	resetTimer(ls.semantic, ln.Filename(), semanticChangeDelay, func() { ls.SemanticRefresh(ln) })
	ls.mu.Unlock()
}

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
	ls.semGen[fname]++
	gen := ls.semGen[fname]
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
	if err != nil || stale() {
		return // failed, or file changed or a newer pass started while we waited
	}
	if len(toks) == 0 {
		ls.clearSemanticTags(ln)
		return
	}
	ls.applySemanticTags(ln, toks)
	if stale() {
		return
	}
	ls.cv.AsyncLock()
	ls.needsRender(ln)
	ls.cv.AsyncUnlock()
}

// applySemanticTags replaces the semantic token tags we own on the
// buffer with new ones derived from the server's tokens, and marks up
// the affected lines. Only uses self-locking Lines methods.
func (ls *LangService) applySemanticTags(ln *lines.Lines, toks []lsp.SemanticToken) {
	fname := ln.Filename()
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
		if !ok {
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

	// lines to update = all lines if previously tagged, plus new-tagged
	// lines. Cleaning removes exactly our previous semantic tags (Error
	// tags from diagnostics pass through untouched):
	update := func(i int) {
		if !ln.IsValidLine(i) {
			return
		}
		tags := ln.AdjustedTags(i)
		out := make(lexer.Line, 0, len(tags)+len(perLine[i]))
		for _, t := range tags {
			if !semanticOwnedTokens[t.Token.Token] {
				out = append(out, t)
			}
		}
		out = append(out, perLine[i]...)
		ln.SetTags(i, out)
		ln.MarkupLines(i, i)
	}
	if tagged {
		for i := range ln.NumLines() {
			update(i)
		}
	} else {
		for i := range perLine {
			update(i)
		}
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
	for i := range ln.NumLines() {
		if ln.IsValidLine(i) {
			removeTags(ln, i, func(tk token.Tokens) bool { return semanticOwnedTokens[tk] })
		}
	}
}

// semanticTokenTypeMap maps LSP semantic token type names to the token
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
	pos := tv.CursorPos
	if !ls.LSPActive(ln) || !ls.lspB.Ready(ln) || !ln.IsValidLine(pos.Line) {
		ls.clearSigHelp()
		return
	}
	rn := ln.Line(pos.Line)
	pos.Char = min(pos.Char, len(rn))
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
	if err != nil || sh == nil {
		ls.clearSigHelp()
		return
	}
	ls.mu.Lock()
	ls.sigHelp = sigHelpCache{valid: true, time: time.Now(), fname: fname, prefix: prefix, sh: sh, act: act}
	ls.mu.Unlock()
	ls.showSigHelp(tv, sh, act)
}

// inCallContext reports whether the text before the cursor ends inside
// an unclosed "(", not nested within another kind of bracket.
// This is the client-side equivalent of the server returning no help:
// it keeps us from requesting signatures while the cursor sits on a
// plain line, and from leaving a stale signature up after ")" closes
// the call.
func inCallContext(prefix string) bool {
	rn := []rune(prefix)
	depth := 0
	for i := len(rn) - 1; i >= 0; i-- {
		switch rn[i] {
		case ')', ']', '}':
			depth++
		case '(', '[', '{':
			if depth == 0 {
				return rn[i] == '('
			}
			depth--
		}
	}
	return false
}

// showSigHelp formats and displays the active signature: in a popup
// anchored near the cursor, and in the status bar as a persistent
// record. The active parameter is marked with brackets.
func (ls *LangService) showSigHelp(tv *TextEditor, sh *lsp.SignatureHelp, act int) {
	txt := lsp.SignatureText(sh, act, sh.ActiveParam(), "[", "]")
	if txt == "" {
		ls.clearSigHelp()
		return
	}
	ls.showSigPopup(tv, txt)
	ls.mu.Lock()
	ls.sigStatus = signatureStatusPrefix + txt
	ls.mu.Unlock()
	// go through the official status pathway: UpdateStatusText
	// regenerates the line from scratch, picking up sigStatus:
	ls.cv.UpdateStatusText()
}

// showSigPopup shows the display-only signature popup anchored below
// the cursor, in the style of the completion popup. It must be called
// from the event loop. Following the completer's logic (Complete.Show
// cancels any existing stage before showing a new one), any existing
// popup is closed first, so popups can never accumulate or overlap.
func (ls *LangService) showSigPopup(tv *TextEditor, txt string) {
	ls.closeSigPopup()
	if tv == nil || tv.Lines == nil || tv.Scene == nil {
		return
	}
	p := editorCursorPos(tv, tv.CursorPos)
	if p == (image.Point{}) {
		return
	}
	p.Y -= 16 // todo: make this the font height
	sc := core.NewScene(tv.Lines.Filename() + "-sig")
	core.StyleMenuScene(sc)
	st := core.NewPopupStage(core.CompleterStage, sc, tv).SetPos(p)
	st.SetClickOff(false) // typing continues in the editor; we manage closing
	sc.OnFocus(func(e events.Event) {
		tv.SetFocus() // never take focus: just return it
	})
	// match the editor's font size so the signature reads as part of the buffer:
	fsz := tv.Styles.Font.Size
	core.NewText(sc).SetType(core.TextSupporting).SetText(txt).Styler(func(s *styles.Style) {
		s.SetTextWrap(false)
		s.Font.Size = fsz
	})
	ls.mu.Lock()
	ls.sigStage = st
	ls.mu.Unlock()
	st.Run()
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

// clearSigHelp closes any signature popup, and removes the signature
// help from the status bar if one was being shown.
func (ls *LangService) clearSigHelp() {
	ls.mu.Lock()
	had := ls.sigStatus != ""
	ls.sigHelp.valid = false
	ls.sigStatus = ""
	ls.mu.Unlock()
	ls.closeSigPopup()
	if had {
		ls.cv.UpdateStatusText() // regenerate the official status line
	}
}
