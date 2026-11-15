// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"os/exec"
	"strings"
	"testing"
	"time"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/core"
	"cogentcore.org/core/text/lines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCodeCommands tests the build, run, and version control commands of
// Code, with their output in tabs, and killing a running command when its
// tab is deleted.
func TestCodeCommands(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}
	tmp := tempDir(t)
	path := writeFiles(t, tmp, map[string]string{"a.txt": "a\n"})
	git := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-c", "user.name=test", "-c", "user.email=test@example.com"}, args...)...)
		cmd.Dir = tmp
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
	}
	git("init", "-q", "-b", "main")
	git("add", ".")
	git("commit", "-q", "-m", "first")
	b, cv := newTestCode(t, tmp)
	core.AssertRender(t, b, "commands", func() {
		output := func(cmdline string) *lines.Lines {
			b.AsyncLock()
			defer b.AsyncUnlock()
			ed := cv.tabEditor(commandTabName(cmdline))
			if ed == nil {
				return nil
			}
			return ed.Lines
		}
		hasOutput := func(cmdline, text string) func() bool {
			return func() bool {
				buf := output(cmdline)
				return buf != nil && strings.Contains(buf.String(), text)
			}
		}
		locked(b, func() {
			log, err := cv.VCSLog() // without an open file: of the project
			require.NoError(t, err)
			assert.Len(t, log, 1)
			cv.ViewFileInIndex(path("a.txt"), 0)
			cv.Settings.BuildCommand = "echo built here"
			cv.RunBuild()
		})
		waitCode(t, 10*time.Second, "the output of the build", hasOutput("echo built here", "built here"))

		locked(b, func() {
			cv.Settings.RunCommand = "echo ran"
			cv.Run()
		})
		waitCode(t, 10*time.Second, "the output of the run", hasOutput("echo ran", "ran"))

		// a running command is killed when its tab is deleted
		var name string
		locked(b, func() {
			cv.Settings.RunCommand = "sleep 30"
			cv.Run()
			name = commandTabName("sleep 30")
			cm, _ := cv.RunningCmds.ByName(name)
			require.NotNil(t, cm)
			// (the process is this test's own sleep)
			require.Equal(t, "sleep 30", cm.CmdStr)
		})
		waitCode(t, 5*time.Second, "the sleep started", func() bool {
			b.AsyncLock()
			defer b.AsyncUnlock()
			cm, _ := cv.RunningCmds.ByName(name)
			return cm != nil && cm.Exec.Process != nil
		})
		locked(b, func() {
			cv.TabDeleted(name)
			_, i := cv.RunningCmds.ByName(name)
			assert.Equal(t, -1, i)
			assert.False(t, cv.RunningCmds.KillByName(name))
			assert.False(t, cv.RunningCmds.DeleteByName(name))

			// no run command: the executable is chosen, for other languages
			cv.Settings.RunCommand = ""
			cv.Settings.MainLang = fileinfo.C
			cv.Settings.RunExec = ""
			cv.Run()
			cv.ChooseRunExec(core.Filename(path("a.txt"))) // not executable: a dialog
			assert.Equal(t, core.Filename(path("a.txt")), cv.Settings.RunExec)
			cv.Settings.MainLang = fileinfo.TeX
			cv.Run() // a dialog
			cv.Settings.BuildCommand = ""
			cv.Settings.MainLang = fileinfo.Unknown

			cv.ExecCmd() // the menu of the commands
			cv.VCSUpdateAll()
			log, err := cv.VCSLog()
			require.NoError(t, err)
			assert.Len(t, log, 1)
			assert.Equal(t, "first", log[0].Message)
			cv.Settings.VersionControl = 0
			cv.Commit() // the message dialog
			cv.Spell()
			assert.NotNil(t, cv.Tabs().TabByName("Spell"))
		})
	})
}
