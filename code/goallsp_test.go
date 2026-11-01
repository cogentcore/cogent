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

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/parse/complete"
	"cogentcore.org/core/text/textpos"
	"github.com/stretchr/testify/assert"

	"cogentcore.org/cogent/code/lang"
)

// TestCodeGoal checks the language services for a Goal file in a real
// Code window, which come from gopls for its Go code: the errors of the
// Go code are shown in the Goal file, with the Go code of the lines with
// errors that is not the same as their source, hover docs and definitions
// are for the Goal file, and the Go file is written when it is saved.
func TestCodeGoal(t *testing.T) {
	if _, err := exec.LookPath("gopls"); err != nil {
		t.Skip("gopls not found in PATH:", err)
	}
	tmp := t.TempDir()
	t.Setenv("GOWORK", "off")
	os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module demo\n\ngo 1.21\n"), 0644)
	fpath := filepath.Join(tmp, "main.goal")
	src := "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tx := 1\n\tfmt.Println(x, y)\n\tls -la\n}\n"
	os.WriteFile(fpath, []byte(src), 0644)
	b := core.NewBody()
	b.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(1000), units.Dp(600)) }) // to see the code
	cv := NewCode(b)
	cv.Defaults()
	cv.ProjectRoot = core.Filename(tmp)
	cv.Settings.ProjectRoot = core.Filename(tmp)
	cv.Settings.InlayHints = true
	imagex.UpdateTestImages = true // the image is for looking at
	defer func() { imagex.UpdateTestImages = false }()
	b.AssertRender(t, "goal", func() {
		b.AsyncLock()
		tv, _, ok := cv.ViewFile(core.Filename(fpath))
		b.AsyncUnlock()
		if !ok {
			t.Fatal("file not opened")
		}
		ln := tv.Lines
		ls := cv.LangSvc()
		diags := func() []lang.Diagnostic {
			ls.mu.Lock()
			defer ls.mu.Unlock()
			return ls.diagCache[fpath]
		}
		waitCode(t, 60*time.Second, "the errors of the Go code", func() bool {
			return len(diags()) >= 2
		})
		var undefY, goalrun bool
		for _, d := range diags() {
			switch {
			case strings.Contains(d.Msg, "undefined: y"):
				undefY = true
				assert.Equal(t, textpos.Region{Start: textpos.Pos{Line: 6, Char: 16}, End: textpos.Pos{Line: 6, Char: 17}}, d.Region)
			case strings.Contains(d.Msg, "undefined: goalrun"):
				goalrun = true
				assert.Equal(t, 7, d.Region.Start.Line)
			}
		}
		assert.True(t, undefY, "undefined: y in %v", diags())
		assert.True(t, goalrun, "undefined: goalrun in %v", diags())
		b.AsyncLock()
		vts := ln.VirtualText(goalLayer, -1)
		b.AsyncUnlock()
		if assert.Len(t, vts, 1, "the Go code of the command") {
			assert.Equal(t, textpos.Pos{Line: 7, Char: 7}, vts[0].Pos)
			assert.Equal(t, `  Go: goalrun.Run("ls", "-la")`, string(vts[0].Text.Join()))
		}

		b.AsyncLock()
		hover := ls.HoverAt(tv, textpos.Pos{Line: 6, Char: 12}) // Println
		b.AsyncUnlock()
		assert.Contains(t, hover, "Println")
		b.AsyncLock()
		tln, tpos, ok := ls.lspTarget(ln, textpos.Pos{Line: 6, Char: 14}) // x
		b.AsyncUnlock()
		assert.True(t, ok)
		locs := ls.fromTargets(ls.lspB.Definition(tln, tpos))
		if assert.Len(t, locs, 1) {
			assert.Equal(t, fpath, locs[0].Filename)
			assert.Equal(t, textpos.Pos{Line: 5, Char: 1}, locs[0].Region.Start)
		}

		// the inlay hints of the Go code, such as the type of x
		waitCode(t, 30*time.Second, "the inlay hints", func() bool {
			b.AsyncLock()
			defer b.AsyncUnlock()
			for _, vt := range ln.VirtualText(inlayLayer, -1) {
				if vt.Pos == (textpos.Pos{Line: 5, Char: 2}) && string(vt.Text.Join()) == " int" {
					return true
				}
			}
			return false
		})

		// the completions of the Go code
		b.AsyncLock()
		ln.InsertText(textpos.Pos{Line: 8}, []rune("\ts := fmt.Spr\n"))
		ms := tv.Complete.MatchFunc(nil, "\ts := fmt.Spr", 8, 13)
		ln.DeleteText(textpos.Pos{Line: 8}, textpos.Pos{Line: 9})
		b.AsyncUnlock()
		assert.True(t, slices.ContainsFunc(ms.Matches, func(c complete.Completion) bool { return c.Label == "Sprintf" }), "Sprintf in %v", ms.Matches)

		// fixing the error removes it
		b.AsyncLock()
		ln.ReplaceText(textpos.Pos{Line: 6, Char: 16}, textpos.Pos{Line: 6, Char: 17}, textpos.Pos{Line: 6, Char: 16}, "x", lines.ReplaceNoMatchCase)
		ls.Input(ln)
		b.AsyncUnlock()
		waitCode(t, 30*time.Second, "the fixed error", func() bool {
			for _, d := range diags() {
				if strings.Contains(d.Msg, "undefined: y") {
					return false
				}
			}
			return true
		})

		// saving writes the Go file
		b.AsyncLock()
		cv.SaveActiveView()
		b.AsyncUnlock()
		gofile := filepath.Join(tmp, "main.go")
		waitCode(t, 30*time.Second, "the Go file", func() bool {
			b, err := os.ReadFile(gofile)
			return err == nil && strings.Contains(string(b), "fmt.Println(x, x)")
		})
	})
}
