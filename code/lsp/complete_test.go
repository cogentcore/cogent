// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lsp

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/parse/complete"
	"cogentcore.org/core/text/textpos"
	"github.com/stretchr/testify/assert"
)

func TestCompletionSeedQualified(t *testing.T) {
	for _, tc := range []struct{ text, want string }{
		{"\tx := core.F", "core.F"},
		{"\tx := core.", "core."},
		{"\tx := F", "F"},
		{"\tx := a.b.c.Fr", "a.b.c.Fr"},
		{"\tf(", ""},
		{"\tx := m.Foo(a.b", "a.b"},
		{"\t// comment word", "word"},
	} {
		if got := completionSeed(tc.text); got != tc.want {
			t.Errorf("completionSeed(%q) = %q, want %q", tc.text, got, tc.want)
		}
	}
}

// TestCompletionQualified verifies the core bug: typing "strings.S" and
// selecting the "Split" completion must produce "strings.Split", keeping
// the package qualifier, not overwrite everything with "Split".
func TestCompletionQualified(t *testing.T) {
	if _, err := exec.LookPath("gopls"); err != nil {
		t.Skip("gopls not found")
	}
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module demo\n\ngo 1.21\n"), 0644)
	srcPath := filepath.Join(tmp, "main.go")
	valid := "package main\n\nimport \"strings\"\n\nfunc main() {\n\tx := strings.Split(\"a,b\", \",\")\n\t_ = x\n}\n"
	os.WriteFile(srcPath, []byte(valid), 0644)
	t.Setenv("GOWORK", "off")
	b := NewBackend(nil)
	b.RootFunc = func(string) string { return tmp }
	t.Cleanup(b.Stop)
	go func() {
		for range b.Diagnostics() {
		}
	}()
	ln := lines.NewLines()
	ln.SetFilename(srcPath)
	ln.SetText([]byte(valid))
	b.Open(ln)
	waitUntil(t, 60*time.Second, "ready", func() bool { return b.Ready(ln) })
	waitUntil(t, 60*time.Second, "loaded", func() bool {
		return b.Hover(ln, textpos.Pos{Line: 2, Char: 10}) != ""
	})

	// simulate typing: replace the Split call with "strings.S" and sync.
	// The buffer is temporarily invalid Go; gopls rebuilds its snapshot
	// asynchronously, so poll with the same request the editor makes.
	typed := "package main\n\nimport \"strings\"\n\nfunc main() {\n\tx := strings.S\n\t_ = x\n}\n"
	ln.SetText([]byte(typed))
	b.Change(ln)

	line := "\tx := strings.S"
	lnNum := 5 // the line with the call, in editor line numbers
	posChar := len([]rune(line))

	waitUntil(t, 30*time.Second, "Split in completions", func() bool {
		ms := b.Completions(ln, line, lnNum, posChar)
		for _, m := range ms.Matches {
			if m.Label == "Split" {
				return true
			}
		}
		for i, m := range ms.Matches {
			if i < 4 {
				t.Logf("poll %d: label=%q text=%q", i, m.Label, m.Text)
			}
		}
		return false
	})

	ms := b.Completions(ln, line, lnNum, posChar)
	if ms.Seed != "strings.S" {
		t.Fatalf("seed = %q, want %q", ms.Seed, "strings.S")
	}
	var comp *complete.Completion
	for i := range ms.Matches {
		if ms.Matches[i].Label == "Split" {
			comp = &ms.Matches[i]
			break
		}
	}
	if comp == nil {
		for i := range ms.Matches {
			t.Logf("match %d: label=%q text=%q", i, ms.Matches[i].Label, ms.Matches[i].Text)
		}
		t.Fatal("no Split completion found")
	}
	// simulate the editor: replace the seed back from the cursor with
	// the edit text and check the resulting line:
	t.Logf("comp.Text=%q", comp.Text)
	ed := b.CompleteEdit(ln, line, posChar, *comp, ms.Seed)
	t.Logf("ed.NewText=%q fd=%d", ed.NewText, ed.ForwardDelete)
	rn := []rune(line)
	st := posChar - len([]rune(ms.Seed))
	out := string(rn[:st]) + ed.NewText + string(rn[posChar+ed.ForwardDelete:])
	want := "\tx := strings.Split"
	if !strings.HasPrefix(out, want) {
		t.Errorf("after completion: %q, want prefix %q", out, want)
	}
	t.Logf("result: %q", out)

}

func TestConvertCompletion(t *testing.T) {
	edit := func(line, st, ed int, text string) *TextEdit {
		return &TextEdit{Range: Range{Start: Position{Line: line, Character: st}, End: Position{Line: line, Character: ed}}, NewText: text}
	}
	for _, tc := range []struct {
		name, line string
		item       CompletionItem
		want       string
		fd, adj    int
	}{
		{"gopls member", "\tx := strings.S", CompletionItem{Label: "Split", TextEdit: edit(0, 14, 15, "Split")}, "strings.Split", 0, 0},
		{"after dot", "\tx := core.", CompletionItem{Label: "Frame", TextEdit: edit(0, 11, 11, "Frame")}, "core.Frame", 0, 0},
		{"no edit, word only", "\tx := strings.S", CompletionItem{Label: "Split"}, "strings.Split", 0, 0},
		{"no edit, qualified", "\tx := strings.S", CompletionItem{Label: "strings.Split"}, "strings.Split", 0, 0},
		{"replaces whole seed", "\tx := strings.S", CompletionItem{Label: "x", TextEdit: edit(0, 6, 15, "bytes.Split")}, "bytes.Split", 0, 0},
		{"forward delete", "\tx := strings.Spl(a)", CompletionItem{Label: "Split", TextEdit: edit(0, 14, 15, "Split")}, "strings.Split", 2, 0},
		{"edit range past cursor", "\tx := strings.Spl", CompletionItem{Label: "Split", TextEdit: edit(0, 14, 17, "Split")}, "strings.Split", 2, 0},
		{"snippet", "\tfmt.Pr", CompletionItem{Label: "Println", InsertTextFormat: InsertTextSnippet, TextEdit: edit(0, 5, 7, "Println(${1:a ...any})")}, "fmt.Println(a ...any)", 0, -1},
		{"non-ascii before", "\tü := strings.S", CompletionItem{Label: "Split", TextEdit: edit(0, 14, 15, "Split")}, "strings.Split", 0, 0},
	} {
		lt := []rune(tc.line)
		posChar := len(lt)
		if tc.fd > 0 { // cursor before the trailing identifier runes
			posChar = len([]rune("\tx := strings.S"))
		}
		seed := completionSeed(string(lt[:posChar]))
		cp, ed := convertCompletion(lt, 0, posChar, seed, tc.item)
		assert.Equal(t, tc.want, cp.Text, tc.name)
		assert.Equal(t, tc.want, ed.NewText, tc.name)
		assert.Equal(t, tc.fd, ed.ForwardDelete, tc.name)
		assert.Equal(t, tc.adj, ed.CursorAdjust, tc.name)
	}
}

// TestConvertCompletionExistingCall checks that completing the name of
// a call that already has its arguments does not insert them again.
func TestConvertCompletionExistingCall(t *testing.T) {
	line := []rune("\tx := strings.Index(s, \"find\")")
	posChar := len([]rune("\tx := strings.In")) // cursor in the name
	seed := completionSeed(string(line[:posChar]))
	item := CompletionItem{Label: "LastIndex", Kind: CompletionFunction, InsertTextFormat: InsertTextSnippet,
		TextEdit: &TextEdit{Range: Range{Start: Position{Character: 14}, End: Position{Character: 19}}, NewText: "LastIndex(${1:s string}, ${2:substr string})"}}
	_, ed := convertCompletion(line, 0, posChar, seed, item)
	assert.Equal(t, "strings.LastIndex", ed.NewText)
	assert.Equal(t, 3, ed.ForwardDelete) // "dex"
	assert.Equal(t, 0, ed.CursorAdjust)
	// without a following paren, the arguments are inserted:
	line = []rune("\tx := strings.In")
	_, ed = convertCompletion(line, 0, len(line), seed, item)
	assert.Equal(t, "strings.LastIndex(s string, substr string)", ed.NewText)
}

func TestPlainSnippet(t *testing.T) {
	for _, tc := range []struct {
		snip, text string
		adj        int
	}{
		{"Println()", "Println()", 0},
		{"Println($0)", "Println()", -1},
		{"Println(${1:a ...any})", "Println(a ...any)", -1},
		{"f(${1:a}, ${2:b})$0", "f(a, b)", -4},
		{"f(${2:a}, ${1:b})", "f(a, b)", -1},
		{"for ${1:i} := range ${2:x} {\n\t$0\n}", "for i := range x {\n\t\n}", -17},
		{"${1:outer ${2:inner}}", "outer inner", 0},
		{"cost \\$5 \\}", "cost $5 }", 0},
		{"${1|one,two|}", "one", 0},
		{"$TM_FILENAME ${TM_LINE:dflt}", " dflt", 0},
		{"a $ b", "a $ b", 0},
		{"π(${1:x})", "π(x)", -1},
	} {
		text, adj := plainSnippet(tc.snip)
		assert.Equal(t, tc.text, text, tc.snip)
		assert.Equal(t, tc.adj, adj, tc.snip)
	}
}

func TestDocText(t *testing.T) {
	src := "package p\r\n\nvar q = \"\U0001F600\" + smileX\n"
	dt := newDocText(src)
	assert.Equal(t, []string{"package p", "", "var q = \"\U0001F600\" + smileX", ""}, dt.lines)
	// smileX starts at utf-16 15, rune 14 (the emoji is 2 utf-16 units)
	r := Range{Start: Position{Line: 2, Character: 15}, End: Position{Line: 2, Character: 21}}
	assert.Equal(t, textpos.Region{Start: textpos.Pos{Line: 2, Char: 14}, End: textpos.Pos{Line: 2, Char: 20}}, dt.region(r))
	assert.Equal(t, "smileX", dt.rangeText(r))
	// end of document positions:
	assert.Equal(t, len(src), dt.offset(Position{Line: 3, Character: 0}))
	assert.Equal(t, len(src), dt.offset(Position{Line: 99, Character: 0}))

	out := dt.apply([]TextEdit{
		{Range: r, NewText: "happy"},
		{Range: Range{Start: Position{Line: 0, Character: 8}, End: Position{Line: 0, Character: 9}}, NewText: "x"},
		// insertions at the same position apply in the given order:
		{Range: Range{Start: Position{Line: 1, Character: 0}, End: Position{Line: 1, Character: 0}}, NewText: "A"},
		{Range: Range{Start: Position{Line: 1, Character: 0}, End: Position{Line: 1, Character: 0}}, NewText: "B"},
	})
	assert.Equal(t, "package x\r\nAB\nvar q = \"\U0001F600\" + happy\n", out)
}

func TestApplyTextEditsOrder(t *testing.T) {
	ln := lines.NewLines()
	ln.SetText([]byte("a\nb\n"))
	ins := func(line, ch int, text string) TextEdit {
		return TextEdit{Range: Range{Start: Position{Line: line, Character: ch}, End: Position{Line: line, Character: ch}}, NewText: text}
	}
	applyTextEdits(ln, []TextEdit{ins(1, 0, "1"), ins(0, 1, "X"), ins(1, 0, "2")})
	assert.Equal(t, "aX\n12b\n", ln.String())
	// deletion through the end of the document:
	applyTextEdits(ln, []TextEdit{{Range: Range{Start: Position{Line: 0, Character: 2}, End: Position{Line: 2, Character: 0}}}})
	assert.Equal(t, "aX\n", ln.String())
}

func TestURIFromPath(t *testing.T) {
	p := filepath.Join(t.TempDir(), "a dir", "f#1.go")
	uri := URIFromPath(p)
	assert.False(t, strings.ContainsAny(string(uri), " #"), uri)
	assert.Equal(t, p, uri.Path())
	assert.Equal(t, "/x/y z", DocumentURI("file:///x/y%20z").Path())
	assert.Equal(t, "", DocumentURI("untitled:foo").Path())
}

// TestCompletionExistingCallGopls renames a call with completion, with
// gopls: the existing arguments must be kept, not duplicated.
func TestCompletionExistingCallGopls(t *testing.T) {
	src := "package main\n\nimport \"strings\"\n\nfunc main() {\n\t_ = strings.Index(\"a\", \"find\")\n}\n"
	b, ln, _ := goplsSetup(t, src)
	line := "\t_ = strings.Index(\"a\", \"find\")"
	posChar := len([]rune("\t_ = strings.In"))
	var ms complete.Matches
	var comp *complete.Completion
	waitUntil(t, 30*time.Second, "LastIndex in completions", func() bool {
		ms = b.Completions(ln, line[:posChar], 5, posChar)
		for i := range ms.Matches {
			if ms.Matches[i].Label == "LastIndex" {
				comp = &ms.Matches[i]
				return true
			}
		}
		return false
	})
	ed := b.CompleteEdit(ln, line, posChar, *comp, ms.Seed)
	rn := []rune(line)
	out := string(rn[:posChar-len([]rune(ms.Seed))]) + ed.NewText + string(rn[posChar+ed.ForwardDelete:])
	assert.Equal(t, "\t_ = strings.LastIndex(\"a\", \"find\")", out)
}

// TestCompletionCallGopls checks the text inserted for a function
// completion with gopls: the call parens without parameter placeholders,
// which cannot be selected in the editor, with the cursor inside them.
func TestCompletionCallGopls(t *testing.T) {
	src := "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Prin\n}\n"
	b, ln, _ := goplsSetup(t, src)
	line := "\tfmt.Prin"
	posChar := len([]rune(line))
	var ms complete.Matches
	var comp *complete.Completion
	waitUntil(t, 30*time.Second, "Println in completions", func() bool {
		ms = b.Completions(ln, line, 5, posChar)
		for i := range ms.Matches {
			if ms.Matches[i].Label == "Println" {
				comp = &ms.Matches[i]
				return true
			}
		}
		return false
	})
	assert.Contains(t, comp.Desc, "Println formats using the default formats")
	ed := b.CompleteEdit(ln, line, posChar, *comp, ms.Seed)
	assert.Equal(t, "fmt.Println()", ed.NewText)
	assert.Equal(t, -1, ed.CursorAdjust)
}
