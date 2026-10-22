// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lsp

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/parse/complete"
	"cogentcore.org/core/text/textpos"

	"cogentcore.org/cogent/code/lang"
)

// isIdentRune reports whether the rune can appear in an identifier.
func isIdentRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

// completionSeed returns the seed for completions from the given line
// prefix: the trailing identifier PLUS any qualifying selector prefixes
// ("core.F" -> "core.F", not "F" as a plain whitespace seed would give),
// so that selecting a completion rewrites the whole qualified reference.
func completionSeed(text string) string {
	rn := []rune(text)
	i := len(rn)
	for i > 0 && isIdentRune(rn[i-1]) {
		i--
	}
	for i > 0 && rn[i-1] == '.' {
		i--
		for i > 0 && isIdentRune(rn[i-1]) {
			i--
		}
	}
	return string(rn[i:])
}

// identSuffixLen returns the number of trailing identifier runes.
func identSuffixLen(rn []rune) int {
	n := 0
	for n < len(rn) && isIdentRune(rn[len(rn)-1-n]) {
		n++
	}
	return n
}

// completionContext determines the completion trigger kind from the text.
func completionContext(text string) *CompletionContext {
	seed := completionSeed(text)
	switch {
	case strings.HasSuffix(seed, "."):
		// just typed a selector: members of the qualifier
		return &CompletionContext{TriggerKind: CompleteTriggerCharacter, TriggerCharacter: "."}
	case seed != "":
		// typing an identifier: gopls only filters its full candidate
		// list when the request is a content-change trigger; Invoked
		// returns everything unfiltered.
		return &CompletionContext{TriggerKind: CompleteContentChange}
	}
	return &CompletionContext{TriggerKind: CompleteInvoked}
}

// Completions returns completions for the given text at position,
// querying the language server with a short timeout. It returns
// nothing when there is no seed (e.g., after punctuation or a space),
// as it is called after each keystroke: see [Backend.CompletionsExplicit].
func (b *Backend) Completions(ln *lines.Lines, text string, posLine, posChar int) complete.Matches {
	return b.completions(ln, text, posLine, posChar, false)
}

// CompletionsExplicit returns completions for the given text at
// position, for an explicit user request, which also returns
// completions when there is no seed.
func (b *Backend) CompletionsExplicit(ln *lines.Lines, text string, posLine, posChar int) complete.Matches {
	return b.completions(ln, text, posLine, posChar, true)
}

func (b *Backend) completions(ln *lines.Lines, text string, posLine, posChar int, explicit bool) complete.Matches {
	srv, ok := b.readyServer(ln)
	if !ok {
		return complete.Matches{}
	}
	seed := completionSeed(text)
	if seed == "" && !explicit {
		// nothing typed after punctuation or space: the editor offers
		// completions after every keystroke, and a list of every symbol
		// in scope after each comma, etc, is not useful.
		return complete.Matches{}
	}
	params := &CompletionParams{
		TextDocumentPositionParams: posParams(ln, textpos.Pos{Line: posLine, Char: posChar}),
		Context:                    completionContext(text),
	}
	// the editor asks for completions right after each keystroke, before
	// the debounced change sync, so the server must be synced first:
	srv.DidChange(params.TextDocument.URI, currentText(ln))
	list, err := srv.Completion(params)
	if err != nil || list == nil {
		return complete.Matches{}
	}
	lt := []rune(lineText(ln, posLine))
	posChar = min(posChar, len(lt))
	ms := complete.Matches{Seed: seed}
	edits := make(map[string]*complete.Edit, len(list.Items))
	for _, item := range list.Items {
		cp, ed := convertCompletion(lt, posLine, posChar, seed, item)
		ms.Matches = append(ms.Matches, cp)
		edits[cp.Text] = ed
	}
	b.mu.Lock()
	b.edits = edits
	b.mu.Unlock()
	return ms
}

// convertCompletion converts one LSP CompletionItem into a
// [complete.Completion] and the edit needed to apply it, for the given
// cursor line text (runes) and cursor position in editor units.
//
// The editor applies a completion by replacing the seed back from the
// cursor with the edit text, while the server's text replaces the range
// of its TextEdit (or, without one, the identifier before the cursor,
// per the spec), which can cover more or less of the seed: for example,
// gopls replaces only "S" of the seed "strings.S" with "Split". So the
// part of the seed that the server's text does not replace is prepended
// to it ("strings." + "Split").
func convertCompletion(lt []rune, posLine, posChar int, seed string, item CompletionItem) (complete.Completion, *complete.Edit) {
	text := item.InsertText
	if item.TextEdit != nil {
		text = item.TextEdit.NewText
	}
	if text == "" {
		text = item.Label
	}
	adj := 0
	if item.InsertTextFormat == InsertTextSnippet {
		text, adj = plainSnippet(text)
	}
	sr := []rune(seed)
	repl := identSuffixLen(sr) // seed runes replaced by the server's text
	fd := forwardDelete(lt, posChar)
	if te := item.TextEdit; te != nil && te.Range.Start.Line == posLine {
		line := string(lt)
		repl = posChar - lang.UTF16ToRune(line, te.Range.Start.Character)
		if te.Range.End.Line == posLine {
			if ed := lang.UTF16ToRune(line, te.Range.End.Character); ed > posChar {
				fd = ed - posChar
			}
		}
	} else if strings.HasPrefix(text, seed) {
		repl = len(sr) // the text includes the whole qualified seed
	}
	if item.Kind.IsCallable() && posChar+fd < len(lt) && lt[posChar+fd] == '(' {
		// replacing the name of a call that already has its arguments,
		// e.g., Index -> LastIndex: insert just the name.
		if i := strings.IndexByte(text, '('); i > 0 {
			text, adj = text[:i], 0
		}
	}
	repl = max(0, min(repl, len(sr)))
	text = string(sr[:len(sr)-repl]) + text
	cp := complete.Completion{
		Text:  text,
		Label: item.Label,
		Icon:  symbolToken(int(item.Kind)).Icon(), // kinds share numbering
		Desc:  completionDesc(item),
	}
	return cp, &complete.Edit{NewText: text, ForwardDelete: fd, CursorAdjust: adj}
}

// forwardDelete returns the number of identifier runes past the cursor,
// which a completion should overwrite, following the same logic as the
// golang CompleteEdit.
func forwardDelete(lt []rune, cursorPos int) int {
	n := 0
	for i := cursorPos; i < len(lt) && isIdentRune(lt[i]); i++ {
		n++
	}
	return n
}

// CompleteEdit returns the recorded edit for a selected completion,
// falling back on the completion text itself, which already includes
// any needed seed prefix, if the edit was not recorded.
func (b *Backend) CompleteEdit(ln *lines.Lines, text string, cursorPos int, comp complete.Completion, seed string) complete.Edit {
	b.mu.Lock()
	ed, has := b.edits[comp.Text]
	b.mu.Unlock()
	if has {
		return *ed
	}
	return complete.Edit{NewText: comp.Text, ForwardDelete: forwardDelete([]rune(text), cursorPos)}
}

// completionDesc returns a description for a completion item,
// combining detail and documentation.
func completionDesc(item CompletionItem) string {
	desc := item.Detail
	if item.Documentation != nil && item.Documentation.Value != "" {
		if desc != "" {
			desc += "\n\n"
		}
		if item.Documentation.Kind == "markdown" {
			desc += MarkdownText(item.Documentation.Value)
		} else {
			desc += item.Documentation.Value
		}
	}
	const maxDesc = 500
	if utf8.RuneCountInString(desc) > maxDesc {
		desc = string([]rune(desc)[:maxDesc]) + "…"
	}
	return desc
}

////////   Snippets

// plainSnippet converts snippet syntax into the plain text to insert,
// returning the text and the cursor adjustment in runes relative to its
// end (zero or negative). The cursor goes at the lowest-numbered tab
// stop ($1, ${1:placeholder}, ...), after any placeholder text, or else
// at the final $0 stop, as there is no support for stepping through
// tab stops. Placeholders and variable defaults insert their text,
// choices insert their first option, and other variables are empty.
// See https://microsoft.github.io/language-server-protocol/specification#snippet_syntax
func plainSnippet(snip string) (text string, cursorAdjust int) {
	sp := &snippetParser{src: []rune(snip), stops: make(map[int]int)}
	sp.parse(false)
	cur, best := -1, 0
	for n, pos := range sp.stops {
		if n > 0 && (cur < 0 || n < best) {
			cur, best = pos, n
		}
	}
	if pos, has := sp.stops[0]; has && cur < 0 {
		cur = pos
	}
	text = string(sp.out)
	if cur < 0 {
		return text, 0
	}
	return text, cur - len(sp.out)
}

// snippetParser converts snippet syntax into plain text.
type snippetParser struct {
	src []rune
	i   int
	out []rune

	// stops are the output positions of the tab stops, by number
	// (the first occurrence of each).
	stops map[int]int
}

// parse parses text until the end of the source, or until an
// unescaped '}' if nested in a placeholder.
func (sp *snippetParser) parse(nested bool) {
	for sp.i < len(sp.src) {
		c := sp.src[sp.i]
		switch {
		case c == '\\' && sp.i+1 < len(sp.src):
			sp.out = append(sp.out, sp.src[sp.i+1])
			sp.i += 2
		case c == '}' && nested:
			return
		case c == '$':
			sp.i++
			sp.dollar()
		default:
			sp.out = append(sp.out, c)
			sp.i++
		}
	}
}

// dollar parses a tab stop, placeholder, choice or variable,
// after the '$'.
func (sp *snippetParser) dollar() {
	if sp.i >= len(sp.src) || sp.src[sp.i] != '{' {
		name := sp.name()
		if name == "" {
			sp.out = append(sp.out, '$')
			return
		}
		sp.stop(name)
		return
	}
	sp.i++ // {
	name := sp.name()
	if sp.i < len(sp.src) {
		switch sp.src[sp.i] {
		case ':': // placeholder or variable default
			sp.i++
			sp.parse(true)
		case '|': // choice: insert the first option
			sp.i++
			sp.choice()
		case '/': // variable transform: not supported
			sp.skipBraced()
		}
	}
	if sp.i < len(sp.src) && sp.src[sp.i] == '}' {
		sp.i++
	}
	sp.stop(name)
}

// name parses a tab stop number or variable name.
func (sp *snippetParser) name() string {
	st := sp.i
	if sp.i < len(sp.src) && unicode.IsDigit(sp.src[sp.i]) {
		for sp.i < len(sp.src) && unicode.IsDigit(sp.src[sp.i]) {
			sp.i++
		}
		return string(sp.src[st:sp.i])
	}
	for sp.i < len(sp.src) && (sp.src[sp.i] == '_' || unicode.IsLetter(sp.src[sp.i]) || (sp.i > st && unicode.IsDigit(sp.src[sp.i]))) {
		sp.i++
	}
	return string(sp.src[st:sp.i])
}

// stop records a tab stop at the current output position,
// if the name is a number (not a variable).
func (sp *snippetParser) stop(name string) {
	n, err := strconv.Atoi(name)
	if err != nil {
		return
	}
	if _, has := sp.stops[n]; !has {
		sp.stops[n] = len(sp.out)
	}
}

// choice outputs the first option of a choice, consuming the rest
// through the closing '|'.
func (sp *snippetParser) choice() {
	first := true
	for sp.i < len(sp.src) && sp.src[sp.i] != '|' {
		c := sp.src[sp.i]
		switch {
		case c == '\\' && sp.i+1 < len(sp.src):
			sp.i++
			c = sp.src[sp.i]
		case c == ',':
			first = false
			sp.i++
			continue
		}
		if first {
			sp.out = append(sp.out, c)
		}
		sp.i++
	}
	sp.i++ // |
}

// skipBraced skips to the '}' that closes the current element.
func (sp *snippetParser) skipBraced() {
	depth := 0
	for ; sp.i < len(sp.src); sp.i++ {
		switch sp.src[sp.i] {
		case '\\':
			sp.i++
		case '{':
			depth++
		case '}':
			if depth == 0 {
				return
			}
			depth--
		}
	}
}
