// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/textpos"
	"github.com/stretchr/testify/assert"

	"cogentcore.org/cogent/code/lang"
)

// TestCodeGoal checks the language services for a Goal file in a real
// Code window, which come from goalls, a proxy for gopls for the Go code
// of the Goal file, built here: the errors of the Go code are shown in the
// Goal file, with the Go code of the lines with errors that is not the
// same as their source, hover docs, definitions, and inlay hints are for
// the Goal file, formatting formats it with its imports fixed, and the Go
// file is written when it is saved, which also formats the Goal file.
func TestCodeGoal(t *testing.T) {
	if _, err := exec.LookPath("gopls"); err != nil {
		t.Skip("gopls not found in PATH:", err)
	}
	bin := filepath.Join(t.TempDir(), "goalls")
	if out, err := exec.Command("go", "build", "-o", bin, "cogentcore.org/lab/goal/cmd/goalls").CombinedOutput(); err != nil {
		t.Fatalf("building goalls: %v\n%s", err, out)
	}
	prev := Settings.LSExecs
	Settings.LSExecs = map[string]string{"goalls": bin}
	defer func() { Settings.LSExecs = prev }()

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
		assert.True(t, ls.LSPActive(ln))
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
				assert.Contains(t, d.Msg, `Go: goalrun.Run("ls", "-la")`)
			}
		}
		assert.True(t, undefY, "undefined: y in %v", diags())
		assert.True(t, goalrun, "undefined: goalrun in %v", diags())

		b.AsyncLock()
		hover := ls.HoverAt(tv, textpos.Pos{Line: 6, Char: 12}) // Println
		b.AsyncUnlock()
		assert.Contains(t, hover, "Println")
		locs := ls.lspB.Definition(ln, textpos.Pos{Line: 6, Char: 14}) // x
		if assert.Len(t, locs, 1) {
			assert.True(t, strings.HasSuffix(locs[0].Filename, "main.goal"), locs[0].Filename)
			assert.Equal(t, textpos.Pos{Line: 5, Char: 1}, locs[0].Region.Start)
		}

		// the inlay hints, such as the type of x
		waitCode(t, 30*time.Second, "the inlay hints", func() bool {
			ls.InlayRefresh(ln)
			b.AsyncLock()
			defer b.AsyncUnlock()
			for _, vt := range ln.VirtualText(inlayLayer, -1) {
				if vt.Pos == (textpos.Pos{Line: 5, Char: 2}) && string(vt.Text.Join()) == " int" {
					return true
				}
			}
			return false
		})

		// formatting fixes the imports: strings is added
		b.AsyncLock()
		ln.InsertText(textpos.Pos{Line: 8}, []rune("\ts := strings.ToUpper(\"a\")\n\t_ = s\n"))
		ls.Change(ln)
		cv.LangFormat()
		b.AsyncUnlock()
		assert.Contains(t, ln.String(), "import (\n\t\"fmt\"\n\t\"strings\"\n)")

		// saving formats the Goal file, and writes the Go file
		b.AsyncLock()
		ln.InsertText(textpos.Pos{Line: 10}, []rune("\tz:=1\n\t_ = z\n"))
		cv.SaveActiveView()
		b.AsyncUnlock()
		saved, _ := os.ReadFile(fpath)
		assert.Contains(t, string(saved), "\tz := 1\n")
		gofile := filepath.Join(tmp, "main.go")
		waitCode(t, 30*time.Second, "the Go file", func() bool {
			b, err := os.ReadFile(gofile)
			return err == nil && strings.Contains(string(b), `strings.ToUpper("a")`)
		})
	})
}
