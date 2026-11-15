// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"os/exec"
	"strings"
	"testing"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/text/textpos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// lspProgram is the program for the tests of the language server commands.
const lspProgram = `package main

import "fmt"

type Shape interface{ Area() float64 }

type Sq struct{ S float64 }

func (s Sq) Area() float64 { return s.S * s.S }

func area(sh Shape) float64 {
	return sh.Area()
}

func main() {
	sq := Sq{S: 2}
	fmt.Println(area(sq))
	var unused int
}
`

// posOf returns the position of the n-th (from 0) occurrence of the
// given text in the given source, plus the given number of characters.
func posOf(src, text string, n, plus int) textpos.Pos {
	i := 0
	for range n + 1 {
		j := strings.Index(src[i:], text)
		if j < 0 {
			panic("posOf: no " + text)
		}
		i += j + 1
	}
	i--
	ln := strings.Count(src[:i], "\n")
	ch := i - (strings.LastIndex(src[:i], "\n") + 1)
	return textpos.Pos{Line: ln, Char: ch + plus}
}

// TestCodeLSPCommands tests the commands of Code that use the language
// server (gopls): going to definitions, finding references and
// implementations, problems, renaming, signature help, inlay hints,
// selections, quick fixes, and workspace symbols.
func TestCodeLSPCommands(t *testing.T) {
	if _, err := exec.LookPath("gopls"); err != nil {
		t.Skip("gopls not found in PATH:", err)
	}
	t.Setenv("GOWORK", "off")
	tmp := tempDir(t)
	path := writeFiles(t, tmp, map[string]string{
		"go.mod":  "module demo\n\ngo 1.21\n",
		"main.go": lspProgram,
	})
	b, cv := newTestCode(t, tmp)
	core.AssertRender(t, b, "lsp-commands", func() {
		var tv *TextEditor
		locked(b, func() {
			tv, _, _ = cv.ViewFileInIndex(path("main.go"), 0)
		})
		waitCode(t, 30*time.Second, "gopls loaded", func() bool {
			return len(cv.LangSvc().lspB.Symbols(tv.Lines)) > 0
		})
		waitCode(t, 30*time.Second, "the unused variable problem", func() bool {
			b.AsyncLock()
			defer b.AsyncUnlock()
			return len(cv.LangSvc().diagsIn(tv.Lines, textpos.Region{End: textpos.Pos{Line: 100}})) > 0
		})
		at := func(text string, n, plus int) {
			cv.ViewFileInIndex(path("main.go"), 0)
			cv.SetActiveEditorIndex(0)
			tv.SelectReset()
			tv.SetCursorShow(posOf(lspProgram, text, n, plus))
		}
		cursorLine := func() int { return cv.ActiveEditor().CursorPos.Line }

		locked(b, func() {
			at("area(sq)", 0, 1)
			cv.GotoDefinition()
			assert.Equal(t, posOf(lspProgram, "func area", 0, 0).Line, cursorLine(), "the definition")
			at("sq)", 0, 1)
			cv.GotoTypeDefinition()
			assert.Equal(t, posOf(lspProgram, "type Sq", 0, 0).Line, cursorLine(), "the type")
			at("Shape interface", 0, 1)
			cv.FindImplementations()
			assert.Equal(t, posOf(lspProgram, "type Sq", 0, 0).Line, cursorLine(), "the implementation")
			at("Sq{S", 0, 0)
			cv.WorkspaceSymbolPrompt() // a dialog
			cv.workspaceSymbolsMenu(tv, "Sq")
			cv.workspaceSymbolsMenu(tv, "NoSuchSymbol")

			at("Area()", 1, 1)
			cv.FindReferences()
			refs := cv.tabEditor(referencesTab)
			require.NotNil(t, refs)
			assert.Contains(t, refs.Lines.String(), "main.go:")

			cv.ShowProblems(File)
			probs := cv.tabEditor(problemsTab)
			require.NotNil(t, probs)
			assert.Contains(t, probs.Lines.String(), "declared and not used")
			cv.ShowProblems(All)
			cv.NextProblem()
			assert.Equal(t, posOf(lspProgram, "var unused", 0, 0).Line, cursorLine(), "the problem")

			// quick fixes of the problem
			at("unused int", 0, 1)
			m := core.NewScene()
			cv.LangSvc().codeActionsMenu(m, tv)
			cv.LangSvc().quickFixButtons(m, tv)
			cv.QuickFix() // a dialog
			cv.LangSvc().Lookup(tv)

			at("sq :=", 0, 1)
			cv.RenameSymbol() // a dialog
			cv.LangSvc().previewRename(tv.Lines, posOf(lspProgram, "sq :=", 0, 1), "sq", "square")
			fv := cv.Tabs().TabByName("Find").Child(0).(*FindPanel)
			assert.True(t, fv.Renaming)
			assert.Equal(t, 2, len(findResults(fv)[0].Matches), "the variable and its use")

			at("area(sq)", 0, 6)
			cv.ShowSignatureHelp() // a dialog
			assert.NotNil(t, windowBody(signatureWindow+": area"))
			at("package", 0, 0)
			cv.ShowSignatureHelp() // none

			cv.ShowInlayHints() // not enabled
			cv.SetInlayHints(true)
		})
		locked(b, func() {
			cv.ShowInlayHints()
			cv.SetInlayHints(false)

			at("s.S * s.S", 0, 2)
			cv.ExpandSelection()
			assert.True(t, tv.HasSelection())
			cv.ExpandSelection()
			cv.ShrinkSelection()
			assert.True(t, tv.HasSelection())

			cv.LangFormat()
			cv.Lookup()
		})
	})
}
