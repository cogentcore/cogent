// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package refs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testBib is a BibTeX file with three references.
const testBib = `@article{Smith2020,
  author = {Smith, A.},
  title = {First},
  year = {2020}
}

@book{Jones-2019,
  author = {Jones, B.},
  title = {Second},
  year = {2019}
}

@article{Unused2000,
  author = {Nobody},
  title = {Third},
  year = {2000}
}
`

// TestBibTexCited tests extracting the references cited in markdown files
// from a BibTeX file, and its errors.
func TestBibTexCited(t *testing.T) {
	dir := t.TempDir()
	write := func(name, s string) string {
		fn := filepath.Join(dir, name)
		require.NoError(t, os.WriteFile(fn, []byte(s), 0o644))
		return fn
	}
	src := write("all.bib", testBib)
	write("a.md", "As shown [@Smith2020; @Jones-2019], and [@Smith2020].\n")
	write("b.md", "Missing [@Missing1999].\n")
	out := filepath.Join(dir, "out.bib")
	require.NoError(t, BibTexCited(dir, src, out, true))
	b, err := os.ReadFile(out)
	require.NoError(t, err)
	s := string(b)
	assert.Contains(t, s, "Smith2020")
	assert.Contains(t, s, "Jones-2019")
	assert.NotContains(t, s, "Unused2000", "only the cited references")

	assert.Error(t, BibTexCited(dir, "", out, false), "no source")
	assert.Error(t, BibTexCited(t.TempDir(), src, out, false), "no markdown files")
	assert.Error(t, BibTexCited(dir, filepath.Join(dir, "none.bib"), out, false))
	assert.Error(t, BibTexCited(dir, write("bad.bib", "@article{x, title = {"), out, false), "invalid")
	assert.Error(t, BibTexCited(dir, src, filepath.Join(dir, "none", "out.bib"), false), "no output directory")
}
