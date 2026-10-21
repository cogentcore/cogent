// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lsp

import (
	"fmt"
	"slices"

	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/textpos"

	"cogentcore.org/cogent/code/lang"
)

// SemanticToken is a decoded semantic token in editor units:
// a rune-based line/char region with the legend type name and
// the legend modifier names.
type SemanticToken struct {
	Start      textpos.Pos
	End        textpos.Pos
	Type       string
	Modifiers  []string
	Deprecated bool
}

// SemanticTokensFull returns the decoded semantic tokens for the given
// buffer, in editor (rune) units. Returns nil if the server does not
// support semantic tokens or the request fails.
func (b *Backend) SemanticTokensFull(ln *lines.Lines) ([]SemanticToken, error) {
	srv, ok := b.readyServer(ln)
	if !ok {
		return nil, nil
	}
	legend := srv.SemanticTokensLegend()
	if legend == nil {
		return nil, nil
	}
	st, err := srv.SemanticTokensFull(docID(ln).URI)
	if err != nil || st == nil {
		return nil, err
	}
	return decodeSemanticTokens(ln, legend, st)
}

// decodeSemanticTokens converts the relative-encoded token data into
// absolute editor-unit tokens. Data is five ints per token:
// deltaLine, deltaStartUTF16, length, type, modifiers, where the first
// two are relative to the previous token's start.
func decodeSemanticTokens(ln *lines.Lines, legend *SemanticTokensLegend, st *SemanticTokens) ([]SemanticToken, error) {
	if len(st.Data)%5 != 0 {
		return nil, fmt.Errorf("lsp: semanticTokens data length %d not a multiple of 5", len(st.Data))
	}
	out := make([]SemanticToken, 0, len(st.Data)/5)
	line, char16 := 0, 0
	for d := range slices.Chunk(st.Data, 5) {
		line += d[0]
		if d[0] == 0 {
			char16 += d[1]
		} else {
			char16 = d[1]
		}
		if !ln.IsValidLine(line) {
			continue
		}
		txt := string(ln.Line(line))
		typ := ""
		if d[3] >= 0 && d[3] < len(legend.TokenTypes) {
			typ = legend.TokenTypes[d[3]]
		}
		var mods []string
		for mi, bits := 0, d[4]; mi < len(legend.TokenModifiers) && bits > 0; mi, bits = mi+1, bits>>1 {
			if bits&1 != 0 {
				mods = append(mods, legend.TokenModifiers[mi])
			}
		}
		out = append(out, SemanticToken{
			Start:      textpos.Pos{Line: line, Char: lang.UTF16ToRune(txt, char16)},
			End:        textpos.Pos{Line: line, Char: lang.UTF16ToRune(txt, char16+d[2])},
			Type:       typ,
			Modifiers:  mods,
			Deprecated: slices.Contains(mods, "deprecated"),
		})
	}
	return out, nil
}
