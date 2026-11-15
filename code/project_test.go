// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/vcs"
	"cogentcore.org/core/core"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/text/textpos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCodeProject tests opening a path in an empty Code, saving its
// project file, and opening that project, with its settings, in another
// empty Code, and making a new project.
func TestCodeProject(t *testing.T) {
	tmp := tempDir(t)
	path := writeFiles(t, tmp, map[string]string{
		"proj/main.go":     "package main\n\nfunc main() {}\n",
		"proj/other.go":    "package main\n",
		"proj/README.md":   "# proj\n",
		"other/a.txt":      "a\n",
		"other/other.code": "",
	})
	t.Chdir(tmp) // opening a path changes the current directory
	b, cv := newTestCode(t, "")
	core.AssertRender(t, b, "project", func() {
		locked(b, func() {
			assert.True(t, cv.IsEmpty())
			assert.Same(t, cv, cv.OpenPath(core.Filename(path("proj/main.go"))))
			assert.False(t, cv.IsEmpty())
			assert.Equal(t, core.Filename(path("proj")), cv.ProjectRoot)
			assert.Equal(t, "proj", cv.Name)
			assert.Equal(t, fileinfo.Go, cv.Settings.MainLang)
			assert.Equal(t, core.Filename(path("proj/proj")), cv.Settings.RunExec)
			assert.Equal(t, path("proj/main.go"), cv.ActiveLines().Filename(), "the file is viewed")
			assert.Contains(t, RecentPaths, path("proj"))
			wd, _ := os.Getwd()
			assert.Equal(t, path("proj"), wd)
			assert.True(t, cv.InRootPath(path("proj/other.go")))

			cv.Settings.BuildCommand = "go build ./..."
			assert.False(t, cv.SaveProjectIfExists(false), "not saved yet")
			cv.SaveProject()
			assert.FileExists(t, path("proj/proj.code"))
			assert.True(t, cv.SaveProjectIfExists(true))
			cv.SaveProjectAs(core.Filename(path("proj/proj.code")))
			pf, has := CheckForProjectAtPath(path("proj"))
			assert.True(t, has)
			assert.Equal(t, path("proj/proj.code"), pf)

			cv.ActiveLines().InsertText(textpos.Pos{}, []rune("// x\n"))
			assert.True(t, cv.SaveAllCheck(true, nil), "a dialog for the unsaved file")
			cv.ActiveLines().Undo()
			cv.ActiveLines().ClearNotSaved()
			called := false
			assert.False(t, cv.SaveAllCheck(false, func() { called = true }))
			assert.True(t, called)

			cv.UpdateFiles()
			cv.EditRecentPaths()
			cv.OpenFile(path("proj/other.go")) // in this project
			assert.Equal(t, path("proj/other.go"), cv.ActiveLines().Filename())
		})
	})

	b2, cv2 := newTestCode(t, "")
	core.AssertRender(t, b2, "project-open", func() {
		locked(b2, func() {
			cv2.OpenRecent(core.Filename(path("proj/proj.code")))
			assert.Equal(t, core.Filename(path("proj")), cv2.ProjectRoot)
			assert.Equal(t, "go build ./...", cv2.Settings.BuildCommand, "the saved settings")
			assert.Equal(t, path("proj/proj.code"), string(cv2.ProjectFilename))
		})
	})

	b3, cv3 := newTestCode(t, "")
	core.AssertRender(t, b3, "project-new", func() {
		locked(b3, func() {
			ncv := cv3.NewProject(core.Filename(tmp), "newproj", fileinfo.Python, vcs.Git)
			require.Same(t, cv3, ncv)
			assert.DirExists(t, path("newproj"))
			assert.Equal(t, fileinfo.Python, cv3.Settings.MainLang)
			assert.Equal(t, vcs.Git, cv3.Settings.VersionControl)
			cv3.NewFile("n.py", false)
			assert.FileExists(t, path("newproj/n.py"))
		})
	})

	b4, cv4 := newTestCode(t, "")
	core.AssertRender(t, b4, "project-path", func() {
		locked(b4, func() {
			// a path with a project file opens the project
			cv4.OpenPath(core.Filename(path("other")))
			assert.Equal(t, path("other/other.code"), string(cv4.Settings.ProjectFilename))
		})
	})
}

// TestCodeSettings tests applying project settings and the named splits,
// and the editors of the settings.
func TestCodeSettings(t *testing.T) {
	tmp := tempDir(t)
	writeFiles(t, tmp, map[string]string{"a.go": "package a\n"})
	b, cv := newTestCode(t, tmp)
	saved := slices.Clone(AvailableSplits)
	t.Cleanup(func() { AvailableSplits = saved; AvailableSplitNames = saved.Names() })
	core.AssertRender(t, b, "settings", func() {
		locked(b, func() {
			cv.SplitsSetView("Small")
			assert.Equal(t, SplitName("Small"), cv.Settings.SplitName)
			assert.False(t, cv.Settings.Splits[2] > 0, "one editor")
			cv.SplitsSetView("Wide")
			assert.False(t, cv.Settings.TabsUnder)
			cv.SplitsSetView("none")
			assert.Equal(t, SplitName("Wide"), cv.Settings.SplitName)

			cv.SplitsSaveAs("Mine", "my splits")
			sp, _, ok := AvailableSplits.SplitByName("Mine")
			require.True(t, ok)
			assert.Equal(t, "Mine", sp.Label())
			assert.Contains(t, AvailableSplitNames, "Mine")
			cv.SplitsSave("Mine")
			_, _, ok = AvailableSplits.SplitByName("")
			assert.False(t, ok)

			cv.GrabSettings()
			cv.Settings.Splits = [4]float32{.2, .4, .4, .2}
			cv.Settings.TabsUnder = false
			cv.ApplySettingsAction()
			cv.EditProjectSettings()
			cv.SplitsEdit()
		})

		// the editor of the splits can save them to the settings when changed
		var save *core.Button
		var sv *core.Body
		for _, w := range core.AllRenderWindows {
			if sc := w.MainScene(); sc != nil && sc.Body != nil && sc.Body.Title == "Available Splitter Settings: can duplicate an existing (using context menu) as starting point for new one" {
				sv = sc.Body
			}
		}
		require.NotNil(t, sv)
		locked(sv, func() {
			save = core.ButtonByText(sv.Scene, "Save to settings")
			require.NotNil(t, save)
			AvailableSplitsChanged = true
			save.Restyle()
			assert.False(t, save.StateIs(states.Disabled), "enabled when changed")
		})
	})
}

// TestSettingsFiles tests saving and opening the named splits and
// registers, and the commands of projects.
func TestSettingsFiles(t *testing.T) {
	tmp := tempDir(t)
	var sp Splits
	sp.CopyFrom(StandardSplits)
	sp.Add("X", "x", []float32{.1, .2, .3, .4}, true)
	fn := core.Filename(filepath.Join(tmp, "splits.json"))
	require.NoError(t, sp.Save(fn))
	var sp2 Splits
	require.NoError(t, sp2.Open(fn))
	assert.Equal(t, sp.Names(), sp2.Names())
	require.NoError(t, sp2.Open(core.Filename(filepath.Join(tmp, "none.json"))), "nothing to open")

	rg := Registers{"a": "some long text that is more than twenty"}
	rfn := core.Filename(filepath.Join(tmp, "regs.toml"))
	require.NoError(t, rg.Save(rfn))
	var rg2 Registers
	require.NoError(t, rg2.Open(rfn))
	assert.Equal(t, rg, rg2)
	assert.Equal(t, []string{"a: some long text that "}, rg2.Names())

	var ps ProjectSettings
	assert.Equal(t, "make", ps.buildCommand())
	assert.Equal(t, "", ps.runCommand())
	ps.MainLang = fileinfo.Go
	assert.Equal(t, "go build -v", ps.buildCommand())
	assert.Equal(t, "core run", ps.runCommand())
	ps.MainLang = fileinfo.TeX
	assert.Equal(t, "latexmk -pdf", ps.buildCommand())
	assert.Equal(t, "", ps.runCommand())
	ps.MainLang = fileinfo.C
	exe := filepath.Join(tmp, "my prog")
	require.NoError(t, os.WriteFile(exe, []byte("#!/bin/sh\n"), 0o755))
	ps.RunExec = core.Filename(exe)
	assert.True(t, ps.RunExecIsExec())
	assert.Equal(t, "'"+exe+"'", ps.runCommand())
	ps.RunExec = "none"
	assert.False(t, ps.RunExecIsExec())
	ps.BuildCommand, ps.RunCommand = "b", "r"
	assert.Equal(t, "b", ps.buildCommand())
	assert.Equal(t, "r", ps.runCommand())

	assert.Equal(t, "a/b.go", shellQuote("a/b.go"))
	assert.Equal(t, "''", shellQuote(""))
	assert.Equal(t, `'it'\''s'`, shellQuote("it's"))

	pf := core.Filename(filepath.Join(tmp, "p.code"))
	ps.addStandardIgnores()
	require.NoError(t, ps.Save(pf))
	var ps2 ProjectSettings
	require.NoError(t, ps2.Open(pf))
	assert.Equal(t, "b", ps2.BuildCommand)
}
