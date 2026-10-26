// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"cogentcore.org/core/icons"
	"cogentcore.org/core/text/parse/syms"
	"cogentcore.org/core/text/token"
	"os"
	"path/filepath"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/text/textpos"

	"cogentcore.org/cogent/code/lang"

	"cogentcore.org/core/text/lines"
	"github.com/stretchr/testify/assert"
)

// TestDocsMarkup checks the rendering of gopls hover markdown for a
// struct field: highlighted code, a rule, the text, and a link.
func TestDocsMarkup(t *testing.T) {
	md := "```go\nfield FPath string `edit:\"-\" display:\"-\" table:\"-\"`\n```\n\n---\n\nfull path to file\n\n\n---\n\n[`(cdebug.Break).FPath` on pkg.go.dev](https://pkg.go.dev/cogentcore.org/cogent/code/cdebug#Break.FPath)"
	buf := lines.NewLines()
	setDocsHighlighting(buf)
	lns, mus := docsMarkup(buf, md)
	var texts []string
	for _, ln := range lns {
		texts = append(texts, string(ln))
	}
	assert.Equal(t, []string{
		"field FPath string `edit:\"-\" display:\"-\" table:\"-\"`",
		"────────────────────────────────────────",
		"full path to file",
		"────────────────────────────────────────",
		"(cdebug.Break).FPath on pkg.go.dev",
	}, texts)
	assert.Greater(t, mus[0].NumSpans(), 3, "code is highlighted")
	for i := range mus {
		assert.Equal(t, texts[i], string(mus[i].Join()), "markup text matches line %d", i)
	}
	links := mus[4].GetLinks()
	if assert.Len(t, links, 1) {
		assert.Equal(t, "https://pkg.go.dev/cogentcore.org/cogent/code/cdebug#Break.FPath", links[0].URL)
	}
}

func TestDocumentTitle(t *testing.T) {
	base := "http://127.0.0.1:64180/gopls/cQanU89rn7Y"
	assert.Equal(t, "Assembler: main.main", documentTitle(base+"/assembly?pkg=demo&view=1#main.main", "arm64 assembly for main.main"))
	assert.Equal(t, "Documentation: fmt", documentTitle(base+"/pkg/fmt?view=1#Println", "fmt package - fmt - Gopls packages"))
	assert.Equal(t, "Free symbols: main.go", documentTitle(base+"/freesymbols?file=file%3A%2F%2F%2Fx%2Fa%2520dir%2Fmain.go&range=9:5:9:12", ""))
	assert.Equal(t, "Document: other page", documentTitle(base+"/other", "other page"))
}

// TestLocationsMarkup checks the References / Problems tab text:
// grouped by file, with links for the locations.
func TestLocationsMarkup(t *testing.T) {
	cv := NewCode()
	dir := t.TempDir()
	cv.Settings.ProjectRoot = core.Filename(dir)
	fa := filepath.Join(dir, "a.go")
	os.WriteFile(fa, []byte("package a\n\tvar x = y\n"), 0644)
	items := []locationItem{
		{loc: lang.Location{Filename: fa, Region: textpos.Region{Start: textpos.Pos{Line: 1, Char: 9}, End: textpos.Pos{Line: 1, Char: 10}}}},
		{loc: lang.Location{Filename: fa, Region: textpos.Region{Start: textpos.Pos{Line: 0, Char: 8}, End: textpos.Pos{Line: 0, Char: 9}}}, sev: "error", msg: "bad"},
	}
	sortLocationItems(items)
	buf := lines.NewLines()
	lns, mus, _ := locationsMarkup(cv, buf.FontStyle(), "Header", items)
	var texts []string
	for i, ln := range lns {
		texts = append(texts, string(ln))
		assert.Equal(t, string(ln), string(mus[i].Join()), "markup text matches line %d", i)
	}
	assert.Equal(t, []string{"Header", "", "a.go: 2", "\ta.go:1:9: error: bad", "\ta.go:2:10:  var x = y"}, texts)
	links := mus[4].GetLinks()
	if assert.Len(t, links, 1) {
		assert.Equal(t, "location:1", links[0].URL)
	}
}

func TestFlattenSymbols(t *testing.T) {
	sl := []lang.Symbol{{Name: "T", Children: []lang.Symbol{{Name: "F"}, {Name: "M"}}}, {Name: "main"}}
	var names []string
	for _, sy := range flattenSymbols(sl, "") {
		names = append(names, sy.Name)
		assert.Empty(t, sy.Children)
	}
	assert.Equal(t, []string{"T", "T.F", "T.M", "main"}, names)
}

func TestNestMethods(t *testing.T) {
	sl := []lang.Symbol{
		{Name: "T", Children: []lang.Symbol{{Name: "F"}}},
		{Name: "(*T).M", Detail: "func()"},
		{Name: "(G[K]).N"},
		{Name: "G"},
		{Name: "(*Other).X"}, // type not among the symbols: kept as is
		{Name: "main"},
	}
	out := nestMethods(sl)
	var names []string
	for _, sy := range flattenSymbols(out, "") {
		names = append(names, sy.Name)
	}
	assert.Equal(t, []string{"T", "T.F", "T.M", "G", "G.N", "(*Other).X", "main"}, names)
	assert.Equal(t, "func()", out[0].Children[1].Detail, "method detail kept")
}

func TestSymNodeIcons(t *testing.T) {
	for kind, ic := range map[token.Tokens]icons.Icon{
		token.NameType: icons.Title, token.NameStruct: icons.Title, token.NameInterface: icons.Title,
		token.NameConstant: icons.Constant, token.NameField: icons.Field, token.NameMethod: icons.Method,
		token.NameFunction: icons.Function, token.NameVar: icons.Variable,
	} {
		sn := NewSymNode().SetSymbol(syms.Symbol{Kind: kind})
		assert.Equal(t, ic, sn.GetIcon(), kind.String())
	}
}

func TestIgnoreAndMergeProblems(t *testing.T) {
	at := func(line int) textpos.Region { return textpos.Region{Start: textpos.Pos{Line: line}} }
	unkeyed := lang.Diagnostic{Region: at(1), Msg: "image.Point struct literal uses unkeyed fields", Severity: lang.DiagWarn}
	typeErr := lang.Diagnostic{Region: at(2), Msg: "undefined: x", Severity: lang.DiagErr}
	other := lang.Diagnostic{Region: at(3), Msg: "undefined: y", Severity: lang.DiagErr}

	diags := []lang.Diagnostic{unkeyed, typeErr}
	assert.Equal(t, diags, filterIgnored(diags, nil))
	assert.Equal(t, []lang.Diagnostic{typeErr}, filterIgnored(diags, []string{"uses unkeyed fields"}))
	assert.Len(t, diags, 2, "filterIgnored must not modify its argument")

	// pulled has everything at the time; published adds a newer error:
	assert.Equal(t, []lang.Diagnostic{unkeyed, typeErr, other}, mergeDiags(diags, []lang.Diagnostic{typeErr, other}))

	var se ProjectSettings
	se.addStandardIgnores()
	assert.Len(t, se.IgnoreProblems, len(StandardIgnoreProblems))
	assert.False(t, se.IgnoreProblems["struct literal uses unkeyed fields"])
	se.IgnoreProblems["struct literal uses unkeyed fields"] = true
	se.addStandardIgnores()
	assert.True(t, se.IgnoreProblems["struct literal uses unkeyed fields"], "user setting kept")

	assert.False(t, showSeverity(lang.DiagHint, false))
	assert.True(t, showSeverity(lang.DiagHint, true))
	assert.True(t, showSeverity(lang.DiagWarn, false))
	assert.Equal(t, "1 error, 0 warnings, 2 hints", diagSummaryText([]locationItem{{sev: "error"}, {sev: "hint"}, {sev: "info"}}))
}
