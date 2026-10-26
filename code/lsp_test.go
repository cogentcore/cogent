// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/textpos"
	"cogentcore.org/core/text/token"

	"cogentcore.org/cogent/code/lang"
	"cogentcore.org/cogent/code/lsp"
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
		ls.clearDiags(ln, ln.Filename())
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

	wg.Go(func() {
		for {
			select {
			case <-stop:
				return
			default:
				ls.applyDiags(ln, diags)
				ls.applyDiags(ln, nil)
			}
		}
	})

	wg.Go(func() {
		for {
			select {
			case <-stop:
				return
			default:
				ln.ReMarkup()
				time.Sleep(time.Millisecond)
			}
		}
	})

	wg.Go(func() {
		for {
			select {
			case <-stop:
				return
			default:
				ln.InsertText(textpos.Pos{Line: 0, Char: 0}, []rune("//t\n"))
				ln.DeleteText(textpos.Pos{Line: 0, Char: 0}, textpos.Pos{Line: 0, Char: 4})
			}
		}
	})

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

// waitCode polls cond until true or fails after the timeout, for
// conditions satisfied by asynchronous goroutines (non-blocking conds).
func waitCode(t *testing.T, d time.Duration, desc string, cond func() bool) {
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

// TestDiagsFlowWithGopls runs the full editor diagnostic pipeline
// against a real gopls server: Open -> publishDiagnostics -> consumeDiags
// -> applyDiags tags the buffer; then fix the code -> Change -> tags
// cleared. This is the end-to-end regression test for "typing garbage
// flags nothing" at the service layer (the pixel-level visibility of
// token.Error is checked separately by TestErrorTagRenderVisible).
func TestDiagsFlowWithGopls(t *testing.T) {
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
	// a file that is never opened, whose type errors gopls also reports:
	otherPath := filepath.Join(tmp, "other.go")
	if err := os.WriteFile(otherPath, []byte("package demo\n\nvar s = \"🙂\"; var y = undefinedOther\n"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOWORK", "off")

	cv := NewCode()
	ls := NewLangService(cv)
	defer ls.Stop()

	ln := lines.NewLines()
	ln.SetFilename(demoPath)
	ln.SetText([]byte(badSrc))
	cv.OpenFiles.Add(ln) // so consumeDiags can resolve the file

	ls.lspB.Open(ln) // what Activated does for the active editor

	waitCode(t, 60*time.Second, "error diagnostic tagged on line 3", func() bool {
		return errTagCount(ln, 3) > 0
	})
	waitCode(t, 10*time.Second, "status-bar summary stored", func() bool {
		return strings.Contains(ls.DiagStatusMsg(ln), "1 error")
	})
	ls.mu.Lock()
	var msg string
	for _, p := range ls.prevDiags[demoPath] {
		msg = p.sev + ": " + p.msg
	}
	ls.mu.Unlock()
	if !strings.Contains(msg, "undefinedThing") {
		t.Errorf("recorded diagnostic message %q does not mention undefinedThing", msg)
	}
	var other []lang.Diagnostic
	waitCode(t, 60*time.Second, "problems of the unopened file cached", func() bool {
		ls.mu.Lock()
		defer ls.mu.Unlock()
		other = ls.diagCache[otherPath]
		return len(other) > 0
	})
	found := false
	for _, d := range other {
		if strings.Contains(d.Msg, "undefinedOther") {
			found = true
			// `var s = "🙂"; var y = ` is 22 UTF-16 units but 21 runes:
			if want := (textpos.Pos{Line: 2, Char: 21}); d.Region.Start != want {
				t.Errorf("unopened file diagnostic at %v, want %v", d.Region.Start, want)
			}
		}
	}
	if !found {
		t.Errorf("no undefinedOther diagnostic for the unopened file: %v", other)
	}

	fixed := "package demo\n\nfunc Demo() {\n}\n"
	ln.SetText([]byte(fixed))
	ls.Change(ln)
	waitCode(t, 60*time.Second, "diagnostic tags cleared after fix", func() bool {
		return errTagCount(ln, 3) == 0
	})
	waitCode(t, 10*time.Second, "status-bar summary cleared after fix", func() bool {
		return ls.DiagStatusMsg(ln) == ""
	})

	// break it again, then close the buffer as Code does: the buffer is
	// closed first, clearing its filename, and then the service is told:
	ln.SetText([]byte(badSrc))
	ls.Change(ln)
	waitCode(t, 60*time.Second, "error diagnostics again", func() bool {
		return strings.Contains(ls.DiagStatusMsg(ln), "1 error")
	})
	ln.Close()
	ls.Closed(ln)
	ls.mu.Lock()
	_, hasStatus := ls.diagStatus[demoPath]
	ls.mu.Unlock()
	if hasStatus {
		t.Error("diagnostics status remains after close")
	}
	// the cached problems are kept for the Problems tab
	if ls.lspB.Filename(ln) != "" {
		t.Error("buffer still tracked by the backend after close")
	}
}

func TestDiagAt(t *testing.T) {
	cv := NewCode()
	ls := NewLangService(cv)
	defer ls.Stop()
	ln := codeTestLines(t)

	ls.mu.Lock()
	ls.prevDiags[ln.Filename()] = []tagPos{
		{line: 2, st: 8, ed: 17, msg: "undefined: undefined", sev: "error"},
	}
	ls.mu.Unlock()

	if got := ls.diagAt(ln, textpos.Pos{Line: 2, Char: 10}); got != "error: undefined: undefined" {
		t.Errorf("diagAt in region: %q", got)
	}
	// hover positions are often the word-end position, which equals the
	// exclusive end of the diagnostic region: must still match:
	if got := ls.diagAt(ln, textpos.Pos{Line: 2, Char: 17}); got != "error: undefined: undefined" {
		t.Errorf("diagAt at region end: %q", got)
	}
	if got := ls.diagAt(ln, textpos.Pos{Line: 2, Char: 18}); got != "" {
		t.Errorf("diagAt past region end: %q", got)
	}
	if got := ls.diagAt(ln, textpos.Pos{Line: 2, Char: 0}); got != "" {
		t.Errorf("diagAt outside region: %q", got)
	}
	if got := ls.diagAt(ln, textpos.Pos{Line: 0, Char: 0}); got != "" {
		t.Errorf("diagAt other line: %q", got)
	}
	if got := joinHover("error: bad", "docs here"); got != "error: bad\n\ndocs here" {
		t.Errorf("joinHover both: %q", got)
	}
	if got := joinHover("", "docs"); got != "docs" || joinHover("err", "") != "err" {
		t.Error("joinHover single-sided failed")
	}
	err := lang.Diagnostic{Severity: lang.DiagErr}
	warn := lang.Diagnostic{Severity: lang.DiagWarn}
	info := lang.Diagnostic{Severity: lang.DiagInfo}
	if got := diagSummary([]lang.Diagnostic{err, info}); got != "lsp: 1 error" {
		t.Errorf("diagSummary singular: %q", got)
	}
	if got := diagSummary([]lang.Diagnostic{err, warn, err, warn, warn}); got != "lsp: 2 errors, 3 warnings" {
		t.Errorf("diagSummary plural: %q", got)
	}
	if got := diagSummary([]lang.Diagnostic{info}); got != "" {
		t.Errorf("diagSummary none: %q", got)
	}
}

// TestErrorTagRenderVisible verifies the tag->rich-decoration plumbing:
// two Lines buffers with identical text and highlighting, one with a
// token.Error diagnostic tag, must produce different laid-out rich text
// (the dotted underline from highlighting.Properties for token.Error).
// This is what makes diagnostics visible to the user.
func TestErrorTagRenderVisible(t *testing.T) {
	const src = "some undefined symbol here\n"

	mk := func(tag bool) (*lines.Lines, int) {
		ln, vid := lines.NewLinesFromBytes("test.go", 80, []byte(src))
		ln.Highlighter.StyleName = "default"
		ln.Highlighter.Init(ln.FileInfo(), nil)
		if !ln.Highlighter.Has {
			t.Skip("no highlighting available for test setup")
		}
		if tag {
			ln.AddTag(0, 5, 23, token.Error)
		}
		ln.MarkupLines(0, 0)
		return ln, vid
	}

	plain, pvid := mk(false)
	tagged, tvid := mk(true)

	pt := plain.ViewMarkupLine(pvid, 0)
	tt := tagged.ViewMarkupLine(tvid, 0)
	if pt.String() == tt.String() {
		t.Fatalf("token.Error tag has no effect on rich markup: %q\n"+
			"diagnostics would not be flagged visually", tt.String())
	}
	nDeco := 0
	for si := 0; si < tt.NumSpans(); si++ {
		st, rr := tt.Span(si)
		if len(rr) > 0 && st.Decoration.HasFlag(rich.DottedUnderline) {
			nDeco++
		}
	}
	if nDeco == 0 {
		t.Fatalf("token.Error span carries no dotted-underline decoration: %q\n"+
			"diagnostics would not be flagged visually (check highlighting.Properties / MarkupLineRich)",
			tt.String())
	}
	// plain text must not be decorated:
	for si := 0; si < pt.NumSpans(); si++ {
		st, rr := pt.Span(si)
		if len(rr) > 0 && st.Decoration.HasFlag(rich.DottedUnderline) {
			t.Errorf("plain markup unexpectedly decorated at span %d", si)
		}
	}
}

// TestRenameResults verifies the grouping of rename edit previews into
// search.Results for the Find panel.
func TestRenameResults(t *testing.T) {
	prev := []*lsp.EditPreview{
		{Filename: "/x/a.go", Region: textpos.Region{Start: textpos.Pos{Line: 2, Char: 4}, End: textpos.Pos{Line: 2, Char: 10}}, Old: "smileX", New: "happy", LineText: "var smileX = 1"},
		{Filename: "/x/a.go", Region: textpos.Region{Start: textpos.Pos{Line: 3, Char: 8}, End: textpos.Pos{Line: 3, Char: 14}}, Old: "smileX", New: "happy", LineText: "var y = smileX"},
		{Filename: "/x/b.go", Region: textpos.Region{Start: textpos.Pos{Line: 2, Char: 8}, End: textpos.Pos{Line: 2, Char: 14}}, Old: "smileX", New: "happy", LineText: "var q = smileX"},
	}
	res := renameResults(prev)
	if len(res) != 2 {
		t.Fatalf("groups = %d, want 2", len(res))
	}
	if res[0].Filepath != "/x/a.go" || res[0].Count != 2 || len(res[0].Matches) != 2 {
		t.Errorf("group a: %+v", res[0])
	}
	if res[1].Filepath != "/x/b.go" || res[1].Count != 1 {
		t.Errorf("group b: %+v", res[1])
	}
	m := res[0].Matches[0]
	if m.TextMatch.Start != 4 || m.TextMatch.End != 10 {
		t.Errorf("TextMatch = %v, want {4 10}", m.TextMatch)
	}
	if string(m.Text) != "var smileX = 1" {
		t.Errorf("Text = %q", string(m.Text))
	}
	// multi-line region clamps to end of first line:
	prev2 := []*lsp.EditPreview{
		{Filename: "/x/c.go", Region: textpos.Region{Start: textpos.Pos{Line: 1, Char: 3}, End: textpos.Pos{Line: 2, Char: 5}}, Old: "a\nb", New: "x", LineText: "foo bar"},
	}
	r2 := renameResults(prev2)
	if len(r2) != 1 || r2[0].Matches[0].TextMatch.End != 7 {
		t.Errorf("multiline clamp: %+v", r2)
	}
}
