// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package book

import (
	"image"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"cogentcore.org/cogent/author"
	"cogentcore.org/core/base/iox/imagex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeBook writes the files of a small book in a new temporary
// directory, which becomes the current directory.
func writeBook(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Chdir(dir)
	goalrun.Config.Dir = dir // as Book does
	goalrun.Errors = nil
	files := map[string]string{
		"metadata.yaml": "title: Test Book\nlanguage: en\npublisher: Me\ngenre: Test\n" +
			"identifier: 1234\nidentifier_scheme: ISBN\ncopyright:\n  year: 2026\n  owner: Me\n",
		"frontmatter.md": "# Preface {-}\n\nA preface.\n",
		"chapter-02.md":  "# Second\n\nMore, as in [@Smith2020].\n",
		"chapter-01.md":  "# First\n\nThe start.\n",
		"endmatter.md":   "# Epilogue {-}\n\nThe end.\n",
		"appendix-a.md":  "# Appendix\n",
		"glossary.md":    "# Glossary {-}\n\nterm\n: a word\n",
		"allrefs.bib":    "@article{Smith2020,\n  author = {Smith, A.},\n  title = {First},\n  year = {2020}\n}\n",
	}
	for name, s := range files {
		require.NoError(t, os.WriteFile(name, []byte(s), 0o644))
	}
	require.NoError(t, imagex.Save(image.NewRGBA(image.Rect(0, 0, 10, 10)), "cover.png"))
	return dir
}

// TestMarkdown tests combining the files of a book into one markdown file,
// with the chapters in order, and the references.
func TestMarkdown(t *testing.T) {
	writeBook(t)
	bk := NewBookData("test")
	require.NoError(t, bk.getMetadata())
	assert.Equal(t, "Test Book", bk.Metadata["title"])
	bk.savePandocInputs()
	assert.FileExists(t, bk.pdi("html.css"))
	require.NoError(t, bk.Refs())
	assert.FileExists(t, "references.bib")
	fn := bk.Markdown()
	assert.Equal(t, []string{"chapter-01.md", "chapter-02.md"}, bk.Chapters)
	assert.Equal(t, []string{"appendix-a.md"}, bk.Appendicies)
	b, err := os.ReadFile(fn)
	require.NoError(t, err)
	s := string(b)
	assert.True(t, strings.HasPrefix(s, "---\ntitle: Test Book"), s)
	assert.Less(t, strings.Index(s, "# First"), strings.Index(s, "# Second"))
	assert.Contains(t, s, `id="glossary"`)
	assert.Contains(t, s, "::: {#refs}")
}

// TestBook tests making the HTML, LaTeX, ePUB, and DOCX versions of a
// book with pandoc.
func TestBook(t *testing.T) {
	if _, err := exec.LookPath("pandoc-crossref"); err != nil {
		t.Skip("pandoc and pandoc-crossref are not installed")
	}
	dir := writeBook(t)
	c := &author.Config{Formats: []author.Formats{author.HTML, author.LaTeX, author.EPUB, author.DOCX}}
	require.NoError(t, Book(c))
	for _, ext := range []string{".html", ".tex", ".epub", ".docx"} {
		assert.FileExists(t, filepath.Join(dir, "book"+ext))
	}
	b, err := os.ReadFile("book.html")
	require.NoError(t, err)
	assert.Contains(t, string(b), "Smith", "the reference")
	assert.Contains(t, string(b), "The start", "the chapters")

	require.NoError(t, os.Remove("metadata.yaml"))
	assert.Error(t, Book(c), "no metadata")
}
