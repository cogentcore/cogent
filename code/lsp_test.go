// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/textpos"
	"cogentcore.org/core/text/token"

	"cogentcore.org/cogent/code/lang"
)

// these tests exercise the diagnostic tagging paths without a shown
// scene, which also pins the requirement that these paths never call
// AsyncLock, and never wrap self-locking lines.Lines methods in
// ln.Lock() (the Lines lock is not reentrant; recursion wedges the
// whole editor -- see also code/lsp/backend_test.go).

const tstSrcCode = "package a\n\nvar x = undefined\n\nfunc F() {\n}\n"

func codeTestLines(t *testing.T) *lines.Lines {
	t.Helper()
	fpath := filepath.Join(t.TempDir(), "a.go")
	if err := os.WriteFile(fpath, []byte(tstSrcCode), 0644); err != nil {
		t.Fatal(err)
	}
	ln := lines.NewLines()
	ln.SetFilename(fpath)
	ln.SetText([]byte(tstSrcCode))
	return ln
}

// errTagCount returns the number of token.Error diagnostic tags on a line.
func errTagCount(ln *lines.Lines, line int) int {
	n := 0
	for _, t := range ln.AdjustedTags(line) {
		if t.Token.Token == token.Error {
			n++
		}
	}
	return n
}

func guardCode(t *testing.T, d time.Duration, desc string, fn func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn()
	}()
	select {
	case <-done:
	case <-time.After(d):
		t.Fatalf("possible deadlock: %s", desc)
	}
}

func TestApplyClearDiagsTags(t *testing.T) {
	cv := NewCode()
	ls := NewLangService(cv)
	defer ls.Stop()
	ln := codeTestLines(t)

	diags := []lang.Diagnostic{
		{Region: textpos.NewRegion(2, 8, 2, 17), Msg: "undefined: undefined", Severity: lang.DiagErr},
		{Region: textpos.NewRegion(4, 5, 4, 6), Msg: "meh", Severity: lang.DiagWarn},
		{Region: textpos.NewRegion(0, 0, 0, 3), Msg: "fyi", Severity: lang.DiagInfo}, // not tagged
	}

	guardCode(t, 10*time.Second, "applyDiags on plain buffer", func() {
		ls.applyDiags(ln, diags)
		if got := errTagCount(ln, 2); got != 1 {
			t.Errorf("line 2: %d error tags, want 1", got)
		}
		if got := errTagCount(ln, 4); got != 1 {
			t.Errorf("line 4: %d error tags, want 1", got)
		}
		if got := errTagCount(ln, 0); got != 0 {
			t.Errorf("info diagnostic was tagged: %d", got)
		}

		// re-apply with fewer problems: old tags must be replaced, not accumulate:
		ls.applyDiags(ln, diags[:1])
		if got := errTagCount(ln, 4); got != 0 {
			t.Errorf("stale tag on line 4 after re-apply: %d, want 0", got)
		}
		if got := errTagCount(ln, 2); got != 1 {
			t.Errorf("line 2 after re-apply: %d error tags, want 1", got)
		}

		// empty list clears everything:
		ls.applyDiags(ln, nil)
		if got := errTagCount(ln, 2); got != 0 {
			t.Errorf("tags not cleared with empty diags: %d", got)
		}
	})

	// clearDiags cleans up tracking state without error:
	guardCode(t, 10*time.Second, "applyDiags + clearDiags", func() {
		ls.applyDiags(ln, diags)
		ls.clearDiags(ln)
		if got := errTagCount(ln, 2); got != 0 {
			t.Errorf("clearDiags left %d tags", got)
		}
	})
}

// TestDiagsTaggingConcurrentMarkup reproduces the goroutine interleave
// from the live editor: diagnostic tagging (self-locking Lines calls)
// contending with core's async markup and text edits.
func TestDiagsTaggingConcurrentMarkup(t *testing.T) {
	cv := NewCode()
	ls := NewLangService(cv)
	defer ls.Stop()
	ln := codeTestLines(t)

	diags := []lang.Diagnostic{
		{Region: textpos.NewRegion(2, 8, 2, 17), Msg: "err", Severity: lang.DiagErr},
	}

	stop := make(chan struct{})
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				ls.applyDiags(ln, diags)
				ls.applyDiags(ln, nil)
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				ln.ReMarkup()
				time.Sleep(time.Millisecond)
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				ln.InsertText(textpos.Pos{Line: 0, Char: 0}, []rune("//t\n"))
				ln.DeleteText(textpos.Pos{Line: 0, Char: 0}, textpos.Pos{Line: 0, Char: 4})
			}
		}
	}()

	time.Sleep(300 * time.Millisecond)
	close(stop)
	guardCode(t, 30*time.Second, "diags tagging vs markup goroutines", wg.Wait)
}

func TestInputUnknownLangNoop(t *testing.T) {
	cv := NewCode()
	ls := NewLangService(cv)
	defer ls.Stop()

	fpath := filepath.Join(t.TempDir(), "data.unknownext")
	if err := os.WriteFile(fpath, []byte("stuff\n"), 0644); err != nil {
		t.Fatal(err)
	}
	ln := lines.NewLines()
	ln.SetFilename(fpath)
	ln.SetText([]byte("stuff\n"))

	ls.Input(ln) // unsupported language: no debounce timer, no backend call
	ls.mu.Lock()
	nt := len(ls.debounce)
	ls.mu.Unlock()
	if nt != 0 {
		t.Errorf("debounce timer created for unsupported language: %d", nt)
	}
	if ls.LSPActive(ln) {
		t.Error("LSPActive true for unsupported language")
	}
	ls.Closed(ln) // must be safe with nothing tracked
}
