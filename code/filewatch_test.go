// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/core"
	"cogentcore.org/core/text/textpos"
	"github.com/stretchr/testify/assert"
)

// TestCodeFilesChanged changes and deletes open files on disk in a real
// Code window: the file tree watcher reloads files that have not been
// edited and closes them if deleted, and prompts for edited files.
// The temporary directory is a symbolic link on macOS, which the file
// tree evaluates but the file names of the open buffers do not.
func TestCodeFilesChanged(t *testing.T) {
	tmp := t.TempDir()
	path := func(name string) string { return filepath.Join(tmp, name) }
	for _, name := range []string{"a.txt", "b.txt", "c.txt", "d.txt"} {
		os.WriteFile(path(name), []byte(name+"\n"), 0644)
	}
	b := core.NewBody()
	cv := NewCode(b)
	cv.Defaults()
	cv.ProjectRoot = core.Filename(tmp)
	cv.Settings.ProjectRoot = core.Filename(tmp)
	// the window is only used for its event handling, not its image:
	imagex.UpdateTestImages = true
	defer func() { imagex.UpdateTestImages = false }()
	b.AssertRender(t, "files-changed", func() {
		b.AsyncLock()
		ta, _, _ := cv.ViewFileInIndex(path("a.txt"), 0)
		tb, _, _ := cv.ViewFileInIndex(path("b.txt"), 1)
		lna, lnb := ta.Lines, tb.Lines
		lnc, _ := cv.RecycleFile(path("c.txt"))
		lnd, _ := cv.RecycleFile(path("d.txt"))
		var prompted atomic.Bool
		lnb.FileModPromptFunc = func() { prompted.Store(true) }
		lnb.InsertText(textpos.Pos{}, []rune("edit "))
		lnd.InsertText(textpos.Pos{}, []rune("edit "))
		b.AsyncUnlock()
		time.Sleep(500 * time.Millisecond) // watcher start

		// not edited: reloaded
		os.WriteFile(path("a.txt"), []byte("changed a\n"), 0644)
		waitCode(t, 5*time.Second, "a.txt reloaded", func() bool {
			return lna.String() == "changed a\n"
		})
		// edited and shown: prompted, not reloaded
		os.WriteFile(path("b.txt"), []byte("changed b\n"), 0644)
		waitCode(t, 5*time.Second, "b.txt prompt", prompted.Load)
		assert.Equal(t, "edit b.txt\n", lnb.String())

		// not edited: closed
		os.Remove(path("c.txt"))
		waitCode(t, 5*time.Second, "c.txt closed", func() bool {
			b.AsyncLock()
			defer b.AsyncUnlock()
			return cv.OpenFiles.IndexByKey(path("c.txt")) < 0
		})
		assert.Equal(t, "", lnc.Filename())

		// edited: prompted to close, still open
		os.Remove(path("d.txt"))
		waitCode(t, 5*time.Second, "d.txt delete prompt", func() bool {
			b.AsyncLock()
			defer b.AsyncUnlock()
			return cv.deletedAsked[lnd]
		})
		b.AsyncLock()
		assert.GreaterOrEqual(t, cv.OpenFiles.IndexByKey(path("d.txt")), 0)
		b.AsyncUnlock()
	})
}
