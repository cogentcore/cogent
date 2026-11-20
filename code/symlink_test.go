// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/text/textpos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCodeSymlinkPaths tests that a file of a project opened by a path
// through a symbolic link, as ~/proj for ~/src/proj, is the same open file
// as the one opened from the file tree, whose root has its links evaluated,
// and that Find finds it once. It was opened a second time, and Find found
// it under both paths, and on disk.
func TestCodeSymlinkPaths(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symbolic links")
	}
	tmp := tempDir(t)
	real := filepath.Join(tmp, "real")
	path := writeFiles(t, real, map[string]string{"a.txt": "apple pie\n", "b.txt": "apple cider\n"})
	link := filepath.Join(tmp, "link")
	require.NoError(t, os.Symlink(real, link))
	b, cv := newTestCode(t, link)
	core.RenderImage(b, func() {
		locked(b, func() {
			require.Equal(t, real, string(cv.Files.Filepath))
			cv.ViewFileInIndex(path("a.txt"), 0)                // as from the tree
			cv.ViewFileInIndex(filepath.Join(link, "a.txt"), 0) // the same file
			cv.ViewFileInIndex(filepath.Join(link, "b.txt"), 0)
			var names []string
			for _, ln := range cv.OpenFiles.Values {
				names = append(names, ln.Filename())
			}
			assert.ElementsMatch(t, []string{path("a.txt"), path("b.txt")}, names)
			assert.NotNil(t, cv.GetOpenFile(filepath.Join(link, "a.txt")))

			cv.Find("apple", "", textpos.UseCase, textpos.Literal, All, nil)
			fv := cv.Tabs().TabByName("Find").Child(0).(*FindPanel)
			var found []string
			for _, r := range findResults(fv) {
				found = append(found, r.Filepath)
			}
			assert.ElementsMatch(t, []string{path("a.txt"), path("b.txt")}, found)
		})
	})
}
