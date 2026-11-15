// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"os"
	"path/filepath"
	"testing"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"github.com/stretchr/testify/require"
)

// newTestCode returns a new body with a [Code] for the given project
// directory, to be run with [core.AssertRender] or [core.RenderImage],
// whose window is only used for its event handling, not its image.
// At the end of the test, before the directory is removed, the window
// is closed without saving its open files, which stops watching its
// files: an edited file prompts to save instead, which keeps the window
// open, and its file tree logs errors for the removed files during the
// later tests.
func newTestCode(t *testing.T, dir string) (*core.Body, *Code) {
	t.Helper()
	b := core.NewBody()
	// (large enough for the lines that are clicked to be shown)
	b.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(1200), units.Dp(800)) })
	cv := NewCode(b)
	cv.Defaults()
	cv.ProjectRoot = core.Filename(dir)
	cv.Settings.ProjectRoot = core.Filename(dir)
	imagex.UpdateTestImagesFor(t)
	// the keys and projects of Code set GO111MODULE (see [SetGoMod])
	t.Setenv("GO111MODULE", os.Getenv("GO111MODULE"))
	t.Cleanup(core.TrackWindows()) // (after the close below)
	t.Cleanup(func() { closeTestCode(b, cv) })
	return b, cv
}

// closeTestCode closes the window of the given [Code], if it is still
// open, without saving its open files. The close at the end of
// [core.AssertRender] opens a dialog to save the edited files instead,
// which ignores further closes until one of its buttons is clicked.
func closeTestCode(b *core.Body, cv *Code) {
	if !b.AsyncLockIfShown() {
		return
	}
	defer b.AsyncUnlock()
	if bt := core.ButtonByText(core.TopScene(b), "Close without saving"); bt != nil {
		bt.Send(events.Click)
		return
	}
	for _, ln := range cv.OpenFiles.Values {
		ln.ClearNotSaved()
	}
	b.Close()
}

// writeFiles writes the given files, by relative path, in the given
// directory, returning a function for the absolute path of a file.
func writeFiles(t *testing.T, dir string, files map[string]string) func(name string) string {
	t.Helper()
	path := func(name string) string { return filepath.Join(dir, filepath.FromSlash(name)) }
	for name, text := range files {
		require.NoError(t, os.MkdirAll(filepath.Dir(path(name)), 0o755))
		require.NoError(t, os.WriteFile(path(name), []byte(text), 0o644))
	}
	return path
}

// locked runs the given function with the window of the given body
// locked, as for the event loop.
func locked(b *core.Body, fun func()) {
	b.AsyncLock()
	defer b.AsyncUnlock()
	fun()
}

// tempDir returns a new temporary directory for the test, with its
// symbolic links evaluated (as on macOS), as in the file tree, so that
// the names of the files opened from it are the same as those of the tree.
func tempDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	return dir
}

// windowBody returns the body of the open window with the given title,
// or nil if there is none.
func windowBody(title string) *core.Body {
	for _, w := range core.AllRenderWindows {
		if sc := w.MainScene(); sc != nil && sc.Body != nil && sc.Body.Title == title {
			return sc.Body
		}
	}
	return nil
}
