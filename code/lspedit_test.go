// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/parse/complete"
	"cogentcore.org/core/text/textpos"
	"github.com/stretchr/testify/assert"
)

// editorTest is a real [TextEditor] driven by key events, wired to the
// language service the same way as in a Code window, for testing the
// interactive language features (completion, signature help) as the
// user experiences them. Run with -v to see each step.
type editorTest struct {
	t  *testing.T
	ls *LangService
	ed *TextEditor

	// matches are the completions from the most recent MatchFunc call
	// made by the editor's completer, and seed is the seed for them.
	matches []complete.Completion
	seed    string
	nmatch  int // number of MatchFunc calls
}

// newEditorTest returns a new editorTest for a Go file with the given
// source in a temporary module, with the cursor at the given position,
// once gopls has loaded the package.
func newEditorTest(t *testing.T, src string, pos textpos.Pos) *editorTest {
	if _, err := exec.LookPath("gopls"); err != nil {
		t.Skip("gopls not found in PATH:", err)
	}
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module demo\n\ngo 1.21\n"), 0644); err != nil {
		t.Fatal(err)
	}
	fpath := filepath.Join(tmp, "main.go")
	if err := os.WriteFile(fpath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOWORK", "off")

	et := &editorTest{t: t}
	cv := NewCode()
	et.ls = NewLangService(cv)
	cv.langSvc = et.ls
	t.Cleanup(et.ls.Stop)

	b := core.NewBody()
	et.ed = NewTextEditor(b)
	et.ed.Code = cv
	ln := lines.NewLines()
	ln.SetFilename(fpath)
	ln.SetText([]byte(src))
	et.ed.SetLines(ln)
	cv.OpenFiles.Add(ln) // for diagnostics
	// as in Code:
	et.ed.OnInput(func(e events.Event) { et.ls.EditorInput(et.ed) })
	if et.ed.Complete == nil {
		et.ed.Complete = core.NewComplete()
	}
	et.ls.Activated(et.ed)
	match := et.ed.Complete.MatchFunc
	et.ed.Complete.MatchFunc = func(data any, text string, posLine, posChar int) complete.Matches {
		ms := match(data, text, posLine, posChar)
		et.matches, et.seed = ms.Matches, ms.Seed
		et.nmatch++
		return ms
	}

	waitCode(t, 60*time.Second, "gopls loaded", func() bool {
		return len(et.ls.lspB.Symbols(ln)) > 0
	})
	et.ed.SetCursorShow(pos)
	return et
}

// typeText types the given text into the editor, one key event per
// rune, running the deferred functions after each, as the event loop
// does after the next render.
func (et *editorTest) typeText(text string) {
	for _, r := range text {
		et.ls.ScheduleSigHelp(et.ed) // as in Code.codeKeys, before the editor
		et.ed.HandleEvent(events.NewKey(events.KeyChord, r, 0, 0))
		for _, f := range et.ed.Deferred {
			f()
		}
		et.ed.Deferred = nil
		et.t.Logf("typed %q: line %q cursor %d  completions(seed %q): %s  signature: %q",
			r, et.line(), et.ed.CursorPos.Char, et.seed, et.labels(5), et.ls.SigStatusMsg())
	}
}

// line returns the text of the cursor line.
func (et *editorTest) line() string {
	return string(et.ed.Lines.Line(et.ed.CursorPos.Line))
}

// labels returns up to n of the current completion labels, for logging.
func (et *editorTest) labels(n int) string {
	var ls []string
	for i, m := range et.matches {
		if i == n {
			ls = append(ls, "...")
			break
		}
		ls = append(ls, m.Label)
	}
	return "[" + strings.Join(ls, " ") + "]"
}

// hasMatch returns whether the current completions include the label.
func (et *editorTest) hasMatch(label string) bool {
	return slices.ContainsFunc(et.matches, func(c complete.Completion) bool { return c.Label == label })
}

const editorTestSrc = "package main\n\nimport \"strings\"\n\nfunc main() {\n\t\n}\n"

// TestEditorCompletionTyping types a qualified identifier forward, one
// key at a time, and checks that completions are offered at each step,
// which requires the server to have the text as of each keystroke:
// the editor requests completions right away, before the debounced
// change sync.
func TestEditorCompletionTyping(t *testing.T) {
	et := newEditorTest(t, editorTestSrc, textpos.Pos{Line: 5, Char: 1})
	for _, step := range []struct{ typed, seed, want string }{
		{"strings.", "strings.", "Split"},
		{"S", "strings.S", "Split"},
		{"p", "strings.Sp", "Split"},
		{"l", "strings.Spl", "Split"},
	} {
		n := et.nmatch
		et.typeText(step.typed)
		if et.nmatch == n {
			t.Fatalf("after typing %q: completer was not called", step.typed)
		}
		if et.seed != step.seed {
			t.Errorf("after typing %q: seed = %q, want %q", step.typed, et.seed, step.seed)
		}
		if !et.hasMatch(step.want) {
			t.Errorf("after typing %q (line %q): %q not in completions %s", step.typed, et.line(), step.want, et.labels(10))
		}
	}
	// backspace re-offers completions for the shorter prefix:
	et.ed.HandleEvent(events.NewKey(events.KeyChord, 0, key.CodeBackspace, 0))
	if et.seed != "strings.Sp" || !et.hasMatch("Split") {
		t.Errorf("after backspace: seed %q, completions %s", et.seed, et.labels(10))
	}
}

// TestEditorSignatureHelp types a call and checks that the signature
// help is shown inside the parentheses, with the active parameter
// tracking the arguments (including inside string arguments, where
// gopls returns nothing), and goes away as soon as the call is closed,
// including by typing over the auto-inserted close paren, which moves
// the cursor without changing the text.
func TestEditorSignatureHelp(t *testing.T) {
	et := newEditorTest(t, editorTestSrc, textpos.Pos{Line: 5, Char: 1})
	et.typeText("strings.Replace(")
	if got := et.line(); got != "\tstrings.Replace()" {
		t.Fatalf("expected auto-inserted close paren: %q", got)
	}
	sigHas := func(after, want string) {
		t.Helper()
		if sig := et.ls.SigStatusMsg(); !strings.Contains(sig, want) {
			t.Errorf("%s: signature %q should contain %q", after, sig, want)
		}
	}
	sigHas("after open paren", "[s string]")
	et.typeText(`"a`)
	sigHas("inside string argument", "[s string]")
	et.typeText(`", `)
	sigHas("after first argument", "[old string]")
	et.typeText(`"b", "c", 1`)
	sigHas("in last argument", "[n int]")
	if et.matches != nil && et.seed == "" {
		t.Errorf("completions offered with empty seed: %s", et.labels(5))
	}
	et.typeText(")") // types over the auto-inserted paren
	if got := et.line(); got != "\tstrings.Replace(\"a\", \"b\", \"c\", 1)" {
		t.Errorf("typing over the close paren: line %q", got)
	}
	if sig := et.ls.SigStatusMsg(); sig != "" {
		t.Errorf("after closing the call: signature still shown: %q", sig)
	}

	et.ed.SetCursorShow(textpos.Pos{Line: 5, Char: et.ed.Lines.LineLen(5)})
	et.ed.HandleEvent(events.NewKey(events.KeyChord, 0, key.CodeReturnEnter, 0))
	et.typeText("strings.ToLower(")
	sigHas("in second call", "[s string]")
	et.typeText(")") // types over right after it is inserted too
	if got := et.line(); got != "\tstrings.ToLower()" {
		t.Errorf("typing over the close paren: line %q", got)
	}
	if sig := et.ls.SigStatusMsg(); sig != "" {
		t.Errorf("after typing over the close paren: signature still shown: %q", sig)
	}
}

// TestEditorLookup checks the Ctrl+. lookup: completions for the text
// up to the cursor, headed by go to definition, when there is no error,
// and quick fixes when there is an error with fixes.
func TestEditorLookup(t *testing.T) {
	src := "package main\n\nimport \"strings\"\n\nfunc main() {\n\t_ = strings.Split(\"a\", \",\")\n}\n"
	et := newEditorTest(t, src, textpos.Pos{Line: 5, Char: 16}) // strings.Spl|it
	n := et.nmatch
	if et.ls.Lookup(et.ed) {
		t.Fatal("quick fixes shown with no error")
	}
	if et.nmatch == n {
		t.Fatal("completer was not called")
	}
	t.Logf("completions(seed %q): %s", et.seed, et.labels(6))
	if et.seed != "strings.Spl" || !et.hasMatch("Split") || !et.hasMatch("SplitN") {
		t.Errorf("completions for strings.Spl: seed %q, %s", et.seed, et.labels(10))
	}
	if len(et.matches) == 0 || et.matches[0].Text != gotoDefCompletion {
		t.Fatalf("first completion is not go to definition: %s", et.labels(3))
	}
	gd := et.matches[0]
	if !strings.Contains(gd.Label, "strings.go") || !strings.Contains(gd.Desc, "Split") {
		t.Errorf("go to definition item: label %q, desc %q", gd.Label, gd.Desc)
	}
	if et.ls.gotoLoc == nil || filepath.Base(et.ls.gotoLoc.Filename) != "strings.go" {
		t.Errorf("go to definition location: %v", et.ls.gotoLoc)
	}
	if len(et.matches) < 2 || et.matches[1].Text != docsCompletion || !strings.HasPrefix(et.matches[1].Label, "Docs: func strings.Split") {
		t.Errorf("second completion is not docs: %s", et.labels(3))
	}
	if !strings.Contains(et.ls.docsMD, "```go") {
		t.Errorf("docs markdown: %q", et.ls.docsMD)
	}

	// typing after an explicit request goes back to plain completions:
	et.ed.SetCursorShow(textpos.Pos{Line: 5, Char: 18})
	et.typeText("x")
	if len(et.matches) > 0 && et.matches[0].Text == gotoDefCompletion {
		t.Errorf("go to definition offered while typing")
	}

	// an error with a quick fix: undefined fmt, fixed by adding the import
	et.ed.SetCursorShow(textpos.Pos{Line: 5, Char: et.ed.Lines.LineLen(5)})
	et.ed.HandleEvent(events.NewKey(events.KeyChord, 0, key.CodeReturnEnter, 0))
	et.typeText("fmt.Println()")
	et.ls.Change(et.ed.Lines)
	pos := textpos.Pos{Line: 6, Char: 2}
	waitCode(t, 30*time.Second, "undefined fmt diagnostic", func() bool {
		return len(et.ls.diagsAt(et.ed.Lines, pos)) > 0
	})
	et.ed.SetCursorShow(pos)
	if !et.ls.Lookup(et.ed) {
		t.Errorf("quick fixes not shown for error: %v", et.ls.diagsAt(et.ed.Lines, pos))
	}
}

// TestEditorExpandSelection expands the selection from a word in a call
// argument outward, and shrinks it back.
func TestEditorExpandSelection(t *testing.T) {
	src := "package main\n\nimport \"strings\"\n\nfunc main() {\n\t_ = strings.Index(\"abc\", \"b\")\n}\n"
	et := newEditorTest(t, src, textpos.Pos{Line: 5, Char: 8}) // in "strings"
	selText := func() string {
		r := et.ed.SelectRegion
		return string(et.ed.Lines.Region(r.Start, r.End).ToBytes())
	}
	var sels []string
	for et.ls.ExpandSelection(et.ed) {
		sels = append(sels, selText())
	}
	t.Logf("expansions: %q", sels)
	if len(sels) < 4 || sels[0] != "strings" || !strings.Contains(sels[len(sels)-1], "func main()") {
		t.Fatalf("expansions: %q", sels)
	}
	if !slices.Contains(sels, `strings.Index("abc", "b")`) {
		t.Errorf("call expression not among expansions: %q", sels)
	}
	for i := len(sels) - 2; i >= 0; i-- {
		et.ls.ShrinkSelection(et.ed)
		if got := selText(); got != sels[i] {
			t.Errorf("shrink %d: %q, want %q", i, got, sels[i])
		}
	}
	et.ls.ShrinkSelection(et.ed)
	if et.ed.HasSelection() {
		t.Errorf("selection after shrinking all the way: %q", selText())
	}
}

// TestEditorOccurrences checks the highlighting of the occurrences of
// the symbol at the cursor, which does not replace other highlights.
func TestEditorOccurrences(t *testing.T) {
	src := "package main\n\nfunc main() {\n\tx := 1\n\tx++\n\t_ = x\n}\n"
	et := newEditorTest(t, src, textpos.Pos{Line: 3, Char: 1}) // on x
	ed := et.ed
	regs := et.ls.occurrences(ed, ed.Lines, ed.CursorPos)
	if len(regs) != 3 {
		t.Fatalf("occurrences of x: %v", regs)
	}
	et.ls.setOccurrences(ed, regs)
	assert.Equal(t, regs, ed.Highlights)
	// moving to a symbol with one occurrence clears them:
	et.ls.setOccurrences(ed, et.ls.occurrences(ed, ed.Lines, textpos.Pos{Line: 2, Char: 6}))
	assert.Empty(t, ed.Highlights)
	// other highlights (e.g., from Find) are not replaced:
	find := textpos.Region{Start: textpos.Pos{Line: 2, Char: 0}, End: textpos.Pos{Line: 2, Char: 4}}
	ed.HighlightRegion(find)
	et.ls.setOccurrences(ed, regs)
	assert.Equal(t, []textpos.Region{find}, ed.Highlights)
	// until they are reset:
	ed.HighlightsReset()
	et.ls.setOccurrences(ed, regs)
	assert.Equal(t, regs, ed.Highlights)
}
