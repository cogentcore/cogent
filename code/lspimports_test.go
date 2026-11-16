// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"os"
	"os/exec"
	"testing"
	"time"

	"cogentcore.org/core/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCodeSaveFixesImports tests that saving a Go file fixes its imports
// and formats it with the language server, as goimports did: a missing
// import is added, an unused one removed, and the code formatted, in the
// buffer and the file that is saved.
func TestCodeSaveFixesImports(t *testing.T) {
	if _, err := exec.LookPath("gopls"); err != nil {
		t.Skip("gopls not found in PATH:", err)
	}
	t.Setenv("GOWORK", "off")
	tmp := tempDir(t)
	path := writeFiles(t, tmp, map[string]string{
		"go.mod":  "module demo\n\ngo 1.21\n",
		"main.go": "package main\n\nimport \"os\"\n\nfunc main() {\n  println(strings.ToUpper(\"a\"))\n}\n",
	})
	want := "package main\n\nimport \"strings\"\n\nfunc main() {\n\tprintln(strings.ToUpper(\"a\"))\n}\n"
	b, cv := newTestCode(t, tmp)
	showed := make(chan struct{}, 1)
	b.OnFinal(events.Show, func(e events.Event) { showed <- struct{}{} })
	b.RunWindow()
	<-showed
	var tv *TextEditor
	locked(b, func() {
		tv, _, _ = cv.ViewFileInIndex(path("main.go"), 0)
	})
	waitCode(t, 30*time.Second, "gopls loaded", func() bool {
		return len(cv.LangSvc().lspB.Symbols(tv.Lines)) > 0
	})
	locked(b, func() {
		tv.Lines.SetChanged(true) // (saving only saves changed files)
		cv.SetActiveEditorIndex(0)
		cv.SaveActiveView()
		assert.Equal(t, want, tv.Lines.String(), "the buffer")
	})
	saved, err := os.ReadFile(path("main.go"))
	require.NoError(t, err)
	assert.Equal(t, want, string(saved), "the saved file")
}
