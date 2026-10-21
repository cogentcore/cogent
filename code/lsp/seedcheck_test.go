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
