// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
	"image"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/text/textpos"
)

// TestCodeOccurrencesClick checks occurrence highlighting in a real Code
// window driven by mouse clicks, which move the cursor without Input or
// key events, including replacing the highlight of a jump to a location.
func TestCodeOccurrencesClick(t *testing.T) {
	if _, err := exec.LookPath("gopls"); err != nil {
		t.Skip("gopls not found in PATH:", err)
	}
	tmp := t.TempDir()
	t.Setenv("GOWORK", "off")
	os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module demo\n\ngo 1.21\n"), 0644)
	fpath := filepath.Join(tmp, "main.go")
	os.WriteFile(fpath, []byte("package main\n\nfunc main() {\n\tx := 1\n\tx++\n\t_ = x\n}\n"), 0644)
	b := core.NewBody()
	cv := NewCode(b)
	cv.Defaults()
	cv.ProjectRoot = core.Filename(tmp)
	cv.Settings.ProjectRoot = core.Filename(tmp)
	// the window is only used for its event handling, not its image,
	// which varies with timing:
	imagex.UpdateTestImages = true
	defer func() { imagex.UpdateTestImages = false }()
	b.AssertRender(t, "occurrences-click", func() {
		jump := textpos.Region{Start: textpos.Pos{Line: 2, Char: 5}, End: textpos.Pos{Line: 2, Char: 9}}
		b.AsyncLock()
		tv, ok := cv.OpenFileAtRegion(fpath, jump)
		b.AsyncUnlock()
		if !ok {
			t.Fatal("file not opened")
		}
		waitCode(t, 30*time.Second, "gopls loaded", func() bool {
			return len(cv.LangSvc().lspB.Symbols(tv.Lines)) > 0
		})
		highlights := func() []textpos.Region {
			b.AsyncLock()
			defer b.AsyncUnlock()
			return append([]textpos.Region{}, tv.Highlights...)
		}
		if hl := highlights(); len(hl) != 1 || hl[0] != jump {
			t.Fatalf("jump highlight: %v", hl)
		}
		b.AsyncLock()
		pt := tv.CharStartPos(textpos.Pos{Line: 4, Char: 1}).ToPoint() // the x in x++
		b.AsyncUnlock()
		pt.Y += 4
		src := b.Scene.RenderWindow().SystemWindow.Events()
		src.MouseButton(events.MouseDown, events.Left, pt, 0)
		src.MouseButton(events.MouseUp, events.Left, pt, 0)
		waitCode(t, 5*time.Second, "occurrences of x highlighted", func() bool {
			return len(highlights()) == 3
		})
	})
}

// TestCodeSymbolsClick checks that clicking a symbol in the Symbols panel
// goes to it with a single click, in the same file and in another file,
// and that the symbol tree is not replaced when the editor gets focus
// (which used to happen on every focus change, losing clicks).
func TestCodeSymbolsClick(t *testing.T) {
	if _, err := exec.LookPath("gopls"); err != nil {
		t.Skip("gopls not found in PATH:", err)
	}
	tmp := t.TempDir()
	t.Setenv("GOWORK", "off")
	os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module demo\n\ngo 1.21\n"), 0644)
	fpath := filepath.Join(tmp, "main.go")
	os.WriteFile(fpath, []byte("package main\n\ntype T struct {\n\tF int\n}\n\nfunc main() {}\n"), 0644)
	os.WriteFile(filepath.Join(tmp, "other.go"), []byte("package main\n\n// Other is other.\nfunc Other() {}\n"), 0644)
	b := core.NewBody()
	b.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(1000), units.Dp(1000)) }) // room for the panel
	cv := NewCode(b)
	cv.Defaults()
	cv.ProjectRoot = core.Filename(tmp)
	cv.Settings.ProjectRoot = core.Filename(tmp)
	cv.Settings.Symbols.Scope = SymScopePackage
	imagex.UpdateTestImages = true
	defer func() { imagex.UpdateTestImages = false }()
	b.AssertRender(t, "symbols-click", func() {
		b.AsyncLock()
		tv, _, ok := cv.ViewFile(core.Filename(fpath))
		b.AsyncUnlock()
		if !ok {
			t.Fatal("file not opened")
		}
		waitCode(t, 30*time.Second, "gopls loaded", func() bool {
			return len(cv.LangSvc().lspB.Symbols(tv.Lines)) > 0
		})
		b.AsyncLock()
		cv.Symbols()
		sv := cv.Tabs().TabByName("Symbols").Child(0).(*SymbolsPanel)
		symTree := sv.symTree
		b.AsyncUnlock()
		click := func(name, file string, line int) {
			t.Helper()
			var node *core.Tree
			var bb image.Rectangle
			waitCode(t, 5*time.Second, "node "+name+" laid out", func() bool {
				b.AsyncLock()
				defer b.AsyncUnlock()
				sv.symTree.WalkDown(func(n tree.Node) bool {
					if tr := core.AsTree(n); tr != nil && tr.Name == "tv_"+name {
						node = tr
					}
					return tree.Continue
				})
				if node != nil && node.Parts != nil {
					bb = node.Parts.Geom.TotalBBox // the node's own row
				}
				return bb.Dx() > 0
			})
			pt := bb.Min.Add(bb.Size().Div(2))
			src := b.Scene.RenderWindow().SystemWindow.Events()
			src.MouseButton(events.MouseDown, events.Left, pt, 0)
			src.MouseButton(events.MouseUp, events.Left, pt, 0)
			waitCode(t, 5*time.Second, "went to "+name, func() bool {
				b.AsyncLock()
				defer b.AsyncUnlock()
				ae := cv.ActiveEditor()
				return ae != nil && ae.Lines != nil && filepath.Base(ae.Lines.Filename()) == file && ae.CursorPos.Line == line
			})
		}
		click("main()", "main.go", 6)
		click("Other()", "other.go", 3)
		click("T", "main.go", 2)
		b.AsyncLock()
		same := sv.symTree == symTree
		b.AsyncUnlock()
		if !same {
			t.Error("symbol tree was replaced")
		}
	})
}
