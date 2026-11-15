// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lsp

import (
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"cogentcore.org/cogent/code/lang"
	"cogentcore.org/core/text/textpos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// navSrc is the program for the tests of navigation with gopls.
const navSrc = `package main

import (
	"fmt"
	"os"
)

type Shape interface{ Area() float64 }

type Sq struct{ S float64 }

func (s Sq) Area() float64 { return s.S * s.S }

func area(sh Shape) float64 {
	return sh.Area()
}

func main() {
	sq := Sq{S: 2}
	fmt.Println(area(sq), area(sq))
}
`

// srcPos returns the position of the n-th (from 0) occurrence of the given
// text in src, plus the given number of characters.
func srcPos(src, text string, n, plus int) textpos.Pos {
	i := -1
	for range n + 1 {
		j := strings.Index(src[i+1:], text)
		if j < 0 {
			panic("srcPos: no " + text)
		}
		i += j + 1
	}
	ln := strings.Count(src[:i], "\n")
	return textpos.Pos{Line: ln, Char: i - (strings.LastIndex(src[:i], "\n") + 1) + plus}
}

// TestNavigationGopls tests the navigation requests of the backend with
// gopls: definitions, types, implementations, references, occurrences,
// symbols, and selection ranges.
func TestNavigationGopls(t *testing.T) {
	b, ln, srcPath := goplsSetup(t, navSrc)
	lineOf := func(text string) int { return srcPos(navSrc, text, 0, 0).Line }
	only := func(locs []lang.Location) lang.Location {
		t.Helper()
		require.Len(t, locs, 1, "%v", locs)
		return locs[0]
	}

	def := only(b.Definition(ln, srcPos(navSrc, "area(sq)", 0, 2)))
	assert.Equal(t, srcPath, def.Filename)
	assert.Equal(t, lineOf("func area"), def.Region.Start.Line)
	assert.Equal(t, lineOf("type Sq"), only(b.TypeDefinition(ln, srcPos(navSrc, "sq)", 0, 1))).Region.Start.Line)
	assert.Equal(t, lineOf("type Sq"), only(b.Implementations(ln, srcPos(navSrc, "Shape interface", 0, 2))).Region.Start.Line)

	refs := b.References(ln, srcPos(navSrc, "area(sq)", 0, 2))
	assert.Len(t, refs, 3, "the declaration and the two calls")
	occ := b.Occurrences(ln, srcPos(navSrc, "sq :=", 0, 1))
	assert.Len(t, occ, 3, "sq and its two uses")
	assert.Empty(t, b.Occurrences(ln, textpos.Pos{Line: 1}), "nothing at a blank line")

	ws := b.WorkspaceSymbols(ln, "Sq")
	assert.True(t, slices.ContainsFunc(ws, func(s lang.Symbol) bool { return s.Name == "Sq" }), "%v", ws)

	sel := b.SelectionRanges(ln, srcPos(navSrc, "s.S * s.S", 0, 2))
	require.Greater(t, len(sel), 2)
	assert.True(t, sel[1].Start.Char <= sel[0].Start.Char, "growing ranges")

	hints, err := b.InlayHintTexts(ln, textpos.Region{End: textpos.Pos{Line: ln.NumLines()}})
	assert.NoError(t, err)
	_ = hints // (none without the gopls settings)

	m := b.CompletionsExplicit(ln, "\tfmt.", srcPos(navSrc, "fmt.Println", 0, 0).Line, 5)
	assert.NotEmpty(t, m.Matches, "the members of fmt")

	// organizing the imports, for the unused import
	waitUntil(t, 30*time.Second, "the unused import diagnostic", func() bool {
		fd := b.FileDiagnostics(ln, []string{srcPath})
		return len(fd) > 0 && len(fd[0].Diags) > 0
	})
	var diags []lang.Diagnostic
	for _, ds := range b.FileDiagnostics(ln, []string{srcPath}) {
		diags = append(diags, ds.Diags...)
	}
	i := slices.IndexFunc(diags, func(d lang.Diagnostic) bool { return strings.Contains(d.Msg, `"os" imported and not used`) })
	require.GreaterOrEqual(t, i, 0, "%v", diags)
	acts, err := b.CodeActionsQuick(ln, diags[i].Region, diags[i:i+1])
	require.NoError(t, err)
	j := slices.IndexFunc(acts, func(ca CodeAction) bool { return ca.Kind == "source.organizeImports" })
	require.GreaterOrEqual(t, j, 0, "%v", acts)
	assert.NotEmpty(t, acts[j].Description())
	_, err = b.RunCodeAction(ln, &acts[j])
	require.NoError(t, err)
	assert.NotContains(t, ln.String(), `"os"`, "the import is removed")

	// formatting
	ln.InsertText(textpos.Pos{Line: lineOf("func main")}, []rune("var   x = 1\n"))
	b.Change(ln)
	assert.True(t, b.Format(ln))
	assert.Contains(t, ln.String(), "var x = 1\n")
	require.NoError(t, os.WriteFile(srcPath, []byte(ln.String()), 0o644))
	b.Save(ln)
	assert.True(t, SameFile(ln.Filename(), srcPath))
	assert.False(t, SameFile(ln.Filename(), srcPath+"x"))
	assert.NotNil(t, b.AppliedEdits())
}
