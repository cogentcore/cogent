// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lsp

import (
	"slices"
	"strings"

	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/textpos"
)

// signatureTriggerChars are the characters that trigger signature help
// in most language servers; gopls uses "(" and ",".
var signatureTriggerChars = []rune{'(', ',', '<'}

// IsSignatureTrigger reports whether the last character of the given
// text is one that should trigger a signature help request.
func IsSignatureTrigger(text string) bool {
	rn := []rune(text)
	return len(rn) > 0 && slices.Contains(signatureTriggerChars, rn[len(rn)-1])
}

// signatureContext determines the signature help trigger kind from the
// text before the cursor and whether signature help was already showing
// (retrigger).
func signatureContext(text string, wasShowing bool) *SignatureHelpContext {
	ctx := &SignatureHelpContext{IsRetrigger: wasShowing}
	switch {
	case IsSignatureTrigger(text):
		rn := []rune(text)
		ctx.TriggerKind = SigHelpTriggerChar
		ctx.TriggerCharacter = string(rn[len(rn)-1])
	case wasShowing:
		ctx.TriggerKind = SigHelpContentChange
	default:
		ctx.TriggerKind = SigHelpInvoked
	}
	return ctx
}

// activeParameter computes the index of the parameter the cursor is at,
// by counting top-level commas since the innermost unmatched open paren
// before the cursor. Returns -1 if the cursor is not inside a call.
// This is the standard client-side computation; some servers also
// provide ActiveParameter, which takes precedence when set.
func activeParameter(text string, charPos int) int {
	rn := []rune(text)
	charPos = min(charPos, len(rn))
	depth := 0
	paren := -1
	for i := charPos - 1; i >= 0 && paren < 0; i-- {
		switch rn[i] {
		case ')', ']', '}':
			depth++
		case '(', '[', '{':
			switch {
			case depth > 0:
				depth--
			case rn[i] == '(':
				paren = i
			default:
				return -1 // crossed a non-paren bracket: not in a call
			}
		}
	}
	if paren < 0 {
		return -1
	}
	idx := 0
	depth = 0
	for i := paren + 1; i < charPos; i++ {
		switch rn[i] {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
		case ',':
			if depth == 0 {
				idx++
			}
		case '"', '\'', '`': // skip string literal
			q := rn[i]
			for i++; i < charPos && rn[i] != q; i++ {
				if rn[i] == '\\' {
					i++
				}
			}
		}
	}
	return idx
}

// SignatureHelp returns the signature help for the given editor
// position, and the index of the active signature, or nil if the server
// has nothing to show. wasShowing indicates that signature help was
// already being shown, so that the server can do incremental updates.
// The returned ActiveParameter is always set, within the range of the
// active signature's parameters, using the client-side computation if
// the server does not provide it.
func (b *Backend) SignatureHelp(ln *lines.Lines, pos textpos.Pos, wasShowing bool) (*SignatureHelp, int, error) {
	srv, ok := b.readyServer(ln)
	if !ok {
		return nil, -1, nil
	}
	text := lineText(ln, pos.Line)
	rn := []rune(text)
	params := &SignatureHelpParams{
		TextDocumentPositionParams: posParams(ln, pos),
		Context:                    signatureContext(string(rn[:min(pos.Char, len(rn))]), wasShowing),
	}
	sh, err := srv.SignatureHelp(params)
	if err != nil || sh == nil {
		return nil, -1, err
	}
	act := 0
	if sh.ActiveSignature != nil && *sh.ActiveSignature >= 0 && *sh.ActiveSignature < len(sh.Signatures) {
		act = *sh.ActiveSignature
	}
	ap := max(activeParameter(text, pos.Char), 0)
	if sh.ActiveParameter != nil && *sh.ActiveParameter >= 0 {
		ap = *sh.ActiveParameter
	}
	ap = max(min(ap, len(sh.Signatures[act].Parameters)-1), 0)
	sh.ActiveParameter = &ap
	return sh, act, nil
}

// SignatureText formats the given signature of the signature help into
// a single-line display string, with the active parameter marked using
// the given prefix, suffix markers.
func SignatureText(sh *SignatureHelp, sig, activeParam int, mark, unmark string) string {
	if sh == nil || sig < 0 || sig >= len(sh.Signatures) {
		return ""
	}
	si := &sh.Signatures[sig]
	if activeParam < 0 || activeParam >= len(si.Parameters) {
		return si.Label
	}
	var sb strings.Builder
	prev := 0
	for i, p := range si.Parameters {
		txt := p.LabelText(si.Label)
		off := strings.Index(si.Label[prev:], txt)
		if txt == "" || off < 0 {
			return si.Label // label form doesn't match; show it plain
		}
		st := prev + off
		sb.WriteString(si.Label[prev:st])
		if i == activeParam {
			sb.WriteString(mark + txt + unmark)
		} else {
			sb.WriteString(txt)
		}
		prev = st + len(txt)
	}
	sb.WriteString(si.Label[prev:])
	return sb.String()
}

// ActiveParam returns the active parameter index of the signature
// help, as set by [Backend.SignatureHelp], or 0 if not set.
func (sh *SignatureHelp) ActiveParam() int {
	if sh == nil || sh.ActiveParameter == nil {
		return 0
	}
	return *sh.ActiveParameter
}
