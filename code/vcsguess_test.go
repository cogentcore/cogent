// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"cogentcore.org/core/base/vcs"
	"cogentcore.org/core/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGuessVersionControl tests guessing the version control system
// from the repository at or above the project root, without the file
// tree, which is not yet built when a project is opened.
func TestGuessVersionControl(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}
	dir := t.TempDir()
	cmd := exec.Command("git", "init", "-q")
	cmd.Dir = dir
	require.NoError(t, cmd.Run())
	sub := filepath.Join(dir, "sub")
	require.NoError(t, os.Mkdir(sub, 0o755))

	for _, root := range []string{dir, sub} {
		cv := &Code{}
		cv.Settings.ProjectRoot = core.Filename(root)
		cv.guessVersionControl()
		assert.Equal(t, vcs.Git, cv.Settings.VersionControl, root)
	}

	cv := &Code{}
	cv.Settings.ProjectRoot = core.Filename(t.TempDir())
	cv.guessVersionControl()
	assert.Equal(t, vcs.NoVCS, cv.Settings.VersionControl)
}
