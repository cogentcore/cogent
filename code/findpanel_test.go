// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"testing"
	"time"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/metadata"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/text/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// findResults returns the results shown in the Find panel.
func findResults(fv *FindPanel) []search.Results {
	res, _ := metadata.Get[[]search.Results](fv.TextEditor().Lines, "SearchResults")
	return res
}

// TestFindReplace tests finding in the files of each location, with and
// without regular expressions and in open files, going to the results,
// and replacing them one at a time and all at once.
func TestFindReplace(t *testing.T) {
	tmp := tempDir(t)
	path := writeFiles(t, tmp, map[string]string{
		"a.txt":     "apple pie\nbanana apple\n",
		"b.txt":     "Apple cider\n",
		"sub/c.txt": "apple in sub\n",
		"d.go":      "package d // apple\n",
	})
	b, cv := newTestCode(t, tmp)
	core.AssertRender(t, b, "find-replace", func() {
		var fv *FindPanel
		find := func(find, repl string, ignoreCase, re bool, loc Locations, langs ...fileinfo.Known) []search.Results {
			locked(b, func() {
				if loc == File || loc == Dir { // (each find shows its first result)
					cv.ViewFileInIndex(path("a.txt"), 0)
					cv.SetActiveEditorIndex(0)
				}
				cv.Find(find, repl, ignoreCase, re, loc, langs)
				fv = cv.Tabs().TabByName("Find").Child(0).(*FindPanel)
			})
			return findResults(fv)
		}
		count := func(res []search.Results) int {
			n := 0
			for _, r := range res {
				n += r.Count
			}
			return n
		}
		locked(b, func() { cv.ViewFileInIndex(path("a.txt"), 0) })

		assert.Equal(t, 5, count(find("apple", "pear", true, false, All)), "all, ignoring case")
		assert.Equal(t, 4, count(find("apple", "pear", false, false, All)), "case")
		assert.Equal(t, 1, count(find("apple", "pear", false, false, All, fileinfo.Go)), "only in Go files")
		assert.Equal(t, 2, count(find("apple", "pear", false, false, File)), "in the open file")
		assert.Equal(t, 1, count(find("^banana", "", false, true, File)), "regexp in the file")
		assert.Equal(t, 3, count(find("app.e", "", false, true, Dir)), "regexp in the directory of the open file")
		res := find("apple", "", false, false, Open)
		assert.Equal(t, 3, count(res), "in the open directories (the root), with the open file")
		assert.Equal(t, path("a.txt"), res[0].Filepath, "the open file first")
		find("", "", false, false, All) // nothing
		find("(", "", false, true, All) // invalid: a snackbar

		locked(b, func() {
			fv.Params().Regexp = true
			fv.Params().Find = "("
			assert.False(t, fv.CompileRegexp())
			assert.False(t, fv.CheckValidRegexp())
			fv.Params().Regexp = false
			assert.True(t, fv.CompileRegexp())
			assert.Nil(t, fv.Re)
		})

		// the find shows the first result in its file
		res = find("apple", "pear", false, false, File)
		require.Len(t, res, 1)
		locked(b, func() {
			fv.NextFind()
			fv.PrevFind()
			fv.FindAction() // again
		})
		// replace one at a time
		ln := cv.GetOpenFile(path("a.txt"))
		locked(b, func() { assert.True(t, fv.ReplaceAction(), "more to replace") })
		assert.Equal(t, "pear pie\nbanana apple\n", ln.String())
		locked(b, func() { assert.False(t, fv.ReplaceAction(), "the last one") })
		assert.Equal(t, "pear pie\nbanana pear\n", ln.String())

		// regexp replace all, in the file
		find(`(\w+) pie`, "${1} tart", false, true, File)
		locked(b, fv.ReplaceAll)
		waitCode(t, 5*time.Second, "replaced all", func() bool {
			b.AsyncLock()
			defer b.AsyncUnlock()
			return ln.String() == "pear tart\nbanana pear\n"
		})

		// the confirmation dialog of replace all
		find("banana", "kiwi", false, false, File)
		locked(b, fv.ReplaceAllAction)
		var ok *core.Button
		locked(b, func() { ok = core.ButtonByText(core.TopScene(b), "Replace all") })
		require.NotNil(t, ok)
		locked(b, func() { ok.Send(events.Click) })
		waitCode(t, 5*time.Second, "replaced all after confirming", func() bool {
			b.AsyncLock()
			defer b.AsyncUnlock()
			return ln.String() == "pear tart\nkiwi pear\n"
		})

		// a link to a file outside of the project
		locked(b, func() { assert.False(t, fv.OpenFindURL(findURL("/no/such/file", 0, 0, 1, 0, 1), fv.TextEditor())) })
		locked(b, func() { assert.False(t, fv.OpenFindURL("find:///x", fv.TextEditor())) })
	})
}

// TestConfigFindButton tests that the Find dialog shows the options of the
// last find (the settings), as the arguments of Find: its Ignore case was
// the opposite (from when it was Use case), which flipped the setting each
// time Find was done with the dialog.
func TestConfigFindButton(t *testing.T) {
	for _, ignore := range []bool{false, true} {
		cv := &Code{}
		cv.Settings.Find.Find = "apple"
		cv.Settings.Find.IgnoreCase = ignore
		cv.Settings.Find.Regexp = !ignore
		fb := cv.ConfigFindButton(core.NewFuncButton(core.NewBody()).SetFunc(cv.Find))
		assert.Equal(t, "apple", fb.Args[0].Value)
		assert.Equal(t, ignore, fb.Args[2].Value, "ignore case")
		assert.Equal(t, !ignore, fb.Args[3].Value, "regexp")
	}
}
