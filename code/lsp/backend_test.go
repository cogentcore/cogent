// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lsp

import (
	"encoding/json"
	"fmt"
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

		assert.Equal(t, textpos.Pos{Line: 3, Char: idx}, posFromLSP(ln, Position{Line: 3, Character: idx}))
		assert.Equal(t, textpos.Pos{Line: 3, Char: idx + 1}, posFromLSP(ln, Position{Line: 3, Character: idx + 2}))

		// clamping for out-of-range positions: end of buffer
		last := ln.NumLines() - 1
		end := textpos.Pos{Line: last, Char: utf8.RuneCountInString(lineText(ln, last))}
		assert.Equal(t, end, posFromLSP(ln, Position{Line: 999, Character: 999}))
		assert.Equal(t, end, posFromLSP(ln, Position{Line: last + 1, Character: 0}))

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

func TestParsePrepareRenameShapes(t *testing.T) {
	// null and empty mean not renameable:
	for _, raw := range []string{"null", ""} {
		rng, ph, ok, err := parsePrepareRename(json.RawMessage(raw))
		assert.NoError(t, err)
		assert.False(t, ok)
		assert.Nil(t, rng)
		assert.Equal(t, "", ph)
	}
	// plain Range form:
	rng, ph, ok, err := parsePrepareRename(json.RawMessage(
		`{"start":{"line":4,"character":5},"end":{"line":4,"character":10}}`))
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "", ph)
	if assert.NotNil(t, rng) {
		assert.Equal(t, 4, rng.Start.Line)
		assert.Equal(t, 5, rng.Start.Character)
		assert.Equal(t, 10, rng.End.Character)
	}
	// PrepareRenameResult form with placeholder:
	rng, ph, ok, err = parsePrepareRename(json.RawMessage(
		`{"range":{"start":{"line":1,"character":2},"end":{"line":1,"character":3}},"placeholder":"Greet"}`))
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "Greet", ph)
	if assert.NotNil(t, rng) {
		assert.Equal(t, 1, rng.Start.Line)
	}
	// MarkupRange form: range parsed, no placeholder:
	rng, ph, ok, err = parsePrepareRename(json.RawMessage(
		`{"range":{"start":{"line":7,"character":0},"end":{"line":7,"character":4}},"markup":{"kind":"markdown","value":"x"}}`))
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "", ph)
	if assert.NotNil(t, rng) {
		assert.Equal(t, 7, rng.Start.Line)
	}
	// malformed:
	_, _, _, err = parsePrepareRename(json.RawMessage(`{oops`))
	assert.Error(t, err)
}

func TestApplyWorkspaceEdit(t *testing.T) {
	dir := t.TempDir()
	fa := filepath.Join(dir, "a.go")
	fb := filepath.Join(dir, "b.go")
	// b.go has an astral char before the target words on line 2, so the
	// LSP (utf-16) positions differ from rune positions by one:
	// quote(8) emoji(9,10) quote(11) space(12) +(13) space(14) smileX(15:20)
	srcA := "package p\n\nvar smileX = 1\nvar y = smileX\n"
	srcB := "package p\n\nvar q = \"\U0001F600\" + smileX\n\nvar z = smileX\n"
	if err := os.WriteFile(fa, []byte(srcA), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fb, []byte(srcB), 0644); err != nil {
		t.Fatal(err)
	}

	b := NewBackend(nil)
	defer b.Stop()

	// a.go is "open" against a never-started server: edits must land in
	// the buffer only, not on disk:
	ln := lines.NewLines()
	ln.SetFilename(fa)
	ln.SetText([]byte(srcA))
	srv := NewServer(b.Reg, &ServerConfig{ID: "fake", Exec: "/does/not/exist"}, dir)
	uriA := URIFromPath(fa)
	uriB := URIFromPath(fb)
	b.mu.Lock()
	b.docs[normalizePath(fa)] = &docRef{ln: ln, srv: srv, langID: "go", fname: fa}
	b.mu.Unlock()

	we := &WorkspaceEdit{
		Changes: map[string][]TextEdit{
			string(uriA): {
				{Range: Range{Start: Position{Line: 2, Character: 4}, End: Position{Line: 2, Character: 10}}, NewText: "happy"},
				{Range: Range{Start: Position{Line: 3, Character: 8}, End: Position{Line: 3, Character: 14}}, NewText: "happy"},
			},
			string(uriB): {
				{Range: Range{Start: Position{Line: 2, Character: 15}, End: Position{Line: 2, Character: 21}}, NewText: "happy"},
			},
		},
		DocumentChanges: []DocumentEdit{
			{
				TextDocument: VersionedTextDocumentIdentifier{TextDocumentIdentifier: TextDocumentIdentifier{URI: uriB}, Version: 3},
				Edits:        []TextEdit{{Range: Range{Start: Position{Line: 4, Character: 8}, End: Position{Line: 4, Character: 14}}, NewText: "happy"}},
			},
		},
	}
	out := b.ApplyWorkspaceEdit(we)
	if assert.Len(t, out, 2) {
		assert.Equal(t, normalizePath(fa), out[0].Filename)
		assert.Equal(t, normalizePath(fb), out[1].Filename)
		assert.Len(t, out[0].Edits, 2)
		// edits for one file merge across Changes and DocumentChanges:
		assert.Len(t, out[1].Edits, 2)
	}

	// open buffer updated in memory, left modified for the user to save:
	assert.Equal(t, "package p\n\nvar happy = 1\nvar y = happy\n", ln.String())
	gotA, err := os.ReadFile(fa)
	if assert.NoError(t, err) {
		assert.Equal(t, srcA, string(gotA), "open buffer must not be written to disk")
	}
	// closed file rewritten on disk, with the astral utf-16 offset handled:
	gotB, err := os.ReadFile(fb)
	if assert.NoError(t, err) {
		assert.Equal(t, "package p\n\nvar q = \"\U0001F600\" + happy\n\nvar z = happy\n", string(gotB))
	}
	// the server is notified of the new buffer text:
	srv.mu.Lock()
	ds := srv.docs[uriA]
	srv.mu.Unlock()
	if assert.NotNil(t, ds) {
		assert.Equal(t, "package p\n\nvar happy = 1\nvar y = happy\n", ds.text)
	}
}

func TestApplyEditNotification(t *testing.T) {
	b := NewBackend(nil)
	defer b.Stop()
	dir := t.TempDir()
	f := filepath.Join(dir, "c.go")
	src := "package p\n\nvar old = 1\n"
	if err := os.WriteFile(f, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	we := WorkspaceEdit{Changes: map[string][]TextEdit{
		string(URIFromPath(f)): {{Range: Range{Start: Position{Line: 2, Character: 4}, End: Position{Line: 2, Character: 7}}, NewText: "new"}},
	}}
	applied, reason := b.handleApplyEdit(we)
	assert.True(t, applied, reason)
	select {
	case fe := <-b.applies:
		assert.Len(t, fe, 1)
		assert.Equal(t, normalizePath(f), fe[0].Filename)
	default:
		t.Fatal("no AppliedEdits notification delivered")
	}
	// nothing to apply: rejected:
	applied, _ = b.handleApplyEdit(WorkspaceEdit{})
	assert.False(t, applied)
}

func TestBackendRenameGopls(t *testing.T) {
	if _, err := exec.LookPath("gopls"); err != nil {
		t.Skip("gopls not found in PATH:", err)
	}
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module demo\n\ngo 1.21\n"), 0644); err != nil {
		t.Fatal(err)
	}
	greetPath := filepath.Join(tmp, "greet.go")
	usePath := filepath.Join(tmp, "use.go")
	extraPath := filepath.Join(tmp, "extra.go")
	srcGreet := "package demo\n\nfunc Greet(name string) string {\n\treturn name\n}\n"
	srcUse := "package demo\n\nvar _ = Greet\n"
	srcExtra := "package demo\n\nvar _ = Greet // never opened in the editor\n"
	for f, src := range map[string]string{greetPath: srcGreet, usePath: srcUse, extraPath: srcExtra} {
		if err := os.WriteFile(f, []byte(src), 0644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("GOWORK", "off")

	b := NewBackend(nil)
	b.RootFunc = func(fpath string) string { return tmp }
	defer b.Stop()

	lnGreet := lines.NewLines()
	lnGreet.SetFilename(greetPath)
	lnGreet.SetText([]byte(srcGreet))
	lnUse := lines.NewLines()
	lnUse.SetFilename(usePath)
	lnUse.SetText([]byte(srcUse))
	if !b.Supported(lnGreet) {
		t.Skip("no language server for Go found at runtime")
	}
	go func() {
		for range b.Diagnostics() {
		} // drain
	}()
	b.Open(lnGreet)
	b.Open(lnUse)
	waitUntil(t, 60*time.Second, "server ready", func() bool { return b.Ready(lnGreet) })

	// gopls can answer null while it is still loading the package,
	// so retry until the rename succeeds or the deadline passes:
	var out []lang.FileEdits
	deadline := time.Now().Add(60 * time.Second)
	for {
		var err error
		guard(t, 30*time.Second, "Backend.Rename", func() {
			out, err = b.Rename(lnGreet, textpos.Pos{Line: 2, Char: 6}, "Hello")
		})
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("rename never succeeded: %v", err)
		}
		time.Sleep(500 * time.Millisecond)
	}

	got := map[string]bool{}
	for _, f := range out {
		got[f.Filename] = true
	}
	assert.True(t, got[normalizePath(greetPath)], "greet.go missing from rename edits")
	assert.True(t, got[normalizePath(usePath)], "use.go missing from rename edits")
	assert.True(t, got[normalizePath(extraPath)], "unopened extra.go missing from rename edits")

	// open buffers updated in memory:
	assert.Contains(t, lnGreet.String(), "func Hello(name string)")
	assert.NotContains(t, lnGreet.String(), "Greet")
	assert.Contains(t, lnUse.String(), "var _ = Hello")

	// open files stay unsaved on disk; the unopened file is rewritten:
	dg, err := os.ReadFile(greetPath)
	if assert.NoError(t, err) {
		assert.Equal(t, srcGreet, string(dg))
	}
	de, err := os.ReadFile(extraPath)
	if assert.NoError(t, err) {
		assert.Equal(t, "package demo\n\nvar _ = Hello // never opened in the editor\n", string(de))
	}

	// the server sees the renamed symbol in the synced buffer:
	syms := b.Symbols(lnGreet)
	found := false
	for _, sy := range syms {
		if sy.Name == "Hello" {
			found = true
		}
	}
	assert.True(t, found, "Hello not in document symbols after rename")

	// rename at a non-identifier position must fail with a reason,
	// via the advisory-prepareRename fallback to the rename request
	// (analysis is warm at this point):
	if _, err := b.RenameEdits(lnGreet, textpos.Pos{Line: 1, Char: 0}, "Nope"); err == nil {
		t.Fatal("rename at empty line unexpectedly succeeded")
	} else {
		t.Logf("bad-position rename error: %v", err)
	}
}

func TestPreviewWorkspaceEdit(t *testing.T) {
	dir := t.TempDir()
	fa := filepath.Join(dir, "a.go")
	fb := filepath.Join(dir, "b.go")
	srcA := "package p\n\nvar smileX = 1\nvar y = smileX\n"
	srcB := "package p\n\nvar q = smileX\n"
	if err := os.WriteFile(fa, []byte(srcA), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fb, []byte(srcB), 0644); err != nil {
		t.Fatal(err)
	}
	b := NewBackend(nil)
	defer b.Stop()

	ln := lines.NewLines()
	ln.SetFilename(fa)
	ln.SetText([]byte(srcA))
	srv := NewServer(b.Reg, &ServerConfig{ID: "fake", Exec: "/does/not/exist"}, dir)
	uriA := URIFromPath(fa)
	uriB := URIFromPath(fb)
	b.mu.Lock()
	b.docs[normalizePath(fa)] = &docRef{ln: ln, srv: srv, langID: "go", fname: fa}
	b.mu.Unlock()

	we := &WorkspaceEdit{
		Changes: map[string][]TextEdit{
			string(uriA): {
				{Range: Range{Start: Position{Line: 3, Character: 8}, End: Position{Line: 3, Character: 14}}, NewText: "happy"},
				{Range: Range{Start: Position{Line: 2, Character: 4}, End: Position{Line: 2, Character: 10}}, NewText: "happy"},
			},
		},
		DocumentChanges: []DocumentEdit{
			{
				TextDocument: VersionedTextDocumentIdentifier{TextDocumentIdentifier: TextDocumentIdentifier{URI: uriB}, Version: 2},
				Edits:        []TextEdit{{Range: Range{Start: Position{Line: 2, Character: 8}, End: Position{Line: 2, Character: 14}}, NewText: "happy"}},
			},
		},
	}
	prev := b.PreviewWorkspaceEdit(we)
	if assert.Len(t, prev, 3) {
		// sorted by file, then position:
		assert.Equal(t, normalizePath(fa), prev[0].Filename)
		assert.Equal(t, 2, prev[0].Region.Start.Line)
		assert.Equal(t, normalizePath(fa), prev[1].Filename)
		assert.Equal(t, 3, prev[1].Region.Start.Line)
		assert.Equal(t, normalizePath(fb), prev[2].Filename)

		for _, pv := range prev {
			assert.Equal(t, "smileX", pv.Old)
			assert.Equal(t, "happy", pv.New)
		}
		assert.True(t, prev[0].Open, "a.go is open in a tracked buffer")
		assert.False(t, prev[2].Open, "b.go is not open")
		assert.Equal(t, "var smileX = 1", prev[0].LineText)
		assert.Equal(t, "var y = smileX", prev[1].LineText)
		assert.Equal(t, "var q = smileX", prev[2].LineText)
	}
	// nothing applied:
	assert.Equal(t, srcA, ln.String())
	gotB, err := os.ReadFile(fb)
	if assert.NoError(t, err) {
		assert.Equal(t, srcB, string(gotB))
	}
}

func TestParseCodeActions(t *testing.T) {
	raw := json.RawMessage(`[
		{"title":"Organize imports","kind":"source.organizeImports","isPreferred":true,"edit":{"changes":{}}},
		{"title":"Remove unused import \"fmt\"","kind":"quickfix","diagnostics":[{"range":{"start":{"line":2,"character":7},"end":{"line":2,"character":12}},"severity":1,"message":"\"fmt\" imported and not used"}],"edit":{"changes":{}}},
		{"title":"Fix all gopls-error codes","command":"gopls.fix_all","arguments":[{"file":"file:///tmp/x.go"}]},
		{"title":"Extract function","kind":"refactor.extract.function","disabled":{"reason":"select an expression"}}
	]`)
	acts, err := parseCodeActions(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(acts) != 4 {
		t.Fatalf("got %d actions, want 4", len(acts))
	}
	assert.Equal(t, "source.organizeImports", acts[0].Kind)
	assert.True(t, acts[0].IsPreferred, "preferred not parsed")
	assert.NotNil(t, acts[0].Edit, "inline edit not parsed")
	assert.Equal(t, 1, len(acts[1].Diagnostics), "diagnostics not parsed")
	assert.Equal(t, SeverityError, acts[1].Diagnostics[0].Severity)
	// plain Command union item (LSP 3.8 legacy form):
	assert.Equal(t, "command", acts[2].Kind)
	if assert.NotNil(t, acts[2].Command) {
		assert.Equal(t, "gopls.fix_all", acts[2].Command.Command)
		assert.Equal(t, 1, len(acts[2].Command.Arguments))
	}
	// disabled action keeps its reason:
	if assert.NotNil(t, acts[3].Disabled) {
		assert.Equal(t, "select an expression", acts[3].Disabled.Reason)
	}
	// empty results:
	for _, s := range []string{"null", ""} {
		out, err := parseCodeActions(json.RawMessage(s))
		assert.NoError(t, err)
		assert.Equal(t, 0, len(out))
	}
}

func TestCodeActionsGopls(t *testing.T) {
	if _, err := exec.LookPath("gopls"); err != nil {
		t.Skip("gopls not found in PATH:", err)
	}
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module demo\n\ngo 1.21\n"), 0644); err != nil {
		t.Fatal(err)
	}
	srcPath := filepath.Join(tmp, "main.go")
	src := "package main\n\nimport \"fmt\"\n\nfunc Hello() string {\n\treturn \"hi\"\n}\n"
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOWORK", "off")

	b := NewBackend(nil)
	b.RootFunc = func(fpath string) string { return tmp }
	defer b.Stop()

	ln := lines.NewLines()
	ln.SetFilename(srcPath)
	ln.SetText([]byte(src))
	if !b.Supported(ln) {
		t.Skip("no language server for Go found at runtime")
	}
	// collect published diagnostics; quick fixes are keyed off these,
	// so the code action context needs them:
	var diags []lang.Diagnostic
	deadline := time.Now().Add(60 * time.Second)
	b.Open(ln)
	for time.Now().Before(deadline) {
		select {
		case fd, ok := <-b.Diagnostics():
			if !ok {
				t.Fatal("diagnostics channel closed")
			}
			if normalizePath(fd.Filename) != normalizePath(srcPath) {
				continue
			}
			if len(fd.Diags) > 0 {
				diags = fd.Diags
			}
		case <-time.After(2 * time.Second):
		}
		if len(diags) > 0 {
			break
		}
	}
	if len(diags) == 0 {
		t.Fatal("no diagnostics for unused import")
	}
	found := false
	for _, d := range diags {
		if strings.Contains(d.Msg, "not used") {
			found = true
		}
	}
	assert.True(t, found, "no unused-import diagnostic: %v", diags)

	// region over the import path, as a cursor on fmt would give:
	reg := textpos.Region{
		Start: textpos.Pos{Line: 2, Char: 8},
		End:   textpos.Pos{Line: 2, Char: 11},
	}
	var acts []CodeAction
	err := fmt.Errorf("never queried")
	for time.Now().Before(deadline) {
		guard(t, 30*time.Second, "Backend.CodeActions", func() {
			acts, err = b.CodeActions(ln, reg, diags)
		})
		if err == nil && len(acts) > 0 {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if err != nil || len(acts) == 0 {
		t.Fatalf("no code actions offered: %v", err)
	}
	var titles []string
	for _, ca := range acts {
		titles = append(titles, ca.Kind+": "+ca.Title)
	}
	t.Logf("actions: %v", titles)
	hasFix := false
	for _, ca := range acts {
		if strings.HasPrefix(ca.Kind, "quickfix") || strings.Contains(ca.Kind, "organizeImports") {
			hasFix = true
		}
	}
	assert.True(t, hasFix, "no quick fix / organize imports offered: %v", titles)

	// apply the first actionable fix and verify the import goes away:
	var sel *CodeAction
	for i := range acts {
		if acts[i].Disabled == nil && strings.HasPrefix(acts[i].Kind, "quickfix") {
			sel = &acts[i]
			break
		}
	}
	if sel == nil {
		for i := range acts {
			if acts[i].Disabled == nil && strings.Contains(acts[i].Kind, "organizeImports") {
				sel = &acts[i]
				break
			}
		}
	}
	if sel == nil {
		t.Fatalf("no appliable fix in %v", titles)
	}
	var out []lang.FileEdits
	guard(t, 30*time.Second, "Backend.RunCodeAction", func() {
		out, err = b.RunCodeAction(ln, sel)
	})
	if !assert.NoError(t, err, "RunCodeAction %q", sel.Title) {
		t.Fatalf("could not apply %q", sel.Title)
	}
	assert.True(t, len(out) > 0, "no edits applied")
	assert.NotContains(t, ln.String(), `"fmt"`, "unused import still present after fix %q", sel.Title)
}

func TestSignatureContextAndActiveParam(t *testing.T) {
	cases := []struct {
		text string
		pos  int
		want int
		desc string
	}{
		{"foo(a, b", 8, 1, "second arg"},
		{"foo(", 4, 0, "just after paren"},
		{"foo(a", 5, 0, "first arg"},
		{"foo(bar(a, b), c", 16, 1, "nested call outer second arg"},
		{"x := 1", 6, -1, "not in a call"},
		{"foo(a[1, 2]", 11, 0, "inside brackets"},
		{"foo(\"a,b\", c", 11, 1, "comma inside string ignored"},
		{"foo(a, b", 3, -1, "pos before paren: no call context"},
	}
	for _, tc := range cases {
		got := activeParameter(tc.text, tc.pos)
		assert.Equal(t, tc.want, got, tc.desc)
	}

	assert.True(t, IsSignatureTrigger("foo("))
	assert.True(t, IsSignatureTrigger("foo(a,"))
	assert.False(t, IsSignatureTrigger("foo(a"))
	assert.False(t, IsSignatureTrigger(""))

	c := signatureContext("foo(", false)
	assert.Equal(t, SigHelpTriggerChar, c.TriggerKind)
	assert.Equal(t, "(", c.TriggerCharacter)
	c = signatureContext("foo(a", true)
	assert.Equal(t, SigHelpContentChange, c.TriggerKind)
	c = signatureContext("foo(a", false)
	assert.Equal(t, SigHelpInvoked, c.TriggerKind)
}

func TestParameterLabelText(t *testing.T) {
	label := "Read(f *os.File, p []byte) (n int, err error)"
	pi := ParameterInformation{Label: json.RawMessage(`"f *os.File"`)}
	assert.Equal(t, "f *os.File", pi.LabelText(label))
	// offset form: offsets are UTF-16 code units into the signature label
	pi = ParameterInformation{Label: json.RawMessage(`[5,15]`)}
	assert.Equal(t, "f *os.File", pi.LabelText(label))
	pi = ParameterInformation{Label: json.RawMessage(`null`)}
	assert.Equal(t, "", pi.LabelText(label))
}

func TestSignatureText(t *testing.T) {
	sh := &SignatureHelp{
		Signatures: []SignatureInformation{{
			Label:      "Greet(name string, loud bool) string",
			Parameters: []ParameterInformation{{Label: json.RawMessage(`"name string"`)}, {Label: json.RawMessage(`"loud bool"`)}},
		}},
	}
	assert.Equal(t, "Greet([name string], loud bool) string", SignatureText(sh, 0, 0, "[", "]"))
	assert.Equal(t, "Greet(name string, [loud bool]) string", SignatureText(sh, 0, 1, "[", "]"))
	assert.Equal(t, "Greet(name string, loud bool) string", SignatureText(sh, 0, -1, "[", "]"))
	assert.Equal(t, "", SignatureText(nil, 0, 0, "[", "]"))
	assert.Equal(t, "", SignatureText(sh, 5, 0, "[", "]"))
}

func TestInlayHintLabelText(t *testing.T) {
	ih := InlayHint{Label: json.RawMessage(`"n:"`)}
	assert.Equal(t, "n:", ih.LabelText())
	ih = InlayHint{Label: json.RawMessage(`[{"value":"err "},{"value":":error"}]`)}
	assert.Equal(t, "err :error", ih.LabelText())
	ih = InlayHint{Label: json.RawMessage(`null`)}
	assert.Equal(t, "", ih.LabelText())
}

func TestDecodeSemanticTokens(t *testing.T) {
	// two lines, second token on line 1 is after a unicode char:
	src := "var x = 1\ns := \"\u5909\""
	ln := lines.NewLines()
	ln.SetText([]byte(src))
	legend := &SemanticTokensLegend{
		TokenTypes:     []string{"variable", "number", "string"},
		TokenModifiers: []string{"declaration", "deprecated"},
	}
	// tokens: x variable, 1 number, s variable, and the quoted string,
	// which in UTF-16 columns runs 5..9 (opening quote at 5, CJK char
	// adds one extra utf-16 unit, so the rune end (8) must differ from
	// the utf-16 end (9)).
	st := &SemanticTokens{Data: []int{
		0, 4, 1, 0, 1, // x: variable, declaration
		0, 4, 1, 1, 0, // 1: number (delta start from x = +4)
		1, 0, 1, 0, 1, // line 1: s: variable
		0, 5, 4, 2, 0, // the quoted string: 4 utf-16 units from col 5
	}}
	toks, err := decodeSemanticTokens(ln, legend, st)
	if err != nil {
		t.Fatal(err)
	}
	if len(toks) != 4 {
		t.Fatalf("got %d tokens, want 4", len(toks))
	}
	assert.Equal(t, "variable", toks[0].Type)
	assert.Equal(t, textpos.Pos{Line: 0, Char: 4}, toks[0].Start)
	assert.ElementsMatch(t, []string{"declaration"}, toks[0].Modifiers)
	assert.Equal(t, "number", toks[1].Type)
	assert.Equal(t, textpos.Pos{Line: 0, Char: 8}, toks[1].Start)
	assert.Equal(t, "variable", toks[2].Type)
	assert.Equal(t, 1, toks[2].Start.Line)
	assert.Equal(t, "string", toks[3].Type)
	assert.Equal(t, textpos.Pos{Line: 1, Char: 5}, toks[3].Start)
	assert.Equal(t, textpos.Pos{Line: 1, Char: 8}, toks[3].End)

	// deprecated bit:
	st = &SemanticTokens{Data: []int{0, 4, 1, 0, 2}}
	toks, err = decodeSemanticTokens(ln, legend, st)
	assert.NoError(t, err)
	if assert.Equal(t, 1, len(toks)) {
		assert.True(t, toks[0].Deprecated)
	}

	// bad data length:
	_, err = decodeSemanticTokens(ln, legend, &SemanticTokens{Data: []int{0, 1, 2}})
	assert.Error(t, err)
}

func TestSemanticProviderLegend(t *testing.T) {
	leg := semanticTokensProviderLegend(json.RawMessage(`{"legend":{"tokenTypes":["variable"],"tokenModifiers":[]},"full":true}`))
	if assert.NotNil(t, leg) {
		assert.Equal(t, []string{"variable"}, leg.TokenTypes)
	}
	assert.Nil(t, semanticTokensProviderLegend(json.RawMessage(`true`)))
	assert.Nil(t, semanticTokensProviderLegend(json.RawMessage(`{"legend":{"tokenTypes":[]}}`)))
}

// goplsSetup sets up a temp module with the given source and a backend
// with the file open and the server ready.
func goplsSetup(t *testing.T, src string) (*Backend, *lines.Lines, string) {
	t.Helper()
	if _, err := exec.LookPath("gopls"); err != nil {
		t.Skip("gopls not found in PATH:", err)
	}
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module demo\n\ngo 1.21\n"), 0644); err != nil {
		t.Fatal(err)
	}
	srcPath := filepath.Join(tmp, "main.go")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOWORK", "off")
	b := NewBackend(nil)
	b.RootFunc = func(fpath string) string { return tmp }
	t.Cleanup(b.Stop)
	ln := lines.NewLines()
	ln.SetFilename(srcPath)
	ln.SetText([]byte(src))
	if !b.Supported(ln) {
		t.Skip("no language server for Go found at runtime")
	}
	go func() {
		for range b.Diagnostics() {
		} // drain
	}()
	b.Open(ln)
	waitUntil(t, 60*time.Second, "server ready", func() bool { return b.Ready(ln) })
	return b, ln, srcPath
}

func TestSignatureHelpGopls(t *testing.T) {
	src := "package main\n\nimport \"strings\"\n\nfunc main() {\n\tstrings.Split(\"a,b\", \",\")\n}\n"
	b, ln, _ := goplsSetup(t, src)

	// cursor inside the call, on the second argument:
	// line 5: "\tstrings.Split("a,b", ",")" -- pos after the second `", `
	pos := textpos.Pos{Line: 5, Char: 21}
	var sh *SignatureHelp
	act := -1
	var err error
	deadline := time.Now().Add(60 * time.Second)
	for {
		guard(t, 20*time.Second, "Backend.SignatureHelp", func() {
			sh, act, err = b.SignatureHelp(ln, pos, false)
		})
		if err == nil && sh != nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("signature help never arrived: %v", err)
		}
		time.Sleep(500 * time.Millisecond)
	}
	if act != 0 || len(sh.Signatures) == 0 {
		t.Fatalf("bad signature help: act=%d sigs=%d", act, len(sh.Signatures))
	}
	sig := sh.Signatures[act]
	// gopls gives the bare function label for cross-package calls:
	if !strings.Contains(sig.Label, "Split(") {
		t.Errorf("signature label: %q", sig.Label)
	}
	if len(sig.Parameters) != 2 {
		t.Fatalf("got %d params, want 2: %q", len(sig.Parameters), sig.Label)
	}
	if sh.ActiveParameter == nil || *sh.ActiveParameter != 1 {
		t.Errorf("active parameter: %v, want 1", sh.ActiveParameter)
	}
	txt := SignatureText(sh, act, *sh.ActiveParameter, "[", "]")
	t.Logf("marked signature: %s", txt)
	if !strings.Contains(txt, "[") {
		t.Errorf("active param not marked: %q", txt)
	}
}

func TestSemanticTokensGopls(t *testing.T) {
	src := "package main\n\nimport \"fmt\"\n\nfunc Hello(name string) {\n\tfmt.Println(\"hello\", name)\n}\n"
	b, ln, _ := goplsSetup(t, src)

	var toks []SemanticToken
	var err error
	deadline := time.Now().Add(60 * time.Second)
	for {
		guard(t, 30*time.Second, "Backend.SemanticTokensFull", func() {
			toks, err = b.SemanticTokensFull(ln)
		})
		if err == nil && len(toks) > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("semantic tokens never arrived: %v", err)
		}
		time.Sleep(500 * time.Millisecond)
	}
	byType := map[string]int{}
	var helloTok, nameTok, fmtTok, strTok *SemanticToken
	for i := range toks {
		tk := &toks[i]
		byType[tk.Type]++
		line := string(ln.Line(tk.Start.Line))
		word := ""
		if tk.End.Char <= len([]rune(line)) {
			word = string([]rune(line)[tk.Start.Char:tk.End.Char])
		}
		switch word {
		case "Hello":
			helloTok = tk
		case "name":
			if nameTok == nil {
				nameTok = tk
			}
		case "fmt":
			fmtTok = tk
		}
		if tk.Type == "string" {
			strTok = tk
		}
	}
	t.Logf("token types: %v", byType)
	if helloTok == nil || helloTok.Type != "function" {
		t.Errorf("Hello token: %+v", helloTok)
	}
	if nameTok == nil || nameTok.Type != "parameter" {
		t.Errorf("name token: %+v", nameTok)
	}
	if fmtTok == nil {
		t.Errorf("no fmt token")
	}
	if strTok == nil {
		t.Errorf("no string literal token")
	}
	// all decoded regions must be valid in the buffer:
	nln := ln.NumLines()
	for _, tk := range toks {
		if tk.Start.Line < 0 || tk.Start.Line >= nln {
			t.Fatalf("token line out of range: %+v", tk)
		}
		lc := len([]rune(string(ln.Line(tk.Start.Line))))
		if tk.Start.Char < 0 || tk.End.Char > lc || tk.End.Char < tk.Start.Char {
			t.Fatalf("token chars out of range: %+v line len %d", tk, lc)
		}
	}
}

func TestInlayHintsGopls(t *testing.T) {
	src := "package main\n\nfunc add(a, b int) int { return a + b }\n\nvar _ = add(1, 2)\n"
	b, ln, _ := goplsSetup(t, src)

	var hints []InlayHint
	var err error
	deadline := time.Now().Add(60 * time.Second)
	for {
		guard(t, 30*time.Second, "Backend.InlayHints", func() {
			hints, err = b.InlayHints(ln, textpos.Region{
				Start: textpos.Pos{Line: 0, Char: 0},
				End:   textpos.Pos{Line: ln.NumLines(), Char: 0},
			})
		})
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("inlay hints request never succeeded: %v", err)
		}
		time.Sleep(500 * time.Millisecond)
	}
	// gopls only provides hints when enabled via initialization options,
	// so an empty result is valid; verify decoding of whatever comes back.
	for _, h := range hints {
		txt := h.LabelText()
		if txt == "" {
			t.Errorf("hint with empty label at %+v", h.Position)
		}
		if h.Kind != InlayHintType && h.Kind != InlayHintParameter && h.Kind != 0 {
			t.Errorf("unexpected hint kind: %v", h.Kind)
		}
	}
	t.Logf("%d inlay hints", len(hints))
}

// TestInlayHintsConfig verifies the settings path that turns gopls's
// inlay hints on (they are gated behind settings, default off): options
// set before the server starts arrive through initializationOptions, and
// options set on a running server arrive through the dynamic
// didChangeConfiguration / workspace/configuration re-fetch.
func TestInlayHintsConfig(t *testing.T) {
	src := "package main\n\nimport \"strings\"\n\nfunc main() {\n\tx := 1\n\t_, _ = x, strings.Split(\"a,b\", \",\")\n}\n"
	hintOpts := map[string]map[string]any{
		"gopls": {"hints": map[string]any{
			"parameterNames":      true,
			"assignVariableTypes": true,
		}},
	}

	// startup path: options must be on the registry before the server
	// starts -- the editor applies the setting in NewLangService, before
	// any file opens -- arriving through initializationOptions.
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module demo\n\ngo 1.21\n"), 0644); err != nil {
		t.Fatal(err)
	}
	srcPath := filepath.Join(tmp, "main.go")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	b := NewBackend(nil)
	b.RootFunc = func(string) string { return tmp }
	t.Cleanup(b.Stop)
	b.SetOptions(hintOpts)
	ln := lines.NewLines()
	ln.SetFilename(srcPath)
	ln.SetText([]byte(src))
	b.Open(ln)
	waitUntil(t, 60*time.Second, "server ready (hint options)", func() bool { return b.Ready(ln) })
	waitUntil(t, 60*time.Second, "hints via init options", func() bool {
		hs, err := b.InlayHints(ln, textpos.Region{End: textpos.Pos{Line: ln.NumLines()}})
		return err == nil && len(hs) > 0
	})

	// dynamic path: enable on a running server that started without any
	// hint options, through Reconfigure (didChangeConfiguration +
	// workspace/configuration re-fetch + didChange snapshot refresh).
	b2, ln2, _ := goplsSetup(t, src)
	hs0, err := b2.InlayHints(ln2, textpos.Region{End: textpos.Pos{Line: ln2.NumLines()}})
	if err != nil {
		t.Fatalf("baseline InlayHints: %v", err)
	}
	if len(hs0) > 0 {
		t.Skipf("server returned %d hints without options; cannot test the off state", len(hs0))
	}
	b2.SetOptions(hintOpts)
	b2.Reconfigure()
	waitUntil(t, 60*time.Second, "hints via Reconfigure", func() bool {
		hs, err := b2.InlayHints(ln2, textpos.Region{End: textpos.Pos{Line: ln2.NumLines()}})
		return err == nil && len(hs) > 0
	})

	// the shared defaults must never be mutated by option merging
	for _, sc := range DefaultServers {
		if sc.ID == "gopls" {
			if _, has := sc.InitOptions["hints"]; has {
				t.Error("DefaultServers gopls InitOptions were mutated by SetOptions")
			}
		}
	}
}
