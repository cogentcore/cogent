// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lsp

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"

	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/parse/lexer"
	"cogentcore.org/core/text/textpos"
	"cogentcore.org/core/text/token"
	"github.com/stretchr/testify/assert"

	"cogentcore.org/cogent/code/lang"
)

// all exported methods on [lines.Lines] lock the buffer internally, and
// that lock is NOT reentrant: calling any of them while holding ln.Lock()
// (or from a helper that does) self-deadlocks and wedges the entire
// editor, since the LSP read loop and markup goroutines then pile up
// behind the abandoned lock. These tests guard against reintroducing
// Lock-wrappers around Lines calls in this package.

// src with unicode variety: CJK (BMP) and an emoji (astral, surrogate pair)
// to exercise the UTF-16 <-> rune conversions.
const tstSrc = "package demo\n\nfunc \u5909() {\n\ts := \"sushi \U0001F363 here\" // comment\n\tfmt.Pr\n}\n"

func testLines(t *testing.T) (*lines.Lines, string) {
	t.Helper()
	fpath := filepath.Join(t.TempDir(), "demo.go")
	if err := os.WriteFile(fpath, []byte(tstSrc), 0644); err != nil {
		t.Fatal(err)
	}
	ln := lines.NewLines()
	ln.SetFilename(fpath)
	ln.SetText([]byte(tstSrc))
	if ln.Filename() != fpath {
		t.Fatalf("Lines filename %q != %q", ln.Filename(), fpath)
	}
	return ln, fpath
}

// guard runs fn in a goroutine and fails the test if it does not complete
// in time: the signature of a deadlock (recursive Lines lock).
func guard(t *testing.T, d time.Duration, desc string, fn func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn()
	}()
	select {
	case <-done:
	case <-time.After(d):
		t.Fatalf("possible deadlock: %s -- exported lines.Lines methods lock internally and the lock is not reentrant; do not wrap them in ln.Lock()", desc)
	}
}

// waitUntil polls cond until true or timeout, for conditions that depend
// on asynchronous goroutines (non-blocking conditions only).
func waitUntil(t *testing.T, d time.Duration, desc string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for: %s", desc)
}

func TestLinesHelpersNoDeadlock(t *testing.T) {
	ln, fpath := testLines(t)

	guard(t, 5*time.Second, "conversion helpers on Lines", func() {
		assert.Equal(t, tstSrc, currentText(ln))

		// plain ASCII positions round-trip unchanged:
		p := toLSPPos(ln, textpos.Pos{Line: 0, Char: 7})
		assert.Equal(t, Position{Line: 0, Character: 7}, p)

		// astral (surrogate-pair) char: the UTF-16 offset of the emoji
		// itself equals its rune offset, since nothing astral precedes
		// it; characters after it shift by +1.
		emoji := "\U0001F363"
		ln3 := lineText(ln, 3)
		bi := strings.Index(ln3, emoji)
		if bi < 0 {
			t.Fatalf("emoji not found in line: %q", ln3)
		}
		idx := utf8.RuneCountInString(ln3[:bi])
		pEmoji := toLSPPos(ln, textpos.Pos{Line: 3, Char: idx})
		assert.Equal(t, idx, pEmoji.Character)
		pAfter := toLSPPos(ln, textpos.Pos{Line: 3, Char: idx + 1})
		assert.Equal(t, idx+2, pAfter.Character)

		lnNum, ch := posFromLSP(ln, Position{Line: 3, Character: idx})
		assert.Equal(t, 3, lnNum)
		assert.Equal(t, idx, ch)
		lnNum, ch = posFromLSP(ln, Position{Line: 3, Character: idx + 2})
		assert.Equal(t, 3, lnNum)
		assert.Equal(t, idx+1, ch)

		// clamping for out-of-range positions:
		lnNum, ch = posFromLSP(ln, Position{Line: 999, Character: 999})
		assert.Equal(t, ln.NumLines()-1, lnNum)
		assert.Equal(t, utf8.RuneCountInString(lineText(ln, lnNum)), ch)

		// invalid line for toLSPPos does not panic:
		p = toLSPPos(ln, textpos.Pos{Line: 999, Char: 5})
		assert.Equal(t, 999, p.Line)

		assert.Equal(t, ln3, lineText(ln, 3))
		assert.Equal(t, "", lineText(ln, 999))

		reg := regionFromLSP(ln, Range{
			Start: Position{Line: 3, Character: 2},
			End:   Position{Line: 3, Character: idx + 2},
		})
		assert.Equal(t, textpos.Pos{Line: 3, Char: 2}, reg.Start)
		assert.Equal(t, textpos.Pos{Line: 3, Char: idx + 1}, reg.End)
	})

	// regionForFile with the file NOT tracked: falls back to reading disk:
	b := NewBackend(nil)
	defer b.Stop()
	guard(t, 5*time.Second, "regionForFile disk fallback", func() {
		reg := b.regionForFile(fpath, Range{
			Start: Position{Line: 4, Character: 1},
			End:   Position{Line: 4, Character: 7},
		})
		assert.Equal(t, textpos.Pos{Line: 4, Char: 1}, reg.Start)
	})
}

func TestHandleDiagsNoDeadlock(t *testing.T) {
	ln, fpath := testLines(t)
	b := NewBackend(nil)
	defer b.Stop()
	b.docs[normalizePath(fpath)] = &docRef{ln: ln, langID: "go", fname: fpath}

	got := make(chan lang.FileDiags, 8)
	go func() {
		for fd := range b.Diagnostics() {
			got <- fd
		}
	}()

	uri := URIFromPath(fpath)
	guard(t, 10*time.Second, "handleDiags fan-out", func() {
		for range 20 {
			b.handleDiags(uri, []Diagnostic{
				{
					Range:    Range{Start: Position{Line: 4, Character: 2}, End: Position{Line: 4, Character: 8}},
					Severity: SeverityWarning,
					Message:  "multi\nline   message",
					Source:   "x",
					Code:     "C1",
				},
				{
					Range:    Range{Start: Position{Line: 4, Character: 2}, End: Position{Line: 4, Character: 5}},
					Severity: SeverityError,
					Message:  "undefined: fmt",
				},
			})
		}
	})

	select {
	case fd := <-got:
		assert.Equal(t, fpath, fd.Filename)
		if assert.Len(t, fd.Diags, 2) {
			assert.Equal(t, "multi line message", fd.Diags[0].Msg) // collapsed to one line
			assert.Equal(t, lang.DiagWarn, fd.Diags[0].Severity)
			assert.Equal(t, "x", fd.Diags[0].Source)
			assert.Equal(t, "C1", fd.Diags[0].Code)
			assert.Equal(t, lang.DiagErr, fd.Diags[1].Severity)
			assert.Equal(t, textpos.Pos{Line: 4, Char: 2}, fd.Diags[1].Region.Start)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no diagnostics delivered on channel")
	}
}

func TestLinesConcurrencyStress(t *testing.T) {
	ln, fpath := testLines(t)
	b := NewBackend(nil)
	defer b.Stop()
	b.docs[normalizePath(fpath)] = &docRef{ln: ln, langID: "go", fname: fpath}
	uri := URIFromPath(fpath)

	stop := make(chan struct{})
	var wg sync.WaitGroup
	var diagsRcvd atomic.Int64

	// consume diagnostics so handleDiags never blocks on the channel:
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			case <-b.Diagnostics():
				diagsRcvd.Add(1)
			}
		}
	}()

	// goroutine converting positions (jsonrpc read goroutine proxy):
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				_ = currentText(ln)
				posFromLSP(ln, Position{Line: 3, Character: 7})
				toLSPPos(ln, textpos.Pos{Line: 3, Char: 7})
				lineText(ln, 1)
				regionFromLSP(ln, Range{Start: Position{Line: 1, Character: 0}, End: Position{Line: 3, Character: 4}})
			}
		}
	}()

	// goroutine applying diagnostic tags (applyDiags pattern from code pkg):
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				ln.AddTag(4, 1, 7, token.Error)
				tags := ln.AdjustedTags(4)
				out := make(lexer.Line, 0, len(tags))
				for _, tg := range tags {
					if tg.Token.Token == token.Error {
						continue
					}
					out = append(out, tg)
				}
				ln.SetTags(4, out)
				ln.MarkupLines(4, 4)
			}
		}
	}()

	// goroutine editing text (UI typing proxy):
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				ln.InsertText(textpos.Pos{Line: 1, Char: 0}, []rune("//x\n"))
				ln.DeleteText(textpos.Pos{Line: 1, Char: 0}, textpos.Pos{Line: 1, Char: 4})
			}
		}
	}()

	// goroutine triggering core's own asynchronous markup pass, which
	// holds the Lines lock internally -- this was one of the parties in
	// the original deadlock:
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

	// goroutine publishing diagnostics (server read loop proxy):
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				b.handleDiags(uri, []Diagnostic{{
					Range:    Range{Start: Position{Line: 4, Character: 1}, End: Position{Line: 4, Character: 4}},
					Severity: SeverityError,
					Message:  "stress",
				}})
				time.Sleep(time.Millisecond)
			}
		}
	}()

	time.Sleep(500 * time.Millisecond)
	close(stop)
	guard(t, 30*time.Second, "concurrency stress goroutines to finish", wg.Wait)
	assert.Positive(t, diagsRcvd.Load())
}

// TestBackendGoplsFlow runs the full Backend lifecycle against a real
// gopls server: the code path that deadlocked in the live editor, where
// serverFor -> Filename and currentText ran against the same buffer that
// the async diagnostics fan-out and markup passes were contending for.
func TestBackendGoplsFlow(t *testing.T) {
	if _, err := exec.LookPath("gopls"); err != nil {
		t.Skip("gopls not found in PATH:", err)
	}
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module demo\n\ngo 1.21\n"), 0644); err != nil {
		t.Fatal(err)
	}
	demoPath := filepath.Join(tmp, "demo.go")
	badSrc := "package demo\n\nfunc Demo() {\n\tx := undefinedThing()\n\t_ = x\n}\n"
	if err := os.WriteFile(demoPath, []byte(badSrc), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOWORK", "off")

	b := NewBackend(nil)
	b.RootFunc = func(fpath string) string { return tmp }
	defer b.Stop()

	ln := lines.NewLines()
	ln.SetFilename(demoPath)
	ln.SetText([]byte(badSrc))
	if !b.Supported(ln) {
		t.Skip("no language server for Go found at runtime")
	}

	var hasErr, cleared atomic.Bool
	done := make(chan struct{})
	go func() {
		defer close(done)
		for fd := range b.Diagnostics() {
			nerr := 0
			for _, d := range fd.Diags {
				if d.Severity == lang.DiagErr {
					nerr++
				}
			}
			if nerr > 0 {
				hasErr.Store(true)
			} else if hasErr.Load() {
				cleared.Store(true)
			}
		}
	}()

	guard(t, 60*time.Second, "Backend.Open (serverFor/currentText path)", func() {
		b.Open(ln)
	})
	waitUntil(t, 60*time.Second, "error diagnostics via Backend channel", hasErr.Load)

	// hover and completion exercise the position conversion paths:
	guard(t, 15*time.Second, "Hover/Completions conversions", func() {
		b.Hover(ln, textpos.Pos{Line: 3, Char: 7})
		b.Completions(ln, "\tx := undefined", 3, 14)
	})

	// fix the buffer and sync:
	fixed := "package demo\n\nfunc Demo() {\n}\n"
	ln.SetText([]byte(fixed))
	guard(t, 15*time.Second, "Backend.Change", func() {
		b.Change(ln)
	})
	waitUntil(t, 60*time.Second, "diagnostics cleared after fix", func() bool { return cleared.Load() })

	guard(t, 15*time.Second, "Backend.Symbols", func() {
		syms := b.Symbols(ln)
		found := false
		for _, sy := range syms {
			if sy.Name == "Demo" {
				found = true
			}
		}
		assert.True(t, found, "Demo not in document symbols")
	})

	guard(t, 15*time.Second, "Backend.Close/Stop", func() {
		b.Close(ln)
	})
	b.Stop()
	<-done
}
