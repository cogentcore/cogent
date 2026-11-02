// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/textpos"
)

// TestCodeInlayHints checks that the inlay hints of gopls are shown as
// virtual text in a real Code window when the InlayHints setting is
// turned on, with gopls running, and removed when it is turned off.
func TestCodeInlayHints(t *testing.T) {
	if _, err := exec.LookPath("gopls"); err != nil {
		t.Skip("gopls not found in PATH:", err)
	}
	tmp := t.TempDir()
	t.Setenv("GOWORK", "off")
	os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module demo\n\ngo 1.21\n"), 0644)
	fpath := filepath.Join(tmp, "main.go")
	os.WriteFile(fpath, []byte("package main\n\nimport \"strings\"\n\nfunc main() {\n\tx := 1\n\t_, _ = x, strings.Split(\"a,b\", \",\")\n}\n"), 0644)
	b := core.NewBody()
	b.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(1000), units.Dp(600)) }) // to see the hints
	cv := NewCode(b)
	cv.Defaults()
	cv.ProjectRoot = core.Filename(tmp)
	cv.Settings.ProjectRoot = core.Filename(tmp)
	imagex.UpdateTestImagesFor(t) // the image is for looking at
	b.AssertRender(t, "inlay-hints", func() {
		b.AsyncLock()
		tv, _, ok := cv.ViewFile(core.Filename(fpath))
		b.AsyncUnlock()
		if !ok {
			t.Fatal("file not opened")
		}
		ln := tv.Lines
		hints := func() map[textpos.Pos]string {
			b.AsyncLock()
			defer b.AsyncUnlock()
			m := map[textpos.Pos]string{}
			for _, vt := range ln.VirtualText(inlayLayer, -1) {
				m[vt.Pos] = string(vt.Text.Join())
			}
			return m
		}
		// turning the setting on with gopls running configures it then,
		// and they are shown without any change
		waitCode(t, 60*time.Second, "gopls loaded", func() bool {
			return len(cv.LangSvc().lspB.Symbols(ln)) > 0
		})
		b.AsyncLock()
		cv.SetInlayHints(true)
		b.AsyncUnlock()
		waitCode(t, 60*time.Second, "inlay hints", func() bool {
			return len(hints()) > 0
		})
		hs := hints()
		// the parameter names of strings.Split, and the type of x
		assert := func(pos textpos.Pos, txt string) {
			t.Helper()
			if hs[pos] != txt {
				t.Errorf("hint at %v: %q, want %q (all: %v)", pos, hs[pos], txt, hs)
			}
		}
		assert(textpos.Pos{Line: 6, Char: 25}, "s: ")
		assert(textpos.Pos{Line: 6, Char: 32}, "sep: ")
		assert(textpos.Pos{Line: 5, Char: 2}, " int")
		if s := string(ln.Line(6)); s != "\t_, _ = x, strings.Split(\"a,b\", \",\")" {
			t.Errorf("the hints are display-only, but the text is %q", s)
		}

		b.AsyncLock()
		cv.SetInlayHints(false)
		b.AsyncUnlock()
		waitCode(t, 30*time.Second, "inlay hints removed", func() bool {
			return len(hints()) == 0
		})
		b.AsyncLock()
		cv.SetInlayHints(true)
		b.AsyncUnlock()
		waitCode(t, 30*time.Second, "inlay hints shown again", func() bool {
			return len(hints()) > 0
		})
	})
}
